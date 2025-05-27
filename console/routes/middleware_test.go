package routes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/kubernetes"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/internal/uctest"
)

func TestHTTPHostInjection(t *testing.T) {
	// Create a request with a Host header that is not the console URL
	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.Host = "otherexample.com"
	u := uctest.MustParseURL("example.com")
	rr := httptest.NewRecorder()
	handler := rejectNonConsoleHostMiddleware(func() *url.URL { return u }).Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusBadRequest)
	assert.Equal(t, rr.Body.String(), "Invalid host otherexample.com\n")
}

func TestAllowHosts(t *testing.T) {
	// regular console URLs, non regional
	runRequest(t, "http://console.debug.userclouds.com", "console.debug.userclouds.com")
	runRequest(t, "http://console.userclouds.com", "console.userclouds.com")
	runRequest(t, "http://console.staging.userclouds.com", "console.staging.userclouds.com")
	runRequest(t, "https://console.dev.userclouds.tools:3333", "console.dev.userclouds.tools:3333")

	// Any cloud Universe is fine since we want to test against all AWS regions
	for _, rg := range region.MachineRegionsForUniverse(universe.Debug) {
		t.Setenv(region.RegionEnvVar, string(rg))
		t.Setenv(kubernetes.EnvIsKubernetes, "false")
		runRequest(t, "http://console.debug.userclouds.com", fmt.Sprintf("console.%v.debug.userclouds.com", rg))
		runRequest(t, "http://console.userclouds.com", fmt.Sprintf("console.%v.userclouds.com", rg))
		runRequest(t, "http://console.staging.userclouds.com", fmt.Sprintf("console.%v.staging.userclouds.com", rg))

		// EKS
		t.Setenv(kubernetes.EnvIsKubernetes, "true")
		runRequest(t, "http://console.debug.userclouds.com", fmt.Sprintf("console.%v-eks.debug.userclouds.com", rg))
		runRequest(t, "http://console.userclouds.com", fmt.Sprintf("console.%v-eks.userclouds.com", rg))
		runRequest(t, "http://console.staging.userclouds.com", fmt.Sprintf("console.%v-eks.staging.userclouds.com", rg))
	}
}

func runRequest(t *testing.T, consoleURL, host string) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	t.Run(host, func(t *testing.T) {
		cb := func() *url.URL { return uctest.MustParseURL(consoleURL) }
		req := httptest.NewRequest("GET", consoleURL, nil)
		req.Host = host

		rr := httptest.NewRecorder()
		handler := rejectNonConsoleHostMiddleware(cb).Apply(handler)
		handler.ServeHTTP(rr, req)
		assert.Equal(t, rr.Code, http.StatusOK, assert.Errorf("request failed for console URL: %v, host: %v", consoleURL, host))
	})
}
