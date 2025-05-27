package analyzer

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// FriendlyErrorAnalyzer checks our sources to make sure we return after calling http.Error and other "final" method
var FriendlyErrorAnalyzer = &analysis.Analyzer{
	Name:     "friendlyerror",
	Doc:      "checks that ucerr.Friendly doesn't include errors",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runFE,
}

func runFE(pass *analysis.Pass) (any, error) {
	safeLines := findBypassComments(pass, "lint-safe-wrap")

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

		// only care about ucerr.Friendlyf here
		// TODO (sgarrity 10/23): handle aliasing in imports for the package name
		if i.Name != "ucerr" || s.Sel.Name != "Friendlyf" {
			return
		}

		// allow bypassing for the case we are including unwrapped (eg. native package) errors for debugging
		if _, ok := safeLines[sForP(pass, s.Pos())]; ok {
			return
		}

		// check the arguments to Friendlyf
		// the first arg doesn't matter (we know it's a safely wrapper error or nil),
		// and the function requires 2+ args so this array index should be safe
		for _, arg := range c.Args[1:] {
			if i, ok := arg.(*ast.Ident); ok {
				if pass.TypesInfo.Types[i].Type.String() == "error" {
					pass.Reportf(i.Pos(), "don't use errors as format-string arguments to ucerr.Friendlyf or you'll leak stack traces")
				}
				continue
			}
			s := getCallSelectorOrNil(arg)
			if s == nil {
				continue
			}
			if pos := getSprintfCallPosition(s); pos != nil {
				pass.Reportf(*pos, "Friendlyf already accepts format strings, don't add fmt.Sprintf as well")
			}
			// if this identifier is eg. a package, we won't have a type and don't want to crash
			if pass.TypesInfo.Types[s.X].Type == nil {
				continue
			}
			// err.Error() is basically the same as the case above of just using err
			if pass.TypesInfo.Types[s.X].Type.String() == "error" && s.Sel.Name == "Error" {
				pass.Reportf(s.X.Pos(), "don't use err.Error() as an argument to ucerr.Friendlyf or you'll leak stack traces")
			}
		}

	})
	return nil, nil
}
