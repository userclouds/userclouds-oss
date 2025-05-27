package manager

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/tenantplex"
)

// AddLoginApp adds a new app to the tenant's plex config.
func (m *Manager) AddLoginApp(ctx context.Context, tenantID uuid.UUID, azc *authz.Client, app tenantplex.App) error {
	tp, err := m.GetTenantPlex(ctx, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	for _, a := range tp.PlexConfig.PlexMap.Apps {
		if app.ID == a.ID {
			return ucerr.Friendlyf(nil, "ID '%v' is already associated with an existing app", app.ID)
		}

		if app.ClientID == a.ClientID {
			return ucerr.Friendlyf(nil, "ClientID '%v' is already associated with an existing app", app.ClientID)
		}
	}

	if (app.TokenValidity == tenantplex.TokenValidity{}) {
		app.TokenValidity = tenantplex.TokenValidity{
			Access:          ucjwt.DefaultValidityAccess,
			Refresh:         ucjwt.DefaultValidityRefresh,
			ImpersonateUser: ucjwt.DefaultValidityImpersonateUser,
		}
	}
	tp.PlexConfig.PlexMap.Apps = append(tp.PlexConfig.PlexMap.Apps, app)

	if err := m.SaveTenantPlex(ctx, tp); err != nil {
		return ucerr.Wrap(err)
	}

	if azc != nil { // this should only be nil in tests and provisioning
		appAlias := LoginAppAlias(app.ID)
		if _, err := azc.CreateObject(ctx, app.ID, authz.LoginAppObjectTypeID, appAlias, authz.OrganizationID(app.OrganizationID)); err != nil {
			uclog.Errorf(ctx, "error creating authz object for app %v: %v", app.ID, err)
		}
	}

	return nil
}

// DeleteLoginApp deletes an app from the tenant's plex config.
func (m *Manager) DeleteLoginApp(ctx context.Context, tenantID uuid.UUID, azc *authz.Client, appID uuid.UUID) error {
	tp, err := m.GetTenantPlex(ctx, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	newApps := []tenantplex.App{}
	for _, a := range tp.PlexConfig.PlexMap.Apps {
		if appID == a.ID {
			continue
		}
		newApps = append(newApps, a)
	}

	tp.PlexConfig.PlexMap.Apps = newApps

	if err := m.SaveTenantPlex(ctx, tp); err != nil {
		return ucerr.Wrap(err)
	}

	if azc != nil {
		if err := azc.DeleteObject(ctx, appID); err != nil {
			uclog.Errorf(ctx, "error deleting authz object for app %v: %v", appID, err)
		}
	}

	return nil
}

// UpdateLoginApp updates an app in the tenant's plex config with matching ID
func (m *Manager) UpdateLoginApp(ctx context.Context, tenant uuid.UUID, app tenantplex.App) error {
	tp, err := m.GetTenantPlex(ctx, tenant)
	if err != nil {
		return ucerr.Wrap(err)
	}

	found := false
	different := false
	for i, a := range tp.PlexConfig.PlexMap.Apps {
		if a.ID == app.ID {
			if app.ClientID != a.ClientID {
				return ucerr.Friendlyf(nil, "cannot change ClientID for app '%v'", app.ID)
			}

			if !app.Equals(&tp.PlexConfig.PlexMap.Apps[i]) {
				different = true
				tp.PlexConfig.PlexMap.Apps[i] = app
			}
			found = true
			break
		}
	}

	if !found {
		return ucerr.Friendlyf(nil, "no app found with ID '%v'", app.ID)
	}

	if different {
		if err := m.SaveTenantPlex(ctx, tp); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}

// GetLoginApp returns the app with the specified app ID
func (m *Manager) GetLoginApp(ctx context.Context, tenant uuid.UUID, appID uuid.UUID) (*tenantplex.App, error) {
	tp, err := m.GetTenantPlex(ctx, tenant)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	for _, a := range tp.PlexConfig.PlexMap.Apps {
		if a.ID == appID {
			return &a, nil
		}
	}

	return nil, ucerr.Friendlyf(nil, "no app found with ID '%v'", appID)
}

// GetLoginApps returns the apps for the tenant's plex config that match the given orgID (nil for all)
func (m *Manager) GetLoginApps(ctx context.Context, tenant uuid.UUID, orgID uuid.UUID) ([]tenantplex.App, error) {
	tp, err := m.GetTenantPlex(ctx, tenant)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	apps := []tenantplex.App{}
	for _, a := range tp.PlexConfig.PlexMap.Apps {
		if orgID.IsNil() || a.OrganizationID == orgID {
			apps = append(apps, a)
		}
	}

	return apps, nil
}

// GetEmployeeApp returns the employee app for the tenant's plex config
func (m *Manager) GetEmployeeApp(ctx context.Context, tenant uuid.UUID) (*tenantplex.App, error) {
	tp, err := m.GetTenantPlex(ctx, tenant)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	employeeApp := tp.PlexConfig.PlexMap.EmployeeApp
	if employeeApp == nil {
		return nil, ucerr.Friendlyf(nil, "no employee app configured")
	}

	return employeeApp, nil
}

// LoginAppAlias returns the alias to use for the AuthZ object for an app
func LoginAppAlias(appID uuid.UUID) string {
	return fmt.Sprintf("login_app_%s", appID)
}

// NewLoginAppForOrganization is a helper function for creating a login app for an organization
func NewLoginAppForOrganization(ctx context.Context, tenantID uuid.UUID, orgName string, orgID uuid.UUID) (*tenantplex.App, error) {
	id := uuid.Must(uuid.NewV4())
	cs, err := crypto.GenerateClientSecret(ctx, id.String())
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &tenantplex.App{
		ID:             id,
		Name:           "Login for " + orgName,
		Description:    "Login app for " + orgName,
		OrganizationID: orgID,
		ClientID:       crypto.MustRandomHex(32),
		ClientSecret:   *cs,
	}, nil
}

// CopyUCAppSettings is a helper function that copies the relevant settings from the UserClouds Console app to another login app
func CopyUCAppSettings(ucApp *tenantplex.App, otherApp *tenantplex.App) {
	otherApp.AllowedRedirectURIs = ucApp.AllowedRedirectURIs
	otherApp.AllowedLogoutURIs = ucApp.AllowedLogoutURIs
	otherApp.MessageElements = ucApp.MessageElements
	otherApp.GrantTypes = ucApp.GrantTypes
	otherApp.ImpersonateUserConfig = ucApp.ImpersonateUserConfig
	otherApp.PageParameters = ucApp.PageParameters
	otherApp.ProviderAppIDs = ucApp.ProviderAppIDs
	otherApp.RestrictedAccess = ucApp.RestrictedAccess
	otherApp.TokenValidity = ucApp.TokenValidity
}
