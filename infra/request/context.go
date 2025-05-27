package request

import (
	"context"
	"net/http"

	"github.com/go-http-utils/headers"
	"github.com/gofrs/uuid"
)

type contextKey int

const (
	ctxRequestData contextKey = 1
)

// GetRequestID returns a per request id if one was set
func GetRequestID(ctx context.Context) uuid.UUID {
	return getRequestData(ctx).requestID
}

type requestData struct {
	// we use this context even for non-HTTP requests (for example, worker tasks), so we need to know if it's an HTTP request or not
	isHTTPRequest bool
	requestID     uuid.UUID
	hostname      string
	authHeader    string
	userAgent     string
	method        string
	path          string
	remoteIP      string
	forwardedFor  string
	sdkVersion    string
}

func getRequestData(ctx context.Context) requestData {
	val := ctx.Value(ctxRequestData)
	rd, ok := val.(*requestData)
	if !ok {
		return requestData{requestID: uuid.Nil}
	}
	return *rd

}

// NewRequestID sets a per request id if one was not set
func NewRequestID(ctx context.Context) context.Context {
	return SetRequestIDIfNotSet(ctx, uuid.Must(uuid.NewV4()))
}

// SetRequestIDIfNotSet sets a per request id if one was not set - for use for non-HTTP request flow (go routines, worker tasks, etc...)
func SetRequestIDIfNotSet(ctx context.Context, requestID uuid.UUID) context.Context {
	rd := getRequestData(ctx)
	if rd.requestID.IsNil() {
		rd.requestID = requestID
		rd.isHTTPRequest = false
		return context.WithValue(ctx, ctxRequestData, &rd)
	}
	return ctx
}

// SetRequestID sets or resets - for use for non-HTTP request flow (go routines, worker tasks, etc...)
func SetRequestID(ctx context.Context, requestID uuid.UUID) context.Context {
	rd := getRequestData(ctx)
	if rd.requestID.IsNil() {
		rd.requestID = requestID
		rd.isHTTPRequest = false
		return context.WithValue(ctx, ctxRequestData, &rd)
	}
	rd.requestID = requestID
	return ctx
}

// SetRequestData sets capture a bunch of data from the request and saves into a struct in the context
func SetRequestData(ctx context.Context, req *http.Request, requestID uuid.UUID) context.Context {
	if req == nil {
		return ctx
	}
	rd := &requestData{
		isHTTPRequest: true,
		requestID:     requestID,
		hostname:      req.Host,
		userAgent:     req.UserAgent(),
		authHeader:    req.Header.Get(headers.Authorization),
		method:        req.Method,
		path:          req.URL.Path,
		remoteIP:      req.RemoteAddr,
		forwardedFor:  req.Header.Get(headers.XForwardedFor),
		sdkVersion:    req.Header.Get(HeaderSDKVersion),
	}
	return context.WithValue(ctx, ctxRequestData, rd)
}

// GetHostname returns the hostname used for this particular request
func GetHostname(ctx context.Context) string {
	return getRequestData(ctx).hostname
}

// GetAuthHeader returns the Authorization Header for this particular request
func GetAuthHeader(ctx context.Context) string {
	return getRequestData(ctx).authHeader
}

// GetUserAgent returns the User-Agent header for this particular request
func GetUserAgent(ctx context.Context) string {
	return getRequestData(ctx).userAgent
}

// GetSDKVersion returns the UserClouds SDK version (from header) for this particular request
func GetSDKVersion(ctx context.Context) string {
	return getRequestData(ctx).sdkVersion
}

// GetForwardedFor returns the X-Forwarded-For header for this particular request
func GetForwardedFor(ctx context.Context) string {
	return getRequestData(ctx).forwardedFor
}

// GetRemoteIP returns the remote IP for this particular request
func GetRemoteIP(ctx context.Context) string {
	return getRequestData(ctx).remoteIP
}

// GetRequestDataMap returns the a map of request data for a particular request, this is useful when we want to pass unstructured data to to other systems (sentry, tracing, etc...) and we don't have a reference to the request object
func GetRequestDataMap(ctx context.Context) map[string]string {
	rd := getRequestData(ctx)
	if !rd.isHTTPRequest {
		return nil
	}
	return map[string]string{
		"method":    rd.method,
		"path":      rd.path,
		"hostname":  rd.hostname,
		"requestID": rd.requestID.String(),
	}
}
