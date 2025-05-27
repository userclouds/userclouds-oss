package tenantplex

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
)

const (
	// we need to use a different prefix for testing to avoid interference from services running on the same machine
	cachePrefix     = "plexstorage"
	cacheTestPrefix = "plexstoragetest" // because we use redis for testing and plex global collection key is not scoped to tenant/orgconfig region
)

const (
	tenantPlexPrefix              = "TENANTPLEX"    // Primary key for tenant url
	tenantPlexCollectionKeyString = "TENANTPLEXCOL" // Global collection for tenant url
	isModifiedPrefix              = "MOD"           // Shared is modified key prefix among all items
	dependencyPrefix              = "DEP"           // Share dependency key prefix among all items
)

// TODO - I want to remove cacheNameProvider and cacheTTLProvider, I think hard coding constants in the methods will work better

// PlexStorageCacheNameProvider is the base implementation of the CacheNameProvider interface
type PlexStorageCacheNameProvider struct {
	cache.NoRateLimitKeyNameProvider
	basePrefix string // Base prefix for all keys TenantID
}

// NewPlexStorageCacheNameProvider creates a new BasesCacheNameProvider
func NewPlexStorageCacheNameProvider(useTestPrefix bool) *PlexStorageCacheNameProvider {
	cachePrefixVal := cachePrefix
	if useTestPrefix {
		cachePrefixVal = cacheTestPrefix
	}
	return &PlexStorageCacheNameProvider{basePrefix: cachePrefixVal}
}

const (
	// TenantPlexKeyID - primary key for tenantPlex
	TenantPlexKeyID = "TenantPlexKeyID"

	dependencyKeyID = "DependencyKeyID"
	// IsModifiedKeyID is the key value indicating change in last TTL
	IsModifiedKeyID = "IsModifiedKeyID"
)

// GetPrefix returns the base prefix for all keys
func (c *PlexStorageCacheNameProvider) GetPrefix() string {
	return c.basePrefix
}

// GetAllKeyIDs returns all the key IDs
func (c *PlexStorageCacheNameProvider) GetAllKeyIDs() []string {
	return []string{TenantPlexKeyID, dependencyKeyID, IsModifiedKeyID}
}

// GetKeyNameStatic is a shortcut for GetKeyName without components
func (c *PlexStorageCacheNameProvider) GetKeyNameStatic(id cache.KeyNameID) cache.Key {
	return c.GetKeyName(id, []string{})
}

// GetKeyNameWithID is a shortcut for GetKeyName with a single uuid ID component
func (c *PlexStorageCacheNameProvider) GetKeyNameWithID(id cache.KeyNameID, itemID uuid.UUID) cache.Key {
	return c.GetKeyName(id, []string{itemID.String()})
}

// GetKeyNameWithString is a shortcut for GetKeyName with a single string component
func (c *PlexStorageCacheNameProvider) GetKeyNameWithString(id cache.KeyNameID, itemName string) cache.Key {
	return c.GetKeyName(id, []string{itemName})
}

// GetKeyName gets the key name for the given key name ID and components
func (c *PlexStorageCacheNameProvider) GetKeyName(id cache.KeyNameID, components []string) cache.Key {
	switch id {
	case TenantPlexKeyID:
		return c.tenantPlexKey(components[0])

	case dependencyKeyID:
		return c.dependencyKey(components[0])
	case IsModifiedKeyID:
		return c.isModifiedKey(components[0])

	}
	return ""
}

// tenantPlexKey primary key for tenant plex
func (c *PlexStorageCacheNameProvider) tenantPlexKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, tenantPlexPrefix, id))
}

// dependencyKey returns key name for dependency keys
func (c *PlexStorageCacheNameProvider) dependencyKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, dependencyPrefix, id))
}

// isModifiedKey returns key name for isModified key
func (c *PlexStorageCacheNameProvider) isModifiedKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, isModifiedPrefix, id))
}

// GetPrimaryKey returns the primary cache key name for tenant plex
func (ot TenantPlex) GetPrimaryKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(TenantPlexKeyID, ot.ID)
}

// GetGlobalCollectionKey returns the global collection key name for tenant plex
func (ot TenantPlex) GetGlobalCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // unused because TenantPlex is a table with a single entry
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for tenant plex
func (TenantPlex) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for tenant plex
}

// GetSecondaryKeys returns the secondary cache key names for tenant plex
func (ot TenantPlex) GetSecondaryKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// GetPerItemCollectionKey returns the per item collection key name for tenant plex
func (ot TenantPlex) GetPerItemCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per tenant url
}

// GetDependenciesKey returns the dependencies key name for tenant plex
func (ot TenantPlex) GetDependenciesKey(c cache.KeyNameProvider) cache.Key {
	return "" // unused
}

// GetIsModifiedKey returns the isModifiedKey key name for tenant plex
func (ot TenantPlex) GetIsModifiedKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(IsModifiedKeyID, ot.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for tenant plex
func (TenantPlex) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for tenant url dependencies
func (ot TenantPlex) GetDependencyKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{} // unused since we depend on the code to delete tenant urls first before tenant
}

// TTL returns the TTL for tenant plex
func (ot TenantPlex) TTL(c cache.TTLProvider) time.Duration {
	return c.TTL(TenantPlexTTL)
}

// PlexStorageCacheTTLProvider implements the cache.CacheTTLProvider interface
type PlexStorageCacheTTLProvider struct {
	tenantPlexTTL time.Duration
}

// NewPlexStorageCacheTTLProvider creates a new cache.CacheTTLProvider
func NewPlexStorageCacheTTLProvider(tenantPlexTTL time.Duration) *PlexStorageCacheTTLProvider {
	return &PlexStorageCacheTTLProvider{tenantPlexTTL: tenantPlexTTL}
}

const (
	// TenantPlexTTL - TTL for tenantURL
	TenantPlexTTL = "TENANT_URL_TTL"
)

// TTL returns the TTL for given type
func (c *PlexStorageCacheTTLProvider) TTL(id cache.KeyTTLID) time.Duration {
	switch id {
	case TenantPlexTTL:
		return c.tenantPlexTTL
	}
	return cache.SkipCacheTTL
}
