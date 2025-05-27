package analyzer

import (
	"go/ast"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"userclouds.com/infra/ucerr"
)

// JSONTagAnalyzer checks our sources to make sure we annotate every struct field with a JSON tag (or no fields at all)
var JSONTagAnalyzer = &analysis.Analyzer{
	Name:     "jsontags",
	Doc:      "checks structs with json tags to make sure they all have them (or none)",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runTags("json"),
}

// DBTagAnalyzer checks that structs have all or none of their fields annotated
// with sqlx's DB tag.
var DBTagAnalyzer = &analysis.Analyzer{
	Name:     "dbtags",
	Doc:      "checks structs with db tags to make sure they all have them (or none)",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runTags("db"),
}

// YAMLTagAnalyzer checks that structs have all or none of their fields annotated
// with a YAML tag.
var YAMLTagAnalyzer = &analysis.Analyzer{
	Name:     "yamltags",
	Doc:      "checks structs with yaml tags to make sure they all have them (or none)",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runTags("yaml"),
}

func runTags(tag string) func(*analysis.Pass) (any, error) {
	return func(pass *analysis.Pass) (any, error) {
		insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
		nodeFilter := []ast.Node{(*ast.StructType)(nil)}
		var overallErr error // communicate errors from the AST-walking code
		insp.Preorder(nodeFilter, func(n ast.Node) {
			// short-circuit if we've hit a parse error already
			if overallErr != nil {
				return
			}
			s := n.(*ast.StructType) // safe because of nodeFilter
			var (
				exported int
				untagged int
			)
			seen := make(map[string]struct{})
			for _, f := range s.Fields.List {
				var exportedIdents int
				for _, ident := range f.Names {
					if ident.IsExported() {
						untagged++
						exported++
						exportedIdents++
					}
				}
				if f.Tag == nil {
					continue
				}
				// reflect.StructTag requires this unquoted, but ast gives it to us
				// quoted.
				v, err := strconv.Unquote(f.Tag.Value)
				if err != nil {
					overallErr = ucerr.Errorf("%v: failed to unquote struct tag", pass.Fset.Position(f.Tag.Pos()))
					return
				}
				tags := reflect.StructTag(v)
				if t, ok := tags.Lookup(tag); ok {
					// if this is embedded (has 0 names) then it's basically a pass-through and we should
					// just validate the embedded struct (which will happen automatically)
					if len(f.Names) == 0 {
						continue
					}

					if exportedIdents == 0 {
						pass.Reportf(f.Pos(), "unexported fields can't have %s tag", tag)
						continue
					}
					untagged -= exportedIdents // applicable even if tags are invalid
					if t == "-" {
						// no further validation required
						continue
					}
					parts := strings.SplitN(t, ",", 1) // split key from options (e.g., omitempty)
					var key string
					if len(parts) > 0 {
						key = parts[0]
					}
					if key == "" {
						pass.Reportf(f.Tag.Pos(), "can't use empty string as %s key", tag)
						continue
					}
					if _, ok := seen[t]; ok {
						pass.Reportf(f.Tag.Pos(), "duplicate %s tag %q", tag, t)
						continue
					}
					seen[t] = struct{}{}
					if len(f.Names) > 1 {
						pass.Reportf(f.Pos(), "multiple fields can't share %s key %q", tag, key)
					}
				}
			}
			if untagged != exported && untagged != 0 {
				// unclear whether the tagged or untagged fields are wrong, use struct position
				pass.Reportf(s.Pos(), "all or no exported fields must have the %q tag", tag)
			}
		})
		return nil, ucerr.Wrap(overallErr)
	}
}
