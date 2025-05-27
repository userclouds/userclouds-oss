package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"userclouds.com/infra/assert"
)

func TestIPHostname(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {} // no-op
	sc := NewSecurityChecker()
	wrappedHandler := Middleware(sc).Apply(http.HandlerFunc(h))

	// test normal case
	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusOK)

	// test IP in host header, should "fail"
	req = httptest.NewRequest(http.MethodGet, "http://1.1.1.1", nil)
	rr = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// we expect 204 right now so as not to screw up our ALB health check metrics :/
	// when we have reasonable traffic volumes this maybe should 401?
	assert.Equal(t, rr.Code, http.StatusNoContent)
}
