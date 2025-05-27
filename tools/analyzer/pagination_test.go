package analyzer

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"userclouds.com/infra/assert"
)

func TestPagination(t *testing.T) {
	path, err := filepath.Abs("testdata")
	assert.NoErr(t, err)
	analysistest.Run(t, path, PaginationAnalyzer, "./pagination")
}

func TestPaginationResult(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, PaginationResultAnalyzer, "paginationresult/...")
}
