package responsewriter

import (
	"net/http"
)

// StatusResponseWriter Type required to capture status code of nested handlers.
type StatusResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

// NewStatusResponseWriter wraps the response writer
func NewStatusResponseWriter(w http.ResponseWriter) *StatusResponseWriter {
	srw := &StatusResponseWriter{
		ResponseWriter: w,
		StatusCode:     http.StatusOK,
	}
	return srw
}

// WriteHeader writes response code
func (srw *StatusResponseWriter) WriteHeader(code int) {
	srw.StatusCode = code
	srw.ResponseWriter.WriteHeader(code)
}
