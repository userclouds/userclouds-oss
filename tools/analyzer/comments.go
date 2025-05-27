package analyzer

import (
	"fmt"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// we have to iterate over Files.Comments separately because for some reason,
// inspect.Preorder is missing line-ending comments (even with ast.Comment filter, etc)
// might as well share this code for multiple linters, even though we aren't (yet)
// implementing a generic system that applies to all results (see golang philosophy on
// not littering the code with lint-safe comments, but they're occasionally useful).
func findBypassComments(pass *analysis.Pass, safeComment string) map[string]bool {
	safeLines := make(map[string]bool)
	for _, f := range pass.Files {
		for _, c := range f.Comments {
			if strings.Contains(c.Text(), safeComment) {
				safeLines[sForP(pass, c.Pos())] = true
			}
		}
	}

	return safeLines
}

// string for position, returns [filename]:[line]
func sForP(pass *analysis.Pass, p token.Pos) string {
	pos := pass.Fset.Position(p)
	return fmt.Sprintf("%s:%d", pos.Filename, pos.Line)
}
