package pagetype_test

import (
	"os"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/internal/pageparameters/pagetype"
	param "userclouds.com/internal/pageparameters/parameter"
)

func TestPageTypeValidity(t *testing.T) {
	for _, pt := range pagetype.Types() {
		assert.IsNil(t, pt.Validate())
	}

	var badType pagetype.Type = "bad page type"
	assert.NotNil(t, badType.Validate())
}

func TestSupportedParameters(t *testing.T) {
	for _, pt := range pagetype.Types() {
		for _, pn := range pt.ParameterNames() {
			assert.True(t, pt.SupportsParameterName(pn))
		}
		assert.False(t, pt.SupportsParameterName("bad parameter name"))
	}
}

func validateTestParameters(t *testing.T, paramNames []param.Name, paramsByName map[param.Name]param.Parameter) {
	t.Helper()

	for _, p := range paramsByName {
		assert.IsNil(t, p.Validate())
	}

	assert.Equal(t, len(paramNames), len(paramsByName))
	for _, pn := range paramNames {
		_, found := paramsByName[pn]
		assert.True(t, found)
		delete(paramsByName, pn)
	}
	assert.Equal(t, len(paramsByName), 0)
}

func TestPageTypeTestParameters(t *testing.T) {
	for _, pt := range pagetype.Types() {
		validateTestParameters(t, pt.ParameterNames(), pt.TestParameters())
		validateTestParameters(t, pt.RenderParameterNames(), pt.TestRenderParameters())
	}
}

func TestMain(m *testing.M) {
	// Adjust working dir to match what our services expect.
	os.Chdir("../../..")
	os.Exit(m.Run())
}
