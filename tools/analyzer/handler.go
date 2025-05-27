package analyzer

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// HandlerAnalyzer checks to make sure we only use UTC times
var HandlerAnalyzer = &analysis.Analyzer{
	Name:     "handler",
	Doc:      "Checks HTTP handler config for appropriate trailing slashes",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runHandler,
}

func runHandler(pass *analysis.Pass) (any, error) {
	safeLines := findBypassComments(pass, "handle-safe")

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		c := n.(*ast.CallExpr)

		// bypass comment for eg. reverse proxy handling
		if safeLines[sForP(pass, c.Pos())] {
			return
		}

		// we're only looking for mux.Handle & mux.HandleFunc
		s, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}

		// eg. a package-level selectee doesn't matter
		id, ok := s.X.(*ast.Ident)
		if !ok {
			return
		}

		// if we don't have type info, can't help
		if pass.TypesInfo.Types[id].Type == nil {
			return
		}

		// only worry about ServeMux.Handle*
		typ := pass.TypesInfo.Types[id].Type.Underlying().String()
		if !strings.Contains(typ, "http.ServeMux") || strings.Contains(typ, "uchttp.ServeMux") {
			return
		}

		if s.Sel.Name == "HandleFunc" {
			// for HandleFunc, you never want trailing slashes, since golang will auto-add them
			b, ok := c.Args[0].(*ast.BasicLit)
			if !ok {
				// this would be rare but maybe?
				return
			}
			// trim off the first and last chars since BasicLit strings are literally "foo" or `foo`
			// and make sure that we aren't handling / itself, since that's legit for home screens etc
			v := b.Value[1 : len(b.Value)-1]
			if strings.HasSuffix(v, "/") && v != "/" {
				pass.Reportf(c.Args[0].Pos(), "HandleFunc paths shouldn't have trailing slashes")
			}
		} else if s.Sel.Name == "Handle" {
			// for Handle, you almost always want a trailing slash to ensure sub-mux delegation
			// you also really want to strip the prefix off for consistency
			// note to future developer(s): if you have a good reason to use Handle() instead
			//   of HandleFunc() for a single route, I apologize and you can fix me now :)
			b, ok := c.Args[0].(*ast.BasicLit)
			if !ok {
				// again, unusual
				return
			}
			v := b.Value[1 : len(b.Value)-1]
			if !strings.HasSuffix(v, "/") {
				pass.Reportf(c.Args[0].Pos(), "Handle paths should almost always have trailing slashes")
			}

			// if we're attaching a root handler eg. from main.go, we don't need to strip anything
			if v == "/" {
				return
			}

			c2, ok := c.Args[1].(*ast.CallExpr)
			if !ok {
				pass.Reportf(c.Args[1].Pos(), "Handle should almost always use http.StripPrefix()")
				return
			}

			s2, ok := c2.Fun.(*ast.SelectorExpr)
			if !ok {
				pass.Reportf(c.Args[1].Pos(), "Handle should almost always use http.StripPrefix()")
				return
			}

			if id, ok := s2.X.(*ast.Ident); ok && id.Name == "http" && s2.Sel.Name == "StripPrefix" {
				b, ok := c2.Args[0].(*ast.BasicLit)
				if !ok {
					// weird
					return
				}

				// don't complain about matching strings for public fileservers
				// TODO: this should go away once we optimize our static fileserving
				if v == "/public/" {
					return
				}

				if v2 := b.Value[1 : len(b.Value)-1]; fmt.Sprintf("%s/", v2) != v {
					pass.Reportf(c2.Args[0].Pos(), "http.StripPrefix almost always should have path without trailing slash as first arg")
				}

				return
			}

			pass.Reportf(c.Args[1].Pos(), "Handle should almost always use http.StripPrefix()")
		}

	})

	return nil, nil
}
