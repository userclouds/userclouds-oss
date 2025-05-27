package uctrace

import (
	"context"
	"net/http"
)

// Tracer manages the creation of spans.
type Tracer struct {
}

// NewTracer creates a new Tracer with the given name.
func NewTracer(name string) Tracer {
	return Tracer{}
}

// StartSpan creates a new span with the given name. If requireParentSpan is
// true, it creates a no-op (non-recording) span if we are not already tracing.
func (t Tracer) StartSpan(ctx context.Context, name string, requireParentSpan bool) (context.Context, Span) {
	return ctx, Span{}
}

// Span is the individual component of a trace.
type Span struct {
}

// End completes the Span.
func (s Span) End() {
}

// IsRecording returns the recording state of the Span. It will return true if
// the Span is active and events can be recorded.
func (s Span) IsRecording() bool {
	return false
}

// RecordError sets the span status to "error" and records an error event
// capturing the error message.
func (s Span) RecordError(err error) {
}

// SetStringAttribute sets the given string attribute on the span.
func (s Span) SetStringAttribute(key Attribute, value string) {
}

// SetStringSliceAttribute sets the given string slice attribute on the span.
func (s Span) SetStringSliceAttribute(key Attribute, value []string) {
}

// SetIntAttribute sets the given int attribute on the span.
func (s Span) SetIntAttribute(key Attribute, value int) {
}

// SetIntSliceAttribute sets the given int slice attribute on the span.
func (s Span) SetIntSliceAttribute(key Attribute, value []int) {
}

// SetInt64Attribute sets the given int64 attribute on the span.
func (s Span) SetInt64Attribute(key Attribute, value int64) {
}

// SetInt64SliceAttribute sets the given int64 slice attribute on the span.
func (s Span) SetInt64SliceAttribute(key Attribute, value []int64) {
}

// SetFloat64Attribute sets the given float64 attribute on the span.
func (s Span) SetFloat64Attribute(key Attribute, value float64) {
}

// SetFloat64SliceAttribute sets the given float64 slice attribute on the span.
func (s Span) SetFloat64SliceAttribute(key Attribute, value []float64) {
}

// SetBoolAttribute sets the given bool attribute on the span.
func (s Span) SetBoolAttribute(key Attribute, value bool) {
}

// SetBoolSliceAttribute sets the given bool slice attribute on the span.
func (s Span) SetBoolSliceAttribute(key Attribute, value []bool) {
}

// GetCurrentSpan returns the current Span from the given context.
func GetCurrentSpan(ctx context.Context) Span {
	return Span{}
}

// MakeHTTPClient creates an HTTP client that will record spans for outgoing
// HTTP requests.
func MakeHTTPClient() *http.Client {
	return http.DefaultClient
}
