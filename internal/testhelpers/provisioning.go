package testhelpers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/provisioning"
	tenantProvisioning "userclouds.com/internal/provisioning/tenant"
	"userclouds.com/internal/provisioning/types"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/internal/testkeys"
	"userclouds.com/internal/uctest"
	"userclouds.com/plex/manager"
)

// TestTenantSubDomain is the subdomain used for test tenants, using *.test.userclouds.tools to resolve to localhost
// see terraform/configurations/aws/common/domains/userclouds-tools.tf
const TestTenantSubDomain = "some.tenant.test.userclouds.tools"
const testTenantProtocol = "http"

// UpdateTenantURLForTestTenant updates the tenantURL to point to the Server URL (updating the port on the tenant URL)
func UpdateTenantURLForTestTenant(t *testing.T, tenantURL, serverURL string) string {
	tenantParsed := uctest.MustParseURL(tenantURL)
	tenantParsed.Host = fmt.Sprintf("%s:%s", tenantParsed.Hostname(), uctest.MustParseURL(serverURL).Port())
	return tenantParsed.String()
}

// NewCompanyForTest provisions a new company for test use and sets up an ACL (AuthZ group) to control access to the company.
func NewCompanyForTest(ctx context.Context,
	t *testing.T,
	s *companyconfig.Storage,
	consoleTenantDB *ucdb.DB,
	consoleTenantID uuid.UUID,
	consoleCompanyID uuid.UUID,
	options ...provisioning.CompanyOption) *companyconfig.Company {
	name := NewCompanyName()
	company := companyconfig.NewCompany(name, companyconfig.CompanyTypeCustomer)
	ProvisionTestCompany(ctx, t, s, &company, consoleTenantDB, consoleTenantID, consoleCompanyID, options...)
	assert.Equal(t, company.Name, name, assert.Must(), assert.Errorf("ProvisionTestCompany failed, unexpected company name"))
	return &company
}

// ProvisionTestCompany provisions a new company for test use and sets up an ACL (AuthZ group) to control access to the company.
func ProvisionTestCompany(
	ctx context.Context,
	t *testing.T,
	s *companyconfig.Storage,
	company *companyconfig.Company,
	consoleTenantDB *ucdb.DB,
	consoleTenantID uuid.UUID,
	consoleCompanyID uuid.UUID,
	options ...provisioning.CompanyOption) {

	cacheCfg := cachetesthelpers.NewCacheConfig()
	pi := types.ProvisionInfo{
		CompanyStorage: s,
		TenantDB:       consoleTenantDB,
		LogDB:          nil,
		CacheCfg:       cacheCfg,
		TenantID:       consoleTenantID,
	}
	po, err := provisioning.NewProvisionableCompany(ctx, "ProvisionTestCompany", pi, company, consoleCompanyID, options...)
	assert.NoErr(t, err, assert.Errorf("NewProvisionableCompany failed"))
	assert.NoErr(t, po.Provision(ctx), assert.Errorf("Provision failed"))
	assert.NoErr(t, po.Validate(ctx), assert.Errorf("Validate failed"))
}

// ProvisionTestCompanyWithoutACL provisions a new company for test use but doesn't set
// AuthZ on the company itself (suitable for tests, but not for prod).
func ProvisionTestCompanyWithoutACL(ctx context.Context, t *testing.T, s *companyconfig.Storage) *companyconfig.Company {
	return NewCompanyForTest(ctx, t, s, nil, uuid.Nil, uuid.Nil)
}

// TestProvisionOption implements options for test tenant provisioning
type TestProvisionOption interface {
	apply(*TestProvisionConfig)
}

type optFunc func(*TestProvisionConfig)

func (o optFunc) apply(po *TestProvisionConfig) {
	o(po)
}

// TestProvisionConfig contains various options for for test tenant provisioning
type TestProvisionConfig struct {
	override         *ucdb.Config
	cleanup          bool
	useOrganizations bool
	employeeIDs      []uuid.UUID
	tenantID         uuid.UUID
	tenantName       string
}

// TenantID specifies the tenant ID to use for provisioning
func TenantID(tenantID uuid.UUID) TestProvisionOption {
	return optFunc(func(po *TestProvisionConfig) {
		po.tenantID = tenantID
	})
}

// TenantName specifies the tenant name to use for provisioning
func TenantName(tenantName string) TestProvisionOption {
	return optFunc(func(po *TestProvisionConfig) {
		po.tenantName = tenantName
	})
}

// Cleanup specifies that the tenant should be deleted after provisioning and validation
func Cleanup() TestProvisionOption {
	return optFunc(func(po *TestProvisionConfig) {
		po.cleanup = true
	})
}

// OverrideDBConfig allows ProvisionTestTenant to override the bootstraped DB for testing
func OverrideDBConfig(override *ucdb.Config) TestProvisionOption {
	return optFunc(func(po *TestProvisionConfig) {
		po.override = override
	})
}

// UseOrganizations specifies that the tenant should be created with organizations enabled
func UseOrganizations() TestProvisionOption {
	return optFunc(func(po *TestProvisionConfig) {
		po.useOrganizations = true
	})
}

// EmployeeIDs specifies the employee IDs to provision as shadow objects
func EmployeeIDs(employeeIDs []uuid.UUID) TestProvisionOption {
	return optFunc(func(po *TestProvisionConfig) {
		po.employeeIDs = employeeIDs
	})
}

// this exists to ensure that the use of the same companyconfig *ucdb.Config
// doesn't trigger race detection across parallel tests
var tenantProvisioningLock sync.Mutex

// ProvisionTestTenant wraps the regular tenant provisioning path to make it
// easier to use in tests by providing reasonable defaults for most settings.
func ProvisionTestTenant(
	ctx context.Context,
	t *testing.T,
	ccs *companyconfig.Storage,
	companyConfigDBCfg *ucdb.Config,
	logDBCfg *ucdb.Config,
	companyID uuid.UUID,
	opts ...TestProvisionOption) (*companyconfig.Tenant, *ucdb.DB) {

	// Get the optional parameters if any
	to := &TestProvisionConfig{}
	for _, v := range opts {
		v.apply(to)
	}
	tenantID := to.tenantID
	if tenantID.IsNil() {
		tenantID = uuid.Must(uuid.NewV4())
	}
	tenantName := to.tenantName
	if tenantName == "" {
		tenantName = NewTenantName()
	}

	company, err := ccs.GetCompany(ctx, companyID)
	assert.NoErr(t, err, assert.Errorf("GetCompany failed"))

	tenantURL, err := tenantProvisioning.GenerateTenantURL(company.Name, tenantName, testTenantProtocol, TestTenantSubDomain)
	uclog.Infof(ctx, "Provisioning tenant %s for company: %s with URL %s", tenantName, company.Name, tenantURL)
	assert.NoErr(t, err, assert.Errorf("GenerateTenantURL failed"))

	tenant := &companyconfig.Tenant{
		BaseModel:        ucdb.NewBaseWithID(tenantID),
		Name:             tenantName,
		CompanyID:        companyID,
		TenantURL:        tenantURL,
		UseOrganizations: to.useOrganizations,
	}

	cacheCfg := cachetesthelpers.NewCacheConfig()
	// Try to re-use existing plex config if it exists
	// but since there are two failure modes, we're going to default
	// to creating a new one, and then overwriting it if we find one
	//
	// We use testkeys.Config rather than generating them each time since as
	// of 10/23 profiling runs, that saves us almost 10% per test
	_, tc, err := tenantProvisioning.CreatePlexConfig(ctx, tenant, tenantProvisioning.UseKeys(&testkeys.Config))
	assert.NoErr(t, err)
	tc.Keys = testkeys.Config
	tenantPlex := &tenantplex.TenantPlex{
		VersionBaseModel: ucdb.NewVersionBaseWithID(tenantID),
		PlexConfig:       *tc,
	}

	mgr, err := manager.NewFromCompanyConfig(ctx, ccs, tenantID, cacheCfg)
	// if this fails (because we haven't provisioned the tenant / DB yet, that's ok)
	if err == nil {
		defer mgr.Close(ctx)
		existingTP, err := mgr.GetTenantPlex(ctx, tenantID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			// if we found the database, but not the TenantPlex, there's something
			// deeper wrong here and we should fail
			assert.NoErr(t, err)
		}
		tenantPlex = existingTP
	}

	tenantProvisioningLock.Lock()
	pt, err := tenantProvisioning.NewProvisionableTenant(ctx,
		"ProvisionTestTenant",
		tenant,
		tenantPlex,
		ccs,
		companyConfigDBCfg,
		to.override,
		logDBCfg,
		cacheCfg,
		to.employeeIDs,
	)
	tenantProvisioningLock.Unlock()
	assert.NoErr(t, err, assert.Errorf("NewProvisionableTenant failed"))
	ctx = ucdb.SetRetryConfig(ctx, 100, 3000, 6000, 10000, 150000)
	defer func() {
		ctx = ucdb.SetRetryConfig(ctx)
	}()
	assert.NoErr(t, pt.Provision(ctx), assert.Errorf("Provision failed"))
	// (sgarrity 8/24): Validate was designed to be called before provisioning
	// to see if an object was in a good state, not after (it's never caught
	// an issue AFAIK?), and this is expensive in all of our tests
	// assert.NoErr(t, pt.Validate(ctx), assert.Errorf("Validate failed"))

	if to.cleanup {
		assert.NoErr(t, pt.Cleanup(ctx), assert.Errorf("Cleanup failed"))
		assert.NoErr(t, pt.Close(ctx), assert.Errorf("Close failed"))
	}
	return tenant, pt.TenantDB
}

// ProvisionConsoleCompanyAndTenant creates and returns the User Clouds company,
// Console tenant, and Console Tenant DB (for IDP/AuthZ).
func ProvisionConsoleCompanyAndTenant(ctx context.Context, t *testing.T, s *companyconfig.Storage, companyConfigDBCfg *ucdb.Config, logDBCfg *ucdb.Config) (*companyconfig.Company, *companyconfig.Tenant, *ucdb.DB) {
	ucCompany := ProvisionTestCompanyWithoutACL(ctx, t, s)
	ucCompany.Type = companyconfig.CompanyTypeInternal
	assert.NoErr(t, s.SaveCompany(ctx, ucCompany))
	consoleTenant, consoleTenantDB := ProvisionTestTenant(ctx, t, s, companyConfigDBCfg, logDBCfg, ucCompany.ID, UseOrganizations())
	// Idempotently re-provision the userclouds test company so authz gets set up properly,
	// now that the console app exists. We do the same thing in our provision scripts.
	ProvisionTestCompany(ctx, t, s, ucCompany, consoleTenantDB, consoleTenant.ID, ucCompany.ID)
	return ucCompany, consoleTenant, consoleTenantDB
}

// NewCompanyName creates a new dummy company name
func NewCompanyName() string {
	return fmt.Sprintf("c%s", newName())
}

// NewTenantName creates a new dummy tenant name
func NewTenantName() string {
	return fmt.Sprintf("t%s", newName())
}

// 24 to fix inside 30char limit
func newName() string {
	return uuid.Must(uuid.NewV4()).String()[0:24]
}
