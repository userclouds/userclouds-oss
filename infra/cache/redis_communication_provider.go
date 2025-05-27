package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"

	"userclouds.com/infra/request"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

const (
	maxLocalRetries                   = 3
	invalidationHandlersMessage       = 1
	globalInvalidationHandlersMessage = 2
)

type invalidateMessage struct {
	SenderID  uuid.UUID `json:"senderID" yaml:"senderID"`
	Keys      []Key     `json:"key" yaml:"key"`
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
	Region    string    `json:"region" yaml:"region"`
	Code      int       `json:"code" yaml:"code"`
	Route     string    `json:"route" yaml:"route"`
	F         []string  `json:"f" yaml:"f"`
}

// CommunicationProvider is an interface for passing messages to other instances of the cache across machine and cluster boundaries
type CommunicationProvider interface {
	// Subscribe returns a channel that will receive invalidation messages for the given cache name
	Subscribe(ctx context.Context, f func(ctx context.Context, msg invalidateMessage)) error
	// Publish returns any secondary keys which also contain the item for lookup by another dimension (ie TypeName, Alias, etc)
	Publish(ctx context.Context, channelNames []string, onMachine bool, message string, localMessage string) error
	// Shutdown closes the communication provider
	Shutdown(ctx context.Context)
}

// RedisCacheCommunicationProvider implements CacheCommunicationProvider using redis pub/sub mechanism
type RedisCacheCommunicationProvider struct {
	localRC        *redis.Client
	remoteRCs      []*redis.Client
	subscriber     func(ctx context.Context, msg invalidateMessage)
	name           string
	done           chan bool
	subscriberLock sync.RWMutex
}

// NewRedisCacheCommunicationProvider returns a new RedisCacheCommunicationProvider
func NewRedisCacheCommunicationProvider(ctx context.Context, cc *Config, onMachine bool, localOnly bool, name string) (CommunicationProvider, error) {
	if cc == nil || cc.RedisCacheConfig == nil {
		return nil, nil
	}

	localRedisCfg := cc.GetLocalRedis()
	if localRedisCfg == nil {
		uclog.Errorf(ctx, "Can't determine local redis config for region")
		return nil, nil
	}
	lc, err := GetRedisClient(ctx, localRedisCfg)
	if err != nil {
		uclog.Errorf(ctx, "failed to create local redis client[%v]: %v", localRedisCfg, err)
		// TODO we should add retries on connection failure
		// Continue here for now but we may end up in an inconsistent state if other services/machines/regions did connect to the cache
		return nil, ucerr.Wrap(err)
	}

	rcs := make([]*redis.Client, 0)

	// This CacheCommunicationProvider will publish to the remote redis clusters if !localOnly
	if !localOnly {
		remoteRedisCfg := cc.GetRemoteRedisClusters()
		uclog.Verbosef(ctx, "Creating redis cache communication provider for %s with local %d remote redis clusters", name, len(remoteRedisCfg))
		rcs = make([]*redis.Client, 0, len(remoteRedisCfg))
		for _, cfg := range remoteRedisCfg {
			rc, err := GetRedisClient(ctx, &cfg)
			if err != nil {
				uclog.Errorf(ctx, "failed to create remote redis client[%v]: %v", cfg, err)
				// TODO we should add retries on connection failure
				continue
			}
			rcs = append(rcs, rc)
		}
	}

	c := &RedisCacheCommunicationProvider{
		name:           name,
		localRC:        lc,
		remoteRCs:      rcs,
		done:           make(chan bool),
		subscriberLock: sync.RWMutex{},
	}

	if !onMachine {
		return c, nil
	}
	// We can't use the context passed in because it will be cancelled when the request is done
	ctx = context.Background()

	sub := lc.Subscribe(ctx, name)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		ch := sub.Channel()
		wg.Done()
		for {
			select {
			case <-c.done:
				return
			case msg := <-ch:
				c.handleInvalidationMessage(request.SetRequestIDIfNotSet(ctx, uuid.Must(uuid.NewV4())), msg)
			}
		}
	}()
	wg.Wait()
	return c, nil
}

func (r *RedisCacheCommunicationProvider) handleInvalidationMessage(ctx context.Context, msg *redis.Message) {
	var m invalidateMessage
	if err := json.Unmarshal([]byte(msg.Payload), &m); err != nil {
		uclog.Errorf(ctx, "Received unexpected invalidation message %v with %v", msg, err)
		return
	}
	r.subscriberLock.RLock()
	if r.subscriber != nil && len(m.Keys) > 0 {
		r.subscriber(ctx, m)
	}
	r.subscriberLock.RUnlock()
}

// Subscribe sets the handler to be called when an invalidation message is received
func (r *RedisCacheCommunicationProvider) Subscribe(ctx context.Context, f func(ctx context.Context, msg invalidateMessage)) error {
	r.subscriberLock.Lock()
	defer r.subscriberLock.Unlock()
	r.subscriber = f
	return nil
}

// Publish publishes the invalidation message to the local and remote caches
func (r *RedisCacheCommunicationProvider) Publish(ctx context.Context, channelNames []string, onMachine bool, message string, localMessage string) error {
	// Publish update to the local region
	var err error
	if onMachine {
		for _, channelName := range channelNames {
			for i := range maxLocalRetries {
				if err = r.localRC.Publish(ctx, channelName, string(localMessage)).Err(); err == nil {
					break
				}
				uclog.Errorf(ctx, "Failed to invalidate key %s to local cluster with %v. Try %d", message, err, i)
			}
		}
	}

	// Publish update to the remote regions
	if len(r.remoteRCs) != 0 {
		// Using go's threadpool, if this proves to be a performance bottleneck we will a single dedicate worker thread
		// but for now it seems like better exchange of complexity
		go func(ctx context.Context, remoteRCs []*redis.Client, msg string) {
			for _, rc := range remoteRCs {
				for _, channelName := range channelNames {
					for i := range maxLocalRetries {
						var err error
						if err = rc.Publish(ctx, channelName, string(message)).Err(); err == nil {
							break
						}
						uclog.Errorf(ctx, "Failed to invalidate key %s to remote cluster with %v for %v. Try %d", message, err, rc, i)
					}
				}
			}
		}(context.Background() /* can't use passed in context as it maybe cancelled */, r.remoteRCs, string(message))
	}
	return nil
}

// Shutdown stops the subscriber listening for messages
func (r *RedisCacheCommunicationProvider) Shutdown(ctx context.Context) {
	select {
	case r.done <- true:
		uclog.Verbosef(ctx, "Shut down redis cache communication provider [%v]", r.name)
		return
	default:
		uclog.Verbosef(ctx, "Shut down of redis cache communication provider [%v] was a nop", r.name)
	}
}
