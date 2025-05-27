package jsonapi_test

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/request"
	"userclouds.com/infra/ucerr"
	"userclouds.com/test/testlogtransport"
)

type test struct {
	Foo string `json:"foo"`
}

type testV struct {
	Foo string `json:"foo"`
}

func (v testV) Validate() error {
	if v.Foo == "" {
		return ucerr.New("foo")
	}
	return nil
}

func TestUnmarshal(t *testing.T) {
	b := []byte(`{"foo":"bar"}`)
	req, err := http.NewRequest("POST", "localhost", bytes.NewReader(b))
	assert.NoErr(t, err)

	var f test
	assert.IsNil(t, jsonapi.Unmarshal(req, &f))
	assert.Equal(t, f.Foo, "bar")

	b = []byte(`{"foo":""}`)
	req, err = http.NewRequest("POST", "localhost", bytes.NewReader(b))
	assert.NoErr(t, err)

	var v testV
	assert.NotNil(t, jsonapi.Unmarshal(req, &v))
}

func TestMarshal(t *testing.T) {
	s := test{"baz"}

	rr := httptest.NewRecorder()
	jsonapi.Marshal(rr, s)
	assert.Equal(t, rr.Code, http.StatusOK)
	b, err := io.ReadAll(rr.Body)
	assert.NoErr(t, err)
	assert.Equal(t, rr.Header()["Content-Type"], []string{"application/json"})
	assert.Equal(t, strings.TrimSpace(string(b)), `{"foo":"baz"}`)

	// check Code()
	// NB: this test doesn't actually check the ordering of .WriteHeader vs .Header().Add()
	// effectively, since order doesn't seem to matter on ResponseRecorder but does on ResponseWriter
	rr = httptest.NewRecorder()
	jsonapi.Marshal(rr, s, jsonapi.Code(http.StatusCreated))
	assert.Equal(t, rr.Code, http.StatusCreated)
	b, err = io.ReadAll(rr.Body)
	assert.NoErr(t, err)
	assert.Equal(t, rr.Header()["Content-Type"], []string{"application/json"})
	assert.Equal(t, strings.TrimSpace(string(b)), `{"foo":"baz"}`)

	// check empty arrays
	rr = httptest.NewRecorder()
	var arr []int
	jsonapi.Marshal(rr, arr)
	assert.Equal(t, rr.Code, http.StatusOK)
	b, err = io.ReadAll(rr.Body)
	assert.NoErr(t, err)
	assert.Equal(t, rr.Header()["Content-Type"], []string{"application/json"})
	assert.Equal(t, strings.TrimSpace(string(b)), `[]`)
}

func TestMarshalFailure(t *testing.T) {
	// json.Marshal (or in this case, enc.Encode) can fail for a couple of reasons ... this link is helpful
	// https://stackoverflow.com/questions/33903552/what-input-will-cause-golangs-json-marshal-to-return-an-error

	tt := testlogtransport.InitLoggerAndTransportsForTests(t)
	rr := httptest.NewRecorder()
	jsonapi.Marshal(rr, make(chan int))
	tt.AssertLogsContainString("jsonapi.Marshal failed on request : json: unsupported type: chan int")

	tt = testlogtransport.InitLoggerAndTransportsForTests(t)
	rr = httptest.NewRecorder()
	jsonapi.Marshal(rr, math.Inf(1))
	tt.AssertLogsContainString("jsonapi.Marshal failed on request : json: unsupported value: +Inf")

	// check request ID logging ... NB the tests above ensure it doesn't break if the middleware isn't present
	h := request.Middleware().Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonapi.Marshal(w, make(chan int))
	}))

	tt = testlogtransport.InitLoggerAndTransportsForTests(t)
	rr = httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rr, r)
	assert.Equal(t, rr.Code, http.StatusOK) // NB: 200 because we've already written headers when it fails :)
	rid := request.GetRequestIDFromHeader(rr.Header())
	tt.AssertLogsContainString(fmt.Sprintf("jsonapi.Marshal failed on request %v: json: unsupported type: chan int", rid))
}
