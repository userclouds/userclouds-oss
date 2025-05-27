package jsonapi_test

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/request"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/test/testlogtransport"
)

func TestMarshalError(t *testing.T) {
	rid := uuid.Must(uuid.NewV4())
	ctx := request.SetRequestIDIfNotSet(context.Background(), rid)

	base := ucerr.New("test")

	unspecifiedError := fmt.Sprintf(`{"error":"an unspecified error occurred","request_id":"%v"}`, rid)
	unknownError := fmt.Sprintf(`{"error":"an unknown error occurred","request_id":"%v"}`, rid)

	rr := httptest.NewRecorder()
	jsonapi.MarshalError(ctx, rr, base)
	assert.Equal(t, rr.Code, http.StatusInternalServerError)
	b, err := io.ReadAll(rr.Body)
	assert.NoErr(t, err)
	assert.Equal(t, strings.TrimSpace(string(b)), unspecifiedError)

	// make sure we don't disclose internal stack traces
	wrapped := ucerr.Wrap(base)
	rr = httptest.NewRecorder()
	jsonapi.MarshalError(ctx, rr, wrapped, jsonapi.Code(http.StatusForbidden))
	assert.Equal(t, rr.Code, http.StatusForbidden)
	b, err = io.ReadAll(rr.Body)
	assert.NoErr(t, err)
	assert.Equal(t, strings.TrimSpace(string(b)), unspecifiedError)

	// make sure we don't crash on nil
	rr = httptest.NewRecorder()
	jsonapi.MarshalError(ctx, rr, nil)
	assert.Equal(t, rr.Code, http.StatusInternalServerError)
	b, err = io.ReadAll(rr.Body)
	assert.NoErr(t, err)
	assert.Equal(t, strings.TrimSpace(string(b)), unknownError)
}

func TestStackCapture(t *testing.T) {
	ctx := context.Background()

	tt := testlogtransport.InitLoggerAndTransportsForTests(t)

	err := ucerr.Wrap(ucerr.New("test error"))
	rr := httptest.NewRecorder()
	// wrap in another function that we should track
	func() {
		jsonapi.MarshalError(ctx, rr, err)
	}()

	// make sure the calling function is in the stack trace
	tt.AssertLogsContainString("in func1")

	rr = httptest.NewRecorder()
	// wrap in another function that we should track
	func() {
		jsonapi.MarshalErrorL(ctx, rr, err, "foo")
	}()

	// make sure the calling function is in the stack trace
	tt.AssertLogsContainString("in func2")

	rr = httptest.NewRecorder()
	// wrap in another function that we should track
	func() {
		jsonapi.MarshalError(ctx, rr, err)
	}()

	// make sure the calling function is in the stack trace
	tt.AssertLogsContainString("in func3")

	rr = httptest.NewRecorder()
	// wrap in another function that we should track
	func() {
		jsonapi.MarshalErrorL(ctx, rr, err, "foo")
	}()

	// make sure the calling function is in the stack trace
	tt.AssertLogsContainString("in func4")

	rr = httptest.NewRecorder()
	// wrap in another function that we should track
	func() {
		jsonapi.MarshalSQLError(ctx, rr, err)
	}()

	// make sure the calling function is in the stack trace
	tt.AssertLogsContainString("in func5")
}

// this has to be defined external to the test function so we can hang Validate on it
type jsonMarshalTestStruct struct {
	ID int `json:"id"`
}

func (f jsonMarshalTestStruct) Validate() error {
	if f.ID == 0 {
		return ucerr.Friendlyf(nil, "id is zero")
	}
	if err := f.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

func (f jsonMarshalTestStruct) extraValidate() error {
	if f.ID == 1 {
		return ucerr.Friendlyf(nil, "id is one")
	} else if f.ID == 2 {
		return ucerr.New("id is two")
	}
	return nil
}

func TestMarshalingJSONErrors(t *testing.T) {
	ctx := context.Background()

	type testcase struct {
		name  string
		input string
		valid bool // is the input valid, even if we always JSONMarshalError the result
		err   string
		code  int
	}

	// input to expected error
	tcs := []testcase{
		{
			name:  "valid input, unknown error",
			input: `{"id": 3}`,
			valid: true,
			err:   `{"error":"an unknown error occurred","request_id":"00000000-0000-0000-0000-000000000000"}`,
			code:  http.StatusInternalServerError,
		},
		{
			name:  "validation error",
			input: `{"id": 0}`,
			err:   `{"error":"id is zero","request_id":"00000000-0000-0000-0000-000000000000"}`,
			code:  http.StatusBadRequest,
		},
		{
			name:  "wrong type in struct",
			input: `{"id": "foo"}`, // wrong type to unmarshal
			err:   `{"error":"json: cannot unmarshal string into Go struct field jsonMarshalTestStruct.id of type int","request_id":"00000000-0000-0000-0000-000000000000"}`,
			code:  http.StatusBadRequest,
		},
		{
			name:  "extra validation friendly error",
			input: `{"id": 1}`,
			err:   `{"error":"id is one","request_id":"00000000-0000-0000-0000-000000000000"}`,
			code:  http.StatusBadRequest,
		},
		{
			// NB: this is a design decision to ensure we never return errors that weren't marked
			// Friendly ... we used to use BaseError(err).Error() to override this behavior in Validate
			// calls, but that seems risky and introduced confusion when we *did* use Friendly
			name:  "extra validation non-friendly error",
			input: `{"id": 2}`,
			err:   `{"error":"an unspecified error occurred","request_id":"00000000-0000-0000-0000-000000000000"}`,
			code:  http.StatusBadRequest,
		}}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(tc.input)))

			var f jsonMarshalTestStruct
			err := jsonapi.Unmarshal(r, &f)
			if tc.valid {
				assert.NoErr(t, err)
			} else {
				assert.NotNil(t, err, assert.Must())
			}

			rr := httptest.NewRecorder()
			jsonapi.MarshalError(ctx, rr, err)

			assert.Equal(t, rr.Code, tc.code)
			bs, err := io.ReadAll(rr.Body)
			assert.NoErr(t, err)
			assert.Equal(t, strings.TrimSpace(string(bs)), tc.err)
		})
	}
}

func TestContextCanceledHandling(t *testing.T) {
	ctx := context.Background()
	rr := httptest.NewRecorder()
	err := context.Canceled
	jsonapi.MarshalError(ctx, rr, err)
	assert.Equal(t, rr.Code, uchttp.StatusClientClosedConnectionError)

	h := func(w http.ResponseWriter, r *http.Request) {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusConflict))
	}
	rr = httptest.NewRecorder()
	h(rr, nil)
	assert.Equal(t, rr.Code, uchttp.StatusClientClosedConnectionError)
}

func TestMarshalSQLError(t *testing.T) {
	err := ucerr.Wrap(sql.ErrNoRows)

	rr := httptest.NewRecorder()
	jsonapi.MarshalSQLError(context.Background(), rr, err)
	assert.Equal(t, rr.Code, http.StatusNotFound)
	b, err := io.ReadAll(rr.Body)
	assert.NoErr(t, err)
	assert.Equal(t, strings.TrimSpace(string(b)), `{"error":"not found","request_id":"00000000-0000-0000-0000-000000000000"}`)
}

type testReporter struct {
	t *testing.T
}

func (tr testReporter) ReportingFunc() func(context.Context, error) {
	return func(ctx context.Context, err error) {
		assert.FailContinue(tr.t, "should not be called")
	}
}

// regression for https://usercloudsworkspace.slack.com/archives/C02A3HELPPU/p1699375101059129
func TestMarshalErrorCodes(t *testing.T) {
	ctx := context.Background()

	rr := httptest.NewRecorder()
	err := ucerr.New("testing")

	r := testReporter{t}
	jsonapi.SetErrorReporter(r)
	jsonapi.MarshalError(ctx, rr, err, jsonapi.Code(http.StatusConflict))
}
