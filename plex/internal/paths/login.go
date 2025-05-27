package paths

import (
	"context"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/plex/internal/reactdev"
)

// LoginURL returns the base login URL for Plex's "universal login" screen
func LoginURL(ctx context.Context, sessionID uuid.UUID) (*url.URL, error) {
	// Redirect to login page which allows for username/password, passwordless, and social login. The actual method
	// of user authentication/credentialing is orthogonal to how the caller invokes the /authorize endpoint.
	redirectTo := reactdev.UIBaseURL(ctx)
	redirectTo.Path = redirectTo.Path + LoginUISubPath
	query := url.Values{}
	query.Set("session_id", sessionID.String())
	redirectTo.RawQuery = query.Encode()
	return redirectTo, nil
}

// PasswordlessLoginURL returns the passwordless login URL
func PasswordlessLoginURL(ctx context.Context, sessionID uuid.UUID) (*url.URL, error) {
	redirectTo := reactdev.UIBaseURL(ctx)
	redirectTo.Path = redirectTo.Path + PasswordlessLoginUISubPath
	query := url.Values{}
	query.Set("session_id", sessionID.String())
	redirectTo.RawQuery = query.Encode()
	return redirectTo, nil
}
