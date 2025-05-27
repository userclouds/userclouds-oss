package usersync

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantdb"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/manager"
	"userclouds.com/worker/storage"
)

// NB: this is a bit strange, but using the same object for factory + client lets us keep
// persistent objects across sync runs without a lot of storage in the factory
// Probably should split it out later with `map[uuid.UUID]iface.ManagementClient` so we can
// just reuse clients (and their state) but not needed yet
type syncFactory struct {
	provider.Factory
	iface.ManagementClient

	// this is what the test are expecting
	pages         int
	usersToReturn []iface.UserProfile

	// this is what the tests sent us
	usersSaved []iface.UserProfile

	currentPage int
}

func (s *syncFactory) NewManagementClient(ctx context.Context, tc *tenantplex.TenantConfig, p tenantplex.Provider, appID, appOrgID uuid.UUID) (iface.ManagementClient, error) {
	return s, nil
}

// our implementation is really lazy and ignores time for now
func (s *syncFactory) ListUsersUpdatedDuring(ctx context.Context, since, until time.Time) ([]iface.UserProfile, error) {
	if s.currentPage >= s.pages {
		return []iface.UserProfile{}, nil
	}

	upp := len(s.usersToReturn) / s.pages
	s.currentPage++
	return s.usersToReturn[(s.currentPage-1)*upp : s.currentPage*upp], nil
}

func (s *syncFactory) ListUsersForEmail(ctx context.Context, email string, authnType idp.AuthnType) ([]iface.UserProfile, error) {
	var us []iface.UserProfile
	for _, u := range s.usersSaved {
		if u.Email == email {
			us = append(us, u)
		}
	}
	return us, nil
}

func (s *syncFactory) CreateUserWithPassword(ctx context.Context, username, password string, profile iface.UserProfile) (string, error) {
	u := iface.UserProfile{
		ID: uuid.Must(uuid.NewV4()).String(),
		Authns: []idp.UserAuthn{{
			AuthnType: idp.AuthnTypePassword,
			Username:  username,
			Password:  password,
		}},
		UserBaseProfile: idp.UserBaseProfile{
			Name:          profile.Name,
			Nickname:      profile.Nickname,
			Email:         profile.Email,
			EmailVerified: profile.EmailVerified,
			Picture:       profile.Picture,
		},
	}
	s.usersSaved = append(s.usersSaved, u)
	return u.ID, nil
}

func (s *syncFactory) CreateUserWithOIDC(ctx context.Context, provider oidc.ProviderType, issuerURL string, oidcSubject string, profile iface.UserProfile) (string, error) {
	u := iface.UserProfile{
		ID: uuid.Must(uuid.NewV4()).String(),
		Authns: []idp.UserAuthn{{
			AuthnType:     idp.AuthnTypeOIDC,
			OIDCProvider:  provider,
			OIDCIssuerURL: issuerURL,
			OIDCSubject:   oidcSubject,
		}},
		UserBaseProfile: idp.UserBaseProfile{
			Name:          profile.Name,
			Nickname:      profile.Nickname,
			Email:         profile.Email,
			EmailVerified: profile.EmailVerified,
			Picture:       profile.Picture,
		},
	}
	s.usersSaved = append(s.usersSaved, u)
	return u.ID, nil
}

func (s *syncFactory) AddOIDCAuthnToUser(ctx context.Context, userID string, provider oidc.ProviderType, issuerURL string, oidcSubject string) error {
	var found bool
	for i, u := range s.usersSaved {
		if u.ID == userID {
			s.usersSaved[i].Authns =
				append(s.usersSaved[i].Authns,
					idp.UserAuthn{
						AuthnType:     idp.AuthnTypeOIDC,
						OIDCProvider:  provider,
						OIDCIssuerURL: issuerURL,
						OIDCSubject:   oidcSubject,
					})
			found = true
		}
	}

	if !found {
		return ucerr.Errorf("user %s not found", userID)
	}
	return nil
}

func (s *syncFactory) AddPasswordAuthnToUser(ctx context.Context, userID, email, password string) error {
	var found bool
	for i, u := range s.usersSaved {
		if u.ID == userID {
			s.usersSaved[i].Authns = append(s.usersSaved[i].Authns, idp.UserAuthn{
				AuthnType: idp.AuthnTypePassword,
				Username:  email,
				Password:  password,
			})
			found = true
		}
	}

	if !found {
		return ucerr.Errorf("user %s not found", userID)
	}
	return nil
}

func TestAuth0Sync(t *testing.T) {
	ctx := context.Background()

	// override our prod delays so we don't slow down tests (no rate limiting on our test fixtures yet ;) )
	perRequestDelay = 0

	// set up test tenant
	_, _, s := testhelpers.NewTestStorage(t)
	company, tenant, _, _, _, _ := testhelpers.CreateTestServer(ctx, t)
	// add an auth0 provider since we don't have one by default
	a0p := tenantplex.Provider{
		ID:   uuid.Must(uuid.NewV4()),
		Name: "Auth0",
		Type: tenantplex.ProviderTypeAuth0,
		Auth0: &tenantplex.Auth0Provider{
			Domain: "userclouds.auth0.com",
			Management: tenantplex.Auth0Management{
				ClientID:     "clientid",
				ClientSecret: secret.NewTestString("clientsecret"),
				Audience:     "https://userclouds.auth0.com/api/v2/",
			},
		},
	}

	mgr, err := manager.NewFromCompanyConfig(ctx, s, tenant.ID, cachetesthelpers.NewCacheConfig())
	assert.NoErr(t, err)
	defer mgr.Close(ctx)

	tp, err := mgr.GetTenantPlex(ctx, tenant.ID)
	assert.NoErr(t, err)
	tp.PlexConfig.PlexMap.Providers = append(tp.PlexConfig.PlexMap.Providers, a0p)
	tp.PlexConfig.PlexMap.Policy.ActiveProviderID = a0p.ID
	assert.NoErr(t, mgr.SaveTenantPlex(ctx, tp))

	// get the tenantDB to shove in the context
	ti, err := s.GetTenantInternal(ctx, tenant.ID)
	assert.NoErr(t, err)
	tenDB, err := ucdb.New(ctx, &ti.TenantDBConfig, migrate.SchemaValidator(tenantdb.Schema))
	assert.NoErr(t, err)
	tenantCtx := multitenant.SetTenantState(ctx, tenantmap.NewTenantState(tenant, company, uctest.MustParseURL(tenant.TenantURL), tenDB, nil, nil, "", s, false, nil, nil))

	plexStorage := storage.New(tenDB)

	sf := &syncFactory{pages: 2, usersToReturn: []iface.UserProfile{
		{
			Authns: []idp.UserAuthn{{
				AuthnType: idp.AuthnTypePassword,
				Username:  "bob",
				Password:  "foo",
			}},
		},
		{
			Authns: []idp.UserAuthn{{
				AuthnType: idp.AuthnTypePassword,
				Username:  "alice",
				Password:  "bar",
			}},
		},
	}}
	factory = sf

	t.Run("test sync", func(t *testing.T) {
		assert.NoErr(t, AllUsers(tenantCtx, tenant.ID, &tp.PlexConfig, plexStorage))
		// TODO: simplify pagination in tests, this is really verbose for very little
		assert.Equal(t, len(listRecords(ctx, t, plexStorage)), len(sf.usersToReturn))
		runs := listRuns(ctx, t, plexStorage) // this will vary by date

		var total, fails int
		for _, r := range runs {
			total += r.TotalRecords
			fails += r.FailedRecords
		}
		assert.Equal(t, total, len(sf.usersToReturn))
		assert.Equal(t, fails, 0)

		// run again with the same stuff, should be no new records but one new "run"
		assert.NoErr(t, AllUsers(tenantCtx, tenant.ID, &tp.PlexConfig, plexStorage))

		assert.Equal(t, len(listRecords(ctx, t, plexStorage)), len(sf.usersToReturn))
		newRuns := listRuns(ctx, t, plexStorage)
		assert.Equal(t, len(newRuns), len(runs)+1)

		// run again with a new user, should be one new record and one new "run"
		sf.usersToReturn = append(sf.usersToReturn, iface.UserProfile{
			Authns: []idp.UserAuthn{{
				AuthnType: idp.AuthnTypePassword,
				Username:  "eve",
				Password:  "evil",
			}},
		})
		sf.pages = 3

		assert.NoErr(t, AllUsers(tenantCtx, tenant.ID, &tp.PlexConfig, plexStorage))

		assert.Equal(t, len(listRecords(ctx, t, plexStorage)), len(sf.usersToReturn))
		newRuns = listRuns(ctx, t, plexStorage)
		assert.Equal(t, len(newRuns), len(runs)+2)

		// now remove the run state and try again, should be no new records
		pager, err := storage.NewIDPSyncRunPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
		assert.NoErr(t, err)
		runs, _, err = plexStorage.ListIDPSyncRunsPaginated(ctx, *pager)
		assert.NoErr(t, err)
		for _, r := range runs {
			assert.NoErr(t, plexStorage.DeleteIDPSyncRun(ctx, r.ID))
		}

		assert.NoErr(t, AllUsers(tenantCtx, tenant.ID, &tp.PlexConfig, plexStorage))
		assert.Equal(t, len(listRecords(ctx, t, plexStorage)), len(sf.usersToReturn))
	})

	t.Run("test reset after run errors", func(t *testing.T) {
		// TODO is there a way to make this a time.Time const shared between test and code?
		beginningOfAuth0Time, err := time.Parse(time.RFC3339, "2013-01-01T00:00:00Z")
		assert.NoErr(t, err)

		// scope this all to an imaginary active provider
		ap := uuid.Must(uuid.NewV4())

		// start at 2013
		got, err := getLastSyncRun(ctx, plexStorage, &tenantplex.Provider{ID: ap})
		assert.NoErr(t, err)
		assert.Equal(t, got.Until, beginningOfAuth0Time)

		// start from the last run
		fp := uuid.Must(uuid.NewV4())
		s := time.Now().UTC().Add(-time.Hour)

		// this line is brought to you by https://github.com/lib/pq/issues/329
		// TL;DR timezones are hard, and libpq uses a 0-offset timezone for for "TIMESTAMP WITHOUT TIME ZONE",
		// which is like UTC but with a different name, because that's "less wrong" than using UTC?
		// It also truncates at microsecond precision ... FML.
		// If we used all TIMESTAMPTZ instead of TIMESTAMP, we wouldn't need this line, but not worth
		// changing that now (and storing TZs with timestamps generally seems like a recipe for problems? UTC is king.)
		// We could also solve this by round-tripping this value (lr) through the DB, but that's worse (I think).
		s = s.In(time.FixedZone("", 0)).Round(time.Microsecond)

		u := s.Add(time.Minute)
		lr := &storage.IDPSyncRun{
			BaseModel:           ucdb.NewBase(),
			Type:                storage.SyncRunTypeUser,
			ActiveProviderID:    ap,
			FollowerProviderIDs: []uuid.UUID{fp},
			Since:               s,
			Until:               u,
		}
		lr.Deleted = lr.Deleted.In(time.FixedZone("", 0)) // see comment above about libpq
		assert.NoErr(t, plexStorage.SaveIDPSyncRun(ctx, lr))

		got, err = getLastSyncRun(ctx, plexStorage, &tenantplex.Provider{ID: ap})
		assert.NoErr(t, err)
		assert.Equal(t, got, lr)

		// if the last run had an error, start from 2013 again
		lr.Error = "oops"
		assert.NoErr(t, plexStorage.SaveIDPSyncRun(ctx, lr))

		got, err = getLastSyncRun(ctx, plexStorage, &tenantplex.Provider{ID: ap})
		assert.NoErr(t, err)
		assert.Equal(t, got.Until, beginningOfAuth0Time)
	})

	t.Run("TestFactory", func(t *testing.T) {
		// NB: this test ensure that stuff like multitenant.MustGet...() doesn't sneak into this path,
		// which we use without that middleware
		f := provider.ProdFactory{}
		_, err := f.NewManagementClient(ctx, &tp.PlexConfig, tp.PlexConfig.PlexMap.Providers[0], uuid.Nil, uuid.Nil) // TODO: this assumes organizations are disabled for this tenant
		assert.NoErr(t, err)
	})
}

func listRecords(ctx context.Context, t *testing.T, plexStorage *storage.Storage) []storage.IDPSyncRecord {
	t.Helper()
	pager, err := storage.NewIDPSyncRecordPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	assert.NoErr(t, err)

	records, _, err := plexStorage.ListIDPSyncRecordsPaginated(ctx, *pager)
	assert.NoErr(t, err)

	return records
}

func listRuns(ctx context.Context, t *testing.T, plexStorage *storage.Storage) []storage.IDPSyncRun {
	t.Helper()
	pager, err := storage.NewIDPSyncRunPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	assert.NoErr(t, err)
	runs, _, err := plexStorage.ListIDPSyncRunsPaginated(ctx, *pager)
	assert.NoErr(t, err)

	return runs
}
