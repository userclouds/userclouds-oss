package cache

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
)

// getLayeringCacheProvider creates a new CacheLayeringWrapper with two RedisClientCacheProviders pointed at two different Redis databases
// This emulates client/server cache layering, with DB(0) being the client cache and DB(1) being the server cache
func getLayeringCacheProvider(t *testing.T, prefix string) *LayeringWrapper {
	t.Helper()

	rc1 := NewLocalRedisClientForDB(0)
	err := rc1.Ping(context.Background()).Err()
	assert.NoErr(t, err, assert.Must())
	rc2 := NewLocalRedisClientForDB(1)
	err = rc2.Ping(context.Background()).Err()
	assert.NoErr(t, err, assert.Must())

	rcp1 := NewRedisClientCacheProvider(rc1, "", KeyPrefixRedis(prefix))
	rcp2 := NewRedisClientCacheProvider(rc2, "", KeyPrefixRedis(prefix))

	return NewLayeringWrapper(rcp1, rcp2)
}

func TestLayeringCache(t *testing.T) {
	ctx := context.Background()

	t.Run("TestDependencyAdding", func(t *testing.T) {
		t.Parallel()
		rcp := getLayeringCacheProvider(t, "")
		testDependencyAdding(ctx, t, rcp)
	})

	t.Run("TestDependencyAddingMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getLayeringCacheProvider(t, "")
		testDependencyAddingMultiThreaded(ctx, t, rcp, 5, 40, 30)
	})

	t.Run("TestDeleteMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getLayeringCacheProvider(t, "")
		testDeleteMultiThreaded(ctx, t, rcp, 1000, 10)
	})

	t.Run("TestGetMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getLayeringCacheProvider(t, "")
		testGetMultiThreaded(ctx, t, rcp, 1000, 10)
	})

	t.Run("TestMultiGet", func(t *testing.T) {
		t.Parallel()
		rcp := getLayeringCacheProvider(t, "")
		testMultiGet(ctx, t, rcp)
	})

	t.Run("TestFlush", func(t *testing.T) {
		rcp := getLayeringCacheProvider(t, "")
		testFlush(ctx, t, rcp)
	})

	t.Run("TestReleaseSentinelMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getLayeringCacheProvider(t, "")
		testReleaseSentinelMultiThreaded(ctx, t, rcp)
	})

	t.Run("TestSetValueMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getLayeringCacheProvider(t, "")
		testSetValueMultiThreaded(ctx, t, rcp)
	})

	t.Run("TestWriteSentinelMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getLayeringCacheProvider(t, "")
		testWriteSentinelMultiThreaded(ctx, t, rcp)
	})

	t.Run("TestTakeItemLockSerial", func(t *testing.T) {
		// t.Parallel() not running this test in parallel as it looks at global collection key
		rcp := getLayeringCacheProvider(t, "")
		testTakeItemLockSerial(ctx, t, rcp)
	})

	t.Run("TestTakeItemLockMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getLayeringCacheProvider(t, "")
		testTakeItemLockMultiThreaded(ctx, t, rcp)
	})

	t.Run("TestTakeCollectionLockMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getLayeringCacheProvider(t, "")
		testTakeCollectionLockMultiThreaded(ctx, t, rcp)
	})

	t.Run("TestGetValuesMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getLayeringCacheProvider(t, "get-values-test")
		testGetValuesMultiThreaded(ctx, t, rcp, "get-values-test")
	})
	t.Run("TestGetValuesParams", func(t *testing.T) {
		t.Parallel()
		rcp := getLayeringCacheProvider(t, "")
		testGetValuesParams(ctx, t, rcp)
	})
	t.Run("TestGetItemFromCacheValidation", func(t *testing.T) {
		rcp := getLayeringCacheProvider(t, "")
		testGetItemFromCacheValidation(ctx, t, rcp)
	})

	t.Run("TestGetItemsArrayFromCacheValidation", func(t *testing.T) {
		rcp := getLayeringCacheProvider(t, "")
		testGetItemsArrayFromCacheValidation(ctx, t, rcp)
	})
	t.Run("TestGetItemsFromCacheValidation", func(t *testing.T) {
		rcp := getLayeringCacheProvider(t, "")
		testGetItemsFromCacheValidation(ctx, t, rcp)
	})
	t.Run("TestRateLimiting", func(t *testing.T) {
		t.Parallel()
		lcp := getLayeringCacheProvider(t, "")
		testSupportedRateLimitsSingleThreaded(ctx, t, lcp)
	})

	t.Run("TestReadLockReleasedWhenOuterLockFails", func(t *testing.T) {
		t.Parallel()
		lcp := getLayeringCacheProvider(t, "")
		testReadLockReleasedWhenOuterLockFails(ctx, t, lcp)
	})
}

// testReadLockReleasedWhenOuterLockFails validates that a read lock taken in the inner cache is released when
// the outer cache refuses to grant a lock for the same key. Otherwise the inner cache key stays locked until the
// sentinel expires, blocking reads of that key, and the caller gets back a sentinel that it can't use to set a value.
func testReadLockReleasedWhenOuterLockFails(ctx context.Context, t *testing.T, l *LayeringWrapper) {
	key := Key(uuid.Must(uuid.NewV4()).String())

	// Take a read lock on the key in the outer cache only, so the outer cache refuses to grant a lock for it below
	_, _, outerSentinel, _, err := l.cacheOuter.GetValue(ctx, key, true)
	assert.NoErr(t, err)
	assert.NotEqual(t, outerSentinel, NoLockSentinel, assert.Must(), assert.Errorf("Expected to lock key %v in the outer cache", key))

	// The key is missing from the inner cache, so this read takes a lock there, but it can't take one in the outer cache
	val, _, sentinel, _, err := l.GetValue(ctx, key, true)
	assert.NoErr(t, err)
	assert.IsNil(t, val, assert.Errorf("Expected a miss on key %v in both caches", key))

	// Since we couldn't lock the key in both caches, we don't hold a usable lock and the inner cache lock must be released
	assert.Equal(t, sentinel, NoLockSentinel, assert.Errorf("Expected no lock on key %v after failing to lock the outer cache", key))
	validateKeyContents(t, l.cacheInner, string(key), "", false, false)
}
