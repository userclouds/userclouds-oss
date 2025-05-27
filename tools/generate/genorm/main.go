package genorm

import (
	"context"
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/tools/go/packages"

	"userclouds.com/infra/codegen"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/repopath"
	"userclouds.com/tools/generate"
)

const versionColName = "_version"  // used by tables with version not included in the primary key
const versionPKColName = "version" // used by tables that include version in the primary key

type data struct {
	PackageName string // the name of the package we're generating code for

	StorageClassPrefix string // the prefix for the storage class name

	TypeName               string // the name of the type
	TypeNamePlural         string // name of the type as a plural (eg special case for "-y" -> "-ies")
	TypePackageName        string // the name of the package where the type lives, if different
	TypePackageNameWithDot string // the name of the package where the type lives, if different with "." appended
	FullyQualifiedTypeName string // the name of the type (with import if required)
	BaseModel              string // the name of the BaseModel type, if any

	DatabaseName string
	Table        string

	GenStorageObject  bool
	Get               bool
	GetByName         bool
	List              bool
	MultiGet          bool
	NonPaginatedList  bool
	Delete            bool
	Versioned         bool // TODO: we should auto-detect if the object is versioned and set this appropriately
	Save              bool
	PriorSave         bool
	WithCache         bool
	CachePages        bool
	IncludeInsertOnly bool
	FollowerReads     bool

	Columns       string // names of the database columns eg "id, updated, name, location" that doesn't include created
	RegionColumns string // Columns + ", crdb_region"
	SelectColumns string // Columns + ", created"
	Values        string // substitution params for save, eg "$1, NOW(), $2, $3"
	RegionValues  string // substitution params plus one more for region
	OnConflict    string // names of columns to use in the ON CONFLICT clause
	Fields        string // names of the struct fields eg "i.ID, i.Name, i.Location"
	RegionFields  string // Fields + ", reg"

	ColumnNames []string // this is equiv to strings.Split(Columns, ", ") but kept separately for template convenience

	Updates        string // conflicted updates list
	ImmutableWhere string // immutable fields

	Imports string

	HasPreSave            bool
	HasAdditionalSaveKeys bool
	HasPreDelete          bool
	HasUserID             bool
	HasVersion            bool

	AccessPrimaryDbOption bool
}

type parseData struct {
	cols      []string
	vals      []string
	fields    []string
	baseModel string

	immutable []string

	hasUserID    bool
	hasCreated   bool // this is used to include "created" in the safe column names, even though it's not in generated queries
	hasVersion   bool
	hasVersionPK bool

	lastVal int // count of the $ substitutions we're using, tracked here in case of embedded structs/recursion
}

// Run implements the generator interface
func Run(ctx context.Context, p *packages.Package, path string, args ...string) {

	flagMultiGet := true
	flagGet := true
	flagGetByName := false
	flagVersioned := false
	flagList := true
	flagDelete := true
	flagSave := true
	flagPriorSave := false
	flagWithCache := false
	flagCachePages := false
	flagNonPaginatedList := false
	flagIncludeInsertOnly := false
	flagFollowerReads := false
	flagStorageClassPrefix := false
	flagAccessPrimaryDbOption := false
	var flagColumnListOnly bool

	// TODO: we can't use the standard flag package here because we're running
	// in-process with parallelgen, which is going to be a standard codegen problem
	// but it's much much (3x) faster to do this. Should we factor out our own
	// ucflags helpers for codegen?
	// NB: nonFlagArgs[0] will always be "genorm"
	nonFlagArgs := []string{}
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			switch arg {
			case "--noget":
				flagGet = false
				flagMultiGet = false
			case "--versioned":
				flagVersioned = true
				flagMultiGet = false // Not supporting getting multiple objects by IDs (for now). We may add that in the future.
			case "--getbyname":
				flagGetByName = true
			case "--nomultiget":
				flagMultiGet = false
			case "--nolist":
				flagList = false
			case "--columnlistonly":
				flagColumnListOnly = true
			case "--nodelete":
				flagDelete = false
			case "--nosave":
				flagSave = false
			case "--priorsave":
				flagPriorSave = true
			case "--cache":
				flagWithCache = true
			case "--cachepages":
				flagCachePages = true
			case "--nonpaginatedlist":
				flagNonPaginatedList = true
			case "--includeinsertonly":
				flagIncludeInsertOnly = true
			case "--followerreads":
				flagFollowerReads = true
			case "--storageclassprefix":
				flagStorageClassPrefix = true
			case "--accessprimarydboption":
				flagAccessPrimaryDbOption = true
			default:
				uclog.Fatalf(ctx, "unrecognized flag %s", arg)
			}
		} else {
			nonFlagArgs = append(nonFlagArgs, arg)
		}
	}

	if flagPriorSave && !flagSave {
		uclog.Fatalf(ctx, "priorsave flag requires save flag")
	}

	if strings.TrimSpace(nonFlagArgs[1]) == "" {
		uclog.Fatalf(ctx, "must specify type name as 1nd arg. got '%v' command %v", nonFlagArgs[1], strings.Join(nonFlagArgs, " "))
	}

	if strings.TrimSpace(nonFlagArgs[2]) == "" {
		uclog.Fatalf(ctx, "must specify table name as 2rd arg. got '%v' command %v", nonFlagArgs[2], strings.Join(nonFlagArgs, " "))
	}

	if strings.TrimSpace(nonFlagArgs[3]) == "" {
		uclog.Fatalf(ctx, "must specify database name as 3th arg. got '%v' command %v", nonFlagArgs[3], strings.Join(nonFlagArgs, " "))
	}

	if flagStorageClassPrefix && len(nonFlagArgs) < 5 {
		uclog.Fatalf(ctx, "must specify storage class prefix as 4th arg. got '%v' command %v", nonFlagArgs[4], strings.Join(nonFlagArgs, " "))
	}

	if flagWithCache && flagAccessPrimaryDbOption {
		uclog.Fatalf(ctx, "accessprimarydboption flag and cache flag cannot be used together")
	}

	imports := codegen.NewImports()
	imports.Add("context")
	imports.Add("userclouds.com/infra/ucerr")

	databaseNames := strings.Split(nonFlagArgs[3], ",")
	data := data{
		FullyQualifiedTypeName: nonFlagArgs[1],
		PackageName:            p.Name,
		Table:                  nonFlagArgs[2],
		DatabaseName:           "", // This gets initialized with values from databaseNames[] prior to executing the template
		Get:                    flagGet,
		GetByName:              flagGetByName,
		Versioned:              flagVersioned,
		List:                   flagList,
		MultiGet:               flagMultiGet && flagGet,
		Delete:                 flagDelete,
		Save:                   flagSave,
		PriorSave:              flagPriorSave,
		WithCache:              flagWithCache,
		CachePages:             flagCachePages,
		NonPaginatedList:       flagNonPaginatedList,
		IncludeInsertOnly:      flagIncludeInsertOnly,
		FollowerReads:          flagFollowerReads && flagWithCache,
		AccessPrimaryDbOption:  flagAccessPrimaryDbOption,
	}

	if flagStorageClassPrefix {
		data.StorageClassPrefix = nonFlagArgs[4]
	}

	scope := p.Types.Scope()

	// TODO: automatically generating Storage might get confusing if we have multiple
	// types using auto-gen'd ORM in a package (it won't be non-deterministic, but not
	// consistent between packages)
	stor := scope.Lookup(data.StorageClassPrefix + "Storage")
	if stor == nil {
		data.GenStorageObject = true
	}

	// figure out our type names
	if strings.Contains(data.FullyQualifiedTypeName, ".") {
		parts := strings.Split(data.FullyQualifiedTypeName, ".")
		if len(parts) != 2 {
			uclog.Fatalf(ctx, "don't understand type name with more than one .: %v", data.FullyQualifiedTypeName)
		}
		data.TypePackageName = parts[0]
		data.TypeName = parts[1]
		data.TypePackageNameWithDot = fmt.Sprintf("%s.", data.TypePackageName)
	} else {
		data.TypeName = data.FullyQualifiedTypeName
	}

	data.TypeNamePlural = generate.GetPluralName(data.TypeName)

	// load our type
	var obj types.Object
	if data.TypePackageName != "" {
		// this is for un-renamed imports
		for n, o := range p.TypesInfo.Implicits {
			if imp, ok := n.(*ast.ImportSpec); ok {
				if o.Name() == data.TypePackageName {
					imports.Add(strings.Trim(imp.Path.Value, `"`))

					// TODO this is janky but I think it works consistently
					levels := strings.Count(p.PkgPath, "/")
					upPath := strings.Repeat("../", levels)
					newPackage := strings.Replace(imp.Path.Value, "userclouds.com/", "", 1)
					newPackage = strings.Trim(newPackage, `"`)
					newPath := fmt.Sprintf("%s%s", upPath, newPackage)

					np := generate.GetPackageForPath(filepath.Join(path, newPath), false)
					obj = np.Types.Scope().Lookup(data.TypeName)
					break
				}
			}
		}

		// TODO support renamed imports using p.TypesInfo.Defs to map names -> packages
		// and for renamed imports
		// for n, o := range p.TypesInfo.Defs {
		// 	if n.Name == parts[0] {
		// 		scope = o.Pkg().Scope()
		// 		break
		// 	}
		// }
	} else {
		obj = p.Types.Scope().Lookup(data.TypeName)
	}

	// just in case we failed
	if obj == nil {
		uclog.Fatalf(ctx, "couldn't load type %s [%s] in package scope[%v]", data.TypeName, data.FullyQualifiedTypeName, data.TypePackageName)
	}

	s, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		uclog.Fatalf(ctx, "can't autogen orm for non-struct types (yet?)")
	}

	var pd parseData
	if err := parseStructBase(s, &pd); err != nil {
		uclog.Fatalf(ctx, "error parsing struct: %v\n", err)
	}

	// we're going to create a parallel pd.vals slice to use for data.Values
	// so that we can replace the version value with our extra one if needed, without
	// affecting the original pd.vals slice that we use to generate things like the WHERE clause
	// TODO: I don't love this factoring but it's the best I've come up with so far
	insertVals := slices.Clone(pd.vals)

	// this is defined outside the if to avoid scoping issues, if unused it's just ignored
	versionExtraParam := fmt.Sprintf("$%d", pd.lastVal+1)
	if pd.hasVersion {
		// we need to pass in an extra local var for this
		pd.fields = append(pd.fields, "newVersion")
		pd.lastVal++
		insertVals[findIdx(pd.cols, versionColName)] = versionExtraParam
	}
	regionParam := fmt.Sprintf("$%d", pd.lastVal+1)

	data.Columns = strings.Join(pd.cols, ", ")
	data.RegionColumns = strings.Join(append(pd.cols, "crdb_region"), ", ")
	data.SelectColumns = fmt.Sprintf("%s, created", data.Columns)
	data.Values = strings.Join(insertVals, ", ")
	data.RegionValues = strings.Join(append(insertVals, regionParam), ", ")
	if pd.hasVersionPK {
		data.OnConflict = "id, version, deleted"
	} else {
		data.OnConflict = "id, deleted"
	}
	data.Fields = strings.Join(pd.fields, ", ")
	data.RegionFields = strings.Join(append(pd.fields, "reg"), ", ")
	data.HasUserID = pd.hasUserID
	data.HasVersion = pd.hasVersion
	data.BaseModel = pd.baseModel

	var updates []string
	for i, col := range pd.cols {
		// skip ID because it's by definition immutable
		if col == "id" {
			continue
		}
		if col == versionColName {
			// we want to save an incremented version, so we've got an extra param to pass in
			// rather than the object's value
			updates = append(updates, fmt.Sprintf("%s = %s", col, versionExtraParam))
			continue
		}
		updates = append(updates, fmt.Sprintf("%s = %s", pd.cols[i], pd.vals[i]))
	}

	var wheres []string
	for _, im := range pd.immutable {
		idx := findIdx(pd.cols, im)
		wheres = append(wheres, fmt.Sprintf("%s.%s = %s", data.Table, pd.cols[idx], pd.vals[idx]))
	}

	// if there's an version, we need to check it and also update it correctly
	if data.HasVersion {
		versionIdx := findIdx(pd.cols, versionColName)
		wheres = append(wheres, fmt.Sprintf("%s.%s = %s", data.Table, versionColName, pd.vals[versionIdx]))
	}

	// we always want to check the ID, whether or not there are immutable columns
	wheres = append(wheres, fmt.Sprintf("%s.id = %s", data.Table, pd.vals[findIdx(pd.cols, "id")]))

	data.Updates = strings.Join(updates, ", ")
	data.ImmutableWhere = strings.Join(wheres, " AND ")

	// now let's check for an preDelete() etc functions, which is how we delegate non-automated work
	// we have to check the method sets for both the type and the pointer type just in case
	// this lives on the Storage object because a) delete takes an ID, not an object, and b) we might
	// need to do other queries here (like make sure other stuff is already deleted?)
	if stor != nil {
		if generate.HasMethod(stor.Type(), fmt.Sprintf("preDelete%s", data.TypeName)) {
			data.HasPreDelete = true
		}
		if generate.HasMethod(stor.Type(), fmt.Sprintf("preSave%s", data.TypeName)) {
			data.HasPreSave = true
		}
		if generate.HasMethod(stor.Type(), fmt.Sprintf("additionalSaveKeysFor%s", data.TypeName)) {
			data.HasAdditionalSaveKeys = true
		}
	}
	if data.MultiGet {
		imports.Add("time")
		imports.Add("github.com/lib/pq")
		imports.Add("userclouds.com/infra/uctypes/set")
	}
	if data.Versioned && data.Get {
		imports.Add("github.com/gofrs/uuid")
	}
	if data.Get {
		imports.Add("github.com/gofrs/uuid")
		imports.Add("time")
		imports.Add("errors")
		imports.Add("database/sql")
	}
	if data.GenStorageObject {
		imports.Add("github.com/jmoiron/sqlx")
	}
	if data.List {
		imports.Add("fmt")
		imports.Add("userclouds.com/infra/pagination")
	}
	if data.Delete {
		imports.Add("github.com/gofrs/uuid")
		imports.Add("errors")
		imports.Add("database/sql")
	}
	if data.Save {
		imports.Add("errors")
		imports.Add("database/sql")
	}
	if data.WithCache {
		if data.Get || data.Versioned {
			imports.Add("userclouds.com/infra/uclog")
		}
		imports.Add("userclouds.com/infra/cache")
		if data.Delete || data.Versioned {
			imports.Add("userclouds.com/infra/ucdb")
		}
	}
	if data.AccessPrimaryDbOption {
		imports.Add("userclouds.com/infra/featureflags")
	}
	data.Imports = imports.String()

	// this strange flag exists for logserver right now
	// TODO: remove me once logserver DB patterns are normalized
	var fn string
	var fh *os.File
	var err error

	if !flagColumnListOnly {
		fn = filepath.Join(path, fmt.Sprintf("%s_orm_generated.go", strings.ToLower(data.TypeName)))
		fh, err = os.Create(fn)
		if err != nil {
			uclog.Fatalf(ctx, "error opening orm output file: %v", err)
		}

		temp := template.Must(template.New("genorm").Delims("<<", ">>").Parse(templateString))
		if err := temp.Execute(fh, data); err != nil {
			uclog.Fatalf(ctx, "error executing orm template: %v", err)
		}
	}

	// and finally, write out our list of "used columns" to a file for later validation

	// sort to make validation checks faster
	// we do this here so we don't mess up ordering in the existing generated files
	data.ColumnNames = pd.cols
	if pd.hasCreated {
		data.ColumnNames = append(data.ColumnNames, "created")
	}
	sort.Strings(data.ColumnNames)

	// genorm runs in the same directory it was called from, so we have to find our root
	baseDir := repopath.BaseDir()

	for _, dbName := range databaseNames {
		data.DatabaseName = dbName
		fn = filepath.Join(baseDir, fmt.Sprintf("internal/%s/columns_%s_generated.go", data.DatabaseName, data.Table))
		generate.WriteFileIfChanged(ctx, fn, columnTemplateString, data)
	}
}

func findIdx(cols []string, name string) int {
	ctx := context.Background() // no reason to pass this in all the time

	// find the index of the immutable column
	for i, col := range cols {
		if name == col {
			return i
		}
	}

	uclog.Fatalf(ctx, "couldn't find column %s in %v", name, cols)
	return -1 // no-op but this way we don't need to check RV
}

func parseStructBase(s *types.Struct, p *parseData) error {
	// Only do this scan on top level struct so that the recursion doesn't override the base model
	if s.NumFields() > 0 {
		f := s.Field(0)

		typeName := f.Type().String()
		switch typeName {
		case "userclouds.com/infra/ucdb.SystemAttributeBaseModel":
			p.baseModel = "SystemAttributeBase"
		case "userclouds.com/infra/ucdb.VersionBaseModel":
			p.baseModel = "VersionBase"
		case "userclouds.com/infra/ucdb.UserBaseModel":
			p.baseModel = "UserBase"
		case "userclouds.com/infra/ucdb.BaseModel":
			p.baseModel = "Base"
		}
	}

	return ucerr.Wrap(parseStruct(s, p))
}

func parseStruct(s *types.Struct, p *parseData) error {
	for i := range s.NumFields() {
		f := s.Field(i)
		if f.Embedded() {
			s2, ok := f.Type().Underlying().(*types.Struct)
			if !ok {
				return ucerr.Errorf("embedded type %s is not a struct", f.Name()) // unexpected right now
			}

			if err := parseStruct(s2, p); err != nil {
				return ucerr.Errorf("parseStruct %v: %w", f.Name(), err)
			}

			continue
		}

		if !f.Exported() {
			continue
		}

		name := f.Name()
		tags := reflect.StructTag(s.Tag(i))
		tagVal, ok := tags.Lookup("db")
		if !ok {
			return ucerr.Errorf("field %s has no db name", name)
		}

		var dbName string
		if strings.Contains(tagVal, ",") {
			parts := strings.Split(tagVal, ",")
			if len(parts) != 2 {
				return ucerr.Errorf("don't understand field %v struct tag %v", name, tagVal)
			}
			dbName = parts[0]
			if parts[1] == "immutable" {
				p.immutable = append(p.immutable, dbName)
			} else {
				return ucerr.Errorf("don't understand field %v struct tag part 2 %v", name, parts[1])
			}
		} else {
			dbName = tagVal
		}

		if dbName == "user_id" && f.Type().String() == "github.com/gofrs/uuid.UUID" {
			p.hasUserID = true
		}

		if dbName == versionColName && f.Type().String() == "int" {
			p.hasVersion = true
		}

		if dbName == versionPKColName && f.Type().String() == "int" {
			p.hasVersionPK = true
		}

		// we never write save statements for created, it's managed by the DB as DEFAULT NOW()
		var val string
		if dbName == "created" || dbName == "-" {
			p.hasCreated = true
			continue
		} else if dbName == "updated" {
			val = "CLOCK_TIMESTAMP()"
		} else {
			p.lastVal++
			val = fmt.Sprintf("$%d", p.lastVal)
			// append to fields here, because we don't want to do it in the updated case
			p.fields = append(p.fields, fmt.Sprintf("obj.%s", name))
		}

		p.cols = append(p.cols, dbName)
		p.vals = append(p.vals, val)

	}

	return nil
}

// TODO: should we catch err.SqlNoRows in INSERT/ON CONFLICT and return a uc-specific failure code?
//   not needed but maybe nicer? we don't wrap elsewhere

var templateString = `// NOTE: automatically generated file -- DO NOT EDIT

package << .PackageName >>

import (
<< .Imports >>
)
<<- if .GenStorageObject >>

// <<.StorageClassPrefix>>Storage provides an object for database access
type <<.StorageClassPrefix>>Storage struct {
	db *ucdb.DB
}

// New returns a <<.StorageClassPrefix>>Storage object
func New(db *ucdb.DB) *<<.StorageClassPrefix>>Storage {
	return &<<.StorageClassPrefix>>Storage{db: db}
}
<<- end >>
<<- if or .Get .MultiGet >>

// Is<<.TypeName>>SoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *<<.StorageClassPrefix>>Storage) Is<<.TypeName>>SoftDeleted(ctx context.Context, id uuid.UUID<<- if .AccessPrimaryDbOption ->>, usePrimaryDbOnly bool<<- end ->>) (bool, error) {
	<<- if .WithCache >>
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *<<.FullyQualifiedTypeName>>
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[<<.FullyQualifiedTypeName>>](ctx, *s.cm, s.cm.N.GetKeyNameWithID(<<.FullyQualifiedTypeName>>KeyID, id), s.cm.N.GetKeyNameWithID(<<.TypePackageNameWithDot>>IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[Is<<.TypeName>>SoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[Is<<.TypeName>>SoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return false, nil
		}
	}
	<<- end >>
	const q = "/* lint-sql-ok */ SELECT deleted FROM <<.Table>> WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	<<- if .WithCache >>
	if err := s.db.GetContextWithDirty(ctx, "Is<<.TypeName>>SoftDeleted", &deleted, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
	<<- else if .AccessPrimaryDbOption >>
	useReplica := featureflags.IsEnabledGlobally(ctx, featureflags.ReadFromReadReplica)
	if err := s.db.GetContextWithDirty(ctx, "Is<<.TypeName>>SoftDeleted", &deleted, q, usePrimaryDbOnly || !useReplica, id); err != nil {
	<<- else >>
	if err := s.db.GetContext(ctx, "Is<<.TypeName>>SoftDeleted", &deleted, q, id); err != nil {
	<<- end >>
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}
<<- end >>
<<- if .Get >>
<<- if .Versioned >>

// GetLatest<<.TypeName>> looks up the latest version of <<.TypeName>> by ID
func (s *<<.StorageClassPrefix>>Storage) GetLatest<<.TypeName>>(ctx context.Context, id uuid.UUID) (*<<.FullyQualifiedTypeName>>, error) {
	<<- if .WithCache >>
	var err error
	sentinel := cache.NoLockSentinel
	<<- if .FollowerReads >>
	conflict := cache.GenerateTombstoneSentinel()
	<<- end >>
	if s.cm != nil {
		var cachedObj *<<.FullyQualifiedTypeName>>
		<<- if .FollowerReads >>
		cachedObj, conflict, sentinel, err = cache.GetItemFromCacheWithModifiedKey[<<.FullyQualifiedTypeName>>](ctx, *s.cm, s.cm.N.GetKeyNameWithID(<<.FullyQualifiedTypeName>>KeyID, id), s.cm.N.GetKeyNameWithID(<<.TypePackageNameWithDot>>IsModifiedKeyID, id), true)
		<<- else >>
		cachedObj, _, sentinel, err = cache.GetItemFromCache[<<.FullyQualifiedTypeName>>](ctx, *s.cm, s.cm.N.GetKeyNameWithID(<<.FullyQualifiedTypeName>>KeyID, id), true)
		<<- end >>
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[GetLatest<<.TypeName>>] error reading from local cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[GetLatest<<.TypeName>>] error reading from local cache: %v", err)
			}
		} else if cachedObj != nil {
			return cachedObj, nil
		}
	}
	<<- end >>
	const q = "SELECT <<.SelectColumns>> FROM <<.Table>> WHERE id=$1 AND deleted='0001-01-01 00:00:00' ORDER BY version DESC LIMIT 1;"
	var obj <<.FullyQualifiedTypeName>>
	<<- if .WithCache >>
	if err := s.db.GetContextWithDirty(ctx, "GetLatest<<.TypeName>>", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
	<<- else >>
	if err := s.db.GetContext(ctx, "GetLatest<<.TypeName>>", &obj, q, id); err != nil {
	<<- end >>

		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "<<.TypeName>> %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	<<- if .WithCache >>
	if s.cm != nil {
		cache.SaveItemToCache(ctx, *s.cm, obj, sentinel, false, nil)
	}
	<<- end >>
	return &obj, nil
}

<<- else >>

// Get<<.TypeName>> loads a <<.TypeName>> by ID
func (s *<<.StorageClassPrefix>>Storage) Get<<.TypeName>>(ctx context.Context, id uuid.UUID<<- if .AccessPrimaryDbOption ->>, accessPrimaryDBOnly bool<<- end ->>) (*<<.FullyQualifiedTypeName>>, error) {
	<<- if .WithCache >>
	return cache.ServerGetItem(ctx, s.cm, id, <<.FullyQualifiedTypeName>>KeyID, <<.TypePackageNameWithDot>>IsModifiedKeyID,
		func(id uuid.UUID, conflict cache.Sentinel, obj *<<.FullyQualifiedTypeName>>) error {
			const q = "SELECT <<.SelectColumns>> FROM <<.Table>> WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

			if err := s.db.GetContextWithDirty(ctx, "Get<<.TypeName>>", obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ucerr.Friendlyf(err, "<<.TypeName>> %v not found", id)
				}
				return ucerr.Wrap(err)
			}
			return nil
		})
	<<- else >>
	const q = "SELECT <<.SelectColumns>> FROM <<.Table>> WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj <<.FullyQualifiedTypeName>>
	<<- if .AccessPrimaryDbOption >>
	useReplica := featureflags.IsEnabledGlobally(ctx, featureflags.ReadFromReadReplica)
	if err := s.db.GetContextWithDirty(ctx, "Get<<.TypeName>>", &obj, q, accessPrimaryDBOnly || !useReplica, id); err != nil {
	<<- else >>
	if err := s.db.GetContext(ctx, "Get<<.TypeName>>", &obj, q, id); err != nil {
	<<- end >>
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "<<.TypeName>> %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
	<<- end >>
}

<<- if .GetByName >>

// get<<.TypeName>>ByColumns loads a <<.TypeName>> using the provided column names and values as a WHERE clause
func (s *<<.StorageClassPrefix>>Storage) get<<.TypeName>>ByColumns(ctx context.Context, secondaryKey cache.Key, columnNames []string, columnValues []any) (*<<.FullyQualifiedTypeName>>, error) {
	<<- if .WithCache >>
	var err error
	<<- if .FollowerReads >>
	conflict := cache.GenerateTombstoneSentinel()
	<<- end >>
	if s.cm != nil {
		var cachedObj *<<.FullyQualifiedTypeName>>
		<<- if .FollowerReads >>
		mkey := s.cm.N.GetKeyNameStatic(<<.FullyQualifiedTypeName>>CollectionKeyID)
		<<- if .CachePages >>
		// If there is IsModifiedCollectionKey use that instead of the GlobalCollectionKey
		if s.cm.N.GetKeyNameWithString(<<.TypePackageNameWithDot>>IsModifiedCollectionKeyID, string(s.cm.N.GetKeyNameStatic(<<.FullyQualifiedTypeName>>CollectionKeyID))) != "" {
			mkey = s.cm.N.GetKeyNameWithString(<<.TypePackageNameWithDot>>IsModifiedCollectionKeyID, string(s.cm.N.GetKeyNameStatic(<<.FullyQualifiedTypeName>>CollectionKeyID)))
		}
		<<- end >>
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[<<.FullyQualifiedTypeName>>](ctx, *s.cm, secondaryKey, mkey, false)
		<<- else >>
		cachedObj, _, _, err = cache.GetItemFromCache[<<.FullyQualifiedTypeName>>](ctx, *s.cm, secondaryKey, false)
		<<- end >>
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[get<<.TypeName>>ByColumns] error reading from local cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[get<<.TypeName>>ByColumns] error reading from local cache: %v", err)
			}
		} else if cachedObj != nil {
			return cachedObj, nil
		}
	}
	<<- end >>
	args := ""
	for i := range columnNames {
		if i > 0 {
			args += " AND "
		}
		args += fmt.Sprintf("%s=$%d", columnNames[i], i+1)
	}
	q := fmt.Sprintf("SELECT <<.SelectColumns>> FROM <<.Table>> WHERE %s AND deleted='0001-01-01 00:00:00';", args)

	var obj <<.FullyQualifiedTypeName>>
	<<- if .WithCache >>
	if err := s.db.GetContextWithDirty(ctx, "Get<<.TypeName>>ForName", &obj, q, cache.IsTombstoneSentinel(string(conflict)), columnValues...); err != nil {
	<<- else >>
	if err := s.db.GetContext(ctx, "Get<<.TypeName>>ForName", &obj, q, columnValues...); err != nil {
	<<- end >>
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "<<.TypeName>> %v not found", columnValues)
		}
		return nil, ucerr.Wrap(err)
	}

	<<- if .WithCache >>
	// Trigger an async cache update
	if s.cm != nil {
		go func() {
			if _, err := s.Get<<.TypeName>>(context.Background(), obj.ID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					uclog.Debugf(ctx, "Object retrieval failed for <<.TypeName>> with ID %v: not found in database", obj.ID)
				} else {
					uclog.Errorf(ctx, "Error retrieving <<.TypeName>> with ID %v from cache: %v", obj.ID, err)
				}
			}
		}()
	}
	<<- end >>
	return &obj, nil
}
<<- end >>

// Get<<.TypeName>>SoftDeleted loads a <<.TypeName>> by ID iff it's soft-deleted
func (s *<<.StorageClassPrefix>>Storage) Get<<.TypeName>>SoftDeleted(ctx context.Context, id uuid.UUID<<- if .AccessPrimaryDbOption ->>, accessPrimaryDBOnly bool<<- end ->>) (*<<.FullyQualifiedTypeName>>, error) {
	<<- if .WithCache >>
	var err error
	conflict := cache.GenerateTombstoneSentinel()

	if s.cm != nil {
		var cachedObj *<<.FullyQualifiedTypeName>>
		cachedObj, conflict, _, err = cache.GetItemFromCacheWithModifiedKey[<<.FullyQualifiedTypeName>>](ctx, *s.cm, s.cm.N.GetKeyNameWithID(<<.FullyQualifiedTypeName>>KeyID, id), s.cm.N.GetKeyNameWithID(<<.TypePackageNameWithDot>>IsModifiedKeyID, id), false)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				uclog.Warningf(ctx, "[Get<<.TypeName>>SoftDeleted] error reading from cache: %v", err)
			} else {
				uclog.Errorf(ctx, "[Get<<.TypeName>>SoftDeleted] error reading from cache: %v", err)
			}
		} else if cachedObj != nil {
			return nil, ucerr.Friendlyf(err, "soft-deleted <<.TypeName>> %v not found", id)
		}
	}
	<<- end >>
	const q = "SELECT <<.SelectColumns>> FROM <<.Table>> WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj <<.FullyQualifiedTypeName>>
	<<- if .WithCache >>
	if err := s.db.GetContextWithDirty(ctx, "Get<<.TypeName>>SoftDeleted", &obj, q, cache.IsTombstoneSentinel(string(conflict)), id); err != nil {
	<<- else if .AccessPrimaryDbOption >>
	useReplica := featureflags.IsEnabledGlobally(ctx, featureflags.ReadFromReadReplica)
	if err := s.db.GetContextWithDirty(ctx, "Get<<.TypeName>>SoftDeleted", &obj, q, accessPrimaryDBOnly || !useReplica, id); err != nil {
	<<- else >>
	if err := s.db.GetContext(ctx, "Get<<.TypeName>>SoftDeleted", &obj, q, id); err != nil {
	<<- end >>
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted <<.TypeName>> %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}
<<- if .MultiGet >>

// Get<<.TypeName>>sForIDs loads multiple <<.TypeName>> for a given list of IDs
func (s *<<.StorageClassPrefix>>Storage) Get<<.TypeName>>sForIDs(ctx context.Context, errorOnMissing bool, << if .AccessPrimaryDbOption >>accessPrimaryDBOnly bool, << end >>ids ...uuid.UUID) ([]<<.FullyQualifiedTypeName>>, error) {
	items := make([]<<.FullyQualifiedTypeName>>, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	<<- if .AccessPrimaryDbOption >>
	useReplica := featureflags.IsEnabledGlobally(ctx, featureflags.ReadFromReadReplica)
	dirty := accessPrimaryDBOnly || !useReplica
	<<- else >>
	dirty := true
	<<- end >>

	<<- if .WithCache >>
	if len(ids) == 0 {
		return items, nil
	}

	if len(ids) != missed.Size() {
		// We have duplicate IDs in the list
		ids = missed.Items()
	}

	var cachedItemsCount, dbItemsCount int
	sentinelsMap := make(map[uuid.UUID]cache.Sentinel)
	if s.cm != nil {
		keys := make([]cache.Key, 0, len(ids))
		modKeys := make([]cache.Key, 0, len(ids))
		locks := make([]bool, 0, len(keys))
		for _, id := range ids {
			locks = append(locks, true)
			keys = append(keys, s.cm.N.GetKeyNameWithID(<<.FullyQualifiedTypeName>>KeyID, id))
			modKeys = append(modKeys, s.cm.N.GetKeyNameWithID(<<.TypePackageNameWithDot>>IsModifiedKeyID, id))
		}
		cachedItems, sentinels, cdirty, err := cache.GetItemsFromCache[<<.FullyQualifiedTypeName>>](ctx, *s.cm, keys, modKeys, locks)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		for i, item := range cachedItems {
			if item != nil {
				items = append(items, *item)
				missed.Evict(item.ID)
				cachedItemsCount++
			} else if sentinels != nil { // sentinels array will be nil if we are not using a cache
				sentinelsMap[ids[i]] = sentinels[i]
			}
		}
		dirty = cdirty
	}
	<<- end >>
	if missed.Size() > 0 {
		itemsFromDB, err := s.get<<.TypeNamePlural>>HelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
		<<- if .WithCache >>
		dbItemsCount = len(itemsFromDB)
		if s.cm != nil && len(sentinelsMap) > 0 {
			for _, item := range itemsFromDB {
				cache.SaveItemToCache(ctx, *s.cm, item, sentinelsMap[item.ID], false, nil)
			}
		}
		<<- end >>
	}
	<<- if .WithCache >>
	uclog.Verbosef(ctx, "Get<<.TypeName>>Map: returning %d <<.TypeName>>. from DB: %d from cache %d", len(items), dbItemsCount, cachedItemsCount)
	<<- end >>

	return items, nil
}

// get<<.TypeNamePlural>>HelperForIDs loads multiple <<.TypeName>> for a given list of IDs from the DB
func (s *<<.StorageClassPrefix>>Storage) get<<.TypeNamePlural>>HelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]<<.FullyQualifiedTypeName>>, error) {
	const q = "SELECT <<.SelectColumns>> FROM <<.Table>> WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []<<.FullyQualifiedTypeName>>
	if err := s.db.SelectContextWithDirty(ctx, "Get<<.TypeNamePlural>>ForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested <<.TypeNamePlural>>  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

<<- end >>
<<- end >>
<<- end >>
<<- if .List >>

// List<<.TypeNamePlural>>Paginated loads a paginated list of <<.TypeNamePlural>> for the specified paginator settings
func (s *<<.StorageClassPrefix>>Storage) List<<.TypeNamePlural>>Paginated(ctx context.Context, p pagination.Paginator<<- if .AccessPrimaryDbOption ->>, accessPrimaryDBOnly bool<<- end ->>) ([]<<.FullyQualifiedTypeName>>, *pagination.ResponseFields, error) {
	return s.listInner<<.TypeNamePlural>>Paginated(ctx, p, false<<- if .AccessPrimaryDbOption ->>, accessPrimaryDBOnly<<- end ->>)
}

// listInner<<.TypeNamePlural>>Paginated loads a paginated list of <<.TypeNamePlural>> for the specified paginator settings
func (s *<<.StorageClassPrefix>>Storage) listInner<<.TypeNamePlural>>Paginated(ctx context.Context, p pagination.Paginator, forceDBRead bool<<- if .AccessPrimaryDbOption ->>, accessPrimaryDBOnly bool<<- end ->>) ([]<<.FullyQualifiedTypeName>>, *pagination.ResponseFields, error) {
	<<- if .WithCache >>
	var lkey, ckey cache.Key
	sentinel := cache.NoLockSentinel
	<<- if .FollowerReads >>
	conflict := cache.GenerateTombstoneSentinel()
	<<- end >>
	<<- if .CachePages >>
	cachable := p.IsCachable()
	<<- else >>
	cachable := p.IsCachable() && p.GetCursor() == ""
	<<- end >>

	<<- if .FollowerReads >>
	if s.cm != nil { // We need the check the cache even if the result is not cachable to see if a stale read is allowed
	<<- else >>
	if s.cm != nil && cachable {
	<<- end >>
		lkey = s.cm.N.GetKeyNameStatic(<<.FullyQualifiedTypeName>>CollectionKeyID)
		<<- if .CachePages >>
		ckey = s.cm.N.GetKeyName(<<.FullyQualifiedTypeName>>CollectionPageKeyID, []string{string(p.GetCursor()), fmt.Sprintf("%v", p.GetLimit())})
		<<- else >>
		ckey = lkey
		<<- end >>
		var err error
		var v *[]<<.FullyQualifiedTypeName>>
		<<- if .CachePages >>
		partialHit := false
		// We only try to fetch the page of data if we could use it
		if cachable && !forceDBRead {
			v, _, sentinel, partialHit, err = cache.GetItemsArrayFromCache[<<.FullyQualifiedTypeName>>](ctx, *s.cm, ckey, false)
			if err != nil {
				return nil, nil, ucerr.Wrap(err)
			}
		}
		<<- if .FollowerReads >>
		// If the page is not in the cache or if request is not cachable, we need to check the global collection cache to see if we can use follower reads
		if v == nil || !cachable {
			mkey := lkey
			// If there is IsModifiedCollectionKey use that instead of the GlobalCollectionKey
			if s.cm.N.GetKeyNameWithString(<<.TypePackageNameWithDot>>IsModifiedCollectionKeyID, string(s.cm.N.GetKeyNameStatic(<<.FullyQualifiedTypeName>>CollectionKeyID))) != "" {
				mkey = s.cm.N.GetKeyNameWithString(<<.TypePackageNameWithDot>>IsModifiedCollectionKeyID, string(s.cm.N.GetKeyNameStatic(<<.FullyQualifiedTypeName>>CollectionKeyID)))
			}
			_, conflict, _, _, err = cache.GetItemsArrayFromCache[<<.FullyQualifiedTypeName>>](ctx, *s.cm, mkey, false)
			if err != nil {
				return nil, nil, ucerr.Wrap(err)
			}
		}
		<<- end >>
		<<- else >>
		<<- if .FollowerReads >>
		v, conflict, sentinel, _, err = cache.GetItemsArrayFromCache[<<.FullyQualifiedTypeName>>](ctx, *s.cm, ckey, cachable)
		if err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
		<<- else >>
		v, _, sentinel, _, err = cache.GetItemsArrayFromCache[<<.FullyQualifiedTypeName>>](ctx, *s.cm, ckey, true)
		if err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
		<<- end >>
		<<- end >>
		<<- if .CachePages >>
		if cachable {
			if v != nil {
				if partialHit {
					uclog.Verbosef(ctx, "Partial cache hit for <<.FullyQualifiedTypeName>> launching async refresh")
					go func(ctx context.Context) {
						if _, _, err := s.listInner<<.TypeNamePlural>>Paginated(ctx, p, true<<- if .AccessPrimaryDbOption ->>, accessPrimaryDBOnly<<- end ->>); err != nil { // lint: ucpagination-safe
							uclog.Errorf(ctx, "Error fetching <<.FullyQualifiedTypeName>> async for cache update: %v", err)
						}
					}(context.WithoutCancel(ctx))
				}

				v, respFields := pagination.ProcessResults(*v, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())
				return v, &respFields, nil
			}
			sentinel, err = cache.TakeGlobalCollectionLock(ctx, cache.Read, *s.cm, <<.FullyQualifiedTypeName>>{})
			if err != nil {
				uclog.Errorf(ctx, "Error taking global collection lock for <<.TypeNamePlural>>: %v", err)
			} else if sentinel != cache.NoLockSentinel {
				defer cache.ReleasePerItemCollectionLock(ctx, *s.cm, []cache.Key{lkey}, <<.FullyQualifiedTypeName>>{}, sentinel)
			}
		}
		<<- else >>
		if cachable && v != nil {
			v, respFields := pagination.ProcessResults(*v, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())
			return v, &respFields, nil
		}
		<<- end >>
	}
	<<- end >>
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT <<.SelectColumns>> FROM (SELECT <<.SelectColumns>> FROM <<.Table>> WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []<<.FullyQualifiedTypeName>>
	<<- if .WithCache >>
	if err := s.db.SelectContextWithDirty(ctx, "List<<.TypeNamePlural>>Paginated", &objsDB, q, cache.IsTombstoneSentinel(string(conflict)), queryFields...); err != nil {
	<<- else if .AccessPrimaryDbOption >>
	useReplica := featureflags.IsEnabledGlobally(ctx, featureflags.ReadFromReadReplica)
	if err := s.db.SelectContextWithDirty(ctx, "List<<.TypeNamePlural>>Paginated", &objsDB, q, accessPrimaryDBOnly || !useReplica, queryFields...); err != nil {
	<<- else >>
	if err := s.db.SelectContext(ctx, "List<<.TypeNamePlural>>Paginated", &objsDB, q, queryFields...); err != nil {
	<<- end >>
		return nil, nil, ucerr.Wrap(err)
	}
	objs, respFields := pagination.ProcessResults(objsDB, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())
	if respFields.HasNext {
		if err := p.ValidateCursor(respFields.Next); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	if respFields.HasPrev {
		if err := p.ValidateCursor(respFields.Prev); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}
	<<- if .WithCache >>
	<<- if .CachePages >>
	if s.cm != nil && cachable {
	<<- else >>
	if s.cm != nil && cachable && !respFields.HasNext && !respFields.HasPrev { /* only cache single page collections */
	<<- end >>
		cache.SaveItemsToCollection(ctx, *s.cm, <<.FullyQualifiedTypeName>>{}, objsDB, lkey, ckey, sentinel, true)
	}
	<<- end >>

	return objs, &respFields, nil
}
<<- if .HasUserID >>

// List<<.TypeNamePlural>>ForUserID loads the list of <<.TypeNamePlural>> with a matching UserID field
func (s *<<.StorageClassPrefix>>Storage) List<<.TypeNamePlural>>ForUserID(ctx context.Context, userID uuid.UUID) ([]<<.FullyQualifiedTypeName>>, error) {
	const q = "SELECT <<.SelectColumns>> FROM <<.Table>> WHERE user_id=$1 AND deleted='0001-01-01 00:00:00';"
	var objs []<<.FullyQualifiedTypeName>>
	if err := s.db.SelectContext(ctx, "List<<.TypeNamePlural>>ForUserID", &objs, q, userID); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return objs, nil
}
<<- end >>
<<- end >>
<<- if .NonPaginatedList >>

// List<<.TypeNamePlural>>NonPaginated loads a <<.TypeName>> up to a limit of 10 pages
func (s *<<.StorageClassPrefix>>Storage) List<<.TypeNamePlural>>NonPaginated(ctx context.Context) ([]<<.FullyQualifiedTypeName>>, error) {
	<<- if .WithCache >>
	<<- if not .CachePages >>
	sentinel := cache.NoLockSentinel
	if s.cm != nil {
		var err error
		var v *[]<<.FullyQualifiedTypeName>>
		v, _, sentinel, _, err = cache.GetItemsArrayFromCache[<<.FullyQualifiedTypeName>>](ctx, *s.cm, s.cm.N.GetKeyNameStatic(<<.FullyQualifiedTypeName>>CollectionKeyID), true)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if v != nil {
			return *v, nil
		}
	}
	<<- end >>
	<<- end >>
	pager, err := <<.TypePackageNameWithDot>>New<<.TypeName>>PaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	objs := make([]<<.FullyQualifiedTypeName>>, 0)

	pageCount := 0
	for {
		objRead, respFields, err := s.List<<.TypeNamePlural>>Paginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		objs = append(objs, objRead...)
		pageCount++
		<<- if .WithCache >>
		<<- if not .CachePages >>
		if s.cm != nil {
			// Save individual items to cache under their own primary keys (optional)
			cache.SaveItemsFromCollectionToCache(ctx, *s.cm, objRead, sentinel)
		}
		<<- end >>
		<<- end >>
		if !pager.AdvanceCursor(*respFields) {
			break
		}
		if pageCount >= 10 {
			return nil, ucerr.Errorf("List<<.TypeNamePlural>>NonPaginated exceeded max page count of 10")
		}
	}
	<<- if .WithCache >>
	<<- if not .CachePages >>
	if s.cm != nil {
		ckey := s.cm.N.GetKeyNameStatic(<<.FullyQualifiedTypeName>>CollectionKeyID)
		cache.SaveItemsToCollection(ctx, *s.cm, <<.FullyQualifiedTypeName>>{}, objs, ckey, ckey, sentinel, true)
	}
	<<- end >>
	<<- end >>
	return objs, nil
}
<<- end >>
<<- if .Save >>

// Save<<.TypeName>> saves a <<.TypeName>>
func (s *<<.StorageClassPrefix>>Storage) Save<<.TypeName>>(ctx context.Context, obj *<<.FullyQualifiedTypeName>>) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	<<- if .HasPreSave >>
	if err := s.preSave<<.TypeName>>(ctx, obj); err != nil {
		return ucerr.Wrap(err)
	}
	<<- end >>
	<<- if .WithCache >>
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, <<.FullyQualifiedTypeName>>KeyID, <<- if .HasAdditionalSaveKeys >> s.additionalSaveKeysFor<<.TypeName>>(obj)<<- else >> nil<<- end >>, func(i *<<.FullyQualifiedTypeName>>) error {
		return ucerr.Wrap(s.saveInner<<.TypeName>>(ctx, obj))
	}))
	<<- else >>
	return ucerr.Wrap(s.saveInner<<.TypeName>>(ctx, obj))
	<<- end >>
}

// Save<<.TypeName>> saves a <<.TypeName>>
func (s *<<.StorageClassPrefix>>Storage) saveInner<<.TypeName>>(ctx context.Context, obj *<<.FullyQualifiedTypeName>>) error {
	<<- if .HasVersion >>
	// this query has three basic parts
	// 1) INSERT INTO <<.Table>> is used for create only ... any updates will fail with a CONFLICT on (id, deleted)
	// 2) in that case, WHERE will take over to chose the correct row (if any) to update. This includes a check that obj.Version ($3)
	//    matches the _version currently in the database, so that we aren't writing stale data. If this fails, sql.ErrNoRows is returned.
	// 3) if the WHERE matched a row (including version check), the UPDATE will set the new values including $[max] which is newVersion,
	//    which is set to the current version + 1. This is returned in the RETURNING clause so that we can update obj.Version with the new value.
	newVersion := obj.Version + 1
	const q = "INSERT INTO <<.Table>> (<<.Columns>>) VALUES (<<.Values>>) ON CONFLICT (<<.OnConflict>>) DO UPDATE SET <<.Updates>> WHERE (<<.ImmutableWhere>>) RETURNING created, updated, _version; /* allow-multiple-target-use no-match-cols-vals */"
	<<- else >>
	const q = "INSERT INTO <<.Table>> (<<.Columns>>) VALUES (<<.Values>>) ON CONFLICT (<<.OnConflict>>) DO UPDATE SET <<.Updates>> WHERE (<<.ImmutableWhere>>) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	<<- end >>
	if err := s.db.GetContext(ctx, "Save<<.TypeName>>", obj, q, <<.Fields>>); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "<<.TypeName>> %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}
<<- if .PriorSave >>

// PriorVersionSave<<.TypeName>> saves an older version of <<.TypeName>>
func (s *<<.StorageClassPrefix>>Storage) PriorVersionSave<<.TypeName>>(ctx context.Context, obj *<<.FullyQualifiedTypeName>>) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInner<<.TypeName>>(ctx, obj))
}
<<- end >>
<<- end >>
<<- if not .HasVersion >>
<<- if .IncludeInsertOnly >>

// Insert<<.TypeName>> inserts a <<.TypeName>> without resolving conflict with existing rows
func (s *<<.StorageClassPrefix>>Storage) Insert<<.TypeName>>(ctx context.Context, obj *<<.FullyQualifiedTypeName>>) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	<<- if .HasPreSave >>
	if err := s.preSave<<.TypeName>>(ctx, obj); err != nil {
		return ucerr.Wrap(err)
	}
	<<- end >>
	<<- if .WithCache >>
	return ucerr.Wrap(cache.CreateItemServer(ctx, s.cm, obj, <<.FullyQualifiedTypeName>>KeyID, <<- if .HasAdditionalSaveKeys >> s.additionalSaveKeysFor<<.TypeName>>(obj)<<- else >> nil<<- end >>, func(i *<<.FullyQualifiedTypeName>>) error {
		return ucerr.Wrap(s.insertInner<<.TypeName>>(ctx, obj))
	}))
	<<- else >>
	return ucerr.Wrap(s.insertInner<<.TypeName>>(ctx, obj))
	<<- end >>
}

// insertInner<<.TypeName>> inserts a <<.TypeName>> without resolving conflict with existing rows
func (s *<<.StorageClassPrefix>>Storage) insertInner<<.TypeName>>(ctx context.Context, obj *<<.FullyQualifiedTypeName>>) error {
	const q = "INSERT INTO <<.Table>> (<<.Columns>>) VALUES (<<.Values>>) RETURNING id, created, updated;"
	if err := s.db.GetContext(ctx, "Insert<<.TypeName>>", obj, q, <<.Fields>>); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
<<- end >>
<<- end >>
<<- if .Delete >>
<<- if .Versioned >>

// Delete<<.TypeName>>ByVersion soft-deletes a <<.TypeName>> which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s <<.StorageClassPrefix>>Storage) Delete<<.TypeName>>ByVersion(ctx context.Context, objID uuid.UUID, version int) error {
	<<- if .HasPreDelete >>
	if err := s.preDelete<<.TypeName>>(ctx, objID, version); err != nil {
		return ucerr.Wrap(err)
	}
	<<- end >>
	<<- if .WithCache >>
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[<<.FullyQualifiedTypeName>>](ctx, *s.cm, s.cm.N.GetKeyNameWithID(<<.FullyQualifiedTypeName>>KeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := <<$.FullyQualifiedTypeName>>{<<$.BaseModel>>Model: ucdb.New<<$.BaseModel>>WithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[<<.FullyQualifiedTypeName>>](ctx, *s.cm, *obj, sentinel)
	}
	<<- end >>
	const q = "UPDATE <<.Table>> SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND version=$2 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	_, err := s.db.ExecContext(ctx, "Delete<<.TypeName>>ByVersion", q, objID, version)
	return ucerr.Wrap(err)
}

// DeleteAll<<.TypeName>>Versions soft-deletes all versions of a <<.TypeName>>
func (s <<.StorageClassPrefix>>Storage) DeleteAll<<.TypeName>>Versions(ctx context.Context, objID uuid.UUID) error {
	<<- if .HasPreDelete >>
	if err := s.preDelete<<.TypeName>>(ctx, objID, -1); err != nil {
		return ucerr.Wrap(err)
	}
	<<- end >>
	<<- if .WithCache >>
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[<<.FullyQualifiedTypeName>>](ctx, *s.cm, s.cm.N.GetKeyNameWithID(<<.FullyQualifiedTypeName>>KeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := <<$.FullyQualifiedTypeName>>{<<$.BaseModel>>Model: ucdb.New<<$.BaseModel>>WithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[<<.FullyQualifiedTypeName>>](ctx, *s.cm, *obj, sentinel)
	}
	<<- end >>
	const q = "UPDATE <<.Table>> SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	_, err := s.db.ExecContext(ctx, "DeleteAll<<.TypeName>>Versions", q, objID)
	return ucerr.Wrap(err)
}
<<- else >>

// Delete<<.TypeName>> soft-deletes a <<.TypeName>> which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *<<.StorageClassPrefix>>Storage) Delete<<.TypeName>>(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInner<<.TypeName>>(ctx, objID, false))
}

// deleteInner<<.TypeName>> soft-deletes a <<.TypeName>> which is currently alive
func (s *<<.StorageClassPrefix>>Storage) deleteInner<<.TypeName>>(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	<<- if .HasPreDelete >>
	if err := s.preDelete<<.TypeName>>(ctx, objID, wrappedDelete); err != nil {
		return ucerr.Wrap(err)
	}
	<<- end >>
	<<- if .WithCache >>
	if s.cm != nil {
		obj, _, _, err := cache.GetItemFromCache[<<.FullyQualifiedTypeName>>](ctx, *s.cm, s.cm.N.GetKeyNameWithID(<<.FullyQualifiedTypeName>>KeyID, objID), false)
		if err != nil {
			return ucerr.Wrap(err)
		}

		objBase := <<$.FullyQualifiedTypeName>>{<<$.BaseModel>>Model: ucdb.New<<$.BaseModel>>WithID(objID)}
		if obj == nil {
			obj = &objBase
		}
		sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, *obj)
		if err != nil {
			uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
			return ucerr.Wrap(err)
		}
		// This generates an extra invalidation for items that don't exist
		defer cache.DeleteItemFromCache[<<.FullyQualifiedTypeName>>](ctx, *s.cm, *obj, sentinel)
	}
	<<- end >>
	const q = "UPDATE <<.Table>> SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "Delete<<.TypeName>>", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting <<.TypeName>> %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "<<.TypeName>> %v not found", objID)
	}
	return nil
}
<<- end >>

<<- if .WithCache >>

// FlushCacheFor<<.TypeName>> flushes cache for <<.TypeName>>. It may flush a larger scope then
func (s *<<.StorageClassPrefix>>Storage) FlushCacheFor<<.TypeName>>(ctx context.Context, objID uuid.UUID) error {
	if s.cm == nil {
		return nil
	}
	return ucerr.Wrap(s.cm.Flush(ctx, "<<.TypeName>>"))
}
<<- end >>
<<- end >>
`

var columnTemplateString = `// NOTE: automatically generated file -- DO NOT EDIT

package << .DatabaseName >>

func init() {
	UsedColumns["<< .Table >>"] = []string{
		<<- range $i, $col := .ColumnNames >>
		"<< $col >>",
		<<- end >>
	}
}
`
