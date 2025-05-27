package appimport

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider/auth0"
	"userclouds.com/plex/manager"
	"userclouds.com/worker/storage"
)

// RunImport runs an import of the apps in auth0 `providerID` to the plexMap
func RunImport(ctx context.Context,
	tenantID uuid.UUID,
	plexmap *tenantplex.PlexMap,
	providerID uuid.UUID,
	ccs *companyconfig.Storage,
	tm *tenantmap.StateMap,
	azc *authz.Client,
	ss *storage.Storage) error {

	var a0p *tenantplex.Provider
	for i, p := range plexmap.Providers {
		if p.ID == providerID {
			if p.Type != tenantplex.ProviderTypeAuth0 {
				return ucerr.Errorf("provider %v is not an auth0 provider", providerID)
			}

			a0p = &plexmap.Providers[i]
			break
		}
	}

	run := &storage.IDPSyncRun{
		BaseModel:           ucdb.NewBase(),
		Type:                storage.SyncRunTypeApp,
		ActiveProviderID:    providerID,
		FollowerProviderIDs: []uuid.UUID{},
		Error:               "in progress", // TODO: actual state management
	}
	if err := ss.SaveIDPSyncRun(ctx, run); err != nil {
		return ucerr.Wrap(err)
	}

	a0mc, err := auth0.NewManagementClient(ctx, *a0p.Auth0)
	if err != nil {
		run.Error = "failed to authenticate to Auth0"
		if err := ss.SaveIDPSyncRun(ctx, run); err != nil {
			uclog.Errorf(ctx, "failed to save sync run: %v", err)
		}

		return ucerr.Wrap(err)
	}

	apps, err := a0mc.ListApps(ctx)
	if err != nil {
		run.Error = "failed to list Auth0 apps"
		if err := ss.SaveIDPSyncRun(ctx, run); err != nil {
			uclog.Errorf(ctx, "failed to save sync run: %v", err)
		}

		return ucerr.Wrap(err)
	}

	var es error // combined errors
	for _, app := range apps {
		// TODO should we apply "global" settings anywhere?
		if app.Global {
			continue
		}

		uclog.Debugf(ctx, "a0app import: found app %s (%v)", app.Name, app.ClientID)

		run.TotalRecords++

		rec := &storage.IDPSyncRecord{
			BaseModel: ucdb.NewBase(),
			SyncRunID: run.ID,
			ObjectID:  app.ClientID,
		}

		// validate there are no unknown settings here?
		if app.ClientSecret == "" {
			rec.Error += "missing client secret"
			run.FailedRecords++
			if err := ss.SaveIDPSyncRecord(ctx, rec); err != nil {
				uclog.Errorf(ctx, "failed to save sync record: %v", err)
			}

			// probably missing mgmt permission
			es = ucerr.Combine(es, ucerr.Errorf("missing client secret for app %s (%v)", app.Name, app.ClientID))
			continue
		}

		// lookup or create new plex app?
		plexApp, createPlexApp, err := checkPlexApp(ctx, plexmap, app, providerID)
		if err != nil {
			deltas := diffPlexApp(ctx, plexApp, app)
			if len(deltas) == 0 {
				rec.Warning = "app already synced, no changes detected"
			} else {
				// TODO: should this be anything other than newline-delimited?
				rec.Warning = strings.Join(deltas, ". \n")
			}

			run.WarningRecords++
			if err := ss.SaveIDPSyncRecord(ctx, rec); err != nil {
				uclog.Errorf(ctx, "failed to save sync record: %v", err)
			}

			es = ucerr.Combine(es, ucerr.Errorf("error importing plexApp for a0app %s (%v): %w", app.Name, app.ClientID, err))
			continue
		}

		if createPlexApp {
			// this is an "internal" error so we just don't save the import record
			ts, err := tm.GetTenantStateForID(ctx, tenantID)
			if err != nil {
				return ucerr.Wrap(err)
			}
			mgr := manager.NewFromDB(ts.TenantDB, ts.CacheConfig)

			if err := mgr.AddLoginApp(ctx, tenantID, azc, *plexApp); err != nil {
				return ucerr.Wrap(err)
			}
		}

		// do we need to create a matching auth0 app for login (grant-type controlled?)
		_, createProviderApp, err := checkProviderApp(ctx, app, plexApp, a0p)
		if err != nil {
			rec.Error += "client secret mismatch"
			run.FailedRecords++
			if err := ss.SaveIDPSyncRecord(ctx, rec); err != nil {
				uclog.Errorf(ctx, "failed to save sync record: %v", err)
			}

			es = ucerr.Combine(es, ucerr.Errorf("error importing providerApp for a0app %s (%v): %w", app.Name, app.ClientID, err))
			continue
		}

		// update the plex app with the provider
		if createProviderApp {
			// this is an "internal" error so we just don't save the import record
			if err := updatePlexAppWithProvider(ctx, tm, tenantID, plexApp, a0p); err != nil {
				es = ucerr.Combine(es, ucerr.Errorf("error saving plexApp for a0app %s (%v): %w", app.Name, app.ClientID, err))
				continue
			}
		}

		// save it
		if err := ss.SaveIDPSyncRecord(ctx, rec); err != nil {
			uclog.Errorf(ctx, "failed to save sync record: %v", err)
		}
	}

	if es == nil {
		run.Error = ""
	} else {
		run.Error = "run completed with issues"
	}

	// save with record stats etc
	if err := ss.SaveIDPSyncRun(ctx, run); err != nil {
		return ucerr.Wrap(err)
	}

	return ucerr.Wrap(es)
}
