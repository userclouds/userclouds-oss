package appimport

import (
	"context"
	"slices"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider/auth0"
	"userclouds.com/plex/manager"
)

// checkProviderApp will find or create an Auth0App for the given auth0.App, and ensure that it exists in the given plexApp.ProviderAppIDs
func checkProviderApp(ctx context.Context, app auth0.App, plexApp *tenantplex.App, a0p *tenantplex.Provider) (*tenantplex.Auth0App, bool, error) {
	var create bool

	// find the matching auth0 app, if it exists
	var a0app *tenantplex.Auth0App
	for i, a := range a0p.Auth0.Apps {
		if a.ClientID == app.ClientID {
			a0app = &a0p.Auth0.Apps[i]
			break
		}
	}

	if a0app != nil {
		// app already exists, ensure it matches and that it's in ProviderAppIDs
		cs, err := a0app.ClientSecret.Resolve(ctx)
		if err != nil {
			return nil, false, ucerr.Wrap(err)
		}

		if cs != app.ClientSecret {
			return nil, false, ucerr.Errorf("client secret mismatch for a0app %s (%v)", app.Name, app.ClientID)
		}

		if !slices.Contains(plexApp.ProviderAppIDs, a0app.ID) {
			// if we found it but it wasn't in the plexApp, we mark create=true so we save the plexApp
			create = true
			plexApp.ProviderAppIDs = append(plexApp.ProviderAppIDs, a0app.ID)
		}
	} else {
		// create a new one, and add it to the plexApp
		id := uuid.Must(uuid.NewV4())
		cs, err := crypto.CreateClientSecret(ctx, id.String(), app.ClientSecret)
		if err != nil {
			return nil, false, ucerr.Wrap(err)
		}

		a0app = &tenantplex.Auth0App{
			ID:           id,
			Name:         app.Name,
			ClientID:     app.ClientID,
			ClientSecret: *cs,
		}

		create = true

		// save it in the provider
		a0p.Auth0.Apps = append(a0p.Auth0.Apps, *a0app)

		// and link it from the plexApp
		plexApp.ProviderAppIDs = append(plexApp.ProviderAppIDs, a0app.ID)
	}
	return a0app, create, nil
}

func updatePlexAppWithProvider(ctx context.Context, tm *tenantmap.StateMap, tenantID uuid.UUID, plexApp *tenantplex.App, provider *tenantplex.Provider) error {
	ts, err := tm.GetTenantStateForID(ctx, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	mgr := manager.NewFromDB(ts.TenantDB, ts.CacheConfig)

	tp, err := mgr.GetTenantPlex(ctx, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	var found bool
	for i, a := range tp.PlexConfig.PlexMap.Apps {
		if a.ID == plexApp.ID {
			tp.PlexConfig.PlexMap.Apps[i] = *plexApp
			found = true

			break
		}
	}

	if !found {
		return ucerr.Errorf("tried to save plexmap for %v but couldn't find app %v", tenantID, plexApp.ID)
	}

	found = false
	for i, p := range tp.PlexConfig.PlexMap.Providers {
		if p.ID == provider.ID {
			tp.PlexConfig.PlexMap.Providers[i] = *provider
			found = true
		}
	}

	if !found {
		return ucerr.Errorf("tried to save plexmap for %v but couldn't find provider %v", tenantID, provider.ID)
	}

	// save the plex app
	if err := mgr.SaveTenantPlex(ctx, tp); err != nil {
		return ucerr.Errorf("error saving plexApp for a0app %s (%v): %w", plexApp.Name, plexApp.ClientID, err)
	}
	return nil
}
