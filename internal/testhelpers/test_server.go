package testhelpers

import (
	"context"
	"fmt"
	"net"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	authzRoutes "userclouds.com/authz/routes"
	idpRoutes "userclouds.com/idp/routes"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/messaging/email"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/uctest"
	"userclouds.com/plex/test"
)

// CreateTestServer creates a test server with a tenant and a tenantDB (provisions a company and a tenant)
func CreateTestServer(ctx context.Context, t *testing.T, opts ...TestProvisionOption) (
	*companyconfig.Company,
	*companyconfig.Tenant,
	*companyconfig.Storage,
	*ucdb.DB,
	*uchttp.ServeMux,
	*tenantmap.StateMap) {

	dbCfg, logDBCfg, companyConfigStorage := NewTestStorage(t)
	company := ProvisionTestCompanyWithoutACL(ctx, t, companyConfigStorage)
	tenant, tenantDB := ProvisionTestTenant(ctx, t, companyConfigStorage, dbCfg, logDBCfg, company.ID, opts...)
	tenantsMap := NewTestTenantStateMap(companyConfigStorage)
	hb := builder.NewHandlerBuilder()
	jwtVerifier := uctest.JWTVerifier{}
	em, err := email.NewClient(ctx)
	assert.NoErr(t, err)
	test.InitForExternalTests(ctx, t, hb, companyConfigStorage, jwtVerifier, em, tenant.ID)
	idpRoutes.InitForTests(hb, tenantsMap, companyConfigStorage, tenant.ID, jwtVerifier)
	authzRoutes.InitForTests(hb, tenantsMap, companyConfigStorage, jwtVerifier)
	handler := hb.Build()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	UpdateTenantURL(ctx, t, companyConfigStorage, tenant, server)

	return company, tenant, companyConfigStorage, tenantDB, handler, tenantsMap
}

// UpdateTenantURL updates the tenant URL to the based on the server URL's port
func UpdateTenantURL(ctx context.Context, t *testing.T, companyConfigStorage *companyconfig.Storage, tenant *companyconfig.Tenant, server *httptest.Server) {
	port := strings.Split(server.URL, ":")[2]
	assert.True(t, strings.HasSuffix(tenant.TenantURL, "test.userclouds.tools"), assert.Must(), assert.Errorf("tenant URL must be under test.userclouds.tools"))
	newTenantURL := fmt.Sprintf("%s:%s", tenant.TenantURL, port)

	uclog.Debugf(ctx, "Update test tenant URL from '%s' to '%s'", tenant.TenantURL, newTenantURL)
	tenant.TenantURL = newTenantURL
	assert.NoErr(t, companyConfigStorage.SaveTenant(ctx, tenant))
	assertDNSResolution(t, tenant.TenantURL)
}

func assertDNSResolution(t *testing.T, tenantURL string) {
	parsedURL, err := url.Parse(tenantURL)
	assert.NoErr(t, err)
	addresses, err := net.LookupHost(parsedURL.Hostname())
	assert.NoErr(t, err)
	// see terraform/configurations/aws/common/domains/userclouds-tools.tf
	assert.Equal(t, len(addresses), 1, assert.Must(), assert.Errorf("tenant URL '%s' is not resolvable, is there an issue with .*test.userclouds.tools down ?", parsedURL.Hostname()))
	assert.Equal(t, addresses[0], "127.0.0.1", assert.Must(), assert.Errorf("tenant URL '%s' is not resolvable to 127.0.0.1", parsedURL.Hostname()))
}
