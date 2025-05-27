package parameter_test

import (
	"os"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/internal/pageparameters/parameter"
)

func TestNameValidity(t *testing.T) {
	for _, pn := range parameter.Names() {
		assert.True(t, pn.Validate() == nil)
	}

	var badName parameter.Name = "bad name"
	assert.True(t, badName.Validate() != nil)
}

func TestMain(m *testing.M) {
	// Adjust working dir to match what our services expect.
	os.Chdir("../../..")
	os.Exit(m.Run())
}
