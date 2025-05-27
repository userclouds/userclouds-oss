package appimport

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider/auth0"
)

func checkPlexApp(ctx context.Context, plexmap *tenantplex.PlexMap, app auth0.App, providerID uuid.UUID) (*tenantplex.App, bool, error) {
	var create bool
	plexApp, _, err := plexmap.FindAppForClientID(app.ClientID)
	if err != nil {
		uclog.Debugf(ctx, "plexApp not found for a0app %s (%v), creating one", app.Name, app.ClientID)

		if err := validateAuth0App(app); err != nil {
			uclog.Errorf(ctx, "error validating auth0 app -- we'll try anyway: %v", err)
		}

		// NB: don't shadow err
		gts, e := mapAuth0GrantTypesToUC(app.GrantTypes)
		if e != nil {
			return nil, false, ucerr.Wrap(e)
		}

		id := uuid.Must(uuid.NewV4())
		cs, err := crypto.CreateClientSecret(ctx, id.String(), app.ClientSecret)
		if err != nil {
			return nil, false, ucerr.Wrap(err)
		}

		// not found, create one
		plexApp = &tenantplex.App{
			ID:                  id,
			Name:                app.Name,
			Description:         app.Description,
			ClientID:            app.ClientID,
			ClientSecret:        *cs,
			AllowedRedirectURIs: app.Callbacks,
			AllowedLogoutURIs:   app.AllowedLogoutURLs,
			GrantTypes:          gts,
			SyncedFromProvider:  providerID,
			TokenValidity: tenantplex.TokenValidity{ // TODO: sync these with Auth0's expiration times
				Access:          ucjwt.DefaultValidityAccess,
				Refresh:         ucjwt.DefaultValidityRefresh,
				ImpersonateUser: ucjwt.DefaultValidityImpersonateUser,
			},
		}

		create = true

		// don't forget to zero out the error since we've "solved" it
		err = nil

		uclog.Debugf(ctx, "creating new plexapp %v to match auth0 app %v", plexApp.ID, app.ClientID)
	} else {
		// TODO: if we already had this app, should we check if it's the same?
		uclog.Debugf(ctx, "found extant plex app %v for auth0 app %v", plexApp.ID, app.ClientID)
	}

	return plexApp, create, ucerr.Wrap(err)
}

// used to help users debug, returns an array of strings with differences we found between plexapp and auth0app
func diffPlexApp(ctx context.Context, plexApp *tenantplex.App, app auth0.App) []string {
	var deltas []string

	if plexApp.Name != app.Name {
		deltas = append(deltas, "Name differs")
	}

	if plexApp.Description != app.Description {
		deltas = append(deltas, "Description differs")
	}

	if plexApp.ClientID != app.ClientID {
		deltas = append(deltas, "Client ID differs")
	}

	cs, err := plexApp.ClientSecret.Resolve(ctx)
	if err != nil {
		uclog.Errorf(ctx, "error resolving client secret: %v", err)
		deltas = append(deltas, "Couldn't compare Client Secret")
	}
	if cs != app.ClientSecret {
		deltas = append(deltas, "Client Secret differs")
	}

	// TODO: stringset to diff
	if len(plexApp.AllowedRedirectURIs) != len(app.Callbacks) {
		deltas = append(deltas, "Allowed Redirect URIs differ")
	}

	// TODO: stringset to diff
	if len(plexApp.AllowedLogoutURIs) != len(app.AllowedLogoutURLs) {
		deltas = append(deltas, "Allowed Logout URIs differ")
	}

	auth0gts, err := mapAuth0GrantTypesToUC(app.GrantTypes)
	if err != nil {
		uclog.Errorf(ctx, "error mapping auth0 grant types to uc: %v", err)
	}
	if len(plexApp.GrantTypes) != len(auth0gts) {
		deltas = append(deltas, "Grant Types differ")
	}

	// TODO: as we support more of auth0's features (see validate.go) we need to update this too

	return deltas
}
