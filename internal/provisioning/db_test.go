package provisioning_test

import (
	"context"
	"os/exec"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/tools/generate/genschemas"
)

func TestOverrideDBConfig(t *testing.T) {
	ctx := context.Background()

	companyConfigDBCfg, logDBCfg, s := testhelpers.NewTestStorage(t)
	consoleTenant, consoleTenantDB := setupConsoleTenant(t, s, companyConfigDBCfg, logDBCfg)

	// set up a secondary container to test switching
	_, secondaryConnStr, name, _ := genschemas.StartTemporaryPostgres(ctx, "tempdb", 543)
	// NB: defer evaluates arguments at time of defer, not time of call
	// so this needs to be wrapped in a function :facepalm:
	defer func() {
		assert.NoErr(t, exec.Command("docker", "rm", "-f", name).Run())
	}()

	secondaryClusterConfig := genschemas.ConfigFromConnectionString(t, secondaryConnStr)

	// this is what we rely on to distinguish the two clusters later
	assert.NotEqual(t, secondaryClusterConfig.Port, companyConfigDBCfg.Port)

	company := companyconfig.NewCompany("company", companyconfig.CompanyTypeCustomer)
	testhelpers.ProvisionTestCompany(ctx, t, s, &company, consoleTenantDB, consoleTenant.ID, consoleTenant.CompanyID)
	tid := uuid.Must(uuid.NewV4())
	// first provision a tenant normally
	tenant, _ := testhelpers.ProvisionTestTenant(ctx, t, s, companyConfigDBCfg, logDBCfg, company.ID, testhelpers.TenantID(tid), testhelpers.TenantName("testoverridedb"))
	assert.Equal(t, tenant.ID, tid)
	assert.Equal(t, tenant.Name, "testoverridedb")

	// reprovision it to a new cluster
	movedTenant, _ := testhelpers.ProvisionTestTenant(ctx, t, s, companyConfigDBCfg, logDBCfg,
		company.ID, testhelpers.TenantID(tid), testhelpers.TenantName("testoverridedb"),
		testhelpers.OverrideDBConfig(&secondaryClusterConfig))
	assert.Equal(t, movedTenant.ID, tid)

	movedTI, err := s.GetTenantInternal(ctx, movedTenant.ID)
	assert.NoErr(t, err)
	assert.Equal(t, movedTI.TenantDBConfig.Port, secondaryClusterConfig.Port) // Port is the distinguishing field in testdbs

	// ensure that reprovisioning doesn't move it back
	unmovedTenant, _ := testhelpers.ProvisionTestTenant(ctx, t, s, companyConfigDBCfg, logDBCfg,
		company.ID, testhelpers.TenantID(tid), testhelpers.TenantName("testoverridedb"),
		testhelpers.OverrideDBConfig(&secondaryClusterConfig))
	assert.Equal(t, unmovedTenant.ID, tid)

	unmovedTI, err := s.GetTenantInternal(ctx, unmovedTenant.ID)
	assert.NoErr(t, err)
	assert.Equal(t, unmovedTI.TenantDBConfig.Port, secondaryClusterConfig.Port)

	// provision a new tenant using overrides
	newTID := uuid.Must(uuid.NewV4())
	newTenant, _ := testhelpers.ProvisionTestTenant(ctx, t, s, companyConfigDBCfg, logDBCfg,
		company.ID, testhelpers.TenantID(newTID), testhelpers.TenantName("testnewoverridedb"),
		testhelpers.OverrideDBConfig(&secondaryClusterConfig))
	assert.Equal(t, newTenant.ID, newTID)

	newTI, err := s.GetTenantInternal(ctx, newTenant.ID)
	assert.NoErr(t, err)
	assert.Equal(t, newTI.TenantDBConfig.Port, secondaryClusterConfig.Port)
}
