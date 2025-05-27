package ucjson

import (
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucjson/ucjsontest"
)

func TestMarshalUnmarshal(t *testing.T) {
	// native struct
	type Test struct {
		ID string `json:"id"`
	}

	jsonData := []byte(`{"id":"123"}`)

	var test Test
	assert.NoErr(t, Unmarshal(jsonData, &test))
	assert.Equal(t, test.ID, "123")

	newJSONData, err := Marshal(test)
	assert.NoErr(t, err)
	assert.Equal(t, newJSONData, jsonData)

	// easyjson struct FIXME
	jsonData = []byte(`{"id":"123"}`)

	var easyjsonTest ucjsontest.EasyJSONTest
	assert.NoErr(t, Unmarshal(jsonData, &easyjsonTest))
	assert.Equal(t, easyjsonTest.ID, "123")

	newJSONData, err = Marshal(easyjsonTest)
	assert.NoErr(t, err)
	assert.Equal(t, newJSONData, jsonData)
}
