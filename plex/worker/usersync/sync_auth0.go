package usersync

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/worker/storage"
)

func syncAllUsersAuth0(ctx context.Context, tenantID uuid.UUID, tc *tenantplex.TenantConfig, ts *storage.Storage, ap *tenantplex.Provider) error {
	active, err := factory.NewManagementClient(ctx, tc, *ap, uuid.Nil, uuid.Nil) // TODO: this assumes organizations are disabled for this tenant
	if err != nil {
		return ucerr.Errorf("failed to create mgmt client for %v (%v): %w", ap.ID, tenantID, err)
	}

	// list all the follower providers (sync sinks? sync targets.)
	fps := tc.PlexMap.ListFollowerProviders()

	if len(fps) == 0 {
		// can't sync to nothing
		return ucerr.Errorf("no follower providers configured for sync in tenant %v", tenantID)
	}

	var followers []iface.ManagementClient
	var followerIDs []uuid.UUID
	for _, p := range fps {
		if p.Type != tenantplex.ProviderTypeUC {
			return ucerr.Errorf("non-UC follower providers not yet supported for sync: %v", p.ID)
		}

		followerIDs = append(followerIDs, p.ID)
		f, err := factory.NewManagementClient(ctx, tc, p, uuid.Nil, uuid.Nil) // TODO: this assumes organizations are disabled for this tenant
		if err != nil {
			return ucerr.Wrap(err)
		}
		followers = append(followers, f)
	}

	// where did we leave off last time?
	lastRun, err := getLastSyncRun(ctx, ts, ap)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// max out our sync run at maxSyncDuration to try to minimize the chance Auth0
	// refuses to return all the users we need (max 1000 per their docs)
	// https://auth0.com/docs/api/management/v2#!/Users/get_users
	//
	// we run each one as a sync run so we can resume automatically if something goes wrong
	for {
		runStart := time.Now().UTC()
		thisRun := &storage.IDPSyncRun{
			BaseModel:           ucdb.NewBase(),
			Type:                storage.SyncRunTypeUser,
			ActiveProviderID:    ap.ID,
			FollowerProviderIDs: followerIDs,
			Since:               lastRun.Until,
			Until:               runStart,
		}

		if thisRun.Until.Sub(thisRun.Since) > maxSyncDuration {
			thisRun.Until = thisRun.Since.Add(maxSyncDuration)
		}

		uclog.Debugf(ctx, "kicking off user sync (ID %v, %v to %v)", thisRun.ID, thisRun.Since, thisRun.Until)

		err = sync(ctx, ts, active, followers, thisRun)
		if err != nil {
			thisRun.Error = err.Error()
		}

		// we use e instead of err since we want to actually return the sync error, if any
		if e := ts.SaveIDPSyncRun(ctx, thisRun); e != nil {
			// we don't return this error since it's probably less important than the sync error, if any
			uclog.Errorf(ctx, "error saving sync run: %v", e)
		}

		if err != nil {
			return ucerr.Wrap(err)
		}

		// if we synced the full duration, we're done...if we got cut back in scope, either by
		// maxSyncDuration or by auth0 search limits -> backoff, we're not done
		if thisRun.Until == runStart || thisRun.Until.After(runStart) {
			break
		}

		lastRun = thisRun

		uclog.Debugf(ctx, "finished worker sync (ID %v, %v to %v)", thisRun.ID, thisRun.Since, thisRun.Until)
		// rate limit is 2/sec on free tier at least
		time.Sleep(perRequestDelay)
	}

	return nil
}

func getLastSyncRun(ctx context.Context, ts *storage.Storage, ap *tenantplex.Provider) (*storage.IDPSyncRun, error) {
	lastRun, err := ts.GetLatestSuccessfulIDPSyncRun(ctx, ap.ID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Wrap(err)
		}

		// I don't love this, but Auth0 doesn't expose a way to query the first user
		// so we're starting with Auth0-didn't-exist-before-2013 :)
		// ideally we'd be able to check tenant creation data but there's no apparent API for that
		first, err := time.Parse(time.RFC3339, "2013-01-01T00:00:00Z")
		if err != nil {
			// this should never happen
			return nil, ucerr.Wrap(err)
		}

		// NB: we set "until" here because the calling code uses that as "ok, we've synced up to this point"
		// and then sets "thisRun.Since = lastRun.Until"
		lastRun = &storage.IDPSyncRun{
			Until: first,
		}
	}
	return lastRun, nil
}

// unified error handling to save sync records
func handleError(ctx context.Context, ts *storage.Storage, err error, sr *storage.IDPSyncRecord) {
	uclog.Debugf(ctx, "error syncing user %v: %v", sr.ObjectID, err)
	if err != nil {
		sr.Error = err.Error()
	} else {
		sr.Error = "unknown"
	}

	// save the sync record including the error
	if err := ts.SaveIDPSyncRecord(ctx, sr); err != nil {
		uclog.Errorf(ctx, "error saving sync record: %v", err)
	}
}

const maxAuth0Results = 1000 // https://auth0.com/docs/manage-users/user-search/retrieve-users-with-get-users-endpoint

func sync(ctx context.Context,
	ts *storage.Storage,
	active iface.ManagementClient,
	followers []iface.ManagementClient,
	thisSyncRun *storage.IDPSyncRun) error {

	// we specify both bounds to ensure that we don't miss any users during slow saves, etc
	users, err := active.ListUsersUpdatedDuring(ctx, thisSyncRun.Since, thisSyncRun.Until)
	if err != nil {
		return ucerr.Wrap(err)
	}

	uclog.Debugf(ctx, "got %d users from active on run %v", len(users), thisSyncRun.ID)

	// naive backoff implementation, if we hit max length then throw away
	// that result and try again with a smaller window
	if len(users) == maxAuth0Results {
		d := thisSyncRun.Until.Sub(thisSyncRun.Since)
		thisSyncRun.Until = thisSyncRun.Until.Add(-d / 2)
		time.Sleep(perRequestDelay) // for auth0 rate limiting
		return ucerr.Wrap(sync(ctx, ts, active, followers, thisSyncRun))
	}

	return ucerr.Wrap(syncUsers(ctx, ts, followers, thisSyncRun, users, true /* log all sync records */))
}
