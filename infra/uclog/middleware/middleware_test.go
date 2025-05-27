package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/uclog/responsewriter"
)

// this test ensures I don't break loggingMiddleware.WriteHeader again, but it doesn't
// actually confirm the log line is correct. Without adding more test-only overhead to logger,
// I didn't see an easy way to test this yet.
func TestMiddleware(t *testing.T) {
	mw := HTTPLoggerMiddleware()

	h := func(code int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
			fmt.Fprintf(w, "%d", code)
		})
	}

	for _, code := range []int{http.StatusOK, http.StatusInternalServerError} {
		handler := mw.Apply(h(code))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		lrw := responsewriter.StatusResponseWriter{ResponseWriter: rr}
		handler.ServeHTTP(&lrw, req)
		assert.Equal(t, lrw.StatusCode, code)
	}
}
