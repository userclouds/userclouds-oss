package usersync

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/worker/storage"
)

// syncAllUsersCognito syncs all users between active and follower providers
// for cognito ... since cognito actually exposes a
func syncAllUsersCognito(ctx context.Context, tenantID uuid.UUID, tc *tenantplex.TenantConfig, ts *storage.Storage, ap *tenantplex.Provider) error {
	active, err := factory.NewManagementClient(ctx, tc, *ap, uuid.Nil, uuid.Nil) // the admin API doesn't require appID, AWS doesn't support orgs
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
			return ucerr.Errorf("non-UC follower providers (type '%v') not yet supported for sync: %v", p.Type, p.ID)
		}

		followerIDs = append(followerIDs, p.ID)
		// TODO (sgarrity 4/24): this feels like a fragile way to get org ID
		f, err := factory.NewManagementClient(ctx, tc, p, tc.PlexMap.Apps[0].ID, tc.PlexMap.Apps[0].OrganizationID)
		if err != nil {
			return ucerr.Wrap(err)
		}
		followers = append(followers, f)
	}

	runStart := time.Now().UTC()
	thisRun := &storage.IDPSyncRun{
		BaseModel:           ucdb.NewBase(),
		Type:                storage.SyncRunTypeUser,
		ActiveProviderID:    ap.ID,
		FollowerProviderIDs: followerIDs,
		Until:               runStart,
	}

	users, err := active.ListUsers(ctx)
	if err != nil {
		thisRun.Error = ucerr.Wrap(err).Error()
	} else {
		uclog.Debugf(ctx, "got %d users from Cognito to sync", len(users))
		// only call sync users if ListUsers worked
		if err := syncUsers(ctx, ts, followers, thisRun, users, false); err != nil {
			thisRun.Error = ucerr.Wrap(err).Error()
		}
	}

	if err := ts.SaveIDPSyncRun(ctx, thisRun); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}
