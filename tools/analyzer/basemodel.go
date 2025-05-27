package analyzer

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// BaseModelAnalyzer ensures we use ucdb.NewBase() instead of ucdb.BaseModel{} literal construction
// so that we get things like correct soft-delete behavior
var BaseModelAnalyzer = &analysis.Analyzer{
	Name:     "basemodel",
	Doc:      "checks for manual construction of BaseModel vs NewBase*",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runBaseModel,
}

func runBaseModel(pass *analysis.Pass) (any, error) {
	safeLines := findBypassComments(pass, "basemodel-safe")

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CompositeLit)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		c := n.(*ast.CompositeLit)

		s, ok := c.Type.(*ast.SelectorExpr)
		if !ok {
			// if this is a "naked" ident (eg. inside the ucdb package, still bad)
			if id, ok := c.Type.(*ast.Ident); ok {
				if id.Name == "BaseModel" && !safeLines[sForP(pass, id.Pos())] {
					pass.Reportf(id.Pos(), "use ucdb.NewBaseWithID() instead of a composite literal")
				}
			}

			// TODO: are there other patterns we might care about?
			return
		}

		// if we aren't constructing a BaseModel, don't care (for now)
		if s.Sel.Name != "BaseModel" && s.Sel.Name != "UserBaseModel" {
			return
		}

		// if this isn't a CompositeLiteral referencing an object in ucdb, we don't care
		id, ok := s.X.(*ast.Ident)
		if !ok {
			return
		}

		// if all matches, error
		if id.Name == "ucdb" && !safeLines[sForP(pass, id.Pos())] {
			pass.Reportf(id.Pos(), "use ucdb.NewBaseWithID() instead of a composite literal")
		}

	})

	return nil, nil
}
