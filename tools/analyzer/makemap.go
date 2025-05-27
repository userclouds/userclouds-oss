package analyzer

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// MakeMapAnalyzer prevents Steve from adding random fixed sizes to maps
var MakeMapAnalyzer = &analysis.Analyzer{
	Name:     "makemap",
	Doc:      "checks for random int sizes in make(map[])",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runMakeMap,
}

func runMakeMap(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		c := n.(*ast.CallExpr)

		// if the function isn't an ident (eg it's a selector), it's not make
		i, ok := c.Fun.(*ast.Ident)
		if !ok {
			return
		}

		if i.Name != "make" {
			return
		}

		// ideally we just leave off the second arg and we're ok
		if len(c.Args) != 2 {
			return
		}

		if _, ok = c.Args[0].(*ast.MapType); !ok {
			return
		}

		// if it's not a literal, there's probably a good reason for the size estimate
		b, ok := c.Args[1].(*ast.BasicLit)
		if !ok {
			return
		}

		if b.Kind == token.INT {
			pass.Reportf(b.Pos(), "make(map[]) called with an integer literal size...are you sure?")
		}
	})

	return nil, nil
}
