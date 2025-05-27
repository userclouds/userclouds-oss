package httpdump

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
)

// LoggingResponseWriter Type required to capture status code of nested handlers.
type LoggingResponseWriter struct {
	http.ResponseWriter
	Recorder   *httptest.ResponseRecorder
	StatusCode int
}

// NewLoggingResponseWriter wraps the response writer and captures the HTTP response
func NewLoggingResponseWriter(w http.ResponseWriter) *LoggingResponseWriter {
	lrw := &LoggingResponseWriter{
		ResponseWriter: w,
		Recorder:       httptest.NewRecorder(),
		StatusCode:     http.StatusOK,
	}
	return lrw
}

// Header implements http.ResponseWriter
func (lrw *LoggingResponseWriter) Header() http.Header {
	return lrw.Recorder.Header()
}

// WriteHeader writes response code
func (lrw *LoggingResponseWriter) WriteHeader(code int) {
	lrw.StatusCode = code
	lrw.Recorder.WriteHeader(code)
	lrw.ResponseWriter.WriteHeader(code)
}

// Write implements http.ResponseWriter
func (lrw *LoggingResponseWriter) Write(bs []byte) (int, error) {
	lrw.Recorder.Write(bs)
	return lrw.ResponseWriter.Write(bs)
}

// DumpResponse extracts the raw HTTP response as a string
func (lrw *LoggingResponseWriter) DumpResponse() string {
	responseBytes, err := httputil.DumpResponse(lrw.Recorder.Result(), true)
	if err != nil {
		return "++++ FAILED TO HTTP DUMP REQUEST\n"
	}
	return fmt.Sprintf("\n++++ BEGIN DUMP HTTP RESPONSE:\n%s\n++++ END DUMP HTTP RESPONSE", string(responseBytes))
}
