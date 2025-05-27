package analyzer

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// ValidateAnalyzer checks to make sure we only use UTC times
var ValidateAnalyzer = &analysis.Analyzer{
	Name:     "validate",
	Doc:      "Checks Validate calls to ensure we're not accidentally calling embedded BaseModel",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runValidate,
}

func runValidate(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		c := n.(*ast.CallExpr)

		// only check x.y
		s, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}

		// ignore calls not to Validate
		if s.Sel.Name != "Validate" {
			return
		}

		id := findIdent(s.X)
		if id == nil {
			pass.Reportf(s.X.Pos(), "linter implementation error: couldn't find Ident on this line")
			return
		}

		t := pass.TypesInfo.Types[id]

		if t.Type == nil {
			// TODO: not sure why this is required yet but it panics in our test case
			// (but skipping nil Types doesn't actually change test results)
			return
		}
		// TODO: there's a hidden failure mode here when MethodSet is empty, but I haven't figured it out yet
		// I think it has something to do with types loaded outside of the current package (which is significant),
		// but it's not always that so this whole linter is still helpful.
		// TODO: there's a second thing we should check but we currently don't, which is when a .Validate() method
		// delegates to a second-order Validate(), eg. Authn -> BaseModel when it should go Authn -> UserBaseModel -> BaseModel
		ms := types.NewMethodSet(t.Type)
		for i := range ms.Len() {
			m := ms.At(i)
			if m.Obj().Name() != "Validate" {
				continue
			}
			idx := m.Index()
			if len(idx) != 1 {
				pass.Reportf(c.Pos(), "Validate is calling an embedded method")
				return
			}
		}
	})
	return nil, nil
}

// findIdent recursively walks an ast.Expr to find the relevant Ident node
// that we can check the Validate() methods on that object
func findIdent(n ast.Expr) *ast.Ident {
	id, ok := n.(*ast.Ident)
	if ok {
		return id
	}

	sel, ok := n.(*ast.SelectorExpr)
	if ok {
		return findIdent(sel.Sel)
	}

	idx, ok := n.(*ast.IndexExpr)
	if ok {
		return findIdent(idx.X)
	}

	star, ok := n.(*ast.StarExpr)
	if ok {
		return findIdent(star.X)
	}

	paren, ok := n.(*ast.ParenExpr)
	if ok {
		return findIdent(paren.X)
	}
	return nil
}
