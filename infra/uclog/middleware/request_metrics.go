package middleware

import (
	"fmt"
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucmetrics"
)

const httpSubsystem = ucmetrics.Subsystem("http")

// Note: scraped metrics are already segmented by UC service (prom
// `job`) and node (prom `instance`), so adding `handler` here might be
// pushing the cardinality too high. If the metrics queries take too
// long to run, we can probably remove this `handler` label.
var (
	metricHTTPRequestCount                  = ucmetrics.CreateCounter(httpSubsystem, "requests_total", "The total number of processed HTTP requests", "method", "handler", "status", "tenant_id")
	metricHTTPRequestDurationSeconds        = ucmetrics.CreateHistogram(httpSubsystem, "request_duration_seconds", "time taken to handle requests", "handler", "tenant_id")
	metricHTTPResponseSizeCompressedBytes   = ucmetrics.CreateHistogram(httpSubsystem, "response_size_bytes_compressed", "Size of compressed response in bytes.", "method", "handler")
	metricHTTPResponseSizeUncompressedBytes = ucmetrics.CreateHistogram(httpSubsystem, "response_size_bytes_uncompressed", "Size of uncompressed response in bytes.", "method", "handler")
	metricDBCalls                           = ucmetrics.CreateHistogram(httpSubsystem, "db_calls_per_request", "The total number of processed DB calls", "handler", "method")
	metricDBDurationSeconds                 = ucmetrics.CreateHistogram(httpSubsystem, "db_total_duration_seconds_per_request", "time taken to handle DB queries", "handler")
	metricCacheHits                         = ucmetrics.CreateHistogram(httpSubsystem, "cache_hits_per_request", "Tracks distribution of number of cache hits per request", "handler", "method")
	metricCacheMisses                       = ucmetrics.CreateHistogram(httpSubsystem, "cache_misses_per_request", "Tracks distribution of number of cache misses per request", "handler", "method")
	metricCacheCalls                        = ucmetrics.CreateHistogram(httpSubsystem, "cache_calls_per_request", "The total number of cache calls", "handler", "method")
	metricCacheDurationSeconds              = ucmetrics.CreateHistogram(httpSubsystem, "cache_total_duration_seconds_per_request", "time taken to handle cache requests", "handler")
)

// record prometheus metrics
func recordMetrics(r *http.Request, handlerName string, statusCode int, ll *logLine, tenantID uuid.UUID) {
	if handlerName == "" { // Make sure we don't have an empty handler name
		handlerName = "unknown"
	}
	metricHTTPRequestCount.WithLabelValues(r.Method, handlerName, fmt.Sprintf("%d", statusCode), tenantID.String()).Inc()
	metricHTTPRequestDurationSeconds.WithLabelValues(handlerName, tenantID.String()).Observe(ll.Duration.Seconds())
	metricHTTPResponseSizeCompressedBytes.WithLabelValues(r.Method, handlerName).Observe(float64(ll.ResponseSizePost))
	metricHTTPResponseSizeUncompressedBytes.WithLabelValues(r.Method, handlerName).Observe(float64(ll.ResponseSizePre))
	metricDBDurationSeconds.WithLabelValues(handlerName).Observe(ll.DBDuration.Seconds())
	metricDBCalls.WithLabelValues(handlerName, r.Method).Observe(float64(ll.DBCalls))
	metricCacheHits.WithLabelValues(handlerName, r.Method).Observe(float64(ll.CacheHits))
	metricCacheMisses.WithLabelValues(handlerName, r.Method).Observe(float64(ll.CacheMisses))
	metricCacheCalls.WithLabelValues(handlerName, r.Method).Observe(float64(ll.CacheCalls))
	metricCacheDurationSeconds.WithLabelValues(handlerName).Observe(ll.CacheDuration.Seconds())
}
