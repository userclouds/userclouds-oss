package analyzer

import (
	"go/ast"
	"go/token"
)

func getCallSelectorOrNil(arg ast.Expr) *ast.SelectorExpr {
	c, ok := arg.(*ast.CallExpr)
	if !ok {
		return nil
	}
	s, ok := c.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	return s
}

func getSprintfCallPosition(s *ast.SelectorExpr) *token.Pos {
	i, ok := s.X.(*ast.Ident)
	if !ok {
		return nil
	}
	if i.Name == "fmt" && s.Sel.Name == "Sprintf" {
		ps := i.Pos()
		return &ps
	}
	return nil
}
