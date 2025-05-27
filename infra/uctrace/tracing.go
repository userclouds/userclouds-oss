package uctrace

import (
	"context"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Tracer manages the creation of spans. You should create a Tracer for each
// logical component producing traces (e.g. one per golang package). Spans get
// annotated with the Tracer name that created them.
type Tracer struct {
	tracer trace.Tracer
}

// NewTracer creates a new Tracer with the given name.
func NewTracer(name string) Tracer {
	return Tracer{tracer: otel.Tracer(name)}
}

var noopTracer = noop.NewTracerProvider().Tracer("no-op")

// StartSpan creates a new span with the given name. If requireParentSpan is true,
// it creates a no-op (non-recording) span if we are not already tracing.
func (t Tracer) StartSpan(ctx context.Context, name string, requireParentSpan bool) (context.Context, Span) {
	var span, parentSpan trace.Span
	if requireParentSpan {
		parentSpan = trace.SpanFromContext(ctx)
		if parentSpan.IsRecording() {
			ctx, span = t.tracer.Start(ctx, name)
		} else {
			_, span = noopTracer.Start(ctx, name)
		}
	} else {
		ctx, span = t.tracer.Start(ctx, name)
	}
	return ctx, Span{span: span}
}

// Span is the individual component of a trace. It represents a single named
// and timed operation of a workflow that is traced.
type Span struct {
	span trace.Span
}

// End completes the Span. The Span is considered complete and ready to be
// delivered through the rest of the telemetry pipeline after this method
// is called. Therefore, updates to the Span are not allowed after this
// method has been called.
func (s Span) End() {
	s.span.End()
}

// IsRecording returns the recording state of the Span. It will return true if
// the Span is active and events can be recorded.
func (s Span) IsRecording() bool {
	return s.span.IsRecording()
}

// RecordError sets the span status to "error" and records an error event
// capturing the error message.
func (s Span) RecordError(err error) {
	if err != nil {
		s.span.SetStatus(codes.Error, err.Error())
		s.span.RecordError(err)
	}
}

// SetAttributes sets one or more KeyValue attributes on the span.
func (s Span) SetAttributes(kv ...attribute.KeyValue) {
	s.span.SetAttributes(kv...)
}

// SetStringAttribute sets the given string attribute on the span.
func (s Span) SetStringAttribute(key Attribute, value string) {
	s.span.SetAttributes(attribute.String(string(key), value))
}

// SetStringSliceAttribute sets the given string slice attribute on the span.
func (s Span) SetStringSliceAttribute(key Attribute, value []string) {
	s.span.SetAttributes(attribute.StringSlice(string(key), value))
}

// SetIntAttribute sets the given int attribute on the span.
func (s Span) SetIntAttribute(key Attribute, value int) {
	s.span.SetAttributes(attribute.Int(string(key), value))
}

// SetIntSliceAttribute sets the given int slice attribute on the span.
func (s Span) SetIntSliceAttribute(key Attribute, value []int) {
	s.span.SetAttributes(attribute.IntSlice(string(key), value))
}

// SetInt64Attribute sets the given int64 attribute on the span.
func (s Span) SetInt64Attribute(key Attribute, value int64) {
	s.span.SetAttributes(attribute.Int64(string(key), value))
}

// SetInt64SliceAttribute sets the given int64 slice attribute on the span.
func (s Span) SetInt64SliceAttribute(key Attribute, value []int64) {
	s.span.SetAttributes(attribute.Int64Slice(string(key), value))
}

// SetFloat64Attribute sets the given float64 attribute on the span.
func (s Span) SetFloat64Attribute(key Attribute, value float64) {
	s.span.SetAttributes(attribute.Float64(string(key), value))
}

// SetFloat64SliceAttribute sets the given float64 slice attribute on the span.
func (s Span) SetFloat64SliceAttribute(key Attribute, value []float64) {
	s.span.SetAttributes(attribute.Float64Slice(string(key), value))
}

// SetBoolAttribute sets the given bool attribute on the span.
func (s Span) SetBoolAttribute(key Attribute, value bool) {
	s.span.SetAttributes(attribute.Bool(string(key), value))
}

// SetBoolSliceAttribute sets the given bool slice attribute on the span.
func (s Span) SetBoolSliceAttribute(key Attribute, value []bool) {
	s.span.SetAttributes(attribute.BoolSlice(string(key), value))
}

// GetCurrentSpan returns the current Span from the given context.
func GetCurrentSpan(ctx context.Context) Span {
	return Span{span: trace.SpanFromContext(ctx)}
}

// MakeHTTPClient creates an HTTP client that will record spans for outgoing
// HTTP requests.
func MakeHTTPClient() *http.Client {
	return &http.Client{Transport: otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithFilter(func(r *http.Request) bool {
			// Only record spans for outgoing HTTP requests if we are already
			// tracing for an incoming request. In theory, it is helpful to
			// record all the time, but in practice, it's not very helpful to
			// have a bunch of spans for outgoing requests that are not related
			// to any incoming request.
			return GetCurrentSpan(r.Context()).IsRecording()
		}),
	)}
}
