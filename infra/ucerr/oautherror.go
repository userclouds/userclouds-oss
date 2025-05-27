package ucerr

import (
	"fmt"
	"net/http"
)

// Note to future self: we duplicate a small part of this code in jsonclient
// to avoid a case where if a handler calls another service via `jsonclient` and gets an
// OAuth-style error with a code (say, 404), and then calls MarshalError with the error it received,
// it would propagate the status code from the nested service call back out. Usually if a service
// gets a 404 while processing something complex, it may want to return a 400 or 500 or something else.
// We may want a `MarshalOAuthError` method instead of using this logic everywhere.

// OAuthError implements error but can be marshalled to JSON
// to make an OAuth/OIDC-compliant error.
// TODO: Should this be a private type like `ucError`?
type OAuthError struct {
	ErrorType  string `json:"error" yaml:"error"`
	ErrorDesc  string `json:"error_description,omitempty" yaml:"error_description,omitempty"`
	Code       int    `json:"-" yaml:"-"`
	underlying error
}

// Error implements interface `error` for type `OAuthError`
func (o OAuthError) Error() string {
	return fmt.Sprintf("%s: %s [http status: %d]", o.ErrorType, o.ErrorDesc, o.Code)
}

// Unwrap implements errors.Unwrap for errors.Is & errors.As
func (o OAuthError) Unwrap() error {
	return o.underlying // ok if this returns nil
}

// newWrappedOAuthError returns a new OAuthError wrapping a given error.
func newWrappedOAuthError(err error, errorType string, code int) error {
	return new(wrappedText, "", OAuthError{
		ErrorType:  errorType,
		ErrorDesc:  UserFriendlyMessage(err),
		Code:       code,
		underlying: err,
	}, 2, nil)
}

// For future reference, here's the mapping of other OAuth errors to HTTP codes
// that will likely become relevant to us:
// invalid_scope - http.StatusBadRequest
// invalid_client - http.StatusBadRequest or http.StatusUnauthorized (depending)
// insufficient_scope - http.StatusForbidden
// unauthorized_client - http.StatusForbidden
// It's probably worth double checking the spec, Auth0 compatibility, etc. when the time comes.

// ErrIncorrectUsernamePassword indicates a bad username or password.
var ErrIncorrectUsernamePassword = OAuthError{
	ErrorType: "invalid_grant",
	ErrorDesc: "incorrect username or password",
	Code:      http.StatusBadRequest,
}

// ErrInvalidAuthorizationCode indicates a bad authorization code.
var ErrInvalidAuthorizationCode = OAuthError{
	ErrorType: "invalid_grant",
	ErrorDesc: "invalid code",
	Code:      http.StatusBadRequest,
}

// ErrInvalidClientSecret indicates a bad client_secret.
var ErrInvalidClientSecret = OAuthError{
	ErrorType: "invalid_grant",
	ErrorDesc: "invalid client secret",
	Code:      http.StatusBadRequest,
}

// ErrInvalidAuthHeader indicates a bad HTTP Authorization header in an auth'd request.
var ErrInvalidAuthHeader = OAuthError{
	ErrorType: "invalid_token",
	ErrorDesc: "invalid 'Authorization' header",
	Code:      http.StatusUnauthorized,
}

// ErrInvalidCodeVerifier indicates a bad code_verifier argument in a Authorization Code w/PKCE login.
var ErrInvalidCodeVerifier = OAuthError{
	ErrorType: "invalid_grant",
	ErrorDesc: "invalid code verifier",
	Code:      http.StatusBadRequest,
}

// NewServerError returns a new internal server error.
func NewServerError(err error) error {
	return newWrappedOAuthError(err, "server_error", http.StatusInternalServerError)
}

// NewRequestError returns a new bad request error.
func NewRequestError(err error) error {
	return newWrappedOAuthError(err, "invalid_request", http.StatusBadRequest)
}

// NewUnsupportedGrantError returns a new error signifying an unsupported OAuth `grant_type`.
func NewUnsupportedGrantError(grant string) error {
	return new(wrappedText, "", OAuthError{
		ErrorType: "unsupported_grant_type",
		ErrorDesc: fmt.Sprintf("unsupported `grant_type` specified: %s", grant),
		Code:      http.StatusBadRequest,
	}, 1, nil)
}

// NewUnsupportedResponseError returns a new error signifying an unsupported OAuth `response_type`.
func NewUnsupportedResponseError(responseType string) error {
	return new(wrappedText, "", OAuthError{
		ErrorType: "unsupported_response_type",
		ErrorDesc: fmt.Sprintf("unsupported `response_type` specified: %s", responseType),
		Code:      http.StatusBadRequest,
	}, 1, nil)
}

// NewInvalidTokenError returns an error signifying a bad token of some kind.
func NewInvalidTokenError(err error) error {
	return newWrappedOAuthError(err, "invalid_token", http.StatusUnauthorized)
}

// NewInvalidClientError returns an error signifying a bad client ID.
func NewInvalidClientError(err error) error {
	return newWrappedOAuthError(err, "invalid_client", http.StatusBadRequest)
}
