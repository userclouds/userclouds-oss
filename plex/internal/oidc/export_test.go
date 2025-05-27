package oidc

import (
	"context"
	"net/http"
	"net/url"

	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider"
)

// NewTestHandler is exported only for testing (in _test.go files)
func NewTestHandler(pf provider.Factory) *Handler {
	return &Handler{factory: pf}
}

// ValidateClient is exported only for testing (in _test.go files)
func ValidateClient(ctx context.Context, r *http.Request, pf *url.Values) (*tenantplex.App, error) {
	return validateClient(ctx, r, pf)
}
