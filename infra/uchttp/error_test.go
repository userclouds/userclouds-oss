package uchttp_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucerr"
	. "userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/test/testlogtransport"
)

func TestErrorSanitizer(t *testing.T) {
	ctx := context.Background()

	err := ucerr.Friendlyf(
		ucerr.Wrap(
			errors.New("base error"), // lint: ucwrapper-safe
		),
		"this is safe")

	rr := httptest.NewRecorder()
	Error(ctx, rr, err, http.StatusInternalServerError)

	assert.Equal(t, rr.Code, http.StatusInternalServerError)
	assert.Equal(t, strings.TrimSpace(rr.Body.String()), "this is safe")

	rr = httptest.NewRecorder()
	err = ucerr.Wrap(ucerr.New("test"))
	Error(ctx, rr, err, http.StatusInternalServerError)
	assert.Equal(t, rr.Code, http.StatusInternalServerError)
	assert.Equal(t, strings.TrimSpace(rr.Body.String()), "an unspecified error occurred")
}

func TestTenantErrorMuting(t *testing.T) {
	ctx := context.Background()

	tt := testlogtransport.InitLoggerAndTransportsForTests(t)

	err := ucerr.Errorf("generated in tenantmap: %w", ucerr.NewWarning("invalid tenant name"))

	rr := httptest.NewRecorder()
	Error(ctx, rr, err, http.StatusInternalServerError)
	tt.AssertMessagesByLogLevel(uclog.LogLevelWarning, 0)
	tt.AssertMessagesByLogLevel(uclog.LogLevelError, 1)
}

func TestErrorWrapping(t *testing.T) {
	ctx := context.Background()

	tt := testlogtransport.InitLoggerAndTransportsForTests(t)

	err := ucerr.Wrap(ucerr.New("test error"))
	rr := httptest.NewRecorder()
	// wrap in another function that we should track
	func() {
		Error(ctx, rr, err, http.StatusForbidden)
	}()

	// make sure the calling function is in the stack trace
	tt.AssertLogsContainString("in func1")

	rr = httptest.NewRecorder()
	func() {
		ErrorL(ctx, rr, err, http.StatusForbidden, "foo")
	}()

	// make sure the calling function is in the stack trace
	tt.AssertLogsContainString("in func2")
}

func TestErrorWarning(t *testing.T) {
	ctx := context.Background()

	tt := testlogtransport.InitLoggerAndTransportsForTests(t)

	rr := httptest.NewRecorder()

	err := ucerr.NewWarning("foo")
	werr := ucerr.Wrap(err)
	func() {
		Error(ctx, rr, werr, http.StatusForbidden)
	}()

	var warn ucerr.Warning
	assert.True(t, errors.As(err, &warn))
	assert.Equal(t, warn.Error(), "foo\nfoo (File infra/ucerr/warning.go:16, in NewWarning)")

	// wrapping shouldn't change this
	assert.True(t, errors.As(werr, &warn))
	assert.Equal(t, warn.Error(), "foo\nfoo (File infra/ucerr/warning.go:16, in NewWarning)")

	assert.Equal(t, rr.Code, http.StatusForbidden)
	tt.AssertMessagesByLogLevel(uclog.LogLevelWarning, 1)
	tt.AssertMessagesByLogLevel(uclog.LogLevelError, 0)
	tt.AssertLogsContainString("HTTP 403 error: foo")
}

func TestContextCanceledHandling(t *testing.T) {
	ctx := context.Background()
	tt := testlogtransport.InitLoggerAndTransportsForTests(t)
	rr := httptest.NewRecorder()

	// make this a slightly more involved test to get more realism
	tdb := testdb.New(t)

	wg := sync.WaitGroup{}
	wg.Add(1)
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		var unused bool
		err := tdb.GetContext(ctx, "t", &unused, "SELECT PG_SLEEP(10);")
		assert.NotNil(t, err)
		assert.ErrorIs(t, err, context.Canceled)
		err = ucerr.Wrap(err)
		Error(ctx, rr, err, http.StatusInternalServerError)
		wg.Done()
	}()

	cancel()

	wg.Wait()
	assert.Equal(t, rr.Code, StatusClientClosedConnectionError)
	tt.AssertLogsContainString("context canceled", "query execution canceled")
}

func TestClientDisconnectHandling(t *testing.T) {
	tdb := testdb.New(t)

	rr := httptest.NewRecorder()
	connEstablished := make(chan bool)
	reqCanceled := make(chan bool)
	respSent := make(chan bool)

	// Server side: wait for the connection to be closed by the client, then make a db query (which
	// should immediately fail due to the context being canceled) and send the error response to the
	// httptest recorder
	server := httptest.NewServer(http.HandlerFunc(func(_w http.ResponseWriter, r *http.Request) {
		connEstablished <- true
		<-reqCanceled
		_, err := tdb.ExecContext(r.Context(), "sleep_second", "SELECT pg_sleep(1)")
		if err != nil {
			// Intentionally using rr instead of w, since the client will
			// disconnect before it gets any response, so we won't be able to
			// see what goes there
			Error(r.Context(), rr, err, http.StatusInternalServerError)
		} else {
			rr.WriteHeader(http.StatusOK)
		}
		respSent <- true
	}))
	defer server.Close()

	// Client side: send a request, wait to ensure that it is received by the server, then close the
	// connection so that the server's request context is canceled.
	reqCtx, cancelReq := context.WithCancel(context.Background())
	go func() {
		<-connEstablished
		cancelReq()
	}()
	req, err := http.NewRequest("GET", server.URL, nil)
	assert.NoErr(t, err)
	req = req.WithContext(reqCtx)
	_, err = http.DefaultClient.Do(req)
	assert.NotNil(t, err)
	reqCanceled <- true

	// Make sure the error written by the server is a 499
	<-respSent
	assert.Equal(t, rr.Code, StatusClientClosedConnectionError)
}
