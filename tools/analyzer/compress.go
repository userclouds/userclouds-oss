package analyzer

import (
	"go/ast"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// CompressAnalyzer checks for places where we could use a compound if statement to reduce scope
var CompressAnalyzer = &analysis.Analyzer{
	Name:     "compress",
	Doc:      "checks for over-scoped vars specifically around if statements",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runCompress,
}

func runCompress(pass *analysis.Pass) (any, error) {
	safeLines := findBypassComments(pass, "scope-safe")
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.BlockStmt)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		b := n.(*ast.BlockStmt)

		objects := make(map[types.Object]int)

		for _, s := range b.List {
			// if this is an assignment, we handle it carefully
			if a, ok := s.(*ast.AssignStmt); ok {
				// first go through the left hand side looking for new vars
				var decls []types.Object
				for _, e := range a.Lhs {
					i, ok := e.(*ast.Ident)
					// if it's not an ident (like an assignment into a map), treat it like everything else for usage
					if !ok {
						v := &compVisitor{objects: objects, pass: pass}
						ast.Walk(v, e)
						continue
					}

					// if it's not a throwaway, this might be interesting
					if i.Name != "_" {
						if obj := pass.TypesInfo.ObjectOf(i); obj != nil {
							// only add objects if this is where they're actually declared,
							// but if we're using objects that aren't declared here,
							// anything on this line should be given a pass (because we can't
							// safely inline it without incorrectly scoping the non-declared var)
							if pass.Fset.Position(i.Pos()).Line == getDeclarationLine(pass, i) {
								// and of course, ignore things that have a safe comment
								if _, ok := safeLines[sForP(pass, i.Pos())]; !ok {
									decls = append(decls, obj)
								}
							} else {
								decls = nil
								break
							}
						}
					}
				}

				// only track these if there's a single value
				// TODO note that this misses another possible case like
				// a, b := foo()
				// if a {
				//   do(b)
				// }
				// where b isn't used anywhere else. but that's for another day
				if len(decls) == 1 {
					objects[decls[0]] = 0
				}

				v := &compVisitor{objects: objects, pass: pass}
				for _, e := range a.Rhs {
					ast.Walk(v, e)
				}

				continue
			}

			// if statements also get special handling
			if i, ok := s.(*ast.IfStmt); ok {
				v := &compVisitor{objects: objects, pass: pass}

				// stuff that's inside a call expression might be a legit factored-out var
				if _, ok = i.Cond.(*ast.CallExpr); ok {
					ast.Walk(v, i.Cond)
				}

				// stuff in the right-hand side of an init clause is ok
				if i.Init != nil {
					a := i.Init.(*ast.AssignStmt)
					for _, e := range a.Rhs {
						ast.Walk(v, e)
					}
				}

				// for binary expressions, they're ok only if they're "complex", eg not an ident
				if b, ok := i.Cond.(*ast.BinaryExpr); ok {
					_, ok = b.X.(*ast.Ident)
					if !ok {
						ast.Walk(v, b.X)
					}

					_, ok = b.Y.(*ast.Ident)
					if !ok {
						ast.Walk(v, b.Y)
					}
				}

				// same for unary expressions
				if u, ok := i.Cond.(*ast.UnaryExpr); ok {
					_, ok = u.X.(*ast.Ident)
					if !ok {
						ast.Walk(v, u.X)
					}
				}

				// now walk the condition but filter out objects declared right above
				v.lineFilter = pass.Fset.Position(i.Pos()).Line
				v.filteredObjects = nil
				ast.Walk(v, i.Cond)

				// walk the body, but no credit for things that appeared in the init/cond parts above
				// using filteredObjects from above
				v.lineFilter = 0
				ast.Walk(v, i.Body)

				continue
			}

			// anything else we just traverse to mark objects as used
			v := &compVisitor{objects: objects, pass: pass}
			ast.Walk(v, s)
		}

		// anything that was declared and not used correctly in the same block gets flagged
		for k := range objects {
			if objects[k] == 0 {
				p := pass.Fset.Position(k.Pos())
				if !strings.HasSuffix(p.Filename, "_test.go") {
					pass.Reportf(k.Pos(), "var only used in if, collapse into an init condition?")
				}
			}
		}
	})

	return nil, nil
}

func getDeclarationLine(pass *analysis.Pass, i *ast.Ident) int {
	obj := pass.TypesInfo.ObjectOf(i)
	if obj == nil {
		return 0
	}
	return pass.Fset.Position(obj.Pos()).Line
}

type compVisitor struct {
	objects         map[types.Object]int
	pass            *analysis.Pass
	lineFilter      int
	filteredObjects []types.Object
}

func (v *compVisitor) Visit(n ast.Node) ast.Visitor {
	i, ok := n.(*ast.Ident)
	if !ok {
		return v
	}

	obj := v.pass.TypesInfo.ObjectOf(i)
	if obj == nil {
		return v
	}

	if slices.Contains(v.filteredObjects, obj) {
		return v
	}

	// our line filter only matters if the declaration was an assignment
	if v.lineFilter != 0 {
		declLine := v.pass.Fset.Position(obj.Pos()).Line
		if v.lineFilter-declLine < 3 {
			v.filteredObjects = append(v.filteredObjects, obj)
			return v
		}
	}

	// mark this object as "used"
	v.objects[obj]++

	return v
}
