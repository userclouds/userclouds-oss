package auditlog

import (
	"context"
	"net/http"

	"userclouds.com/infra/middleware"
	"userclouds.com/internal/multitenant"
)

type contextKey int

const ctxAuditLogStorage contextKey = 1

// MustGetAuditLogStorage get a audit log storage, or panics
func mustGetAuditLogStorage(ctx context.Context) *Storage {
	if ts := multitenant.GetTenantState(ctx); ts != nil {
		return NewStorage(ts.TenantDB)
	}

	// hopefully auditlog.Middleware is in use :)
	val := ctx.Value(ctxAuditLogStorage)
	s, ok := val.(*Storage)
	if !ok {
		panic("auditlog storage not set in context")
	}
	return s

}

// Middleware returns a middleware that adds the tenant DB to the request context
// for the purposes of easy audit logging (eg so every auditlog.Post() doesn't require
// a tenantDB argument)
//
// NB: this middleware is not required if multitenant.Middleware is used (most services
// besides console), because we can grab the tenantDB there
func Middleware(s *Storage) middleware.Middleware {
	return middleware.Func((func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ctxAuditLogStorage, s)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}))
}
