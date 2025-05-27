package analyzer

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"userclouds.com/infra/assert"
)

func TestValidate(t *testing.T) {
	path, err := filepath.Abs("testdata")
	assert.NoErr(t, err)
	analysistest.Run(t, path, ValidateAnalyzer, "./validate")
}
