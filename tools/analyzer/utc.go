package analyzer

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// UTCAnalyzer checks to make sure we only use UTC times
var UTCAnalyzer = &analysis.Analyzer{
	Name:     "utc",
	Doc:      "Checks date/time statements to ensure UTC usage from now/today",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runUTC,
}

func runUTC(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	// first pass to find safe UTC call sites
	safeCalls := make(map[ast.Expr]bool)
	insp.Preorder(nodeFilter, func(n ast.Node) {
		c := n.(*ast.CallExpr)

		s, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}

		if s.Sel.Name == "UTC" || s.Sel.Name == "Unix" {
			safeCalls[s.X] = true
		}
	})

	safeLines := findBypassComments(pass, "utc-safe")

	// second pass for Now() and Today()
	insp.Preorder(nodeFilter, func(n ast.Node) {
		c := n.(*ast.CallExpr)

		s, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}

		// note this assumes we don't have packages other than time & date that define Now() and Today()
		// if we ever introduce those, we can add s.X.Name checks here, but don't need them now
		if s.Sel.Name == "Now" {
			// safe by comment line
			if safeLines[sForP(pass, s.Pos())] {
				return
			}

			// safe by .UTC()?
			if safeCalls[n.(ast.Expr)] {
				return
			}

			pass.Reportf(s.Pos(), "Now() called without UTC()")
		} else if s.Sel.Name == "Today" {
			// the loomdate package doesn't handle timezones at all, so this is safe
			x, ok := s.X.(*ast.Ident)
			if ok {
				if x.Name == "loomdate" {
					return
				}
			}

			// safe by comment line
			if safeLines[sForP(pass, s.Pos())] {
				return
			}

			pass.Reportf(s.Pos(), "Today() called instead of TodayUTC()")
		}
	})
	return nil, nil
}
