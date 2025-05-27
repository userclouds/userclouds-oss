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

	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/test/testlogtransport"
)

// testRateLimitItem implements the RateLimitableItem interface
type testRateLimitItem struct {
	id        uuid.UUID
	rateLimit int64
	minBucket int
	maxBucket int
}

func newTestRateLimitItem(rateLimit int64, minBucket int, maxBucket int) testRateLimitItem {
	return testRateLimitItem{
		id:        uuid.Must(uuid.NewV4()),
		rateLimit: rateLimit,
		minBucket: minBucket,
		maxBucket: maxBucket,
	}
}

func (testRateLimitItem) retryPause() int {
	return 100 + rand.Intn(6)*7
}

func (trli testRateLimitItem) Validate() error {
	if trli.id.IsNil() {
		return ucerr.Errorf("id must be non-nil: %v", trli)
	}

	if trli.rateLimit < 1 {
		return ucerr.Errorf("rateLimit must be greater than or equal to 1: %v", trli)
	}

	if trli.minBucket < 0 {
		return ucerr.Errorf("minBucket must be non-negative: %v", trli)
	}

	if trli.maxBucket < trli.minBucket {
		return ucerr.Errorf("maxBucket must be greater than or equal to MinBucket: %v", trli)
	}

	return nil
}

func (trli testRateLimitItem) GetRateLimitKeys(knp KeyNameProvider) []RateLimitKey {
	var keys []RateLimitKey
	for i := trli.minBucket; i <= trli.maxBucket; i++ {
		keys = append(keys, knp.GetRateLimitKeyName("testRateLimitItem", fmt.Sprintf("%v_%d", trli.id, i)))
	}
	return keys
}

func (trli testRateLimitItem) GetRateLimit() int64 {
	return trli.rateLimit
}

func (testRateLimitItem) TTL(TTLProvider) time.Duration {
	return time.Minute * 5
}

// testItem implements the SingleItem interface
type testItem struct {
	ID    uuid.UUID
	Name  string
	Items []testItem
}

// GetPrimaryKey returns the primary cache key name for testitem
func (ti testItem) GetPrimaryKey(knp KeyNameProvider) Key {
	return knp.GetKeyNameWithID(KeyNameID("testItem"), ti.ID)

}

// GetGlobalCollectionKey returns the global collection cache key names for testitem
func (ti testItem) GetGlobalCollectionKey(knp KeyNameProvider) Key {
	return knp.GetKeyNameStatic(KeyNameID("testItemCOL"))
}

// GetGlobalCollectionPagesKey returns the global collection key name for testItem
func (ti testItem) GetGlobalCollectionPagesKey(c KeyNameProvider) Key {
	return ""
}

// GetPerItemCollectionKey returns the per item collection key name for testitem
func (ti testItem) GetPerItemCollectionKey(knp KeyNameProvider) Key {
	return ti.GetPrimaryKey(knp)
}

// GetDependenciesKey return  dependencies cache key name for testitem
func (ti testItem) GetDependenciesKey(knp KeyNameProvider) Key {
	return knp.GetKeyNameWithID(KeyNameID("TestItemDEP"), ti.ID)
}

// GetIsModifiedKey returns the isModifiedKey key name for testitem
func (ti testItem) GetIsModifiedKey(knp KeyNameProvider) Key {
	return knp.GetKeyNameWithID(KeyNameID("TestItemMod"), ti.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for testitem
func (testItem) GetIsModifiedCollectionKey(c KeyNameProvider) Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for testitem dependencies
func (ti testItem) GetDependencyKeys(knp KeyNameProvider) []Key {
	keys := []Key{}
	for _, item := range ti.Items {
		keys = append(keys, item.GetDependenciesKey(knp))
	}
	return keys
}

// GetSecondaryKeys returns the secondary cache key names for testitem
func (ti testItem) GetSecondaryKeys(knp KeyNameProvider) []Key {
	return []Key{knp.GetKeyNameWithString(KeyNameID("testItemName"), ti.Name)}
}

// TTL returns the TTL for testitem
func (testItem) TTL(TTLProvider) time.Duration {
	return time.Minute * 5
}

func (ti testItem) Validate() error {
	if strings.Contains(strings.ToLower(ti.Name), "happy festivus") {
		return ucerr.Errorf("Festivus for the rest of us! - %s", ti.Name)
	}
	return nil
}

// testNameProv implements the KeyNameProvider interface
type testNameProv struct {
	prefix string
}

func (testNameProv) GetAllKeyIDs() []string {
	return []string{"testItem", "testItemCOL", "TestItemDEP", "TestItemMod", "testItemName, testRateLimitItem"}
}

func (tnp testNameProv) GetKeyName(id KeyNameID, components []string) Key {
	if len(components) == 0 {
		return Key(fmt.Sprintf("%v_%v", tnp.prefix, id))
	}
	return Key(fmt.Sprintf("%v_%v_%v", tnp.prefix, id, components[0]))
}
func (tnp testNameProv) GetKeyNameWithID(id KeyNameID, itemID uuid.UUID) Key {
	return tnp.GetKeyName(id, []string{itemID.String()})
}
func (tnp testNameProv) GetKeyNameWithString(id KeyNameID, itemName string) Key {
	return tnp.GetKeyName(id, []string{itemName})
}
func (tnp testNameProv) GetKeyNameStatic(id KeyNameID) Key {
	return tnp.GetKeyName(id, []string{})
}

func (tnp testNameProv) GetRateLimitKeyName(id KeyNameID, keySuffix string) RateLimitKey {
	return RateLimitKey(fmt.Sprintf("%v_%v_%v", tnp.prefix, "RATELIMIT", keySuffix))
}

// GetPrefix returns the base prefix for all keys
func (tnp testNameProv) GetPrefix() string {
	return tnp.prefix
}

// testTTLProv implements the TTLProvider interface
type testTTLProv struct{}

func (testTTLProv) TTL(KeyTTLID) time.Duration {
	return time.Minute * 5
}

func testTakeItemLockSerial[client Provider](ctx context.Context, t *testing.T, c client) {
	tN := testNameProv{}
	cm := NewManager(c, tN, testTTLProv{})
	sm := NewWriteThroughCacheSentinelManager()

	// Set up an item with one dependency
	depKeyName := Key("placeholder" + uuid.Must(uuid.NewV4()).String())
	item := testItem{ID: uuid.Must(uuid.NewV4()), Name: uuid.Must(uuid.NewV4()).String(), Items: []testItem{{ID: uuid.Must(uuid.NewV4()), Name: "testItem1"}}}
	err := c.AddDependency(ctx, []Key{item.GetDependenciesKey(cm.N)}, []Key{depKeyName}, time.Minute)
	assert.NoErr(t, err)
	setKeyValue(ctx, t, c, depKeyName, "testVal")

	s, err := TakeItemLock(ctx, Read, cm, item)
	assert.NoErr(t, err)
	assert.True(t, sm.IsReadSentinelPrefix(s))
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), "", false, true)
	validateDepKeyContents(t, c, string(item.GetDependenciesKey(tN)), []string{string(depKeyName)}, true)
	s1, err := TakeItemLock(ctx, Read, cm, item)
	assert.NoErr(t, err)
	assert.Equal(t, s1, NoLockSentinel)
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), "", false, true)
	validateDepKeyContents(t, c, string(item.GetDependenciesKey(tN)), []string{string(depKeyName)}, true)
	s, err = TakeItemLock(ctx, Create, cm, item)
	assert.NoErr(t, err)
	assert.True(t, sm.IsWriteSentinelPrefix(s))
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), string(s), true, true)
	validateDepKeyContents(t, c, string(item.GetDependenciesKey(tN)), []string{string(depKeyName)}, true)
	s1, err = TakeItemLock(ctx, Create, cm, item)
	assert.NoErr(t, err)
	assert.True(t, sm.IsWriteSentinelPrefix(s1))
	assert.NotEqual(t, s1, s)
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s1), true, true)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(s1), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), string(s1), true, true)
	validateDepKeyContents(t, c, string(item.GetDependenciesKey(tN)), []string{string(depKeyName)}, true)
	validateKeyContents(t, c, string(depKeyName), "testVal", true, true)
	s1, err = TakeItemLock(ctx, Delete, cm, item)
	assert.NoErr(t, err)
	assert.True(t, sm.IsDeleteSentinelPrefix(s1))
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s1), true, true)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(s1), true, true)
	validateKeyContents(t, c, string(item.GetPerItemCollectionKey(tN)), string(s1), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), string(s1), true, true)
	validateKeyContents(t, c, string(item.GetDependenciesKey(tN)), string(GenerateTombstoneSentinel()), true, true)
	validateKeyContents(t, c, string(depKeyName), "", false, true)
	s, err = TakeItemLock(ctx, Read, cm, item)
	assert.NoErr(t, err)
	assert.Equal(t, s, NoLockSentinel)
	s, err = TakeItemLock(ctx, Create, cm, item)
	assert.NoErr(t, err)
	assert.Equal(t, s, NoLockSentinel)
	s, err = TakeItemLock(ctx, Update, cm, item)
	assert.NoErr(t, err)
	assert.Equal(t, s, NoLockSentinel)
	s, err = TakeItemLock(ctx, Delete, cm, item)
	assert.NoErr(t, err)
	assert.Equal(t, s, NoLockSentinel)
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s1), true, true)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(s1), true, true)
	validateKeyContents(t, c, string(item.GetPerItemCollectionKey(tN)), string(s1), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), string(s1), true, true)

	// Make sure we don't overwrite the tombstone in global collection on item operations
	item = testItem{ID: uuid.Must(uuid.NewV4()), Name: uuid.Must(uuid.NewV4()).String(), Items: []testItem{{ID: uuid.Must(uuid.NewV4()), Name: "testItem1"}}}
	setKeyValueDirect(ctx, t, c, item.GetGlobalCollectionKey(tN), string(GenerateTombstoneSentinel()))

	s, err = TakeItemLock(ctx, Read, cm, item)
	assert.NoErr(t, err)
	assert.True(t, sm.IsReadSentinelPrefix(s))
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), "", false, true)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(GenerateTombstoneSentinel()), true, true)
	validateKeyContents(t, c, string(item.GetIsModifiedKey(tN)), "", false, true)
	SaveItemToCache(ctx, cm, item, s, false, nil)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(GenerateTombstoneSentinel()), true, true)
	validateKeyContents(t, c, string(item.GetIsModifiedKey(tN)), "", false, true)
	itemOut, conflict, s, err := GetItemFromCache[testItem](ctx, cm, item.GetPrimaryKey(cm.N), false)
	assert.NoErr(t, err)
	assert.Equal(t, s, NoLockSentinel)
	assert.Equal(t, conflict, NoLockSentinel)
	assert.NotNil(t, itemOut, assert.Must())
	assert.Equal(t, itemOut.ID, item.ID)

	// Make sure we set the item tombstone on create operations
	item = testItem{ID: uuid.Must(uuid.NewV4()), Name: uuid.Must(uuid.NewV4()).String(), Items: []testItem{{ID: uuid.Must(uuid.NewV4()), Name: "testItem1"}}}
	setKeyValueDirect(ctx, t, c, item.GetGlobalCollectionKey(tN), string(GenerateTombstoneSentinel()))

	s, err = TakeItemLock(ctx, Create, cm, item)
	assert.NoErr(t, err)
	assert.True(t, sm.IsWriteSentinelPrefix(s))
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), string(s), true, true)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(GenerateTombstoneSentinel()), true, true)
	validateKeyContents(t, c, string(item.GetIsModifiedKey(tN)), "", false, true)
	SaveItemToCache(ctx, cm, item, s, true, nil)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(GenerateTombstoneSentinel()), true, true)
	validateKeyContents(t, c, string(item.GetIsModifiedKey(tN)), string(GenerateTombstoneSentinel()), true, true)
	itemOut, conflict, s, err = GetItemFromCacheWithModifiedKey[testItem](ctx, cm, item.GetPrimaryKey(cm.N), item.GetIsModifiedKey(cm.N), false)
	assert.NoErr(t, err)
	assert.Equal(t, s, NoLockSentinel)

	// This needed to work around the fact that we don't read isModified key if there is a
	// value in the "key" in layered cache to prevent always hitting outer cache. TODO
	// think of better solution to this
	if !c.Layered(ctx) {
		assert.True(t, IsTombstoneSentinel(string(conflict)))
	}
	assert.NotNil(t, itemOut, assert.Must())
	assert.Equal(t, itemOut.ID, item.ID)

	// Make sure we set the item tombstone on update operations
	item = testItem{ID: uuid.Must(uuid.NewV4()), Name: uuid.Must(uuid.NewV4()).String(), Items: []testItem{{ID: uuid.Must(uuid.NewV4()), Name: "testItem1"}}}
	setKeyValueDirect(ctx, t, c, item.GetGlobalCollectionKey(tN), string(GenerateTombstoneSentinel()))

	s, err = TakeItemLock(ctx, Update, cm, item)
	assert.NoErr(t, err)
	assert.True(t, sm.IsWriteSentinelPrefix(s))
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), string(s), true, true)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(GenerateTombstoneSentinel()), true, true)
	validateKeyContents(t, c, string(item.GetIsModifiedKey(tN)), "", false, true)
	SaveItemToCache(ctx, cm, item, s, true, nil)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(GenerateTombstoneSentinel()), true, true)
	validateKeyContents(t, c, string(item.GetIsModifiedKey(tN)), string(GenerateTombstoneSentinel()), true, true)

	itemOut, conflict, s, err = GetItemFromCacheWithModifiedKey[testItem](ctx, cm, item.GetPrimaryKey(cm.N), item.GetIsModifiedKey(cm.N), false)
	assert.NoErr(t, err)
	assert.Equal(t, s, NoLockSentinel)
	if !c.Layered(ctx) {
		assert.True(t, IsTombstoneSentinel(string(conflict)))
	}
	assert.NotNil(t, itemOut, assert.Must())
	assert.Equal(t, itemOut.ID, item.ID)

	// Make sure we set the item tombstone on delete operations
	item = testItem{ID: uuid.Must(uuid.NewV4()), Name: uuid.Must(uuid.NewV4()).String(), Items: []testItem{{ID: uuid.Must(uuid.NewV4()), Name: "testItem1"}}}
	setKeyValueDirect(ctx, t, c, item.GetGlobalCollectionKey(tN), string(GenerateTombstoneSentinel()))

	s, err = TakeItemLock(ctx, Delete, cm, item)
	assert.NoErr(t, err)
	assert.True(t, sm.IsDeleteSentinelPrefix(s))
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), string(s), true, true)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(GenerateTombstoneSentinel()), true, true)
	validateKeyContents(t, c, string(item.GetIsModifiedKey(tN)), "", false, true)
	DeleteItemFromCache(ctx, cm, item, s)
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(GenerateTombstoneSentinel()), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), string(GenerateTombstoneSentinel()), true, true)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(GenerateTombstoneSentinel()), true, true)
	validateKeyContents(t, c, string(item.GetIsModifiedKey(tN)), string(GenerateTombstoneSentinel()), true, true)

	itemOut, conflict, s, err = GetItemFromCacheWithModifiedKey[testItem](ctx, cm, item.GetPrimaryKey(cm.N), item.GetIsModifiedKey(cm.N), false)
	assert.NoErr(t, err)
	assert.Equal(t, s, NoLockSentinel)
	assert.True(t, IsTombstoneSentinel(string(conflict)))
	assert.IsNil(t, itemOut)
}

func validateDeletedOrInDeletion(ctx context.Context, t *testing.T, c Provider, tN KeyNameProvider, i *testItem) {
	t.Helper()
	sm := NewWriteThroughCacheSentinelManager()

	// The keys can be either blank or contain a delete sentinel
	keys := []Key{i.GetPrimaryKey(tN), i.GetGlobalCollectionKey(tN), i.GetPerItemCollectionKey(tN), i.GetSecondaryKeys(tN)[0]}
	for _, k := range keys {
		if found, val := getKeyValue(ctx, t, c, k, true); found &&
			!sm.IsDeleteSentinelPrefix(Sentinel(val)) &&
			!sm.IsReadSentinelPrefix(Sentinel(val)) &&
			!IsTombstoneSentinel(val) {
			assert.True(t, sm.IsDeleteSentinelPrefix(Sentinel(val)))
		}
	}
	// The dependency key should be tombstoned
	if found, val := getKeyValue(ctx, t, c, i.GetDependenciesKey(tN), true); !found || !IsTombstoneSentinel(val) {
		validateKeyContents(t, c, string(i.GetDependenciesKey(tN)), string(GenerateTombstoneSentinel()), true, true)
	}
}

func testTakeItemLockMultiThreaded[client Provider](ctx context.Context, t *testing.T, c client) {
	tN := testNameProv{}
	cm := NewManager(c, tN, testTTLProv{})

	// Set up an item with one dependency
	item := &testItem{ID: uuid.Must(uuid.NewV4()), Name: uuid.Must(uuid.NewV4()).String(), Items: []testItem{{ID: uuid.Must(uuid.NewV4()), Name: "testItem1"}}}
	depKey := Key(uuid.Must(uuid.NewV4()).String())
	assert.NoErr(t, c.AddDependency(ctx, []Key{item.GetDependenciesKey(cm.N)}, []Key{depKey}, time.Minute))
	setKeyValue(ctx, t, c, depKey, "testVal")
	itemLock := sync.Mutex{}
	itemOldP := item

	threadCount := 20
	wg := sync.WaitGroup{}
	// 	Have threadCount threads read the values and confirm they come back as expected
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()

			for range 100 {
				var err error
				var lItem testItem
				itemLock.Lock()
				if item == nil {
					// Validate the deletion of the keys for item from previous loop iteration
					validateDeletedOrInDeletion(ctx, t, c, tN, itemOldP)
					// The dependency key should be cleared
					validateKeyContents(t, c, string(depKey), "", false, true)
					// Create a new item for next loop iteration
					item = &testItem{ID: uuid.Must(uuid.NewV4()), Name: uuid.Must(uuid.NewV4()).String(), Items: []testItem{{ID: uuid.Must(uuid.NewV4()), Name: "testItem1"}}}
					depKey = Key(uuid.Must(uuid.NewV4()).String())
					assert.NoErr(t, c.AddDependency(ctx, []Key{item.GetDependenciesKey(cm.N)}, []Key{depKey}, time.Minute))
					setKeyValue(ctx, t, c, depKey, "testVal")
				}
				// Create a local copy of the item which will not be reset by another thread that does the deletion
				lItem = *item
				itemLock.Unlock()

				// Create in flight reads that should overlap with Delete on other threads
				// At least one of these will succeed prior to the Delete and some of the rest will be in flight
				s := NoLockSentinel
				retry := true
				for retry {
					retry = false
					s, err = TakeItemLock(ctx, Read, cm, lItem)
					if err != nil && jsonclient.IsHTTPStatusConflict(err) {
						retry = true
					}
				}
				assert.NoErr(t, err)
				saveItem := false
				itemLock.Lock()
				if item != nil && item.ID == lItem.ID {
					saveItem = true
				}
				itemLock.Unlock()
				if saveItem {
					SaveItemToCache(ctx, cm, &lItem, s, false, nil)
				}

				// Now delete the item (only one thread gets to actually reset it)
				retry = true
				for retry {
					retry = false
					s, err = TakeItemLock(ctx, Delete, cm, lItem)
					if err != nil && jsonclient.IsHTTPStatusConflict(err) {
						retry = true
					}
				}
				assert.NoErr(t, err)

				itemLock.Lock()
				if item != nil && item.ID == lItem.ID {
					itemOldP = item
					item = nil
				}
				itemLock.Unlock()

				ReleaseItemLock(ctx, cm, Delete, lItem, s)
			}
		}(i)
	}
	wg.Wait()
}

func testTakeCollectionLockMultiThreaded[client Provider](ctx context.Context, t *testing.T, c client) {
	tN := testNameProv{}
	cm := NewManager(c, tN, testTTLProv{})
	sm := NewWriteThroughCacheSentinelManager()

	// Set up an item with one dependency
	var item *testItem
	depKey := Key(uuid.Must(uuid.NewV4()).String())
	itemLock := sync.Mutex{}
	itemOldP := item

	threadCount := 5
	wg := sync.WaitGroup{}
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()

			for range 10 {
				var err error
				var lItem testItem
				var ldepKey Key
				itemLock.Lock()
				if item == nil {
					if itemOldP != nil {
						// Validate the deletion of the keys for item from previous loop iteration
						validateDeletedOrInDeletion(ctx, t, c, tN, itemOldP)
						// The dependency key should be cleared or held by read lock (which should fail to set it)
						found, val := getKeyValue(ctx, t, c, depKey, true)
						if found && !sm.IsReadSentinelPrefix(Sentinel(val)) {
							assert.True(t, sm.IsReadSentinelPrefix(Sentinel(val)))
						}
					}
					// Create a new item for next loop iteration
					item = &testItem{ID: uuid.Must(uuid.NewV4()), Name: uuid.Must(uuid.NewV4()).String(), Items: []testItem{{ID: uuid.Must(uuid.NewV4())}}}
					depKey = Key(uuid.Must(uuid.NewV4()).String())
				}
				// Create a local copy of the item which will not be reset by another thread that does the deletion
				lItem = *item
				ldepKey = depKey
				itemLock.Unlock()

				// Create in flight reads that should overlap with Delete on other threads
				// At least one of these will succeed prior to the Delete and some of the rest will be in flight
				s, err := TakePerItemCollectionLock(ctx, Read, cm, []Key{ldepKey}, lItem)
				assert.NoErr(t, err)
				saveItem := false
				itemLock.Lock()
				if item != nil && item.ID == lItem.ID {
					saveItem = true
				}
				itemLock.Unlock()
				if saveItem {
					citem := testItem{ID: uuid.Must(uuid.NewV4()), Name: uuid.Must(uuid.NewV4()).String(), Items: []testItem{{ID: uuid.Must(uuid.NewV4())}}}
					SaveItemsToCollection(ctx, cm, &lItem, []testItem{citem}, ldepKey, ldepKey, s, false)
				}
				ReleasePerItemCollectionLock(ctx, cm, []Key{ldepKey}, lItem, s)
				// Now delete the item (only one thread gets to actually reset it)
				retry := true
				for retry {
					retry = false
					s, err = TakeItemLock(ctx, Delete, cm, lItem)
					if err != nil && jsonclient.IsHTTPStatusConflict(err) {
						retry = true
					}
				}
				assert.NoErr(t, err)

				itemLock.Lock()
				if item != nil && item.ID == lItem.ID {
					itemOldP = item
					item = nil
				}
				itemLock.Unlock()

				ReleaseItemLock(ctx, cm, Delete, lItem, s)
			}
		}(i)
	}
	wg.Wait()
}

func getRandomIDs(itemsIDs []uuid.UUID, itemCount int) []uuid.UUID {
	ids := set.NewUUIDSet()
	for ids.Size() < itemCount {
		ids.Insert(itemsIDs[rand.Intn(len(itemsIDs))])
	}
	return ids.Items()
}

func saveItemToCache(ctx context.Context, t *testing.T, cm Manager, item testItem) {
	t.Helper()
	sentinel, err := TakeItemLock(ctx, Create, cm, item)
	assert.NoErr(t, err, assert.Errorf("Error locking item %+v", item))
	assert.NotEqual(t, sentinel, NoLockSentinel, assert.Must(), assert.Errorf("Failed to take lock for item %+v", item))
	SaveItemToCache(ctx, cm, item, sentinel, false, nil)
}

func testGetValuesMultiThreaded[client Provider](ctx context.Context, t *testing.T, c client, cachePrefix string) {
	const itemCount = 30
	rand.New(rand.NewSource(time.Now().Unix()))
	const threadsCount = 10
	cm := NewManager(c, testNameProv{prefix: cachePrefix}, testTTLProv{})
	itemsMap := make(map[uuid.UUID]*testItem)
	itemsIDs := make([]uuid.UUID, 0, itemCount)
	for i := range itemCount {
		id := uuid.Must(uuid.NewV4())
		item := testItem{ID: id, Name: fmt.Sprintf("testItem-%d-%v", i, id)}
		saveItemToCache(ctx, t, cm, item)
		itemsMap[item.ID] = &item
		itemsIDs = append(itemsIDs, item.ID)
	}

	// Now add a bunch of IDs that are not cached
	for range 30 {
		itemsIDs = append(itemsIDs, uuid.Must(uuid.NewV4()))
	}
	// Sanity check, everything behaves as expected
	getAndVerifyItems(ctx, t, itemsIDs, cm, itemsMap)

	wg := sync.WaitGroup{}
	wg.Add(threadsCount)
	for i := range threadsCount {
		go func(threadID int) {
			defer wg.Done()
			for range 1 {
				ic := rand.Intn(7) + 3 // between 3 and 10
				ids := getRandomIDs(itemsIDs, ic)
				getAndVerifyItems(ctx, t, ids, cm, itemsMap)
			}
		}(i)
	}
	wg.Wait()
}

func getAndVerifyItems(ctx context.Context, t *testing.T, itemIDs []uuid.UUID, cm Manager, itemsMap map[uuid.UUID]*testItem) {
	keys := make([]Key, 0, len(itemIDs))
	mkeys := make([]Key, 0, len(itemIDs))
	for _, id := range itemIDs {
		keys = append(keys, cm.N.GetKeyNameWithID(KeyNameID("testItem"), id))
		mkeys = append(mkeys, cm.N.GetKeyNameWithID(KeyNameID("testItemMod"), id))
	}
	// Add a few item IDs that we will always lock (since other misses might already be locked by other threads)
	mustLockItems := set.NewUUIDSet()
	for range 4 {
		id := uuid.Must(uuid.NewV4())
		keys = append(keys, cm.N.GetKeyNameWithID(KeyNameID("testItem"), id))
		itemIDs = append(itemIDs, id)
		mustLockItems.Insert(id)
	}

	locks := getLocks(len(itemIDs), true)
	cachedItems, sentinels, dirty, err := GetItemsFromCache[testItem](ctx, cm, keys, mkeys, locks)
	assert.NoErr(t, err)
	assert.Equal(t, len(cachedItems), len(itemIDs))
	assert.Equal(t, len(sentinels), len(itemIDs))
	assert.Equal(t, dirty, false)
	for i, itemID := range itemIDs {
		item, ok := itemsMap[itemID]
		if ok {
			// verify cache hit
			assert.Equal(t, sentinels[i], NoLockSentinel)
			ci := cachedItems[i]
			assert.Equal(t, ci.ID, item.ID, assert.Errorf("Item ID mismatch"))
			assert.Equal(t, ci.Name, item.Name, assert.Errorf("Item Name mismatch"))
			assert.Equal(t, len(ci.Items), 0, assert.Errorf("nested items is not empty"))
		} else {
			// verify cache miss
			assert.IsNil(t, cachedItems[i])
			if mustLockItems.Contains(itemID) {
				assert.NotEqual(t, sentinels[i], NoLockSentinel, assert.Errorf("%v items should be locked", itemID))
			}
		}
	}
}

func getLocks(count int, value bool) []bool {
	locks := make([]bool, 0, count)
	for range count {
		locks = append(locks, value)
	}
	return locks
}

func testGetItemFromCacheValidation[client Provider](ctx context.Context, t *testing.T, c client) {
	tt := testlogtransport.InitLoggerAndTransportsForTests(t)
	cm := NewManager(c, testNameProv{}, testTTLProv{})
	item := testItem{ID: uuid.Must(uuid.NewV4()),
		// Invalid name, will fail validation when loading
		Name: fmt.Sprintf("testItem-Happy Festivus-%v", uuid.Must(uuid.NewV4())),
	}
	saveItemToCache(ctx, t, cm, item)
	key := cm.N.GetKeyNameWithID(KeyNameID("testItem"), (item.ID))
	ci, _, sentinel, err := GetItemFromCache[testItem](ctx, cm, key, true)
	assert.IsNil(t, ci)
	assert.Equal(t, sentinel, NoLockSentinel)
	assert.NoErr(t, err)
	tt.AssertMessagesByLogLevel(uclog.LogLevelError, 1) // Make sure we log the validation error
	validateNonExistingKey(ctx, t, c, key)
}

func testGetItemsArrayFromCacheValidation[client Provider](ctx context.Context, t *testing.T, c client) {
	tt := testlogtransport.InitLoggerAndTransportsForTests(t)
	cm := NewManager(c, testNameProv{}, testTTLProv{})
	items := []testItem{
		{
			ID: uuid.Must(uuid.NewV4()),
			// Invalid name, will fail validation when loading
			Name: fmt.Sprintf("testItem-Happy Festivus-%v", uuid.Must(uuid.NewV4())),
		},
		{
			ID: uuid.Must(uuid.NewV4()),
			// Valid name/item
			Name: fmt.Sprintf("testItem-%v", uuid.Must(uuid.NewV4())),
		},
	}
	ckey := cm.N.GetKeyNameStatic(KeyNameID("testItemCollection"))
	// Make the collection unique so there is no interference with other tests
	ckey = Key(fmt.Sprintf("%v_%v", ckey, uuid.Must(uuid.NewV4())))
	ci, _, sentinel, _, err := GetItemsArrayFromCache[testItem](ctx, cm, ckey, true) // just calling so we can get the lock
	assert.IsNil(t, ci, assert.Errorf("Should not have found items"), assert.Must())
	assert.NoErr(t, err)
	assert.NotEqual(t, sentinel, NoLockSentinel, assert.Errorf("Should have gotten a lock"), assert.Must())
	SaveItemsToCollection(ctx, cm, testItem{}, items, ckey, ckey, sentinel, true)

	ci, _, sentinel, _, err = GetItemsArrayFromCache[testItem](ctx, cm, ckey, true)
	assert.IsNil(t, ci)
	assert.Equal(t, sentinel, NoLockSentinel)
	assert.NoErr(t, err)
	tt.AssertMessagesByLogLevel(uclog.LogLevelError, 1) // Make sure we log the validation error
	validateNonExistingKey(ctx, t, c, ckey)
}

func testGetItemsFromCacheValidation[client Provider](ctx context.Context, t *testing.T, c client) {
	tt := testlogtransport.InitLoggerAndTransportsForTests(t)
	cm := NewManager(c, testNameProv{}, testTTLProv{})
	items := []testItem{
		{
			ID: uuid.Must(uuid.NewV4()),
			// Valid name/item
			Name: fmt.Sprintf("testItem-%v", uuid.Must(uuid.NewV4())),
		},
		{
			ID: uuid.Must(uuid.NewV4()),
			// Invalid name, will fail validation when loading
			Name: fmt.Sprintf("testItem-Happy Festivus-%v", uuid.Must(uuid.NewV4())),
		},
		{
			ID: uuid.Must(uuid.NewV4()),
			// Valid name/item
			Name: fmt.Sprintf("testItem-%v", uuid.Must(uuid.NewV4())),
		},
		{
			ID: uuid.Must(uuid.NewV4()),
			// Invalid name, will fail validation when loading
			Name: fmt.Sprintf("HAPPY FESTIVUS-%v", uuid.Must(uuid.NewV4())),
		},
		{
			ID: uuid.Must(uuid.NewV4()),
			// Valid name/item
			Name: fmt.Sprintf("testItem-%v", uuid.Must(uuid.NewV4())),
		},
	}
	ids := []uuid.UUID{}
	keys := []Key{}
	mkeys := []Key{}
	fakeItemID := uuid.Must(uuid.NewV4())
	for _, item := range items {
		saveItemToCache(ctx, t, cm, item)
		ids = append(ids, item.ID)
		keys = append(keys, cm.N.GetKeyNameWithID(KeyNameID("testItem"), item.ID))
		mkeys = append(mkeys, cm.N.GetKeyNameWithID(KeyNameID("testItemMod"), item.ID))
	}
	ids = append(ids, fakeItemID) // Add a non-existent ID
	keys = append(keys, cm.N.GetKeyNameWithID(KeyNameID("testItem"), fakeItemID))

	cachedItems, sentinels, dirty, err := GetItemsFromCache[testItem](ctx, cm, keys, mkeys, getLocks(len(ids), true))
	assert.NoErr(t, err)
	assert.Equal(t, len(cachedItems), len(ids))
	assert.Equal(t, len(sentinels), len(ids))
	assert.Equal(t, dirty, false)

	tt.AssertMessagesByLogLevel(uclog.LogLevelError, 2) // Two validation error for the objects that failed validation
	// items 0, 2 & 4 - valid and should be cached
	for _, i := range []int{0, 2, 4} {
		assert.NotNil(t, cachedItems[i], assert.Errorf("Should have found item %v", i))
		assert.Equal(t, *cachedItems[i], items[i], assert.Errorf("Item %v mismatch", i))
		assert.Equal(t, sentinels[i], NoLockSentinel, assert.Errorf("Item %v should not be locked", i))
	}
	// items 1 &3 - where cached, but invalid, so we should not load them, should not lock them and should have deleted them
	for _, i := range []int{1, 3} {
		assert.IsNil(t, cachedItems[i], assert.Errorf("Should not have found item %v", i))
		assert.Equal(t, sentinels[i], NoLockSentinel, assert.Errorf("Item %v should not be locked", i))
		validateNonExistingKey(ctx, t, c, keys[i])

	}
	// last item - not cached, should not load it, but we should lock it
	assert.IsNil(t, cachedItems[5])
	assert.NotEqual(t, sentinels[5], NoLockSentinel)
}

func testGetAPIForTombstoneSentinel[client Provider](ctx context.Context, t *testing.T, c client, tombstoneTTL time.Duration) {
	tN := testNameProv{}
	cm := NewManager(c, tN, testTTLProv{})
	sm := NewTombstoneCacheSentinelManager()

	// Set up an item with one dependency
	depKeyName := Key("placeholder" + uuid.Must(uuid.NewV4()).String())
	item := testItem{ID: uuid.Must(uuid.NewV4()), Name: uuid.Must(uuid.NewV4()).String(), Items: []testItem{{ID: uuid.Must(uuid.NewV4()), Name: "testItem1"}}}
	err := c.AddDependency(ctx, []Key{item.GetDependenciesKey(cm.N)}, []Key{depKeyName}, time.Minute)
	assert.NoErr(t, err)
	setKeyValue(ctx, t, c, depKeyName, "testVal")

	// Do basic read of item (empty cache)
	s, err := TakeItemLock(ctx, Read, cm, item)
	assert.NoErr(t, err)
	assert.True(t, sm.IsReadSentinelPrefix(s))
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), "", false, true)
	validateDepKeyContents(t, c, string(item.GetDependenciesKey(tN)), []string{string(depKeyName)}, true)
	// Repeat the read with the cache populated
	s1, err := TakeItemLock(ctx, Read, cm, item)
	assert.NoErr(t, err)
	assert.Equal(t, s1, NoLockSentinel)
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), "", false, true)
	validateDepKeyContents(t, c, string(item.GetDependenciesKey(tN)), []string{string(depKeyName)}, true)

	// Emulate a creation call sequence
	s, err = TakeItemLock(ctx, Create, cm, item)
	assert.NoErr(t, err)
	assert.True(t, IsTombstoneSentinel(string(s)))
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), string(s), true, true)
	validateDepKeyContents(t, c, string(item.GetDependenciesKey(tN)), []string{string(depKeyName)}, true)
	SaveItemToCache(ctx, cm, item, s, false, nil)
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), string(s), true, true)
	validateDepKeyContents(t, c, string(item.GetDependenciesKey(tN)), []string{string(depKeyName)}, true)
	// Validate that we can't read after create
	s, err = TakeItemLock(ctx, Read, cm, item)
	assert.NoErr(t, err)
	assert.Equal(t, s, NoLockSentinel)

	item = testItem{ID: uuid.Must(uuid.NewV4()), Name: uuid.Must(uuid.NewV4()).String(), Items: []testItem{{ID: uuid.Must(uuid.NewV4()), Name: "testItem1"}}}
	err = c.AddDependency(ctx, []Key{item.GetDependenciesKey(cm.N)}, []Key{depKeyName}, time.Minute)
	assert.NoErr(t, err)

	// Emulate a update call sequence
	s, err = TakeItemLock(ctx, Update, cm, item)
	assert.NoErr(t, err)
	assert.True(t, IsTombstoneSentinel(string(s)))
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), string(s), true, true)
	validateDepKeyContents(t, c, string(item.GetDependenciesKey(tN)), []string{string(depKeyName)}, true)
	SaveItemToCache(ctx, cm, item, s, false, nil)
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), string(s), true, true)
	validateDepKeyContents(t, c, string(item.GetDependenciesKey(tN)), []string{string(depKeyName)}, true)
	// Validate that we can't read after update
	s, err = TakeItemLock(ctx, Read, cm, item)
	assert.NoErr(t, err)
	assert.Equal(t, s, NoLockSentinel)
	// Wait for tombstone to expire
	time.Sleep(tombstoneTTL)
	// Validate that we can read after tombstone expires
	s, err = TakeItemLock(ctx, Read, cm, item)
	assert.NoErr(t, err)
	assert.True(t, sm.IsReadSentinelPrefix(s))

	item = testItem{ID: uuid.Must(uuid.NewV4()), Name: uuid.Must(uuid.NewV4()).String(), Items: []testItem{{ID: uuid.Must(uuid.NewV4()), Name: "testItem1"}}}
	err = c.AddDependency(ctx, []Key{item.GetDependenciesKey(cm.N)}, []Key{depKeyName}, time.Minute)
	assert.NoErr(t, err)

	// Emulate a delete call sequence
	s, err = TakeItemLock(ctx, Delete, cm, item)
	assert.NoErr(t, err)
	assert.True(t, IsTombstoneSentinel(string(s)))
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), string(s), true, true)
	validateKeyContents(t, c, string(item.GetDependenciesKey(tN)), string(s), true, true)
	SaveItemToCache(ctx, cm, item, s, false, nil)
	validateKeyContents(t, c, string(item.GetPrimaryKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetGlobalCollectionKey(tN)), string(s), true, true)
	validateKeyContents(t, c, string(item.GetSecondaryKeys(tN)[0]), string(s), true, true)
	validateKeyContents(t, c, string(item.GetDependenciesKey(tN)), string(s), true, true)
	// Validate that we can't read after delete
	s, err = TakeItemLock(ctx, Read, cm, item)
	assert.NoErr(t, err)
	assert.Equal(t, s, NoLockSentinel)

}
