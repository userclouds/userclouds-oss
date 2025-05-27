package cache

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/ucmetrics"
)

const (
	defaultInvalidationDelay = 100 * time.Millisecond
	cacheFlushSuffix         = "*"
	cacheDependencySuffixT   = "?T"
	cacheDependencySuffixF   = "?F"
	cacheLogSuffix           = "~"
)

var (
	invalidationMsgLatencySeconds = ucmetrics.CreateHistogram("cache", "invalidation_latency", "time taken for invalidation message to arrive", "name", "source_region")
	invalidationKeysCount         = ucmetrics.CreateHistogram("cache", "invalidation_keys_count", "number of keys in invalidation message", "name", "source_region")
	invalidationMsgCount          = ucmetrics.CreateCounter("cache", "invalidation_count", "number of invalidation messages received", "name", "source_region")
)

// postInvalidationHandlerT is type for a function that is called after the cache is invalidated and all invalidation handlers are called to perform any post invalidation tasks
type postInvalidationHandlerT func(ctx context.Context, id uuid.UUID, m invalidateMessage) error

// InvalidationWrapperOption defines a way to pass optional configuration parameters.
type InvalidationWrapperOption interface {
	apply(*cacheInvalidationWrapperConfig)
}

type optFunc func(*cacheInvalidationWrapperConfig)

func (o optFunc) apply(po *cacheInvalidationWrapperConfig) {
	o(po)
}

// OnMachine specifies that the cache exists on the machine vs off machine
func OnMachine() InvalidationWrapperOption {
	return optFunc(func(po *cacheInvalidationWrapperConfig) {
		po.onMachine = true
	})
}

// Layered specifies that the cache exists both on the machine and off machine
func Layered() InvalidationWrapperOption {
	return optFunc(func(po *cacheInvalidationWrapperConfig) {
		po.layered = true
	})
}

// InvalidationHandlersOnlySub specifies that only the invalidation handlers should be called for invalidation messages received from the subscription
func InvalidationHandlersOnlySub() InvalidationWrapperOption {
	return optFunc(func(po *cacheInvalidationWrapperConfig) {
		po.invalidationHandlersOnlySub = true
	})
}

// InvalidationHandlersLocalPublish specifies that only the message to the local region should always be published with InvalidationHandlersOnly code
func InvalidationHandlersLocalPublish(invalidationChannelName string, filters []string) InvalidationWrapperOption {
	return optFunc(func(po *cacheInvalidationWrapperConfig) {
		po.invalidationHandlersLocalPublish = true
		po.subCacheName = invalidationChannelName
		po.subCacheFilters = filters
	})
}

// PostInvalidationHandler set a handler to be called after invalidation is performed and invalidation handlers have been called
func PostInvalidationHandler(h postInvalidationHandlerT) InvalidationWrapperOption {
	return optFunc(func(po *cacheInvalidationWrapperConfig) {
		po.postInvalidationHandler = h
	})
}

// InvalidationDelay allows specification of the delay after invalidation message is sent
func InvalidationDelay(invalidationDelay time.Duration) InvalidationWrapperOption {
	return optFunc(func(po *cacheInvalidationWrapperConfig) {
		po.invalidationDelay = invalidationDelay
	})
}

// SubCacheName allows specification of another cache which is being used underneath this cache
func SubCacheName(subCacheName string) InvalidationWrapperOption {
	return optFunc(func(po *cacheInvalidationWrapperConfig) {
		po.subCacheName = subCacheName
	})
}

// RegionalChannelName allows override of the default channel name (RegionalRedisCacheName) for testing layered caches
func RegionalChannelName(regionalChannelName string) InvalidationWrapperOption {
	return optFunc(func(po *cacheInvalidationWrapperConfig) {
		po.regionalChannelName = regionalChannelName
	})
}

// cacheInvalidationWrapperConfig describes optional parameters for configuring invalidating cache wrapper
type cacheInvalidationWrapperConfig struct {
	invalidationDelay                time.Duration
	onMachine                        bool
	layered                          bool
	invalidationHandlersOnlySub      bool
	invalidationHandlersLocalPublish bool
	postInvalidationHandler          postInvalidationHandlerT
	subCacheName                     string
	subCacheFilters                  []string
	regionalChannelName              string // used in tests to override the default channel name (RegionalRedisCacheName) for testing layered caches
}

// InvalidationWrapper is on machine cache that has invalidation for writes that occur on other machines
type InvalidationWrapper struct {
	id                               uuid.UUID
	cache                            Provider
	comms                            CommunicationProvider
	onMachine                        bool
	invalidationHandlersOnlySub      bool
	invalidationHandlersLocalPublish bool
	invalidationDelay                time.Duration
	invalidationHandlers             map[Key][]InvalidationHandler
	postInvalidationHandler          postInvalidationHandlerT
	handlersLock                     sync.RWMutex
	subCacheName                     string
	subCacheFilters                  []string
}

// NewInvalidationWrapper creates a new LocalStoreCache
func NewInvalidationWrapper(cp Provider, comms CommunicationProvider, opts ...InvalidationWrapperOption) (*InvalidationWrapper, error) {

	// Get the optional parameters if any
	var to cacheInvalidationWrapperConfig
	for _, v := range opts {
		v.apply(&to)
	}

	if to.onMachine && to.invalidationDelay == 0 {
		to.invalidationDelay = defaultInvalidationDelay
	}

	l := &InvalidationWrapper{id: uuid.Must(uuid.NewV4()),
		cache:                            cp,
		comms:                            comms,
		onMachine:                        to.onMachine,
		invalidationHandlersOnlySub:      to.invalidationHandlersOnlySub,
		invalidationHandlersLocalPublish: to.invalidationHandlersLocalPublish,
		invalidationDelay:                to.invalidationDelay,
		invalidationHandlers:             map[Key][]InvalidationHandler{},
		postInvalidationHandler:          to.postInvalidationHandler,
		subCacheName:                     to.subCacheName,
		subCacheFilters:                  to.subCacheFilters,
	}

	if l.onMachine {
		if err := l.subscribeToInvalidation(); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}
	return l, nil
}
func (l *InvalidationWrapper) logError(ctx context.Context, err error, format string, args ...any) {
	if errors.Is(err, context.Canceled) {
		uclog.Warningf(ctx, format, args...)
	} else {
		uclog.Errorf(ctx, format, args...)
	}
}

func (l *InvalidationWrapper) subscribeToInvalidation() error {
	ctx := context.Background()
	err := l.comms.Subscribe(ctx, func(ctx context.Context, m invalidateMessage) {
		latency := time.Now().UTC().Sub(m.Timestamp)
		uclog.Verbosef(ctx, "Cache[%v] Subscriber %v received message invalidated keys %v code %v from %v [%v] latency: %v [%v]",
			l.GetCacheName(ctx), l.id, m.Keys, m.Code, m.SenderID, m.Region, latency, m.Timestamp)

		if m.SenderID == l.id || len(m.Keys) < 1 {
			return
		}

		if m.Code == invalidationHandlersMessage && !l.invalidationHandlersOnlySub {
			// This is a message for invalidation handlers only
			uclog.Verbosef(ctx, "Cache[%v] Subscriber %v discarding message for invalidation handlers only", l.GetCacheName(ctx), l.id)
			return
		}

		// Run this async because it sometimes up to 10ms to update the counters which delays the notification
		go func() {
			// only track metrics for invalidation messages that we process
			cn := l.GetCacheName(ctx)
			invalidationMsgCount.WithLabelValues(cn, m.Region).Inc()
			invalidationKeysCount.WithLabelValues(cn, m.Region).Observe(float64(len(m.Keys)))
			invalidationMsgLatencySeconds.WithLabelValues(cn, m.Region).Observe(latency.Seconds())
		}()

		if err := l.handleInvalidation(ctx, m); err != nil {
			l.logError(ctx, err, "Failed to handle invalidation message %v with %v", m, err)
		}
	})
	if err != nil {
		l.logError(ctx, err, "Failed to subscribe to cache %v with %v", l.GetCacheName(ctx), err)
	}
	return ucerr.Wrap(err)

}

func (l *InvalidationWrapper) handleInvalidation(ctx context.Context, m invalidateMessage) error {
	if !l.invalidationHandlersOnlySub {
		firstKey := string(m.Keys[0]) // caller should ensure that there is at least one key

		if len(m.Keys) == 1 && strings.HasSuffix(firstKey, cacheFlushSuffix) {
			// Process flushes first
			if err := l.cache.Flush(ctx, strings.TrimSuffix(firstKey, cacheFlushSuffix), false); err != nil {
				return ucerr.Errorf("Failed to flush invalidated prefix %v with %v", m.Keys, err)
			}
		} else if len(m.Keys) == 1 && (strings.HasSuffix(firstKey, cacheDependencySuffixT) || strings.HasSuffix(firstKey, cacheDependencySuffixF)) {
			// Process clearing of dependencies next
			setTombstone := strings.HasSuffix(firstKey, cacheDependencySuffixT)
			key := strings.TrimSuffix(strings.TrimSuffix(firstKey, cacheDependencySuffixF), cacheDependencySuffixT)
			if err := l.cache.ClearDependencies(ctx, Key(key), setTombstone); err != nil {
				return ucerr.Errorf("Failed to flush invalidated prefix %v with %v", m.Keys, err)
			}
		} else if len(m.Keys) == 1 && strings.HasSuffix(firstKey, cacheLogSuffix) {
			// Process logging of keys next
			if err := l.cache.LogKeyValues(ctx, strings.TrimSuffix(firstKey, cacheLogSuffix)); err != nil {
				return ucerr.Errorf("Failed to log keys with prefix %v with %v", m.Keys, err)
			}
			// No need to call the invalidation handlers
			return nil
		} else {
			// Otherwise delete all the keys that have been invalidated
			if err := l.cache.DeleteValue(ctx, m.Keys, true /* set tombstone to wait for db propagation */, true /* delete regardless of state */); err != nil {
				return ucerr.Errorf("Failed to delete invalidated keys %v with %v", m.Keys, err)
			}
		}
	}
	// call registered handlers when invalidation message is received
	l.handlersLock.RLock()
	defer l.handlersLock.RUnlock()
	for registeredKey, handlers := range l.invalidationHandlers {
		for _, k := range m.Keys {
			suffixFreeKey := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(string(k), cacheFlushSuffix), cacheDependencySuffixT), cacheDependencySuffixF)
			if strings.HasPrefix(string(registeredKey), suffixFreeKey) {
				for _, h := range handlers {
					if err := h(ctx, k, strings.HasSuffix(string(k), cacheFlushSuffix)); err != nil {
						l.logError(ctx, err, "Handler failed to handle invalidation message %v with %v", m, err)
					}
				}
			}
		}
	}
	// call post invalidation handler if registered
	if l.postInvalidationHandler != nil {
		if err := l.postInvalidationHandler(ctx, l.id, m); err != nil {
			l.logError(ctx, err, "Failed to run post invalidation handlers %v with %v", m, err)
		}
	}

	return nil

}

// Shutdown shuts down the cache communication provider listening for invalidations
func (l *InvalidationWrapper) Shutdown(ctx context.Context) {
	if l.comms != nil {
		l.comms.Shutdown(ctx)
	}
}

// GetValue gets the value in cache key (if any) and tries to lock the key for Read is lockOnMiss = true
func (l *InvalidationWrapper) GetValue(ctx context.Context, key Key, lockOnMiss bool) (*string, *string, Sentinel, bool, error) {
	return l.cache.GetValue(ctx, key, lockOnMiss)
}

// GetValues gets the value in cache keys (if any) and tries to lock the keys[i] for Read is lockOnMiss[i] = true
func (l *InvalidationWrapper) GetValues(ctx context.Context, keys []Key, lockOnMiss []bool) ([]*string, []*string, []Sentinel, error) {
	return l.cache.GetValues(ctx, keys, lockOnMiss)
}

// SetValue sets the value in cache key(s) to val with given expiration time if the sentinel matches lkey and returns true if the value was set
func (l *InvalidationWrapper) SetValue(ctx context.Context, lkey Key, keysToSet []Key, val string, sentinel Sentinel,
	ttl time.Duration) (bool, bool, error) {
	var keyset, conflict bool
	var err error
	if keyset, conflict, err = l.cache.SetValue(ctx, lkey, keysToSet, val, sentinel, ttl); err != nil {
		return keyset, conflict, ucerr.Wrap(err)
	}

	// Key is set to a new value or a conflict is detected - invalidate the value in the cache. For conflict this is only needed to invalidate any on machine
	// caches that might need to be refreshed as the cache itself has the sentinel in the key
	if (keyset || conflict) && IsInvalidatingSentinelValue(sentinel) {
		if err := l.invalidateValue(ctx, keysToSet); err != nil {
			return keyset, conflict, ucerr.Wrap(err)
		}
	}

	return keyset, conflict, ucerr.Wrap(err)
}

// DeleteValue deletes the value(s) in passed in keys, force is true also deletes keys with sentinel or tombstone values
func (l *InvalidationWrapper) DeleteValue(ctx context.Context, key []Key, setTombstone bool, force bool) error {

	if err := l.cache.DeleteValue(ctx, key, setTombstone, force); err != nil {
		return ucerr.Wrap(err)
	}

	return ucerr.Wrap(l.invalidateValue(ctx, key))
}

// WriteSentinel writes the sentinel value into the given keys, returns NoLockSentinel if it couldn't acquire the lock
func (l *InvalidationWrapper) WriteSentinel(ctx context.Context, stype SentinelType, keys []Key) (Sentinel, error) {
	return l.cache.WriteSentinel(ctx, stype, keys)
}

// ReleaseSentinel clears the sentinel value from the given keys
func (l *InvalidationWrapper) ReleaseSentinel(ctx context.Context, keys []Key, s Sentinel) {
	l.cache.ReleaseSentinel(ctx, keys, s)
}

// AddDependency adds the given cache key(s) as dependencies of an item represented by by key. Fails if any of the dependency keys passed in contain tombstone
func (l *InvalidationWrapper) AddDependency(ctx context.Context, keysIn []Key, dependentKey []Key, ttl time.Duration) error {
	return ucerr.Wrap(l.cache.AddDependency(ctx, keysIn, dependentKey, ttl))
}

// ClearDependencies clears the dependencies of an item represented by key and removes all dependent keys from the cache
func (l *InvalidationWrapper) ClearDependencies(ctx context.Context, key Key, setTombstone bool) error {

	if err := l.cache.ClearDependencies(ctx, key, setTombstone); err != nil {
		return ucerr.Wrap(err)
	}

	msgKey := key + cacheDependencySuffixT
	if !setTombstone {
		msgKey = key + cacheDependencySuffixF
	}

	return ucerr.Wrap(l.invalidateValue(ctx, []Key{Key(msgKey)}))
}

// Flush flushes the cache
func (l *InvalidationWrapper) Flush(ctx context.Context, prefix string, flushTombstones bool) error {

	if err := l.cache.Flush(ctx, prefix, flushTombstones); err != nil {
		return ucerr.Wrap(err)
	}

	return ucerr.Wrap(l.invalidateValue(ctx, []Key{Key(prefix + cacheFlushSuffix)}))
}

// RegisterInvalidationHandler registers a handler for cache invalidation. The invalidation handler is only called for invalidations of the given key that are initiated outside
// the current cache client. For invalidations initiated by the current cache client, the handler is not called and it is assumed that the caller performs the necessary
// operations of the callback.
func (l *InvalidationWrapper) RegisterInvalidationHandler(ctx context.Context, handler InvalidationHandler, key Key) error {
	l.handlersLock.Lock()
	l.invalidationHandlers[key] = append(l.invalidationHandlers[key], handler)
	l.handlersLock.Unlock()
	return nil
}

// GetCacheName returns the name of the cache
func (l *InvalidationWrapper) GetCacheName(ctx context.Context) string {
	return l.cache.GetCacheName(ctx)
}

// LogKeyValues triggers logging of the key values in the cache with given prefix. It sends a message to all subscribers to the invalidation channel,
// in case of central cache (like redis) the centralized invalidation subscriber in each region will log the key values. In case of a local on
// machine cache, each individual machine will log values in its cache
func (l *InvalidationWrapper) LogKeyValues(ctx context.Context, prefix string) error {

	if err := l.cache.LogKeyValues(ctx, prefix); err != nil {
		return ucerr.Wrap(err)
	}

	return ucerr.Wrap(l.invalidateValue(ctx, []Key{Key(prefix + cacheLogSuffix)}))
}

// Layered returns true if the cache provider is a multi-layered cache
func (l *InvalidationWrapper) Layered(ctx context.Context) bool {
	return l.cache.Layered(ctx)
}

func (l *InvalidationWrapper) invalidateValue(ctx context.Context, keys []Key) error {
	channelNames := []string{l.GetCacheName(ctx)}
	message, err := json.Marshal(
		invalidateMessage{
			SenderID:  l.id,
			Keys:      keys,
			Timestamp: time.Now().UTC(),
			Region:    string(region.Current()),
			Route:     l.subCacheName,
		})
	if err != nil {
		return ucerr.Wrap(err)
	}

	localMessage := message
	if l.invalidationHandlersLocalPublish {
		localMessage, err = json.Marshal(
			invalidateMessage{
				SenderID:  l.id,
				Keys:      keys,
				Timestamp: time.Now().UTC(),
				Region:    string(region.Current()),
				Code:      invalidationHandlersMessage,
				Route:     l.subCacheName,
				F:         l.subCacheFilters,
			})
		if err != nil {
			return ucerr.Wrap(err)
		}

		if l.subCacheName != "" && filterKeys(keys, l.subCacheFilters) {
			channelNames = append([]string{l.subCacheName}, channelNames...)
		}
	}

	uclog.Verbosef(ctx, "Cache[%s] Subscriber[%v] sending invalidate message for key %s to %v", l.GetCacheName(ctx), l.id, keys, channelNames)

	sendInvalidationToLocalCh := l.onMachine || /* in memory cache running on local machine */
		l.invalidationHandlersLocalPublish || /* off machine cache that needs to notify invalidation handlers on local machines */
		l.subCacheName != "" /* off machine cache that sits on the top of in memory cache and need to notify worker to send invalidation to other regions */
	err = l.comms.Publish(ctx, channelNames, sendInvalidationToLocalCh, string(message), string(localMessage))
	if err != nil {
		return ucerr.Wrap(err)
	}

	// If specified wait for the invalidation to propagate to other machines
	if l.invalidationDelay > 0 {
		time.Sleep(l.invalidationDelay)
	}

	return nil
}

func filterKeys(keys []Key, filters []string) bool {
	if len(filters) > 0 {
		found := false
		for _, f := range filters {
			for _, k := range keys {
				if strings.HasSuffix(string(k), cacheFlushSuffix) || strings.Contains(string(k), f) {
					found = true
					break
				}
			}
		}
		return found
	}
	return true
}

// ReleaseRateLimitSlot will release the earliest rate limit slot for the provided set of keys
func (l *InvalidationWrapper) ReleaseRateLimitSlot(ctx context.Context, keys []RateLimitKey) (int64, error) {
	totalSlots, err := l.cache.ReleaseRateLimitSlot(ctx, keys)
	if err != nil {
		return 0, ucerr.Wrap(err)
	}

	return totalSlots, nil
}

// ReserveRateLimitSlot will return if a rate limit slot can be reserved, given the specified
// limit and provided keys, actually reserving the slot if requested
func (l *InvalidationWrapper) ReserveRateLimitSlot(
	ctx context.Context,
	keys []RateLimitKey,
	limit int64,
	ttl time.Duration,
	takeSlot bool,
) (bool, int64, error) {
	reserved, totalSlots, err := l.cache.ReserveRateLimitSlot(ctx, keys, limit, ttl, takeSlot)
	if err != nil {
		return false, 0, ucerr.Wrap(err)
	}

	return reserved, totalSlots, nil
}

// SupportsRateLimits returns true if rate limiting is supported
func (l *InvalidationWrapper) SupportsRateLimits(ctx context.Context) bool {
	return l.cache.SupportsRateLimits(ctx)
}

// RunCrossRegionInvalidations creates a CacheInvalidationWrapper that will receive invalidation message for the off machine regional cache
// and actually invalidate keys in it. It shouldn't be used for anything else and is called once per region from worker process. It also propagates
// invalidation messages from in the in region cache to other regions. It connects to redis instances in all regions.
func RunCrossRegionInvalidations(ctx context.Context, cc *Config, regionalName string, globalName string, opts ...InvalidationWrapperOption) (*InvalidationWrapper, CommunicationProvider) {
	comms, err := NewRedisCacheCommunicationProvider(ctx, cc, true, true, regionalName)
	if err != nil || comms == nil {
		uclog.Errorf(ctx, "failed to create redis cache communication provider for in region invalidation: %v", err)
		return nil, nil
	}

	commsG, err := NewRedisCacheCommunicationProvider(ctx, cc, true, false, globalName)
	if err != nil || commsG == nil {
		uclog.Errorf(ctx, "failed to create redis cache communication provider for cross region invalidation: %v", err)
		return nil, nil
	}

	redisCfg := cc.GetLocalRedis()
	lc, err := GetRedisClient(ctx, redisCfg)
	if err != nil {
		uclog.Errorf(ctx, "failed to create local redis client for cross region[%v]: %v", redisCfg, err)
		return nil, nil
	}

	// We need to invalidate the keys in this regions redis machine when we receive a message on the global channel. After invalidation in region redis machien,
	// we need to send a message to the to the in region channel so that the local caches can invalidate the keys (this way the in region cache is always up to date
	// before FE reads it after invalidating local caches)
	combOpts := append(opts, OnMachine(), PostInvalidationHandler(
		func(ctx context.Context, id uuid.UUID, m invalidateMessage) error {
			m.SenderID = id // reset the id to ourselves (ie worker) so we can ignore this message when we receive it
			if m.Code == globalInvalidationHandlersMessage {
				m.Code = invalidationHandlersMessage
			}

			// Reroute the message to appropriate regional channel for layered caches (is IDPCache or )
			channelName := regionalName
			if m.Route != "" {
				channelName = m.Route
				// if we are routing the message, check the filters and only send the message if the filters match
				if !filterKeys(m.Keys, m.F) {
					uclog.Verbosef(ctx, "Cache[%v] Subscriber %v [WI:%v] ignoring message for in region invalidation because filters didn't match: %v", channelName, id, region.Current(), m)
					return nil
				}
			}
			message, err := json.Marshal(m)
			if err != nil {
				uclog.Errorf(ctx, "Cache[%v] Subscriber %v [WI:%v] failed to marshal message [%v] for in region invalidation: %v", channelName, id, region.Current(), m, err)
				return ucerr.Wrap(err)
			}
			uclog.Verbosef(ctx, "Cache[%v] Subscriber %v [WI:%v] sending message for in region invalidation: %v", channelName, id, region.Current(), string(message))
			if err := comms.Publish(ctx, []string{channelName}, true, string(message), string(message)); err != nil {
				uclog.Errorf(ctx, "Cache[%v] Subscriber %v [WI:%v] failed to publish message[%v] for in region invalidation: %v", channelName, id, region.Current(), string(message), err)
				return ucerr.Wrap(err)
			}
			return nil
		}))
	inRegionRedisCache, err := NewInvalidationWrapper(
		NewRedisClientCacheProvider(lc, regionalName), // In region redis cache
		commsG, // Global channel provider
		combOpts...,
	)
	if err != nil {
		uclog.Errorf(ctx, "failed to create local invalidation wrapper for cross region[%v]: %v. Can't recieve cross region messages and invalidate in region cache", redisCfg, err)
		return nil, nil
	}

	uclog.Verbosef(ctx, "Cache[%v] to [%v] Invalidator ID %v region[%v]", globalName, regionalName, inRegionRedisCache.id, region.Current())

	id := uuid.Must(uuid.NewV4())
	// We need to subscribe to the in region channel and then propogate the message to the global channel (as long as we are not the sender)
	err = comms.Subscribe(ctx, func(ctx context.Context, m invalidateMessage) {
		latency := time.Now().UTC().Sub(m.Timestamp)
		uclog.Verbosef(ctx, "Cache[%v] Subscriber %v [W:GP:%v] received message invalidated keys %v from %v [%v] latency: %v [%v]",
			regionalName, id, region.Current(), m.Keys, m.SenderID, m.Region, latency, m.Timestamp)

		if m.SenderID == inRegionRedisCache.id || len(m.Keys) < 1 {
			return
		}

		// We should never propogate messages from regional to global channel if they didn't originate from this region
		if m.Region != string(region.Current()) {
			uclog.Warningf(ctx, "Cache[%v] Subscriber %v [W:GP:%v] received message for region %v but we are in region %v. Ignoring message - are you running two workers ?", regionalName, id, region.Current(), m.Region, region.Current())
			return
		}

		// Reset the code from invalidationHandlersMessage to globalInvalidationHandlersMessage because we want other regions to invalidate the regional cache
		// the worker in that region will reset the code back invalidationHandlersMessage before publishing the message to the in regional channel
		if m.Code == invalidationHandlersMessage {
			m.Code = globalInvalidationHandlersMessage
		}
		m.SenderID = inRegionRedisCache.id
		message, err := json.Marshal(m)
		if err != nil {
			uclog.Errorf(ctx, "Cache[%v] Subscriber %v [W:%v] failed to marshal message [%v] for cross region invalidation: %v", regionalName, id, region.Current(), m, err)
			return
		}

		uclog.Verbosef(ctx, "Cache[%v] Subscriber %v [W:GP:%v] sending message for cross region invalidation[%v]: %v", regionalName, id, region.Current(), globalName, string(message))
		if err := commsG.Publish(ctx, []string{globalName}, false, string(message), ""); err != nil {
			uclog.Errorf(ctx, "Cache[%v] Subscriber %v [W:%v] failed to publish message[%v] for cross region invalidation: %v", regionalName, id, region.Current(), string(message), err)
			return
		}
	})
	if err != nil {
		uclog.Errorf(ctx, "failed to subscribe to in region redis channel. Unable to propogate regional messages to cross region channel: %v", err)
		return nil, nil
	}

	uclog.Verbosef(ctx, "Cache[%v] to [%v] Propogator ID %v region [%v]", regionalName, globalName, id, region.Current())

	return inRegionRedisCache, comms
}

// RunInRegionLocalHandlersSubscriber creates a CacheInvalidationWrapper that will receive invalidation message for regional channel (RegionalRedisCacheName)
// and will only run invalidation handlers on the local machine (ie it doesn't invalidate the redis cache). It is used for edges cache on Authz machines
func RunInRegionLocalHandlersSubscriber(ctx context.Context, cc *Config, handlerChannel string, opts ...InvalidationWrapperOption) *InvalidationWrapper {
	comms, err := NewRedisCacheCommunicationProvider(ctx, cc, true, true, handlerChannel)
	if err != nil || comms == nil {
		uclog.Errorf(ctx, "failed to create redis cache communication provider for in region channel: %v", err)
		return nil
	}
	redisCfg := cc.GetLocalRedis()
	lc, err := GetRedisClient(ctx, redisCfg)
	if err != nil {
		uclog.Errorf(ctx, "failed to create local redis client for in region channel[%v]: %v", redisCfg, err)
		return nil
	}
	combOpts := append(opts, OnMachine(), InvalidationHandlersOnlySub())
	sharedCache, err := NewInvalidationWrapper(
		NewRedisClientCacheProvider(lc, handlerChannel),
		comms,
		combOpts...,
	)
	if err != nil {
		uclog.Errorf(ctx, "failed to create cache invalidation wrapper for in region invalidation handlers subscription: %v", err)
		return nil
	}
	return sharedCache
}

// InitializeInvalidatingCacheFromConfig creates a CacheInvalidationWrapper from the given config and CommsProvider
func InitializeInvalidatingCacheFromConfig(ctx context.Context, cc *Config, cacheName string, cachePrefixValue string, opts ...InvalidationWrapperOption) (Provider, error) {
	if cc == nil || cc.RedisCacheConfig == nil {
		return nil, nil
	}

	// Get the optional parameters if any
	var to cacheInvalidationWrapperConfig
	for _, v := range opts {
		v.apply(&to)
	}

	if to.layered && to.onMachine {
		return nil, ucerr.Errorf("Can't specify Layered cache and On machine at the same time")
	}

	if to.onMachine && to.invalidationDelay == 0 {
		to.invalidationDelay = defaultInvalidationDelay
	}

	if to.layered {
		regionalChannelName := RegionalRedisCacheName
		if to.regionalChannelName != "" {
			regionalChannelName = to.regionalChannelName
		}

		cp1, err := InitializeInvalidatingCacheFromConfig(ctx, cc, cacheName, cachePrefixValue, OnMachine(), InvalidationDelay(to.invalidationDelay))
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		cp2, err := InitializeInvalidatingCacheFromConfig(ctx, cc, regionalChannelName, cachePrefixValue, SubCacheName(cacheName))
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		lp := NewLayeringWrapper(cp1, cp2)
		return lp, nil
	}

	// We currently only support one comms provider so we can hardcode it
	comms, err := NewRedisCacheCommunicationProvider(ctx, cc, to.onMachine, true, cacheName)
	if err != nil || comms == nil {
		uclog.Errorf(ctx, "failed to create redis cache communication provider: %v", err)
		if err == nil {
			// We want to make sure we return an error if we failed to create the comms provider,
			// otherwise the cache object are initialized with nil values which causes panics later.
			return nil, ucerr.Errorf("failed to create redis cache communication provider")
		}
		return nil, ucerr.Wrap(err)
	}

	var cp Provider

	// Hardcode the mapping between onMachine == InMemory and offMachine == Redis since that's what we support for now
	if to.onMachine {
		cp = NewInMemoryClientCacheProvider(cacheName)
	} else {
		redisCfg := cc.GetLocalRedis()
		lc, err := GetRedisClient(ctx, redisCfg)
		if err != nil {
			uclog.Errorf(ctx, "failed to create local redis client[%v]: %v", redisCfg, err)
			return nil, ucerr.Wrap(err)
		}
		cp = NewRedisClientCacheProvider(lc, cacheName, KeyPrefixRedis(cachePrefixValue))
	}

	// Create the invalidating cache using CommsProvider and CacheProvider
	invalidatingCache, err := NewInvalidationWrapper(cp, comms, opts...)
	if err != nil {
		uclog.Errorf(ctx, "failed to create cache invalidation wrapper: %v", err)
		return nil, ucerr.Wrap(err)
	}

	uclog.Verbosef(ctx, "Cache[%v] invalidating wrapper ID %v", cacheName, invalidatingCache.id)

	return invalidatingCache, nil
}
