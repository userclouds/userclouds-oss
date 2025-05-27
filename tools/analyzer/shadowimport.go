package analyzer

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"userclouds.com/infra/ucerr"
)

// ShadowImportAnalyzer checks for variable or function definitions that shadow imported package names
var ShadowImportAnalyzer = &analysis.Analyzer{
	Name:     "shadowimport",
	Doc:      "checks for variable or function definitions that shadow imported package names",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runShadowImport,
}

func runShadowImport(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// For each file, collect all imported package names
	for _, file := range pass.Files {
		if isGeneratedTestFile(file) {
			continue
		}

		// Map to store imported package names
		importedPkgs := make(map[string]token.Pos)

		// Collect all imported package names
		for _, imp := range file.Imports {
			if imp.Name != nil {
				// If import has an alias, use that
				importedPkgs[imp.Name.Name] = imp.Pos()
				continue
			}

			// Extract the package name from the import path
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return nil, ucerr.Errorf("failed to unquote import path: %v", err)
			}
			parts := strings.Split(path, "/")
			pkgName := parts[len(parts)-1]

			// Skip if the package name is "_" (blank import)
			if pkgName == "_" {
				continue
			}

			importedPkgs[pkgName] = imp.Pos()
		}

		// Skip if no imports to check
		if len(importedPkgs) == 0 {
			continue
		}

		// Define node filter for declarations that might shadow imports
		nodeFilter := []ast.Node{
			(*ast.AssignStmt)(nil),
			(*ast.FuncDecl)(nil),
			(*ast.GenDecl)(nil),
		}

		// Inspect the file for shadowing
		insp.WithStack(nodeFilter, func(n ast.Node, push bool, stack []ast.Node) bool {
			if !push {
				return true
			}

			// inspect runs across the package, but we don't want to use imports from other files
			if pass.Fset.Position(n.Pos()).Filename != pass.Fset.Position(file.Pos()).Filename {
				return true
			}

			switch node := n.(type) {
			case *ast.AssignStmt:
				// Check for short variable declarations (x := value)
				if node.Tok != token.DEFINE {
					return true
				}

				// Check each variable being defined
				for _, lhs := range node.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok {
						if ident.Name == "_" {
							return true
						}
						if pos, exists := importedPkgs[ident.Name]; exists {
							pass.Reportf(ident.Pos(), "variable assignment %q shadows imported package name from line %d",
								ident.Name, pass.Fset.Position(pos).Line)
						}
					}
				}

			case *ast.GenDecl:
				// Check var declarations
				if node.Tok != token.VAR {
					return true
				}

				for _, spec := range node.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						for _, name := range valueSpec.Names {
							if name.Name == "_" {
								return true
							}
							if pos, exists := importedPkgs[name.Name]; exists {
								pass.Reportf(name.Pos(), "variable definition %q shadows imported package name from line %d",
									name.Name, pass.Fset.Position(pos).Line)
							}
						}
					}
				}
			}

			return true
		})
	}

	return nil, nil
}
