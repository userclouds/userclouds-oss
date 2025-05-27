package tenantcache

import (
	"context"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/cache"
)

func TestGetRedisClient(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	t.Run("nil config", func(t *testing.T) {
		client, err := GetRedisClient(ctx, nil)
		assert.IsNil(t, client)
		assert.Contains(t, err.Error(), "Redis config is nil")
	})

	t.Run("redis client", func(t *testing.T) {
		client, err := GetRedisClient(ctx, &cache.RedisConfig{
			Host:   "localhost",
			Port:   6379,
			DBName: 0,
		})
		assert.NoErr(t, err)
		assert.NotNil(t, client)
	})
	t.Run("redis client no connection", func(t *testing.T) {
		client, err := GetRedisClient(ctx, &cache.RedisConfig{
			Host:   "localhost",
			Port:   6388,
			DBName: 0,
		})
		assert.IsNil(t, client)
		assert.Contains(t, err.Error(), "6388: connect: connection refused")
	})

}
