package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// HTTPErrorAnalyzer checks our sources to make sure we return after calling http.Error and other "final" method
var HTTPErrorAnalyzer = &analysis.Analyzer{
	Name:     "httperror",
	Doc:      "checks httperror and jsonapi return usage",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runHE,
}

// We enumerate jsonapi.*Error here ... we could make this regex matching,
// but we really don't want so many of them that that is needed :)
var functionsToCheck = []string{
	"http.Error",
	"uchttp.Error",
	"uchttp.ErrorL",
	"http.Redirect",
	"jsonapi.Marshal",
	"jsonapi.MarshalError",
	"jsonapi.MarshalErrorL",
	"jsonapi.MarshalSQLError",
	"jsonapi.MarshalSQLErrorL",
}

func runHE(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil), // definitions
		(*ast.FuncLit)(nil),  // inlined closures
	}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		// exempt test or sample files
		if fn := pass.Fset.Position(n.Pos()).Filename; strings.Contains(fn, "_test.go") || strings.Contains(fn, "/samples/") {
			return
		}

		var body *ast.BlockStmt
		switch n := n.(type) {
		case *ast.FuncDecl:
			body = n.Body
		case *ast.FuncLit:
			body = n.Body
		}
		if body == nil {
			return
		}
		if len(body.List) == 0 {
			return
		}
		last := body.List[len(body.List)-1]
		v := &heVisitor{
			pass:   pass,
			ignore: last.Pos(), // ok if last expr in function is error marshaler
		}
		ast.Walk(v, body)
	})
	return nil, nil
}

type heVisitor struct {
	pass          *analysis.Pass
	ignore        token.Pos // whitelisted error marshaler, usually the last line of a function
	preceding     token.Pos // jsonapi error marshaler earlier in the block
	originalError token.Pos // the original jsonapi/http/uchttp call that caused us to worry, inside an if block TODO hard to understand?
	name          string    // error marshaler's name
}

func (v *heVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil { // previous node was a leaf
		if v.hasPreceding() {
			// Error (maybe)! Code path with an error marshaler ended without a return.
			// Unless this was the last line of a function (handled in v.report)
			v.report(v.preceding)
		}
		return nil
	}
	switch stmt := n.(type) {
	case *ast.ReturnStmt:
		// a return forgives all sins
		v.reset()
		return nil
	case *ast.ExprStmt:
		// any other expression that follows a "regulated" call is an error
		// specifically even another http.Error etc, because two in a row doesn't make sense
		if v.hasPreceding() {
			v.report(v.preceding)
			return nil
		}

		// if we aren't following one of these calls, let's see if we *are* one of these
		// calls that we need to watch for in the future
		name, ok := v.exprToCheck(stmt)
		if !ok {
			return nil
		}

		v.preceding = stmt.Pos()
		v.name = name

		return nil
	case *ast.SwitchStmt:
		if len(stmt.Body.List) > 0 {
			hasFinalExpr := false
			for _, caseStmt := range stmt.Body.List {
				// technically must be a CaseClause according to AST docs, so unsafe cast
				caseBody := caseStmt.(*ast.CaseClause)

				// nothing to check in a fallthrough case
				if len(caseBody.Body) > 0 {
					caseV := heVisitor{
						pass:   v.pass,
						ignore: caseBody.Body[len(caseBody.Body)-1].Pos(),
					}
					ast.Walk(&caseV, caseBody)
					if caseV.preceding != token.NoPos {
						hasFinalExpr = true
						// Don't break because that will fail to lint other case blocks statements
					}
				}
			}
			if hasFinalExpr {
				// At least one case block had a "final" expression like http.Error, etc
				v.preceding = stmt.Pos()
				v.name = "switch statement"
			}
		}
		return nil
	case *ast.IfStmt:
		// we count as a non-return statement ourselves...
		// TODO: unify with default case?
		if v.hasPreceding() {
			v.report(v.preceding)
		}

		// this is a static-check error but don't crash if we run before static-check
		if len(stmt.Body.List) == 0 {
			return nil
		}

		// check the statements inside the if, and else if applicable
		// ignore the last statement just in case we have a pattern like
		//
		// if x {
		// 	http.Error(...)
		// } else {
		// 	http.Error(...)
		// }
		// return

		ifV := heVisitor{
			pass:   v.pass,
			ignore: stmt.Body.List[len(stmt.Body.List)-1].Pos(),
		}
		ast.Walk(&ifV, stmt.Body)

		var elseV *heVisitor
		if stmt.Else != nil {
			body, ok := stmt.Else.(*ast.BlockStmt)
			if ok {
				elseV = &heVisitor{
					pass:   v.pass,
					ignore: body.List[len(body.List)-1].Pos(),
				}
				ast.Walk(elseV, body)
			} else {
				// must be else if (stmt.Else can only be Block or If)
				elseIfStmt := stmt.Else.(*ast.IfStmt)
				elseV = &heVisitor{
					pass: v.pass,
				}
				ast.Walk(elseV, elseIfStmt)
			}
		}

		// if *any* branches of the if statement end with one of our calls,
		// we can classify this entire statement as the call itself and look
		// for the *next* statement to be a return or EOF. This allows for some
		// branches to be non-errors, but still ensure the code returns after.
		if ifV.preceding == ifV.ignore ||
			(elseV != nil && elseV.preceding != token.NoPos) {
			v.preceding = stmt.Pos()
			v.name = "if statement"
		}

		if ifV.preceding == ifV.ignore && (elseV == nil || elseV.preceding == token.NoPos) {
			if ifV.originalError != token.NoPos {
				v.originalError = ifV.originalError
			} else {
				v.originalError = ifV.preceding
			}
			v.name = ifV.name
		} else if elseV != nil && elseV.preceding != token.NoPos && ifV.preceding != ifV.ignore {
			if elseV.originalError != token.NoPos {
				v.originalError = elseV.originalError
			} else {
				v.originalError = elseV.preceding
			}
			v.name = elseV.name
		}

		return nil
	case *ast.FuncLit:
		// if we come across a function literal here, that means that we're inside another Body
		// (either of a function declaration, or a nested function literal). Either way we can
		// safely skip this analysis pass since we'll analyze this .Body on it's own as part of
		// the inspect.Preorder call in runHE
		return nil
	default:
		// likewise, any statement that's not an expression or a return is an error
		// if it follows a controlled call
		if v.hasPreceding() {
			// Error! This should be a return.
			v.report(v.preceding)
			return nil
		}
	}
	return v
}

func (v *heVisitor) report(pos token.Pos) {
	if pos == v.ignore {
		return
	}
	if v.originalError != token.NoPos {
		pos = v.originalError
	}
	v.pass.Reportf(pos, "%s not followed by return", v.name)
	v.reset() // only complain once
}

func (v *heVisitor) hasPreceding() bool {
	return v.preceding != token.NoPos
}

func (v *heVisitor) reset() {
	v.preceding = token.NoPos
	v.name = ""
}

// returns [name], true if this is an expr we care about linting
func (v *heVisitor) exprToCheck(stmt *ast.ExprStmt) (string, bool) {
	pkgName, funcName := getFunctionName(v.pass, stmt)
	if funcName == "" {
		return "", false
	}

	// Special exception: allow tests for a given package to violate this rule for
	// functions in that package (e.g. jsonapi_test can violate this rule for
	// calls to jsonapi.*), so it's easier to test the methods.
	if strings.HasSuffix(v.pass.Pkg.Name(), "_test") {
		if strings.TrimSuffix(v.pass.Pkg.Name(), "_test") == pkgName {
			return "", false
		}
	}

	fullName := fmt.Sprintf("%s.%s", pkgName, funcName)

	// is this a function we're trying to check?
	if slices.Contains(functionsToCheck, fullName) {
		return fullName, true
	}

	return "", false
}

func getFunctionName(pass *analysis.Pass, expr *ast.ExprStmt) (string, string) {
	call, ok := expr.X.(*ast.CallExpr)
	if !ok {
		return "", ""
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		// If we are inside the package where a function is implemented,
		// the call will be an Ident instead of Selector expression.
		ident, ok := call.Fun.(*ast.Ident)
		if ok {
			return pass.Pkg.Name(), ident.Name
		}
		return "", ""
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return "", ""
	}

	return ident.Name, selector.Sel.Name
}
