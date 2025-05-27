package cache

import (
	"testing"
	"time"
)

// Methods to use in tests where we want to disable tombstones so we can test some aspects of the cache

// SetTombstoneTTL sets the TTL for tombstone keys in the cache
func (c *RedisClientCacheProvider) SetTombstoneTTL(t *testing.T, ttl time.Duration) {
	t.Helper() // testing.T is passed to this method to ensure it is only used in tests
	c.tombstoneTTL = ttl
}

// SetTombstoneTTL sets the TTL for tombstone keys in the cache
func (c *InMemoryClientCacheProvider) SetTombstoneTTL(t *testing.T, ttl time.Duration) {
	t.Helper() // testing.T is passed to this method to ensure it is only used in tests
	c.tombstoneTTL = ttl
}
