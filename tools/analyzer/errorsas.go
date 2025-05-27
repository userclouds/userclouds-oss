package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// ErrorsAsAnalyzer ensures we use errors.As() instead of casts to handle error type conversion
var ErrorsAsAnalyzer = &analysis.Analyzer{
	Name:     "errorsas",
	Doc:      "checks errors.As usage",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runEA,
}

func runEA(pass *analysis.Pass) (any, error) {
	safeLines := findBypassComments(pass, "ecast-safe")

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.TypeAssertExpr)(nil),
		(*ast.BinaryExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		if _, ok := safeLines[sForP(pass, n.Pos())]; ok {
			return
		}

		if a, ok := n.(*ast.TypeAssertExpr); ok {
			checkTypeAssertExpr(pass, a)
		} else if b, ok := n.(*ast.BinaryExpr); ok {
			checkBinaryExpr(pass, b)
		}
	})

	return nil, nil
}

func checkTypeAssertExpr(pass *analysis.Pass, a *ast.TypeAssertExpr) {
	errIfc := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

	// if we're casting a non-error (eg. an any) to an error, we actually have to use a cast
	// instead of errors.As(), so don't complain here
	if typ := pass.TypesInfo.TypeOf(a.X); typ != nil && !types.Implements(typ, errIfc) {
		return
	}

	// if the type we're casting to implements error, then complain
	if typ := pass.TypesInfo.TypeOf(a.Type); typ != nil && types.Implements(typ, errIfc) {
		parts := strings.Split(typ.String(), ".")
		pass.Reportf(a.Type.Pos(), "use errors.As() instead of casting to %s", parts[len(parts)-1])
	}

	// and if we're casting directly to error from something that implements error (why?), complain
	if i, ok := a.Type.(*ast.Ident); ok {
		if i.Name == "error" {
			// this should be rare to nonexistant, but complain anyway?
			pass.Reportf(i.Pos(), "use errors.As() instead of casting to error")
		}
	}
}

func checkBinaryExpr(pass *analysis.Pass, a *ast.BinaryExpr) {
	if a.Op != token.EQL && a.Op != token.NEQ {
		return
	}

	typX := pass.TypesInfo.TypeOf(a.X)
	typY := pass.TypesInfo.TypeOf(a.Y)

	if typX != nil && typX.String() == "error" && typY != nil && typY.String() == "error" {
		pass.Reportf(a.Y.Pos(), "use errors.Is() instead of comparing to a specific error")
	}
}
