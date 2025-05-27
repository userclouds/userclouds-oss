package multitenant

import (
	"errors"
	"net/http"

	"userclouds.com/infra/middleware"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctrace"
	"userclouds.com/internal/tenantmap"
)

// Middleware determines the Tenant from the request Host header
// and sets it on the context.
func Middleware(tm *tenantmap.StateMap) middleware.Middleware {
	return middleware.Func((func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			tenantState, err := tm.GetTenantStateForHostname(ctx, r.Host)
			if err != nil {
				if errors.Is(err, tenantmap.ErrInvalidTenantName) {
					uchttp.Error(ctx, w, err, http.StatusNotFound)
				} else {
					uchttp.Error(ctx, w, err, http.StatusBadRequest)
				}
				return
			}

			ctx = SetTenantState(ctx, tenantState)
			// This is a duplicate but we don't set tenantState in non multi-tenant services (like console/logserver) so duplicate the ID
			// to make the GetTenantID function work in both cases
			ctx = uclog.SetTenantID(ctx, tenantState.ID)
			uclog.Verbosef(ctx, "domain %v mapped to tenant %v", r.Host, tenantState.ID)

			currSpan := uctrace.GetCurrentSpan(ctx)
			SetTenantAttributes(currSpan, tenantState)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}))
}

// SetTenantAttributes sets the tenant attributes on the span
func SetTenantAttributes(span uctrace.Span, ts *tenantmap.TenantState) {
	if ts == nil {
		return
	}
	uctrace.SetTenantAttributes(span, ts.ID, ts.TenantName, ts.GetTenantURL(), ts.CompanyID, ts.CompanyName)
}
