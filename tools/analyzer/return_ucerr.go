package analyzer

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// ReturnUCErrAnalyzer encourages us to wrap errors with ucerr.Wrap()
var ReturnUCErrAnalyzer = &analysis.Analyzer{
	Name:     "returnuc",
	Doc:      "checks for unwrapped error returns",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runReturnUC,
}

func runReturnUC(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	ignore := findBypassComments(pass, "ucerr-ignore")

	insp.Preorder(nodeFilter, func(n ast.Node) {
		// let's start by exempting ucerr packages from this check for obvious reasons :)
		if strings.Contains(pass.Pkg.Name(), "ucerr") {
			return
		}

		// TODO (sgarrity 4/25): i'd like to fix the generating code, but this is a quick fix
		if strings.HasSuffix(pass.Fset.Position(n.Pos()).Filename, "_easyjson.go") {
			return
		}

		// and test or sample files
		if fn := pass.Fset.Position(n.Pos()).Filename; strings.Contains(fn, "_test.go") || strings.Contains(fn, "/samples/") {
			return
		}

		// collect the return values & function bodies from both declared and inline functions
		var results *ast.FieldList
		var body *ast.BlockStmt
		var name string
		if fn, ok := n.(*ast.FuncDecl); ok {
			results = fn.Type.Results
			body = fn.Body
			name = fn.Name.Name
		}

		if fn, ok := n.(*ast.FuncLit); ok {
			results = fn.Type.Results
			body = fn.Body
		}

		// if the fn eg. doesn't return anything, don't bother
		if results == nil || body == nil {
			return
		}

		if len(results.List) == 0 {
			return
		}

		// we also shouldn't enforce this on Unwrap functions as that would be bad
		if name == "Unwrap" && len(results.List) == 1 {
			if id, ok := results.List[0].Type.(*ast.Ident); ok && id.Name == "error" {
				return
			}
		}

		// does this function return any errors? if so, walk it to check each return
		var idx int
		for _, res := range results.List {
			if id, ok := res.Type.(*ast.Ident); ok && id.Name == "error" {
				ast.Walk(retVisitor{idx, pass, body, ignore}, body)
			}

			// len(res.Names) != 1 with grouped return values like `func foo() (a, b int, err error)`,
			// but can also be zero (usually) if return values are unnamed
			if len(res.Names) > 0 {
				idx += len(res.Names)
			} else {
				idx++
			}
		}
	})

	return nil, nil
}

type retVisitor struct {
	errorIndex int
	pass       *analysis.Pass
	body       *ast.BlockStmt

	ignore map[string]bool
}

func (r retVisitor) Visit(n ast.Node) ast.Visitor {
	// if we're just being called as the DFS is popping back up, ignore it
	if n == nil {
		return r
	}

	if r.ignore[sForP(r.pass, n.Pos())] {
		return r
	}

	// if we find a function literal defined inside this block, skip it
	// and we'll hit it separately from Preorder above. Otherwise our index into
	// return params might be off
	if _, ok := n.(*ast.FuncLit); ok {
		return nil
	}

	// skip anything that's not return
	rs, ok := n.(*ast.ReturnStmt)
	if !ok {
		return r
	}

	// if we think the error is eg. param 2 (index 1), and the actual return statement just invokes another fn
	// with multiple return values as well, don't barf. We could be annoying and error here though?
	// TODO: handle this case better?
	if r.errorIndex >= len(rs.Results) {
		return nil
	}

	er := rs.Results[r.errorIndex]

	if id, ok := er.(*ast.Ident); ok {
		// no need to wrap nil
		if id.Name == "nil" {
			return nil
		}

		// Check if the identifier is a package-level error constant. If you define that using errors.New(),
		// the UCWrapper linter will catch it, so we're going to let this one slide.
		obj := r.pass.TypesInfo.ObjectOf(id)
		packageLevel := true
		current := obj.Parent()
		for {
			// if our current scope contains this statement, it's not package-level and we're done
			if current.Contains(r.body.Pos()) {
				packageLevel = false
				break
			}

			parent := current.Parent()
			// if we reach something with no parent, we're at the package and we're done
			if parent == nil {
				break
			}
			current = parent
		}
		if packageLevel {
			return nil
		}
	}

	if r.ignore[sForP(r.pass, er.Pos())] {
		return r
	}

	// returning the result of a function should ideally be wrapped
	if c, ok := er.(*ast.CallExpr); ok {
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			r.pass.Reportf(c.Pos(), "function error return value at pos %d should be wrapped with ucerr.Wrap()", r.errorIndex)
			return nil
		}

		// check the selector expression all the way down (eg. selector.Ident or selector.selector.Ident or ...)
		return r.checkSelector(c, sel)
	}

	// this is the likely case where it was an identifier that wasn't nil or package-scoped
	r.pass.Reportf(er.Pos(), "error return value at pos %d should be wrapped with ucerr.Wrap()", r.errorIndex)

	return nil
}

func (r retVisitor) checkSelector(c *ast.CallExpr, sel *ast.SelectorExpr) ast.Visitor {
	id, ok := sel.X.(*ast.Ident)
	if ok {
		// correctly wrapping this RV gets us a pass
		if id.Name == "ucerr" {
			if strings.HasPrefix(sel.Sel.Name, "Wrap") {

				if sel.Sel.Name == "Wrap" {
					// check to make sure we don't call ucerr.Wrap(nil) because I've done that and it's dumb.
					if id, ok := c.Args[0].(*ast.Ident); ok {
						if id.Name == "nil" {
							r.pass.Reportf(id.Pos(), "don't call ucerr.Wrap(nil), it's useless")
						}
					}
				}

				return nil
			} else if sel.Sel.Name == "Errorf" ||
				strings.HasPrefix(sel.Sel.Name, "New") ||
				sel.Sel.Name == "Friendlyf" {
				// ucerr.Errorf and ucerr.New* are all valid (there are many error helper methods in ucerr that all start with 'New')
				return nil
			}
		}

		// Don't require wrapping around uctrace.WrapX(). These usually wrap
		// functions defined inline (so the error wrapping doesn't buy us
		// anything), and when they have multiple return values, it's a huge
		// pain to intercept/wrap the error return value
		if id.Name == "uctrace" && strings.HasPrefix(sel.Sel.Name, "Wrap") {
			return nil
		}

		r.pass.Reportf(c.Pos(), "error return value should be wrapped with ucerr.Wrap()")
		return nil
	}

	subSel, ok := sel.X.(*ast.SelectorExpr)
	if ok {
		return r.checkSelector(c, subSel)
	}

	return nil
}
