package provisioning

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/provisioning"
	"userclouds.com/authz/ucauthz"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/provisioning/types"
	"userclouds.com/plex/manager"
)

// CompanyOption defines a way to modify ProvisioningCompany's behavior.
type CompanyOption interface {
	apply(*CompanyObjectProvisioner)
}

type cOptFunc func(*CompanyObjectProvisioner)

func (o cOptFunc) apply(po *CompanyObjectProvisioner) {
	o(po)
}

// Owner allows specification of the username of an owning user.
func Owner(userID uuid.UUID) CompanyOption {
	return cOptFunc(func(po *CompanyObjectProvisioner) {
		po.ownerUserID = userID
	})
}

// CompanyObjectProvisioner provisions, validates, and de-provisions companies.
type CompanyObjectProvisioner struct {
	types.Named
	types.NoopClose
	types.Parallelizable
	types.ProvisionInfo
	company          *companyconfig.Company
	ownerUserID      uuid.UUID
	consoleCompanyID uuid.UUID
}

// NewProvisionableCompany creates a new object which can provision, validate, and clean up companies.
func NewProvisionableCompany(ctx context.Context,
	name string, // TODO (sgarrity 6/23): clean up these names-that-aren't-object-name provisioning params
	pi types.ProvisionInfo,
	company *companyconfig.Company,
	consoleCompanyID uuid.UUID,
	opts ...CompanyOption) (types.Provisionable, error) {

	provs := make([]types.Provisionable, 0)
	name = fmt.Sprintf("%s:Company(%s)", name, company.Name)

	// Create the row for the company
	po := &CompanyObjectProvisioner{
		Named:            types.NewNamed(name),
		Parallelizable:   types.NewParallelizable(),
		ProvisionInfo:    pi,
		company:          company,
		consoleCompanyID: consoleCompanyID,
	}

	for _, v := range opts {
		v.apply(po)
	}

	// Create the corresponding organization
	// TODO seems wrong to have this test option in mainline code, same as Provision
	if pi.TenantDB != nil {
		o, err := provisioning.NewOrganizationProvisioner(name, pi, company.ID, company.Name, "")
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		provs = append(provs, o)

		if !po.ownerUserID.IsNil() {
			adminProv := provisioning.NewEntityAuthZ(
				name,
				pi,
				nil,
				nil,
				nil,
				[]authz.Edge{
					{BaseModel: ucdb.NewBase(), EdgeTypeID: ucauthz.AdminEdgeTypeID, SourceObjectID: po.ownerUserID, TargetObjectID: company.ID},
				},
				types.Validate,
			)
			provs = append(provs, adminProv)
		}
	}

	// TODO (sgarrity 6/23): this is a bit janky until we improve the provisioning framework
	// the company needs to be provisioned last (but leaving this parallel stuff in since validate can be parallel)
	// because it requires the login app to have been created etc
	provs = append(provs, po)

	p := types.NewRestrictedParallelProvisioner(provs, name, types.Validate)
	return p, nil
}

// Provision implements Provisionable
func (po *CompanyObjectProvisioner) Provision(ctx context.Context) error {
	uclog.Infof(ctx, "Start provisioning company '%s' (id: %s)...", po.company.Name, po.company.ID)
	if err := po.CompanyStorage.SaveCompany(ctx, po.company); err != nil {
		uclog.Errorf(ctx, "Failed to save company '%s' (id: %s): %v", po.company.Name, po.company.ID, err)
		return ucerr.Wrap(err)
	}

	// TODO: seems wrong to have this test option in mainline code, same as NewProvisionable Company
	// we check consoleTenantDB here as well because on initial setup (eg. `make devsetup`), we have a
	// known consoleTenantID, but not a DB created yet
	if !po.TenantID.IsNil() && po.TenantDB != nil {
		mgr := manager.NewFromDB(po.TenantDB, po.CacheCfg)
		loginApps, err := mgr.GetLoginApps(ctx, po.TenantID, po.company.ID)
		if err != nil {
			return ucerr.Wrap(err)
		}
		if len(loginApps) != 1 {
			return ucerr.Errorf("expected 1 login app for new company, got %d", len(loginApps))
		}

		consoleLoginApps, err := mgr.GetLoginApps(ctx, po.TenantID, po.consoleCompanyID)
		if err != nil {
			return ucerr.Wrap(err)
		}
		if len(consoleLoginApps) != 1 {
			return ucerr.Errorf("expected 1 login app for console, got %d", len(consoleLoginApps))
		}

		// copy the parameters from console's default app to this new app
		ucApp := consoleLoginApps[0]
		app := loginApps[0]
		manager.CopyUCAppSettings(&ucApp, &app)
		if err := mgr.UpdateLoginApp(ctx, po.TenantID, app); err != nil {
			return ucerr.Wrap(err)
		}
	}
	uclog.Infof(ctx, "Successfully provisioned company '%s' (id: %s)!", po.company.Name, po.company.ID)
	return nil
}

// Validate implements Provisionable.
func (po *CompanyObjectProvisioner) Validate(ctx context.Context) error {
	if _, err := po.CompanyStorage.GetCompany(ctx, po.company.ID); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// Cleanup implements Provisionable.
func (po *CompanyObjectProvisioner) Cleanup(ctx context.Context) error {
	// This will only work if all tenants have already been deleted
	if err := po.CompanyStorage.DeleteCompany(ctx, po.company.ID); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
