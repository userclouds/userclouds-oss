package authn

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantmap"
)

// AuthN stores runtime AuthN-specific tenant state.
type AuthN struct {
	ID                     uuid.UUID
	UseOrganizations       bool
	CompanyID              uuid.UUID
	Manager                *Manager
	ConfigStorage          *storage.Storage
	UserMultiRegionStorage *storage.UserMultiRegionStorage
}

// NewAuthN creates a new NewAuthN object from a multitenant.Tenant
func NewAuthN(ctx context.Context, ts *tenantmap.TenantState) *AuthN {
	s := storage.NewFromTenantState(ctx, ts)
	userMultiRegionStorage := storage.NewUserMultiRegionStorage(ctx, ts.UserRegionDbMap, ts.ID)
	m := NewManager(s, userMultiRegionStorage)

	return &AuthN{
		ID:                     ts.ID,
		UseOrganizations:       ts.UseOrganizations,
		CompanyID:              ts.CompanyID,
		Manager:                m,
		ConfigStorage:          s,
		UserMultiRegionStorage: userMultiRegionStorage,
	}
}

// MustGetTenantAuthn returns the AuthN from the context, and requires that
// multitenant.Middleware was used on the handler.
func MustGetTenantAuthn(ctx context.Context) *AuthN {
	tenant := multitenant.MustGetTenantState(ctx)
	return NewAuthN(ctx, tenant)
}

// GetManager returns a credentials.Manager for a given hostname.
func GetManager(ctx context.Context, tenantMap *tenantmap.StateMap, hostname string) (*Manager, error) {
	ts, err := tenantMap.GetTenantStateForHostname(ctx, hostname)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return NewAuthN(ctx, ts).Manager, nil
}
