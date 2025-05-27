package tenant

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/companyconfig"
	tenantProvisioning "userclouds.com/internal/provisioning/tenant"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/manager"
)

// CreateTenant creates a new tenant and provisions it.
func CreateTenant(ctx context.Context,
	storage *companyconfig.Storage,
	tm *tenantmap.StateMap,
	ccDBCfg, lDBCfg *ucdb.Config,
	tenant companyconfig.Tenant,
	userID uuid.UUID,
	consoleAuthzClient *authz.Client,
	consoleTenantInfo companyconfig.TenantInfo,
	cacheCfg *cache.Config) error {

	if err := createTenant(ctx, storage, tm, ccDBCfg, lDBCfg, tenant, userID, consoleAuthzClient, consoleTenantInfo, cacheCfg); err != nil {
		uclog.Errorf(ctx, "failed to create tenant: %v", err)
		// mark the tenant as failed
		if ten, err := storage.GetTenant(ctx, tenant.ID); err == nil {
			ten.State = companyconfig.TenantStateFailedToProvision
			if err := storage.SaveTenant(ctx, ten); err != nil {
				return ucerr.Errorf("failed to save tenant after failing to create: %w", err)
			}
		}
		return ucerr.Errorf("failed to create tenant: %w", err)
	}
	return nil
}

func createTenant(ctx context.Context,
	storage *companyconfig.Storage,
	tm *tenantmap.StateMap,
	ccDBCfg, lDBCfg *ucdb.Config,
	tenant companyconfig.Tenant,
	userID uuid.UUID,
	consoleAuthzClient *authz.Client,
	consoleTenantInfo companyconfig.TenantInfo,
	cacheCfg *cache.Config) error {

	// mark this as being-created
	tenant.State = companyconfig.TenantStateCreating

	_, tc, err := tenantProvisioning.CreatePlexConfig(ctx, &tenant)
	if err != nil {
		return ucerr.Wrap(err)
	}
	tenantPlex := &tenantplex.TenantPlex{
		VersionBaseModel: ucdb.NewVersionBaseWithID(tenant.ID),
		PlexConfig:       *tc,
	}

	pt, err := tenantProvisioning.NewProvisionableTenant(ctx, "API Create Tenant", &tenant, tenantPlex, storage, ccDBCfg, ccDBCfg, lDBCfg, cacheCfg, []uuid.UUID{userID})
	if err != nil {
		return ucerr.Wrap(err)
	}

	if err := pt.Provision(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	// Get the employees of the company
	consoleRBACClient := authz.NewRBACClient(consoleAuthzClient)
	companyGroup, err := consoleRBACClient.GetGroup(ctx, tenant.CompanyID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	companyMembers, err := companyGroup.GetMemberships(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}
	companyAdminMap := map[uuid.UUID]bool{}
	for _, member := range companyMembers {
		if member.Role == ucauthz.AdminRole {
			companyAdminMap[member.User.ID] = true
		}
	}

	// TODO (sgarrity 6/23): we are taking advantage of the fact that authz trusts
	// console with god-mode to make this work, since we no longer have the user's
	// tokens in context after we went async ... need a better soln
	// NB: if orgs are enabled, this means that we need to grab the correct org's login
	// app for these credentials to work ... ugh
	ts, err := tm.GetTenantStateForID(ctx, consoleTenantInfo.TenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	mgr := manager.NewFromDB(ts.TenantDB, ts.CacheConfig)

	las, err := mgr.GetLoginApps(ctx, consoleTenantInfo.TenantID, tenant.CompanyID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if len(las) == 0 {
		return ucerr.Errorf("no login apps")
	}
	loginApp := las[0]

	cs, err := loginApp.ClientSecret.Resolve(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}
	loginAppTokenSource, err := jsonclient.ClientCredentialsForURL(consoleTenantInfo.TenantURL, loginApp.ClientID, cs, []string{tenant.TenantURL})
	if err != nil {
		return ucerr.Wrap(err)
	}
	tenantAuthzClient, err := apiclient.NewAuthzClientWithTokenSource(ctx, storage, consoleTenantInfo.TenantID, tenant.TenantURL, loginAppTokenSource)
	if err != nil {
		return ucerr.Wrap(err)
	}
	tenantRBACClient := authz.NewRBACClient(tenantAuthzClient)
	tenantGroup, err := tenantRBACClient.GetGroup(ctx, tenant.CompanyID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// Add all company admins to the tenant
	for adminID := range companyAdminMap {
		if _, err := tenantAuthzClient.CreateObject(ctx, adminID, authz.UserObjectTypeID, "", authz.OrganizationID(tenant.CompanyID), authz.IfNotExists()); err != nil {
			return ucerr.Wrap(err)
		}
		user, err := tenantRBACClient.GetUser(ctx, adminID)
		if err != nil {
			return ucerr.Wrap(err)
		}

		if _, err := tenantGroup.AddUserRole(ctx, *user, ucauthz.AdminRole); err != nil {
			return ucerr.Wrap(err)
		}
	}

	// TODO: plex's tenantconfig cache will timeout and update soon

	uclog.Debugf(ctx, "finished setting up console authz for tenant %s", tenant.ID)

	// mark the tenant as active
	ten, err := storage.GetTenant(ctx, tenant.ID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ten.State = companyconfig.TenantStateActive
	if err := storage.SaveTenant(ctx, ten); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}
