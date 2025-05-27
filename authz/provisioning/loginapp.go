package provisioning

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/provisioning/types"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/manager"
)

// NewLoginAppProvisioner returns a Provisionable for provisioning an LoginApp
func NewLoginAppProvisioner(name string, pi types.ProvisionInfo, orgID uuid.UUID, orgName string, apps []tenantplex.App, ucApp tenantplex.App) types.Provisionable {
	o := LoginAppProvisioner{
		Named:          types.NewNamed(fmt.Sprintf("%s:LoginApp[%v]", name, orgID)),
		Parallelizable: types.NewParallelizable(),
		ProvisionInfo:  pi,
		orgName:        orgName,
		orgID:          orgID,
		ucApp:          ucApp,
		apps:           apps,
	}
	return &o
}

// LoginAppProvisioner is a Provisionable object used to set up an LoginApp
type LoginAppProvisioner struct {
	types.Named
	types.NoopClose
	types.Parallelizable
	types.ProvisionInfo
	orgID   uuid.UUID
	orgName string
	ucApp   tenantplex.App
	apps    []tenantplex.App
}

// Provision implements Provisionable and creates LoginApp
func (o *LoginAppProvisioner) Provision(ctx context.Context) error {
	mgr, err := manager.NewFromCompanyConfig(ctx, o.CompanyStorage, o.TenantID, o.CacheCfg)
	if err != nil {
		return ucerr.Wrap(err)
	}
	defer mgr.Close(ctx)

	switch len(o.apps) {
	case 0:
		// add a new app for this company
		app, err := manager.NewLoginAppForOrganization(ctx, o.TenantID, o.orgName, o.orgID)
		if err != nil {
			return ucerr.Wrap(err)
		}

		manager.CopyUCAppSettings(&o.ucApp, app)

		if err := mgr.AddLoginApp(ctx, o.TenantID, nil, *app); err != nil {
			return ucerr.Wrap(err)
		}
	case 1:
		// update the app for this company
		app := o.apps[0]
		manager.CopyUCAppSettings(&o.ucApp, &app)
		if err := mgr.UpdateLoginApp(ctx, o.TenantID, app); err != nil {
			return ucerr.Wrap(err)
		}
	default:
		for _, app := range o.apps[1:] {
			// remove all but the first app for this company
			if err := mgr.DeleteLoginApp(ctx, o.TenantID, nil, app.ID); err != nil {
				return ucerr.Wrap(err)
			}
		}
	}

	return nil
}

// Validate implements Provisionable and validates that the LoginApp was created
func (o *LoginAppProvisioner) Validate(ctx context.Context) error {
	mgr, err := manager.NewFromCompanyConfig(ctx, o.CompanyStorage, o.TenantID, o.CacheCfg)
	if err != nil {
		return ucerr.Wrap(err)
	}
	defer mgr.Close(ctx)

	apps, err := mgr.GetLoginApps(ctx, o.TenantID, o.orgID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// TODO this may prove fragile if the user deletes a login app
	if len(apps) != 1 {
		uclog.Errorf(ctx, "Expect to have exactly 1 login app per org. Instead got %v", apps)
	}

	// We don't validate the contents as user may have changed them

	return nil
}

// Cleanup implements Provisionable and removes the LoginApp
func (o *LoginAppProvisioner) Cleanup(ctx context.Context) error {
	// TODO I think we should delete the LoginApps created for each org, but only if the user made no modifications to them
	return nil
}
