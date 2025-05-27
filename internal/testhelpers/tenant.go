package testhelpers

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/ucdb"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/manager"
)

// NewNamedTenant returns a new test tenant  with the provided name
func NewNamedTenant(name string) *companyconfig.Tenant {
	return &companyconfig.Tenant{
		BaseModel: ucdb.NewBase(),
		Name:      name,
		CompanyID: uuid.Must(uuid.NewV4()),
		TenantURL: "https://test.tenant.dev.userclouds.tools:3333",
	}
}

// NewTenant returns a new test tenant
func NewTenant() *companyconfig.Tenant {
	return NewNamedTenant("test tenant")
}

// FixupTenantURL updates a tenant's URL in all relevant DB locations: companyconfig tenant table,
// tenant plex config, and UC provider in plex map.
// This also means we don't need to override the Host header of clients in test requests.
// This also updates the originalTenant in place since we always do that
func FixupTenantURL(t *testing.T, companyConfigStorage *companyconfig.Storage, originalTenant *companyconfig.Tenant, newTenantURL string, tenantDB *ucdb.DB) {
	ctx := context.Background()
	// Update `tenants` table
	originalTenant.TenantURL = newTenantURL
	err := companyConfigStorage.SaveTenant(ctx, originalTenant)
	assert.NoErr(t, err)

	// Update Plex Map to point to our tenant server instead of the default dummy tenant URL
	// TODO: should `UpdateTenantURL` do this automatically? It's a little tricky to wade into a plex map
	// which may have multiple providers, but maybe we can assume any UC providers should be fixed up to point to this IDP.
	cacheCfg := cachetesthelpers.NewCacheConfig()
	mgr := manager.NewFromDB(tenantDB, cacheCfg)
	tp, err := mgr.GetTenantPlex(ctx, originalTenant.ID)
	assert.NoErr(t, err)

	// The default plex map should have 1 provider (UC)
	assert.Equal(t, len(tp.PlexConfig.PlexMap.Providers), 1, assert.Must())
	assert.Equal(t, tp.PlexConfig.PlexMap.Providers[0].Type, tenantplex.ProviderTypeUC, assert.Must())
	tp.PlexConfig.PlexMap.Providers[0].UC.IDPURL = newTenantURL
	err = mgr.SaveTenantPlex(ctx, tp)
	assert.NoErr(t, err)
}
