package usersync

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/worker/storage"
)

const maxSyncDuration = time.Hour * 168 * 52 // TODO: one year, adjust as needed for tenant scale?

var perRequestDelay = time.Millisecond * 500 // this exists so we can override it in tests

var factory provider.Factory = provider.ProdFactory{} // same re tests

// AllUsers syncs all users between active and follower providers
func AllUsers(ctx context.Context, tenantID uuid.UUID, tc *tenantplex.TenantConfig, ts *storage.Storage) error {
	// find the active provider (sync source)
	ap, err := tc.PlexMap.GetActiveProvider()
	if err != nil {
		return ucerr.Wrap(err)
	}

	// we're forking this code for now because cognito supports a list-all-users endpoint, while auth0 doesn't
	// so for auth0 we have to fake it with this awkward time window thing (see sync_auth0.go)
	if ap.Type == tenantplex.ProviderTypeCognito {
		return ucerr.Wrap(syncAllUsersCognito(ctx, tenantID, tc, ts, ap))
	}

	if ap.Type == tenantplex.ProviderTypeAuth0 {
		return ucerr.Wrap(syncAllUsersAuth0(ctx, tenantID, tc, ts, ap))
	}

	// we only handle auth0 | cognito -> UC sync right now
	return ucerr.Errorf("active provider type '%v' not yet supported for sync", ap.Type)
}

// shared function
func syncUsers(ctx context.Context,
	ts *storage.Storage,
	followers []iface.ManagementClient,
	thisSyncRun *storage.IDPSyncRun,
	users []iface.UserProfile,
	logAllSyncRuns bool, // this is a bit of a kludgy param to avoid massive noise on cognito (non-delta) sync
) error {

	var err error

	for _, u := range users {
		uclog.Debugf(ctx, "syncing user %v", u)
		thisSyncRun.TotalRecords++

		// this is a bit kludgy, but easier than passing thisSyncRun into handleError
		// we'll always increment it and then decrement it later if we save a record successfully
		thisSyncRun.FailedRecords++

		sr := &storage.IDPSyncRecord{
			BaseModel: ucdb.NewBase(),
			SyncRunID: thisSyncRun.ID,
			ObjectID:  u.Email, // TODO: is email still the right UserID here?
		}

		for _, follower := range followers {
			us := []iface.UserProfile{}

			// if the user has an email, we can look up by that
			if u.Email != "" {
				// TODO: why is this a list and not a get -- shouldn't email be unique?
				us, err = follower.ListUsersForEmail(ctx, u.Email, idp.AuthnTypeAll)
				if err != nil {
					handleError(ctx, ts, err, sr)
					continue
				}
			} else {
				for _, a := range u.Authns {
					if a.AuthnType == idp.AuthnTypeOIDC {
						u, err := follower.GetUserForOIDC(ctx, a.OIDCProvider, a.OIDCIssuerURL, a.OIDCSubject, u.Email)
						if err != nil {
							handleError(ctx, ts, err, sr)
							continue
						}
						us = append(us, *u)
					}
				}
			}

			if len(us) == 1 {
				if u.Email != us[0].Email || u.Name != us[0].Name || u.Picture != us[0].Picture || u.Nickname != us[0].Nickname {
					// TODO: remove this log line, sensitive data
					uclog.Warningf(ctx, "user %s has different profiles in active and follower (*%v* vs *%v*)", u.Email, u, us[0])
					if err := follower.UpdateUser(ctx, us[0].ID, u); err != nil {
						handleError(ctx, ts, err, sr)
						continue
					}
				}

				// this is the case of an existing, already matching user
				if !logAllSyncRuns {
					sr = nil
				}
			} else if len(us) > 1 {
				handleError(ctx, ts, ucerr.New("got multiple users for one email"), sr)
				continue
			} else {
				// create a new user
				if len(u.Authns) == 0 {
					handleError(ctx, ts, ucerr.Errorf("user %s has no authns", u.Email), sr)
					continue
				}

				var uid string
				// create a new user using the first authn
				if u.Authns[0].AuthnType == idp.AuthnTypePassword {
					uid, err = follower.CreateUserWithPassword(ctx, u.Authns[0].Username, idp.PlaceholderPassword, u)
					if err != nil {
						handleError(ctx, ts, err, sr)
						continue
					}
				} else if u.Authns[0].AuthnType == idp.AuthnTypeOIDC {
					uid, err = follower.CreateUserWithOIDC(ctx, u.Authns[0].OIDCProvider, u.Authns[0].OIDCIssuerURL, u.Authns[0].OIDCSubject, u)
					if err != nil {
						handleError(ctx, ts, err, sr)
						continue
					}
				} else {
					handleError(ctx, ts, ucerr.Errorf("unknown authn type %s for user %v", u.Authns[0].AuthnType, u.Email), sr)
					continue
				}

				// if there are more authns ([1:]), add them to the user we just created
				for _, a := range u.Authns[1:] {
					if a.AuthnType == idp.AuthnTypePassword {
						if err = follower.AddPasswordAuthnToUser(ctx, uid, a.Username, idp.PlaceholderPassword); err != nil {
							handleError(ctx, ts, err, sr)
							continue
						}
					} else if a.AuthnType == idp.AuthnTypeOIDC {
						if err = follower.AddOIDCAuthnToUser(ctx, uid, a.OIDCProvider, a.OIDCIssuerURL, a.OIDCSubject); err != nil {
							handleError(ctx, ts, err, sr)
							continue
						}
					} else {
						handleError(ctx, ts, ucerr.Errorf("unknown authn type %s for user %v", a.AuthnType, u.Email), sr)
						continue
					}
				}
			}
		}

		// we successfully saved the sync record, so unmark the failure (see increment comment above)
		thisSyncRun.FailedRecords--

		// this will only be nil if logAllSyncRuns is false, and we found a matching user
		if sr != nil {
			if err := ts.SaveIDPSyncRecord(ctx, sr); err != nil {
				uclog.Errorf(ctx, "error saving sync record: %v", err)
				continue
			}
		}
	}

	return nil
}
