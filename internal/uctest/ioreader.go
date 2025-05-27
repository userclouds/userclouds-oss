package uctest

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"userclouds.com/infra/assert"
)

// IOReaderFromJSONStruct is a helper to JSON serialize a structure into a buffer
// and return an io.Reader which reads from the buffer (useful for testing http requests).
func IOReaderFromJSONStruct(t *testing.T, s any) io.Reader {
	bs, err := json.Marshal(s)
	assert.NoErr(t, err)
	return bytes.NewReader(bs)
}
