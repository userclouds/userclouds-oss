package testhelpers

import (
	"context"
	"net/http"

	"userclouds.com/internal/security"
)

// SecurityValidator implements security.ReqValidator and is a test-only validator for Plex tests
type SecurityValidator struct {
}

// ValidateRequest implements security.ReqValidator
func (*SecurityValidator) ValidateRequest(r *http.Request) (security.Status, error) {
	return security.Status{}, nil
}

// FinishedRequest implements security.ReqValidator
func (*SecurityValidator) FinishedRequest(status security.Status, r *http.Request, success bool) {
	// Do nothing
}

// IsCallBlocked implements security.ReqValidator
func (*SecurityValidator) IsCallBlocked(ctx context.Context, username string) bool {
	return false
}
