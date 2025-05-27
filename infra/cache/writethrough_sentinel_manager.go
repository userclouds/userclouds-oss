package cache

import (
	"strings"

	"github.com/gofrs/uuid"
)

const sentinelPrefix = "sentinel_"
const sentinelWritePrefix = "write_"
const sentinelDeletePrefix = "delete_"
const sentinelReadPrefix = "read_"

// WriteThroughCacheSentinelManager is the implementation of sentinel management for a write through cache
type WriteThroughCacheSentinelManager struct {
}

// NewWriteThroughCacheSentinelManager creates a new BaseCacheSentinelManager
func NewWriteThroughCacheSentinelManager() *WriteThroughCacheSentinelManager {
	return &WriteThroughCacheSentinelManager{}
}

// GenerateSentinel generates a sentinel value for the given sentinel type
func (c *WriteThroughCacheSentinelManager) GenerateSentinel(stype SentinelType) Sentinel {
	id := uuid.Must(uuid.NewV4()).String()
	switch stype {
	case Read:
		return Sentinel(readSentinelPrefix() + id)
	case Create, Update:
		return Sentinel(writeSentinelPrefix() + id)
	case Delete:
		return Sentinel(deleteSentinelPrefix() + id)

	}
	return NoLockSentinel
}

// CanAlwaysSetSentinel returns true if the sentinel can be set without reading current value
func (c *WriteThroughCacheSentinelManager) CanAlwaysSetSentinel(newVal Sentinel) bool {
	return false // We could return true for delete sentinel if we didn't need to check for tombstones. TODO we could optimize this
}

// CanSetSentinelGivenCurrVal returns true if new sentinel can be set for the given current sentinel
func (c *WriteThroughCacheSentinelManager) CanSetSentinelGivenCurrVal(currVal Sentinel, newVal Sentinel) bool {
	// If we are doing a read - read sentinel loses to all other sentinels including other in progress reads
	if c.IsReadSentinelPrefix(newVal) {
		return false
	}

	// If there is delete in progress, writes can't take a lock delete don't need to
	if c.IsDeleteSentinelPrefix(currVal) {
		return false
	}
	// If there is a write in progress, take the lock from it and depend on clean up on value conflict in SetValue
	// This means that if we finish before an earlier write(s), we will write value into the cache, the writes finishing after us
	// will check the value and clear it if it doesn't match what they got from server. If the earlier write(s) finish before us, they will
	// bump our lock to conflict so we will not write the value into the cache but will clear the lock

	return true
}

// CanSetValue returns operation to take given existing key value, new value, and sentinel for the operation
func (c *WriteThroughCacheSentinelManager) CanSetValue(currVal string, val string, sentinel Sentinel) (set bool, clear bool, conflict bool, refresh bool) {
	if currVal == string(sentinel) {
		// The sentinel is still in the key which means nothing interrupted the operation and value can be safely stored in the cache
		return true, false, false, false
	} else if c.IsWriteSentinelPrefix(sentinel) {
		// We are doing a write of an item and we are interleaved with other write(s)
		if !c.IsSentinelValue(currVal) && val != currVal {
			// If there is a tombstone in the cache, we overlapped with either a delete or an invalidation
			if !IsTombstoneSentinel(currVal) {
				// There is a value in the cache and it doesn't match what we got from the server, clear the cache because we had interleaving writes
				// finish before us with a different value. We can't tell what the server side order of completion was
				return false, true, false, false
			}
		} else if strings.HasPrefix(currVal, string(sentinel)) {
			// Another write that was interleaved with this one and finished first, setting the sentinel to indicate conflict. We can't tell what the server
			// side order of completion was so clear the cache
			return false, true, false, false
		} else if c.IsWriteSentinelPrefix(Sentinel(currVal)) {
			// There is another write in progress that started after us. There is no way to tell if that write will commit same value to the cache
			// so upgrade its lock to conflict so it doesn't commit its result
			return false, false, true, false
		}
	}
	return false, false, false, false
}

// IsSentinelValue returns true if the value passed in is a sentinel value
func (c *WriteThroughCacheSentinelManager) IsSentinelValue(v string) bool {
	return strings.HasPrefix(v, sentinelPrefix) || IsTombstoneSentinel(v)
}

// IsInvalidatingSentinelValue returns true if the sentinel requires invalidating the value across other
func (c *WriteThroughCacheSentinelManager) IsInvalidatingSentinelValue(v Sentinel) bool {
	return c.IsWriteSentinelPrefix(v) || c.IsDeleteSentinelPrefix(v)
}

// IsReadSentinelPrefix returns true if the sentinel value is a read sentinel
func (c *WriteThroughCacheSentinelManager) IsReadSentinelPrefix(v Sentinel) bool {
	return strings.HasPrefix(string(v), readSentinelPrefix())
}

// IsWriteSentinelPrefix returns true if the sentinel value is a write sentinel
func (c *WriteThroughCacheSentinelManager) IsWriteSentinelPrefix(v Sentinel) bool {
	return strings.HasPrefix(string(v), writeSentinelPrefix())
}

// IsDeleteSentinelPrefix returns true if the sentinel value is a delete sentinel
func (c *WriteThroughCacheSentinelManager) IsDeleteSentinelPrefix(v Sentinel) bool {
	return strings.HasPrefix(string(v), deleteSentinelPrefix())
}

func deleteSentinelPrefix() string {
	return sentinelPrefix + sentinelDeletePrefix
}

func writeSentinelPrefix() string {
	return sentinelPrefix + sentinelWritePrefix
}

func readSentinelPrefix() string {
	return sentinelPrefix + sentinelReadPrefix
}

// IsInvalidatingSentinelValue returns true if the sentinel requires invalidating the value across other
func IsInvalidatingSentinelValue(v Sentinel) bool {
	return strings.HasPrefix(string(v), writeSentinelPrefix()) || strings.HasPrefix(string(v), deleteSentinelPrefix())
}
