package analyzer

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// ErrorFormatAnalyzer checks our sources to make sure we don't call ucerr.New w/ fmt.Sprintf
var ErrorFormatAnalyzer = &analysis.Analyzer{
	Name:     "errorformat",
	Doc:      "checks that ucerr.Friendly doesn't include errors",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runFmtErr,
}

func runFmtErr(pass *analysis.Pass) (any, error) {
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
		// check the first arguments to ucerr.New, make sure it is not fmt.Sprintf
		if i.Name != "ucerr" || s.Sel.Name != "New" || len(c.Args) == 0 {
			return
		}

		sl := getCallSelectorOrNil(c.Args[0])
		if sl == nil {
			return
		}
		if pos := getSprintfCallPosition(sl); pos != nil {
			pass.Reportf(*pos, "Use ucerr.Errorf instead of ucerr.New(fmt.Sprintf(....))")
		}
	})
	return nil, nil
}
