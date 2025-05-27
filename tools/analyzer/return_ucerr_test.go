package analyzer

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"userclouds.com/infra/assert"
)

func TestReturnUCError(t *testing.T) {
	path, err := filepath.Abs("testdata")
	assert.NoErr(t, err)
	analysistest.Run(t, path, ReturnUCErrAnalyzer, "./return_ucerr")
}
