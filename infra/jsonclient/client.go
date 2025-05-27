package jsonclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-http-utils/headers"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/request"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uctrace"
)

const maxRetries = 3
const backoff = time.Millisecond * 500

var tracer = uctrace.NewTracer("jsonclient")

// Error defines a jsonclient error for non-2XX/3XX status codes
// TODO: decide how to handle 3XX results
type Error struct {
	StatusCode int         `json:"http_status_code" yaml:"http_status_code"`
	Body       string      `json:"response_body" yaml:"response_body"`
	Headers    http.Header `json:"response_headers,omitempty" yaml:"response_headers,omitempty"`
}

type structuredError struct {
	Code  int `json:"http_status_code" yaml:"http_status_code"`
	Error any `json:"error" yaml:"error"`
}

// SDKStructuredError is the standard structured error returned by our APIs for SDK clients to handle
type SDKStructuredError struct {
	Error       string    `json:"error" yaml:"error"`
	ID          uuid.UUID `json:"id" yaml:"id"`
	SecondaryID uuid.UUID `json:"secondary_id" yaml:"secondary_id"`
	Identical   bool      `json:"identical" yaml:"identical"`
}

type sdkStructuredErrorBody struct {
	Error SDKStructuredError `json:"error" yaml:"error"`
}

// Error implements UCError
func (e Error) Error() string {
	parts := make([]string, 0, 3)
	if e.Body != "" {
		parts = append(parts, fmt.Sprintf("Body: %v", e.Body))
	}
	if len(e.Headers) > 0 {
		parts = append(parts, fmt.Sprintf("Headers: %v", e.Headers))
	}
	if len(parts) > 0 {
		return fmt.Sprintf("HTTP Status: %d\n%s", e.StatusCode, strings.Join(parts, "\n"))
	}
	return fmt.Sprintf("HTTP Status %d", e.StatusCode)
}

// Friendly implements UCError
func (e Error) Friendly() string {
	if e.Body != "" {
		return e.Body
	}
	return fmt.Sprintf("HTTP Status %d", e.StatusCode)
}

// FriendlyStructure implements UCError
// If the body is JSON, we'll convert it to map[string]any so it gets
// preserved as something easily unmarshaled in the returned error.
func (e Error) FriendlyStructure() any {
	var objmap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(e.Body), &objmap); err == nil {
		return (structuredError{e.StatusCode, objmap})
	}
	return nil
}

// Code return the HTTP status code and implements private interface jsonapi.errorWithCode
func (e Error) Code() int {
	return e.StatusCode
}

// Client defines a JSON-focused HTTP client
// TODO: someday this should use a custom http.Client etc
type Client struct {
	baseURL string

	// Keep options contained so they can be cloned & augmented per request.
	options      options
	optionsMutex sync.RWMutex
}

// New returns a new Client
func New(url string, opts ...Option) *Client {
	c := &Client{
		baseURL: url,
		options: options{
			headers: make(http.Header),
		},
	}

	for _, opt := range opts {
		opt.apply(&c.options)
	}

	return c
}

// Apply applies options to an existing client (useful for updating a header/cookie/etc on an existing client).
func (c *Client) Apply(opts ...Option) {
	c.optionsMutex.Lock()
	defer c.optionsMutex.Unlock()
	for _, opt := range opts {
		opt.apply(&c.options)
	}
}

// GetBearerToken returns the bearer token associated with this client, if one exists,
// or an error otherwise.
func (c *Client) GetBearerToken() (string, error) {
	c.optionsMutex.RLock()
	defer c.optionsMutex.RUnlock()
	token, err := ucjwt.ExtractBearerToken(&c.options.headers)
	return token, ucerr.Wrap(err)
}

func (c *Client) tokenNeedsRefresh() bool {
	needsRefresh := true
	currentToken, err := c.GetBearerToken()
	if err == nil {
		needsRefresh, err = ucjwt.IsExpired(currentToken)
		if err != nil {
			needsRefresh = true
		}
	}
	return needsRefresh
}

func (c *Client) hasTokenSource() bool {
	c.optionsMutex.RLock()
	defer c.optionsMutex.RUnlock()
	return c.options.tokenSource != nil
}

func (c *Client) hasAccessToken() bool {
	c.optionsMutex.RLock()
	defer c.optionsMutex.RUnlock()
	authHeaders := c.options.headers["Authorization"]
	if len(authHeaders) == 0 {
		return false
	}

	for _, header := range authHeaders {
		val := strings.ToLower(header)
		if strings.HasPrefix(val, "accesstoken ") {
			return true
		}
	}
	return false
}

// ValidateBearerTokenHeader ensures that there is a non-expired bearer token specified directly OR
// that there's a valid token source to refresh it if not specified or expired.
func (c *Client) ValidateBearerTokenHeader() error {
	if c.tokenNeedsRefresh() && !c.hasTokenSource() && !c.hasAccessToken() {
		return ucerr.New("cannot refresh unspecified or expired bearer token without specifying valid ClientCredentialsTokenSource option for jsonclient")
	}
	return nil
}

// refreshBearerToken checks if the current token is invalid or expired, and refreshes it via
// the Client Credentials Flow if needed.
func (c *Client) refreshBearerToken(ctx context.Context) error {
	return ucerr.Wrap(c.refreshBearerTokenRetry(ctx, 1))
}

func (c *Client) refreshBearerTokenRetry(ctx context.Context, retries int) error {
	if c.tokenNeedsRefresh() {
		if !c.hasTokenSource() {
			return ucerr.New("cannot refresh bearer token without specifying valid ClientCredentialsTokenSource option for jsonclient")
		}

		c.optionsMutex.RLock()
		accessToken, err := c.options.tokenSource.GetToken()
		c.optionsMutex.RUnlock()
		if err != nil {
			// TODO: can we simplify this design somehow? would be nice for GetToken to reuse client or at least opts?
			if c.options.retryNetworkErrors && isNetworkError(err) {
				if retries >= maxRetries {
					return ucerr.Errorf("max retries exceeded: %v", err)
				}

				c.logWarning(ctx, http.MethodPost, "refreshBearerToken",
					ucerr.Errorf("network error, retry %d: %v", retries, err).Error(), 0)

				time.Sleep(backoff)
				return ucerr.Wrap(c.refreshBearerTokenRetry(ctx, retries+1))
			}
			return ucerr.Wrap(err)
		}

		c.Apply(HeaderAuthBearer(accessToken))
	}
	return nil
}

// Get makes an HTTP get using this client
// TODO: need to support query params soon :)
func (c *Client) Get(ctx context.Context, path string, response any, opts ...Option) error {
	return ucerr.Wrap(c.makeRequest(ctx, http.MethodGet, path, nil, response, opts))
}

// Post makes an HTTP post using this client
// If response is nil, the response isn't decoded and merely the success or failure is returned
func (c *Client) Post(ctx context.Context, path string, body, response any, opts ...Option) error {
	bs, err := json.Marshal(body)
	if err != nil {
		return ucerr.Wrap(err)
	}

	return ucerr.Wrap(c.makeRequest(ctx, http.MethodPost, path, bs, response, opts))
}

// CreateIfNotExists does a POST and handles the 409 conflict error for the caller
func (c *Client) CreateIfNotExists(ctx context.Context, path string, body, response any, opts ...Option) (bool, uuid.UUID, error) {
	bs, err := json.Marshal(body)
	if err != nil {
		return false, uuid.Nil, ucerr.Wrap(err)
	}

	if err := c.makeRequest(ctx, http.MethodPost, path, bs, response, opts); err != nil {
		var jce Error
		if errors.As(err, &jce) && jce.StatusCode == http.StatusConflict {
			var sdkErrorBody sdkStructuredErrorBody
			if err := json.Unmarshal([]byte(jce.Body), &sdkErrorBody); err == nil && sdkErrorBody.Error.Identical {
				return true, sdkErrorBody.Error.ID, nil
			}
		}
		return false, uuid.Nil, ucerr.Wrap(err)
	}

	return false, uuid.Nil, nil
}

// Put makes an HTTP put using this client
// If response is nil, the response isn't decoded and merely the success or failure is returned
func (c *Client) Put(ctx context.Context, path string, body, response any, opts ...Option) error {
	bs, err := json.Marshal(body)
	if err != nil {
		return ucerr.Wrap(err)
	}

	return ucerr.Wrap(c.makeRequest(ctx, http.MethodPut, path, bs, response, opts))
}

// Patch makes an HTTP patch using this client
// If response is nil, the response isn't decoded and merely the success or failure is returned
func (c *Client) Patch(ctx context.Context, path string, body, response any, opts ...Option) error {
	bs, err := json.Marshal(body)
	if err != nil {
		return ucerr.Wrap(err)
	}

	return ucerr.Wrap(c.makeRequest(ctx, http.MethodPatch, path, bs, response, opts))
}

// Delete makes an HTTP delete using this client
func (c *Client) Delete(ctx context.Context, path string, body any, opts ...Option) error {
	bs, err := json.Marshal(body)
	if err != nil {
		return ucerr.Wrap(err)
	}

	return ucerr.Wrap(c.makeRequest(ctx, http.MethodDelete, path, bs, nil, opts))
}

func isNetworkError(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) || errors.Is(err, io.EOF)
}

func (c *Client) makeRequest(ctx context.Context, method, path string, bs []byte, response any, opts []Option) error {
	return uctrace.Wrap0(ctx, tracer, fmt.Sprintf("%s %s%s", method, c.baseURL, path), true, func(ctx context.Context) error {
		return ucerr.Wrap(c.makeRequestRetry(ctx, method, path, bs, response, opts, 1))
	})
}

func (c *Client) makeRequestRetry(ctx context.Context,
	method,
	path string,
	bs []byte,
	response any,
	opts []Option,
	retries int) error {
	requestID := "N/A"
	start := time.Now().UTC()
	defer func() {
		if logger != nil && !c.options.stopLogging {
			logger.Debugf(ctx, "jsonclient request %s %s took %s request id: %s", method, path, time.Since(start), requestID)
		}
	}()

	return uctrace.Wrap0(ctx, tracer, fmt.Sprintf("Client.makeRequestRetry retry %d", retries), true, func(ctx context.Context) error {
		// auto-refresh bearer token if needed
		// do this before cloning (it's thread safe) so we don't "lose" the refresh
		if c.hasTokenSource() {
			if err := c.refreshBearerToken(ctx); err != nil {
				return ucerr.Wrap(err)
			}
		}

		// Always clone to minimize contention
		c.optionsMutex.RLock()
		options := c.options.clone()
		c.optionsMutex.RUnlock()

		// Concat per-request options
		for _, opt := range opts {
			if opt == nil {
				return ucerr.New("nil option provided to jsonclient request")
			}
			opt.apply(options)
		}

		if options.decodeFunc != nil && response != nil {
			return ucerr.New("`CustomDecoder` option should only be specified with a nil `response`")
		}

		client := uctrace.MakeHTTPClient()

		reqURL := c.buildURL(path)
		req, err := http.NewRequestWithContext(ctx, method, reqURL, bytes.NewReader(bs))
		if err != nil {
			return ucerr.Wrap(err)
		}

		req.Header = options.headers.Clone()
		req.Header.Add(headers.ContentType, "application/json")

		// add our per-request context headers
		for _, fn := range options.perRequestHeaders {
			k, v := fn(ctx)
			if k != "" {
				req.Header.Add(k, v)
			}
		}

		for _, cookie := range options.cookies {
			req.AddCookie(&cookie)
		}

		// https://github.com/golang/go/issues/29865
		// Host header is ignored by http.Request.Write, but for test purposes
		// it is very useful to override the Host header.
		if req.Header.Get("Host") != "" {
			req.Host = req.Header.Get("Host")
		} else if !options.bypassRouting {
			// this lets us inject service-dependent intra-datacenter rerouting logic as needed (implemented in internal/apiclient/routing)
			Router.Reroute(ctx, req)
		}

		res, err := client.Do(req)
		if res != nil {
			requestID = request.GetRequestIDFromHeader(res.Header)
		} else {
			requestID = "[HTTP response N/A]"
		}
		if err != nil {
			if options.retryNetworkErrors && isNetworkError(err) {
				if retries >= maxRetries {
					return ucerr.Errorf("max retries exceeded: %v", err)
				}
				c.logWarning(ctx, req.Method, req.URL.String(), ucerr.Errorf("network error, retry %d: %v", retries, err).Error(), 0)
				time.Sleep(backoff)
				return ucerr.Wrap(c.makeRequestRetry(ctx, method, path, bs, response, opts, retries+1))
			}
			return ucerr.Wrap(err)
		}
		defer res.Body.Close()
		body := ""
		// If the response was not an error OR if the caller specified UnmarshalOnError, try to deserialize
		// the response into the provided struct.
		if res.StatusCode < http.StatusBadRequest || options.unmarshalOnError {
			if options.decodeFunc != nil {
				if err := options.decodeFunc(ctx, res.Body); err != nil {
					return ucerr.Wrap(err)
				}
			} else if response != nil {
				if err := json.NewDecoder(res.Body).Decode(response); err != nil {
					return ucerr.Wrap(err)
				}
			}
		} else {
			// An error was returned and the caller is not intentionally capturing the body; log full error response for debugging purposes.
			// TODO: hide behind a flag for perf / PII / etc reasons?
			b, err := io.ReadAll(res.Body)
			if err != nil {
				body = "<unable to decode response>"
			} else {
				body = string(b)
			}

			c.logWarning(ctx, method, reqURL, body, res.StatusCode)

			if options.parseOAuthError {
				var oauthe oAuthError
				// OAuth standard requires us to return a body with error descriptions
				// in many cases, so try to decode response but ignore the error if it fails.
				err = json.NewDecoder(bytes.NewReader(b)).Decode(&oauthe)
				if err == nil {
					// Ensure we use the actual code from the http response.
					oauthe.Code = res.StatusCode
					return ucerr.Wrap(oauthe)
				}
			}
		}

		// TODO: validate that 2xx is received, not 3xx or something else?
		if res.StatusCode >= http.StatusBadRequest {
			return ucerr.Wrap(Error{StatusCode: res.StatusCode, Body: body, Headers: res.Header})
		}
		return nil
	})
}

// normalize trailing and leading slashes
func (c *Client) buildURL(path string) string {
	if path != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(c.baseURL, "/"), strings.TrimPrefix(path, "/"))
	}
	return c.baseURL
}

// GetHTTPStatusCode returns the underlying HTTP status code or -1 if no code could be extracted.
func GetHTTPStatusCode(err error) int {
	var jce Error
	var oauthe oAuthError
	if errors.As(err, &jce) {
		return jce.StatusCode
	} else if errors.As(err, &oauthe) {
		return oauthe.Code
	}
	return -1
}

// GetDetailedErrorInfo returns detailed information about the error if available and nil otherwise
func GetDetailedErrorInfo(err error) *SDKStructuredError {
	var jce Error
	if errors.As(err, &jce) {
		var sdkErrorBody sdkStructuredErrorBody
		if err := json.Unmarshal([]byte(jce.Body), &sdkErrorBody); err == nil {
			return &sdkErrorBody.Error
		}
	}
	return nil
}

// IsHTTPNotFound returns true if the error is an HTTP 404 Not Found error
func IsHTTPNotFound(err error) bool {
	return GetHTTPStatusCode(err) == http.StatusNotFound
}

// IsHTTPStatusConflict returns true if the error is an HTTP 409 Conflict error
func IsHTTPStatusConflict(err error) bool {
	return GetHTTPStatusCode(err) == http.StatusConflict
}
