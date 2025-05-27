package storage

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/tenantplex"
)

const (
	lookAsideCacheTTL        = 24 * time.Hour
	defaultInvalidationDelay = 50 * time.Millisecond
	tenantplexCacheName      = "tenantplexCache"
)

// Storage provides an object for database access
type Storage struct {
	db   *ucdb.DB
	cp   cache.Provider
	cm   *cache.Manager
	ttlP *tenantplex.PlexStorageCacheTTLProvider
}

var sharedCache cache.Provider
var sharedCacheOnce sync.Once

// New returns a Storage object
func New(ctx context.Context, db *ucdb.DB, cc *cache.Config) *Storage {
	s := &Storage{
		db: db,
	}

	if err := s.initializeCache(ctx, false, cc); err != nil {
		uclog.Fatalf(ctx, "Failed to create plex cache manager: %v", err)
	}

	return s
}

// NewForTests returns a Storage object using test cache prefix
func NewForTests(ctx context.Context, db *ucdb.DB, cc *cache.Config) *Storage {
	s := &Storage{db: db}
	if err := s.initializeCache(ctx, true, cc); err != nil {
		uclog.Fatalf(ctx, "Failed to create plex cache manager: %v", err)
	}

	return s
}

func (s *Storage) initializeCache(ctx context.Context, useTestPrefix bool, cc *cache.Config) error {
	if cc == nil || cc.RedisCacheConfig == nil {
		return nil
	}
	invalidationDelay := defaultInvalidationDelay
	if universe.Current().IsTestOrCI() {
		invalidationDelay = 1 * time.Millisecond // speed up tests
	}

	np := tenantplex.NewPlexStorageCacheNameProvider(useTestPrefix)
	sharedCacheOnce.Do(func() {
		var err error
		sharedCache, err = cache.InitializeInvalidatingCacheFromConfig(
			ctx,
			cc,
			tenantplexCacheName,
			np.GetPrefix(),
			cache.Layered(),
			cache.InvalidationDelay(invalidationDelay),
		)
		if err != nil {
			uclog.Errorf(ctx, "failed to create cache invalidation wrapper: %v", err)
		}
	})

	if sharedCache != nil {
		s.cp = sharedCache
		s.ttlP = tenantplex.NewPlexStorageCacheTTLProvider(lookAsideCacheTTL)
		cm := cache.NewManager(s.cp, np, s.ttlP)
		s.cm = &cm
	}
	return nil
}

// GetPlexTokenForAuthCode loads a PlexToken by looking up the auth code.
func (s *Storage) GetPlexTokenForAuthCode(ctx context.Context, authCode string) (*PlexToken, error) {
	const q = "SELECT id, created, updated, deleted, client_id, auth_code, access_token, id_token, refresh_token, idp_subject, scopes, session_id, underlying_token FROM plex_tokens WHERE auth_code=$1 AND deleted='0001-01-01 00:00:00';"

	var obj PlexToken
	if err := s.db.GetContext(ctx, "GetPlexTokenForAuthCode", &obj, q, authCode); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Wrap(ErrCodeNotFound)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetPlexTokenForAccessToken loads a PlexToken by looking up the access token.
func (s *Storage) GetPlexTokenForAccessToken(ctx context.Context, token string) (*PlexToken, error) {
	const q = "SELECT id, created, updated, deleted, client_id, auth_code, access_token, id_token, refresh_token, idp_subject, scopes, session_id, underlying_token FROM plex_tokens WHERE access_token=$1 AND deleted='0001-01-01 00:00:00';"

	var obj PlexToken
	if err := s.db.GetContext(ctx, "GetPlexTokenForAccessToken", &obj, q, token); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// ListRecentOTPStates returns a limited list of recent OTP state objects
func (s *Storage) ListRecentOTPStates(ctx context.Context, purpose OTPPurpose, email string, limit int) ([]OTPState, error) {
	const q = "SELECT id, created, updated, deleted, session_id, user_id, email, code, expires, used, purpose FROM otp_states WHERE purpose=$1 AND email=$2 AND deleted='0001-01-01 00:00:00' ORDER BY created DESC LIMIT $3;"

	var obj []OTPState
	if err := s.db.SelectContext(ctx, "ListRecentOTPStates", &obj, q, purpose, email, limit); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return obj, nil
}

//go:generate genorm OIDCLoginSession oidc_login_sessions tenantdb

//go:generate genorm MFAState mfa_states tenantdb

//go:generate genorm OTPState otp_states tenantdb

//go:generate genorm PKCEState pkce_states tenantdb

//go:generate genorm DelegationState delegation_states tenantdb

//go:generate genorm DelegationInvite delegation_invites tenantdb

//go:generate genorm --nodelete PlexToken plex_tokens tenantdb

//go:generate genorm SAMLSession saml_sessions tenantdb
