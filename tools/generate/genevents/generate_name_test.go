package genevents

import (
	"testing"

	"userclouds.com/infra/assert"
)

func TestGenerateName(t *testing.T) {
	assert.Equal(t, generateName("GetUser"), "Get User")
	assert.Equal(t, generateName("GetJSONKey"), "Get JSON Key")
	assert.Equal(t, generateName("GetIDForUser"), "Get ID For User")
	assert.Equal(t, generateName("getID"), "Get ID")
}
