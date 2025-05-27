package storage

import (
	"context"
	"sync"
	"time"

	"github.com/gofrs/uuid"

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
	db *ucdb.DB
	cm *cache.Manager
}

var sharedCache cache.Provider
var sharedCacheOnce sync.Once

// New returns a Storage object
func New(ctx context.Context, db *ucdb.DB, cc *cache.Config) *Storage {
	s := &Storage{
		db: db,
	}
	cm, err := getCacheManager(ctx, false, cc)

	if err != nil {
		uclog.Fatalf(ctx, "Failed to create plex cache manager: %v", err)
	}
	s.cm = cm
	return s
}

// NewForTests returns a Storage object using test cache prefix
func NewForTests(ctx context.Context, db *ucdb.DB, cc *cache.Config) *Storage {
	s := &Storage{db: db}
	cm, err := getCacheManager(ctx, true, cc)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to create plex cache manager: %v", err)
	}
	s.cm = cm

	return s
}

// getCacheManager returns a cache manager for the storage layer (this method is not private to tooling and testing purposes)
func getCacheManager(ctx context.Context, useTestPrefix bool, cc *cache.Config) (*cache.Manager, error) {
	if cc == nil || cc.RedisCacheConfig == nil {
		return nil, nil
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
			"",
			cache.Layered(),
			cache.InvalidationDelay(invalidationDelay),
		)
		if err != nil {
			uclog.Errorf(ctx, "failed to create cache invalidation wrapper: %v", err)
		}
	})

	if sharedCache == nil {
		return nil, nil
	}
	ttlP := tenantplex.NewPlexStorageCacheTTLProvider(lookAsideCacheTTL)
	cm := cache.NewManager(sharedCache, np, ttlP)

	return &cm, nil
}

// RegisterTenantPlexChangeHandler registers a handler for tenantplex changes
func (s *Storage) RegisterTenantPlexChangeHandler(ctx context.Context, handler cache.InvalidationHandler, tenantID uuid.UUID) error {
	if sharedCache == nil {
		return ucerr.Errorf("sharedCache is not initialized")
	}
	if s.cm == nil {
		return ucerr.Errorf("storage class was initialized without cache config")
	}
	key := s.cm.N.GetKeyNameWithString(tenantplex.TenantPlexKeyID, tenantID.String())
	return ucerr.Wrap(sharedCache.RegisterInvalidationHandler(ctx, handler, key))
}

//go:generate genorm --cache --followerreads --nomultiget --nolist tenantplex.TenantPlex plex_config tenantdb
