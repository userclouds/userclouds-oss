package routing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/request"
)

func testURL(t *testing.T, url string, expectedHost string, expectedPath string) {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()

	jsonclient.Router = serviceRouter{}
	request.Middleware().Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonclient.Router.Reroute(r.Context(), r)
		assert.Equal(t, r.URL.Host, expectedHost)
		assert.Equal(t, r.URL.Path, expectedPath)
	})).ServeHTTP(w, req)
}

func TestReroute(t *testing.T) {
	testURL(t, "https://test.tenant.userclouds.com/authz/foo", "localhost:5200", "/authz/foo")
	testURL(t, "https://test.tenant.userclouds.com/authn/foo", "localhost:5100", "/authn/foo")
	testURL(t, "https://test.tenant.userclouds.com/logserver/foo", "localhost:5500", "/logserver/foo")
	testURL(t, "https://test.tenant.userclouds.com/random", "localhost:5000", "/random")
}

func TestRerouteForOnPrem(t *testing.T) {
	t.Setenv(universe.EnvKeyUniverse, "onprem")
	t.Setenv("K8S_POD_NAMESPACE", "seinfeld")
	testURL(t, "https://test.tenant.userclouds.io/authz/foo", "userclouds-authz.seinfeld.svc.cluster.local:80", "/authz/foo")
	testURL(t, "https://test.tenant.userclouds.io/authn/foo", "userclouds-userstore.seinfeld.svc.cluster.local:80", "/authn/foo")
	testURL(t, "https://test.tenant.userclouds.io/logserver/foo", "userclouds-logserver.seinfeld.svc.cluster.local:80", "/logserver/foo")
	testURL(t, "https://test.tenant.userclouds.io/random", "userclouds-plex.seinfeld.svc.cluster.local:80", "/random")
}

func TestRerouteOnlyToSelf(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://test.tenant.userclouds.com/authz/foo", nil)
	w := httptest.NewRecorder()

	jsonclient.Router = serviceRouter{}
	request.Middleware().Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subserviceRequest := httptest.NewRequest(http.MethodGet, "https://userclouds.us.auth0.com/oidc/token", nil)
		jsonclient.Router.Reroute(r.Context(), subserviceRequest)
		assert.Equal(t, subserviceRequest.URL.Host, "userclouds.us.auth0.com")
		assert.Equal(t, subserviceRequest.URL.Path, "/oidc/token")
	})).ServeHTTP(w, req)
}
