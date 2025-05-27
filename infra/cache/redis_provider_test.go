package cache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucerr"
)

func getRedisClient() (*redis.Client, error) {
	// TODO the client should be reading file config and picking a right test config
	rc := NewLocalRedisClient()
	if err := rc.Ping(context.Background()).Err(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return rc, nil
}

func getRedisCacheProvider(t *testing.T, prefix string) *RedisClientCacheProvider {
	t.Helper()
	rc, err := getRedisClient()
	assert.NoErr(t, err)
	return NewRedisClientCacheProvider(rc, "", KeyPrefixRedis(prefix))
}

func TestRedisCache(t *testing.T) {
	ctx := context.Background()

	t.Run("TestDependencyAdding", func(t *testing.T) {
		t.Parallel()
		rcp := getRedisCacheProvider(t, "")
		testDependencyAdding(ctx, t, rcp)

	})

	t.Run("TestDependencyAddingMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getRedisCacheProvider(t, "")
		testDependencyAddingMultiThreaded(ctx, t, rcp, 10, 400, 30)
	})

	t.Run("TestDeleteMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getRedisCacheProvider(t, "")
		testDeleteMultiThreaded(ctx, t, rcp, 1000, 10)
	})

	t.Run("TestGetMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getRedisCacheProvider(t, "")
		testGetMultiThreaded(ctx, t, rcp, 1000, 10)
	})

	t.Run("TestMultiGet", func(t *testing.T) {
		t.Parallel()
		rcp := getRedisCacheProvider(t, "")
		testMultiGet(ctx, t, rcp)
	})

	t.Run("TestFlush", func(t *testing.T) {
		rcp := getRedisCacheProvider(t, "")
		testFlush(ctx, t, rcp)
	})

	t.Run("TestReleaseSentinelMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getRedisCacheProvider(t, "")
		testReleaseSentinelMultiThreaded(ctx, t, rcp)
	})

	t.Run("TestSetValueMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getRedisCacheProvider(t, "")
		testSetValueMultiThreaded(ctx, t, rcp)
	})

	t.Run("TestWriteSentinelMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getRedisCacheProvider(t, "")
		testWriteSentinelMultiThreaded(ctx, t, rcp)
	})

	t.Run("TestTakeItemLockSerial", func(t *testing.T) {
		// t.Parallel() not running this test in parallel as it look at global collection key
		rcp := getRedisCacheProvider(t, "")
		testTakeItemLockSerial(ctx, t, rcp)
	})

	t.Run("TestTakeItemLockMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getRedisCacheProvider(t, "")
		testTakeItemLockMultiThreaded(ctx, t, rcp)
	})

	t.Run("TestTakeCollectionLockMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getRedisCacheProvider(t, "")
		testTakeCollectionLockMultiThreaded(ctx, t, rcp)
	})

	t.Run("TestPrefix", func(t *testing.T) {
		t.Parallel()
		rcpT1 := getRedisCacheProvider(t, "tenant1")
		rcpT2 := getRedisCacheProvider(t, "tenant2")
		_, _, _, _, err := rcpT1.GetValue(ctx, "tenant1:key1", false)
		assert.NoErr(t, err)
		_, _, _, _, err = rcpT1.GetValue(ctx, "tenant2:key1", false)
		assert.NotNil(t, err)
		_, _, _, _, err = rcpT2.GetValue(ctx, "tenant2:key1", false)
		assert.NoErr(t, err)
		_, _, _, _, err = rcpT2.GetValue(ctx, "tenant1:key1", false)
		assert.NotNil(t, err)
	})
	t.Run("TestGetValuesMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getRedisCacheProvider(t, "get-values-test")
		testGetValuesMultiThreaded(ctx, t, rcp, "get-values-test")
	})
	t.Run("TestGetValuesParams", func(t *testing.T) {
		t.Parallel()
		rcp := getRedisCacheProvider(t, "")
		testGetValuesParams(ctx, t, rcp)
	})
	t.Run("TestGetItemFromCacheValidation", func(t *testing.T) {
		rcp := getRedisCacheProvider(t, "")
		testGetItemFromCacheValidation(ctx, t, rcp)
	})

	t.Run("TestGetItemsArrayFromCacheValidation", func(t *testing.T) {
		rcp := getRedisCacheProvider(t, "")
		testGetItemsArrayFromCacheValidation(ctx, t, rcp)
	})
	t.Run("TestGetsItemsFromCacheValidation", func(t *testing.T) {
		rcp := getRedisCacheProvider(t, "")
		testGetItemsFromCacheValidation(ctx, t, rcp)
	})

	t.Run("TestTombstoneSentinel", func(t *testing.T) {
		sm := NewTombstoneCacheSentinelManager()

		rc, err := getRedisClient()
		assert.NoErr(t, err)
		rcp := NewRedisClientCacheProvider(rc, "", SentinelManagerRedis(sm))

		testTombstoneSentinelManager(ctx, t, rcp)
	})

	t.Run("TestAPITombstoneSentinel", func(t *testing.T) {
		sm := NewTombstoneCacheSentinelManager()

		rc, err := getRedisClient()
		assert.NoErr(t, err)
		rcp := NewRedisClientCacheProvider(rc, "", SentinelManagerRedis(sm))
		// To make debugging easier - increase the TTL to higher value so it doesn't expire while debugging
		ttl := 1 * time.Second
		rcp.SetTombstoneTTL(t, ttl)

		testGetAPIForTombstoneSentinel(ctx, t, rcp, ttl)
	})

	t.Run("TestReadOnlyOption", func(t *testing.T) {
		rc, err := getRedisClient()
		assert.NoErr(t, err)
		rcpReadOnly := NewRedisClientCacheProvider(rc, "", ReadOnlyRedis())
		rcpReadWrite := NewRedisClientCacheProvider(rc, "")
		// To make debugging easier - increase the TTL to higher value so it doesn't expire while debugging
		ttl := 1 * time.Second
		rcpReadWrite.SetTombstoneTTL(t, ttl)

		testReadOnlyBehavior(ctx, t, rcpReadWrite, rcpReadOnly)
	})

	t.Run("TestRateLimitingSingleThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getRedisCacheProvider(t, "")
		testSupportedRateLimitsSingleThreaded(ctx, t, rcp)
	})

	t.Run("TestRateLimitingMultiThreaded", func(t *testing.T) {
		t.Parallel()
		rcp := getRedisCacheProvider(t, "")
		testSupportedRateLimitsMultiThreaded(ctx, t, rcp)
	})
}
