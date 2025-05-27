package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"text/template"
	"time"

	"github.com/gofrs/uuid"

	cacheMetrics "userclouds.com/infra/cache/metrics"
	"userclouds.com/infra/middleware"
	"userclouds.com/infra/queuemetrics"
	"userclouds.com/infra/request"
	dbMetrics "userclouds.com/infra/ucdb/metrics"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uclog/responsewriter"
)

// Type required for templating HTTP Logger output.
type logLine struct {
	Status           string
	Duration         time.Duration
	DBDuration       time.Duration
	DBCalls          int
	DBInFlight       int
	CacheHits        int
	CacheMisses      int
	CacheCalls       int
	CacheDuration    time.Duration
	Hostname         string
	Tenant           uuid.UUID
	Method           string
	RequestURI       string
	ResponseSizePre  int
	ResponseSizePost int
	QueueWaitTime    time.Duration
}

// logHTTPRequest logs an event that should affect a counter with optional format-string parsing
func logHTTPRequest(ctx context.Context, level uclog.LogLevel, f string, args ...any) {
	uclog.Log(ctx, uclog.LogEvent{
		LogLevel: level,
		Name:     "Event.HTTPRequest",
		Message:  fmt.Sprintf(f, args...),
		Count:    1,
	})
}

// logHTTPResponse logs an event that should affect a counter with optional format-string parsing
func logHTTPResponse(ctx context.Context, baseLogLevel uclog.LogLevel, httpCode int, tenantID uuid.UUID, f string, args ...any) {
	var level = baseLogLevel
	if httpCode >= 500 {
		level = uclog.LogLevelError
	} else if httpCode >= 400 && !tenantID.IsNil() {
		level = uclog.LogLevelWarning
	}
	eventName := fmt.Sprintf("Event.HTTPResponse_%d", httpCode)
	uclog.Log(ctx, uclog.LogEvent{
		LogLevel: level,
		TenantID: tenantID,
		Name:     eventName,
		Code:     uclog.EventCode(httpCode),
		Message:  fmt.Sprintf(f, args...),
		Count:    1,
	})
}

// logPostHandlerExecution logs an events post execution of a particular handler
func logPostHandlerExecution(ctx context.Context, handlerName string, errorName string, tenantID uuid.UUID, duration time.Duration, metrics *dbMetrics.DBMetrics) {
	// Log the duration of the handler execution
	logEvent(ctx, handlerName, "Duration", tenantID, int(duration.Milliseconds()))
	if errorName != "" {
		logEvent(ctx, handlerName, errorName, tenantID, 1)
	}
	logDBMetrics(ctx, handlerName, tenantID, metrics)
}

func logDBMetrics(ctx context.Context, handlerName string, tenantID uuid.UUID, metrics *dbMetrics.DBMetrics) {
	if !metrics.HadCalls() {
		return
	}
	mc := metrics.GetCopy()
	if mc.SelectCount > 0 {
		logEvent(ctx, handlerName, "DBSelectCount", tenantID, mc.SelectCount)
		logEvent(ctx, handlerName, "DBSelectDuration", tenantID, int(mc.SelectLatencies.Milliseconds()))

	}
	if mc.GetCount > 0 {
		logEvent(ctx, handlerName, "DBGetCount", tenantID, mc.GetCount)
		logEvent(ctx, handlerName, "DBGetDuration", tenantID, int(mc.GetLatencies.Milliseconds()))

	}
	if mc.WriteCount > 0 {
		logEvent(ctx, handlerName, "DBWriteCount", tenantID, mc.WriteCount)
		logEvent(ctx, handlerName, "DBWriteDuration", tenantID, int(mc.WriteLatencies.Milliseconds()))
	}
}

func logEvent(ctx context.Context, handlerName, eventName string, tenantID uuid.UUID, count int) {
	if count <= 0 {
		return
	}

	uclog.Log(ctx, uclog.LogEvent{
		LogLevel: uclog.LogLevelNonMessage,
		TenantID: tenantID,
		Name:     fmt.Sprintf("%s.%s", handlerName, eventName),
		Count:    count,
	})
}

var temp = template.Must(template.New("mw_logger").Parse("HTTP {{.Status}} | {{.Duration}} ({{.ResponseSizePre}}B -> {{.ResponseSizePost}}B) | DB {{.DBCalls}} {{.DBDuration}} {{.DBInFlight}} | Cache: c:{{.CacheCalls}} h:{{.CacheHits}} m:{{.CacheMisses}} {{.CacheDuration}} | {{.Hostname}} | {{.Tenant}} | W: {{.QueueWaitTime}}  | {{.Method}} {{.RequestURI}}"))

// HTTPLoggerMiddleware returns a Middleware for logging HTTP requests
func HTTPLoggerMiddleware() middleware.Middleware {
	return middleware.Func(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now().UTC()
			ctx := uclog.InitHandlerData(cacheMetrics.InitContext(dbMetrics.InitContext(queuemetrics.InitContext(r.Context()))))
			sdkVersion := request.GetSDKVersion(ctx)
			// log the request start (always logs to the central DB as there is no tenant ID yet)
			logHTTPRequest(ctx, uclog.LogLevelDebug, "%s", fmt.Sprintf("HTTP request | %s | %s %s ", sdkVersion, r.Method, r.RequestURI))

			srw := responsewriter.NewStatusResponseWriter(w)

			// Uncomment to dump HTTP Requests (and comment the 'NewStatusResponseWriter' line above)
			// TODO: better way to turn this on/off? command line arg?
			// lrw := httpdump.NewLoggingResponseWriter(w)
			// Infof(ctx, "%s", httpdump.DumpRequest(r))
			// Infof(ctx, "%s", httpdump.DumpCURLRequest(r))

			// NB: this line is very important, because it ensures that the context values set by
			// responsewriter.SizeLogger are available to the rest of the request processing chain
			// (specifically this middleware) after the SizeLogger has finished processing an outbound
			// response. This is tested in `responsewriter.TestGzip`. The alternative to this approach
			// would be 2x `SizeLogger`` middleware, one outside of `uclog.Middleware` to set up the context,
			// and one inside to do the actual measurement (before it's logged)
			*r = *r.WithContext(ctx)
			next.ServeHTTP(srw, r)

			// Uncomment to dump HTTP Responses
			// Infof(ctx, "%s", lrw.DumpResponse())

			// TODO get rid of this - log the http request to the tenant db now that it is resolved
			tenantID := uclog.GetTenantID(ctx)
			if !tenantID.IsNil() {
				logHTTPRequest(ctx, uclog.LogLevelVerbose, "%s", fmt.Sprintf("HTTP request | %s | %s %s | tenant %v", sdkVersion, r.Method, r.RequestURI, tenantID))
			}
			dbm, err := dbMetrics.GetMetrics(ctx)
			if err != nil {
				uclog.Errorf(ctx, "Unable to get DB metrics data from context %v", err)
			}
			cm, err := cacheMetrics.GetMetrics(ctx)
			if err != nil {
				uclog.Errorf(ctx, "Unable to get cache metrics data from context %v", err)
			}
			calls, hits, misses := cm.GetCounters()
			duration := time.Since(start)
			line := logLine{
				Status:           fmt.Sprintf("%7.d", srw.StatusCode),
				Duration:         duration,
				DBDuration:       dbm.GetTotalDuration(),
				DBCalls:          dbm.GetTotalCalls(),
				DBInFlight:       dbm.GetMaxInFlight(),
				CacheHits:        hits,
				CacheMisses:      misses,
				CacheCalls:       calls,
				CacheDuration:    cm.GetTotalDuration(),
				Hostname:         r.Host,
				Tenant:           tenantID,
				Method:           r.Method,
				RequestURI:       r.RequestURI,
				ResponseSizePre:  responsewriter.GetPreCompressionSize(r.Context()),
				ResponseSizePost: responsewriter.GetCompressedSize(r.Context()),
				QueueWaitTime:    queuemetrics.GetQueueStats(ctx),
			}

			buf := &bytes.Buffer{}
			if err := temp.Execute(buf, line); err != nil {
				uclog.Errorf(ctx, "uclog middleware template render: %v", err)
			}

			baseLogLevel := uclog.LogLevelInfo
			if tenantID.IsNil() {
				// We don't care about non-tenant requests as much (health checks, log server calls, etc
				//so in those cases the log level is debug (unless the responses is HTTP 4xx or 5xx)
				baseLogLevel = uclog.LogLevelDebug
			}
			// and log the request end
			logHTTPResponse(ctx, baseLogLevel, srw.StatusCode, tenantID, "%s", buf.String())

			// fetch information from the handler and log handler specific events
			handlerName := uclog.GetHandlerName(ctx)
			handlerError := uclog.GetHandlerErrorName(ctx)

			// if execution got to a handler
			if handlerName != "" {
				logPostHandlerExecution(ctx, handlerName, handlerError, tenantID, duration, dbm)
			}
			recordMetrics(r, handlerName, srw.StatusCode, &line, tenantID)
		})
	})
}
