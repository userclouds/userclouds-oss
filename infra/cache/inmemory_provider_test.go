package cache

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryCache(t *testing.T) {
	ctx := context.Background()
	const cacheName = "test-in-memory"
	const cacheNameTombstone = "test-in-memory-tomstone"
	t.Run("TestDependencyAdding", func(t *testing.T) {
		t.Parallel()
		c := NewInMemoryClientCacheProvider(cacheName)
		testDependencyAdding(ctx, t, c)
	})

	t.Run("TestDependencyAddingMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := NewInMemoryClientCacheProvider(cacheName)
		testDependencyAddingMultiThreaded(ctx, t, c, 10, 100, 10)
	})

	t.Run("TestDeleteMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := NewInMemoryClientCacheProvider("")
		testDeleteMultiThreaded(ctx, t, c, 1000, 10)
	})

	t.Run("TestMultiGet", func(t *testing.T) {
		t.Parallel()
		c := NewInMemoryClientCacheProvider(cacheName)
		testMultiGet(ctx, t, c)
	})

	t.Run("TestGetMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := NewInMemoryClientCacheProvider("")
		testGetMultiThreaded(ctx, t, c, 1000, 10)
	})

	t.Run("TestFlush", func(t *testing.T) {
		c := NewInMemoryClientCacheProvider(cacheName)
		testFlush(ctx, t, c)
	})

	t.Run("TestReleaseSentinelMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := NewInMemoryClientCacheProvider(cacheName)
		testReleaseSentinelMultiThreaded(ctx, t, c)
	})

	t.Run("TestSetValueMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := NewInMemoryClientCacheProvider(cacheName)
		testSetValueMultiThreaded(ctx, t, c)
	})

	t.Run("TestWriteSentinelMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := NewInMemoryClientCacheProvider(cacheName)
		testWriteSentinelMultiThreaded(ctx, t, c)
	})

	t.Run("TestTakeItemLockSerial", func(t *testing.T) {
		// t.Parallel() not running this test in parallel as it look at global collection key
		c := NewInMemoryClientCacheProvider(cacheName)
		testTakeItemLockSerial(ctx, t, c)
	})

	t.Run("TestTakeItemLockMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := NewInMemoryClientCacheProvider(cacheName)
		testTakeItemLockMultiThreaded(ctx, t, c)
	})

	t.Run("TestTakeCollectionLockMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := NewInMemoryClientCacheProvider(cacheName)
		testTakeCollectionLockMultiThreaded(ctx, t, c)
	})

	t.Run("TestGetValuesMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := NewInMemoryClientCacheProvider(cacheName)
		testGetValuesMultiThreaded(ctx, t, c, cacheName)
	})

	t.Run("TestGetValuesParams", func(t *testing.T) {
		t.Parallel()
		c := NewInMemoryClientCacheProvider(cacheName)
		testGetValuesParams(ctx, t, c)
	})

	t.Run("TestGetItemFromCacheValidation", func(t *testing.T) {
		c := NewInMemoryClientCacheProvider(cacheName)
		testGetItemFromCacheValidation(ctx, t, c)
	})
	t.Run("TestGetItemsArrayFromCacheValidation", func(t *testing.T) {
		c := NewInMemoryClientCacheProvider(cacheName)
		testGetItemsArrayFromCacheValidation(ctx, t, c)
	})
	t.Run("TestGetsItemsFromCacheValidation", func(t *testing.T) {
		c := NewInMemoryClientCacheProvider(cacheName)
		testGetItemsFromCacheValidation(ctx, t, c)
	})

	t.Run("TestTombstoneSentinel", func(t *testing.T) {
		sm := NewTombstoneCacheSentinelManager()
		c := NewInMemoryClientCacheProvider(cacheNameTombstone, SentinelManagerInMem(sm))
		testTombstoneSentinelManager(ctx, t, c)
	})

	t.Run("TestAPITombstoneSentinel", func(t *testing.T) {
		sm := NewTombstoneCacheSentinelManager()
		c := NewInMemoryClientCacheProvider(cacheNameTombstone, SentinelManagerInMem(sm))
		// To make debugging easier - increase the TTL to higher value so it doesn't expire while debugging
		ttl := 1 * time.Second
		c.SetTombstoneTTL(t, ttl)
		testGetAPIForTombstoneSentinel(ctx, t, c, ttl)
	})

	t.Run("TestRateLimiting", func(t *testing.T) {
		t.Parallel()
		cp := NewInMemoryClientCacheProvider(cacheName)
		testUnsupportedRateLimits(ctx, t, cp)
	})
}
