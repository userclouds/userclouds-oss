package ucerr_test

import (
	"errors"
	"testing"

	"userclouds.com/infra/assert"
	. "userclouds.com/infra/ucerr"
)

func TestOAuthErrorIs(t *testing.T) {
	base := errors.New("important error") // lint: ucwrapper-safe
	wrapped := NewServerError(base)

	assert.ErrorIs(t, wrapped, base)
}

func TestOAuthErrorAs(t *testing.T) {
	// `customError` is from `errors_test.go`
	base := &customError{s: "custom error type"}
	wrapped := NewServerError(base)

	var asBase *customError
	// errors.As() should extract the base type.
	assert.True(t, errors.As(wrapped, &asBase))
	assert.NotEqual(t, base.Error(), wrapped.Error())
	assert.Equal(t, base.Error(), asBase.Error())
}

func TestOAuthErrorString(t *testing.T) {
	base := OAuthError{
		ErrorType: "some_error",
		ErrorDesc: "an error happened",
		Code:      500,
	}
	assert.Equal(t, base.Error(), "some_error: an error happened [http status: 500]")
}

func TestNewUnsupportedGrantError(t *testing.T) {
	err := NewUnsupportedGrantError("foo")
	// This is testing an error type that directly calls `ucerr.new` to cons the error.
	assert.Equal(t, err.Error(), `unsupported_grant_type: unsupported `+"`grant_type`"+` specified: foo [http status: 400]
(wrapped) (File infra/ucerr/oautherror_test.go:40, in TestNewUnsupportedGrantError)`)
}

func TestNewRequestError(t *testing.T) {
	err := NewRequestError(Friendlyf(nil, "some underlying error")) // lint: ucwrapper-safe
	// This is testing an error type that indirectly calls `ucerr.new` (via newWrappedOAuthError) to cons the error.
	assert.Equal(t, err.Error(), `invalid_request: some underlying error [http status: 400]
(wrapped) (File infra/ucerr/oautherror_test.go:47, in TestNewRequestError)`)
}
