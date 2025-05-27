package tenant

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctrace"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/provisioning"
	"userclouds.com/internal/provisioning/types"
	"userclouds.com/internal/tenantdb"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/internal/tenantplex/builder"
	"userclouds.com/plex/manager"
	"userclouds.com/plex/serviceconfig"
)

const maxTenantComponentNameLength = 25 // chosen arbitrarily

// GenerateTenantURL generates a consistent URL for tenants
// This is mostly factored out so that we can do eg. a domain-collision check before
// bin/provision adds a tenant
func GenerateTenantURL(companyName, tenantName string, tenantProtocol, tenantSubDomain string) (string, error) {
	if len(companyName) > maxTenantComponentNameLength {
		return "", ucerr.Friendlyf(nil, "Company name '%s' is too long (max %d characters)", companyName, maxTenantComponentNameLength)
	}
	if len(tenantName) > maxTenantComponentNameLength {
		return "", ucerr.Friendlyf(nil, "Tenant name '%s' is too long (max %d characters)", tenantName, maxTenantComponentNameLength)
	}

	compositeName := fmt.Sprintf("%s-%s", companyName, tenantName)
	hostname, err := companyconfig.GenerateSafeHostname(compositeName)
	if err != nil {
		return "", ucerr.Wrap(err) // TODO: should plumb 400 through here
	}
	tenantURL := strings.ToLower(fmt.Sprintf("%s://%s.%s", tenantProtocol, hostname, tenantSubDomain))
	return tenantURL, nil
}

// ProvisionableTenant provisions, validates, and de-provisions tenants.
type ProvisionableTenant struct {
	types.Named
	types.Parallelizable
	tenant               *companyconfig.Tenant
	companyConfigStorage *companyconfig.Storage
	employeeIDs          []uuid.UUID

	// we store this instead of plex config to maintain versioning and detect
	// data races during provisioning
	tenantPlex *tenantplex.TenantPlex

	// if non-nil, saves us reopening a lot of connections
	TenantDB *ucdb.DB

	tdb *tenantDB
	ldb *logDB

	cacheCfg *cache.Config
	p        types.Provisionable
}

// NewProvisionableTenant creates a new object which can provision, validate, or
// de-provision a full tenant.
func NewProvisionableTenant(ctx context.Context,
	name string,
	tenant *companyconfig.Tenant,
	tenantPlex *tenantplex.TenantPlex,
	companyConfigStorage *companyconfig.Storage,
	companyConfigDBCfg,
	overrideBootstrapDBCfg *ucdb.Config,
	statusDBCfg *ucdb.Config,
	cc *cache.Config,
	employeeIDs []uuid.UUID) (*ProvisionableTenant, error) {

	company, err := companyConfigStorage.GetCompany(ctx, tenant.CompanyID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	tenantInternal, err := companyConfigStorage.GetTenantInternal(ctx, tenant.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		// Ignore ErrNoRows (as that is expected for new tenants) but other
		// errors are unexpected.
		uclog.Debugf(ctx, "Failed to load TenantInternal (id: %s) for provisioning: %v", tenant.ID, err)
		return nil, ucerr.Wrap(err)
	}

	if tenantInternal == nil {
		tenantInternal = &companyconfig.TenantInternal{
			BaseModel:         ucdb.NewBaseWithID(tenant.ID),
			PrimaryUserRegion: region.DefaultUserDataRegionForUniverse(universe.Current()),
		}
	}

	// we make a copy of this here so that we can protect this call in tests under a lock
	// (since we share the same companyconfig *ucdb.Config across many tests)
	copyBootstrap := *companyConfigDBCfg

	return &ProvisionableTenant{
		Named:                types.NewNamed(fmt.Sprintf("%s:Tenant(%v)", name, tenant.ID)),
		Parallelizable:       types.AllParallelizable(),
		tenant:               tenant,
		companyConfigStorage: companyConfigStorage,
		tenantPlex:           tenantPlex,
		tdb: &tenantDB{
			tenantID:               tenant.ID,
			company:                company,
			companyConfigStorage:   companyConfigStorage,
			bootstrapDBCfg:         &copyBootstrap,
			tenantInternal:         tenantInternal,
			overrideBootstrapDBCfg: overrideBootstrapDBCfg,
		},
		ldb: &logDB{
			tenantID:             tenant.ID,
			companyConfigStorage: companyConfigStorage,
			bootstrapDBCfg:       statusDBCfg,
			tenantInternal:       tenantInternal,
		},
		employeeIDs: employeeIDs,
		cacheCfg:    cc,
	}, nil
}

// NewProvisionableTenantFromExisting creates a ProvisionableTenant based on an existing tenant in the DB.
// Useful for deprovisioning or validating an existing tenant. TODO - we current only use this from DeleteTenant handler but not from provisioning
func NewProvisionableTenantFromExisting(ctx context.Context,
	name string,
	tenantID uuid.UUID,
	companyConfigStorage *companyconfig.Storage,
	companyConfigDBCfg *ucdb.Config,
	statusDBCfg *ucdb.Config,
	cc *cache.Config) (*ProvisionableTenant, error) {

	name = fmt.Sprintf("%s:Tenant[Existing](%v)", name, tenantID)

	// NOTE: technically we only need the tenant ID + DB Cfg objects to Validate or Cleanup/Nuke a tenant.
	// We could avoid these Gets in that case, but that may cause bugs later.
	tenant, err := companyConfigStorage.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	company, err := companyConfigStorage.GetCompany(ctx, tenant.CompanyID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	tenantInternal, err := companyConfigStorage.GetTenantInternal(ctx, tenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	tenDB, _, _, err := tenantdb.Connect(ctx, companyConfigStorage, tenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	mgr := manager.NewFromDB(tenDB, cc)

	tenantPlex, err := mgr.GetTenantPlex(ctx, tenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	p, err := NewServicesProvisioner(ctx, name, companyConfigStorage, tenDB, &tenantInternal.LogConfig.LogDB, cc, company, tenant.ID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &ProvisionableTenant{
		Named:                types.NewNamed(name),
		Parallelizable:       types.AllParallelizable(),
		tenant:               tenant,
		companyConfigStorage: companyConfigStorage,
		tenantPlex:           tenantPlex,
		p:                    p,
		TenantDB:             tenDB,
		tdb: &tenantDB{
			tenantID:             tenantID,
			company:              company,
			companyConfigStorage: companyConfigStorage,
			bootstrapDBCfg:       companyConfigDBCfg,
			tenantInternal:       tenantInternal,
		},
		ldb: &logDB{
			tenantID:             tenantID,
			companyConfigStorage: companyConfigStorage,
			bootstrapDBCfg:       statusDBCfg,
			tenantInternal:       tenantInternal,
		},
		cacheCfg: cc,
	}, nil
}

// Provision implements Provisionable
func (t *ProvisionableTenant) Provision(ctx context.Context) error {
	return uctrace.Wrap0(ctx, tracer, "ProvisionTenant", true, func(ctx context.Context) error {
		uclog.Infof(ctx, "Start provisioning tenant '%s' (id: %s)...", t.tenant.Name, t.tenant.ID)

		if err := t.tenant.Validate(); err != nil {
			uclog.Errorf(ctx, "Tenant '%s' (id: %s) failed basic validation: %v", t.tenant.Name, t.tenant.ID, err)
			return ucerr.Wrap(err) // TODO: should plumb 400 through here
		}

		if _, err := t.companyConfigStorage.GetCompany(ctx, t.tenant.CompanyID); err != nil {
			uclog.Errorf(ctx, "Tenant '%s' (id: %s) company (id: %s) not loaded: %v", t.tenant.Name, t.tenant.ID, t.tenant.CompanyID, err)
			return ucerr.Wrap(err) // TODO: should plumb 400 through here
		}

		if err := t.companyConfigStorage.SaveTenant(ctx, t.tenant); err != nil {
			uclog.Errorf(ctx, "Failed to save tenant '%s' (id: %s): %v", t.tenant.Name, t.tenant.ID, err)
			return ucerr.Friendlyf(err, "Failed to save tenant, possibly due to another tenant with same name")
		}

		uclog.Infof(ctx, "Provisioning Tenant DB for tenant '%s' (id: %s)", t.tenant.Name, t.tenant.ID)
		if err := t.tdb.provision(ctx); err != nil {
			uclog.Errorf(ctx, "Failed to provision Tenant DB for tenant '%s' (id: %s): %v", t.tenant.Name, t.tenant.ID, err)
			return ucerr.Wrap(err)
		}
		t.TenantDB = t.tdb.TenantDB

		uclog.Infof(ctx, "Provisioning Log DB for tenant '%s' (id: %s)", t.tenant.Name, t.tenant.ID)
		if err := t.ldb.provision(ctx); err != nil {
			uclog.Errorf(ctx, "Failed to provision Log DB for tenant '%s' (id: %s): %v", t.tenant.Name, t.tenant.ID, err)
			return ucerr.Wrap(err)
		}

		uclog.Infof(ctx, "Saving TenantInternal for %v - name:'%s' url:'%s'", t.tenant.ID, t.tenant.Name, t.tenant.TenantURL)
		if t.tdb.tenantInternal != t.ldb.tenantInternal {
			return ucerr.Errorf("internal consistency error: tenantDB.ti != loggingDB.ti")
		}
		if err := t.companyConfigStorage.SaveTenantInternal(ctx, t.tdb.tenantInternal); err != nil {
			return ucerr.Wrap(err)
		}

		// need to open the db connection if this wasn't an existing tenant
		if t.TenantDB == nil {
			var err error
			t.TenantDB, _, _, err = tenantdb.Connect(ctx, t.companyConfigStorage, t.tenant.ID)
			if err != nil {
				return ucerr.Wrap(err)
			}
		}

		uclog.Debugf(ctx, "Writing out Plex Config for tenant '%s' (id: %s)", t.tenant.Name, t.tenant.ID)
		mgr := manager.NewFromDB(t.TenantDB, t.cacheCfg)
		if err := mgr.SaveTenantPlex(ctx, t.tenantPlex); err != nil {
			uclog.Errorf(ctx, "Failed to save tenant '%s' (id: %s) Plex Config: %v", t.tenant.Name, t.tenant.ID, err)
			return ucerr.Wrap(err)
		}

		// create initial m2m secret for tenant. We don't actually care what the value is here, but we need it saved to secrets
		if err := m2m.CreateSecret(ctx, t.tenant); err != nil {
			return ucerr.Wrap(err)
		}

		uclog.Infof(ctx, "Provisioning services in Tenant DB (tenant id: %s)", t.tenant.ID)

		p, err := NewServicesProvisioner(ctx, t.Name(), t.companyConfigStorage, t.TenantDB, &t.tdb.tenantInternal.LogConfig.LogDB, t.cacheCfg, t.tdb.company, t.tenant.ID, t.employeeIDs...)
		if err != nil {
			return ucerr.Wrap(err)
		}
		// If we called Validate first - we don't reuse that provisionable as DB connections may have changed
		if t.p != nil {
			if err := t.p.Close(ctx); err != nil {
				return ucerr.Wrap(err)
			}
		}
		t.p = p

		if err := t.p.Provision(ctx); err != nil {
			uclog.Errorf(ctx, "Failed to provision services in Tenant DB (tenant id: %s): %v", t.tenant.ID, err)
			return ucerr.Wrap(err)
		}
		if err := ProvisionRegionalTenantURLs(ctx, t.companyConfigStorage, t.tenant, false); err != nil {
			return ucerr.Wrap(err)
		}

		uclog.Infof(ctx, "Successfully provisioned tenant '%s' (id: %s)!", t.tenant.Name, t.tenant.ID)
		return nil
	})
}

// ProvisionRegionalTenantURLs provisions region-specific tenant URLs for a tenant
func ProvisionRegionalTenantURLs(ctx context.Context, companyConfigStorage *companyconfig.Storage, tenant *companyconfig.Tenant, addEKSURLs bool) error {
	// currently naive, but since this is what we replace, don't fail weirdly with different tenant URLs (eg in testing)
	pager, err := companyconfig.NewTenantURLPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return ucerr.Wrap(err)
	}
	regions := getRegionsForTenantURL(ctx, tenant)
	if regions == nil {
		return nil
	}
	uclog.Infof(ctx, "Provisioning region-specific tenant URLs for %v (%s)", tenant.ID, tenant.Name)
	for {
		urls, respFields, err := companyConfigStorage.ListTenantURLsPaginated(ctx, *pager)
		if err != nil {
			return ucerr.Wrap(err)
		}

		for _, region := range regions {
			regionalTenantURL := tenant.GetRegionalURL(region, false)
			if err := provisionTenantURL(ctx, companyConfigStorage, tenant.ID, urls, regionalTenantURL); err != nil {
				return ucerr.Wrap(err)
			}
			if addEKSURLs {
				regionalTenantEKSURL := tenant.GetRegionalURL(region, true)
				if err := provisionTenantURL(ctx, companyConfigStorage, tenant.ID, urls, regionalTenantEKSURL); err != nil {
					return ucerr.Wrap(err)
				}
			}
		}
		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}
	return nil
}

func provisionTenantURL(ctx context.Context, companyConfigStorage *companyconfig.Storage, tenantID uuid.UUID, existingTenantURLs []companyconfig.TenantURL, tenantURL string) error {
	tu := &companyconfig.TenantURL{
		BaseModel: ucdb.NewBase(),
		TenantID:  tenantID,
		TenantURL: tenantURL,
		Validated: true,
		System:    true,
	}

	foundURL := false
	for _, url := range existingTenantURLs {
		if url.TenantURL == tu.TenantURL {
			foundURL = true
			break
		}
	}

	if !foundURL {
		uclog.Infof(ctx, "Saving tenant URL: %v for %v", tu.TenantURL, tu.TenantID)
		if err := companyConfigStorage.SaveTenantURL(ctx, tu); err != nil {
			return ucerr.Wrap(err)
		}
	} else {
		uclog.Debugf(ctx, "Skipping saving tenant URL %v for %v as it already exists", tu.TenantURL, tu.TenantID)
	}
	return nil
}

// CleanupRegionalTenantURLs removes tenant URLs with invalid regions and EKS specific URLs if not using EKS (i.e, usingEKSURLs is false)
func CleanupRegionalTenantURLs(ctx context.Context, companyConfigStorage *companyconfig.Storage, tenant *companyconfig.Tenant, usingEKSURLs, dryRun bool) error {
	uv := universe.Current()
	if !uv.IsCloud() {
		return ucerr.Errorf("CleanupTenantURLs is only supported in cloud universes not %v", uv)
	}
	pager, err := companyconfig.NewTenantURLPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return ucerr.Wrap(err)
	}
	var expectedSuffix string
	if uv.IsProd() {
		expectedSuffix = "userclouds.com"
	} else {
		expectedSuffix = fmt.Sprintf("%v.userclouds.com", uv)
	}

	deleteList := make([]companyconfig.TenantURL, 0)
	for {
		tenantURLs, respFields, err := companyConfigStorage.ListTenantURLsPaginated(ctx, *pager)
		if err != nil {
			return ucerr.Wrap(err)
		}

		for _, tu := range tenantURLs {
			tenantURL, err := url.Parse(tu.TenantURL)
			if err != nil {
				return ucerr.Wrap(err)
			}
			if !strings.HasSuffix(tenantURL.Host, expectedSuffix) {
				uclog.Debugf(ctx, "Skipping non-UserClouds URL: %v for %s [%v]", tenantURL, tenant.Name, tenant.ID)
				continue
			}
			hostParts := strings.Split(strings.TrimSuffix(tenantURL.Host, expectedSuffix), ".")
			if len(hostParts) != 2 {
				uclog.Debugf(ctx, "Skipping non-tenant URL: %v for %s [%v]", tenantURL, tenant.Name, tenant.ID)
			}
			var urlRegion string
			regionalTenantPart := hostParts[1]
			if strings.HasSuffix(regionalTenantPart, "-eks") {
				if !usingEKSURLs {
					deleteList = append(deleteList, tu)
					continue
				}
				urlRegion = strings.TrimPrefix(strings.TrimSuffix(regionalTenantPart, "-eks"), "tenant-")

			} else {
				urlRegion = strings.TrimPrefix(regionalTenantPart, "tenant-")
			}
			if !region.IsValid(region.MachineRegion(urlRegion), uv) {
				uclog.Warningf(ctx, "invalid region URL: %v for %s [%v]. will delete", tenantURL, tenant.Name, tenant.ID)
				deleteList = append(deleteList, tu)
			}
		}
		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}
	uclog.Infof(ctx, "Found %v tenant URLs to delete", len(deleteList))
	for _, tu := range deleteList {
		uclog.Infof(ctx, "Deleting tenant URL: %v for %v dry_run=%v", tu.TenantURL, tu.TenantID, dryRun)
		if !dryRun {
			if err := companyConfigStorage.DeleteTenantURL(ctx, tu.ID); err != nil {
				return ucerr.Wrap(err)
			}
		}
	}
	return nil
}

// Validate implements Validateable
func (t *ProvisionableTenant) Validate(ctx context.Context) error {
	tenant, err := t.companyConfigStorage.GetTenant(ctx, t.tenant.ID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if err := tenant.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	if _, err := t.companyConfigStorage.GetCompany(ctx, tenant.CompanyID); err != nil {
		return ucerr.Wrap(err)
	}

	if err := t.tdb.Validate(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	if err := t.ldb.Validate(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	if t.TenantDB == nil {
		var err error
		t.TenantDB, _, _, err = tenantdb.Connect(ctx, t.companyConfigStorage, t.tenant.ID)
		if err != nil {
			return ucerr.Wrap(err)
		}
	}

	mgr := manager.NewFromDB(t.TenantDB, t.cacheCfg)

	tp, err := mgr.GetTenantPlex(ctx, t.tenant.ID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if err := tp.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	// If we didn't call Provision or initialized from NewProvisionableTenantFromExisting first - we initialize the service provisionable
	if t.p == nil {
		p, err := NewServicesProvisioner(ctx, t.Name(), t.companyConfigStorage, t.TenantDB, &t.tdb.tenantInternal.LogConfig.LogDB, t.cacheCfg, t.tdb.company, t.tenant.ID, t.employeeIDs...)
		if err != nil {
			return ucerr.Wrap(err)
		}
		t.p = p
	}

	if err := t.p.Validate(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	// to match how we provision
	for _, region := range getRegionsForTenantURL(ctx, t.tenant) {
		tenantURL := t.tenant.GetRegionalURL(region, false)
		if _, err := t.companyConfigStorage.GetTenantURLByURL(ctx, tenantURL); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}

func getRegionsForTenantURL(ctx context.Context, tenant *companyconfig.Tenant) []region.MachineRegion {
	uv := universe.Current()
	if !uv.IsCloud() {
		uclog.Infof(ctx, "Skipping region-specific tenant URLs for %v (%s) because not in cloud universe: %v", tenant.ID, tenant.Name, uv)
		return nil
	}
	if !companyconfig.CanUseRegionWithTenantURL(tenant.TenantURL) {
		uclog.Errorf(ctx, "TODO: TenantURL [%s] doesn't contain .tenant., this probably means we implemented CNAMEs and forgot to update regional URLs", tenant.TenantURL)
		return nil
	}
	return region.MachineRegionsForUniverse(uv)
}

// Cleanup implements Provsionable
func (t *ProvisionableTenant) Cleanup(ctx context.Context) error {
	if err := t.tdb.cleanup(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	if err := t.ldb.cleanup(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	// We don't clean-up the services

	if err := t.companyConfigStorage.DeleteTenant(ctx, t.tenant.ID); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// Nuke will hard-delete resources.
func (t *ProvisionableTenant) Nuke(ctx context.Context) error {
	if err := t.tdb.nuke(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	if err := t.ldb.nuke(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	// TODO: can add NukeTenant to hard delete, but it's pretty easy to clean this up with 1 line of SQL,
	// unlike all the myriad DB users and per-tenant DBs...
	if err := t.companyConfigStorage.DeleteTenant(ctx, t.tenant.ID); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// CreatePlexConfigOption is an option for CreatePlexConfig
type CreatePlexConfigOption interface {
	apply(*cpcOptions)
}

type cpcOptions struct {
	keys *tenantplex.Keys
}

type cpcOptFunc func(*cpcOptions)

func (c cpcOptFunc) apply(opts *cpcOptions) {
	c(opts)
}

// UseKeys lets you specify keys for CreatePlexConfig, rather than generating them
func UseKeys(keys *tenantplex.Keys) CreatePlexConfigOption {
	return cpcOptFunc(func(o *cpcOptions) {
		o.keys = keys
	})
}

// CreatePlexConfig creates a default plex map for new tenants
func CreatePlexConfig(ctx context.Context, tenant *companyconfig.Tenant, opts ...CreatePlexConfigOption) (*serviceconfig.ServiceConfig, *tenantplex.TenantConfig, error) {
	var os cpcOptions
	for _, opt := range opts {
		opt.apply(&os)
	}

	var keys *tenantplex.Keys
	if os.keys != nil {
		keys = os.keys
	} else {
		var err error
		keys, err = provisioning.GeneratePlexKeys(ctx, tenant.ID)
		if err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	tcb := builder.NewProdTenantConfigBuilder().
		SetKeys(*keys).
		ResetTelephonyProvider()

	tcb.AddProvider().
		MakeActive().
		SetName(fmt.Sprintf("%s UserClouds IDP Provider", tenant.Name)).
		MakeUC().
		SetIDPURL(tenant.TenantURL).
		AddUCAppWithName(fmt.Sprintf("%s UserClouds IDP App", tenant.Name))

	tcb.AddApp().
		SetName(fmt.Sprintf("%s Default App", tenant.Name)).
		SetOrganizationID(tenant.CompanyID)

	tcb.AddEmployeeProvider().
		SetName("Employee IDP Provider").
		SetAppName("Employee IDP App")

	tcb.AddEmployeeApp().
		SetName("Employee Plex App").
		SetOrganizationID(tenant.CompanyID)

	pc := &serviceconfig.ServiceConfig{}
	tc := tcb.Build()

	return pc, &tc, nil
}

// Close cleans up resources that maybe used by a ProvisionableTenant
func (t *ProvisionableTenant) Close(ctx context.Context) error {
	if t.p != nil {
		if err := t.p.Close(ctx); err != nil {
			return ucerr.Wrap(err)
		}
	}

	if t.TenantDB != nil {
		if err := t.TenantDB.Close(ctx); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}
