package request

import (
	"net/http"
)

// RequestIDHeader is the name of the header we use to return request IDs for debugging
const RequestIDHeader = "X-Request-Id"

// GetRequestIDFromHeader returns the request ID a response headers if present, otherwise, returns an empty string
func GetRequestIDFromHeader(header http.Header) string {
	return header.Get(RequestIDHeader)
}
