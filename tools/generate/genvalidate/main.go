package genvalidate

import (
	"context"
	"fmt"
	"go/types"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"

	"userclouds.com/infra/codegen"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/tools/generate"
)

type data struct {
	Package string
	Type    string
	Fields  []field

	HasIDField       bool
	HasExtraValidate bool

	Imports string
}

type field struct {
	Name         string
	Validateable bool

	IsPointer        bool
	IsNotEmptySecret bool

	IntNotZero      bool
	PointerAllowNil bool
	UUIDNotNil      bool

	Length *valuesRange

	ValidateArray bool
	ArrayUnique   bool
	ArrayKeyType  string
	ArrayKeyName  string

	ErrorPrefix string
}

type valuesRange struct {
	// For numeric fields this can be min max values, for strings, min max length
	HasMin bool
	HasMax bool
	Min    int
	Max    int
}

func newStringNotEmpty() *valuesRange {
	return &valuesRange{HasMin: true, HasMax: false, Min: 1, Max: -1}
}

func (f *field) StringNotEmpty() bool {
	if f.Length == nil {
		return false
	}
	return f.Length.HasMin && f.Length.Min == 1 && !f.Length.HasMax
}

func (f *field) HasLength() bool {
	return f.Length != nil
}

func getTagValue(tv string) (bool, int, error) {
	if len(tv) == 0 {
		return false, -1, nil
	}
	val, err := strconv.Atoi(tv)
	if err != nil {
		return false, -1, ucerr.Wrap(err)
	}
	return true, val, nil
}

// Run implements the generator interface
func Run(ctx context.Context, p *packages.Package, path string, args ...string) {

	imports := codegen.NewImports()

	data := data{
		Type:    args[1],
		Package: p.Name,
	}

	typ := p.Types.Scope().Lookup(data.Type)
	if typ == nil {
		uclog.Fatalf(ctx, "couldn't find type %s in package scope", data.Type)
	}

	s, ok := typ.Type().Underlying().(*types.Struct)
	if !ok {
		uclog.Fatalf(ctx, "type %s doesn't appear to be a struct", data.Type)
	}

	for i := range s.NumFields() {
		f := s.Field(i)
		fs := field{
			Name:        f.Name(),
			ErrorPrefix: "ucerr.Friendlyf(nil, ",
		}

		typeName := f.Type().String()

		// if this object has an ID field, use it to improve error messages
		// we could (and someday should) traverse embedded objects to find ID fields but
		// I'm being lazy with BaseModel(s) for now
		if fs.Name == "ID" ||
			typeName == "userclouds.com/infra/ucdb.BaseModel" ||
			typeName == "userclouds.com/infra/ucdb.UserBaseModel" {
			data.HasIDField = true
		}

		fs.IsPointer = typeName == "interface{}" || typeName == "any" || strings.HasPrefix(typeName, "*")

		if fs.IsPointer {
			imports.Add("userclouds.com/infra/ucerr")
		}

		tags := reflect.StructTag(s.Tag(i))
		valTag, ok := tags.Lookup("validate")
		underlyingType := f.Type().Underlying()
		if ok {
			if strings.HasSuffix(valTag, "notfriendly") {
				fs.ErrorPrefix = "ucerr.Errorf("
				valTag = strings.TrimSuffix(valTag, "notfriendly")
				valTag = strings.TrimSuffix(valTag, ",")
			}

			if valTag == "notnil" && (typeName == "github.com/gofrs/uuid.UUID" ||
				f.Type().String() == "[]github.com/gofrs/uuid.UUID") {
				fs.UUIDNotNil = true
				imports.Add("userclouds.com/infra/ucerr")
			} else if typeName == "*string" && strings.Contains(valTag, "notempty") && strings.Contains(valTag, "allownil") {
				imports.Add("userclouds.com/infra/ucerr")
				fs.Length = newStringNotEmpty()
				fs.PointerAllowNil = true
			} else if strings.Contains(valTag, "notempty") && typeName == "userclouds.com/infra/secret.String" {
				fs.IsNotEmptySecret = true
				imports.Add("userclouds.com/infra/ucerr")
			} else if isStringType := typeName == "string" || underlyingType.String() == "string"; valTag == "notempty" && isStringType {
				fs.Length = newStringNotEmpty()
				imports.Add("userclouds.com/infra/ucerr")
			} else if valTag == "notzero" && ((typeName == "int" || underlyingType.String() == "int") ||
				(typeName == "int64" || underlyingType.String() == "int64")) {
				fs.IntNotZero = true
				imports.Add("userclouds.com/infra/ucerr")
			} else if valTag == "allownil" && fs.IsPointer {
				fs.PointerAllowNil = true
			} else if strings.HasPrefix(valTag, "unique,") {
				fs.ArrayUnique = true
				tagParts := strings.Split(valTag, ",")
				if len(tagParts) != 2 {
					uclog.Fatalf(ctx, "unique tag '%s' is not of the form 'unique,Name'", valTag)
				}
				fs.ArrayKeyName = tagParts[1]
			} else if strings.HasPrefix(valTag, "length:") {
				if !isStringType {
					uclog.Fatalf(ctx, "length can only be used on string fields, field %v is: %v", fs.Name, typeName)
				}
				tagValues := strings.Split(strings.TrimPrefix(valTag, "length:"), ",")
				if len(tagValues) != 2 {
					uclog.Fatalf(ctx, "length tag '%s' is not of the form 'length:min,max'", valTag)
				}
				hasMin, minValue, err := getTagValue(tagValues[0])
				if err != nil {
					uclog.Fatalf(ctx, "Failed to read min value (%s) in length tag: '%s'", tagValues[0], valTag)
				}
				hasMax, maxValue, err := getTagValue(tagValues[1])
				if err != nil {
					uclog.Fatalf(ctx, "Failed to read max value (%s) in length tag: '%s'", tagValues[1], valTag)
				}
				fs.Length = &valuesRange{HasMin: hasMin, Min: minValue, HasMax: hasMax, Max: maxValue}
				imports.Add("userclouds.com/infra/ucerr")
			} else if valTag == "skip" {
				continue
			} else {
				uclog.Fatalf(ctx, "didn't understand validate tag: %s with type: %v on field %v", valTag, typeName, fs.Name)
			}
		}

		// if this type has a validate method, we're good there
		// Note we check this first in case the type is an aliased array
		// and defines its own Validate method (which it prefers to use
		// defining a loop over all its Validateable elements)
		// TODO: there's probably a better way to except system structs than time.Time here
		if hasValidate(f.Type(), underlyingType) &&
			typeName != "time.Time" {
			fs.Validateable = true
			imports.Add("userclouds.com/infra/ucerr")
		} else if slice, ok := underlyingType.(*types.Slice); ok {
			// if it's an array, check if the element type is validateable
			// we check .Underlying() because sometimes it's just == f.Type(), and sometimes
			// it dereferences a useful type like `type PlexMaps []PlexMap`
			fs.ValidateArray = true
			if hasValidate(slice.Elem()) {
				fs.Validateable = true
				imports.Add("userclouds.com/infra/ucerr")
			} else {
				// if there is no validation to do on the array elements, then there won't be any code in the loop's body.
				// So we want to avoid it entirely
				fs.ValidateArray = fs.StringNotEmpty() || fs.PointerAllowNil ||
					(fs.HasLength() && (fs.Length.HasMin || fs.Length.HasMax)) ||
					fs.IntNotZero || fs.UUIDNotNil
			}
		}

		if fs.ArrayUnique {
			if !fs.ValidateArray {
				uclog.Fatalf(ctx, "unique validate tag used for non-array field %s", f.Name())
			}
			keyTypeTag, ok := tags.Lookup("keytype")
			if !ok {
				uclog.Fatalf(ctx, "keytype tag missing for unique validate tag used for field %s", f.Name())
			}

			keyPackage, keyType := parseKeyTypeTag(keyTypeTag)
			if len(keyPackage) > 0 {
				imports.Add(keyPackage)
			}
			fs.ArrayKeyType = keyType
		}

		data.Fields = append(data.Fields, fs)
	}

	// now let's check for an extraValidate() function, which is how we delegate non-automated work
	// we have to check the method sets for both the type and the pointer type just in case
	if generate.HasMethod(typ.Type(), "extraValidate") {
		data.HasExtraValidate = true
		imports.Add("userclouds.com/infra/ucerr")
	}
	data.Imports = imports.String()

	// actually write the template to a file
	fn := filepath.Join(path, fmt.Sprintf("%s_validate_generated.go", strings.ToLower(data.Type)))
	generate.WriteFileIfChanged(ctx, fn, templateString, data)
}

func hasValidate(typesToCheck ...types.Type) bool {
	for _, typ := range typesToCheck {
		if generate.HasMethod(typ, "Validate") {
			return true
		}
		// TODO: occasionally we don't load methodsets correctly across packages, and rather
		// than fully fix it, just assume structs should have Validate and let the compiler
		// complain if they don't :)
		if _, ok := typ.(*types.Struct); ok {
			return true
		}
	}
	return false
}

func parseKeyTypeTag(keyTypeTag string) (keyPackage string, keyType string) {
	tagParts := strings.Split(keyTypeTag, "/")
	keyType = tagParts[len(tagParts)-1]
	tagParts = strings.Split(keyTypeTag, ".")
	keyPackage = strings.Join(tagParts[0:len(tagParts)-1], ".")
	return
}

var templateString = `// NOTE: automatically generated file -- DO NOT EDIT

package << .Package >>
<<- if .Imports >>

import (
<< .Imports >>
)<<- end >>

// Validate implements Validateable
func (o << .Type >>) Validate() error {
	<<- range $m := .Fields >>
	<<- if $m.ValidateArray >>
	<<- if $m.ArrayUnique >>
	keysFor<< $m.Name >> := map[<< $m.ArrayKeyType >>]bool{}
	<<- end >>
	<<- if  $m.Validateable  >>
	for _, item := range o.<< $m.Name >> {
	<<- else >>
	for i, item := range o.<< $m.Name >> {
	<<- end >>
		<<- if $m.Validateable >>
		<<- if $m.PointerAllowNil >>
		if item != nil {
			if err := item.Validate(); err != nil {
				return ucerr.Wrap(err)
			}
		}
		<<- else >>
		if err := item.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
		<<- end >>
		<<- else if $m.StringNotEmpty >>
		<<- if $m.PointerAllowNil >>
		if item != nil && *item == "" {
			return << $m.ErrorPrefix >>"<< $.Type >>.<< $m.Name >>[%d]<< if $.HasIDField >> (%v)<< end >> can't be not nil and empty", i<< if $.HasIDField >>, o.ID<< end >>)
		}
		<<- else >>
		if item == "" {
			return << $m.ErrorPrefix >>"<< $.Type >>.<< $m.Name >>[%d]<< if $.HasIDField >> (%v)<< end >> can't be empty", i<< if $.HasIDField >>, o.ID<< end >>)
		}
		<<- end >>
		<<- else if $m.HasLength >>
		<<- if $m.Length.HasMin >>
		if len(item) < << $m.Length.Min >> {
			return << $m.ErrorPrefix >>"<< $.Type >>.<< $m.Name >>[%d] has to have a maximal length of << $m.Length.Man  >> (length: %v)", i, len(o.<< $m.Name >>)
		}
		<<- end >>
		<<- if $m.Length.HasMax >>
		if len(item) > << $m.Length.Max >> {
			return << $m.ErrorPrefix >>"<< $.Type >>.<< $m.Name >>[%d] has to have a maximal length of << $m.Length.Man  >> (length: %v)", i, len(o.<< $m.Name >>)
		}
		<<- end >>
		<<- else if $m.IntNotZero >>
		if item == 0 {
			return << $m.ErrorPrefix >>"<< $.Type >>.<< $m.Name >>[%d]<< if $.HasIDField >> (%v)<< end >> can't be 0", i<< if $.HasIDField >>, o.ID<< end >>)
		}
		<<- else if $m.UUIDNotNil >>
		if item.IsNil() {
			return << $m.ErrorPrefix >>"<< $.Type >>.<< $m.Name >>[%d]<< if $.HasIDField >> (%v)<< end >> can't be nil", i<< if $.HasIDField >>, o.ID<< end >>)
		}
		<<- else if and ($m.IsPointer) (not $m.PointerAllowNil) >>
		if item == nil {
			return << $m.ErrorPrefix >>"<< $.Type >>.<< $m.Name >>[%d]<< if $.HasIDField >> (%v)<< end >> can't be nil", i<< if $.HasIDField >>, o.ID<< end >>)
		}
		<<- end >>
		<<- if $m.ArrayUnique >>
		if _, found := keysFor<< $m.Name >>[item.<< $m.ArrayKeyName >>]; found {
			return << $m.ErrorPrefix >>"duplicate << $m.ArrayKeyName >> '%v' in << $m.Name >>", item.<< $m.ArrayKeyName >>)
		}
		keysFor<< $m.Name >>[item.<< $m.ArrayKeyName >>] = true
		<<- end >>
	}
	<<- else >>
	<<- if $m.Validateable >>
	<<- if $m.PointerAllowNil >>
	if o.<< $m.Name >> != nil {
		if err := o.<< $m.Name >>.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	<<- else if $m.IsNotEmptySecret >>
	if o.<< $m.Name >>.IsEmpty() {
		return << $m.ErrorPrefix >>"<< $.Type >>.<< $m.Name >> can't be empty")
	}
	<<- else >>
	if err := o.<< $m.Name >>.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	<<- end >>
	<<- else if $m.StringNotEmpty >>
	<<- if $m.PointerAllowNil >>
	if o.<< $m.Name >> != nil && *o.<< $m.Name >> == "" {
		return << $m.ErrorPrefix >>"<< $.Type >>.<< $m.Name >><< if $.HasIDField >> (%v)<< end >> can't be not nil and empty"<< if $.HasIDField >>, o.ID<< end >>)
	}
	<<- else >>
	if o.<< $m.Name >> == "" {
		return << $m.ErrorPrefix >>"<< $.Type >>.<< $m.Name >><< if $.HasIDField >> (%v)<< end >> can't be empty"<< if $.HasIDField >>, o.ID<< end >>)
	}
	<<- end >>
	<<- else if $m.HasLength >>
	<<- if and $m.Length.HasMin  $m.Length.HasMax >>
	if len(o.<< $m.Name >>) < << $m.Length.Min >> || len(o.<< $m.Name >>) > << $m.Length.Max >> {
		return << $m.ErrorPrefix >>"<< $.Type >>.<< $m.Name >> length has to be between << $m.Length.Min  >> and << $m.Length.Max >> (length: %v)", len(o.<< $m.Name >>))
	}
	<<- else if $m.Length.HasMin >>
	if len(o.<< $m.Name >>) < << $m.Length.Min >> {
		return << $m.ErrorPrefix >>"<< $.Type >>.<< $m.Name >> has to have a minimal length of << $m.Length.Min  >> (length: %v)", len(o.<< $m.Name >>))
	}
	<<- else if $m.Length.HasMax >>
	if len(o.<< $m.Name >>) > << $m.Length.Max >> {
		return << $m.ErrorPrefix >>"<< $.Type >>.<< $m.Name >> has to have a maximal length of << $m.Length.Man  >> (length: %v)", len(o.<< $m.Name >>)
	}
	<<- end >>
	<<- else if $m.IntNotZero >>
	if o.<< $m.Name >> == 0 {
		return << $m.ErrorPrefix >>"<< $.Type >>.<< $m.Name >><< if $.HasIDField >> (%v)<< end >> can't be 0"<< if $.HasIDField >>, o.ID<< end >>)
	}
	<<- else if $m.UUIDNotNil >>
	if o.<< $m.Name >>.IsNil() {
		return << $m.ErrorPrefix >>"<< $.Type >>.<< $m.Name >><< if $.HasIDField >> (%v)<< end >> can't be nil"<< if $.HasIDField >>, o.ID<< end >>)
	}
	<<- else if and ($m.IsPointer) (not $m.PointerAllowNil) >>
	if o.<< $m.Name >> == nil {
		return << $m.ErrorPrefix >>"<< $.Type >>.<< $m.Name >><< if $.HasIDField >> (%v)<< end >> can't be nil"<< if $.HasIDField >>, o.ID<< end >>)
	}
	<<- end >>
	<<- end >>
	<<- end >>
	<<- if .HasExtraValidate >>
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	<<- end >>
	return nil
}
`
