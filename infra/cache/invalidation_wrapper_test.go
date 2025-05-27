package cache

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
)

func setKey[item any](ctx context.Context, t *testing.T, l Provider, key Key, val item) {
	setKeyHelper(ctx, t, l, key, val, false)
}

func setAndInvalidateKey[item any](ctx context.Context, t *testing.T, l Provider, key Key, val item) {
	setKeyHelper(ctx, t, l, key, val, true)
}

func setKeyHelper[item any](ctx context.Context, t *testing.T, l Provider, key Key, val item, invalidate bool) {
	cv, s := getValueHelper[item](ctx, t, l, key, true)
	sentinel := Read
	if invalidate {
		sentinel = Create
	}
	if cv != nil || invalidate {
		var err error
		s, err = l.WriteSentinel(ctx, sentinel, []Key{key})
		assert.NoErr(t, err)
	}

	valInStr, err := json.Marshal(val)
	assert.NoErr(t, err)

	_, _, err = l.SetValue(ctx, key, []Key{key}, string(valInStr), s, time.Minute)
	assert.NoErr(t, err)
}

func getValueHelper[item any](ctx context.Context, t *testing.T, l Provider, key Key, lockOnMiss bool) (*item, Sentinel) {
	val, _, s, _, err := l.GetValue(ctx, key, lockOnMiss)
	assert.NoErr(t, err)

	if val != nil {
		if IsTombstoneSentinel(*val) {
			return nil, NoLockSentinel
		}

		var i item
		assert.NoErr(t, json.Unmarshal([]byte(*val), &i))
		return &i, NoLockSentinel
	}
	return nil, s
}

func keyName(prefix string, key string) Key {
	return Key(prefix + key)
}

func stringCacheTestWithRedisInvalidation(ctx context.Context, t *testing.T, cp1, cp2 Provider, cacheName string, prefix string, addInvalidation bool) {
	var err error
	lsc1, lsc2 := cp1, cp2
	if addInvalidation {
		comms1 := getRedisCommProvider(ctx, t, cacheName)
		lsc1, err = NewInvalidationWrapper(cp1, comms1, OnMachine())
		assert.NoErr(t, err)
		comms2 := getRedisCommProvider(ctx, t, cacheName)
		lsc2, err = NewInvalidationWrapper(cp2, comms2, OnMachine())
		assert.NoErr(t, err)
	}

	setKey(ctx, t, lsc1, keyName(prefix, "key1"), "val1")
	// Storing same value twice should work
	setKey(ctx, t, lsc1, keyName(prefix, "key1"), "val1")
	// Storing different value twice should work
	setKey(ctx, t, lsc1, keyName(prefix, "key1"), "val0")
	// Store a couple more keys to test multiple messages and different invalidation apis
	setKey(ctx, t, lsc1, keyName(prefix, "key2"), "val3")
	setKey(ctx, t, lsc1, keyName(prefix, "key3"), "val5")
	// Store keys with same prefix to test flush
	setKey(ctx, t, lsc1, keyName(prefix, "key4_1"), "val6")
	setKey(ctx, t, lsc1, keyName(prefix, "key4_2"), "val6")

	// Create a different cache to test invalidation, store same keys/values
	setKey(ctx, t, lsc2, keyName(prefix, "key1"), "val1")
	setKey(ctx, t, lsc2, keyName(prefix, "key2"), "val3")
	setKey(ctx, t, lsc2, keyName(prefix, "key3"), "val5")
	setKey(ctx, t, lsc2, keyName(prefix, "key4_1"), "val6")
	setKey(ctx, t, lsc2, keyName(prefix, "key4_2"), "val6")

	// Check that the values are correct before invalidation
	val, _ := getValueHelper[string](ctx, t, lsc1, keyName(prefix, "key1"), false)
	assert.Equal(t, "val0", *val)

	val, _ = getValueHelper[string](ctx, t, lsc2, keyName(prefix, "key1"), false)
	assert.Equal(t, "val1", *val)

	val, _ = getValueHelper[string](ctx, t, lsc2, keyName(prefix, "key4_1"), false)
	assert.Equal(t, "val6", *val)

	// Update key1 triggering invalidation
	setAndInvalidateKey(ctx, t, lsc1, keyName(prefix, "key1"), "val2")

	val, _ = getValueHelper[string](ctx, t, lsc1, keyName(prefix, "key1"), false)
	assert.Equal(t, "val2", *val)

	// Update key2 triggering invalidation (make sure multiple messages on a channel work)
	setAndInvalidateKey(ctx, t, lsc1, keyName(prefix, "key2"), "val4")

	// Directly invalidate key3 (use a different channel to test multiple channels)
	err = lsc1.DeleteValue(ctx, []Key{keyName(prefix, "key3")}, false, true)
	assert.NoErr(t, err)

	// Trigger a flush invalidating all keys with prefix key4
	err = lsc1.Flush(ctx, string(keyName(prefix, "key4")), true)
	assert.NoErr(t, err)

	// Give the message time to propagate to the other client
	time.Sleep(5 * time.Second)

	// Check that all invalidated keys are cleared from the other cache
	val, _ = getValueHelper[string](ctx, t, lsc2, keyName(prefix, "key1"), false)
	assert.IsNil(t, val)

	val, _ = getValueHelper[string](ctx, t, lsc2, keyName(prefix, "key2"), false)
	assert.IsNil(t, val)

	val, _ = getValueHelper[string](ctx, t, lsc2, keyName(prefix, "key3"), false)
	assert.IsNil(t, val)

	val, _ = getValueHelper[string](ctx, t, lsc2, keyName(prefix, "key4_1"), false)
	assert.IsNil(t, val)

	val, _ = getValueHelper[string](ctx, t, lsc2, keyName(prefix, "key4_2"), false)
	assert.IsNil(t, val)

	// Check that new values are still in the cache that triggered the invalidation
	val, _ = getValueHelper[string](ctx, t, lsc1, keyName(prefix, "key1"), false)
	assert.NoErr(t, err)
	assert.Equal(t, "val2", *val)

	val, _ = getValueHelper[string](ctx, t, lsc1, keyName(prefix, "key2"), false)
	assert.NoErr(t, err)
	assert.Equal(t, "val4", *val)

	val, _ = getValueHelper[string](ctx, t, lsc1, keyName(prefix, "key3"), false)
	assert.IsNil(t, val)
	// Check that flushed keys are no longer in the cache that triggered the invalidation
	val, _ = getValueHelper[string](ctx, t, lsc1, keyName(prefix, "key4_1"), false)
	assert.IsNil(t, val)

	val, _ = getValueHelper[string](ctx, t, lsc1, keyName(prefix, "key4_2"), false)
	assert.IsNil(t, val)
}

func getRedisCommProvider(ctx context.Context, t *testing.T, cacheName string) CommunicationProvider {
	cc := Config{RedisCacheConfig: []RegionalRedisConfig{*NewLocalRedisClientConfigForTests()}}
	comms, err := NewRedisCacheCommunicationProvider(ctx, &cc, true, false, cacheName)
	assert.NoErr(t, err)
	return comms
}

func getInvalidationWrapper(ctx context.Context, t *testing.T, cacheName string) *InvalidationWrapper {
	comms := getRedisCommProvider(ctx, t, cacheName)
	delay := InvalidationDelay(10 * time.Millisecond)
	lsc, err := NewInvalidationWrapper(NewInMemoryClientCacheProvider(cacheName), comms, OnMachine(), delay)
	assert.NoErr(t, err)
	return lsc
}

func invalidationCallbackTest(ctx context.Context, t *testing.T, cp1, cp2 Provider, cacheName string, prefix string, addInvalidation bool) {
	var err error
	lsc1, lsc2 := cp1, cp2
	if addInvalidation {
		comms1 := getRedisCommProvider(ctx, t, cacheName)
		lsc1, err = NewInvalidationWrapper(cp1, comms1, OnMachine())
		assert.NoErr(t, err)
		comms2 := getRedisCommProvider(ctx, t, cacheName)
		lsc2, err = NewInvalidationWrapper(cp2, comms2, OnMachine())
		assert.NoErr(t, err)
	}
	key1 := keyName(prefix, "key1")

	wg := sync.WaitGroup{}

	cbCalledC1 := false
	cb1 := func(ctx context.Context, key Key, flush bool) error {
		assert.Equal(t, key, key1)
		cbCalledC1 = true
		wg.Done()
		return nil
	}
	cbCalledC2 := false
	cb2 := func(ctx context.Context, key Key, flush bool) error {
		assert.Equal(t, key, key1)
		cbCalledC2 = true
		wg.Done()
		return nil
	}
	err = lsc1.RegisterInvalidationHandler(ctx, cb1, key1)
	assert.NoErr(t, err)
	err = lsc2.RegisterInvalidationHandler(ctx, cb2, key1)
	assert.NoErr(t, err)

	// Test to ensure that local sets do not trigger invalidation
	setKey(ctx, t, lsc1, key1, "val1")
	time.Sleep(100 * time.Millisecond) // we can't wait on the callback since they will not be called
	assert.False(t, cbCalledC1)
	assert.False(t, cbCalledC2)

	// Test to ensure that invalidation is called on key creation
	wg.Add(1)
	setAndInvalidateKey(ctx, t, lsc1, key1, "val1")
	wg.Wait()
	assert.False(t, cbCalledC1)
	assert.True(t, cbCalledC2)

	cbCalledC2 = false
	wg.Add(1)
	setAndInvalidateKey(ctx, t, lsc2, key1, "val2")
	wg.Wait()
	assert.True(t, cbCalledC1)
	assert.False(t, cbCalledC2)

	// Test to ensure that invalidation is called on key update
	cbCalledC1 = false
	cbCalledC2 = false
	wg.Add(1)
	setAndInvalidateKey(ctx, t, lsc1, key1, "val1")
	wg.Wait()
	assert.False(t, cbCalledC1)
	assert.True(t, cbCalledC2)
	cbCalledC2 = false
	wg.Add(1)
	setAndInvalidateKey(ctx, t, lsc2, key1, "val2")
	wg.Wait()
	assert.True(t, cbCalledC1)
	assert.False(t, cbCalledC2)

	// Test to ensure that invalidatio is not called on unrelated key
	cbCalledC1 = false
	cbCalledC2 = false
	setAndInvalidateKey(ctx, t, lsc1, keyName(prefix, "key2"), "val1")
	setAndInvalidateKey(ctx, t, lsc1, keyName(prefix, "key2"), "val2")
	time.Sleep(100 * time.Millisecond) // we can't wait on the callback since they will not be called
	assert.False(t, cbCalledC1)
	assert.False(t, cbCalledC2)
}

func TestInvalidationWrapperCache_StringInvalidation(t *testing.T) {
	ctx := context.Background()

	t.Run("TestStringInvalidationInMemory", func(t *testing.T) {
		t.Parallel()

		prefix := uuid.Must(uuid.NewV4()).String()
		cacheName := "TestStringInvalidation" + prefix
		stringCacheTestWithRedisInvalidation(ctx, t, NewInMemoryClientCacheProvider(cacheName), NewInMemoryClientCacheProvider(cacheName), cacheName, prefix, true)
	})

	t.Run("TestStringInvalidationRedis", func(t *testing.T) {
		t.Parallel()

		prefix := uuid.Must(uuid.NewV4()).String()
		cacheName := "TestStringInvalidation" + prefix
		rc1 := NewLocalRedisClientForDB(0)
		rc2 := NewLocalRedisClientForDB(1)
		stringCacheTestWithRedisInvalidation(ctx, t, NewRedisClientCacheProvider(rc1, cacheName), NewRedisClientCacheProvider(rc2, cacheName), cacheName, prefix, true)
	})

	t.Run("TestStringInvalidationMixedRedisInMemory", func(t *testing.T) {
		t.Parallel()
		prefix := uuid.Must(uuid.NewV4()).String()
		cacheName := "TestStringInvalidation" + prefix
		rc1 := NewLocalRedisClientForDB(0)
		stringCacheTestWithRedisInvalidation(ctx, t, NewRedisClientCacheProvider(rc1, cacheName), NewInMemoryClientCacheProvider(cacheName), cacheName, prefix, true)
	})

	t.Run("TestStringInvalidationMixedInMemoryRedis", func(t *testing.T) {
		t.Parallel()
		prefix := uuid.Must(uuid.NewV4()).String()
		cacheName := "TestStringInvalidation" + uuid.Must(uuid.NewV4()).String()
		rc2 := NewLocalRedisClientForDB(1)
		stringCacheTestWithRedisInvalidation(ctx, t, NewInMemoryClientCacheProvider(cacheName), NewRedisClientCacheProvider(rc2, cacheName), cacheName, prefix, true)
	})

	t.Run("TestStringInvalidationMixedInMemoryRedisLayered", func(t *testing.T) {
		t.Parallel()

		rc2 := NewLocalRedisClientForDB(1)

		// We creating two layered cache providers which share same outer redis cache (DB=1) but have different instantiations of redis cache provider
		// and have in memory inner cache providers with same cache name. This is meant to mimic our normal running configuration for server side caching
		// when we use both in memory and redis cache providers.

		// We need to use two different cache names for the layers to prevent the inner cache invalidating the outer cache.
		prefix := uuid.Must(uuid.NewV4()).String()
		cacheName := "TestStringInvalidation" + uuid.Must(uuid.NewV4()).String()
		inMemCacheName := cacheName + "InMem"

		rcp1 := NewRedisClientCacheProvider(rc2, cacheName)
		mp1 := NewInMemoryClientCacheProvider(inMemCacheName)

		comms1 := getRedisCommProvider(ctx, t, inMemCacheName)
		lsc1, err := NewInvalidationWrapper(mp1, comms1, OnMachine())
		assert.NoErr(t, err)
		comms2 := getRedisCommProvider(ctx, t, cacheName)
		lsc2, err := NewInvalidationWrapper(rcp1, comms2, OnMachine())
		assert.NoErr(t, err)

		cp1 := NewLayeringWrapper(lsc1, lsc2)

		rcp2 := NewRedisClientCacheProvider(rc2, cacheName)
		mp2 := NewInMemoryClientCacheProvider(inMemCacheName)

		comms3 := getRedisCommProvider(ctx, t, inMemCacheName)
		lsc3, err := NewInvalidationWrapper(mp2, comms3, OnMachine())
		assert.NoErr(t, err)
		comms4 := getRedisCommProvider(ctx, t, cacheName)
		lsc4, err := NewInvalidationWrapper(rcp2, comms4, OnMachine())
		assert.NoErr(t, err)

		cp2 := NewLayeringWrapper(lsc3, lsc4)

		stringCacheTestWithRedisInvalidation(ctx, t, cp1, cp2, cacheName, prefix, false)
	})

	type testObject struct {
		Name   string `json:"name"`
		Length int    `json:"length"`
	}

	t.Run("TestObjectStorage", func(t *testing.T) {
		t.Parallel()
		comms := getRedisCommProvider(ctx, t, "TestObjectStorage")
		delay := 10 * time.Millisecond
		lsc1, err := NewInvalidationWrapper(NewInMemoryClientCacheProvider("TestObjectStorage"), comms, OnMachine(), InvalidationDelay(delay))
		assert.NoErr(t, err)
		lsc2, err := NewInvalidationWrapper(NewInMemoryClientCacheProvider("TestObjectStorage"), comms, OnMachine(), InvalidationDelay(delay))
		assert.NoErr(t, err)

		setKey(ctx, t, lsc1, "key1", testObject{Name: "val1", Length: 1})
		setKey(ctx, t, lsc2, "key1", testObject{Name: "val1", Length: 2})

		val, _ := getValueHelper[testObject](ctx, t, lsc1, "key1", false)
		assert.Equal(t, "val1", val.Name)

		val, _ = getValueHelper[testObject](ctx, t, lsc2, "key1", false)
		assert.Equal(t, "val1", val.Name)

		err = lsc1.DeleteValue(ctx, []Key{"key1"}, false, true)
		assert.NoErr(t, err)
		setKey(ctx, t, lsc1, "key1", testObject{Name: "val2", Length: 1})

		val, _ = getValueHelper[testObject](ctx, t, lsc1, "key1", false)
		assert.Equal(t, "val2", val.Name)

		val, _ = getValueHelper[testObject](ctx, t, lsc2, "key1", false)
		assert.IsNil(t, val)
	})
}

func TestInvalidationWrapperCache_InvalidationCallBack(t *testing.T) {
	ctx := context.Background()
	t.Run("TestInvalidationCallbackInMemory", func(t *testing.T) {
		t.Parallel()

		prefix := uuid.Must(uuid.NewV4()).String()
		cacheName := "TestInvalidationCallbackInMemory" + prefix
		invalidationCallbackTest(ctx, t, NewInMemoryClientCacheProvider(cacheName), NewInMemoryClientCacheProvider(cacheName), cacheName, prefix, true)
	})

	t.Run("TestInvalidationCallbackRedis", func(t *testing.T) {
		t.Parallel()

		prefix := uuid.Must(uuid.NewV4()).String()
		cacheName := "TestInvalidationCallbackRedis" + prefix
		rc1 := NewLocalRedisClientForDB(0)
		rc2 := NewLocalRedisClientForDB(1)
		invalidationCallbackTest(ctx, t, NewRedisClientCacheProvider(rc1, cacheName), NewRedisClientCacheProvider(rc2, cacheName), cacheName, prefix, true)
	})

	t.Run("TestInvalidationCallbackInMemoryRedis", func(t *testing.T) {
		t.Parallel()

		prefix := uuid.Must(uuid.NewV4()).String()
		cacheName := "TestInvalidationCallbackInMemory" + prefix
		rc1 := NewLocalRedisClientForDB(0)
		invalidationCallbackTest(ctx, t, NewInMemoryClientCacheProvider(cacheName), NewRedisClientCacheProvider(rc1, cacheName), cacheName, prefix, true)
	})

	t.Run("TestInvalidationCallbackInLayered", func(t *testing.T) {
		t.Parallel()

		rc2 := NewLocalRedisClientForDB(1)

		// We creating two layered cache providers which share same outer redis cache (DB=1) but have different instantiations of redis cache provider
		// and have in memory inner cache providers with same cache name. This is meant to mimic our normal running configuration for server side caching
		// when we use both in memory and redis cache providers.

		// We need to use two different cache names for the layers to prevent the inner cache invalidating the outer cache.
		prefix := uuid.Must(uuid.NewV4()).String()
		cacheName := "TestInvalidationCallbackInLayered" + uuid.Must(uuid.NewV4()).String()
		inMemCacheName := cacheName + "InMem"

		rcp1 := NewRedisClientCacheProvider(rc2, cacheName)
		mp1 := NewInMemoryClientCacheProvider(inMemCacheName)

		comms1 := getRedisCommProvider(ctx, t, inMemCacheName)
		lsc1, err := NewInvalidationWrapper(mp1, comms1, OnMachine())
		assert.NoErr(t, err)
		comms2 := getRedisCommProvider(ctx, t, cacheName)
		lsc2, err := NewInvalidationWrapper(rcp1, comms2, OnMachine())
		assert.NoErr(t, err)

		cp1 := NewLayeringWrapper(lsc1, lsc2)

		rcp2 := NewRedisClientCacheProvider(rc2, cacheName)
		mp2 := NewInMemoryClientCacheProvider(inMemCacheName)

		comms3 := getRedisCommProvider(ctx, t, inMemCacheName)
		lsc3, err := NewInvalidationWrapper(mp2, comms3, OnMachine())
		assert.NoErr(t, err)
		comms4 := getRedisCommProvider(ctx, t, cacheName)
		lsc4, err := NewInvalidationWrapper(rcp2, comms4, OnMachine())
		assert.NoErr(t, err)

		cp2 := NewLayeringWrapper(lsc3, lsc4)

		invalidationCallbackTest(ctx, t, cp1, cp2, cacheName, prefix, false)
	})
}
func testRedisCrossRegion(ctx context.Context, t *testing.T, useLayeringCache bool, region1DB int, region2DB int) {
	// This test scenario is setup like our AuthZ service which uses a redis cache and a local on machine cache which is manually maintained via invalidation handlers.
	// In the test we approximate two regions but the setup is not perfect since we are using the same redis instance for both regions. This means that some of the subscribers
	// see messages they normally wouldn't see because in production they are in different redis instances. For purposes of the test we use different regional channel names and
	// different redis DB numbers to emulate the different redis instances.
	//
	//
	// Each region has a a shared redis cache provider (like sharedCache in authz.Storage), an instance of invalidation subscriber (each representing an authz
	// service on different machines) and one cross region subcriber (like the one running worker in each region). The sequence is as follows:
	// 1. We trigger CREATE/UPDATE/DELETE via shared cache provider in region 1. This invalidates redis cache in region 1 and sends invalidation message to the regional channel on
	// redis instance in region 1.
	// 2a. The invalidation message is received by the invalidation subscriber in region 1 and it invalidate their local caches (representing edges caches on machine).
	// 2b. The invalidation message is received by the cross region subscriber in region 1 and it sends the invalidation message to the cross region channel on
	// redis instance in region 2 (or to all remote regions in prod). [note that unlike prod it will also see its own message after sending it]
	// 3. The invalidation message is received by the cross region subscriber in region 2
	// 4. The cross region subscriber in region 2 invalidates the redis cache in region 2
	// 5. The cross region subscriber sends the invalidation message to the local channel on redis instance in region 2.
	// 6. The invalidation message is received by the invalidation subscriber in region 2 and it invalidates their local caches (representing edges caches on machine).
	//
	// The sequence is very similar for Layering caches (IDP, etc) where the invalidation subscriber is inside the cache provider for in memory cache. We still register the
	// invalidation handler to just track the invalidation for testing purposes but the reads of the cache show if it was updated. The layered cache provider consists
	// of a redis cache provider (off machine) and an in memory cache provider (on machine). The redis cache provider is the one that sends the invalidation messages to
	// the regional channel that is picked up by worker and forwarded to the global channel. The worker in the remote region does the redis cache invalidation and then sends
	// the invalidation message to the local channel that is picked up by the invalidation subscriber inside the in memory cache in the remote region.
	//
	// 1. We trigger CREATE/UPDATE/DELETE via shared cache provider in region 1. This invalidates redis cache in region 1 and sends invalidation message to the regional channel on
	// redis instance in region 1.
	// 2a. The invalidation message is received by the invalidation subscriber in region 1 and it invalidate their local caches (representing edges caches on machine).
	// 2b. The invalidation message is received by the cross region subscriber in region 1 and it sends the invalidation message to the cross region channel on
	// redis instance in region 2 (or to all remote regions in prod). [note that unlike prod it will also see its own message after sending it]
	// 3. The invalidation message is received by the cross region subscriber in region 2
	// 4. The cross region subscriber in region 2 invalidates the redis cache in region 2
	// 5. The cross region subscriber sends the invalidation message to the local channel on redis instance in region 2.
	// 6. The invalidation message is received by the invalidation subscriber in region 2 and it invalidates their local caches (representing edges caches on machine).

	region1CacheName := "Region1_" + uuid.Must(uuid.NewV4()).String()
	region2CacheName := "Region2_" + uuid.Must(uuid.NewV4()).String()

	localCacheName := "Local_" + uuid.Must(uuid.NewV4()).String()

	c1Name := region1CacheName
	c2Name := region2CacheName

	opts1 := []InvalidationWrapperOption{InvalidationDelay(100 * time.Millisecond)}
	opts2 := []InvalidationWrapperOption{InvalidationDelay(100 * time.Millisecond)}

	// Select between AuthZ and IDP configs
	if useLayeringCache {
		// In this case (IDP) we want a layered cache composed of on machine in memory cache that will use channel localXCacheName
		// and off machine redis cache that will use channel regionXCacheName
		opts1 = append(opts1, Layered(), RegionalChannelName(region1CacheName))
		opts2 = append(opts2, Layered(), RegionalChannelName(region2CacheName))

		c1Name = localCacheName
		c2Name = localCacheName
	} else {
		// In this case (Authz) we want off machine redis cache that will use channel regionXCacheName
		opts1 = append(opts1, InvalidationHandlersLocalPublish(localCacheName, []string{"Key"}))
		opts2 = append(opts2, InvalidationHandlersLocalPublish(localCacheName, []string{"Key"}))
	}

	rc1 := NewLocalRedisClientForDB(region1DB)
	rc2 := NewLocalRedisClientForDB(region2DB)

	key1Name := Key("Key1_" + uuid.Must(uuid.NewV4()).String())
	key2Name := Key("Key2_" + uuid.Must(uuid.NewV4()).String())
	key3Name := Key("Ke_y3_" + uuid.Must(uuid.NewV4()).String())

	region1HandlersInvoked := make(map[Key]int)
	region2HandlersInvoked := make(map[Key]int)
	handlersMutex := sync.Mutex{}

	invRegion1Handler := func(ctx context.Context, key Key, flush bool) error {
		handlersMutex.Lock()
		defer handlersMutex.Unlock()
		if _, ok := region1HandlersInvoked[key]; !ok {
			region1HandlersInvoked[key] = 0
		}
		region1HandlersInvoked[key]++
		return nil
	}

	invRegion2Handler := func(ctx context.Context, key Key, flush bool) error {
		handlersMutex.Lock()
		defer handlersMutex.Unlock()
		if _, ok := region2HandlersInvoked[key]; !ok {
			region2HandlersInvoked[key] = 0
		}
		region2HandlersInvoked[key]++
		return nil
	}

	ccR1 := &Config{RedisCacheConfig: []RegionalRedisConfig{*NewLocalRedisClientConfigForTests()}}
	ccR1.RedisCacheConfig = append(ccR1.RedisCacheConfig, ccR1.RedisCacheConfig[0])
	ccR1.RedisCacheConfig[0].DBName = uint8(region1DB)
	ccR1.RedisCacheConfig[1].Region = "region2" // create pretend remote region, the name is not important as long as it is different from current region ("mars")
	ccR1.RedisCacheConfig[1].DBName = uint8(region2DB)
	region1Cache, err := InitializeInvalidatingCacheFromConfig(
		ctx,
		ccR1,
		c1Name,
		"",
		opts1...,
	)
	assert.NoErr(t, err)
	//defer region1Cache.Shutdown(ctx)

	v1, commsR1 := RunCrossRegionInvalidations(ctx, ccR1, region1CacheName, GlobalRedisCacheName)
	assert.NotNil(t, v1, assert.Must())
	defer v1.Shutdown(ctx)
	defer commsR1.Shutdown(ctx)

	invalidationCacheRegion1 := RunInRegionLocalHandlersSubscriber(ctx, ccR1, localCacheName)
	defer invalidationCacheRegion1.Shutdown(ctx)
	err = invalidationCacheRegion1.RegisterInvalidationHandler(ctx, invRegion1Handler, key1Name)
	assert.NoErr(t, err)
	err = invalidationCacheRegion1.RegisterInvalidationHandler(ctx, invRegion1Handler, key2Name)
	assert.NoErr(t, err)

	ccR2 := &Config{RedisCacheConfig: []RegionalRedisConfig{*NewLocalRedisClientConfigForTests()}}
	ccR2.RedisCacheConfig[0].DBName = uint8(region2DB)
	ccR2.RedisCacheConfig = append(ccR2.RedisCacheConfig, ccR2.RedisCacheConfig[0]) // create pretend remote region
	ccR2.RedisCacheConfig[1].DBName = uint8(region1DB)
	ccR2.RedisCacheConfig[1].Region = "region2" // to work around current region checking - both invalidators think they are running on mars
	region2Cache, err := InitializeInvalidatingCacheFromConfig(
		ctx,
		ccR2,
		c2Name,
		"",
		opts2...,
	)
	assert.NoErr(t, err)

	v2, commsR2 := RunCrossRegionInvalidations(ctx, ccR2, region2CacheName, GlobalRedisCacheName)
	assert.NotNil(t, v2, assert.Must())
	defer v2.Shutdown(ctx)
	defer commsR2.Shutdown(ctx)

	invalidationCacheRegion2 := RunInRegionLocalHandlersSubscriber(ctx, ccR2, localCacheName)
	defer invalidationCacheRegion2.Shutdown(ctx)
	err = invalidationCacheRegion2.RegisterInvalidationHandler(ctx, invRegion2Handler, key1Name)
	assert.NoErr(t, err)
	err = invalidationCacheRegion2.RegisterInvalidationHandler(ctx, invRegion2Handler, key2Name)
	assert.NoErr(t, err)

	// set dummy values in the cache so we can track of they are cleared
	rc1.Set(ctx, string(key1Name), "placeholder", time.Minute)
	rc2.Set(ctx, string(key1Name), "placeholder", time.Minute)

	s, err := region1Cache.WriteSentinel(ctx, Create, []Key{key1Name})
	assert.NoErr(t, err)
	set, _, err := region1Cache.SetValue(ctx, key1Name, []Key{key1Name}, "val1", s, time.Minute)
	assert.NoErr(t, err)
	assert.True(t, set)

	// Set call should have successfully updated the value in region 1
	cV, err := rc1.Get(ctx, string(key1Name)).Result()
	assert.NoErr(t, err)
	assert.Equal(t, cV, "val1")

	rawValue, _, _, partialHit, err := region1Cache.GetValue(ctx, key1Name, false)
	assert.NoErr(t, err)
	assert.NotNil(t, rawValue, assert.Must())
	assert.Equal(t, *rawValue, "val1")
	// for layered cache because of the shared redis instance the in memory cache in region 1 will be invalidated by message sent to region 2
	assert.Equal(t, partialHit, useLayeringCache)
	// redis cache in region 2 should be invalidated
	cV, err = rc2.Get(ctx, string(key1Name)).Result()
	assert.NoErr(t, err)
	assert.True(t, IsTombstoneSentinel(cV))

	// invalidation handlers in both regions should be called
	expectedCount := 2
	if useLayeringCache {
		expectedCount = 2 // Because the redis instance is shared between the two regions, the invalidation handlers will be called twice
	}
	handlersMutex.Lock()
	assert.Equal(t, region1HandlersInvoked[key1Name], expectedCount)
	assert.Equal(t, region2HandlersInvoked[key1Name], expectedCount)
	handlersMutex.Unlock()

	// Repeat the sequence in reverse with different key
	rc1.Set(ctx, string(key2Name), "placeholder", time.Minute)
	rc2.Set(ctx, string(key2Name), "placeholder", time.Minute)

	s, err = region2Cache.WriteSentinel(ctx, Update, []Key{key2Name})
	assert.NoErr(t, err)
	set, _, err = region2Cache.SetValue(ctx, key2Name, []Key{key2Name}, "val2", s, time.Minute)
	assert.NoErr(t, err)
	assert.True(t, set)

	// Set call should have successfully updated the value in region 1
	cV, err = rc2.Get(ctx, string(key2Name)).Result()
	assert.NoErr(t, err)
	assert.Equal(t, "val2", cV)

	rawValue, _, _, partialHit, err = region2Cache.GetValue(ctx, key2Name, false)
	assert.NoErr(t, err)
	assert.NotNil(t, rawValue, assert.Must())
	assert.Equal(t, "val2", *rawValue)
	// for layered cache because of the shared redis instance the in memory cache in region 1 will be invalidated by message sent to region 2
	assert.Equal(t, partialHit, useLayeringCache)

	// redis cache in region 2 should be invalidated
	cV, err = rc1.Get(ctx, string(key2Name)).Result()
	assert.NoErr(t, err)
	assert.True(t, IsTombstoneSentinel(cV))

	// test a key that shouldn't trigger invalidation handlers (applicable for AuthZ cache)
	if !useLayeringCache {
		rc1.Set(ctx, string(key3Name), "placeholder", time.Minute)
		rc2.Set(ctx, string(key3Name), "placeholder", time.Minute)

		s, err = region2Cache.WriteSentinel(ctx, Update, []Key{key3Name})
		assert.NoErr(t, err)
		set, _, err = region2Cache.SetValue(ctx, key3Name, []Key{key3Name}, "val3", s, time.Minute)
		assert.NoErr(t, err)
		assert.True(t, set)

		// Set call should have successfully updated the value in region 1
		cV, err = rc2.Get(ctx, string(key3Name)).Result()
		assert.NoErr(t, err)
		assert.Equal(t, "val3", cV)

		rawValue, _, _, _, err = region2Cache.GetValue(ctx, key3Name, false)
		assert.NoErr(t, err)
		assert.NotNil(t, rawValue, assert.Must())
		assert.Equal(t, "val3", *rawValue)

		// redis cache in region 2 should be invalidated
		cV, err = rc1.Get(ctx, string(key3Name)).Result()
		assert.NoErr(t, err)
		assert.True(t, IsTombstoneSentinel(cV))
	}

	// invalidation handlers in both regions should be called
	handlersMutex.Lock()
	assert.Equal(t, region1HandlersInvoked[key1Name], expectedCount)
	assert.Equal(t, region2HandlersInvoked[key1Name], expectedCount)
	assert.Equal(t, region1HandlersInvoked[key2Name], expectedCount)
	assert.Equal(t, region2HandlersInvoked[key2Name], expectedCount)
	handlersMutex.Unlock()

	time.Sleep(3 * time.Millisecond)
}

func TestInvalidationWrapperCache_TestInvalidationCrossRegion(t *testing.T) {
	ctx := context.Background()
	// We can get extra coverage by running the tests in parallel because they share the global notification channel and will see each others cross region invalidation messages
	// but we get cross talk on the local channel so making it hard to confirm the results
	t.Run("TestRedisCrossRegionWithInvalidationHandlers", func(t *testing.T) {
		//t.Parallel()

		testRedisCrossRegion(ctx, t, false, 0, 1)
	})
	t.Run("TestRedisCrossRegionWithLayeringCache", func(t *testing.T) {
		//t.Parallel()

		testRedisCrossRegion(ctx, t, true, 3, 4)
	})
}

func TestInvalidationWrapperCache(t *testing.T) {
	t.Parallel()
	const cacheName = "TestObjectStorage"
	ctx := context.Background()

	t.Run("TestDependencyAdding", func(t *testing.T) {
		t.Parallel()
		c := getInvalidationWrapper(ctx, t, cacheName)
		testDependencyAdding(ctx, t, c)
	})
	t.Run("TestDependencyAddingMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := getInvalidationWrapper(ctx, t, cacheName)
		testDependencyAddingMultiThreaded(ctx, t, c, 4, 20, 5)
	})

	t.Run("TestDeleteMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := getInvalidationWrapper(ctx, t, cacheName)
		testDeleteMultiThreaded(ctx, t, c, 50, 5)
	})

	t.Run("TestMultiGet", func(t *testing.T) {
		t.Parallel()
		c := getInvalidationWrapper(ctx, t, cacheName)
		testMultiGet(ctx, t, c)
	})

	t.Run("TestGetMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := getInvalidationWrapper(ctx, t, cacheName)
		testGetMultiThreaded(ctx, t, c, 50, 5)
	})

	t.Run("TestFlush", func(t *testing.T) {
		c := getInvalidationWrapper(ctx, t, cacheName)
		testFlush(ctx, t, c)
	})

	t.Run("TestReleaseSentinelMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := getInvalidationWrapper(ctx, t, cacheName)
		testReleaseSentinelMultiThreaded(ctx, t, c)
	})

	t.Run("TestSetValueMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := getInvalidationWrapper(ctx, t, cacheName)
		testSetValueMultiThreaded(ctx, t, c)
	})

	t.Run("TestWriteSentinelMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := getInvalidationWrapper(ctx, t, cacheName)
		testWriteSentinelMultiThreaded(ctx, t, c)
	})

	t.Run("TestTakeItemLockSerial", func(t *testing.T) {
		// t.Parallel() not running this test in parallel as it look at global collection key
		c := getInvalidationWrapper(ctx, t, cacheName)
		testTakeItemLockSerial(ctx, t, c)
	})

	t.Run("TestTakeItemLockMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := getInvalidationWrapper(ctx, t, cacheName)
		testTakeItemLockMultiThreaded(ctx, t, c)
	})

	t.Run("TestTakeCollectionLockMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := getInvalidationWrapper(ctx, t, cacheName)
		testTakeCollectionLockMultiThreaded(ctx, t, c)
	})

	t.Run("TestGetValuesMultiThreaded", func(t *testing.T) {
		t.Parallel()
		c := getInvalidationWrapper(ctx, t, cacheName)
		testGetValuesMultiThreaded(ctx, t, c, cacheName)
	})

	t.Run("TestGetValuesParams", func(t *testing.T) {
		t.Parallel()
		c := getInvalidationWrapper(ctx, t, cacheName)
		testGetValuesParams(ctx, t, c)
	})

	t.Run("TestGetItemFromCacheValidation", func(t *testing.T) {
		c := getInvalidationWrapper(ctx, t, cacheName)
		testGetItemFromCacheValidation(ctx, t, c)
	})
	t.Run("TestGetItemsArrayFromCacheValidation", func(t *testing.T) {
		c := getInvalidationWrapper(ctx, t, cacheName)
		testGetItemsArrayFromCacheValidation(ctx, t, c)
	})
	t.Run("TestGetsItemsFromCacheValidation", func(t *testing.T) {
		c := NewInMemoryClientCacheProvider(cacheName)
		testGetItemsFromCacheValidation(ctx, t, c)
	})
	t.Run("TestRateLimiting", func(t *testing.T) {
		t.Parallel()
		iw := getInvalidationWrapper(ctx, t, cacheName)
		testUnsupportedRateLimits(ctx, t, iw)
	})
}
