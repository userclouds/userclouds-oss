package auth

import (
	"context"
	"errors"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	"userclouds.com/console/internal/tenantcache"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/plex/manager"
)

// AddEmployeeRoleToCompany adds an employee (in the console tenant) to a company with a given role, and also adds the employee
// to all of the company's tenant user stores
func AddEmployeeRoleToCompany(ctx context.Context,
	consoleTenantID uuid.UUID,
	storage *companyconfig.Storage,
	tenantCache *tenantcache.Cache,
	rbacClient *authz.RBACClient,
	employeeID uuid.UUID,
	companyID uuid.UUID,
	role string,
	tenantRoles companyconfig.TenantRoles,
	tenants []companyconfig.Tenant,
	cacheConfig *cache.Config) error {

	uclog.Debugf(ctx, "add user %s to company %s in role %s", employeeID, companyID, role)

	user, err := rbacClient.GetUser(ctx, employeeID)
	if err != nil {
		// TODO: this is probably indicative of a bad request, not internal server error, how should
		// we propagate that out?
		return ucerr.Wrap(err)
	}

	group, err := rbacClient.GetGroup(ctx, companyID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if !(role == ucauthz.MemberRole || role == ucauthz.AdminRole) {
		return ucerr.New("invalid role")
	}

	if _, err := group.AddUserRole(ctx, *user, role); err != nil {
		return ucerr.Wrap(err)
	}

	// Add this user to all the tenants for this company
	for _, tenant := range tenants {
		if role, ok := tenantRoles[tenant.ID]; ok && (role == ucauthz.MemberRole || role == ucauthz.AdminRole) {
			tenantDB, err := tenantCache.GetTenantDB(ctx, tenant.ID)
			if err != nil {
				return ucerr.Wrap(err)
			}

			mgr := manager.NewFromDB(tenantDB, cacheConfig)
			if err := AddEmployeeRoleToTenant(ctx, consoleTenantID, mgr, storage, tenant, employeeID, role); err != nil {
				return ucerr.Wrap(err)
			}
		}
	}

	return nil
}

// AddEmployeeRoleToTenant adds an employee to a tenant's IDP
// Caller needs to ensure that employeeID is a valid user in the console tenant
func AddEmployeeRoleToTenant(ctx context.Context, consoleTenantID uuid.UUID, manager *manager.Manager, storage *companyconfig.Storage, tenant companyconfig.Tenant,
	employeeID uuid.UUID, role string) error {
	if tenant.ID == consoleTenantID {
		return ucerr.New("cannot add employee to console tenant")
	}

	var ts jsonclient.Option
	_, refreshToken, err := GetAccessTokens(ctx)
	if err != nil {
		// If there is no logged in user (which happens in case of e.g. invite flow), use M2M token
		ts, err = m2m.GetM2MTokenSource(ctx, tenant.ID)
		if err != nil {
			return ucerr.Wrap(err)
		}
	} else {
		ts, err = NewEmployeeTokenSource(ctx, &tenant, refreshToken)
		if err != nil {
			return ucerr.Wrap(err)
		}
	}

	authzClient, err := apiclient.NewAuthzClientWithTokenSource(ctx, storage, tenant.ID, tenant.TenantURL, ts)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// Add the specified role to the user
	tenantRBACClient := authz.NewRBACClient(authzClient)
	companyGroup, err := tenantRBACClient.GetGroup(ctx, tenant.CompanyID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	_, err = authzClient.CreateObject(ctx, employeeID, authz.UserObjectTypeID, "", authz.OrganizationID(tenant.CompanyID))
	if err != nil {
		return ucerr.Wrap(err)
	}
	user, err := tenantRBACClient.GetUser(ctx, employeeID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if _, err := companyGroup.AddUserRole(ctx, *user, role); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// DeleteEmployeeFromCompany deletes an employee (in the console tenant) from a company, and also deletes the employee from all of the company's tenants
func DeleteEmployeeFromCompany(ctx context.Context,
	storage *companyconfig.Storage,
	tenantCache *tenantcache.Cache,
	rbacClient *authz.RBACClient,
	employeeID uuid.UUID,
	companyID uuid.UUID,
	tenants []companyconfig.Tenant,
	cacheConfig *cache.Config) error {

	uclog.Debugf(ctx, "delete user %s from company %s", employeeID, companyID)

	user, err := rbacClient.GetUser(ctx, employeeID)
	if err != nil {
		// TODO: this is probably indicative of a bad request, not internal server error, how should
		// we propagate that out?
		return ucerr.Wrap(err)
	}

	group, err := rbacClient.GetGroup(ctx, companyID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if err := group.RemoveUser(ctx, *user); err != nil {
		return ucerr.Wrap(err)
	}

	// Delete this user from all the tenants for this company
	for _, tenant := range tenants {
		tenantDB, err := tenantCache.GetTenantDB(ctx, tenant.ID)
		if err != nil {
			// Employee missing from one tenant should not prevent deletion from other tenants
			uclog.Errorf(ctx, "error deleting user %s from tenant %s: %v", employeeID, tenant.TenantURL, err)
			continue
		}

		mgr := manager.NewFromDB(tenantDB, cacheConfig)
		if err := DeleteEmployeeFromTenant(ctx, mgr, tenant, employeeID, storage); err != nil {
			// Employee missing from one tenant should not prevent deletion from other tenants
			uclog.Errorf(ctx, "error deleting user %s from tenant %s: %v", employeeID, tenant.TenantURL, err)
		}
	}

	return nil
}

// DeleteEmployeeFromTenant deletes an employee from a specific tenant
func DeleteEmployeeFromTenant(ctx context.Context, manager *manager.Manager, tenant companyconfig.Tenant, employeeID uuid.UUID, storage *companyconfig.Storage) error {

	_, refreshToken := MustGetAccessTokens(ctx)
	ts, err := NewEmployeeTokenSource(ctx, &tenant, refreshToken)
	if err != nil {
		return ucerr.Wrap(err)
	}

	authzClient, err := apiclient.NewAuthzClientWithTokenSource(ctx, storage, tenant.ID, tenant.TenantURL, ts)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if err := authzClient.DeleteObject(ctx, employeeID); err != nil {
		// If the object doesn't exist (404), that's ok
		if !errors.Is(err, authz.ErrObjectNotFound) {
			return ucerr.Wrap(err)
		}
	}

	return nil
}
