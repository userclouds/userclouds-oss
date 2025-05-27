package uchttp

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/url"

	"userclouds.com/infra/request"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctrace"
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

// StatusClientClosedConnectionError should be returned when the client closed the connection before request processing
// finished. It's not an official status code, but was popularized by nginx
const StatusClientClosedConnectionError = 499

// Not sure this is the right name for this package, or that it really
// needs to exist, but I wanted to get rid of http.Error() in favor of
// a method we control so that we can log errors, and this was an
// easy way to replace it. It's basically servicebase / serviceframework,
// but service.Error() didn't immediately seem right. Easy to move there
// later if we want, etc. (and it does imitate the golang http package)

// Error wraps http.Error so we can log the errors for later fixing
func Error(ctx context.Context, w http.ResponseWriter, err error, code int) {
	// capture the caller's stack
	err = ucerr.Wrap(err, ucerr.ExtraSkip())
	span := uctrace.GetCurrentSpan(ctx)
	span.RecordError(err)

	if code == http.StatusInternalServerError && ucerr.IsContextCanceledError(err) {
		// This generally happens because the client closed the connection before request processing
		// finished. We don't want to treat that as a server error in our metrics
		code = StatusClientClosedConnectionError
	}

	if code < 500 {
		// Log HTTP status codes below 500 as warnings and 500 and above as errors
		// We sometimes call this method w/ HTTP 204 and we want those as warning and not errors
		if code == http.StatusNotFound || code == http.StatusNoContent {
			// 404 not found, usually bots (ot 204 no content)
			uclog.Infof(ctx, "HTTP %d error: %v", code, err)
		} else {
			uclog.Warningf(ctx, "HTTP %d error: %v", code, err)
		}
	} else {
		uclog.Errorf(ctx, "unhandled error (HTTP %v): %v", code, err)
	}

	s := ucerr.UserFriendlyMessage(err)
	uclog.Infof(ctx, "sanitized error response: %s, request ID: %v", s, request.GetRequestID(ctx))
	span.SetStringAttribute(uctrace.AttributeUserFriendlyError, s)

	// send off 500s for analysis (we don't care about 4XXs)
	if code == http.StatusInternalServerError {
		if errorReporter != nil {
			errorReporter.ReportingFunc()(ctx, err)
		}
	}
	http.Error(w, s, code)
}

// ErrorL replacement for error that will require errEventName to be set to some string which is unique within
// scope of the handler
func ErrorL(ctx context.Context, w http.ResponseWriter, err error, code int, errEventName string) {
	// capture the caller's stack
	// NB: this will work correctly but we'll also capture Error() in this case, which is fine until we finish ErrorL
	err = ucerr.Wrap(err, ucerr.ExtraSkip())

	// TODO temporary check for during migration
	if errEventName == "" {
		uclog.Warningf(ctx, "Error with blank error counter name that will not be counted in %s with %v", uclog.GetHandlerName(ctx), err)
	}
	uclog.SetHandlerErrorName(ctx, errEventName)
	Error(ctx, w, err, code)
}

// RedirectOAuthError is used in situations where the OAuth/OIDC spec requires us to redirect
// to the calling app with the error instead of returning a 4xx or 5xx error code.
func RedirectOAuthError(w http.ResponseWriter, r *http.Request, redirectBaseURL *url.URL, err error, errEventName string) {
	ctx := r.Context()

	var oauthe ucerr.OAuthError
	redirectURL := *redirectBaseURL
	if errors.As(err, &oauthe) {
		redirectURL.RawQuery = url.Values{
			"error":             []string{oauthe.ErrorType},
			"error_description": []string{oauthe.ErrorDesc},
		}.Encode()
	} else {
		redirectURL.RawQuery = url.Values{
			"error":             []string{"unknown_error_type"},
			"error_description": []string{err.Error()},
		}.Encode()
	}

	// We don't consider the query params in the URL to be "too secret" to log because in general
	// there are many reasons why putting sensitive info in URLs is bad for client-side security.
	uclog.Errorf(ctx, "error in request '%s' (redirecting to '%s'): %v", r.URL.RequestURI(), redirectURL.String(), oauthe)

	// TODO temporary check for during migration
	if errEventName == "" {
		uclog.Warningf(ctx, "Error with blank error counter name that will not be counted in %s", uclog.GetHandlerName(ctx))
	}
	uclog.SetHandlerErrorName(ctx, errEventName)

	// The OAuth spec uses HTTP 302 Found everywhere.
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// SQLReadErrorMapper is a nice wrapper for SQL errors that
// handles 404s for ErrNoRows, 500 otherwise
func SQLReadErrorMapper(err error) int {
	if errors.Is(err, sql.ErrNoRows) {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

// SQLDeleteErrorMapper is a nice wrapper for SQL errors that
// handles 404s for ErrNoRows, 500 otherwise
func SQLDeleteErrorMapper(err error) int {
	// For now the same set of errors applies to read & delete calls.
	return SQLReadErrorMapper(err)
}

// SQLWriteErrorMapper is a nice wrapper for SQL errors that
// handles 409 for unique violations, 500 otherwise
func SQLWriteErrorMapper(err error) int {
	if ucdb.IsUniqueViolation(err) {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}
