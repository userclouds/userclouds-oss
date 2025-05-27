package jsonclient

import (
	"fmt"
	"net/http"
)

// JCOAuthError lets us easily decode an oautherror response from a service we call,
// and keep track of the error code for clients
// NOTE: this is very similar but different to `ucerr.OAuthError` because our usage
// here is very limited (we're duplicating ~8 lines of code), and it prevents a weird
// pass-through bug when a UC service gets an OAuthError from another service (say that
// Plex calls Auth0 and gets an OAuthError), wraps it, and then accidentally returns the
// wrapped-service error code automatically (rather than the one Plex should set)
type oAuthError struct {
	ErrorType string `json:"error" yaml:"error"`
	ErrorDesc string `json:"error_description,omitempty" yaml:"error_description,omitempty"`
	Code      int    `json:"-" yaml:"-"`
}

// Error implements interface `error` for type `OAuthError`
func (o oAuthError) Error() string {
	return fmt.Sprintf("%s: %s [http status: %d]", o.ErrorType, o.ErrorDesc, o.Code)
}

// ErrIncorrectUsernamePassword indicates a bad username or password.
var ErrIncorrectUsernamePassword = oAuthError{
	ErrorType: "invalid_grant",
	ErrorDesc: "incorrect username or password",
	Code:      http.StatusBadRequest,
}
