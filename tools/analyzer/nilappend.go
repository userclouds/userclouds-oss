package analyzer

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// NilAppendAnalyzer checks for unnecessary nil checks before append
var NilAppendAnalyzer = &analysis.Analyzer{
	Name:     "nilappend",
	Doc:      "checks for unnecessary nil checks before append since append can handle nil slices",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runNilAppend,
}

func runNilAppend(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.IfStmt)(nil),
	}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		ifStmt := n.(*ast.IfStmt)

		// Check if this is a nil check (x == nil)
		binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr)
		if !ok || binExpr.Op != token.EQL {
			return
		}

		// Get the variable being checked
		var checkedVar string
		if ident, ok := binExpr.X.(*ast.Ident); ok && isNilIdent(binExpr.Y) {
			checkedVar = ident.Name
		} else if ident, ok := binExpr.Y.(*ast.Ident); ok && isNilIdent(binExpr.X) {
			checkedVar = ident.Name
		} else {
			return
		}

		// Check if the body contains make followed by append
		if len(ifStmt.Body.List) < 1 {
			return
		}

		// Check for make assignment
		assign, ok := ifStmt.Body.List[0].(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return
		}

		// Check if we're assigning to the same variable
		lhsIdent, ok := assign.Lhs[0].(*ast.Ident)
		if !ok || lhsIdent.Name != checkedVar {
			return
		}

		// Check if RHS is make
		makeExpr, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok {
			return
		}
		makeIdent, ok := makeExpr.Fun.(*ast.Ident)
		if !ok || makeIdent.Name != "make" {
			return
		}

		// Look for append after the if statement
		if parent, ok := findParentBlock(pass, ifStmt); ok {
			for _, stmt := range parent.List {
				if assign, ok := stmt.(*ast.AssignStmt); ok {
					if isAppendToVar(assign, checkedVar) {
						pass.Reportf(ifStmt.Pos(), "unnecessary nil check: append can handle nil slices")
						return
					}
				}
			}
		}
	})
	return nil, nil
}

func isNilIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}

func findParentBlock(pass *analysis.Pass, node ast.Node) (*ast.BlockStmt, bool) {
	var result *ast.BlockStmt
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.WithStack([]ast.Node{(*ast.BlockStmt)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}
		block := n.(*ast.BlockStmt)
		for _, stmt := range block.List {
			if stmt == node {
				result = block
				return false
			}
		}
		return true
	})
	return result, result != nil
}

func isAppendToVar(assign *ast.AssignStmt, varName string) bool {
	if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return false
	}

	// Check LHS is our variable
	lhsIdent, ok := assign.Lhs[0].(*ast.Ident)
	if !ok || lhsIdent.Name != varName {
		return false
	}

	// Check RHS is append
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	fun, ok := call.Fun.(*ast.Ident)
	return ok && fun.Name == "append"
}
