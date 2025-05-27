package storage

import (
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
)

// NonInteractiveSessionID is a placeholder session ID for a non-interactive login session.
// We don't use `uuid.Nil` as a sentinel value to differentiate from uninitialized var bugs.
var NonInteractiveSessionID = uuid.Must(uuid.FromString("62d1c4d0-c8e6-4554-87ae-4ee2c7ba2225"))

// ErrCodeNotFound represents a missing OIDC code
var ErrCodeNotFound = ucerr.New("oidc auth code not found")

// we pause for a duration of mfaRetryTimeout after the last failure if
// mfaMaxFailures failures occur within a duration of mfaFailureWindow

// TODO: make these configurable
var mfaFailureWindow = time.Minute
var mfaRetryTimeout = time.Minute * 5

const mfaMaxFailures int = 5
