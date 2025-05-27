package test

import (
	"io"
	"net/http"
	"net/http/httptest"

	"userclouds.com/internal/tenantplex"
)

// RequestFactory understands TenantConfigs such that it can set the host header correctly
type RequestFactory struct {
	tcs      tenantplex.TenantConfigs
	hostName string
}

// NewRequest wraps httptest.NewRequest and just sets Host correctly
func (rf RequestFactory) NewRequest(method, url string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, url, body)
	req.Host = rf.hostName
	return req
}
