package multitenant

import (
	"context"

	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/tenantmap"
)

type contextKey int

const (
	ctxTenant contextKey = 1 // key for the Tenant structure used by the IDP
)

// MustGetTenantState returns a TenantState if one is set for this
// context or panics otherwise.
func MustGetTenantState(ctx context.Context) *tenantmap.TenantState {
	tenant := GetTenantState(ctx)
	if tenant == nil {
		panic(ucerr.New("tenant not set in context, did you forget to use multitenant.Middleware?"))
	}
	return tenant
}

// GetTenantState returns the tenant state, or nil if not set
func GetTenantState(ctx context.Context) *tenantmap.TenantState {
	val := ctx.Value(ctxTenant)
	tenant, ok := val.(*tenantmap.TenantState)
	if !ok {
		return nil
	}
	return tenant
}

// SetTenantState sets the tenant state on a context, used across both tests and production environments.
func SetTenantState(ctx context.Context, tenantState *tenantmap.TenantState) context.Context {
	return context.WithValue(ctx, ctxTenant, tenantState)
}
