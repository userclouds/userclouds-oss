package middleware

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/middleware"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantmap"
)

// TODO (sgarrity 8/24): this whole thing should probably go away over time but
// at least now it uses the multitenant package
var consoleTenantID uuid.UUID

// TenantMatchesOrIsConsole centralizes logserver permission checks
func TenantMatchesOrIsConsole(ctx context.Context, tenantID uuid.UUID) bool {
	contextTenantID := GetTenantID(ctx)
	return contextTenantID == consoleTenantID || contextTenantID == tenantID
}

// GetTenantID gets tenant ID from context if it was passed in with the request
func GetTenantID(ctx context.Context) uuid.UUID {
	ts := multitenant.GetTenantState(ctx)
	if ts == nil {
		// this is an error case (likely meaning the middleware wasn't run), not a permission case
		uclog.Warningf(ctx, "no tenant ID found in context, did you forget logserver.internal.middleware.TenantID?")
		return uuid.Nil
	}
	return ts.ID
}

// ClearTenantID clears tenant ID from context
func ClearTenantID(ctx context.Context) {
	ts := multitenant.GetTenantState(ctx)
	if ts == nil {
		return
	}
	ts.ID = consoleTenantID
}

// TenantID middleware determines the tenant from the request query parameter
// and sets it on the context.
func TenantID(ctID uuid.UUID) middleware.Middleware {
	consoleTenantID = ctID
	return middleware.Func((func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			tenantString := r.URL.Query().Get("tenant_id")
			if tenantID, err := uuid.FromString(tenantString); err == nil {
				ctx = setTenantID(ctx, tenantID)
			} else {
				// this ensures that the tenantID is set to nil if the query param is not present
				// (and that multitenant.GetTenantState(ctx) will return an object)
				// This means that the request must successfully auth as console, but then
				// will have access to any tenant
				ctx = setTenantID(ctx, consoleTenantID)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}))
}

func setTenantID(ctx context.Context, tenantID uuid.UUID) context.Context {
	return multitenant.SetTenantState(ctx, &tenantmap.TenantState{ID: tenantID})
}
