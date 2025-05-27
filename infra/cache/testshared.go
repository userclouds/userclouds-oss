package cache

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/uclog"
	"userclouds.com/test/testlogtransport"
)

// Note: the recursive parameter is used to prevent infinite recursion when dealing with nested cache providers (like CacheInvalidationWrapper)
func validateDepKeyContents(t *testing.T, c Provider, key string, expected []string, recursive bool) {
	t.Helper()

	var actual []string
	var m = make(map[string]struct{})
	var err error

	if mc, ok := c.(*InMemoryClientCacheProvider); ok {
		v, found := mc.cache.Get(key)
		assert.True(t, found, assert.Errorf("validateDepKeyContents: expected %v to be in cache", key))
		actual, ok = v.([]string)
		assert.True(t, ok, assert.Errorf("validateDepKeyContents: expected %v to be a []string", v))

		for _, k := range actual {
			m[k] = struct{}{}
		}

	} else if rcp, ok := c.(*RedisClientCacheProvider); ok {
		m, err = rcp.redisClient.SMembersMap(context.Background(), key).Result()
		assert.NoErr(t, err)
	} else if icw, ok := c.(*InvalidationWrapper); ok && recursive {
		validateDepKeyContents(t, icw.cache, key, expected, false)
		return
	} else if icw, ok := c.(*LayeringWrapper); ok && recursive {
		validateDepKeyContents(t, icw.cacheOuter, key, expected, false)
		validateDepKeyContents(t, icw.cacheInner, key, expected, false)
		return
	} else {
		assert.Fail(t, "validateDepKeyContents: unknown cache provider type: %T", c)
	}

	assert.Equal(t, len(m), len(expected), assert.Errorf("validateDepKeyContents: expected %v to be the same length as %v for key %v", actual, expected, key))
	for _, kv := range expected {
		_, found := m[kv]
		assert.True(t, found, assert.Errorf("validateDepKeyContents: expected %v to be in %v for key %v", kv, actual, key))
	}
}

// Note: the recursive parameter is used to prevent infinite recursion when dealing with nested cache providers (like CacheInvalidationWrapper)
func validateKeyContents(t *testing.T, c Provider, key string, expected string, expectFound, recursive bool) {
	t.Helper()

	found := false
	actual := ""

	if mc, ok := c.(*InMemoryClientCacheProvider); ok {
		var v any
		v, found = mc.cache.Get(key)
		if found {
			actual, ok = v.(string)
			assert.True(t, ok, assert.Errorf("validateKeyContents: expected %v to be a string", v))
		}
	} else if rcp, ok := c.(*RedisClientCacheProvider); ok {
		v, err := rcp.redisClient.Get(context.Background(), key).Result()
		if err == nil {
			found = true
			actual = v
		} else {
			assert.Equal(t, err, redis.Nil)
		}
	} else if icw, ok := c.(*InvalidationWrapper); ok && recursive {
		validateKeyContents(t, icw.cache, key, expected, expectFound, false)
		return
	} else if icw, ok := c.(*LayeringWrapper); ok && recursive {

		if isSentinelValue(expected) {
			topSentinel, baseSentinel := splitSentinels(Sentinel(expected))
			if IsTombstoneSentinel(expected) {
				baseSentinel = GenerateTombstoneSentinel()
			}
			validateKeyContents(t, icw.cacheInner, key, string(topSentinel), expectFound, false)
			validateKeyContents(t, icw.cacheOuter, key, string(baseSentinel), expectFound, false)
		} else {
			validateKeyContents(t, icw.cacheInner, key, expected, expectFound, false)
			validateKeyContents(t, icw.cacheOuter, key, expected, expectFound, false)
		}
		return
	} else {
		assert.Fail(t, "validateKeyContents: unknown cache provider type: %T", c)
	}

	if expectFound {
		assert.True(t, found)

		if IsTombstoneSentinel(expected) {
			// May want to add a parameter to check exact match vs both values being tombstones
			assert.True(t, IsTombstoneSentinel(actual), assert.Errorf("validateKeyContents: expected %v to be a tombstone", actual))
		} else {
			assert.Equal(t, actual, expected, assert.Errorf("validateKeyContents: unexpected %v value for key %v expected %v", actual, key, expected))
		}
	} else {
		assert.False(t, found, assert.Errorf("validateKeyContents: unexpected %v value for key %v expected empty", actual, key))
	}
}

// Note: the recursive parameter is used to prevent infinite recursion when dealing with nested cache providers (like CacheInvalidationWrapper)
func getKeyValue(ctx context.Context, t *testing.T, c Provider, key Key, recursive bool) (bool, string) {
	if mc, ok := c.(*InMemoryClientCacheProvider); ok {
		v, found := mc.cache.Get(string(key))
		actual := ""
		if found {
			actual, ok = v.(string)
			assert.True(t, ok, assert.Errorf("getKeyValue: expected %v to be a string", v))
		}
		return found, actual
	}

	if rcp, ok := c.(*RedisClientCacheProvider); ok {
		actual, err := rcp.redisClient.Get(ctx, string(key)).Result()
		if err == nil {
			return true, actual
		} else if err == redis.Nil {
			return false, ""
		}
		assert.NoErr(t, err)
	} else if icw, ok := c.(*InvalidationWrapper); ok && recursive {
		return getKeyValue(ctx, t, icw.cache, key, false)
	} else if icw, ok := c.(*LayeringWrapper); ok && recursive {
		return getKeyValue(ctx, t, icw.cacheInner, key, false)
	} else {
		assert.Fail(t, "getKeyValue: unknown cache provider type: %T", c)
	}
	return false, ""
}
func validateNonExistingKey(ctx context.Context, t *testing.T, c Provider, key Key) {
	found, actual := getKeyValue(ctx, t, c, key, true)
	assert.False(t, found)
	assert.Equal(t, actual, "")
}

func setKeyValue(ctx context.Context, t *testing.T, c Provider, key Key, value string) {
	t.Helper()

	if icw, ok := c.(*LayeringWrapper); ok {
		valueB, valueT := value, value
		if isSentinelValue(value) {
			topSentinel, baseSentinel := splitSentinels(Sentinel(value))
			if IsTombstoneSentinel(value) {
				baseSentinel = GenerateTombstoneSentinel()
			}
			valueB, valueT = string(baseSentinel), string(topSentinel)
		}
		setKeyValue(ctx, t, icw.cacheOuter, key, valueB)
		setKeyValue(ctx, t, icw.cacheInner, key, valueT)
		return
	}
	v, conflictv, s, _, err := c.GetValue(ctx, key, true)
	assert.NoErr(t, err)
	assert.IsNil(t, v)
	assert.IsNil(t, conflictv)
	r, conflict, err := c.SetValue(ctx, key, []Key{key}, value, s, time.Minute)
	assert.NoErr(t, err)
	assert.False(t, conflict)
	assert.True(t, r)
}

func setKeyValueDirect(ctx context.Context, t *testing.T, c Provider, key Key, value string) {
	t.Helper()

	if mc, ok := c.(*InMemoryClientCacheProvider); ok {
		mc.cache.Set(string(key), value, time.Minute)
	} else if rcp, ok := c.(*RedisClientCacheProvider); ok {
		_, err := rcp.redisClient.Set(ctx, string(key), value, time.Minute).Result()
		assert.NoErr(t, err)
	} else if icw, ok := c.(*InvalidationWrapper); ok {
		setKeyValueDirect(ctx, t, icw.cache, key, value)
	} else if icw, ok := c.(*LayeringWrapper); ok {
		setKeyValueDirect(ctx, t, icw.cacheOuter, key, value)
		setKeyValueDirect(ctx, t, icw.cacheInner, key, value)
	} else {
		assert.Fail(t, "setKeyValueDirect: unknown cache provider type: %T", c)
	}
}

func createSentinelHelper(c Provider, sm SentinelManager, stype SentinelType) Sentinel {
	if _, ok := c.(*LayeringWrapper); ok {
		return combineSentinels(sm.GenerateSentinel(stype), sm.GenerateSentinel(stype))
	}

	return sm.GenerateSentinel(stype)
}

func isSentinelValue(v string) bool {
	return strings.HasPrefix(v, sentinelPrefix) || IsTombstoneSentinel(v)
}

func testDependencyAdding[client Provider](ctx context.Context, t *testing.T, c client) {
	testDepKey1 := uuid.Must(uuid.NewV4()).String()
	testDepKey2 := uuid.Must(uuid.NewV4()).String()
	testDepKeys := []Key{Key(testDepKey1), Key(testDepKey2)}
	testValueKeys := []Key{"testValueKey1", "testValueKey2", "testValueKey1", "testValueKey2"}

	assert.NoErr(t, c.AddDependency(ctx, testDepKeys, testValueKeys, time.Minute))

	// Make sure duplicates are removed when adding contents to empty key
	validateDepKeyContents(t, c, testDepKey1, []string{"testValueKey1", "testValueKey2"}, true)
	validateDepKeyContents(t, c, testDepKey2, []string{"testValueKey1", "testValueKey2"}, true)

	testValueKeys2 := []Key{"testValueKey3", "testValueKey4", "testValueKey3", "testValueKey4"}
	testValueKeys2 = append(testValueKeys2, testValueKeys...)
	assert.NoErr(t, c.AddDependency(ctx, testDepKeys, testValueKeys2, time.Minute))

	// Make sure duplicates are removed when adding contents to non-empty key
	expected := []string{"testValueKey1", "testValueKey2", "testValueKey3", "testValueKey4"}
	validateDepKeyContents(t, c, testDepKey1, expected, true)
	validateDepKeyContents(t, c, testDepKey2, expected, true)

	// Set values for the dependent keys
	expectedKeyValues := []string{"testValueKey1Value", "testValueKey2Value", "testValueKey3Value", "testValueKey4Value"}
	for i, k := range expected {
		setKeyValue(ctx, t, c, Key(k), expectedKeyValues[i])
	}
	// Validate the dependent keys have the correct values
	for i, k := range expected {
		validateKeyContents(t, c, k, expectedKeyValues[i], true, true)
	}

	assert.NoErr(t, c.ClearDependencies(ctx, Key(testDepKey1), true))

	// Make sure the dependent keys are cleared
	for _, k := range expected {
		validateKeyContents(t, c, k, "", false, true)
	}
	// Make sure the dependency keys is set to tombstone
	validateKeyContents(t, c, testDepKey1, string(GenerateTombstoneSentinel()), true, true)

	// Make sure we can't add dependencies to tombstoned key
	err := c.AddDependency(ctx, []Key{Key(testDepKey1)}, testValueKeys2, time.Minute)
	assert.NotNil(t, err)
	validateKeyContents(t, c, testDepKey1, string(GenerateTombstoneSentinel()), true, true)

	// Clear the other dependency key for which all dependent keys are already cleared
	assert.NoErr(t, c.ClearDependencies(ctx, Key(testDepKey2), false))
	validateKeyContents(t, c, testDepKey2, "", false, true)

	// Clear a key that is already tombstoned and make sure it is a nop
	assert.NoErr(t, c.ClearDependencies(ctx, Key(testDepKey1), true))
	validateKeyContents(t, c, testDepKey1, string(GenerateTombstoneSentinel()), true, true)

	// Clear a key that is already tombstoned without tombstoning it and make sure it is a nop
	assert.NoErr(t, c.ClearDependencies(ctx, Key(testDepKey1), false))
	validateKeyContents(t, c, testDepKey1, string(GenerateTombstoneSentinel()), true, true)
}

func testDependencyAddingMultiThreaded[client Provider](ctx context.Context, t *testing.T, c client, threadCount int, itemCountPerThread int, keyCount int) {

	testID := uuid.Must(uuid.NewV4()).String()

	testDepKey1 := uuid.Must(uuid.NewV4()).String()
	testDepKey2 := uuid.Must(uuid.NewV4()).String()
	testDepKeys := []Key{Key(testDepKey1), Key(testDepKey2)}

	wg := sync.WaitGroup{}
	// 	Have threadCount threads add itemCountPerThread dependencies each for total of threadCount*itemCountPerThread dependencies per dependency key
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()
			for j := range itemCountPerThread {
				testValueKey := Key(fmt.Sprintf("testValueKey_%v_%d_%d", testID, threadID, j))
				setKeyValue(ctx, t, c, testValueKey, "testValue")

				var err error
				retry := true
				for retry {
					retry = false
					err = c.AddDependency(ctx, testDepKeys, []Key{testValueKey}, time.Minute)
					if err != nil {
						retry = true
					}
				}
				assert.NoErr(t, err)
			}
		}(i)
	}
	wg.Wait()

	// Calculate the expected value
	expected := make([]string, 0, itemCountPerThread*threadCount)
	for i := range threadCount {
		for j := range itemCountPerThread {
			expected = append(expected, fmt.Sprintf("testValueKey_%v_%d_%d", testID, i, j))
		}
	}
	validateDepKeyContents(t, c, testDepKey1, expected, true)
	validateDepKeyContents(t, c, testDepKey2, expected, true)

	err := c.ClearDependencies(ctx, Key(testDepKey2), true)
	assert.NoErr(t, err)
	validateKeyContents(t, c, testDepKey2, string(GenerateTombstoneSentinel()), true, true)

	// Make sure the dependent keys are cleared
	for _, k := range expected {
		validateKeyContents(t, c, k, "", false, true)
	}

	err = c.ClearDependencies(ctx, Key(testDepKey1), true)
	assert.NoErr(t, err)
	validateKeyContents(t, c, testDepKey1, string(GenerateTombstoneSentinel()), true, true)

	keyNames := make([]Key, keyCount)
	dependencyLists := make(map[Key][]string)
	for i := range keyNames {
		keyNames[i] = Key(uuid.Must(uuid.NewV4()).String())
		dependencyLists[keyNames[i]] = []string{}
	}
	dependencyListsLock := sync.Mutex{}

	// Have threadCount threads add itemCountPerThread dependencies to [Rand:KeyCount] dependency keys
	// for total of threadCount*itemCountPerThread/2 dependencies per dependency key average, writing KeyCount/2 keys on average call. Then validate that each dependency key
	// contains expected values and that clearing the dependency key clears all the dependent keys. I validated this with 10,000 dependencies per dependency key average
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()
			for j := range itemCountPerThread {
				testValueKey := Key(fmt.Sprintf("testValueKey_%v_%d_%d", testID, threadID, j))
				setKeyValue(ctx, t, c, testValueKey, "testValue")

				batchStart := rand.Intn(len(keyNames) - 1)
				depBatch := keyNames[batchStart:]

				err := c.AddDependency(ctx, depBatch, []Key{testValueKey}, time.Minute)
				assert.NoErr(t, err)

				dependencyListsLock.Lock()
				for _, k := range depBatch {
					dependencyLists[k] = append(dependencyLists[k], string(testValueKey))
				}
				dependencyListsLock.Unlock()
			}
		}(i)
	}
	wg.Wait()

	for _, k := range keyNames {
		validateDepKeyContents(t, c, string(k), dependencyLists[k], true)
		err = c.ClearDependencies(ctx, Key(k), true)
		assert.NoErr(t, err)
		validateKeyContents(t, c, string(k), string(GenerateTombstoneSentinel()), true, true)
	}

	for i := range keyNames {
		keyNames[i] = Key(uuid.Must(uuid.NewV4()).String())
		dependencyLists[keyNames[i]] = []string{}
	}

	clearedKeys := make(map[Key]bool)
	// Have each thread either add a dependency or clear the key
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()
			for j := range itemCountPerThread {
				depKey := keyNames[rand.Intn(len(keyNames))]

				if rand.Intn(10) == 9 { // Clear the key one in 10 times
					retry := true
					for retry {
						retry = false
						if err := c.ClearDependencies(ctx, depKey, true); err != nil {
							retry = true
						}
					}
					dependencyListsLock.Lock()
					clearedKeys[depKey] = true
					dependencyListsLock.Unlock()
					assert.NoErr(t, err)
				} else {
					testValueKey := Key(fmt.Sprintf("testValueKey_%v_%d_%d", testID, threadID, j))
					setKeyValue(ctx, t, c, testValueKey, "testValue")

					if err := c.AddDependency(ctx, []Key{depKey}, []Key{testValueKey}, time.Minute); err == nil {
						dependencyListsLock.Lock()
						dependencyLists[depKey] = append(dependencyLists[depKey], string(testValueKey))
						dependencyListsLock.Unlock()
					} else {
						if icw, ok := Provider(c).(*LayeringWrapper); ok {
							// We need to only check the outer cache since the inner cache may not yet have been tombstoned (i.e. ClearDependencies
							// is still running on the other thread and hasn't yet completed)
							validateKeyContents(t, icw.cacheOuter, string(depKey), string(GenerateTombstoneSentinel()), true, true)
						} else {
							validateKeyContents(t, c, string(depKey), string(GenerateTombstoneSentinel()), true, true)
						}
						err = c.DeleteValue(ctx, []Key{testValueKey}, false, true)
						assert.NoErr(t, err)
					}
				}
			}
		}(i)
	}
	wg.Wait()

	for _, k := range keyNames {
		if clearedKeys[k] {
			// Expect the dependencies key to be tombstones and all the dependencies to have been cleared
			validateKeyContents(t, c, string(k), string(GenerateTombstoneSentinel()), true, true)
			for _, dk := range dependencyLists[k] {
				validateKeyContents(t, c, dk, "", false, true)
			}
		} else {
			// Expect the dependencies key to have have all the dependencies that were added
			validateDepKeyContents(t, c, string(k), dependencyLists[k], true)
			err = c.ClearDependencies(ctx, Key(k), true)
			assert.NoErr(t, err)
			validateKeyContents(t, c, string(k), string(GenerateTombstoneSentinel()), true, true)
		}
	}

}

func testMultiGet[client Provider](ctx context.Context, t *testing.T, c client) {
	keyCount := 300
	keyNames := make([]Key, keyCount)
	keyValues := make(map[Key]string)
	for i := range keyNames {
		keyNames[i] = Key(uuid.Must(uuid.NewV4()).String())
		if (i % 2) == 0 {
			keyValues[keyNames[i]] = uuid.Must(uuid.NewV4()).String()
			setKeyValue(ctx, t, c, keyNames[i], keyValues[keyNames[i]])
			validateKeyContents(t, c, string(keyNames[i]), keyValues[keyNames[i]], true, true)
		} else {
			keyValues[keyNames[i]] = ""
		}
	}
	opCount := 300
	batchSizeMax := 15
	for range opCount {
		// Create a batch of keys to read
		batchSize := rand.Intn(batchSizeMax) + 1
		batch := make([]Key, batchSize)
		for i := range batchSize {
			batch[i] = keyNames[rand.Intn(keyCount)]
		}
		values, conflicts, sentinels, err := c.GetValues(ctx, batch, make([]bool, batchSize))
		assert.NoErr(t, err)

		// Nothing should be locked
		for _, s := range sentinels {
			assert.Equal(t, s, NoLockSentinel)
		}

		// No conflicts expected since we didn't lock anything
		for _, c := range conflicts {
			assert.IsNil(t, c)
		}

		// Validate the values
		for i, v := range values {
			val := ""
			if v != nil {
				val = *v
			}
			assert.Equal(t, val, keyValues[batch[i]])
		}
	}

	// Test duplicate keys in batch
	batch := []Key{keyNames[0], keyNames[0]} // Two keys with values in them
	values, conflicts, sentinels, err := c.GetValues(ctx, batch, make([]bool, len(batch)))
	assert.NoErr(t, err)
	assert.Equal(t, len(values), 2)
	assert.Equal(t, values[0], values[1])
	assert.Equal(t, sentinels[0], sentinels[1])
	assert.Equal(t, *values[0], keyValues[batch[0]])
	assert.Equal(t, sentinels[0], NoLockSentinel)
	assert.IsNil(t, conflicts[0])
	assert.IsNil(t, conflicts[1])

	batch = []Key{keyNames[1], keyNames[1]} // Two keys with no values in them
	values, conflicts, sentinels, err = c.GetValues(ctx, batch, make([]bool, len(batch)))
	assert.NoErr(t, err)
	assert.Equal(t, len(values), 2)
	assert.Equal(t, values[0], values[1])
	assert.Equal(t, sentinels[0], sentinels[1])
	assert.IsNil(t, values[0])
	assert.Equal(t, sentinels[0], NoLockSentinel)
	assert.IsNil(t, conflicts[0])
	assert.IsNil(t, conflicts[1])

	sm := NewWriteThroughCacheSentinelManager() // this is a little fragile since CacheProvider may have been passed a different manager
	batch = []Key{keyNames[1], keyNames[1]}     // Two keys with no values in them with locking
	locks := make([]bool, len(batch))
	for i := range batch {
		locks[i] = true
	}
	values, conflicts, sentinels, err = c.GetValues(ctx, batch, locks)
	assert.NoErr(t, err)
	assert.Equal(t, len(values), 2)
	assert.Equal(t, values[0], values[1])
	assert.Equal(t, sentinels[0], sentinels[1])
	assert.IsNil(t, values[0])
	assert.True(t, sm.IsReadSentinelPrefix(sentinels[0]))
	assert.IsNil(t, conflicts[0])
	assert.IsNil(t, conflicts[1])
	keyValues[batch[0]] = string(sentinels[0])

	// Repeat the call for same batch, we should get the sentinels from last call in the conflicts array
	values, conflicts, sentinels, err = c.GetValues(ctx, batch, locks)
	assert.NoErr(t, err)
	assert.Equal(t, len(values), 2)
	assert.Equal(t, values[0], values[1])
	assert.Equal(t, sentinels[0], sentinels[1])
	assert.IsNil(t, values[0])
	assert.Equal(t, sentinels[0], NoLockSentinel)
	assert.Equal(t, conflicts[0], conflicts[1])
	assert.Equal(t, *conflicts[0], keyValues[batch[0]])

	batch = []Key{keyNames[3], keyNames[5]} // Two keys with no values in them only lock one of them
	locks = make([]bool, len(batch))
	locks[1] = true
	values, conflicts, sentinels, err = c.GetValues(ctx, batch, locks)
	assert.NoErr(t, err)
	assert.Equal(t, len(values), 2)
	assert.Equal(t, values[0], values[1])
	assert.IsNil(t, values[0])
	assert.Equal(t, sentinels[0], NoLockSentinel)
	assert.True(t, sm.IsReadSentinelPrefix(sentinels[1]))
	assert.IsNil(t, conflicts[0])
	assert.IsNil(t, conflicts[1])

	keyValues[batch[1]] = string(sentinels[1])

	// Test with locking
	for range opCount {
		// Create a batch of keys to read
		batchSize := rand.Intn(batchSizeMax) + 1
		batch := make([]Key, batchSize)
		for i := range batchSize {
			batch[i] = keyNames[rand.Intn(keyCount)]
		}
		locks := make([]bool, batchSize)
		for i := range batchSize {
			locks[i] = true
		}
		values, conflicts, sentinels, err := c.GetValues(ctx, batch, locks)
		assert.NoErr(t, err)

		// Validate the values
		for i, v := range values {
			// If we already got a sentinel for the value we don't expect it to change
			if sm.IsReadSentinelPrefix(Sentinel(keyValues[batch[i]])) {
				assert.IsNil(t, v)
				if sentinels[i] != NoLockSentinel { // This means the key was in the batch more than once
					found := false
					for j := range i {
						if batch[j] == batch[i] {
							found = true
							break
						}
					}
					assert.IsNil(t, conflicts[i])
					assert.True(t, found)
				} else {
					assert.Equal(t, *conflicts[i], keyValues[batch[i]])
				}
			} else if v != nil { // If there is a value, we expect it to match the value we set
				assert.Equal(t, *v, keyValues[batch[i]])
				assert.IsNil(t, conflicts[i])
			} else { // If there is no value, we expect a sentinel and we expect it to be a read sentinel
				assert.True(t, sm.IsReadSentinelPrefix(sentinels[i]))
				keyValues[batch[i]] = string(sentinels[i])
				assert.IsNil(t, conflicts[i])
			}
		}
	}
}

func testDeleteMultiThreaded[client Provider](ctx context.Context, t *testing.T, c client, keyCount int, threadCount int) {
	keyNames := make([]Key, keyCount)
	for i := range keyNames {
		keyNames[i] = Key(uuid.Must(uuid.NewV4()).String())
		setKeyValue(ctx, t, c, keyNames[i], "testValue")
	}

	// Validate empty array delete
	assert.NoErr(t, c.DeleteValue(ctx, []Key{}, false, true))
	assert.NoErr(t, c.DeleteValue(ctx, []Key{}, false, false))
	// Validate single key delete
	keyName := Key(uuid.Must(uuid.NewV4()).String())
	setKeyValue(ctx, t, c, keyName, "testValue")
	assert.NoErr(t, c.DeleteValue(ctx, []Key{keyName}, false, false))
	validateKeyContents(t, c, string(keyName), "", false, true)
	setKeyValue(ctx, t, c, keyName, "testValue")
	assert.NoErr(t, c.DeleteValue(ctx, []Key{keyName}, false, true))
	validateKeyContents(t, c, string(keyName), "", false, true)
	// Validate sentinel value delete
	sm := NewWriteThroughCacheSentinelManager() // this is a little fragile since CacheProvider may have been passed a different manager
	sv := string(createSentinelHelper(c, sm, Read))
	setKeyValue(ctx, t, c, keyName, sv)
	assert.NoErr(t, c.DeleteValue(ctx, []Key{keyName}, false, false))
	validateKeyContents(t, c, string(keyName), sv, true, true)
	assert.NoErr(t, c.DeleteValue(ctx, []Key{keyName}, false, true))
	validateKeyContents(t, c, string(keyName), "", false, true)
	// Validate tombstone delete
	sv = string(GenerateTombstoneSentinel())
	setKeyValue(ctx, t, c, keyName, sv)
	assert.NoErr(t, c.DeleteValue(ctx, []Key{keyName}, false, false))
	validateKeyContents(t, c, string(keyName), sv, true, true)
	assert.NoErr(t, c.DeleteValue(ctx, []Key{keyName}, false, true))
	validateKeyContents(t, c, string(keyName), "", false, true)

	itemCountPerThread := keyCount / threadCount
	wg := sync.WaitGroup{}
	// 	Have threadCount threads add itemCountPerThread dependencies each for total of threadCount*itemCountPerThread dependencies per dependency key
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()

			start := threadID * itemCountPerThread
			end := min(start+itemCountPerThread, len(keyNames))
			keysForThread := keyNames[start:end]
			assert.NoErr(t, c.DeleteValue(ctx, keysForThread, false, true))
		}(i)
	}
	wg.Wait()
	for _, k := range keyNames {
		validateKeyContents(t, c, string(k), "", false, true)
	}
}

func testGetMultiThreaded[client Provider](ctx context.Context, t *testing.T, c client, keyCount int, threadCount int) {
	keyNames := make([]Key, keyCount)
	keyValues := make([]string, keyCount)
	for i := range keyNames {
		keyNames[i] = Key(uuid.Must(uuid.NewV4()).String())
		keyValues[i] = uuid.Must(uuid.NewV4()).String()
		setKeyValue(ctx, t, c, keyNames[i], keyValues[i])
	}

	wg := sync.WaitGroup{}
	// 	Have threadCount threads read the values and confirm they come back as expected
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()

			for i, k := range keyNames {
				v, conflict, s, _, err := c.GetValue(ctx, k, false)
				assert.NoErr(t, err)
				assert.Equal(t, s, NoLockSentinel)
				assert.NotNil(t, v, assert.Must())
				assert.Equal(t, *v, keyValues[i])
				assert.IsNil(t, conflict)
			}

			for i, k := range keyNames {
				v, conflict, s, _, err := c.GetValue(ctx, k, true)
				assert.NoErr(t, err)
				assert.Equal(t, s, NoLockSentinel)
				assert.Equal(t, *v, keyValues[i])
				assert.IsNil(t, conflict)
			}
		}(i)
	}
	wg.Wait()

	sm := NewWriteThroughCacheSentinelManager() // this is a little fragile since CacheProvider may have been passed a different manager
	wg = sync.WaitGroup{}
	// 	Have threadCount threads delete the values without setting a tombstone
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()

			for _, k := range keyNames {
				// since force = false, the deletes will not affect sentinel values)
				assert.NoErr(t, c.DeleteValue(ctx, []Key{k}, false, false))
			}

			for i, k := range keyNames {
				_, conflict, s, _, err := c.GetValue(ctx, k, true) // since we deleted all keys on this thread (regardless of if other threads also deleted them)
				assert.NoErr(t, err)                               // the key has to be either empty or set to a sentinel (in which case we get NoLockSentinel from Get)
				if s != NoLockSentinel {
					assert.IsNil(t, conflict)
					assert.Equal(t, sm.IsSentinelValue(keyValues[i]), false)
					keyValues[i] = string(s)
				}
			}
		}(i)
	}
	wg.Wait()

	for i, k := range keyNames {
		validateKeyContents(t, c, string(k), keyValues[i], true, true)
	}

	// 	Have threadCount threads delete the values and set tombstones
	keyLock := sync.Mutex{}
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()

			for _, k := range keyNames {
				// since force = false, the deletes will not affect sentinel values
				assert.NoErr(t, c.DeleteValue(ctx, []Key{k}, true, false))
			}

			for i, k := range keyNames {
				_, conflict, s, _, err := c.GetValue(ctx, k, true) // since we deleted all keys on this thread (regardless of if other threads also deleted them)
				assert.NoErr(t, err)                               // the key has to be either tombstone or set to a read sentinel (in which case we get NoLockSentinel from Get)
				assert.Equal(t, s, NoLockSentinel)
				assert.NotNil(t, conflict, assert.Must())
				assert.Equal(t, true, sm.IsSentinelValue(*conflict))
				keyLock.Lock()
				if keyValues[i] != *conflict {
					assert.True(t, IsTombstoneSentinel(*conflict))
					keyValues[i] = *conflict
				}
				keyLock.Unlock()
			}
		}(i)
	}
	wg.Wait()

	for i, k := range keyNames {
		validateKeyContents(t, c, string(k), keyValues[i], true, true)
	}
}

func testFlush[client Provider](ctx context.Context, t *testing.T, c client) {
	keyCount := 1000
	keyNames := make([]Key, keyCount)
	keyValues := make([]string, keyCount)
	for i := range keyNames {
		keyNames[i] = Key(uuid.Must(uuid.NewV4()).String())
		keyValues[i] = uuid.Must(uuid.NewV4()).String()
		if (i % 20) == 0 {
			keyValues[i] = string(GenerateTombstoneSentinel())
		}
		setKeyValue(ctx, t, c, keyNames[i], keyValues[i])
		validateKeyContents(t, c, string(keyNames[i]), keyValues[i], true, true)
	}
	assert.NoErr(t, c.Flush(ctx, "", false /* don't flush tombstones */))
	for i, k := range keyNames {
		if !IsTombstoneSentinel(keyValues[i]) {
			validateKeyContents(t, c, string(k), "", false, true)
		} else {
			validateKeyContents(t, c, string(k), keyValues[i], true, true)
		}
	}
}

func testReleaseSentinelMultiThreaded[client Provider](ctx context.Context, t *testing.T, c client) {

	sm := NewWriteThroughCacheSentinelManager() // this is a little fragile since CacheProvider may have been passed a different manager

	keyCount := 100
	keyNames := make([]Key, keyCount)
	keyValues := make([]string, keyCount)
	for i := range keyNames {
		keyNames[i] = Key(uuid.Must(uuid.NewV4()).String())
		keyValues[i] = string(createSentinelHelper(c, sm, Read))
		setKeyValue(ctx, t, c, keyNames[i], keyValues[i])
	}

	threadCount := 10
	wg := sync.WaitGroup{}
	// 	Have threadCount threads try to release key with non matching sentinel
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()

			for i, k := range keyNames {
				c.ReleaseSentinel(ctx, []Key{k}, createSentinelHelper(c, sm, Read))
				validateKeyContents(t, c, string(k), keyValues[i], true, true)
			}

			for i, k := range keyNames {
				c.ReleaseSentinel(ctx, []Key{k}, "")
				validateKeyContents(t, c, string(k), keyValues[i], true, true)
			}
		}(i)
	}
	wg.Wait()

	wg = sync.WaitGroup{}
	// 	Have threadCount threads try to release key with non matching sentinel
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()
			for i, k := range keyNames {
				c.ReleaseSentinel(ctx, []Key{k}, Sentinel(keyValues[i]))
				validateKeyContents(t, c, string(k), "", false, true)
			}
		}(i)
	}
	wg.Wait()

	for _, k := range keyNames {
		validateKeyContents(t, c, string(k), "", false, true)
	}
}

func testSetValueMultiThreaded[client Provider](ctx context.Context, t *testing.T, c client) {
	keyCount := 100
	keyNames := make([]Key, keyCount)
	keyValues := make([]string, keyCount)
	newKeyValues := make([]string, keyCount)

	sm := NewWriteThroughCacheSentinelManager()

	for i := range keyNames {
		keyNames[i] = Key(uuid.Must(uuid.NewV4()).String())
		keyValues[i] = string(createSentinelHelper(c, sm, Read))
		setKeyValue(ctx, t, c, keyNames[i], keyValues[i])
		validateKeyContents(t, c, string(keyNames[i]), keyValues[i], true, true)
	}

	set, _, err := c.SetValue(ctx, "", []Key{keyNames[0]}, "val", createSentinelHelper(c, sm, Read), time.Minute)
	assert.NotNil(t, err)
	assert.False(t, set)

	threadCount := 10
	wg := sync.WaitGroup{}
	// 	Have threadCount threads try to release key with non matching sentinel
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()

			// Check that value can't be set with wrong read sentinel
			for _, k := range keyNames {
				val := uuid.Must(uuid.NewV4()).String()
				set, _, err := c.SetValue(ctx, k, []Key{k}, val, createSentinelHelper(c, sm, Read), time.Minute)
				assert.NoErr(t, err)
				assert.False(t, set)
			}

			// Check that the sentinel can't be reused once the value is set
			for i, k := range keyNames {
				val := uuid.Must(uuid.NewV4()).String()
				set, _, err := c.SetValue(ctx, k, []Key{k}, val, Sentinel(keyValues[i]), time.Minute)
				assert.NoErr(t, err)
				if set {
					validateKeyContents(t, c, string(k), val, true, true)
					newKeyValues[i] = val
				}

			}
			// This should be a nop and no values should be affected
			for i, k := range keyNames {
				c.ReleaseSentinel(ctx, []Key{k}, Sentinel(keyValues[i]))
			}
		}(i)
	}
	wg.Wait()

	// All the keys should have new values
	for i, k := range keyNames {
		validateKeyContents(t, c, string(k), newKeyValues[i], true, true)
	}

	// Set some write locks
	for i := range keyNames {
		keyNames[i] = Key(uuid.Must(uuid.NewV4()).String())
		keyValues[i] = string(createSentinelHelper(c, sm, Update))
		setKeyValue(ctx, t, c, keyNames[i], keyValues[i])
		validateKeyContents(t, c, string(keyNames[i]), keyValues[i], true, true)
		newKeyValues[i] = ""
	}

	wg = sync.WaitGroup{}
	// 	Have threadCount threads try to release key with non matching sentinel
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()

			// Check that value can't be set with read sentinel
			for _, k := range keyNames {
				val := uuid.Must(uuid.NewV4()).String()
				set, _, err := c.SetValue(ctx, k, []Key{k}, val, createSentinelHelper(c, sm, Read), time.Minute)
				assert.NoErr(t, err)
				assert.False(t, set)
			}

			// The values should set once and clear by all subsequence threads on value mismatch
			for i, k := range keyNames {
				val := uuid.Must(uuid.NewV4()).String()
				set, conflict, err := c.SetValue(ctx, k, []Key{k}, val, Sentinel(keyValues[i]), time.Minute)
				assert.NoErr(t, err)
				assert.False(t, conflict)
				if set {
					// The value should get set only once by the first thread that has sentinel match
					assert.Equal(t, newKeyValues[i], "")
					newKeyValues[i] = val
				}

			}
			// This should be a nop and not values should be affected
			for i, k := range keyNames {
				c.ReleaseSentinel(ctx, []Key{k}, Sentinel(keyValues[i]))
			}
		}(i)
	}
	wg.Wait()

	// Set some write locks
	for i := range keyNames {
		keyNames[i] = Key(uuid.Must(uuid.NewV4()).String())
		keyValues[i] = string(createSentinelHelper(c, sm, Update))
		setKeyValue(ctx, t, c, keyNames[i], keyValues[i])
		validateKeyContents(t, c, string(keyNames[i]), keyValues[i], true, true)
	}

	wg = sync.WaitGroup{}
	// 	Have threadCount threads try to release key with non matching sentinel
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()

			// Create write conflicts with in flight writes
			for _, k := range keyNames {
				val := uuid.Must(uuid.NewV4()).String()
				set, conflict, err := c.SetValue(ctx, k, []Key{k}, val, createSentinelHelper(c, sm, Create), time.Minute)
				assert.NoErr(t, err)
				assert.False(t, set)
				assert.True(t, conflict)
			}
		}(i)
	}
	wg.Wait()

	wg = sync.WaitGroup{}
	// 	Have threadCount threads try to release key with non matching sentinel
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()

			// Try to set the value which should result in key being cleared on conflict or not set on mismatched sentinel
			for i, k := range keyNames {
				val := uuid.Must(uuid.NewV4()).String()
				set, conflict, err := c.SetValue(ctx, k, []Key{k}, val, Sentinel(keyValues[i]), time.Minute)
				assert.NoErr(t, err)
				assert.False(t, set)
				assert.False(t, conflict)
			}
			// This should be a nop and no values should be affected
			for i, k := range keyNames {
				c.ReleaseSentinel(ctx, []Key{k}, Sentinel(keyValues[i]))
			}
		}(i)
	}
	wg.Wait()

	// All the keys should be empty
	for _, k := range keyNames {
		validateKeyContents(t, c, string(k), "", false, true)
	}

}

func testWriteSentinelMultiThreaded[client Provider](ctx context.Context, t *testing.T, c client) {
	keyCount := 100
	keyNames := make([]Key, keyCount)
	keyValues := make([]string, keyCount)
	newKeyValues := make([]string, keyCount)

	for i := range keyNames {
		keyNames[i] = Key(uuid.Must(uuid.NewV4()).String())
	}

	sm := NewWriteThroughCacheSentinelManager()

	threadCount := 10
	wg := sync.WaitGroup{}
	// 	Test writing sentinel into empty key
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()
			// Try to set the value which should result in key being cleared on conflict or not set on mismatched sentinel
			for i, k := range keyNames {
				s, err := c.WriteSentinel(ctx, Read, []Key{k})
				assert.NoErr(t, err)
				if s != NoLockSentinel {
					// The value should get set only once by the first thread that has sentinel match
					assert.Equal(t, newKeyValues[i], "")
					newKeyValues[i] = string(s)
				}
			}
		}(i)
	}
	wg.Wait()
	for i, k := range keyNames {
		validateKeyContents(t, c, string(k), newKeyValues[i], true, true)
	}

	err := c.DeleteValue(ctx, keyNames, false, true)
	assert.NoErr(t, err)

	for i := range keyNames {
		keyNames[i] = Key(uuid.Must(uuid.NewV4()).String())
		keyValues[i] = uuid.Must(uuid.NewV4()).String()
		setKeyValue(ctx, t, c, keyNames[i], keyValues[i])
		validateKeyContents(t, c, string(keyNames[i]), keyValues[i], true, true)
		newKeyValues[i] = ""
	}

	wg = sync.WaitGroup{}
	// 	Test writing sentinel into key containing a value
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()
			for i, k := range keyNames {
				s, err := c.WriteSentinel(ctx, Read, []Key{k})
				assert.NoErr(t, err)
				if s != NoLockSentinel {
					// The sentinel should only be set once
					assert.Equal(t, newKeyValues[i], "")
					newKeyValues[i] = string(s)
				}
			}
		}(i)
	}
	wg.Wait()
	for i, k := range keyNames {
		validateKeyContents(t, c, string(k), newKeyValues[i], true, true)
	}
	assert.NoErr(t, c.DeleteValue(ctx, keyNames, false, true))

	for i := range keyNames {
		keyNames[i] = Key(uuid.Must(uuid.NewV4()).String())
		keyValues[i] = string(sm.GenerateSentinel(Read))
		setKeyValue(ctx, t, c, keyNames[i], keyValues[i])
		validateKeyContents(t, c, string(keyNames[i]), keyValues[i], true, true)
		newKeyValues[i] = ""
	}

	// 	Test writing sentinel into key containing a read sentinel
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()
			for _, k := range keyNames {
				s, err := c.WriteSentinel(ctx, Create, []Key{k})
				assert.NoErr(t, err)
				assert.NotEqual(t, s, NoLockSentinel)
			}
		}(i)
	}
	wg.Wait()

	assert.NoErr(t, c.DeleteValue(ctx, keyNames, false, true))

	for i := range keyNames {
		keyNames[i] = Key(uuid.Must(uuid.NewV4()).String())
		keyValues[i] = string(sm.GenerateSentinel(Create))
		setKeyValue(ctx, t, c, keyNames[i], keyValues[i])
		validateKeyContents(t, c, string(keyNames[i]), keyValues[i], true, true)
		newKeyValues[i] = ""
	}

	// 	Test delete sentinel into key containing a write sentinel
	deleteLock := sync.Mutex{}
	deleteSentinels := map[Key]Sentinel{}

	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()
			for _, k := range keyNames {
				s, err := c.WriteSentinel(ctx, Delete, []Key{k})
				assert.NoErr(t, err)
				deleteLock.Lock()
				if _, ok := deleteSentinels[k]; ok {
					// We shouldn't get a sentinel more than once for delete
					assert.Equal(t, s, NoLockSentinel)
				} else if s != NoLockSentinel {
					deleteSentinels[k] = s
				}
				deleteLock.Unlock()
			}
		}(i)
	}
	wg.Wait()
}

func testGetValuesParams[client Provider](ctx context.Context, t *testing.T, c client) {
	tt := testlogtransport.InitLoggerAndTransportsForTests(t)
	values, conflicts, sentinels, err := c.GetValues(ctx, nil, nil)
	assert.NoErr(t, err)
	assert.Equal(t, len(values), 0)
	assert.Equal(t, len(conflicts), 0)
	assert.Equal(t, len(sentinels), 0)
	tt.AssertMessagesByLogLevel(uclog.LogLevelError, 1) // Make sure we log an error

	values, conflicts, sentinels, err = c.GetValues(ctx, []Key{}, []bool{true})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Number of keys provided to GetValues has to be equal to number of lockOnMiss")
	assert.Equal(t, len(values), 0)
	assert.Equal(t, len(conflicts), 0)
	assert.Equal(t, len(sentinels), 0)

	values, conflicts, sentinels, err = c.GetValues(ctx, []Key{"jerry", "seinfeld"}, []bool{true})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Number of keys provided to GetValues has to be equal to number of lockOnMiss")
	assert.Equal(t, len(values), 0)
	assert.Equal(t, len(sentinels), 0)
	assert.Equal(t, len(conflicts), 0)
}

func testTombstoneSentinelManager[client Provider](ctx context.Context, t *testing.T, c client) {

	testKey1 := uuid.Must(uuid.NewV4()).String()
	testKey2 := uuid.Must(uuid.NewV4()).String()

	// Do a simple set emulating a read operation and validate the is set
	setKeyValue(ctx, t, c, Key(testKey1), "testValue")
	validateKeyContents(t, c, testKey1, "testValue", true, true)

	// Do a create and validate the tombstone sentinel is set on create
	s, err := c.WriteSentinel(ctx, Create, []Key{Key(testKey2)})
	assert.NoErr(t, err)
	assert.True(t, IsTombstoneSentinel(string(s)))
	validateKeyContents(t, c, testKey2, string(GenerateTombstoneSentinel()), true, true)
	v, conflictv, s, _, err := c.GetValue(ctx, Key(testKey2), true)
	assert.NoErr(t, err)
	assert.IsNil(t, v)
	assert.NotNil(t, conflictv)
	assert.True(t, IsTombstoneSentinel(*conflictv))
	assert.Equal(t, s, NoLockSentinel)
	validateKeyContents(t, c, testKey2, string(GenerateTombstoneSentinel()), true, true)

	// Do update and validate the tombstone sentinel is set on update
	s, err = c.WriteSentinel(ctx, Update, []Key{Key(testKey1)})
	assert.NoErr(t, err)
	assert.True(t, IsTombstoneSentinel(string(s)))
	validateKeyContents(t, c, testKey1, string(GenerateTombstoneSentinel()), true, true)
	v, conflictv, s, _, err = c.GetValue(ctx, Key(testKey1), true)
	assert.NoErr(t, err)
	assert.IsNil(t, v)
	assert.NotNil(t, conflictv)
	assert.True(t, IsTombstoneSentinel(*conflictv))
	assert.Equal(t, s, NoLockSentinel)
	validateKeyContents(t, c, testKey1, string(GenerateTombstoneSentinel()), true, true)

	// Do delete and validate the tombstone sentinel is set on delete
	setKeyValueDirect(ctx, t, c, Key(testKey1), "testValue2")
	s, err = c.WriteSentinel(ctx, Delete, []Key{Key(testKey1)})
	assert.NoErr(t, err)
	assert.True(t, IsTombstoneSentinel(string(s)))
	validateKeyContents(t, c, testKey1, string(GenerateTombstoneSentinel()), true, true)
}

// testReadOnlyBehavior verifies that a readonly cache provider doesn't modify the cache
// while still able to read data if it exists
func testReadOnlyBehavior[client Provider](ctx context.Context, t *testing.T, c client, cRO client) {

	testKey1 := uuid.Must(uuid.NewV4()).String()
	testKey2 := uuid.Must(uuid.NewV4()).String()

	// Do a simple set emulating a read operation and validate the is set
	setKeyValue(ctx, t, c, Key(testKey1), "testValue")
	validateKeyContents(t, c, testKey1, "testValue", true, true)

	// Read the value from the read only cache
	v, conflictv, s, _, err := cRO.GetValue(ctx, Key(testKey1), true)
	assert.NoErr(t, err)
	assert.Equal(t, *v, "testValue")
	assert.Equal(t, s, NoLockSentinel)
	assert.IsNil(t, conflictv)

	// Test read values from read only cache
	batch := []Key{Key(testKey1), Key(testKey2)}
	values, conflicts, sentinels, err := cRO.GetValues(ctx, batch, []bool{true, true})
	assert.NoErr(t, err)
	assert.Equal(t, len(values), 2)
	assert.NotNil(t, values[0], assert.Must())
	assert.Equal(t, *values[0], "testValue")
	assert.IsNil(t, values[1])
	assert.Equal(t, sentinels[0], NoLockSentinel)
	assert.Equal(t, sentinels[1], NoLockSentinel)
	assert.IsNil(t, conflicts[0])
	assert.IsNil(t, conflicts[1])

	// Make sure we can never take a lock on a read only cache
	s, err = cRO.WriteSentinel(ctx, Read, []Key{Key(testKey1)})
	assert.NoErr(t, err)
	assert.Equal(t, s, NoLockSentinel)

	s, err = cRO.WriteSentinel(ctx, Create, []Key{Key(testKey1)})
	assert.NoErr(t, err)
	assert.Equal(t, s, NoLockSentinel)

	s, err = cRO.WriteSentinel(ctx, Update, []Key{Key(testKey1)})
	assert.NoErr(t, err)
	assert.Equal(t, s, NoLockSentinel)

	s, err = cRO.WriteSentinel(ctx, Delete, []Key{Key(testKey1)})
	assert.NoErr(t, err)
	assert.Equal(t, s, NoLockSentinel)

	v, conflictv, s, _, err = cRO.GetValue(ctx, Key(testKey1), true)
	assert.NoErr(t, err)
	assert.Equal(t, *v, "testValue")
	assert.Equal(t, s, NoLockSentinel)
	assert.IsNil(t, conflictv)

	// Write the sentinel through RW client and validate that read only client can't release it or store values
	s, err = c.WriteSentinel(ctx, Read, []Key{Key(testKey1)})
	assert.NoErr(t, err)
	assert.NotEqual(t, s, NoLockSentinel)

	cRO.ReleaseSentinel(ctx, []Key{Key(testKey1)}, s)
	validateKeyContents(t, c, testKey1, string(s), true, true)

	set, conflict, err := cRO.SetValue(ctx, Key(testKey1), []Key{Key(testKey1)}, "testValue2", s, time.Minute)
	assert.NoErr(t, err)
	assert.False(t, set)
	assert.False(t, conflict)
	validateKeyContents(t, c, testKey1, string(s), true, true)

	err = cRO.Flush(ctx, "", false)
	assert.NoErr(t, err)
	validateKeyContents(t, c, testKey1, string(s), true, true)

	err = cRO.DeleteValue(ctx, []Key{Key(testKey1)}, false, true)
	assert.NoErr(t, err)
	validateKeyContents(t, c, testKey1, string(s), true, true)

	testValueKeys := []Key{"testValueKey1", "testValueKey2"}
	testValuesStrs := []string{"testValueKey1", "testValueKey2"}

	err = c.AddDependency(ctx, []Key{Key(testKey2)}, testValueKeys, time.Minute)
	assert.NoErr(t, err)
	validateDepKeyContents(t, c, testKey2, testValuesStrs, true)

	err = cRO.AddDependency(ctx, []Key{Key(testKey2)}, []Key{"BadVal"}, time.Minute)
	assert.NoErr(t, err)
	validateDepKeyContents(t, c, testKey2, testValuesStrs, true)

	err = cRO.ClearDependencies(ctx, Key(testKey2), false)
	assert.NoErr(t, err)
	validateDepKeyContents(t, c, testKey2, testValuesStrs, true)

	err = cRO.ClearDependencies(ctx, Key(testKey2), true)
	assert.NoErr(t, err)
	validateDepKeyContents(t, c, testKey2, testValuesStrs, true)
}
