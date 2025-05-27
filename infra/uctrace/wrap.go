package uctrace

import "context"

// Wrap0 takes a function that returns 0 values + potentially an error, and
// wraps its execution in a span. This is useful for capturing any returned
// errors and recording them in the span, since you'd otherwise need to find
// all the potential error return paths in your function and ensure you're
// recording those. (If you want to trace a function that doesn't return an
// error, just call tracer.Start() directly.)
func Wrap0(ctx context.Context, t Tracer, spanName string, requireParentSpan bool, f func(context.Context) error) error {
	ctx, span := t.StartSpan(ctx, spanName, requireParentSpan)
	defer span.End()

	err := f(ctx)
	span.RecordError(err)
	return err // lint: ucerr-ignore
}

// Wrap1 wraps a function execution in a span. See comments above.
func Wrap1[T1 any](ctx context.Context, t Tracer, spanName string, requireParentSpan bool, f func(context.Context) (T1, error)) (T1, error) {
	ctx, span := t.StartSpan(ctx, spanName, requireParentSpan)
	defer span.End()

	r1, err := f(ctx)
	span.RecordError(err)
	return r1, err // lint: ucerr-ignore
}

// Wrap2 wraps a function execution in a span. See comments above.
func Wrap2[T1 any, T2 any](ctx context.Context, t Tracer, spanName string, requireParentSpan bool, f func(context.Context) (T1, T2, error)) (T1, T2, error) {
	ctx, span := t.StartSpan(ctx, spanName, requireParentSpan)
	defer span.End()

	r1, r2, err := f(ctx)
	span.RecordError(err)
	return r1, r2, err // lint: ucerr-ignore
}

// Wrap3 wraps a function execution in a span. See comments above.
func Wrap3[T1 any, T2 any, T3 any](ctx context.Context, t Tracer, spanName string, requireParentSpan bool, f func(context.Context) (T1, T2, T3, error)) (T1, T2, T3, error) {
	ctx, span := t.StartSpan(ctx, spanName, requireParentSpan)
	defer span.End()

	r1, r2, r3, err := f(ctx)
	span.RecordError(err)
	return r1, r2, r3, err // lint: ucerr-ignore
}

// Wrap4 wraps a function execution in a span. See comments above.
func Wrap4[T1 any, T2 any, T3 any, T4 any](ctx context.Context, t Tracer, spanName string, requireParentSpan bool, f func(context.Context) (T1, T2, T3, T4, error)) (T1, T2, T3, T4, error) {
	ctx, span := t.StartSpan(ctx, spanName, requireParentSpan)
	defer span.End()

	r1, r2, r3, r4, err := f(ctx)
	span.RecordError(err)
	return r1, r2, r3, r4, err // lint: ucerr-ignore
}
