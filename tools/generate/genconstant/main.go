package genconstant

import (
	"context"
	"fmt"
	"go/constant"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"userclouds.com/infra/uclog"
	"userclouds.com/tools/generate"
)

// this is easier than passing strings.ToLower into template renderer as a function :)
type constVal struct {
	Name  string
	Value string
}

type constants []constVal

// Len implements sort.Interface
func (c constants) Len() int {
	return len(c)
}

// Swap implements sort.Interface
func (c constants) Swap(left, right int) {
	tmp := c[left]
	c[left] = c[right]
	c[right] = tmp
}

// Less implements sort.Interface
func (c constants) Less(left, right int) bool {
	return c[left].Name < c[right].Name
}

type data struct {
	Package        string
	TypeName       string
	TypeNamePlural string
	AreInts        bool

	Constants      constants
	ValidConstants constants // used for Validate, ignores *Unknown and *Invalid
}

// Run implements the generator interface
func Run(ctx context.Context, p *packages.Package, path string, args ...string) {
	tn := args[1]

	d := data{
		TypeName:       tn,
		TypeNamePlural: generate.GetPluralName(tn),
		Package:        p.Name,
		AreInts:        true,
	}

	for id, obj := range p.TypesInfo.Defs {
		if id.Name == tn {
			continue
		}
		if strings.HasPrefix(id.Name, tn) {
			if obj == nil {
				uclog.Fatalf(ctx, "unexpected nil object in TypesInfo.Defs for %s", id.Name)
			}

			c, ok := obj.(*types.Const)
			if !ok {
				continue
			}

			k := constVal{
				Name:  id.Name,
				Value: strings.ToLower(strings.TrimPrefix(id.Name, tn)),
			}

			if c.Val().Kind() == constant.String {
				k.Value = constant.StringVal(c.Val())
				d.AreInts = false
			}

			d.Constants = append(d.Constants, k)

			if !strings.HasSuffix(id.Name, "Unknown") && !strings.HasSuffix(id.Name, "Invalid") {
				d.ValidConstants = append(d.ValidConstants, k)
			}
		}
	}

	// the ordering needs to be stable to prevent a bunch of annoying code churn,
	// and the ordering in the packages parser isn't guaranteed (and in fact isn't) stable
	sort.Sort(d.Constants)
	sort.Sort(d.ValidConstants)

	fn := fmt.Sprintf("%s/%s_constant_generated.go", path, strings.ToLower(tn))
	generate.WriteFileIfChanged(ctx, fn, tempString, d)
}

var tempString = `// NOTE: automatically generated file -- DO NOT EDIT

package <<.Package>>

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t << .TypeName >>) MarshalText() ([]byte, error) {
	switch t {
	<<- range .Constants >>
	case << .Name >>:
		return []byte("<< .Value >>"), nil
	<<- end >>
	default:
		<<- if .AreInts >>
		return nil, ucerr.Friendlyf(nil, "unknown << .TypeName >> value '%d'", t)
		<<- else >>
		return nil, ucerr.Friendlyf(nil, "unknown << .TypeName >> value '%s'", t)
		<<- end >>
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *<< .TypeName >>) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	<<- range .Constants >>
	case "<< .Value >>":
		*t = << .Name >>
	<<- end >>
	default:
		return ucerr.Friendlyf(nil, "unknown << .TypeName >> value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *<< .TypeName >>) Validate() error {
	switch *t {
	<<- range .ValidConstants >>
	case << .Name >>:
		return nil
	<<- end >>
	default:
		<<- if .AreInts >>
		return ucerr.Friendlyf(nil, "unknown << .TypeName >> value '%d'", *t)
		<<- else >>
		return ucerr.Friendlyf(nil, "unknown << .TypeName >> value '%s'", *t)
		<<- end >>
	}
}

// Enum implements Enum
func (t << .TypeName >>) Enum() []any {
	return []any{
	<<- range .ValidConstants >>
		"<< .Value >>",
	<<- end >>
	}
}

// All<< .TypeNamePlural>> is a slice of all << .TypeName >> values
var All<< .TypeNamePlural>> = []<< .TypeName >>{
	<<- range .ValidConstants >>
	<< .Name >>,
	<<- end >>
}
<<- if .AreInts >>

// just here for easier debugging
func (t << .TypeName >>) String() string {
	bs, err := t.MarshalText()
	if err != nil {
		return err.Error()
	}
	return string(bs)
}
<<- end >>
`
