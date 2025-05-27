package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

const (
	// If the cache is accessed by a number of clients (across all machines) above this value performing create/update/delete operations on same
	// keys, the operation may fail for some of them due to optimistic locking not retrying enough times.
	maxRdConflictRetries = 15
	// RegionalRedisCacheName is default name of the regional redis cache and channel used for inregion cache invalidation
	RegionalRedisCacheName = "redisRegionalCache"
	//GlobalRedisCacheName is default name of the global redis channel used for cache invalidation cross region
	GlobalRedisCacheName = "redisGlobalCache"
	// flushBatchSize is the number of keys to flush at a time
	flushBatchSize = 100
)

// RedisClientCacheProvider is the base implementation of the CacheProvider interface
type RedisClientCacheProvider struct {
	NoLayeringProvider
	redisClient  *redis.Client
	prefix       string
	sm           SentinelManager
	cacheName    string
	readOnly     bool
	tombstoneTTL time.Duration
}

type optionsRedis struct {
	sm       SentinelManager
	prefix   string
	readOnly bool
}

// OptionRedis specifies optional arguement for RedisClientCacheProvider
type OptionRedis interface {
	apply(*optionsRedis)
}

type optFuncRedis func(*optionsRedis)

func (o optFuncRedis) apply(opts *optionsRedis) {
	o(opts)
}

// SentinelManagerRedis allows specifying a custom CacheSentinelManager
func SentinelManagerRedis(sm SentinelManager) OptionRedis {
	return optFuncRedis(func(opts *optionsRedis) {
		opts.sm = sm
	})
}

// KeyPrefixRedis allows specifying a key prefix that all keys managed by this cache have to have
func KeyPrefixRedis(prefix string) OptionRedis {
	return optFuncRedis(func(opts *optionsRedis) {
		opts.prefix = prefix
	})
}

// ReadOnlyRedis specifies that the cache provider will not make any modfications
// It will only read the values via GetValues and GetValue if they are there and
// make not other modifications otherwise
func ReadOnlyRedis() OptionRedis {
	return optFuncRedis(func(opts *optionsRedis) {
		opts.readOnly = true
	})
}

// NewRedisClientCacheProvider creates a new RedisClientCacheProvider
func NewRedisClientCacheProvider(rc *redis.Client, cacheName string, opts ...OptionRedis) *RedisClientCacheProvider {
	var options optionsRedis
	for _, opt := range opts {
		opt.apply(&options)
	}

	sm := options.sm
	if sm == nil {
		sm = NewWriteThroughCacheSentinelManager()
	}

	return &RedisClientCacheProvider{
		redisClient:  rc,
		prefix:       options.prefix,
		sm:           sm,
		cacheName:    cacheName,
		readOnly:     options.readOnly,
		tombstoneTTL: InvalidationTombstoneTTL,
	}
}

func (c *RedisClientCacheProvider) logError(ctx context.Context, err error, format string, args ...any) {
	if errors.Is(err, context.Canceled) {
		uclog.Warningf(ctx, format, args...)
	} else {
		uclog.Errorf(ctx, format, args...)
	}
}

// WriteSentinel writes the sentinel value into the given keys
func (c *RedisClientCacheProvider) WriteSentinel(ctx context.Context, stype SentinelType, keysIn []Key) (Sentinel, error) {
	sentinel := c.sm.GenerateSentinel(stype)
	keys, err := getValidatedStringKeysFromCacheKeys(keysIn, c.prefix)
	if err != nil {
		return NoLockSentinel, ucerr.Wrap(err)
	}
	// There must be at least one key to lock
	if len(keys) == 0 {
		return NoLockSentinel, ucerr.New("WriteSentinel was passed no keys to set")
	}

	if c.readOnly {
		uclog.Verbosef(ctx, "Cache[%v] WriteSentinel - skipping cache due to read only mode", c.cacheName)
		return NoLockSentinel, nil
	}

	lockValue := NoLockSentinel
	// Transactional function to read current value of the key and try to take the lock for this operation depending on the key value
	txf := func(tx *redis.Tx) error {
		// Operation is committed only if the watched keys remain unchanged.
		_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			lockValue = NoLockSentinel
			if !c.sm.CanAlwaysSetSentinel(sentinel) {
				// Check if the primary key for the operation is already locked
				values, err := c.redisClient.MGet(ctx, keys...).Result()
				if err != nil && err != redis.Nil {
					// If we can't read the key, we can't take a lock
					return ucerr.Wrap(err)
				}
				// Get the value of the primary key
				value, ok := values[0].(string)
				if values[0] != nil && !ok {
					return ucerr.Errorf("Cache[%v] WriteSentinel - invalid value  %v in cache", c.cacheName, values[0])
				}
				// If the key is already locked and see if we have precedence
				if err == nil && c.sm.IsSentinelValue(value) {
					if !c.sm.CanSetSentinelGivenCurrVal(Sentinel(value), sentinel) {
						return nil
					}
				}
				// Proceed to take the lock if key is empty (err == redis.Nil) or it doesn't contain sentinel value

				// First make sure that we don't overwrite a tombstone and extend its TTL since it may otherwise expire during the operation
				// leaving the key unlocked
				newKeys := make([]string, 0, len(keys))
				newKeys = append(newKeys, keys[0])
				for i := 1; i < len(keys); i++ {
					if values[i] == nil || !IsTombstoneSentinel(values[i].(string)) {
						newKeys = append(newKeys, keys[i])
					} else if stype != Read {
						// If the key is a tombstone, refresh its TTL
						pipe.Expire(ctx, keys[i], c.tombstoneTTL)
					}
				}
				keys = newKeys
			}

			if err := multiSetWithPipe(ctx, pipe, keys, string(sentinel), SentinelTTL); err != nil {
				return ucerr.Wrap(err)
			}
			lockValue = sentinel
			return nil
		})
		return ucerr.Wrap(err)
	}

	// Retry if the key has been changed.
	for range maxRdConflictRetries {
		err := c.redisClient.Watch(ctx, txf, keys[0])
		if err == nil {
			// Success.
			return lockValue, nil
		}
		if errors.Is(err, redis.TxFailedErr) {
			// Optimistic lock lost. Retry.
			uclog.Verbosef(ctx, "Cache[%v] WriteSentinel - retry on keys %v", c.cacheName, keys)
			continue
		}
		// Return any other error.
		return NoLockSentinel, ucerr.Wrap(err)
	}

	uclog.Debugf(ctx, "Cache[%v] WriteSentinel - reached maximum number of retries on keys %v skipping cache", c.cacheName, keys)
	return NoLockSentinel, ucerr.New("WriteSentinel reached maximum number of retries")
}

// getValidatedStringKeysFromCacheKeys filters out any empty keys and does the type conversion
func getValidatedStringKeysFromCacheKeys[KeyType any](keys []KeyType, prefix string) ([]string, error) {
	strKeys := make([]string, 0, len(keys))
	for _, k := range keys {
		if s := fmt.Sprintf("%v", k); s != "" {
			if strings.HasPrefix(s, prefix) {
				strKeys = append(strKeys, s)
			} else {
				return nil, ucerr.Errorf("Key %v does not have prefix %v", k, prefix)
			}
		}
	}
	return strKeys, nil
}

func getValidatedStringKeyFromCacheKey[KeyType any](key KeyType, prefix string, methodName string) (string, error) {
	s := fmt.Sprintf("%v", key)
	if s == "" {
		return "", ucerr.Errorf("Empty key provided to %s", methodName)
	}
	if strings.HasPrefix(s, prefix) {
		return s, nil
	}
	return "", ucerr.Errorf("Key %v does not have prefix %v", key, prefix)
}

// ReleaseSentinel clears the sentinel value from the given keys
func (c *RedisClientCacheProvider) ReleaseSentinel(ctx context.Context, keysIn []Key, s Sentinel) {
	// Filter out any empty keys
	keys, err := getValidatedStringKeysFromCacheKeys(keysIn, c.prefix)
	// If there are no keys to potentially clear, return
	if err != nil || len(keys) == 0 {
		return
	}

	if c.readOnly {
		uclog.Verbosef(ctx, "Cache[%v] ReleaseSentinel - skipping cache due to read only mode", c.cacheName)
		return
	}

	// Using optimistic concurrency control to clear the sentinels set by our operation. We need to make sure that no ones else
	// writes to the keys between the read and the delete so that we don't accidentally clear another operations sentinel

	// Transactional function to read current value of keys and delete them only if they contain the sentinel value
	txf := func(tx *redis.Tx) error {
		values, err := c.redisClient.MGet(ctx, keys...).Result()
		keysToClear := []string{}
		if err == nil {
			keysToClear = make([]string, 0, len(keys))
			for i, v := range values {
				vS, ok := v.(string)
				if ok && vS == string(s) {
					keysToClear = append(keysToClear, keys[i])
				}
			}

		}

		if len(keysToClear) == 0 {
			return nil
		}

		// Operation is committed only if the watched keys remain unchanged.
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			if len(keysToClear) > 0 {
				if err := pipe.Del(ctx, keysToClear...).Err(); err != nil && err != redis.Nil {
					c.logError(ctx, err, "Cache[%v] error clearing key(s) %v sentinel  %v", c.cacheName, keysToClear, err)
				}
				uclog.Verbosef(ctx, "Cache[%v] cleared key(s) %v sentinel %v", c.cacheName, keysToClear, s)
			}
			return nil
		})
		return ucerr.Wrap(err)
	}

	// Retry if the key has been changed.
	for range maxRdConflictRetries {
		err := c.redisClient.Watch(ctx, txf, keys...)
		if err == nil {
			// Success.
			return
		}
		if errors.Is(err, redis.TxFailedErr) {
			// Optimistic lock lost. Retry.
			uclog.Verbosef(ctx, "Cache[%v] ReleaseSentinel - retry on keys %v", c.cacheName, keys)
			continue
		}
		// Return any other error.
		uclog.Debugf(ctx, "Cache[%v] - ReleaseSentinel - failed on keys %v with %v skipping cache. Keys maybe locked until sentinel expires", c.cacheName, keys, err)
		return
	}
}

// multiSetWithPipe add commands to set the keys and expiration to given pipe
func multiSetWithPipe(ctx context.Context, pipe redis.Pipeliner, keys []string, value string, ttl time.Duration) error {
	var ifaces = make([]any, 0, len(keys)*2)
	for i := range keys {
		ifaces = append(ifaces, keys[i], value)
	}
	if err := pipe.MSet(ctx, ifaces...).Err(); err != nil {
		return ucerr.Wrap(err)
	}
	for i := range keys {
		pipe.Expire(ctx, keys[i], ttl)
	}
	return nil
}

// SetValue sets the value in cache key(s) to val with given expiration time if the sentinel matches and returns true if the value was set
func (c *RedisClientCacheProvider) SetValue(ctx context.Context, lkeyIn Key, keysToSet []Key, val string,
	sentinel Sentinel, ttl time.Duration) (bool, bool, error) {

	keys, err := getValidatedStringKeysFromCacheKeys(keysToSet, c.prefix)
	if err != nil {
		return false, false, ucerr.Wrap(err)
	}
	// There needs to be at least a single key to check for sentinel/set to value
	if len(keys) == 0 {
		return false, false, ucerr.New("No keys provided to SetValue")
	}

	lkey, err := getValidatedStringKeyFromCacheKey(lkeyIn, c.prefix, "SetValue")
	if err != nil {
		return false, false, ucerr.Wrap(err)
	}

	if c.readOnly {
		uclog.Verbosef(ctx, "Cache[%v] SetValue - skipping cache due to read only mode", c.cacheName)
		return false, false, nil
	}

	conflictDetected := false
	valueSet := false

	// Transactional function to read value of pkey and perform the corresponding update depending on its value atomically
	txf := func(tx *redis.Tx) error {

		// Operation is committed only if the watched keys remain unchanged.
		_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			conflictDetected = false
			valueSet = false

			cV, err := c.redisClient.Get(ctx, lkey).Result()
			// Either key is empty or we couldn't get it
			if err != nil {
				return nil
			}

			set, clear, conflict, refresh := c.sm.CanSetValue(cV, val, sentinel)

			if set { // Value can be set
				uclog.Verbosef(ctx, "Cache[%v] set key %v ttl %v", c.cacheName, keys, ttl)
				if err := multiSetWithPipe(ctx, pipe, keys, val, ttl); err != nil {
					return ucerr.Wrap(err)
				}
				valueSet = true
				return nil
			} else if clear { // Intermediate state detected so clear the cache
				uclog.Verbosef(ctx, "Cache[%v] cleared on value mismatch or conflict sentinel key %v curr var %v would store %v", c.cacheName, keys, cV, val)
				if err := pipe.Del(ctx, keys...).Err(); err != nil && err != redis.Nil {
					c.logError(ctx, err, "Cache[%v] - error clearing key(s) %v mismatch -  %v", c.cacheName, keys, err)
				}
				return nil
			} else if conflict { // Conflict detected so upgrade the lock to conflict
				if err := multiSetWithPipe(ctx, pipe, keys, cV+string(sentinel), SentinelTTL); err != nil {
					return ucerr.Wrap(err)
				}
				uclog.Verbosef(ctx, "Cache[%v] lock upgraded to conflict on write collision %v got %v added %v", c.cacheName, lkey, cV, sentinel)
				conflictDetected = true
				return nil
			} else if refresh { // Refresh TTL on current value in the keys
				uclog.Verbosef(ctx, "Cache[%v] refreshing TTL in %v got %v added %v", c.cacheName, keys, cV, sentinel)
				for _, key := range keys {
					if err := pipe.Expire(ctx, key, c.tombstoneTTL).Err(); err != nil && err != redis.Nil {
						c.logError(ctx, err, "Cache[%v] - error reseting expiration key(s) %v mismatch -  %v", c.cacheName, key, err)
					}
				}
				return nil
			}

			uclog.Verbosef(ctx, "Cache[%v] not set key %v on sentinel mismatch got %v expect %v", c.cacheName, lkey, cV, sentinel)
			conflictDetected = true
			return nil
		})
		return ucerr.Wrap(err)
	}

	// Retry if the key has been changed.
	for range maxRdConflictRetries {
		err := c.redisClient.Watch(ctx, txf, lkey)
		if err == nil {
			// Success.
			return valueSet, conflictDetected, nil
		}
		if errors.Is(err, redis.TxFailedErr) {
			// Optimistic lock lost. Retry.
			uclog.Verbosef(ctx, "Cache[%v] SetValue - retry on keys %v", c.cacheName, keys)
			continue
		}
		// Return any other error.
		return false, false, ucerr.Wrap(err)
	}
	uclog.Debugf(ctx, "Cache[%v] SetValue - hit too many retries %v skipping cache.", c.cacheName, keys)
	return false, false, ucerr.New("SetValue hit too many retries")
}

// GetValues gets the value in cache keys (if any) and tries to lock the keys[i] for Read is lockOnMiss[i] = true
func (c *RedisClientCacheProvider) GetValues(ctx context.Context, keysIn []Key, lockOnMiss []bool) ([]*string, []*string, []Sentinel, error) {
	if len(keysIn) == 0 && len(lockOnMiss) == 0 {
		uclog.Errorf(ctx, "Cache[%v] GetValues called with no keys", c.cacheName)
		return nil, nil, nil, nil
	}
	if len(keysIn) != len(lockOnMiss) {
		return nil, nil, nil, ucerr.Errorf("Number of keys provided to GetValues has to be equal to number of lockOnMiss, keys: %d lockOnMiss: %d", len(keysIn), len(lockOnMiss))
	}
	// Create arrays for output values and sentinels
	val := make([]*string, len(keysIn))
	conflicts := make([]*string, len(keysIn))
	sentinels := make([]Sentinel, len(keysIn))

	// Initialize sentinels to NoLockSentinel
	for i := range sentinels {
		sentinels[i] = NoLockSentinel
	}

	// Validate that all the keys have the correct prefix and filter out any empty keys
	keys, err := getValidatedStringKeysFromCacheKeys(keysIn, c.prefix)
	if err != nil {
		return val, conflicts, sentinels, ucerr.Wrap(err)
	}

	// For now fail on empty keys to make code easier to reason about
	if len(keys) != len(keysIn) {
		return val, conflicts, sentinels, ucerr.New("Blank keys are not allowed in GetValues")
	}

	// Get all the values from cache
	valuesOut, err := c.redisClient.MGet(ctx, keys...).Result()

	// If we failed on the read return the error
	if err != nil && err != redis.Nil {
		return val, conflicts, sentinels, ucerr.Wrap(err)
	}

	// Only copy the keys that are not locked by other operations into output array
	for i, v := range valuesOut {
		if v != nil {
			if vS, ok := v.(string); ok {
				if !c.sm.IsSentinelValue(vS) {
					val[i] = &vS
					uclog.Verbosef(ctx, "Cache[%v] hit key %v", c.cacheName, keys[i])
					continue
				}
				conflicts[i] = &vS
			}
			uclog.Verbosef(ctx, "Cache[%v] key %v is locked for in progress op %v", c.cacheName, keys[i], v)
			continue
		} else if !lockOnMiss[i] {
			uclog.Verbosef(ctx, "Cache[%v] miss on key %v lock not request", c.cacheName, keys[i])
		}
	}

	if c.readOnly {
		uclog.Verbosef(ctx, "Cache[%v] GetValues - skipping writing locks due to read only mode", c.cacheName)
		return val, conflicts, sentinels, nil
	}

	// Check if we need to lock any keys (handles err == redis.Nil)
	keysToLock := make(map[string]any)
	for i, lock := range lockOnMiss {
		if lock && (valuesOut == nil || valuesOut[i] == nil) {
			if _, ok := keysToLock[keys[i]]; !ok {
				sentinels[i] = c.sm.GenerateSentinel(Read)
				keysToLock[keys[i]] = string(sentinels[i])
			} else {
				// Duplicate key so copy the sentinel from the first instance
				sentinels[i] = Sentinel((keysToLock[keys[i]]).(string))
			}
		}
	}

	// If there are keys to lock, try to lock them
	if len(keysToLock) != 0 {
		// Since MSetNX is atomic we don't need to worry about the other operation on key between the Get and MSetNX, but
		// if we fail to lock a single key we fail to lock all of them. TODO this is a problem for large number of keys and we should
		// split them into batches
		var r *redis.BoolCmd
		if len(keysToLock) == 1 {
			// If there is only one key to lock, use SetNX instead of Pipe(MSetNX, ExpireLT) to avoid the overhead of creating a pipeline
			for k, v := range keysToLock {
				r = c.redisClient.SetNX(ctx, k, v, SentinelTTL)
			}
		} else {
			pipe := c.redisClient.Pipeline()
			r = pipe.MSetNX(ctx, keysToLock)
			for k := range keysToLock {
				pipe.ExpireLT(ctx, k, SentinelTTL) // We only set expiration time if the current time is greater than SentinelTTL
			}
			_, err = pipe.Exec(ctx)
			if err != nil {
				uclog.Verbosef(ctx, "Cache[%v] miss on keys %v lock fail %v", c.cacheName, keysToLock, err)
				return val, conflicts, sentinels, ucerr.Wrap(err)
			}
		}
		if v, err := r.Result(); v && err == nil {
			uclog.Verbosef(ctx, "Cache[%v] miss on keys %v sentinel set %v", c.cacheName, keysToLock, sentinels)
		} else {
			for i := range sentinels {
				sentinels[i] = NoLockSentinel
			}
			uclog.Verbosef(ctx, "Cache[%v] miss on keys %v sentinel not set  due to conflict", c.cacheName, keysToLock)
		}
	}

	return val, conflicts, sentinels, nil
}

// GetValue gets the value in CacheKey (if any) and tries to lock the key for Read is lockOnMiss = true
func (c *RedisClientCacheProvider) GetValue(ctx context.Context, keyIn Key, lockOnMiss bool) (*string, *string, Sentinel, bool, error) {
	v, conflicts, s, err := c.GetValues(ctx, []Key{keyIn}, []bool{lockOnMiss})

	var value *string
	lock := NoLockSentinel
	var conflict *string

	if err == nil && len(v) > 0 {
		value = v[0]
		lock = s[0]
		conflict = conflicts[0]
	}
	return value, conflict, lock, false, ucerr.Wrap(err)
}

// DeleteValue deletes the value(s) in passed in keys
func (c *RedisClientCacheProvider) DeleteValue(ctx context.Context, keysIn []Key, setTombstone bool, force bool) error {
	setTombstone = setTombstone && c.tombstoneTTL > 0 // don't actually set tombstone if tombstoneTTL is 0
	keysAll, err := getValidatedStringKeysFromCacheKeys(keysIn, c.prefix)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if c.readOnly {
		uclog.Verbosef(ctx, "Cache[%v] DeleteValue - skipping cache due to read only mode", c.cacheName)
		return nil
	}

	tombstoneValue := string(GenerateTombstoneSentinel()) // Generate a unique tombstone value

	if len(keysAll) != 0 {
		// If we are forcing the delete operation, the current value of the keys is not relevant
		if force {
			if !setTombstone {
				// If tombstone is not set, just delete the keys
				return c.redisClient.Del(ctx, keysAll...).Err()
			}
			if len(keysAll) == 1 {
				// If we have single key (common case), use Set instead of MSet to avoid the overhead of creating a pipeline
				uclog.Verbosef(ctx, "Cache[%v] DeleteValue - tombstoned keys %v", c.cacheName, keysAll[0])
				return c.redisClient.Set(ctx, keysAll[0], tombstoneValue, c.tombstoneTTL).Err()
			}
			// If we have multiple keys, use a pipeline to set the tombstone and expiration time
			pipe := c.redisClient.Pipeline()
			if err := multiSetWithPipe(ctx, pipe, keysAll, tombstoneValue, c.tombstoneTTL); err != nil {
				c.logError(ctx, err, "Cache[%v] error setting tombstone on key(s) %v -  %v", c.cacheName, keysAll, err)
				return ucerr.Wrap(err)
			}
			_, err = pipe.Exec(ctx)
			if err == nil {
				uclog.Verbosef(ctx, "Cache[%v] DeleteValue - tombstoned keys %v", c.cacheName, keysAll)
			}
			return ucerr.Wrap(err)
		}

		// If the operation is not forced we need to read current values of the keys and only delete them if they don't contain sentinel or tombstone
		batchSize := 2
		var end int
		for start := 0; start < len(keysAll); start += batchSize {
			end += batchSize
			if end > len(keysAll) {
				end = len(keysAll)
			}

			keys := keysAll[start:end]

			// Transactional function to only clear keys if they don't contain sentinel or tombstone
			txf := func(tx *redis.Tx) error {
				// Operation is committed only if the watched keys remain unchanged.
				_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
					values, err := c.redisClient.MGet(ctx, keys...).Result()
					if err != nil && err != redis.Nil {
						return ucerr.Wrap(err)
					}

					keysToDelete := []string{}
					for i, v := range values {
						if vS, ok := v.(string); ok && !c.sm.IsSentinelValue(vS) {
							keysToDelete = append(keysToDelete, keys[i])
						}
					}

					if len(keysToDelete) > 0 {
						if setTombstone {
							if err := multiSetWithPipe(ctx, pipe, keysToDelete, string(tombstoneValue), c.tombstoneTTL); err != nil {
								c.logError(ctx, err, "Cache[%v] error setting tombstone on key(s) %v -  %v", c.cacheName, keysToDelete, err)
								return ucerr.Wrap(err)
							}
							uclog.Verbosef(ctx, "Cache[%v] DeleteValue - tombstoned keys %v", c.cacheName, keysToDelete)
							return nil
						}
						if err := pipe.Del(ctx, keysToDelete...).Err(); err != nil {
							return ucerr.Wrap(err)
						}
						uclog.Verbosef(ctx, "Cache[%v] DeleteValue - deleted keys %v", c.cacheName, keysToDelete)
					}

					return nil
				})
				return ucerr.Wrap(err)
			}

			// Retry if the key has been changed.
			success := false
			for range maxRdConflictRetries {
				err := c.redisClient.Watch(ctx, txf, keys...)
				if err == nil {
					// Success.
					success = true
					break
				}
				if errors.Is(err, redis.TxFailedErr) {
					// Optimistic lock lost. Retry.
					uclog.Verbosef(ctx, "Cache[%v] DeleteValue - retry on keys %v", c.cacheName, keys)
					continue
				}
				// Return any other error.
				return ucerr.Wrap(err)
			}
			if !success {
				uclog.Warningf(ctx, "Cache[%v] Failed delete values - reached maximum number of retries on keys %v", c.cacheName, keys)
				return ucerr.New("Failed to DeleteValue reached maximum number of retries")
			}
		}
	}
	return nil
}

// AddDependency adds the given cache key(s) as dependencies of an item represented by by key
func (c *RedisClientCacheProvider) AddDependency(ctx context.Context, keysIn []Key, values []Key, ttl time.Duration) error {
	keysAll, err := getValidatedStringKeysFromCacheKeys(keysIn, c.prefix)
	if err != nil {
		return ucerr.Wrap(err)
	}
	i := make([]any, 0, len(values))
	for _, v := range values {
		if v != "" { // Skip empty values
			i = append(i, string(v))
		}
	}

	if len(keysAll) == 0 {
		return ucerr.New("No key provided to AddDependency")
	}

	if len(keysAll) > 500 {
		return ucerr.Errorf("Too many keys %v provided to to AddDependency", len(keysAll))
	}

	if len(i) == 0 {
		return ucerr.New("No non blank values provided to AddDependency")
	}

	if c.readOnly {
		uclog.Verbosef(ctx, "Cache[%v] AddDependency - skipping cache due to read only mode", c.cacheName)
		return nil
	}

	// We are using redis's WRONGTYPE error to detect when a key has been tomstoned. This depends on ClearDependencies using a transaction to detect modification to the
	// set of dependencies and restarting if a new dependency has been added by AddDependencies.

	batchSize := 100
	var end int
	for start := 0; start < len(keysAll); start += batchSize {
		end += batchSize
		if end > len(keysAll) {
			end = len(keysAll)
		}

		keys := keysAll[start:end]

		pipe := c.redisClient.Pipeline()
		_, err := pipe.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			for _, key := range keys {
				if err := pipe.SAdd(ctx, key, i...).Err(); err != nil {
					return ucerr.Wrap(err)
				}
				// Bump expiration which mean that the expired member accumulate in the set and we need to clean them up. Using ZSET sorted by timestamps may be a better option
				if err := pipe.Expire(ctx, key, ttl).Err(); err != nil {
					return ucerr.Wrap(err)
				}
			}
			return nil
		})

		if err != nil {

			// Check for "WRONGTYPE Operation against a key holding the wrong kind of value" in this case the set has been replaced by a tombstone
			if redis.HasErrorPrefix(err, "WRONGTYPE") {
				uclog.Verbosef(ctx, "Cache[%v] AddDependency - key is tombstoned %v", c.cacheName, keys)
				return ucerr.New("Can't add dependency: key is tombstoned")
			}
			uclog.Warningf(ctx, "Cache[%v] AddDependency - Failed to add dependencies - on keys %v with err %v", c.cacheName, keys, err)
			return ucerr.Wrap(err)
		}
	}

	return nil
}

// ClearDependencies clears the dependencies of an item represented by key and removes all dependent keys from the cache
func (c *RedisClientCacheProvider) ClearDependencies(ctx context.Context, keyIn Key, setTombstone bool) error {
	setTombstone = setTombstone && c.tombstoneTTL > 0 // don't actually set tombstone if tombstoneTTL is 0
	key, err := getValidatedStringKeyFromCacheKey(keyIn, c.prefix, "ClearDependencies")
	if err != nil {
		return ucerr.Wrap(err)
	}

	if c.readOnly {
		uclog.Verbosef(ctx, "Cache[%v] ClearDependencies - skipping cache due to read only mode", c.cacheName)
		return nil
	}

	tombstoneValue := string(GenerateTombstoneSentinel()) // Generate a unique tombstone value

	// Using optimistic concurrency control to clear the dependent keys for each value in key. This may cause us to flush more keys than needed but
	// never miss one. We tombstone the key to prevent new dependencies from being added from reads that might have been in flight during deletion.

	// Transactional function to read list of dependent keys and delete them
	txf := func(tx *redis.Tx) error {
		// Operation is committed only if the watched keys remain unchanged.
		_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			keys := []string{}
			var isTombstone bool
			if m, err := tx.SMembersMap(ctx, string(key)).Result(); err == nil {
				keys = make([]string, 0, len(m))
				for k := range m {
					keys = append(keys, k)
				}
			} else if v, err := c.redisClient.Get(ctx, key).Result(); err == nil && IsTombstoneSentinel(v) {
				isTombstone = true
			}

			if len(keys) != 0 {
				if err := pipe.Del(ctx, keys...).Err(); err != nil && err != redis.Nil {
					return ucerr.Wrap(err)
				}
				uclog.Verbosef(ctx, "Cache[%v] cleared dependencies (deleted) %v keys", c.cacheName, keys)
			}
			if setTombstone {
				if err := pipe.Set(ctx, key, tombstoneValue, c.tombstoneTTL).Err(); err != nil {
					return ucerr.Wrap(err)
				}
				uclog.Verbosef(ctx, "Cache[%v] ClearDependencies set tombstone for %v", c.cacheName, key)
			} else if !isTombstone {
				if err := pipe.Del(ctx, key).Err(); err != nil && err != redis.Nil {
					return ucerr.Wrap(err)
				}
				uclog.Verbosef(ctx, "Cache[%v] cleared dependency key %v", c.cacheName, key)
			}
			return nil
		})
		return ucerr.Wrap(err)
	}

	// Retry if the key has been changed.
	for range maxRdConflictRetries {
		err := c.redisClient.Watch(ctx, txf, key)
		if err == nil {
			// Success.
			return nil
		}
		if errors.Is(err, redis.TxFailedErr) {
			// Optimistic lock lost. Retry.
			uclog.Verbosef(ctx, "Cache[%v] ClearDependencies - retry on key %v", c.cacheName, key)
			continue
		}
		// Return any other error.
		return ucerr.Wrap(err)
	}
	uclog.Warningf(ctx, "Failed to clear dependencies - reached maximum number of retries on keys %v", key)
	return ucerr.New("Clear dependencies reached maximum number of retries")
}

// Flush flushes the cache (applies only to the tenant for which the client was created)
func (c *RedisClientCacheProvider) Flush(ctx context.Context, prefix string, flushTombstones bool) error {
	if c.readOnly {
		uclog.Verbosef(ctx, "Cache[%v] Flush - skipping cache due to read only mode", c.cacheName)
		return nil
	}

	pipe := c.redisClient.Pipeline()
	iter := c.redisClient.Scan(ctx, 0, prefix+"*", 1000).Iterator()

	uclog.Verbosef(ctx, "Cache[%v] flushing prefix %v", c.cacheName, prefix)

	var keysChecked int
	_, err := pipe.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		keysIn := make([]string, 0, flushBatchSize)
		keysToDelete := make([]string, 0, flushBatchSize)
		for iter.Next(ctx) {
			keysChecked++
			keysIn = append(keysIn, iter.Val())
			if len(keysIn) == flushBatchSize {
				if !flushTombstones {
					// We are doing the mgets outside the pipe and then pipelining deletes
					values, err := c.redisClient.MGet(ctx, keysIn...).Result()
					if err != nil && err != redis.Nil {
						return ucerr.Wrap(err)
					}

					for i, v := range values {
						vS, ok := v.(string)
						if ok && !IsTombstoneSentinel(vS) {
							keysToDelete = append(keysToDelete, keysIn[i])
						}
					}
				} else {
					keysToDelete = append(keysToDelete, keysIn...)
				}
				if len(keysToDelete) > 0 {
					pipe.Del(ctx, keysToDelete...)
					uclog.Verbosef(ctx, "Cache[%v] flushed %v keys", c.cacheName, len(keysToDelete))
				}
				keysToDelete = keysToDelete[:0]
				keysIn = keysIn[:0]
			}
		}
		uclog.Verbosef(ctx, "Cache[%v] checked %v keys for flush", c.cacheName, keysChecked)

		if len(keysIn) != 0 {
			if !flushTombstones {
				values, err := c.redisClient.MGet(ctx, keysIn...).Result()
				if err != nil && err != redis.Nil {
					return ucerr.Wrap(err)
				}

				for i, v := range values {
					vS, ok := v.(string)
					if ok && !IsTombstoneSentinel(vS) {
						keysToDelete = append(keysToDelete, keysIn[i])
					}
				}
			} else {
				keysToDelete = append(keysToDelete, keysIn...)
			}
			if len(keysToDelete) > 0 {
				pipe.Del(ctx, keysToDelete...)
				uclog.Verbosef(ctx, "Cache[%v] flushed %v keys", c.cacheName, len(keysToDelete))
			}
		}

		if iter.Err() != nil {
			return ucerr.Wrap(iter.Err())
		}

		return nil
	})

	if err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// GetCacheName returns the name of the cache
func (c *RedisClientCacheProvider) GetCacheName(ctx context.Context) string {
	return c.cacheName
}

// RegisterInvalidationHandler registers a handler for cache invalidation
func (c *RedisClientCacheProvider) RegisterInvalidationHandler(ctx context.Context, handler InvalidationHandler, key Key) error {
	return ucerr.Errorf("RegisterInvalidationHandler not supported for RedisClientCacheProvider")
}

// LogKeyValues logs the key values in the cache with given prefix
func (c *RedisClientCacheProvider) LogKeyValues(ctx context.Context, prefix string) error {

	iter := c.redisClient.Scan(ctx, 0, prefix+"*", 1000).Iterator()

	for iter.Next(ctx) {
		val := c.redisClient.Get(ctx, iter.Val())
		err := val.Err()
		var valSet *redis.StringSliceCmd
		if err != nil && redis.HasErrorPrefix(err, "WRONGTYPE") {
			// Check for "WRONGTYPE Operation against a key holding the wrong kind of value" and use SMEMBERS instead
			valSet = c.redisClient.SMembers(ctx, iter.Val())
			err = valSet.Err()
		}

		if err != nil {
			uclog.Errorf(ctx, "Cache[%v] failed to log value of key %v with error %v", c.cacheName, iter.Val(), err)
		} else {
			if valSet != nil {
				uclog.Verbosef(ctx, "Cache[%v] key %v value %v", c.cacheName, iter.Val(), valSet.Val())
			} else {
				uclog.Verbosef(ctx, "Cache[%v] key %v value %v", c.cacheName, iter.Val(), val.Val())
			}
		}
	}

	if iter.Err() != nil {
		return ucerr.Wrap(iter.Err())
	}

	return nil
}

func (c *RedisClientCacheProvider) coerceValueToInt64(key string, value any) (int64, error) {
	if value == nil {
		return 0, nil
	}

	strValue, ok := value.(string)
	if !ok {
		return 0,
			ucerr.Errorf(
				"Cache[%v] key %v - value %v is not a string",
				c.cacheName,
				key,
				value,
			)
	}

	intValue, err := strconv.ParseInt(strValue, 10, 64)
	if err != nil {
		return 0,
			ucerr.Errorf(
				"Cache[%v] key %v - value %v cannot be converted to an int64",
				c.cacheName,
				key,
				value,
			)
	}

	return intValue, nil
}

// ReleaseRateLimitSlot will release the earliest rate limit slot for the provided set of keys
func (c *RedisClientCacheProvider) ReleaseRateLimitSlot(
	ctx context.Context,
	keysIn []RateLimitKey,
) (int64, error) {
	keys, err := getValidatedStringKeysFromCacheKeys(keysIn, c.prefix)
	if err != nil {
		return 0, ucerr.Wrap(err)
	}

	values, err := c.redisClient.MGet(ctx, keys...).Result()
	if err != nil && err != redis.Nil {
		return 0, ucerr.Wrap(err)
	}

	var totalSlots int64
	var bucketsWithValues []int

	for i := range keys {
		value, err := c.coerceValueToInt64(keys[i], values[i])
		if err != nil {
			return 0, ucerr.Wrap(err)
		}

		if value > 0 {
			bucketsWithValues = append(bucketsWithValues, i)
			totalSlots += value
		}
	}

	if totalSlots > 0 {
		for _, i := range bucketsWithValues {
			bucketSlots, err := c.redisClient.Decr(ctx, keys[i]).Result()
			if err != nil {
				return 0, ucerr.Wrap(err)
			}

			if bucketSlots >= 0 {
				break
			}

			_, err = c.redisClient.Incr(ctx, keys[i]).Result()
			if err != nil {
				return 0, ucerr.Wrap(err)
			}
		}
		totalSlots--
	}

	return totalSlots, nil
}

// ReserveRateLimitSlot will return if a rate limit slot can be reserved, given the specified limit
// and provided keys, actually reserving the slot if requested
func (c *RedisClientCacheProvider) ReserveRateLimitSlot(
	ctx context.Context,
	keysIn []RateLimitKey,
	limit int64,
	ttl time.Duration,
	takeSlot bool,
) (bool, int64, error) {
	// validate the bucket keys and retrieve the associated values

	keys, err := getValidatedStringKeysFromCacheKeys(keysIn, c.prefix)
	if err != nil {
		return false, 0, ucerr.Wrap(err)
	}
	finalBucketIndex := len(keys) - 1

	values, err := c.redisClient.MGet(ctx, keys...).Result()
	if err != nil && err != redis.Nil {
		return false, 0, ucerr.Wrap(err)
	}

	// count the number of used slots of all but the final bucket

	var totalSlots int64
	for i := range finalBucketIndex {
		value, err := c.coerceValueToInt64(keys[i], values[i])
		if err != nil {
			return false, 0, ucerr.Wrap(err)
		}

		if value > 0 {
			totalSlots += value
		}
	}

	// keep track of the number of used slots in the final bucket separately

	finalBucketSlots, err := c.coerceValueToInt64(keys[finalBucketIndex], values[finalBucketIndex])
	if err != nil {
		return false, 0, ucerr.Wrap(err)
	}
	totalSlots += finalBucketSlots

	// return if we do need to take a new slot or have reached the limit

	if !takeSlot {
		return totalSlots < limit, totalSlots, nil
	}

	if totalSlots >= limit {
		return false, totalSlots, nil
	}

	// add a slot to the final bucket

	totalSlots -= finalBucketSlots
	finalBucketSlots, err = c.redisClient.Incr(ctx, keys[finalBucketIndex]).Result()
	if err != nil {
		return false, 0, ucerr.Wrap(err)
	}
	totalSlots += finalBucketSlots

	// set the ttl for the final bucket if this is the first slot in that bucket

	if values[finalBucketIndex] == nil {
		c.redisClient.Expire(ctx, keys[finalBucketIndex], ttl)
	}

	// remove a slot from the final bucket and return false if we have exceeded the limit

	if totalSlots > limit {
		totalSlots -= finalBucketSlots
		finalBucketSlots, err = c.redisClient.Decr(ctx, keys[finalBucketIndex]).Result()
		if err != nil {
			return false, 0, ucerr.Wrap(err)
		}
		totalSlots += finalBucketSlots

		return false, totalSlots, nil
	}

	// return true if we have not exceeded the limit

	return totalSlots <= limit, totalSlots, nil
}

// SupportsRateLimits returns true if rate limiting is supported
func (*RedisClientCacheProvider) SupportsRateLimits(context.Context) bool {
	return true
}

func (c RedisClientCacheProvider) String() string {
	return fmt.Sprintf("RedisClientCacheProvider(%s):%v", c.cacheName, *c.redisClient)
}
