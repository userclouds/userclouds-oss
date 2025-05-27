package genstringconstenum

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"userclouds.com/tools/generate"
)

type data struct {
	Package string
	Type    string
	Values  []string
}

// Run implements the generator interface
func Run(ctx context.Context, p *packages.Package, path string, args ...string) {
	tn := args[1]
	data := data{
		Type:    tn,
		Package: p.Name,
	}

	for id, obj := range p.TypesInfo.Defs {
		if obj == nil {
			continue
		}

		// this should only apply to package-scoped consts
		if obj.Parent() != p.Types.Scope() {
			continue
		}

		ot := obj.Type().String()

		// and not to arrays
		if strings.HasPrefix(ot, "[]") || strings.HasPrefix(ot, "func(") {
			continue
		}

		// and not to itself in the type def
		if id.Name == tn {
			continue
		}

		// only to the type we care about
		subs := strings.Split(ot, ".")
		if subs[len(subs)-1] == tn {
			data.Values = append(data.Values, id.Name)
		}
	}

	sort.Strings(data.Values)

	fn := filepath.Join(path, fmt.Sprintf("%s_enum_generated.go", strings.ToLower(data.Type)))
	generate.WriteFileIfChanged(ctx, fn, templateString, data)
}

var templateString = `// NOTE: automatically generated file -- DO NOT EDIT

package << .Package >>

var all<< .Type >>Values = []<< .Type >>{
	<<- range $v := .Values >>
	<< $v >>,
	<<- end >>
}
`
