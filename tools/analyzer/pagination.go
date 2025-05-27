package analyzer

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// PaginationAnalyzer checks for places where we might infinite-loop on paginated queries
var PaginationAnalyzer = &analysis.Analyzer{
	Name:     "pagination",
	Doc:      "checks for common pagination usage errors",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runPagination,
}

// PaginationResultAnalyzer checks for places where pagination result fields are ignored in storage calls
var PaginationResultAnalyzer = &analysis.Analyzer{
	Name:     "paginationresult",
	Doc:      "checks for ignored pagination result fields in storage calls",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runPaginationResult,
}

func runPagination(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.ForStmt)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		f := n.(*ast.ForStmt)

		// if this isn't a simple for loop, we don't care because
		// it has other exit criteria
		if f.Init != nil || f.Cond != nil || f.Post != nil {
			return
		}

		// does this have a List...Pagination call?
		bv := &bodyVisitor{pass: pass}
		ast.Walk(bv, f.Body)

		if bv.foundPagination && !bv.foundAdvance {
			pass.Reportf(f.Pos(), "found a naked for loop with pagination but no advance")
		}

	})

	return nil, nil
}

type bodyVisitor struct {
	pass            *analysis.Pass
	foundPagination bool
	foundAdvance    bool
}

func (v *bodyVisitor) Visit(n ast.Node) ast.Visitor {
	c, ok := n.(*ast.CallExpr)
	if !ok {
		return v
	}

	s, ok := c.Fun.(*ast.SelectorExpr)
	if !ok {
		if i, ok := c.Fun.(*ast.Ident); ok {
			if checkForPagination(i.Name) {
				v.foundPagination = true
			} else if checkForAdvance(i.Name) {
				v.foundAdvance = true
			}
		}
		// the third case is anonymous functions here, but those seem irrelevant for this purpose
		return v
	}

	if checkForPagination(s.Sel.Name) {
		v.foundPagination = true
	} else if checkForAdvance(s.Sel.Name) {
		v.foundAdvance = true
	}

	return v
}

func checkForPagination(name string) bool {
	return strings.HasPrefix(name, "List") && strings.HasSuffix(name, "Paginated")
}

func checkForAdvance(name string) bool {
	return name == "AdvanceCursor"
}

func runPaginationResult(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.AssignStmt)(nil),
	}

	safeLines := findBypassComments(pass, "ucpagination-safe")
	insp.Preorder(nodeFilter, func(n ast.Node) {

		// ignore test or sample files
		if fn := pass.Fset.Position(n.Pos()).Filename; strings.Contains(fn, "_test.go") || strings.Contains(fn, "/samples/") {
			return
		}

		assign := n.(*ast.AssignStmt)

		// we don't handle complicated assignments here.
		if len(assign.Rhs) != 1 {
			return
		}

		// assumption - pagination always returns three values
		if len(assign.Lhs) != 3 {
			return
		}

		if _, ok := safeLines[sForP(pass, assign.Pos())]; ok {
			return
		}

		ident, ok := assign.Lhs[1].(*ast.Ident)
		if !ok {
			// TODO: we might need to do something here? this might be too tight of a check
			// but why would you eg. store the pagination result in a struct member?
			pass.Reportf(n.Pos(), "pagination result field from storage call is ignored")
			return
		}

		typ := pass.TypesInfo.Defs[ident]
		if typ == nil {
			// FIXME
			return
		}

		if typ.Type().String() != "*userclouds.com/infra/pagination.ResponseFields" &&
			typ.Type().String() != "*struct{HasNext bool}" {
			return
		}

		if !isPaginationResultUsed(pass, assign) {
			pass.Reportf(n.Pos(), "pagination result field from storage call is ignored")
		}
	})

	return nil, nil
}

func isPaginationResultUsed(pass *analysis.Pass, assign *ast.AssignStmt) bool {
	if len(assign.Lhs) != 3 {
		return false // Not a pagination call with 3 return values
	}

	// Check second return value (pagination result)
	expr := assign.Lhs[1]
	id, ok := expr.(*ast.Ident)
	if !ok || id.Name == "_" { // Explicitly ignoring pagination result
		return false
	}

	// we could actually check if the pagination result is used, but this is good enough for now
	// notably, we'd need to ensure that either AdvanceCursor is called with the pagination result,
	// or that the pagination result is returned from the function, or that it's shipped to the client
	// in a json.Marshal call, and that gets annoying.

	return true
}
