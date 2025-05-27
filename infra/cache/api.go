package cache

import (
	"context"
	"net/http"
	"time"

	"github.com/goccy/go-json"
	"github.com/gofrs/uuid"

	"userclouds.com/infra"
	"userclouds.com/infra/cache/metrics"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctrace"
)

var tracer = uctrace.NewTracer("infra/cache/client")

// The client cache API is designed to support a consistent, write-through (i.e. values on create/update are written to the cache and a cache client is guaranteed
// read after write consistency). Each item in the cache is stored under its primary key (i.e. itemID -> itemValue). All operations are supported on the primary key.
// The consistency is guaranteed via an optimistic locking mechanism implemented using a sentinel value. The sentinel value is stored in the cache under the key at
// the start of the operation and is checked at the end of the operation. If the sentinel value matches, the result operation of the operation is stored in the cache.
// If the sentinel value does not match (ie the item was modified by another operation), different actions are taken depending on the new value of the primary key.
// The sentinel rules are described in cache_writethrough_sentinel_manager.go.
//
// Every item type that doesn't flush entire cache on create/update/delete also has a dependency list key. The dependency key contains a list of all keys that are in
// the cache that need to be invalidated if the item value changes. On post invalidation the dependency list is set to a tombstone value. The tombstone value blocks
// addition of new dependencies to the list. Failure to update the dependency list for one of the dependencies caused the value to not be stored in the cache. This is
// used to invalidate in flight reads that don't take a shared lock (like the primary key) with the update/delete/create operation. The tombstone expiration time is set
// to longest time a server side operation can take.

//
// The cache is also designed to support secondary keys for an item (ie. itemAlias -> itemID). For simplicity, the secondary keys are only used for Read operations
// (ie you can't update/delete an item using a secondary key). While the items are always stored under their primary key and secondary key(s) (ie the item is
// never stored under the secondary key alone), we can't guarantee that the primary key will always expire at same time as secondary key. We need to invalidate inflight
// reads via secondary keys (ie GetItemByName()), during delete operations via a primary key (our delete operation doesn't return the item that has been deleted so secondary key
// can't be calculated) and also invalidate value stored under secondary key prior to start of the delete. We handle it in two different ways depending on type of the item,
// depending on if the item has a dependency list. Items that don't have a dependency key this via a per type global collection key that is locked by every create/delete/update
// operation and by every read operation. The is most efficient for low change volume types like EdgeType and ObjectType. Items that have a dependency list,
// handle it by adding secondary key(s) value(s) to its own dependency list. This requires extra write/read from the dependency list but is more efficient for high change volume
// item types like Edge and Object.
//
// The cache support one global collection per item type. This collection is meant to contain every item of given type. It makes sense for items with low change volume
// and is invalidate on any create/delete/update operation to any item of that type. This is done by locking per type global collection key on every create/delete/update
// operation and by every read operation (read lock). The global collection key is also used as a marker for tracking follower reads but setting it to a tombstone value for a
// short Tombstone TTL post create/delete/update operation to any item of that type. That means for a tombstone TTL time period after the create/delete/update operation to any item
// all reads will go against master DB region holding the data. This is done to ensure that the follower reads don't return stale data. If global collection page caching is enabled,
// the global collection pages key is used to store set of the pages of the global collection that have been cached. In this case the follower reads tracking is done via a dedicated
// IsGlobalCollection modified key and the global collection key is used for locking/tombstones only. This allows pages of the global collection to be cached without tombstone TTL
// delay post create/delete/update operation to any item of that type as locks can be obtains on global collection key while IsGlobalCollection modified key is set to tombstone value.

// The cache supports any number of per item collection (ie. []Edges on Object). We use that functionality to store paths between two objects as well as edges.

// SingleItem is an interface for any single non array item that can be stored in the cache
// This interface also links the type (ObjectType, EdgeType, Object, Edge) with the cache key names for each type of use
type SingleItem interface {
	// GetPrimaryKey returns the primary cache key where the item is stored and which is used to lock the item
	GetPrimaryKey(KeyNameProvider) Key
	// GetSecondaryKeys returns any secondary keys which also contain the item for lookup by another dimension (ie TypeName, Alias, etc)
	GetSecondaryKeys(KeyNameProvider) []Key
	// GetGlobalCollectionKey returns the key for the collection of all items of this type (ie all ObjectTypes, all EdgeTypes, etc)
	GetGlobalCollectionKey(KeyNameProvider) Key
	// GetGlobalCollectionPagesKey returns the for storing the pages of the global collection
	GetGlobalCollectionPagesKey(KeyNameProvider) Key
	// GetPerItemCollectionKey returns the key for the collection of per item items of another type (ie Edges in/out of a specific Object)
	GetPerItemCollectionKey(KeyNameProvider) Key
	// GetDependenciesKey returns the key containing dependent keys that should invalidated if the item is invalidated
	GetDependenciesKey(KeyNameProvider) Key
	// GetDependencyKeys returns the list of keys for items this item depends on (ie Edge depends on both source and target objects)
	GetDependencyKeys(KeyNameProvider) []Key
	// GetIsModifiedKey returns the key containing a tombstone sentinel if the item has been modified in last TTL seconds
	GetIsModifiedKey(KeyNameProvider) Key
	// GetIsModifiedCollectionKey returns the key containing a tombstone sentinel if the global colleciton has been modified in last TTL seconds
	GetIsModifiedCollectionKey(KeyNameProvider) Key
	// TTL returns the TTL for the item
	TTL(TTLProvider) time.Duration
	// Validate method is used to validate the item. Every SingleItem is expected to implement the Validateable interface
	infra.Validateable
}

// RateLimitKey represents a key for storing rate limits in the cache
type RateLimitKey string

// RateLimitableItem is an interface for an item that can have an associated rate limit that is monitored in the cache
type RateLimitableItem interface {
	// GetRateLimitKeys returns the keys that should be retrieved when evaluating the rate limit for the item
	GetRateLimitKeys(KeyNameProvider) []RateLimitKey
	// GetLimit returns the maximum number of executions allowed for the rate limitable item
	GetRateLimit() int64
	// TTL returns the TTL for rate limit buckets for the item
	TTL(TTLProvider) time.Duration
	// Validate method is used to validate the item. Every RateLimitableItem is expected to implement the Validateable interface
	infra.Validateable
}

// InvalidationHandler is the type for a function that is called when the cache is invalidated
type InvalidationHandler func(ctx context.Context, key Key, flush bool) error

// Capabilities is the interface for expressing the capabilities of a cache provider
type Capabilities interface {
	// Layered returns true if the cache provider is a multi-layered cache
	Layered(context.Context) bool
	// SupportsRateLimits returns true if the cache provider supports rate limiting
	SupportsRateLimits(context.Context) bool
}

// Provider is the interface for the cache backend for a given tenant which can be implemented by in-memory, redis, memcache, etc
type Provider interface {
	Capabilities

	// GetValue gets the value in cache key (if any) and tries to lock the key for Read is lockOnMiss = true
	GetValue(ctx context.Context, key Key, lockOnMiss bool) (*string, *string, Sentinel, bool, error)
	// GetValues gets the value in cache key (if any) and tries to lock the key for Read is lockOnMiss = true
	GetValues(ctx context.Context, keys []Key, lockOnMiss []bool) ([]*string, []*string, []Sentinel, error)
	// SetValue sets the value in cache key(s) to val with given expiration time if the sentinel matches lkey and returns true if the value was set
	SetValue(ctx context.Context, lkey Key, keysToSet []Key, val string, sentinel Sentinel, ttl time.Duration) (bool, bool, error)
	// DeleteValue deletes the value(s) in passed in keys, force is true also deletes keys with sentinel or tombstone values
	DeleteValue(ctx context.Context, keys []Key, setTombstone bool, force bool) error
	// WriteSentinel writes the sentinel value into the given keys, returns NoLockSentinel if it couldn't acquire the lock
	WriteSentinel(ctx context.Context, stype SentinelType, keys []Key) (Sentinel, error)
	// ReleaseSentinel clears the sentinel value from the given keys
	ReleaseSentinel(ctx context.Context, keys []Key, s Sentinel)
	// AddDependency adds the given cache key(s) as dependencies of an item represented by by key. Fails if any of the dependency keys passed in contain tombstone
	AddDependency(ctx context.Context, keysIn []Key, dependentKey []Key, ttl time.Duration) error
	// ClearDependencies clears the dependencies of an item represented by key and removes all dependent keys from the cache
	ClearDependencies(ctx context.Context, key Key, setTombstone bool) error
	// Flush flushes the cache
	Flush(ctx context.Context, prefix string, flushTombstones bool) error
	// GetCacheName returns the global name of the cache if any
	GetCacheName(ctx context.Context) string
	// RegisterInvalidationHandler registers a handler to be called when the specified key is invalidated
	RegisterInvalidationHandler(ctx context.Context, handler InvalidationHandler, key Key) error
	// LogKeyValues is debugging only method that logs the values of the keys with the given prefix
	LogKeyValues(ctx context.Context, prefix string) error
	// ReleaseRateLimitSlot will release a rate limit slot
	ReleaseRateLimitSlot(ctx context.Context, keys []RateLimitKey) (int64, error)
	// ReserveRateLimitSlot will return whether a slot can be reserved based on the specified rate limit, actually reserving the slot if requested
	ReserveRateLimitSlot(ctx context.Context, keys []RateLimitKey, limit int64, ttl time.Duration, takeSlot bool) (bool, int64, error)
}

// NoLayeringProvider should be embedded in Provider implementations that do not support layering
type NoLayeringProvider struct{}

// Layered is from the Capabilities interface
func (NoLayeringProvider) Layered(context.Context) bool {
	return false
}

// NoRateLimitProvider should be embedded in Provider implementations that do not support rate limiting
type NoRateLimitProvider struct{}

// ReleaseRateLimitSlot is from the Provider interface
func (NoRateLimitProvider) ReleaseRateLimitSlot(context.Context, []RateLimitKey) (int64, error) {
	return 0, nil
}

// ReserveRateLimitSlot is from the Provider interface
func (NoRateLimitProvider) ReserveRateLimitSlot(context.Context, []RateLimitKey, int64, time.Duration, bool) (bool, int64, error) {
	return true, 1, nil
}

// SupportsRateLimits is from the Capabilities interface
func (NoRateLimitProvider) SupportsRateLimits(context.Context) bool {
	return false
}

// Manager is the bundle cache classes that are needed to interact with the cache
type Manager struct {
	N        KeyNameProvider
	Provider Provider
	T        TTLProvider
}

// NewManager returns a new CacheManager with given contents
func NewManager(p Provider, n KeyNameProvider, t TTLProvider) Manager {
	return Manager{N: n, Provider: p, T: t}
}

// KeyTTLID is the type for the ID used to identify the cache key TTL via CacheTTLProvider interface
type KeyTTLID string

// TTLProvider is the interface for the container that can provide per item cache TTLs
type TTLProvider interface {
	TTL(id KeyTTLID) time.Duration
}

// SkipCacheTTL is TTL set when cache is not used
const SkipCacheTTL time.Duration = 0

// KeyNameID is the type for the ID used to identify the cache key name via CacheKeyNameProvider interface
type KeyNameID string

// KeyNameProvider is the interface for the container that can provide cache names for cache keys that
// can be shared across different cache providers
type KeyNameProvider interface {
	GetKeyName(id KeyNameID, components []string) Key
	// GetKeyNameWithID is a wrapper around GetKeyName that converts the itemID to []string
	GetKeyNameWithID(id KeyNameID, itemID uuid.UUID) Key
	// GetKeyNameWithString is a wrapper around GetKeyName that converts the itemName to []string
	GetKeyNameWithString(id KeyNameID, itemName string) Key
	// GetKeyNameStatic is a wrapper around GetKeyName that passing in empty []string
	GetKeyNameStatic(id KeyNameID) Key
	// GetPrefix returns the prefix for the cache keys
	GetPrefix() string
	GetRateLimitKeyName(id KeyNameID, keySuffix string) RateLimitKey
	// GetAllKeyIDs returns all the key IDs that are used by the cache
	GetAllKeyIDs() []string
}

// NoRateLimitKeyNameProvider should be embedded in KeyNameProvider implementations that do not support rate limiting
type NoRateLimitKeyNameProvider struct{}

// GetRateLimitKeyName is from the KeyNameProvider interface
func (NoRateLimitKeyNameProvider) GetRateLimitKeyName(KeyNameID, string) RateLimitKey {
	return ""
}

// SentinelManager is the interface for managing cache sentinels to implement concurrency handling
type SentinelManager interface {
	GenerateSentinel(stype SentinelType) Sentinel
	CanAlwaysSetSentinel(newVal Sentinel) bool
	CanSetSentinelGivenCurrVal(currVal Sentinel, newVal Sentinel) bool
	CanSetValue(currVal string, val string, sentinel Sentinel) (set bool, clear bool, conflict bool, refresh bool)
	IsSentinelValue(val string) bool
}

// Flush flushes the cache
func (cm Manager) Flush(ctx context.Context, objType string) error {
	if err := cm.Provider.Flush(ctx, cm.N.GetPrefix(), false); err != nil {
		uclog.Errorf(ctx, "error flushing cache [%v] for %v: %v", cm.Provider, objType, err)
		return ucerr.Wrap(err)
	}
	return nil
}

// getItemLockKeys returns the keys to lock for the given item
func getItemLockKeys(lockType SentinelType, c KeyNameProvider, i SingleItem) []Key {
	keys := []Key{i.GetPrimaryKey(c)} // primary key is always first
	switch lockType {
	case Create:
		// Takes a lock if item does not exist, if read lock is in place
		// If write lock is in place, replaces it with new write lock
		if i.GetGlobalCollectionKey(c) != "" {
			keys = append(keys, i.GetGlobalCollectionKey(c))
		}
		keys = append(keys, i.GetSecondaryKeys(c)...)
	case Update:
		// Takes a write lock if item does not exist or if read lock is in place
		// Do not take a lock if a conflict or delete lock is in place
		// If write lock is in place, upgrade it to conflict lock
		if i.GetGlobalCollectionKey(c) != "" {
			keys = append(keys, i.GetGlobalCollectionKey(c))
		}
		keys = append(keys, i.GetSecondaryKeys(c)...)
	case Delete:
		// Takes all locks regardless of key state
		if i.GetGlobalCollectionKey(c) != "" {
			keys = append(keys, i.GetGlobalCollectionKey(c))
		}
		keys = append(keys, i.GetSecondaryKeys(c)...)
		if i.GetPerItemCollectionKey(c) != "" {
			keys = append(keys, i.GetPerItemCollectionKey(c))
		}
	case Read:
		// Only takes a read lock if the primary key is not set
	}
	return keys
}

// TakeItemLock takes a lock for the given item. Typically used for Create, Update, Delete operations on an item
func TakeItemLock[item SingleItem](ctx context.Context, lockType SentinelType, c Manager, i item) (Sentinel, error) {
	return uctrace.Wrap1(ctx, tracer, "TakeItemLock", true, func(ctx context.Context) (Sentinel, error) {
		return takeLockWorker(ctx, c, lockType, i, getItemLockKeys(lockType, c.N, i))
	})
}

// TakePerItemCollectionLock takes a lock for the collection associated with a given item
func TakePerItemCollectionLock[item SingleItem](ctx context.Context, lockType SentinelType, c Manager, additionalColKeys []Key, i item) (Sentinel, error) {
	return uctrace.Wrap1(ctx, tracer, "TakePerItemCollectionLock", true, func(ctx context.Context) (Sentinel, error) {
		if lockType != Delete && lockType != Read {
			return NoLockSentinel, ucerr.New("Unexpected lock type for collection lock")
		}

		// Lock the primary per item collection and any sub collections that are passed in
		keys := []Key{i.GetPerItemCollectionKey(c.N)}
		keys = append(keys, additionalColKeys...)

		return takeLockWorker(ctx, c, lockType, i, keys)
	})
}

// TakeGlobalCollectionLock takes a lock for the global collection associated with a given item type
func TakeGlobalCollectionLock[item SingleItem](ctx context.Context, lockType SentinelType, c Manager, i item) (Sentinel, error) {
	return uctrace.Wrap1(ctx, tracer, "TakeGlobalCollectionLock", true, func(ctx context.Context) (Sentinel, error) {
		if lockType != Delete && lockType != Read {
			return NoLockSentinel, ucerr.New("Unexpected lock type for global collection lock")
		}

		// Lock the global collection
		keys := []Key{i.GetGlobalCollectionKey(c.N)}

		return takeLockWorker(ctx, c, lockType, i, keys)
	})
}

func takeLockWorker[item SingleItem](ctx context.Context, c Manager, lockType SentinelType, i item, keys []Key) (Sentinel, error) {
	s := NoLockSentinel

	var err error

	// Create/Update:
	//  Takes a lock if item does not exist, if read lock is in place
	//  If write lock is in place, replaces it with new write lock
	//  when the write completes it resets the value in the cache if it is different from value that it wrote to the server or bump the lock to conflict
	// Delete:
	//  Takes all locks regardless of key state
	// Read:
	//  Takes lock only if key is empty or unlocked
	s, err = c.Provider.WriteSentinel(ctx, lockType, keys)

	// If we are deleting, clear the dependencies and tombstone the dependency key prior to starting the delete
	// to ensure that stale data is not returned after the server registers the delete
	if lockType == Delete && err == nil {
		if i.GetGlobalCollectionPagesKey(c.N) != "" {
			// We don't need to tombstone the global collection pages key since follower reads are tracked through isModified key
			if err := c.Provider.ClearDependencies(ctx, i.GetGlobalCollectionPagesKey(c.N), false); err != nil {
				uclog.Warningf(ctx, "Failed to clear global collection pages for key %v: %v", i.GetGlobalCollectionPagesKey(c.N), err)
			}
		}
		if i.GetDependenciesKey(c.N) != "" {
			err = c.Provider.ClearDependencies(ctx, i.GetDependenciesKey(c.N), true)
		}
	}

	// Return a friendly error to the user indicating that the call should be retried
	if err != nil {
		uclog.Warningf(ctx, "Failed to get a lock for keys %v of type %v with %v", keys, lockType, err)
		return NoLockSentinel, ucerr.Wrap(ucerr.WrapWithFriendlyStructure(jsonclient.Error{StatusCode: http.StatusConflict}, jsonclient.SDKStructuredError{
			Error: "Failed to get a cache lock due to contention. Please retry the call",
		}))
	}
	return s, nil
}

// ReleaseItemLock releases the lock for the given item
func ReleaseItemLock[item SingleItem](ctx context.Context, c Manager, lockType SentinelType, i item, sentinel Sentinel) {
	var span uctrace.Span
	ctx, span = tracer.StartSpan(ctx, "ReleaseItemLock", true)
	defer span.End()

	if sentinel == NoLockSentinel {
		return // nothing to clear if the lock wasn't acquired
	}

	keys := getItemLockKeys(lockType, c.N, i)

	c.Provider.ReleaseSentinel(ctx, keys, sentinel)
}

// ReleasePerItemCollectionLock releases the lock for the collection associated with a given item
func ReleasePerItemCollectionLock[item SingleItem](ctx context.Context, c Manager, additionalColKeys []Key, i item, sentinel Sentinel) {
	var span uctrace.Span
	ctx, span = tracer.StartSpan(ctx, "ReleasePerItemCollectionLock", true)
	defer span.End()

	if sentinel == NoLockSentinel {
		return // nothing to clear if the lock wasn't acquired
	}

	// Unlock the primary per item collection and any sub collections that are passed in
	keys := []Key{i.GetPerItemCollectionKey(c.N)}
	keys = append(keys, additionalColKeys...)

	c.Provider.ReleaseSentinel(ctx, keys, sentinel)
}

// GetItemsArrayFromCache gets the value stored in key from the cache. The value should be an array of items
func GetItemsArrayFromCache[item SingleItem](ctx context.Context, c Manager, key Key, lockOnMiss bool) (*[]item, Sentinel, Sentinel, bool, error) {
	var i item
	ttl := i.TTL(c.T)

	return uctrace.Wrap4(ctx, tracer, "GetItemsArrayFromCache", true, func(ctx context.Context) (*[]item, Sentinel, Sentinel, bool, error) {
		items, conflict, sentinel, partialHit, err := getItemFromCacheWorker[[]item](ctx, c, key, "", lockOnMiss, ttl)
		if err != nil || items == nil {
			return items, conflict, sentinel, partialHit, ucerr.Wrap(err)
		}

		for _, cachedItem := range *items {
			if !validateItem(ctx, "GetItemsArrayFromCache", c, cachedItem, key) {
				return nil, "", "", false, nil
			}
		}
		return items, conflict, sentinel, partialHit, ucerr.Wrap(err)
	})
}

// GetItemFromCache gets the the value stored in key from the cache. The value should be single item
func GetItemFromCache[item SingleItem](ctx context.Context, c Manager, key Key, lockOnMiss bool) (*item, Sentinel, Sentinel, error) {
	var i item
	ttl := i.TTL(c.T)

	return uctrace.Wrap3(ctx, tracer, "GetItemFromCache", true, func(ctx context.Context) (*item, Sentinel, Sentinel, error) {
		cachedItem, conflict, s, _, err := getItemFromCacheWorker[item](ctx, c, key, "", lockOnMiss, ttl)
		if err != nil || cachedItem == nil {
			return cachedItem, conflict, s, ucerr.Wrap(err)
		}

		if !validateItem(ctx, "GetItemFromCache", c, *cachedItem, key) {
			return nil, "", "", nil
		}

		return cachedItem, conflict, s, ucerr.Wrap(err)
	})
}

// GetItemFromCacheWithModifiedKey gets the the value stored in the key from the cache, while also returning the state of isModified key. The value should be single item
func GetItemFromCacheWithModifiedKey[item SingleItem](ctx context.Context, c Manager, key Key, isModifiedKey Key, lockOnMiss bool) (*item, Sentinel, Sentinel, error) {
	var i item
	ttl := i.TTL(c.T)

	return uctrace.Wrap3(ctx, tracer, "GetItemFromCache", true, func(ctx context.Context) (*item, Sentinel, Sentinel, error) {
		cachedItem, conflict, s, _, err := getItemFromCacheWorker[item](ctx, c, key, isModifiedKey, lockOnMiss, ttl)
		if err != nil || cachedItem == nil {
			return cachedItem, conflict, s, ucerr.Wrap(err)
		}

		if !validateItem(ctx, "GetItemFromCache", c, *cachedItem, key) {
			return nil, "", "", nil
		}

		return cachedItem, conflict, s, ucerr.Wrap(err)
	})
}

func validateItem[item SingleItem](ctx context.Context, apiName string, c Manager, cachedItem item, key Key) bool {
	if err := cachedItem.Validate(); err != nil {
		uclog.Errorf(ctx, "%s: Failed to validate item %v of type %T: %v", apiName, cachedItem, cachedItem, err)
		if err := c.Provider.DeleteValue(ctx, []Key{key}, false, true); err != nil {
			uclog.Warningf(ctx, "%s: Failed to delete keys %v from cache: %v", apiName, key, err)
		}
		return false
	}
	return true
}

func getItemFromCacheWorker[item any](ctx context.Context, c Manager, key Key, isModifiedKey Key, lockOnMiss bool, ttl time.Duration) (*item, Sentinel, Sentinel, bool, error) {
	if ttl == SkipCacheTTL {
		return nil, "", "", false, nil
	}

	var rawValue, conflictValue *string
	s := NoLockSentinel
	var err error
	partialHit := false

	start := time.Now().UTC()
	if isModifiedKey == "" {
		rawValue, conflictValue, s, partialHit, err = c.Provider.GetValue(ctx, key, lockOnMiss)
	} else {

		// If the cache is layered we don't want to try to fetch isModifiedkey from outer cache if
		// the value itself is preset in the inner cache (in that case isModified is unused)
		if c.Provider.Layered(ctx) {
			rawValue, conflictValue, s, partialHit, err = c.Provider.GetValue(ctx, key, lockOnMiss)
			lockOnMiss = false
		}
		if err != nil || rawValue == nil {
			rawValues, conflictValues, sentinels, errl := c.Provider.GetValues(ctx, []Key{key, isModifiedKey}, []bool{lockOnMiss, false})
			err = errl
			if err == nil && len(rawValues) == 2 && len(conflictValues) == 2 && len(sentinels) == 2 {
				rawValue = rawValues[0]
				conflictValue = conflictValues[1]
				if !c.Provider.Layered(ctx) {
					s = sentinels[0]
				}
			}
		}
	}
	took := time.Now().UTC().Sub(start)
	conflict := NoLockSentinel

	if err != nil {
		return nil, "", "", false, ucerr.Wrap(err)
	}
	if conflictValue != nil {
		conflict = Sentinel(*conflictValue)
	}
	if rawValue == nil {
		metrics.RecordCacheMiss(ctx, took)
		return nil, conflict, s, false, nil
	}

	var loadedItem item

	if err := json.Unmarshal([]byte(*rawValue), &loadedItem); err != nil {
		uclog.Errorf(ctx, "GetItemFromCache: Failed to unmarshal data %v for item of type %T from cache: %v", rawValue, loadedItem, err)
		return nil, conflict, "", partialHit, nil
	}
	metrics.RecordCacheHit(ctx, took)
	return &loadedItem, conflict, "", partialHit, nil
}

// GetItemsFromCache gets the the values stored in keys from the cache.
func GetItemsFromCache[item SingleItem](ctx context.Context, c Manager, keys []Key, mkeys []Key, locksOnMiss []bool) ([]*item, []Sentinel, bool, error) {
	return uctrace.Wrap3(ctx, tracer, "GetItemsFromCache", true, func(ctx context.Context) ([]*item, []Sentinel, bool, error) {
		var i item
		if ttl := i.TTL(c.T); ttl == SkipCacheTTL {
			return nil, nil, true, nil
		}

		start := time.Now().UTC()
		values, _, sentinels, err := c.Provider.GetValues(ctx, keys, locksOnMiss)
		took := time.Now().UTC().Sub(start)
		if err != nil {
			return nil, nil, true, ucerr.Wrap(err)
		}
		items := make([]*item, len(keys))
		hits, misses := 0, 0
		for i, rawValue := range values {
			if rawValue == nil {
				metrics.RecordCacheMiss(ctx, took)
				items[i] = nil
				misses++
			} else {
				var loadedItem item
				if err := json.Unmarshal([]byte(*rawValue), &loadedItem); err != nil {
					// Should we do something else when we fail to unmarshal ?
					uclog.Errorf(ctx, "GetItemsFromCache: Failed to unmarshal data %v for item of type %T from cache: %v", rawValue, loadedItem, err)
					return nil, nil, true, ucerr.Wrap(err)
				}
				if validateItem(ctx, "GetItemsFromCache", c, loadedItem, keys[i]) {
					hits++
					items[i] = &loadedItem
				} else {
					items[i] = nil
					misses++
				}
			}
		}
		metrics.RecordMultiGet(ctx, hits, misses, took)
		dirty := false
		if misses > 0 {
			// If we have to retrieve any items from the DB check if they have been recently modified
			// We don't retrieve modified keys at the same time as values so that we can reduce off machine
			// cache calls in cases where all values are in the on machine keys
			noLocks := make([]bool, len(mkeys))
			start := time.Now().UTC()
			_, conflictValues, _, err := c.Provider.GetValues(ctx, mkeys, noLocks)
			took := time.Now().UTC().Sub(start)
			metrics.RecordCacheHit(ctx, took) // TODO need better way to reason about hit/miss here
			if err != nil {
				return nil, nil, dirty, ucerr.Wrap(err)
			}

			for _, sentinelValue := range conflictValues {
				if sentinelValue != nil && IsTombstoneSentinel(*sentinelValue) {
					dirty = true
					break
				}
			}
		}

		return items, sentinels, dirty, nil
	})
}

// DeleteItemFromCache deletes the values stored in key associated with the item from the cache.
func DeleteItemFromCache[item SingleItem](ctx context.Context, c Manager, i item, sentinel Sentinel) {
	var span uctrace.Span
	ctx, span = tracer.StartSpan(ctx, "DeleteItemFromCache", true)
	defer span.End()

	if sentinel == NoLockSentinel {
		return // nothing to clear if the lock wasn't acquired
	}

	keys := getItemLockKeys(Delete, c.N, i)

	if i.GetIsModifiedKey(c.N) != "" {
		keys = append(keys, i.GetIsModifiedKey(c.N))
	}
	if i.GetIsModifiedCollectionKey(c.N) != "" {
		keys = append(keys, i.GetIsModifiedCollectionKey(c.N))
	}

	if err := c.Provider.DeleteValue(ctx, keys, true, true); err != nil {
		uclog.Warningf(ctx, "Failed to delete keys %v from cache: %v", keys, err)
	}
}

// SaveItemToCache saves the given item to the cache
func SaveItemToCache[item SingleItem](ctx context.Context, c Manager, i item, sentinel Sentinel,
	clearCollection bool, additionalColKeys []Key) {
	var span uctrace.Span
	ctx, span = tracer.StartSpan(ctx, "SaveItemToCache", true)
	defer span.End()

	saveItemToCacheWorker(ctx, c, i, i.GetPrimaryKey(c.N), sentinel, clearCollection, additionalColKeys)
}

// SaveItemsFromCollectionToCache saves the items from a given collection into their separate keys
func SaveItemsFromCollectionToCache[item SingleItem](ctx context.Context, c Manager, items []item, sentinel Sentinel) {
	var span uctrace.Span
	ctx, span = tracer.StartSpan(ctx, "SaveItemsFromCollectionToCache", true)
	defer span.End()

	for _, i := range items {
		saveItemToCacheWorker(ctx, c, i, i.GetGlobalCollectionKey(c.N), sentinel, false, nil)
	}
}

func saveItemToCacheWorker[item SingleItem](ctx context.Context, c Manager, i item, lkey Key, sentinel Sentinel,
	clearCollection bool, additionalColKeys []Key) {
	if i.TTL(c.T) == SkipCacheTTL {
		return
	}

	if sentinel == NoLockSentinel {
		return // no need to do work if we don't have the sentinel
	}

	if b, err := json.Marshal(i); err == nil {
		keyNames := []Key{}
		keyNames = append(keyNames, i.GetSecondaryKeys(c.N)...)
		keyNames = append(keyNames, i.GetPrimaryKey(c.N))
		keyset, _, err := c.Provider.SetValue(ctx, lkey, keyNames, string(b), sentinel, i.TTL(c.T))
		if err != nil {
			uclog.Errorf(ctx, "Error saving item to cache [%v]: %v", c.Provider, err)
		}
		// Clear all the collections that this item might appear in. This is needed for create/update operations that might change the collection
		ckeys := []Key{}
		clearKeysOnError := false
		if clearCollection /* && !conflict - we can't skip clearing on conflict due on machine caches */ {
			// Check if there is a default global collection for all items of this type and it is being used directly for follower reads
			if i.GetGlobalCollectionKey(c.N) != "" && i.GetIsModifiedCollectionKey(c.N) == "" {
				ckeys = append(ckeys, i.GetGlobalCollectionKey(c.N))
			}
			// Put tombstone into isModified keys to disable follower reads
			if i.GetIsModifiedKey(c.N) != "" {
				ckeys = append(ckeys, i.GetIsModifiedKey(c.N))
			}
			if i.GetIsModifiedCollectionKey(c.N) != "" {
				ckeys = append(ckeys, i.GetIsModifiedCollectionKey(c.N))
			}

			// Check if there are any additional collections that this item might appear in passed in by the caller
			if len(additionalColKeys) > 0 {
				ckeys = append(ckeys, additionalColKeys...)
			}
			if err := c.Provider.DeleteValue(ctx, ckeys, true, true /* force delete regardless of value */); err != nil {
				uclog.Errorf(ctx, "Error clearing collection keys from cache [%v]: %v", c.Provider, err)
				clearKeysOnError = true
				keyset = false
			}

			// Check if there is a dependency list for this item (only needed on update to clear secondary collections)
			if i.GetDependenciesKey(c.N) != "" {
				if err := c.Provider.ClearDependencies(ctx, i.GetDependenciesKey(c.N), false); err != nil {
					uclog.Errorf(ctx, "Error clearing dependencies %v from cache [%v]: %v", i.GetDependenciesKey(c.N), c.Provider, err)
					clearKeysOnError = true
					keyset = false
				}
			}
			if i.GetGlobalCollectionPagesKey(c.N) != "" {
				if err := c.Provider.ClearDependencies(ctx, i.GetGlobalCollectionPagesKey(c.N), false); err != nil {
					uclog.Errorf(ctx, "Error clearing pages of global collection %v from cache [%v]: %v", i.GetGlobalCollectionPagesKey(c.N), c.Provider, err)
				}
			}
			uclog.Verbosef(ctx, "Cleared collection keys %v from cache", ckeys)
		}

		depKeys := i.GetDependencyKeys(c.N)
		if len(depKeys) > 0 && keyset {
			if err := c.Provider.AddDependency(ctx, depKeys, keyNames, i.TTL(c.T)); err != nil {
				uclog.Warningf(ctx, "Failed to add dependency %v to key %v: %v", keyNames, depKeys, err)
				clearKeysOnError = true
				keyset = false
			}

		}
		if selfDepKey := i.GetDependenciesKey(c.N); selfDepKey != "" && keyset && len(i.GetSecondaryKeys(c.N)) != 0 {
			if err := c.Provider.AddDependency(ctx, []Key{selfDepKey}, i.GetSecondaryKeys(c.N), i.TTL(c.T)); err != nil {
				// This may fail if the item was deleted between where we stored it in the primary/secondary keys and here
				uclog.Debugf(ctx, "Failed to add secondary key dependency %v to key %v: %v", i.GetSecondaryKeys(c.N), selfDepKey, err)
				clearKeysOnError = true
			}
		}
		// Cache is still in consistent state in this case, we just failed to add the cache the item to do contention
		if clearKeysOnError {
			if err := c.Provider.DeleteValue(ctx, keyNames, true, true); err != nil {
				uclog.Warningf(ctx, "Failed to delete secondary key after dependency failure %v: %v", i.GetSecondaryKeys(c.N), err)
			}
		}
	}
}

// SaveItemsToCollection saves the given collection to collection key associated with the item or global to item type
// If this is a per item collection than "item" argument is the item with with the collection is associated and "cItems" is the collection
// to be stored.
func SaveItemsToCollection[item SingleItem, cItem SingleItem](ctx context.Context, c Manager,
	i item, colItems []cItem, lockKey Key, colKey Key, sentinel Sentinel, isGlobal bool) {

	ttl := i.TTL(c.T)

	var span uctrace.Span
	ctx, span = tracer.StartSpan(ctx, "SaveItemsToCollection", true)
	defer span.End()

	if ttl == SkipCacheTTL {
		return
	}

	if colKey == "" || lockKey == "" {
		return // error condition
	}

	if sentinel == NoLockSentinel {
		return // no need to do work if we don't have the sentinel
	}

	if b, err := json.Marshal(colItems); err == nil {

		saveCollection := true

		// For non-global collections, get a list of items this collection depends on so that can add our collection key to their dependencies list
		if !isGlobal {
			dependentItems := map[Key]bool{}
			dependentKeys := make([]Key, 0, len(colItems))

			for _, ci := range colItems {
				depKeys := ci.GetDependencyKeys(c.N)
				for _, depKey := range depKeys {
					if !dependentItems[depKey] && depKey != "" {
						dependentItems[depKey] = true
						dependentKeys = append(dependentKeys, depKey)
					}
				}
				depKey := ci.GetDependenciesKey(c.N)
				if depKey != "" && !dependentItems[depKey] { // Some items can't be individually deleted/updated so they have no dependencies key
					dependentKeys = append(dependentKeys, depKey)
				}
			}
			// Don't cache the collection if it has too many dependencies
			if len(dependentKeys) > 100 /* TODO figure out the optimal number */ {
				return
			}

			if i.GetDependenciesKey(c.N) != "" && !dependentItems[i.GetDependenciesKey(c.N)] {
				dependentKeys = append(dependentKeys, i.GetDependenciesKey(c.N))
			}

			// We write the collection key into the dependency lists of items it depends on before saving it/
			// That way we save the collection if and only if all the lists are updated successfully.
			if len(dependentKeys) > 0 {
				if err := c.Provider.AddDependency(ctx, dependentKeys, []Key{colKey}, i.TTL(c.T)); err != nil {
					uclog.Warningf(ctx, "Didn't cache collection failed to add dependency %v to key %v: %v", dependentKeys, colKey, err)
					saveCollection = false
				}
			}
		} else if i.GetGlobalCollectionPagesKey(c.N) != "" && colKey != i.GetGlobalCollectionKey(c.N) {
			// Check if this is a page of a global collection vs the whole thing
			if err := c.Provider.AddDependency(ctx, []Key{i.GetGlobalCollectionPagesKey(c.N)}, []Key{colKey}, i.TTL(c.T)); err != nil {
				uclog.Warningf(ctx, "Didn't cache global collection page failed to add dependency %v to key %v: %v", i.GetGlobalCollectionPagesKey(c.N), colKey, err)
				saveCollection = false
			}
		}
		// If we don't save the collection the cache is still in a consistent state - we just don't cache the collection
		if saveCollection {
			if r, _, err := c.Provider.SetValue(ctx, lockKey, []Key{colKey}, string(b), sentinel, ttl); err == nil && r {
				uclog.Verbosef(ctx, "Saved collection %v to cache", colKey)
			}
		}
	}
}
