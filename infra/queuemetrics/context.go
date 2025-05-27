package queuemetrics

import (
	"context"
	"time"
)

type contextKey int

const (
	ctxQueueCounts contextKey = 1 // key for the queue stats
)

// InitContext initializes the context with a wait time of 0
func InitContext(ctx context.Context) context.Context {
	waitTime := time.Duration(0)
	return context.WithValue(ctx, ctxQueueCounts, &waitTime)

}

// SetQueueStats sets wait duration in queue for a request
func SetQueueStats(ctx context.Context, waitTime time.Duration) context.Context {
	val := ctx.Value(ctxQueueCounts)
	waitTimeCtx, ok := val.(*time.Duration)
	if !ok {
		return context.WithValue(ctx, ctxQueueCounts, &waitTime)
	}
	*waitTimeCtx = waitTime
	return ctx
}

// GetQueueStats returns the inflight and request count from the context if set
func GetQueueStats(ctx context.Context) time.Duration {
	val := ctx.Value(ctxQueueCounts)
	waitTime, ok := val.(*time.Duration)
	if !ok {
		return 0
	}
	return *waitTime
}
