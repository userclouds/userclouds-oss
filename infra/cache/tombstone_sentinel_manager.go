package cache

import (
	"strings"

	"github.com/gofrs/uuid"
)

// TombstoneCacheSentinelManager is the implementation of sentinel management using tombstone on all updates and a lock on reads. The earliest read wins and caches the value
// if the lock hasn't been overwritten by a tombstone. This sentinel manager doesn't cache results of writes and updates and doesn't cache reads for tombstone TTL after
// create/update operation
type TombstoneCacheSentinelManager struct {
}

// NewTombstoneCacheSentinelManager creates a new BaseCacheSentinelManager
func NewTombstoneCacheSentinelManager() *TombstoneCacheSentinelManager {
	return &TombstoneCacheSentinelManager{}
}

// GenerateSentinel generates a sentinel value for the given sentinel type
func (c *TombstoneCacheSentinelManager) GenerateSentinel(stype SentinelType) Sentinel {

	switch stype {
	case Read:
		id := uuid.Must(uuid.NewV4()).String()
		return Sentinel(readSentinelPrefix() + id)
	case Create, Update, Delete:
		return GenerateTombstoneSentinel()

	}
	return NoLockSentinel
}

// CanAlwaysSetSentinel returns true if the sentinel can be set without reading current value
func (c *TombstoneCacheSentinelManager) CanAlwaysSetSentinel(newVal Sentinel) bool {
	// Only read operation uses a lock so all other operations can set tombstone without reading current value
	return !c.IsReadSentinelPrefix(newVal)
}

// CanSetSentinelGivenCurrVal returns true if new sentinel can be set for the given current sentinel (should be called after CanSetSentinel)
func (c *TombstoneCacheSentinelManager) CanSetSentinelGivenCurrVal(currVal Sentinel, newVal Sentinel) bool {
	// Only read operation and it loses to all other operations including earlier reads
	return false
}

// CanSetValue returns operation to take given existing key value, new value, and sentinel for the operation
func (c *TombstoneCacheSentinelManager) CanSetValue(currVal string, val string, sentinel Sentinel) (set bool, clear bool, conflict bool, refresh bool) {
	// Only read operations take a lock
	if c.IsReadSentinelPrefix(sentinel) && currVal == string(sentinel) {
		// The sentinel is still in the key which means nothing interrupted the operation and value can be safely stored in the cache
		return true, false, false, false
	}
	// TODO need to refresh the tombstone in the key to full TTL
	return false, false, false, true
}

// IsSentinelValue returns true if the value passed in is a sentinel value
func (c *TombstoneCacheSentinelManager) IsSentinelValue(v string) bool {
	return strings.HasPrefix(v, sentinelPrefix) || IsTombstoneSentinel(v)
}

// IsInvalidatingSentinelValue returns true if the sentinel requires invalidating the value across other
func (c *TombstoneCacheSentinelManager) IsInvalidatingSentinelValue(v Sentinel) bool {
	return !c.IsReadSentinelPrefix(v)
}

// IsReadSentinelPrefix returns true if the sentinel value is a read sentinel
func (c *TombstoneCacheSentinelManager) IsReadSentinelPrefix(v Sentinel) bool {
	return strings.HasPrefix(string(v), readSentinelPrefix())
}
