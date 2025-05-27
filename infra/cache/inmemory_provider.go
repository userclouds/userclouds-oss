package cache

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

const (
	defaultCacheTTL time.Duration = 5 * time.Minute
	gcInterval      time.Duration = 5 * time.Minute
)

const memSentinelTTL = 70 * time.Second

// InMemoryClientCacheProvider is the base implementation of the CacheProvider interface
type InMemoryClientCacheProvider struct {
	NoLayeringProvider
	NoRateLimitProvider
	cache        *cache.Cache
	keysMutex    sync.Mutex
	sm           SentinelManager
	cacheName    string
	tombstoneTTL time.Duration
}

type optionsInMem struct {
	sm SentinelManager
}

// OptionInMem specifies optional arguement for InMemoryClientCacheProvider
type OptionInMem interface {
	apply(*optionsInMem)
}

type optFuncInMem func(*optionsInMem)

func (o optFuncInMem) apply(opts *optionsInMem) {
	o(opts)
}

// SentinelManagerInMem allows to specify a custom sentinel manager
func SentinelManagerInMem(sm SentinelManager) OptionInMem {
	return optFuncInMem(func(opts *optionsInMem) {
		opts.sm = sm
	})
}

// NewInMemoryClientCacheProvider creates a new InMemoryClientCacheProvider
func NewInMemoryClientCacheProvider(cacheName string, opts ...OptionInMem) *InMemoryClientCacheProvider {
	var options optionsInMem
	for _, opt := range opts {
		opt.apply(&options)
	}

	sm := options.sm
	if sm == nil {
		sm = NewWriteThroughCacheSentinelManager()
	}
	// TODO - Underlying library treats 0 and -1 as no expiration, so we need to set it to minimum value and not look up items from the cache, work around that
	cacheEdges := cache.New(defaultCacheTTL, gcInterval)

	return &InMemoryClientCacheProvider{
		cache:        cacheEdges,
		sm:           sm,
		cacheName:    cacheName,
		tombstoneTTL: InvalidationTombstoneTTL,
	}
}

// inMemMultiSet sets all passed in keys to the same value with given TTL. Locking is left up to caller
func (c *InMemoryClientCacheProvider) inMemMultiSet(keys []string, value string, ifNotExists bool, ttl time.Duration) bool {
	r := true
	for _, key := range keys {
		if ifNotExists {
			if _, found := c.cache.Get(key); found {
				r = false
				continue
			}
		}
		c.cache.Set(key, value, ttl)
	}
	return r
}

// inMemMultiGet gets all passed in keys. Locking is left up to caller
func (c *InMemoryClientCacheProvider) inMemMultiGet(keys []string) []string {
	values := make([]string, len(keys))
	for i, key := range keys {
		if x, found := c.cache.Get(key); found {
			if val, ok := x.(string); ok {
				values[i] = val
			} else {
				values[i] = ""
			}
		}
	}
	return values
}

// imMemMultiDelete deletes all passed in keys. Locking is left up to caller
func (c *InMemoryClientCacheProvider) inMemMultiDelete(keys []string) {
	for _, key := range keys {
		c.cache.Delete(key)
	}
}

// inMemMultiExpire sets expiration time on passed in keys to given TTL. Locking is left up to caller
func (c *InMemoryClientCacheProvider) inMemMultiExpire(keys []string, ttl time.Duration) bool {
	r := true
	for _, key := range keys {
		if value, found := c.cache.Get(key); found {
			c.cache.Set(key, value, ttl)
		}
	}
	return r
}

// getStringKeysFromCacheKeys filters out any empty keys and does the type conversion
func (c *InMemoryClientCacheProvider) getStringKeysFromCacheKeys(keys []Key) []string {
	strKeys := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != "" {
			strKeys = append(strKeys, string(k))
		}
	}
	return strKeys
}

// WriteSentinel writes the sentinel value into the given keys
func (c *InMemoryClientCacheProvider) WriteSentinel(ctx context.Context, stype SentinelType, keysIn []Key) (Sentinel, error) {
	sentinel := c.sm.GenerateSentinel(stype)
	keys := c.getStringKeysFromCacheKeys(keysIn)
	if len(keys) == 0 {
		return NoLockSentinel, ucerr.New("Expected at least one key passed to WriteSentinel")
	}

	c.keysMutex.Lock()
	defer c.keysMutex.Unlock()

	// If we are doing a delete we can always take a lock (even in case of other deletes)
	if !c.sm.CanAlwaysSetSentinel(sentinel) {
		// Check if the primary key for the operation is already locked
		if x, found := c.cache.Get(keys[0]); found {
			if value, ok := x.(string); ok {
				// If the key is already locked and see if we have precedence
				if c.sm.IsSentinelValue(value) {
					if !c.sm.CanSetSentinelGivenCurrVal(Sentinel(value), sentinel) {
						return NoLockSentinel, nil
					}
				}
			}
		}

		// First make sure that we don't overwrite a tombstone and extend its TTL since it may otherwise expire during the operation
		// leaving the key unlocked
		newKeys := make([]string, 0, len(keys))
		newKeys = append(newKeys, keys[0])
		for i := 1; i < len(keys); i++ {
			if x, found := c.cache.Get(keys[i]); found {
				if value, ok := x.(string); ok && !IsTombstoneSentinel(value) {
					newKeys = append(newKeys, keys[i])
				} else if stype != Read {
					// If the key is a tombstone, refresh its TTL
					c.cache.Set(keys[i], value, c.tombstoneTTL)
				}
			} else {
				newKeys = append(newKeys, keys[i])
			}
		}
		keys = newKeys
	}
	// If key is not found, doesn't have a lock, or our lock has precedence we can set it
	c.inMemMultiSet(keys, string(sentinel), false, memSentinelTTL)

	return sentinel, nil
}

// ReleaseSentinel clears the sentinel value from the given keys
func (c *InMemoryClientCacheProvider) ReleaseSentinel(ctx context.Context, keysIn []Key, s Sentinel) {
	keys := c.getStringKeysFromCacheKeys(keysIn)
	if len(keys) == 0 {
		return
	}

	c.keysMutex.Lock()
	defer c.keysMutex.Unlock()

	values := c.inMemMultiGet(keys)

	keysToClear := make([]string, 0, len(keys))
	for i, v := range values {
		if v != "" && v == string(s) {
			keysToClear = append(keysToClear, keys[i])
		}
	}
	if len(keysToClear) > 0 {
		c.inMemMultiDelete(keysToClear)
	}

}

// SetValue sets the value in cache key(s) to val with given expiration time if the sentinel matches and returns true if the value was set
func (c *InMemoryClientCacheProvider) SetValue(ctx context.Context, lkeyIn Key, keysToSet []Key, val string,
	sentinel Sentinel, ttl time.Duration) (bool, bool, error) {
	keys := c.getStringKeysFromCacheKeys(keysToSet)
	if len(keys) == 0 {
		return false, false, ucerr.New("Expected at least one key passed to SetValue")
	}

	lkey := string(lkeyIn)
	if lkey == "" {
		return false, false, ucerr.New("Expected at least one key passed to SetValue")
	}

	c.keysMutex.Lock()
	defer c.keysMutex.Unlock()

	if x, found := c.cache.Get(lkey); found {
		if cV, ok := x.(string); ok {
			set, clear, conflict, refresh := c.sm.CanSetValue(cV, val, sentinel)
			if set {
				// The sentinel is still in the key which means nothing interrupted the operation and value can be safely stored in the cache
				uclog.Verbosef(ctx, "Cache[%v] set key %v", c.cacheName, keys)
				c.inMemMultiSet(keys, val, false, ttl)
				return true, false, nil
			} else if clear {
				uclog.Verbosef(ctx, "Cache[%v] cleared on value mismatch or conflict sentinel key %v curr val %v would store %v", c.cacheName, keys, cV, val)
				c.inMemMultiDelete(keys)
				return false, false, nil
			} else if conflict {
				c.inMemMultiSet(keys, cV+string(sentinel), false, memSentinelTTL)
				uclog.Verbosef(ctx, "Cache[%v] lock upgraded to conflict on write collision %v got %v added %v", c.cacheName, lkey, cV, sentinel)
				return false, true, nil
			} else if refresh {
				c.inMemMultiExpire(keys, c.tombstoneTTL)
				uclog.Verbosef(ctx, "Cache[%v] keys expiration TTL reset - %v got %v had %v", c.cacheName, keys, cV, sentinel)
				return false, false, nil
			}

			uclog.Verbosef(ctx, "Cache[%v] not set key %v on sentinel mismatch got %v expect %v", c.cacheName, lkey, cV, sentinel)
			return false, true, nil
		}
	}
	uclog.Verbosef(ctx, "Cache[%v] not set key %v on sentinel %v key not found", c.cacheName, lkey, sentinel)
	return false, false, nil
}

// GetValues gets the values in keys (if any) and tries to lock the key[i] for Read is lockOnMiss[i] = true
func (c *InMemoryClientCacheProvider) GetValues(ctx context.Context, keys []Key, lockOnMiss []bool) ([]*string, []*string, []Sentinel, error) {
	if len(keys) == 0 && len(lockOnMiss) == 0 {
		uclog.Errorf(ctx, "Cache[%v] GetValues called with no keys", c.cacheName)
		return nil, nil, nil, nil
	}
	if len(keys) != len(lockOnMiss) {
		return nil, nil, nil, ucerr.Errorf("Number of keys provided to GetValues has to be equal to number of lockOnMiss, keys: %d lockOnMiss: %d", len(keys), len(lockOnMiss))
	}
	val := make([]*string, len(keys))
	sentinels := make([]Sentinel, len(keys))
	conflicts := make([]*string, len(keys))

	for i := range sentinels {
		sentinels[i] = NoLockSentinel
	}

	keysToGet := make(map[Key]int)

	// Since we do this inmemory there is no roundtrip cost,  so we can do just loop and get each value
	for i, k := range keys {
		if _, ok := keysToGet[k]; !ok {
			v, conflict, s, _, err := c.GetValue(ctx, k, lockOnMiss[i])
			if err != nil {
				return val, conflicts, sentinels, ucerr.Wrap(err)
			}
			val[i] = v
			conflicts[i] = conflict
			sentinels[i] = s
			keysToGet[k] = i // save the index for the key
		} else {
			// Duplicate key so copy the value from the first instance
			val[i] = val[keysToGet[k]]
			conflicts[i] = conflicts[keysToGet[k]]
			sentinels[i] = sentinels[keysToGet[k]]
		}
	}
	return val, conflicts, sentinels, nil
}

// GetValue gets the value in CacheKey (if any) and tries to lock the key for Read is lockOnMiss = true
func (c *InMemoryClientCacheProvider) GetValue(ctx context.Context, keyIn Key, lockOnMiss bool) (*string, *string, Sentinel, bool, error) {
	key := string(keyIn)
	if key == "" {
		return nil, nil, NoLockSentinel, false, ucerr.New("Expected at least one key passed to GetValue")
	}

	c.keysMutex.Lock()
	defer c.keysMutex.Unlock()

	x, found := c.cache.Get(key)

	if !found {
		if lockOnMiss {
			sentinel := c.sm.GenerateSentinel(Read)
			if r := c.inMemMultiSet([]string{key}, string(sentinel), true, memSentinelTTL); r {
				uclog.Verbosef(ctx, "Cache[%v] miss key %v sentinel set %v", c.cacheName, key, sentinel)
				return nil, nil, Sentinel(sentinel), false, nil
			}
		}
		uclog.Verbosef(ctx, "Cache[%v] miss key %v no lock requested", c.cacheName, key)
		return nil, nil, NoLockSentinel, false, nil
	}

	if value, ok := x.(string); ok {
		if c.sm.IsSentinelValue(value) {
			uclog.Verbosef(ctx, "Cache[%v] key %v is locked or tombstoned for in progress op %v", c.cacheName, key, value)
			return nil, &value, NoLockSentinel, false, nil
		}

		uclog.Verbosef(ctx, "Cache[%v] hit key %v", c.cacheName, key)
		return &value, nil, NoLockSentinel, false, nil
	}

	return nil, nil, NoLockSentinel, false, nil
}

// DeleteValue deletes the value(s) in passed in keys
func (c *InMemoryClientCacheProvider) DeleteValue(ctx context.Context, keysIn []Key, setTombstone bool, force bool) error {
	setTombstone = setTombstone && c.tombstoneTTL > 0 // don't actually set tombstone if tombstoneTTL is 0
	keys := c.getStringKeysFromCacheKeys(keysIn)

	tombstoneValue := GenerateTombstoneSentinel() // Generate a unique tombstone value

	c.keysMutex.Lock()
	defer c.keysMutex.Unlock()

	if force {
		// Delete or tombstone regardless of value
		if setTombstone {
			c.inMemMultiSet(keys, string(tombstoneValue), false, c.tombstoneTTL)
			uclog.Verbosef(ctx, "Cache[%v] tombstoned keys %v", c.cacheName, keys)
		} else {
			c.inMemMultiDelete(keys)
			uclog.Verbosef(ctx, "Cache[%v] deleted keys %v", c.cacheName, keys)
		}
	} else {
		// Delete only unlocked keys
		for _, k := range keys {
			if x, found := c.cache.Get(k); found {
				if v, ok := x.(string); ok {
					if c.sm.IsSentinelValue(v) {
						// Skip locked key
						continue
					}
				}
				if setTombstone {
					c.cache.Set(k, string(tombstoneValue), c.tombstoneTTL)
				} else {
					c.cache.Delete(k)
				}
			}
		}
	}
	return nil
}

func (c *InMemoryClientCacheProvider) saveKeyArray(dkeys []string, newKeys []string, ttl time.Duration) error {

	for _, dkey := range dkeys {
		var keyToSet []string
		keyToSet = append(keyToSet, newKeys...)
		if x, found := c.cache.Get(dkey); found {
			keyNames, ok := x.([]string)
			if ok {
				for _, keyName := range keyNames {
					if _, found := c.cache.Get(keyName); found {
						keyToSet = append(keyToSet, keyName)
					}
				}
			} else {
				val, ok := x.(string)
				if !ok || IsTombstoneSentinel(val) {
					return ucerr.New("Can't add dependency: key is tombstoned")
				}
			}
		}
		// Remove duplicates to minimize the length
		keyMap := make(map[string]bool, len(keyToSet))
		uniqueKeysToSet := make([]string, 0, len(keyToSet))
		for _, k := range keyToSet {
			if !keyMap[k] {
				keyMap[k] = true
				uniqueKeysToSet = append(uniqueKeysToSet, k)
			}
		}
		c.cache.Set(dkey, uniqueKeysToSet, ttl)
	}
	return nil
}

func (c *InMemoryClientCacheProvider) deleteKeyArray(dkey string, setTombstone bool) {
	isTombstone := false

	tombstoneValue := GenerateTombstoneSentinel() // Generate a unique tombstone value

	if x, found := c.cache.Get(dkey); found {
		if keyNames, ok := x.([]string); ok {
			c.inMemMultiDelete(keyNames)
		} else if val, ok := x.(string); ok {
			if IsTombstoneSentinel(val) {
				isTombstone = true
			}
		}
	}
	if setTombstone {
		c.cache.Set(dkey, string(tombstoneValue), c.tombstoneTTL)
	} else {
		if !isTombstone {
			c.cache.Delete(dkey)
		}
	}
}

// AddDependency adds the given cache key(s) as dependencies of an item represented by by key
func (c *InMemoryClientCacheProvider) AddDependency(ctx context.Context, keysIn []Key, values []Key, ttl time.Duration) error {
	keys := c.getStringKeysFromCacheKeys(keysIn)
	i := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		i = append(i, string(v))
	}

	c.keysMutex.Lock()
	defer c.keysMutex.Unlock()

	return ucerr.Wrap(c.saveKeyArray(keys, i, ttl))
}

// ClearDependencies clears the dependencies of an item represented by key and removes all dependent keys from the cache
func (c *InMemoryClientCacheProvider) ClearDependencies(ctx context.Context, key Key, setTombstone bool) error {
	setTombstone = setTombstone && c.tombstoneTTL > 0 // don't actually set tombstone if tombstoneTTL is 0
	if key == "" {
		return ucerr.New("Expected at least one key passed to ClearDependencies")
	}

	c.keysMutex.Lock()
	defer c.keysMutex.Unlock()

	c.deleteKeyArray(string(key), setTombstone)
	return nil
}

// Flush flushes the cache (applies only to the tenant for which the client was created)
func (c *InMemoryClientCacheProvider) Flush(ctx context.Context, prefix string, flushTombstones bool) error {
	c.keysMutex.Lock()
	defer c.keysMutex.Unlock()

	for k, v := range c.cache.Items() {
		if strings.HasPrefix(k, prefix) {
			if val, ok := v.Object.(string); ok {
				if !IsTombstoneSentinel(val) || flushTombstones {
					c.cache.Delete(k)
				}
			}
		}
	}
	return nil
}

// GetCacheName returns the name of the cache
func (c *InMemoryClientCacheProvider) GetCacheName(ctx context.Context) string {
	return c.cacheName
}

// RegisterInvalidationHandler registers a handler for cache invalidation
func (c *InMemoryClientCacheProvider) RegisterInvalidationHandler(ctx context.Context, handler InvalidationHandler, key Key) error {
	return ucerr.Errorf("RegisterInvalidationHandler not supported for InMemoryClientCacheProvider")
}

// LogKeyValues logs all keys and values in the cache
func (c *InMemoryClientCacheProvider) LogKeyValues(ctx context.Context, prefix string) error {
	c.keysMutex.Lock()
	defer c.keysMutex.Unlock()

	for k, v := range c.cache.Items() {
		if strings.HasPrefix(k, prefix) {
			uclog.Infof(ctx, "Cache[%v] key %v value %v", c.cacheName, k, v.Object)
		}
	}
	return nil
}
