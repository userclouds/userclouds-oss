package tenantstate

import (
	"context"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/authz/internal"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/multitenant"
)

// AuthZ stores runtime AuthZ-specific tenant state.
type AuthZ struct {
	TenantID         uuid.UUID
	TenantURL        *url.URL
	UseOrganizations bool
	CompanyID        uuid.UUID
	Storage          *internal.Storage
	AuditLogStore    *auditlog.Storage
}

// MustGet returns the tenant from the context, and requires that
// multitenant.Middleware was used on the handler.
func MustGet(ctx context.Context) *AuthZ {
	tenant := multitenant.MustGetTenantState(ctx)
	authz := &AuthZ{
		TenantID:         tenant.ID,
		TenantURL:        tenant.TenantURL,
		UseOrganizations: tenant.UseOrganizations,
		CompanyID:        tenant.CompanyID,
		Storage:          internal.NewStorage(ctx, tenant.ID, tenant.TenantDB, tenant.CacheConfig),
		AuditLogStore:    auditlog.NewStorage(tenant.TenantDB),
	}
	return authz
}
