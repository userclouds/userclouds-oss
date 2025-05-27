package tenantcache

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucerr"
)

// Connect connects to redis client cache.
func Connect(ctx context.Context, cfg *cache.Config, tenantID uuid.UUID) (cache.Provider, error) {
	if cfg == nil {
		return nil, nil
	}

	// TODO (sgarrity 10/23): this shouldn't be redis specific
	rc := cfg.GetLocalRedis()
	if rc == nil || rc.Host == "" {
		return nil, nil
	}

	redisClient, err := GetRedisClient(ctx, rc)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	// TODO - need better packaging for setting the prefix to authz_tenantID via cache cache.KeyPrefixRedis()
	// since I don't want to reach into authz package to get the prefix "authz"
	return cache.NewRedisClientCacheProvider(redisClient, cache.RegionalRedisCacheName, cache.ReadOnlyRedis()), nil
}

// GetRedisClient returns a redis client for the given config. TODO - this should be moved to redis provider once the certificate issue is resolved.
func GetRedisClient(ctx context.Context, cfg *cache.RedisConfig) (*redis.Client, error) {
	if cfg == nil {
		return nil, ucerr.New("Redis config is nil")
	}

	rc, err := cache.GetRedisClient(ctx, cfg)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return rc, nil
}
