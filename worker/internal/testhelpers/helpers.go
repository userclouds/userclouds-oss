package testhelpers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid"

	authzroutes "userclouds.com/authz/routes"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	tenantProvisioning "userclouds.com/internal/provisioning/tenant"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
	plexroutes "userclouds.com/plex/routes"
	"userclouds.com/worker/config"
	"userclouds.com/worker/internal"
)

// GetTenantURLForServerURL returns the tenant URL for a given server URL
func GetTenantURLForServerURL(t *testing.T, srvURL, tenantName, companyName string) string {
	tenantURL, err := tenantProvisioning.GenerateTenantURL(companyName, tenantName, "http", testhelpers.TestTenantSubDomain)
	assert.NoErr(t, err)
	return testhelpers.UpdateTenantURLForTestTenant(t, tenantURL, srvURL)
}

// CreateTestWorkerConfig creates a worker config for testing
func CreateTestWorkerConfig(ctx context.Context, t *testing.T, companyDBConfig, logDBConfig *ucdb.Config, consoleTenant *companyconfig.Tenant, consoleTenantDB *ucdb.DB) *config.Config {
	return &config.Config{
		ConsoleTenantID: consoleTenant.ID,
		CompanyDB:       *companyDBConfig,
		LogDB:           *logDBConfig,
		CacheConfig:     cachetesthelpers.NewCacheConfig(),
		WorkerClient:    workerclient.Config{Type: workerclient.TypeTest},
	}
}

// SetupWorkerForTest sets up a worker for testing, returning the handler, storage, console tenant, and console tenant db
func SetupWorkerForTest(ctx context.Context, t *testing.T, wc workerclient.Client) (http.Handler, *companyconfig.Storage, *companyconfig.Tenant, *ucdb.DB) {
	cdbc, companyDBConfig, ccs := testhelpers.NewTestStorage(t)
	_, consoleTenant, consoleTenantDB := testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, ccs, cdbc, companyDBConfig)

	// set up our "main" services since we need plex & authz
	srv := CreateTestHTTPServer(ctx, t, ccs, consoleTenant.ID)
	tenantServerURL := testhelpers.UpdateTenantURLForTestTenant(t, consoleTenant.TenantURL, srv.URL)
	testhelpers.FixupTenantURL(t, ccs, consoleTenant, tenantServerURL, consoleTenantDB)

	// reload after fixup
	consoleTenant, err := ccs.GetTenant(ctx, consoleTenant.ID)
	assert.NoErr(t, err)
	m2mAuth, err := m2m.GetM2MTokenSource(ctx, consoleTenant.ID)
	assert.NoErr(t, err)
	consoleTenantInfo, err := ccs.GetTenantInfo(ctx, consoleTenant.ID)
	assert.NoErr(t, err)

	wcfg := CreateTestWorkerConfig(ctx, t, companyDBConfig, companyDBConfig, consoleTenant, consoleTenantDB)
	wh := internal.NewHTTPHandler(wcfg, ccs, m2mAuth, *consoleTenantInfo, tenantmap.NewStateMap(ccs, nil), wc, internal.NewRunningTasks())
	return wh, ccs, consoleTenant, consoleTenantDB
}

// CreateTestHTTPServer creates a test http server for testing
func CreateTestHTTPServer(ctx context.Context, t *testing.T, ccs *companyconfig.Storage, consoleTenantID uuid.UUID) *httptest.Server {
	hb := builder.NewHandlerBuilder()
	m2mAuth, err := m2m.GetM2MTokenSource(ctx, consoleTenantID)
	assert.NoErr(t, err)
	plexroutes.InitForTests(ctx, m2mAuth, hb, ccs, uctest.JWTVerifier{}, &testhelpers.SecurityValidator{}, nil, nil, nil, consoleTenantID)
	authzroutes.InitForTests(hb, testhelpers.NewTestTenantStateMap(ccs), ccs, uctest.JWTVerifier{})
	testServer := httptest.NewServer(hb.Build())
	t.Cleanup(testServer.Close)
	return testServer
}
