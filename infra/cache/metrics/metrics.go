package metrics

import (
	"context"
	"sync"
	"time"

	"userclouds.com/infra/ucerr"
)

type contextKey int

const (
	ctxCacheMetrics contextKey = 1
)

// CacheMetrics keeps track of cache calls during a request
type CacheMetrics struct {
	cacheMetricsLock  sync.RWMutex
	calls             int
	hits              int
	misses            int
	deletions         int
	getLatencies      time.Duration
	storeLatencies    time.Duration
	deletionLatencies time.Duration
}

func newCacheMetrics() *CacheMetrics {
	return &CacheMetrics{cacheMetricsLock: sync.RWMutex{}}
}

// GetTotalDuration returns the sum of the time spend calling the cache
func (m *CacheMetrics) GetTotalDuration() time.Duration {
	m.cacheMetricsLock.RLock()
	defer m.cacheMetricsLock.RUnlock()
	return m.getLatencies + m.storeLatencies + m.deletionLatencies
}

// HadCalls returns true if there were any cache calls
func (m *CacheMetrics) HadCalls() bool {
	m.cacheMetricsLock.RLock()
	defer m.cacheMetricsLock.RUnlock()
	return m.calls > 0
}

// GetCounters returns the number of cache calls, hits and misses
func (m *CacheMetrics) GetCounters() (int, int, int) {
	m.cacheMetricsLock.RLock()
	defer m.cacheMetricsLock.RUnlock()
	return m.calls, m.hits, m.misses
}

// GetMetrics returns the cache metrics structure from the context, errors out if it is not there.
func GetMetrics(ctx context.Context) (*CacheMetrics, error) {
	// We don't take a lock as we either perform read only once per request at the end, effectively single threaded, or we take a write lock before
	// modifying the data
	val := ctx.Value(ctxCacheMetrics)
	metrics, ok := val.(*CacheMetrics)
	if !ok {
		return nil, ucerr.Errorf("Can't find cache metric data in context")
	}
	return metrics, nil
}

// ResetContext resets/adds a cache metrics struct to the context to allow keeping track of cache calls during a request
func ResetContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxCacheMetrics, newCacheMetrics())
}

// InitContext adds a cache metrics struct to the context to allow keeping track of cache calls during a request
func InitContext(ctx context.Context) context.Context {
	// We don't take a lock as we always perform only once per request at start, effectively single threaded. If that pattern changes, we need to take a lock
	val := ctx.Value(ctxCacheMetrics)
	if _, ok := val.(*CacheMetrics); !ok {
		return context.WithValue(ctx, ctxCacheMetrics, newCacheMetrics())
	}
	return ctx
}

// RecordCacheHit records a cache hit
func RecordCacheHit(ctx context.Context, duration time.Duration) {
	metricsData, err := GetMetrics(ctx)
	if err != nil {
		return
	}
	// We take a lock as we may have parallelism in the request
	metricsData.cacheMetricsLock.Lock()
	defer metricsData.cacheMetricsLock.Unlock()

	metricsData.calls++
	metricsData.hits++
	metricsData.getLatencies += duration
}

// RecordCacheMiss records a cache miss
func RecordCacheMiss(ctx context.Context, duration time.Duration) {
	metricsData, err := GetMetrics(ctx)
	if err != nil {
		return
	}

	// We take a lock as we may have parallelism in the request
	metricsData.cacheMetricsLock.Lock()
	defer metricsData.cacheMetricsLock.Unlock()

	metricsData.calls++
	metricsData.misses++
	metricsData.getLatencies += duration
}

// RecordMultiGet records getting multiple objects from the cache
func RecordMultiGet(ctx context.Context, hits, misses int, duration time.Duration) {
	metricsData, err := GetMetrics(ctx)
	if err != nil {
		return
	}

	// We take a lock as we may have parallelism in the request
	metricsData.cacheMetricsLock.Lock()
	defer metricsData.cacheMetricsLock.Unlock()

	metricsData.calls++
	metricsData.hits += hits
	metricsData.misses += misses
	metricsData.getLatencies += duration
}

// RecordCacheStore records store data in the cache
func RecordCacheStore(ctx context.Context, start time.Time) {
	metricsData, err := GetMetrics(ctx)
	if err != nil {
		return
	}

	// We take a lock as we may have parallelism in the request
	metricsData.cacheMetricsLock.Lock()
	defer metricsData.cacheMetricsLock.Unlock()

	metricsData.calls++
	metricsData.storeLatencies += time.Now().UTC().Sub(start)
}

// RecordCacheDelete records a deletion from the cache
func RecordCacheDelete(ctx context.Context, start time.Time) {
	metricsData, err := GetMetrics(ctx)
	if err != nil {
		return
	}

	// We take a lock as we may have parallelism in the request
	metricsData.cacheMetricsLock.Lock()
	defer metricsData.cacheMetricsLock.Unlock()

	metricsData.calls++
	metricsData.deletions++
	metricsData.deletionLatencies += time.Now().UTC().Sub(start)
}
