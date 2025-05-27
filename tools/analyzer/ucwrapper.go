package analyzer

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// UCWrapperAnalyzer encourages use of things like ucerr.New instead of errors.New
var UCWrapperAnalyzer = &analysis.Analyzer{
	Name:     "ucwrapper",
	Doc:      "checks for unwrapped functions",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runUCWrapper,
}

var replacements = map[string]string{
	"fmt.Errorf":    "ucerr.Errorf",
	"errors.New":    "ucerr.New",
	"log.Printf":    "uclog.Debugf",
	"log.Println":   "uclog.Debugf",
	"http.Error":    "uchttp.Error",
	"http.Redirect": "uchttp.Redirect",
}

func runUCWrapper(pass *analysis.Pass) (any, error) {
	safeLines := findBypassComments(pass, "ucwrapper-safe")
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		c := n.(*ast.CallExpr)

		// first, make sure this is a selector and not a local function, etc
		// if it's not a selector, it won't be one of our wrappable functions
		s, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}

		// if the right side of the selector isn't an ident, shouldn't matter
		i, ok := s.X.(*ast.Ident)
		if !ok {
			return
		}

		// TODO: handle aliasing in imports for the package name
		fullName := fmt.Sprintf("%s.%s", i.Name, s.Sel.Name)

		for k, v := range replacements {
			// don't suggest that we use eg. uclog.Debugf inside of uclog package
			if strings.HasPrefix(v, pass.Pkg.Name()) {
				continue
			}

			// exempt "cmd" files from the uclog "requirements" because config might not be loaded/avail
			// this might be overly broad, but it's better than a ton of lint suppression comments
			if strings.Contains(pass.Pkg.Path(), "cmd") && strings.HasPrefix(v, "uclog") {
				continue
			}

			// exempt samples from this rule for now to avoid more dependencies to manage
			if strings.Contains(pass.Pkg.Path(), "samples") {
				continue
			}

			if fullName == k {
				// Matched a method on the ban list, report it if there's no bypass comment.
				if _, ok := safeLines[sForP(pass, s.Pos())]; !ok {
					pass.Reportf(s.Pos(), "instead of %s, use %s", fullName, v)
				}
			}
		}
	})

	return nil, nil
}
