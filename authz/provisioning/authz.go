package provisioning

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	"userclouds.com/console/tenantinfo"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/provisioning/types"
	"userclouds.com/plex/manager"
)

// ProvisionCompanyOrgs returns a ProvisionableMaker that can provision company orgs
func ProvisionCompanyOrgs(ctx context.Context, name string, pi types.ProvisionInfo, company *companyconfig.Company) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if !types.DeepProvisioning {
			return nil, nil
		}

		cti, err := tenantinfo.GetConsoleTenantInfo(ctx, pi.CompanyStorage)
		if err != nil {
			return nil, nil
		}

		if company == nil {
			return nil, ucerr.New("cannot provision company orgs with a nil company")
		}

		if err := company.Validate(); err != nil {
			return nil, ucerr.Wrap(err)
		}

		if cti.CompanyID != company.ID {
			return nil, nil
		}

		if pi.CompanyStorage == nil {
			return nil, ucerr.New("cannot provision company orgs with a nil CompanyStorage")
		}

		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot provision company orgs with a nil tenantDB")
		}

		p, err := migrateAddOrganizationsForEachCompany(ctx, name, pi)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		return []types.Provisionable{p}, nil
	}
}

// ProvisionDefaultOrg returns a ProvisionableMaker that can provision the default org
func ProvisionDefaultOrg(name string, pi types.ProvisionInfo, company *companyconfig.Company) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if pi.CompanyStorage == nil {
			return nil, ucerr.New("cannot provision default organization with nil CompanyStorage")
		}

		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot provision default organization with nil tenantDB")
		}

		if company == nil {
			return nil, ucerr.New("cannot provision default organization with nil company")
		}

		if err := company.Validate(); err != nil {
			return nil, ucerr.Wrap(err)
		}

		if pi.TenantID.IsNil() {
			return nil, ucerr.New("cannot provision default organization with nil tenantID")
		}

		p, err := NewOrganizationProvisioner(
			name+":AuthZDefaultOrg",
			pi,
			company.ID, // default company org has same ID as the company
			company.Name,
			"",
		)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		return []types.Provisionable{p}, nil
	}
}

// ProvisionDefaultTypes returns a ProvisionableMaker that can provision default authz types
func ProvisionDefaultTypes(name string, pi types.ProvisionInfo) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot provision default authz types with nil tenantDB")
		}

		return []types.Provisionable{
				NewEntityAuthZ(
					name+":AuthZDefaultTypes",
					pi,
					authz.DefaultAuthZObjectTypes,
					append(authz.DefaultAuthZEdgeTypes, ConsoleAuthZEdgeTypes...),
					nil,
					nil,
					types.Validate,
				),
			},
			nil
	}
}

// ProvisionLoginApp returns a ProvisionableMaker that can provision a login app
func ProvisionLoginApp(ctx context.Context, name string, pi types.ProvisionInfo) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		cti, err := tenantinfo.GetConsoleTenantInfo(ctx, pi.CompanyStorage)
		if err != nil || pi.TenantID != cti.TenantID {
			// if GetFromServiceConfig fails, which can occur during provisioning tests, or
			// this is not the console tenant ID, do nothing
			return nil, nil
		}

		if pi.CompanyStorage == nil {
			return nil, ucerr.New("cannot provision login app with nil CompanyStorage")
		}

		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot provision login app with nil tenantDB")
		}

		p, err := migrateEnsureLoginAppForEachCompany(ctx, name, pi, cti.CompanyID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		return []types.Provisionable{p}, nil
	}
}

// ProvisionLoginAppObjects returns a ProvisionableMaker that can provision login app authz objects
func ProvisionLoginAppObjects(
	ctx context.Context,
	name string,
	pi types.ProvisionInfo,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot provision login app authz objects with nil tenantDB")
		}

		mgr := manager.NewFromDB(pi.TenantDB, pi.CacheCfg)
		tenantPlex, err := mgr.GetTenantPlex(ctx, pi.TenantID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}

			return nil, ucerr.Wrap(err)
		}

		var authzObjects []authz.Object
		for _, app := range tenantPlex.PlexConfig.PlexMap.Apps {
			alias := manager.LoginAppAlias(app.ID)
			authzObjects = append(
				authzObjects,
				authz.Object{
					BaseModel:      ucdb.NewBaseWithID(app.ID),
					Alias:          &alias,
					TypeID:         authz.LoginAppObjectTypeID,
					OrganizationID: app.OrganizationID,
				},
			)
		}

		if tenantPlex.PlexConfig.PlexMap.EmployeeApp != nil {
			alias := manager.LoginAppAlias(tenantPlex.PlexConfig.PlexMap.EmployeeApp.ID)
			authzObjects = append(
				authzObjects,
				authz.Object{
					BaseModel:      ucdb.NewBaseWithID(tenantPlex.PlexConfig.PlexMap.EmployeeApp.ID),
					Alias:          &alias,
					TypeID:         authz.LoginAppObjectTypeID,
					OrganizationID: tenantPlex.PlexConfig.PlexMap.EmployeeApp.OrganizationID,
				},
			)
		}

		return []types.Provisionable{
				NewEntityAuthZ(
					name,
					pi,
					nil,
					nil,
					authzObjects,
					nil,
					types.Provision, types.Validate,
				),
			},
			nil
	}
}

// ProvisionEmployeeEntities returns a ProvisionableMaker that can provision employee authz entities
func ProvisionEmployeeEntities(
	name string,
	pi types.ProvisionInfo,
	company *companyconfig.Company,
	employeeIDs ...uuid.UUID,
) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if len(employeeIDs) == 0 {
			return nil, nil
		}

		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot provision employee authz entities with nil tenantDB")
		}

		if company == nil {
			return nil, ucerr.New("cannot provision employee authz entities with nil company")
		}

		if err := company.Validate(); err != nil {
			return nil, ucerr.Wrap(err)
		}

		var employeeObjects []authz.Object
		var employeeEdges []authz.Edge
		for _, employeeID := range employeeIDs {
			employeeObjects = append(
				employeeObjects,
				authz.Object{
					BaseModel:      ucdb.NewBaseWithID(employeeID),
					TypeID:         authz.UserObjectTypeID,
					OrganizationID: company.ID,
				},
			)
			employeeEdges = append(
				employeeEdges,
				authz.Edge{
					BaseModel:      ucdb.NewBase(),
					SourceObjectID: employeeID,
					TargetObjectID: company.ID,
					EdgeTypeID:     ucauthz.AdminEdgeTypeID,
				},
			)
		}

		return []types.Provisionable{
				NewEntityAuthZ(
					name+":EmployeesAuthZ",
					pi,
					nil,
					nil,
					employeeObjects,
					employeeEdges,
					types.Provision, types.Validate,
				),
			},
			nil
	}
}

func migrateAddOrganizationsForEachCompany(ctx context.Context, name string, pi types.ProvisionInfo) (types.Provisionable, error) {
	name = name + ":MigrateAddOrganizationsForEachCompany"
	provs := make([]types.Provisionable, 0)

	if universe.Current().IsTestOrCI() {
		return types.NewParallelProvisioner(provs, name), nil
	}

	pager, err := companyconfig.NewCompanyPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	for {
		companies, respFields, err := pi.CompanyStorage.ListCompaniesPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		for _, company := range companies {
			op, err := NewOrganizationProvisioner(name, pi, company.ID, company.Name, "")
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			provs = append(provs, op)
		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	p := types.NewParallelProvisioner(provs, name)
	return p, nil
}

func migrateEnsureLoginAppForEachCompany(ctx context.Context, name string, pi types.ProvisionInfo, ucCompanyID uuid.UUID) (types.Provisionable, error) {
	name = name + ":MigrateEnsureLoginAppForEachCompany"
	provs := make([]types.Provisionable, 0)

	pager, err := companyconfig.NewCompanyPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	mgr := manager.NewFromDB(pi.TenantDB, pi.CacheCfg)
	ucApps, err := mgr.GetLoginApps(ctx, pi.TenantID, ucCompanyID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	if len(ucApps) != 1 {
		return nil, ucerr.Errorf("expected 1 login app for company %s, got %d", ucCompanyID, len(ucApps))
	}
	ucApp := ucApps[0]

	for {
		companies, respFields, err := pi.CompanyStorage.ListCompaniesPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		for _, company := range companies {

			apps, err := mgr.GetLoginApps(ctx, pi.TenantID, company.ID)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}

			uclog.Debugf(ctx, "migrateEnsureLoginAppForEachCompany: Got %v apps for company %v", apps, company.Name)
			la := NewLoginAppProvisioner(name, pi, company.ID, company.Name, apps, ucApp)
			provs = append(provs, la)

		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	p := types.NewParallelProvisioner(provs, name)
	return p, nil
}
