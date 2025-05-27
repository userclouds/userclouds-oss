package ucsentry

import (
	"context"
	"time"

	"github.com/getsentry/sentry-go"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/request"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantmap"
)

// Init sets up sentry for UC
func Init(ctx context.Context, cfg *Config, release, machineName string) {
	if cfg == nil || cfg.Dsn == "" {
		uclog.Infof(ctx, "no sentry config present, skipping")
		return
	}
	// https://docs.sentry.io/platforms/go/configuration/options/
	options := sentry.ClientOptions{
		Dsn:              cfg.Dsn,
		EnableTracing:    cfg.TracesSampleRate > 0,
		TracesSampleRate: cfg.TracesSampleRate,
		AttachStacktrace: true,
		ServerName:       machineName,
		Release:          release,
		Environment:      string(universe.Current()),
	}
	if err := sentry.Init(options); err != nil {
		uclog.Errorf(ctx, "failed to init sentry: %v", err)
		return
	}
	uclog.Infof(ctx, "Sentry initialized: EnableTracing=%v TracesSampleRate=%v Release=%v ServerName=%v Environment=%v",
		options.EnableTracing, options.TracesSampleRate,
		options.Release, options.ServerName, options.Environment,
	)
	// put process-level data in the top-level scope for all events from this process
	// use tags so they are searchable / filterable here
	// https://docs.sentry.io/platforms/go/enriching-events/scopes/#configuring-the-scope
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("service", universe.ServiceName())
		scope.SetTag("region", string(region.Current()))
		// we'll add machine / host to the context since we're unlikely to need to search it
	})

	r := rf{}
	jsonapi.SetErrorReporter(r)
	uchttp.SetErrorReporter(r)
}

// Close calls sentry flush to send any remaining events
func Close(ctx context.Context) {
	uclog.Infof(ctx, "closing sentry")
	sentry.Flush(3 * time.Second)
}

// rf implements [uchttp|jsonapi].ErrorReportingFunc
type rf struct{}

// ReportingFunc bundles up a nice error reporting function for jsonapi & uchttp
func (rf rf) ReportingFunc() func(context.Context, error) {
	return func(ctx context.Context, err error) {
		sentry.WithScope(func(scope *sentry.Scope) {
			// https://docs.sentry.io/platforms/go/enriching-events/tags/
			dm := request.GetRequestDataMap(ctx)
			scope.SetTags(dm)

			// https://docs.sentry.io/platforms/go/enriching-events/context/#tructured-context
			if dm != nil { //HTTP request
				scope.SetContext("request", sentry.Context{
					"request_id":    request.GetRequestID(ctx),
					"hostname":      request.GetHostname(ctx),
					"user_agent":    request.GetUserAgent(ctx),
					"remote_ip":     request.GetRemoteIP(ctx),
					"forwarded_for": request.GetForwardedFor(ctx),
					"sdk_version":   request.GetSDKVersion(ctx),
				})
			} else { // panic in the worker? or background task/go routine?
				scope.SetContext("task", sentry.Context{
					"request_id": request.GetRequestID(ctx),
				})
			}
			if ts := multitenant.GetTenantState(ctx); ts != nil {
				SetTenant(scope, ts)
			}
			// https://docs.sentry.io/platforms/go/usage/#capturing-errors
			sentry.CaptureException(err)
		})
	}
}

// SetTenant sets tenant data to the sentry scope
func SetTenant(scope *sentry.Scope, ts *tenantmap.TenantState) {
	scope.SetTags(map[string]string{
		"tenant_id":    ts.ID.String(),
		"tenant_name":  ts.TenantName,
		"company_name": ts.CompanyName,
	})
	stc := sentry.Context{
		"id":           ts.ID.String(),
		"name":         ts.TenantName,
		"company_id":   ts.CompanyID.String(),
		"company_name": ts.CompanyName,
		"url":          ts.GetTenantURL()}

	scope.SetContext("tenant", stc)
}
