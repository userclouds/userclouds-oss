package testhelpers

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/secret"
)

// NewRedisConfigForTests returns a config object that configures the cache client to use a locally running redis
func NewRedisConfigForTests() *cache.Config {
	return &cache.Config{RedisCacheConfig: []cache.RegionalRedisConfig{
		{
			Region:      region.Current(),
			RedisConfig: newLocalRedis(0),
		}}}
}

func newLocalRedis(dbName uint8) cache.RedisConfig {
	return cache.RedisConfig{
		Host:     "127.0.0.1",
		Port:     6379,
		Password: secret.NewTestString(""),
		DBName:   dbName,
	}
}

// NewInMemCache creates a new in-memory cache client for testing
func NewInMemCache(ctx context.Context, t *testing.T, cacheName string, tenantID uuid.UUID, tombstoneTTL time.Duration) *cache.InvalidationWrapper {
	comms, err := cache.NewRedisCacheCommunicationProvider(ctx, NewCacheConfig(), true, false, cacheName)
	assert.NoErr(t, err)
	cacheProvider := cache.NewInMemoryClientCacheProvider(cacheName)
	if cache.InvalidationTombstoneTTL != tombstoneTTL {
		cacheProvider.SetTombstoneTTL(t, tombstoneTTL)
	}
	sc, err := cache.NewInvalidationWrapper(cacheProvider, comms, cache.OnMachine(), cache.InvalidationDelay(10*time.Millisecond))
	assert.NoErr(t, err)
	return sc
}

// NewRedisCache creates a new redis cache client for testing
func NewRedisCache(ctx context.Context, t *testing.T, cacheName string, tenantID uuid.UUID, redisDB int, tombstoneTTL time.Duration) *cache.InvalidationWrapper {
	comms, err := cache.NewRedisCacheCommunicationProvider(ctx, NewCacheConfig(), true, false, cacheName)
	assert.NoErr(t, err)
	cacheProvider := cache.NewRedisClientCacheProvider(cache.NewLocalRedisClientForDB(redisDB), cacheName)
	if cache.InvalidationTombstoneTTL != tombstoneTTL {
		cacheProvider.SetTombstoneTTL(t, tombstoneTTL)
	}
	sc, err := cache.NewInvalidationWrapper(cacheProvider, comms, cache.OnMachine(), cache.InvalidationDelay(10*time.Millisecond))
	assert.NoErr(t, err)
	return sc
}

// NewCacheConfig creates a new cache config for testing
func NewCacheConfig() *cache.Config {
	return &cache.Config{RedisCacheConfig: []cache.RegionalRedisConfig{*cache.NewLocalRedisClientConfigForTests()}}
}

// ValidateValueInCacheByID validates that the given value is in the cache for the given keyID + ID
func ValidateValueInCacheByID[item cache.SingleItem](ctx context.Context, t *testing.T, cm cache.Manager, keyID cache.KeyNameID, id uuid.UUID, expectedValue item) {
	t.Helper()

	ValidateValueInCacheKey(ctx, t, cm, cm.N.GetKeyNameWithID(keyID, id), expectedValue)
}

// ValidateValueInCacheKey validates that the given value is in the cache for the given key
func ValidateValueInCacheKey[item cache.SingleItem](ctx context.Context, t *testing.T, cm cache.Manager, key cache.Key, expectedValue item) {
	t.Helper()

	value, _, _, err := cache.GetItemFromCache[item](ctx, cm, key, false)
	assert.NoErr(t, err)
	assert.NotNil(t, value)
	if value != nil {
		assert.Equal(t, *value, expectedValue)
	}
}

// ForceDeleteCacheValueByID deletes the a cache value for a given object ID & key ID
func ForceDeleteCacheValueByID(ctx context.Context, t *testing.T, cm cache.Manager, keyID cache.KeyNameID, id uuid.UUID) {
	t.Helper()
	ForceDeleteCacheValueByKey(ctx, t, cm, cm.N.GetKeyNameWithID(keyID, id))
}

// ForceDeleteCacheValueByKey deletes the given a cache key from cache
func ForceDeleteCacheValueByKey(ctx context.Context, t *testing.T, cm cache.Manager, key cache.Key) {
	t.Helper()
	assert.NoErr(t, cm.Provider.DeleteValue(ctx, []cache.Key{key}, false, true))
}
