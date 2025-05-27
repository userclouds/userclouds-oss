package provisioning_test

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	authzprovisioning "userclouds.com/authz/provisioning"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/ucdb"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/provisioning/types"
	"userclouds.com/internal/testhelpers"
)

func assertCompanyMatches(t *testing.T, company *companyconfig.Company, expectedName string, expectedID uuid.UUID) {
	t.Helper()
	assert.NotNil(t, company)
	assert.Equal(t, company.Name, expectedName)
	assert.Equal(t, company.ID, expectedID)
}

func TestCompanyProvisioningWithoutAuthZ(t *testing.T) {
	ctx := context.Background()
	ccfg, lcfg, s := testhelpers.NewTestStorage(t)
	cc, ct, cdb := testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, s, ccfg, lcfg)

	company1 := companyconfig.NewCompany("company1", companyconfig.CompanyTypeCustomer)
	company2 := companyconfig.NewCompany("company2", companyconfig.CompanyTypeCustomer)
	// Save original IDs to ensure provision updates in-place.
	company1ID := company1.ID
	company2ID := company2.ID

	// Create 2 test companies
	testhelpers.ProvisionTestCompany(ctx, t, s, &company1, cdb, ct.ID, cc.ID)
	assertCompanyMatches(t, &company1, "company1", company1ID)
	testhelpers.ProvisionTestCompany(ctx, t, s, &company2, cdb, ct.ID, cc.ID)
	assertCompanyMatches(t, &company2, "company2", company2ID)

	// Re-provision existing company1; make sure name changes, but ID stays the same.
	company1.Name = "company1-prime"
	testhelpers.ProvisionTestCompany(ctx, t, s, &company1, cdb, ct.ID, cc.ID)
	assertCompanyMatches(t, &company1, "company1-prime", company1ID)

	// Validate company1 and company2 are as expected from raw storage
	company1Storage, err := s.GetCompany(ctx, company1.ID)
	assert.NoErr(t, err)
	assertCompanyMatches(t, company1Storage, "company1-prime", company1ID)

	company2Storage, err := s.GetCompany(ctx, company2.ID)
	assert.NoErr(t, err)
	assertCompanyMatches(t, company2Storage, "company2", company2ID)
}

func setupConsoleTenant(t *testing.T, s *companyconfig.Storage, companyConfigDBCfg, logDBCfg *ucdb.Config) (*companyconfig.Tenant, *ucdb.DB) {
	ctx := context.Background()
	cacheCfg := cachetesthelpers.NewCacheConfig()
	ucCompany, consoleTenant, consoleTenantDB := testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, s, companyConfigDBCfg, logDBCfg)
	pi := types.ProvisionInfo{
		CompanyStorage: s,
		TenantDB:       consoleTenantDB,
		LogDB:          nil,
		CacheCfg:       cacheCfg,
		TenantID:       consoleTenant.ID,
	}
	// Validate AuthZ tenant provisioning for Console
	consoleTenantAuthZ, err := authzprovisioning.NewTenantAuthZ("console", pi, ucCompany)
	assert.NoErr(t, err)

	assert.IsNil(t, consoleTenantAuthZ.Validate(ctx))

	return consoleTenant, consoleTenantDB
}

func TestCompanyProvisioning(t *testing.T) {
	ctx := context.Background()
	companyConfigDBCfg, logDBCfg, s := testhelpers.NewTestStorage(t)
	consoleTenant, consoleTenantDB := setupConsoleTenant(t, s, companyConfigDBCfg, logDBCfg)
	pi := types.ProvisionInfo{CompanyStorage: s, TenantDB: consoleTenantDB, LogDB: nil, CacheCfg: nil, TenantID: consoleTenant.ID}
	company := companyconfig.NewCompany("company", companyconfig.CompanyTypeCustomer)

	companyOrgAuthZ, err := authzprovisioning.NewOrganizationProvisioner("", pi, company.ID, company.Name, "")
	assert.NoErr(t, err)
	// Not yet valid
	assert.NotNil(t, companyOrgAuthZ.Validate(ctx))

	// Provision a new company and set up AuthZ in Console
	testhelpers.ProvisionTestCompany(ctx, t, s, &company, consoleTenantDB, consoleTenant.ID, consoleTenant.CompanyID)
	assert.NoErr(t, err)
	assert.IsNil(t, companyOrgAuthZ.Validate(ctx))

	// Make sure it's safe to call again (though we aren't testing if duplicate resources get created)
	testhelpers.ProvisionTestCompany(ctx, t, s, &company, consoleTenantDB, consoleTenant.ID, consoleTenant.CompanyID)
}

func TestTenantProvisioning(t *testing.T) {
	ctx := context.Background()
	companyConfigDBCfg, logDBCfg, s := testhelpers.NewTestStorage(t)
	consoleTenant, consoleTenantDB := setupConsoleTenant(t, s, companyConfigDBCfg, logDBCfg)

	company := companyconfig.NewCompany(testhelpers.NewCompanyName(), companyconfig.CompanyTypeCustomer)
	testhelpers.ProvisionTestCompany(ctx, t, s, &company, consoleTenantDB, consoleTenant.ID, consoleTenant.CompanyID)
	tenant, _ := testhelpers.ProvisionTestTenant(ctx, t, s, companyConfigDBCfg, logDBCfg, company.ID)

	// Ensure idempotence
	sameTenant, _ := testhelpers.ProvisionTestTenant(ctx, t, s, companyConfigDBCfg, logDBCfg, company.ID, testhelpers.TenantID(tenant.ID), testhelpers.TenantName(tenant.Name))
	assert.Equal(t, sameTenant.ID, tenant.ID)
	assert.Equal(t, sameTenant.TenantURL, tenant.TenantURL)

	// Make sure we can deprovision the tenant
	testhelpers.ProvisionTestTenant(ctx, t, s, companyConfigDBCfg, logDBCfg, company.ID, testhelpers.TenantID(tenant.ID), testhelpers.TenantName("testtenant"), testhelpers.Cleanup())
}
