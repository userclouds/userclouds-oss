package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

const tempTTL = 1 * time.Minute
const sentinelSeparator = "#"

// LayeringWrapper is a wrapper around a cache that allows layering of two providers
type LayeringWrapper struct {
	cacheInner Provider // cache provider that is read first (closest to the data consumer)
	cacheOuter Provider // cache provider that is read second and is used to set the value in the inner provider (further away from the data consumer)
}

func combineSentinels(sentinelTop Sentinel, sentinelBottom Sentinel) Sentinel {
	if sentinelBottom == "" && sentinelTop == "" {
		return ""
	}
	return Sentinel(string(sentinelTop) + sentinelSeparator + string(sentinelBottom))
}

func splitSentinels(sentinelCombined Sentinel) (Sentinel, Sentinel) {
	split := strings.Split(string(sentinelCombined), sentinelSeparator)
	if len(split) == 2 {
		return Sentinel(split[0]), Sentinel(split[1])
	}
	return Sentinel(split[0]), ""
}

// NewLayeringWrapper creates a new type LayeringWrapper. If the the two layers use same invalidation mechanism, them the LayeringWrapper can
// be wrapped by InvalidationWrapper. If the two layers use different invalidation mechanisms, then each layer should be wrapped by InvalidationWrapper
// separately
func NewLayeringWrapper(cpInner Provider, cpOuter Provider) *LayeringWrapper {
	return &LayeringWrapper{
		cacheInner: cpInner,
		cacheOuter: cpOuter,
	}
}

// GetValue gets the value in cache key (if any) and tries to lock the key for Read is lockOnMiss = true
func (l *LayeringWrapper) GetValue(ctx context.Context, key Key, lockOnMiss bool) (*string, *string, Sentinel, bool, error) {
	// Check the provider closer to data consumer first
	valInner, conflictInner, sentinelInner, partialHit, err := l.cacheInner.GetValue(ctx, key, lockOnMiss)
	// Check if read failed and error out
	if err != nil {
		uclog.Verbosef(ctx, "Cache[%v] Failed to read key %v in inner cache with error: %v", l.GetCacheName(ctx), key, err)
		return nil, nil, NoLockSentinel, false, ucerr.Wrap(err)
	}
	// Check if value is present in inner cache (cache hit)
	if valInner != nil {
		return valInner, conflictInner, sentinelInner, partialHit, nil
	}
	// If we failed to get a lock in the cache closer to consumer, don't try to get it in the cache further away
	if sentinelInner == NoLockSentinel {
		lockOnMiss = false
	}

	// If value is not present in inner cache, check outer cache
	valOuter, conflictOuter, sentinelOuter, _, err := l.cacheOuter.GetValue(ctx, key, lockOnMiss)
	// Check if read failed and error out
	if err != nil {
		uclog.Verbosef(ctx, "Cache[%v] Failed to read key %v in outer cache with error: %v", l.GetCacheName(ctx), key, err)
		return nil, nil, NoLockSentinel, false, ucerr.Wrap(err)
	}
	// Check if value is present in outer cache (cache hit)
	if valOuter != nil {
		// Propogate it to the inner cache if we had a lock.
		if sentinelInner != NoLockSentinel {
			// TODO this happens to be safe today because of the usage vs by design. We shouldn't be setting the value at this level
			// incase there are other changes that need to be made in the inner cache at a higher level (like updating the dependencies)
			if _, _, err := l.cacheInner.SetValue(ctx, key, []Key{key}, *valOuter, sentinelInner, tempTTL); err != nil {
				uclog.Verbosef(ctx, "Cache[%v] Failed to set key %v in inner cache with value %v error: %v", l.GetCacheName(ctx), key, valOuter, err)
				return nil, nil, NoLockSentinel, false, ucerr.Wrap(err)
			}
		}
		return valOuter, conflictOuter, sentinelOuter, true, nil
	}

	// If we didn't get a lock in the cache further away, release the lock we got in the cache closer to the data consumer
	if lockOnMiss && sentinelInner == NoLockSentinel {
		l.cacheInner.ReleaseSentinel(ctx, []Key{key}, sentinelInner)
		sentinelInner = NoLockSentinel
	}

	// If value is not present in either cache, return combined sentinel and conflict
	combinedSentinel := combineSentinels(sentinelInner, sentinelOuter)

	combinedConflict := conflictInner
	if conflictInner != nil || conflictOuter != nil {
		conflictInnerStr := ""
		if conflictInner != nil {
			conflictInnerStr = *conflictInner
		}
		conflictOuterStr := ""
		if conflictOuter != nil {
			conflictOuterStr = *conflictOuter
		}
		combinedStr := string(combineSentinels(Sentinel(conflictInnerStr), Sentinel(conflictOuterStr)))
		if IsTombstoneSentinel(conflictOuterStr) && IsTombstoneSentinel(conflictInnerStr) {
			combinedStr = conflictOuterStr
		}
		combinedConflict = &combinedStr
	}
	return nil, combinedConflict, combinedSentinel, false, nil
}

// GetValues gets the value in cache keys (if any) and tries to lock the keys[i] for Read is lockOnMiss[i] = true
func (l *LayeringWrapper) GetValues(ctx context.Context, keys []Key, lockOnMiss []bool) ([]*string, []*string, []Sentinel, error) {
	valuesCombined := make([]*string, len(keys))
	sentinelsCombined := make([]Sentinel, len(keys))
	conflictsCombined := make([]*string, len(keys))
	// Check the provider closer to data consumer first
	valuesInner, conflictsInner, sentinelsInner, err := l.cacheInner.GetValues(ctx, keys, lockOnMiss)
	// Check if read failed and error out
	if err != nil {
		uclog.Verbosef(ctx, "Cache[%v] Failed to read keys %v in inner cache with error: %v", l.GetCacheName(ctx), keys, err)
		return valuesInner, conflictsInner, sentinelsInner, ucerr.Wrap(err)
	}
	// Check if values are present in inner cache (cache hit)
	cacheHits := 0
	keysMissed := []Key{}
	lockOnMissMissed := []bool{}
	keyIndexMissed := []int{}
	for i, v := range valuesInner {
		if v != nil {
			valuesCombined[i] = v
			cacheHits++
		} else {
			keysMissed = append(keysMissed, keys[i])
			// If we failed to get a lock in the cache closer to consumer, don't try to get it in the cache further away
			if sentinelsInner[i] == NoLockSentinel {
				lockOnMiss[i] = false
			}
			lockOnMissMissed = append(lockOnMissMissed, lockOnMiss[i])
			keyIndexMissed = append(keyIndexMissed, i)
		}
	}
	// All values were found in inner cache
	if cacheHits == len(keys) {
		uclog.Verbosef(ctx, "Cache[%v] Returned all keys %v from inner cache", l.GetCacheName(ctx), keys)
		return valuesCombined, conflictsInner, sentinelsInner, nil
	}

	// If value is not present in inner cache, check outer cache
	valOuter, conflictsOuter, sentinelsOuter, err := l.cacheOuter.GetValues(ctx, keysMissed, lockOnMissMissed)
	// Check if read failed and error out
	if err != nil {
		uclog.Verbosef(ctx, "Cache[%v] Failed to read keys %v in outer cache with error: %v", l.GetCacheName(ctx), keys, err)
		return valuesCombined, conflictsCombined, sentinelsCombined, ucerr.Wrap(err)
	}
	// Check if value is present in outer cache (cache hit)
	for i, v := range valOuter {
		if v != nil {
			// Propogate it to the inner cache
			key := keys[keyIndexMissed[i]]
			if _, _, err := l.cacheInner.SetValue(ctx, key, []Key{key}, *v, sentinelsInner[keyIndexMissed[i]], tempTTL); err != nil {
				uclog.Verbosef(ctx, "Cache[%v] Failed to set key %v in inner cache with value %v error: %v", l.GetCacheName(ctx), key, v, err)
			}
			valuesCombined[keyIndexMissed[i]] = v
		}
	}

	keyIndexMissedReversed := make(map[int]int, len(keyIndexMissed))
	for i, v := range keyIndexMissed {
		keyIndexMissedReversed[v] = i
	}

	for i, v := range valuesCombined {
		if v == nil {
			// If we didn't get a lock in the cache further away, release the lock we got in the cache closer to the data consumer
			if sentinelsOuter[keyIndexMissedReversed[i]] == NoLockSentinel {
				l.cacheInner.ReleaseSentinel(ctx, []Key{keys[i]}, sentinelsInner[i])
				sentinelsInner[i] = NoLockSentinel
			}

			// If value is not present in either cache, return combined sentinel and conflict
			sentinelsCombined[i] = combineSentinels(sentinelsInner[i], sentinelsOuter[keyIndexMissedReversed[i]])

			conflictsCombined[i] = conflictsInner[i]

			if conflictsInner[i] != nil || conflictsOuter[keyIndexMissedReversed[i]] != nil {
				conflictInnerStr := ""
				if conflictsInner[i] != nil {
					conflictInnerStr = *conflictsInner[i]
				}
				conflictOuterStr := ""
				if conflictsOuter[keyIndexMissedReversed[i]] != nil {
					conflictOuterStr = *conflictsOuter[keyIndexMissedReversed[i]]
				}
				combinedStr := string(combineSentinels(Sentinel(conflictInnerStr), Sentinel(conflictOuterStr)))
				if IsTombstoneSentinel(conflictOuterStr) && IsTombstoneSentinel(conflictInnerStr) {
					combinedStr = conflictOuterStr
				}
				conflictsCombined[i] = &combinedStr
			}
		}
	}
	return valuesCombined, conflictsCombined, sentinelsCombined, nil

}

// SetValue sets the value in cache key(s) to val with given expiration time if the sentinel matches lkey and returns true if the value was set
func (l *LayeringWrapper) SetValue(ctx context.Context, lkey Key, keysToSet []Key, val string, sentinel Sentinel,
	ttl time.Duration) (bool, bool, error) {
	sentinelInner, sentinelOuter := splitSentinels(sentinel)
	setCombined := false
	conflictCombined := false
	var err error

	// If we couldn't acquire the lock in either cache, return false
	if sentinelOuter == NoLockSentinel || sentinelInner == NoLockSentinel {
		return false, true, nil
	}

	// Write the value into the outer cache first (further away from the data consumer)
	setCombined, conflictCombined, err = l.cacheOuter.SetValue(ctx, lkey, keysToSet, val, sentinelOuter, ttl)
	if err != nil {
		uclog.Verbosef(ctx, "Cache[%v] Failed to set lkey %v in inner cache with error: %v", l.GetCacheName(ctx), lkey, err)
		return false, false, ucerr.Wrap(err)
	}

	// If we couldn't set the value in the further away cache, return false
	if !setCombined {
		return false, conflictCombined, nil
	}

	// Write the value into the inner cache (closer to the data consumer)
	setInner, conflictInner, err := l.cacheInner.SetValue(ctx, lkey, keysToSet, val, sentinelInner, ttl)
	if err != nil {
		uclog.Verbosef(ctx, "Cache[%v] Failed to set lkey %v in outer cache with error: %v", l.GetCacheName(ctx), lkey, err)
		return false, false, ucerr.Wrap(err)
	}

	if !setInner || conflictInner {
		uclog.Verbosef(ctx, "Cache[%v] Failed to set lkey %v in outer cache while succeeding in inner cache", l.GetCacheName(ctx), lkey)
		// If we couldn't set the value in the closer cache, it maybe due to an invalidation or simultaneous delete with tombstone
		return false, conflictInner, nil
	}

	return true, false, nil
}

// DeleteValue deletes the value(s) in passed in keys, force is true also deletes keys with sentinel or tombstone values
func (l *LayeringWrapper) DeleteValue(ctx context.Context, key []Key, setTombstone bool, force bool) error {
	if err := l.cacheOuter.DeleteValue(ctx, key, setTombstone, force); err != nil {
		uclog.Verbosef(ctx, "Cache[%v] Failed to delete key %v in inner cache with error: %v", l.GetCacheName(ctx), key, err)
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(l.cacheInner.DeleteValue(ctx, key, setTombstone, force))
}

// WriteSentinel writes the sentinel value into the given keys, returns NoLockSentinel if it couldn't acquire the lock
func (l *LayeringWrapper) WriteSentinel(ctx context.Context, stype SentinelType, keys []Key) (Sentinel, error) {
	// First try to lock the key in the cache closer to the data consumer to minimize the contention in the cache further away
	sentineltInner, err := l.cacheInner.WriteSentinel(ctx, stype, keys)
	if err != nil {
		uclog.Verbosef(ctx, "Cache[%v] Failed to delete key %v in inner cache with error: %v", l.GetCacheName(ctx), keys, err)
		return NoLockSentinel, ucerr.Wrap(err)
	}
	if sentineltInner == NoLockSentinel {
		return NoLockSentinel, nil
	}
	// Now try to lock the key in the cache further away from the data consumer
	sentineltOuter, err := l.cacheOuter.WriteSentinel(ctx, stype, keys)
	if err != nil {
		uclog.Verbosef(ctx, "Cache[%v] Failed to write sentinel keys %v in outer cache with error: %v", l.GetCacheName(ctx), keys, err)
		return NoLockSentinel, ucerr.Wrap(err)
	}
	// If failed to lock the key in the cache further away from the data consumer, release the lock in the cache closer to the data consumer
	if sentineltOuter == NoLockSentinel {
		l.cacheInner.ReleaseSentinel(ctx, keys, sentineltInner)
		return NoLockSentinel, nil
	}
	// We got the lock in both caches so combine the sentinels
	return combineSentinels(sentineltInner, sentineltOuter), nil
}

// ReleaseSentinel clears the sentinel value from the given keys
func (l *LayeringWrapper) ReleaseSentinel(ctx context.Context, keys []Key, s Sentinel) {
	sentinelInner, sentinelOuter := splitSentinels(s)
	l.cacheOuter.ReleaseSentinel(ctx, keys, sentinelOuter)
	l.cacheInner.ReleaseSentinel(ctx, keys, sentinelInner)
}

// AddDependency adds the given cache key(s) as dependencies of an item represented by by key. Fails if any of the dependency keys passed in contain tombstone
func (l *LayeringWrapper) AddDependency(ctx context.Context, keysIn []Key, dependentKey []Key, ttl time.Duration) error {
	if err := l.cacheOuter.AddDependency(ctx, keysIn, dependentKey, ttl); err != nil {
		uclog.Verbosef(ctx, "Cache[%v] Failed to add dependency %v to keys %v in outer cache with error: %v", l.GetCacheName(ctx), dependentKey, keysIn, err)
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(l.cacheInner.AddDependency(ctx, keysIn, dependentKey, ttl))
}

// ClearDependencies clears the dependencies of an item represented by key and removes all dependent keys from the cache
func (l *LayeringWrapper) ClearDependencies(ctx context.Context, key Key, setTombstone bool) error {
	if err := l.cacheOuter.ClearDependencies(ctx, key, setTombstone); err != nil {
		uclog.Verbosef(ctx, "Cache[%v] Failed to clear dependency key %v in outer cache with error: %v", l.GetCacheName(ctx), key, err)
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(l.cacheInner.ClearDependencies(ctx, key, setTombstone))
}

// Flush flushes the cache
func (l *LayeringWrapper) Flush(ctx context.Context, prefix string, flushTombstones bool) error {
	if err := l.cacheOuter.Flush(ctx, prefix, flushTombstones); err != nil {
		uclog.Verbosef(ctx, "Cache[%v] Failed to flush with prefix [%v] in outer cache with error: %v", l.GetCacheName(ctx), prefix, err)
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(l.cacheInner.Flush(ctx, prefix, flushTombstones))
}

// GetCacheName returns the name of the cache
func (l *LayeringWrapper) GetCacheName(ctx context.Context) string {
	return fmt.Sprintf("Layered[%s,%s]", l.cacheInner.GetCacheName(ctx), l.cacheOuter.GetCacheName(ctx))
}

// RegisterInvalidationHandler registers a handler to be called when the specified key is invalidated
func (l *LayeringWrapper) RegisterInvalidationHandler(ctx context.Context, handler InvalidationHandler, key Key) error {
	// Only register for invalidation in the inner cache (since outer cache is guaranteed to have correct values (if any) by that point)
	return ucerr.Wrap(l.cacheInner.RegisterInvalidationHandler(ctx, handler, key))
}

// LogKeyValues is debugging only method that logs the values of the keys with the given prefix
func (l *LayeringWrapper) LogKeyValues(ctx context.Context, prefix string) error {
	if err := l.cacheOuter.LogKeyValues(ctx, prefix); err != nil {
		uclog.Verbosef(ctx, "Cache[%v] Failed to log key value with prefix [%v] in outer cache with error: %v", l.GetCacheName(ctx), prefix, err)
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(l.cacheInner.LogKeyValues(ctx, prefix))
}

// Layered returns true if the cache provider is a multi-layered cache
func (*LayeringWrapper) Layered(context.Context) bool {
	return true
}

// ReleaseRateLimitSlot will release the earliest rate limit slot for the provided set of keys
func (l *LayeringWrapper) ReleaseRateLimitSlot(
	ctx context.Context,
	keys []RateLimitKey,
) (totalSlots int64, err error) {
	for _, cacheProvider := range []Provider{l.cacheInner, l.cacheOuter} {
		if cacheProvider.SupportsRateLimits(ctx) {
			totalSlots, err = cacheProvider.ReleaseRateLimitSlot(ctx, keys)
			if err != nil {
				return 0, ucerr.Wrap(err)
			}
			break
		}
	}

	return totalSlots, nil
}

// ReserveRateLimitSlot will return if a rate limit slot can be reserved, given the specified
// limit and provided keys, actually reserving the slot if requested
func (l *LayeringWrapper) ReserveRateLimitSlot(
	ctx context.Context,
	keys []RateLimitKey,
	limit int64,
	ttl time.Duration,
	takeSlot bool,
) (reserved bool, totalSlots int64, err error) {
	for _, cacheProvider := range []Provider{l.cacheInner, l.cacheOuter} {
		if cacheProvider.SupportsRateLimits(ctx) {
			reserved, totalSlots, err = cacheProvider.ReserveRateLimitSlot(ctx, keys, limit, ttl, takeSlot)
			if err != nil {
				return false, 0, ucerr.Wrap(err)
			}
			break
		}
	}

	return reserved, totalSlots, nil
}

// SupportsRateLimits returns true if rate limiting is supported
func (l *LayeringWrapper) SupportsRateLimits(ctx context.Context) bool {
	return l.cacheInner.SupportsRateLimits(ctx) ||
		l.cacheOuter.SupportsRateLimits(ctx)
}
