package provisioning

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/internal"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/provisioning/types"
	"userclouds.com/plex/manager"
)

// NewOrganizationProvisioner returns a Provisionable for provisioning an organization
func NewOrganizationProvisioner(name string, pi types.ProvisionInfo, orgID uuid.UUID, orgName string, region region.DataRegion) (types.Provisionable, error) {
	provs := make([]types.Provisionable, 0)
	name = fmt.Sprintf("%s:Org[%v, %s]", name, orgID, orgName)

	// Validate the inputs
	if orgID.IsNil() {
		return nil, ucerr.Errorf("Can't provision an org with nil orgID")
	}
	if orgName == "" {
		return nil, ucerr.Errorf("Can't provision an org with empty name")
	}
	if pi.TenantDB == nil {
		return nil, ucerr.Errorf("Can't provision an org with nil tenantDB")
	}
	// Create the row for the organization
	o := NewOrganizationObjectProvisioner(name, pi, orgID, orgName, region)
	provs = append(provs, o)

	// Create the object representing the organization in the AuthZ system
	alias := generateOrgGroupObjectAlias(orgName, orgID)
	objProv := NewEntityAuthZ(
		name,
		pi,
		nil,
		nil,
		[]authz.Object{
			{BaseModel: ucdb.NewBaseWithID(orgID), Alias: &alias, TypeID: authz.GroupObjectTypeID, OrganizationID: orgID},
		},
		nil,
		types.Validate,
	)
	provs = append(provs, objProv)

	p := types.NewParallelProvisioner(provs, name)
	return p, nil
}

// NewOrganizationObjectProvisioner returns a Provisionable for provisioning an organization
func NewOrganizationObjectProvisioner(name string,
	pi types.ProvisionInfo,
	orgID uuid.UUID,
	orgName string,
	region region.DataRegion) types.Provisionable {
	o := OrganizationProvisioner{
		Named:          types.NewNamed(name + ":OrgObj"),
		Parallelizable: types.NewParallelizable(),
		ProvisionInfo:  pi,
		orgName:        orgName,
		orgID:          orgID,
		region:         region,
	}
	return &o
}

// OrganizationProvisioner is a Provisionable object used to set up an organization
type OrganizationProvisioner struct {
	types.Named
	types.NoopClose
	types.Parallelizable
	types.ProvisionInfo
	orgID   uuid.UUID
	orgName string
	region  region.DataRegion
}

func generateOrgGroupObjectAlias(name string, id uuid.UUID) string {
	// NOTE: org names can collide with other group name so alias must include UUID
	return fmt.Sprintf("%s (%s)", name, id)
}

// Provision implements Provisionable and creates organization
func (o *OrganizationProvisioner) Provision(ctx context.Context) error {
	s := internal.NewStorage(ctx, o.TenantID, o.TenantDB, o.CacheCfg)
	orgIn, err := s.GetOrganization(ctx, o.orgID)
	// failure other than not-yet-provisioned
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}

	// Add new row if the org doesn't exist or update the org if the name has changed
	if err != nil || orgIn.Name != o.orgName {
		org := &authz.Organization{
			BaseModel: ucdb.NewBaseWithID(o.orgID),
			Name:      o.orgName,
			Region:    o.region,
		}
		if err := s.SaveOrganization(ctx, org); err != nil {
			return ucerr.Wrap(err)
		}
	}

	// create the login app for this organization
	// this logic is duplicated from authz's createorg handler since there we use service clients,
	// and here we don't know if the service is running

	// make provisioning idempotent by not adding a new login app if one already exists
	companyConfigManager := manager.NewFromDB(o.TenantDB, o.CacheCfg)
	apps, err := companyConfigManager.GetLoginApps(ctx, o.TenantID, o.orgID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if len(apps) == 1 {
		return nil
	}

	app, err := manager.NewLoginAppForOrganization(ctx, o.TenantID, o.orgName, o.orgID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if err := companyConfigManager.AddLoginApp(ctx, o.TenantID, nil, *app); err != nil {
		return ucerr.Wrap(err)
	}

	// since we can't rely on authz being up, we used a nil authz client above, so create the object for app directly
	appAlias := manager.LoginAppAlias(app.ID)
	obj := &authz.Object{
		BaseModel:      ucdb.NewBaseWithID(app.ID),
		TypeID:         authz.LoginAppObjectTypeID,
		Alias:          &appAlias,
		OrganizationID: app.OrganizationID,
	}
	if err := s.SaveObject(ctx, obj); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// Validate implements Provisionable and validates that the organization was created
func (o *OrganizationProvisioner) Validate(ctx context.Context) error {
	s := internal.NewStorage(ctx, o.TenantID, o.TenantDB, o.CacheCfg)
	if _, err := s.GetOrganization(ctx, o.orgID); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// Cleanup implements Provisionable and removes the organization
func (o *OrganizationProvisioner) Cleanup(ctx context.Context) error {
	s := internal.NewStorage(ctx, o.TenantID, o.TenantDB, o.CacheCfg)
	if err := s.DeleteOrganization(ctx, o.orgID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}

	return nil
}
