package jsonapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/request"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
)

// ErrorReportingFunc is a function that reports errors
type ErrorReportingFunc interface {
	ReportingFunc() func(context.Context, error)
}

var errorReporter ErrorReportingFunc

// SetErrorReporter sets the error reporter function
// this exists so that we can share the same error
// reporter between jsonapi and uchttp
func SetErrorReporter(f ErrorReportingFunc) {
	errorReporter = f
}

// JSONErrorMessage is a struct that is returned an HTTP error response body
type JSONErrorMessage struct {
	Error     any       `json:"error" yaml:"error"`
	RequestID uuid.UUID `json:"request_id" yaml:"request_id"`
}

type errorWithCode interface {
	Error() string
	Code() int
}

// MarshalErrorL os temporary wrapper that allows for conversion of all the call sites
func MarshalErrorL(ctx context.Context, w http.ResponseWriter, err error, errorName string, os ...Option) {
	// capture the caller's stack frame
	// TODO: like in uchttp.ErrorL, we double-capture stack here for the transition period
	err = ucerr.Wrap(err, ucerr.ExtraSkip())

	// TODO temporary check for during migration
	if errorName == "" {
		uclog.Warningf(ctx, "Error with blank error counter name that will not be counted in %s with %v", uclog.GetHandlerName(ctx), err)
	}
	uclog.SetHandlerErrorName(ctx, errorName)
	MarshalError(ctx, w, err, os...)
}

func hasCode(os []Option) bool {
	for _, o := range os {
		if _, ok := o.(codeOpt); ok {
			return true
		}
	}
	return false
}

// helper to not duplicate-append Code() option
func appendCode(ctx context.Context, os []Option, code int) []Option {
	appendCode := true
	if hasCode(os) {
		// An http status code was provided as an option, which is odd, but we'll allow it and warn.
		uclog.Errorf(
			ctx,
			"jsonapi.MarshalError called with options that already include a status code (%d): %+v. The provided error type's code will not override this.",
			os,
			code)
		appendCode = false
	}
	if appendCode {
		// Only add a Code() option if none was specified by caller.
		os = append(os, Code(code))
	}
	return os
}

// MarshalError sends a nicely encoded go error in JSON
// Passing jsonapi.Code(XXX) will override the default HTTP 500
// Shouldn't ever be called with a nil error, but won't crash
// In the case of a wrapper error, only includes the base error externally
func MarshalError(ctx context.Context, w http.ResponseWriter, err error, os ...Option) {
	// capture the caller's stack frame
	err = ucerr.Wrap(err, ucerr.ExtraSkip())

	var ewn ucerr.ErrorWithName
	if errors.As(err, &ewn) {
		uclog.SetHandlerErrorName(ctx, ewn.Name())
	}

	var oauthe ucerr.OAuthError
	var r any
	if errors.As(err, &oauthe) {
		// We don't sanitize these errors (which is mostly correct because they're required by the
		// OAuth spec to have certain fields)
		r = oauthe
		os = appendCode(ctx, os, oauthe.Code)
		uclog.Debugf(ctx, "did not sanitize oauth error")
	} else {
		var s any
		// see if the error should be marshalled as a JSON Object
		s = ucerr.UserFriendlyStructure(err)
		if s == nil {
			// otherwise just return the user-friendly error string
			s = ucerr.UserFriendlyMessage(err)
		}

		uclog.Debugf(ctx, "friendly error response: %s", s)
		r = JSONErrorMessage{s, request.GetRequestID(ctx)}

		var ewc errorWithCode
		if errors.As(err, &ewc) {
			os = append(os, Code(ewc.Code()))
		} else if ucdb.IsUniqueViolation(err) && !hasCode(os) {
			// TODO: This may not be correct for all cases, in some cases we may want to return 500
			os = appendCode(ctx, os, http.StatusConflict)
		}
	}

	// This generally happens because the client closed the connection before request processing
	// finished. We don't want to treat that as a server error in our metrics (although we should
	// track that separately for timeout awareness esp WN)
	if ucerr.IsContextCanceledError(err) {
		// NB: we use Code() option here appended at the end to ensure we override any other
		// Code() options passed by the caller. We could also do this after the `options`
		// computation below, but this seems more consistent to refactoring
		os = append(os, Code(uchttp.StatusClientClosedConnectionError))
	}

	// need to compute the status code before we can log the error with status
	options := applyOptions(os...)
	if options.code == -1 {
		// default inside MarshalError should be 500
		options.code = http.StatusInternalServerError
	}

	logError(ctx, err, options.code)

	// if this is a 500, send it on for analysis (we don't yet care about 4XXs)
	if options.code == http.StatusInternalServerError {
		if errorReporter != nil {
			errorReporter.ReportingFunc()(ctx, err)
		}
	}

	marshal(w, r, options)
}

// MarshalSQLError is a nice wrapper for SQL errors that
// handles 404s for ErrNoRows, 500 otherwise
func MarshalSQLError(ctx context.Context, w http.ResponseWriter, err error) {
	// capture the caller's stack frame
	err = ucerr.Wrap(err, ucerr.ExtraSkip())

	if errors.Is(err, sql.ErrNoRows) {
		MarshalError(ctx, w, ucerr.Friendlyf(err, "not found"), Code(http.StatusNotFound))
	} else {
		MarshalError(ctx, w, err)
	}
}

// make sure we log the full error to our logs for debugging
func logError(ctx context.Context, err error, statusCode int) {
	if statusCode >= 400 && statusCode < 500 {
		uclog.Warningf(ctx, "HTTP %d error: %v", statusCode, err)
	} else {
		uclog.Errorf(ctx, "unhandled error (HTTP %v): %v", statusCode, err)
	}
}
