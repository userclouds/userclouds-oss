package cache

import (
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

// SentinelType names
const (
	Create SentinelType = "create"
	Update SentinelType = "update"
	Delete SentinelType = "delete"
	Read   SentinelType = "read"
)

const (

	// SentinelTTL value when setting a sentinel value in the cache
	SentinelTTL = 60 * time.Second
	// InvalidationTombstoneTTL value when setting a tombstone value in the cache for cross region invalidation
	InvalidationTombstoneTTL = 2 * time.Second

	// tombstoneSentinelPrefix represents the prefix for sentinel value for a tombstone
	tombstoneSentinelPrefix Sentinel = "Tombstone"

	// NoLockSentinel represents the sentinel value for no lock
	NoLockSentinel Sentinel = ""
)

// IsTombstoneSentinel returns true if the given data is a tombstone sentinel
func IsTombstoneSentinel(data string) bool {
	return strings.HasPrefix(data, string(tombstoneSentinelPrefix))
}

// GenerateTombstoneSentinel generates a tombstone sentinel value
func GenerateTombstoneSentinel() Sentinel {
	return Sentinel(string(tombstoneSentinelPrefix) + uuid.Must(uuid.NewV4()).String())
}
