package companyconfig

import (
	"fmt"
	"net/url"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
)

const (
	cachePrefix                       = "companyconfig"
	tenantURLPrefix                   = "TENANTURL"         // Primary key for tenant url
	tenantURLCollectionKeyString      = "TENANTURLCOL"      // Global collection for tenant url
	sessionsPrefix                    = "SESSION"           // Primary key for session
	sessionsCollectionKeyString       = "SESSIONCOL"        // Global collection for session
	companyPrefix                     = "COMPANY"           // Primary key for company
	companyCollectionKeyString        = "COMPANYCOL"        // Collection of tenant per company
	companyTenantCollectionKeyString  = "TENANTS"           // Global collection for company
	tenantPrefix                      = "TENANT"            // Primary key for tenant
	tenantHostPrefix                  = "TENANTHOST"        // Secondary key for tenant
	tenantCollectionKeyString         = "TENANTCOL"         // Global collection for tenants
	tenantsURLsCollectionKeyString    = "URLSCOL"           // Collection for per tenant urls
	tenantInternalPrefix              = "TENANTINTERNAL"    // Primary key for tenant internal
	tenantInternalCollectionKeyString = "TENANTINTERNALCOL" // Global collection for tenant internal
	sqlshimProxyKeyString             = "SQLSHIMPROXY"      // Primary key for sql shim proxy
	sqlshimProxyCollectionKeyString   = "SQLSHIMPROXYCOL"   // Global collection for sql shim proxy
	dependencyPrefix                  = "DEP"               // Share dependency key prefix among all items
	isModifiedPrefix                  = "MOD"               // Shared is modified key prefix among all items
)

// TODO - I want to remove cacheNameProvider and cacheTTLProvider, I think hard coding constants in the methods will work better

// companyConfigCacheNameProvider is the base implementation of the CacheNameProvider interface
type companyConfigCacheNameProvider struct {
	cache.NoRateLimitKeyNameProvider
	basePrefix string // Base prefix for all keys TenantID_OrgID
}

// NewCompanyConfigCacheNameProvider creates a new BasesCacheNameProvider
func NewCompanyConfigCacheNameProvider() cache.KeyNameProvider {
	return &companyConfigCacheNameProvider{basePrefix: cachePrefix}
}

const (
	// TenantURLKeyID - primary key for tenantURL
	TenantURLKeyID = "TenantURLKeyID"
	// SessionKeyID - primary key for session
	SessionKeyID = "SessionKeyID"
	// CompanyKeyID - primary key for company
	CompanyKeyID = "CompanyKeyID"
	// TenantKeyID - primary key for tenant
	TenantKeyID = "TenantKeyID"
	// TenantHostnameKeyID - secondary key for tenant (by hostname)
	TenantHostnameKeyID = "TenantHostnameKeyID"
	// TenantInternalKeyID - primary key for tenantinternal
	TenantInternalKeyID = "TenantInternalKeyID"
	// TenantURLCollectionKeyID - global collection for tenantURL
	TenantURLCollectionKeyID = "TenantURLCollectionKeyID"
	// SessionCollectionKeyID - global collection for session
	SessionCollectionKeyID = "SessionCollectionKeyID"
	// CompanyCollectionKeyID - global collection for company
	CompanyCollectionKeyID = "CompanyCollectionKeyID"
	// TenantCollectionKeyID - global collection for tenant
	TenantCollectionKeyID = "TenantCollectionKeyID"
	// TenantInternalCollectionKeyID - global collection for tenantinternal
	TenantInternalCollectionKeyID = "TenantInternalCollectionKeyID"
	// CompanyTenantsKeyID - key for collection of tenants for a company
	CompanyTenantsKeyID = "CompanyTenantsKeyID"
	// TenantsURLsKeyID - key for collection of urls for a tenant
	TenantsURLsKeyID = "TenantsURLsKeyID"
	// SQLShimProxyKeyID - primary key for sql shim proxy
	SQLShimProxyKeyID = "SQLShimProxyKeyID"
	// SQLShimProxyCollectionKeyID - global collection for company
	SQLShimProxyCollectionKeyID = "SQLShimProxyCollectionKeyID"

	dependencyKeyID = "DependencyKeyID"
	// IsModifiedKeyID is the key value indicating change in last TTL
	IsModifiedKeyID = "IsModifiedKeyID"
)

// GetPrefix returns the base prefix for all keys
func (c *companyConfigCacheNameProvider) GetPrefix() string {
	return c.basePrefix
}

// GetAllKeyIDs returns all the key ids
func (c *companyConfigCacheNameProvider) GetAllKeyIDs() []string {
	return []string{
		TenantURLKeyID,
		SessionKeyID,
		CompanyKeyID,
		TenantKeyID,
		TenantHostnameKeyID,
		TenantInternalKeyID,
		TenantURLCollectionKeyID,
		SessionCollectionKeyID,
		CompanyCollectionKeyID,
		TenantCollectionKeyID,
		TenantInternalCollectionKeyID,
		CompanyTenantsKeyID,
		TenantsURLsKeyID,
		SQLShimProxyKeyID,
		SQLShimProxyCollectionKeyID,
		dependencyKeyID,
		IsModifiedKeyID,
	}
}

// GetKeyNameForWithID is a shortcut for GetKeyName with a single uuid ID component
func (c *companyConfigCacheNameProvider) GetKeyNameStatic(id cache.KeyNameID) cache.Key {
	return c.GetKeyName(id, []string{})
}

// GetKeyNameForWithID is a shortcut for GetKeyName with a single uuid ID component
func (c *companyConfigCacheNameProvider) GetKeyNameWithID(id cache.KeyNameID, itemID uuid.UUID) cache.Key {
	return c.GetKeyName(id, []string{itemID.String()})
}

// GetKeyNameForWithID is a shortcut for GetKeyName with a single uuid ID component
func (c *companyConfigCacheNameProvider) GetKeyNameWithString(id cache.KeyNameID, itemName string) cache.Key {
	return c.GetKeyName(id, []string{itemName})
}

// GetKeyName gets the key name for the given key name ID and components
func (c *companyConfigCacheNameProvider) GetKeyName(id cache.KeyNameID, components []string) cache.Key {
	switch id {
	case TenantURLKeyID:
		return c.tenantURLKey(components[0])
	case SessionKeyID:
		return c.sessionKey(components[0])
	case CompanyKeyID:
		return c.companyKey(components[0])
	case TenantKeyID:
		return c.tenantKey(components[0])
	case TenantHostnameKeyID:
		return c.tenantHostnameKey(components[0])
	case TenantInternalKeyID:
		return c.tenantInternalKey(components[0])
	case TenantURLCollectionKeyID:
		return c.tenantURLCollectionKey()
	case SessionCollectionKeyID:
		return c.sesssionCollectionKey()
	case CompanyCollectionKeyID:
		return c.companyCollectionKey()
	case TenantCollectionKeyID:
		return c.tenantCollectionKey()
	case TenantInternalCollectionKeyID:
		return c.tenantInternalCollectionKey()
	case CompanyTenantsKeyID:
		return c.companyTenantCollectionKey(components[0])
	case TenantsURLsKeyID:
		return c.tenantsURLsCollectionKey(components[0])
	case SQLShimProxyKeyID:
		return c.sqlshimProxyKey(components[0])
	case SQLShimProxyCollectionKeyID:
		return c.sqlshimProxyCollectionKey()

	case dependencyKeyID:
		return c.dependencyKey(components[0])
	case IsModifiedKeyID:
		return c.isModifiedKey(components[0])

	}
	return ""
}

// tenantURLKey primary key for tenant url
func (c *companyConfigCacheNameProvider) tenantURLKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, tenantURLPrefix, id))
}

// sessionKey primary key for session
func (c *companyConfigCacheNameProvider) sessionKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, sessionsPrefix, id))
}

// companyKey primary key for company
func (c *companyConfigCacheNameProvider) companyKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, companyPrefix, id))
}

// companyTenantCollectionKey is per company tenant collection key
func (c *companyConfigCacheNameProvider) companyTenantCollectionKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v_%v", c.basePrefix, companyPrefix, id, companyTenantCollectionKeyString))
}

// companyTenantCollectionKey is per company tenant collection key
func (c *companyConfigCacheNameProvider) tenantsURLsCollectionKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v_%v", c.basePrefix, tenantPrefix, id, tenantsURLsCollectionKeyString))
}

// tenantKey primary key for tenant
func (c *companyConfigCacheNameProvider) tenantKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, tenantPrefix, id))
}

// tenantHostName primary key for tenant
func (c *companyConfigCacheNameProvider) tenantHostnameKey(host string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, tenantHostPrefix, host))
}

// tenantInternalKey primary key for tenant internal
func (c *companyConfigCacheNameProvider) tenantInternalKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, tenantInternalPrefix, id))
}

// dependencyKey returns key name for dependency keys
func (c *companyConfigCacheNameProvider) dependencyKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, dependencyPrefix, id))
}

// isModifiedKey returns key name for isModified key
func (c *companyConfigCacheNameProvider) isModifiedKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, isModifiedPrefix, id))
}

// tenantURLCollectionKey returns key name for tenant url collection
func (c *companyConfigCacheNameProvider) tenantURLCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", c.basePrefix, tenantURLCollectionKeyString))
}

// sesssionCollectionKey returns key name for session collection
func (c *companyConfigCacheNameProvider) sesssionCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", c.basePrefix, sessionsCollectionKeyString))
}

// companyCollectionKey returns key name for company collection
func (c *companyConfigCacheNameProvider) companyCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", c.basePrefix, companyCollectionKeyString))
}

// tenantCollectionKey returns key name for tenant collection
func (c *companyConfigCacheNameProvider) tenantCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", c.basePrefix, tenantCollectionKeyString))
}

// tenantInternalCollectionKey returns key name for tenant internal collection
func (c *companyConfigCacheNameProvider) tenantInternalCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", c.basePrefix, tenantInternalCollectionKeyString))
}

func (c *companyConfigCacheNameProvider) sqlshimProxyKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, sqlshimProxyKeyString, id))
}

func (c *companyConfigCacheNameProvider) sqlshimProxyCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", c.basePrefix, sqlshimProxyCollectionKeyString))
}

// GetPrimaryKey returns the primary cache key name for tenant url
func (ot TenantURL) GetPrimaryKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(TenantURLKeyID, ot.ID)
}

// GetGlobalCollectionKey returns the global collection key name for tenant url
func (ot TenantURL) GetGlobalCollectionKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameStatic(TenantURLCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for tenant url
func (TenantURL) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for tenant url
}

// GetSecondaryKeys returns the secondary cache key names for tenant url
func (ot TenantURL) GetSecondaryKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{} // unused
}

// GetPerItemCollectionKey returns the per item collection key name for tenant url
func (ot TenantURL) GetPerItemCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per tenant url
}

// GetDependenciesKey returns the dependencies key name for tenant url
func (ot TenantURL) GetDependenciesKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(dependencyKeyID, ot.ID)
}

// GetIsModifiedKey returns the isModifiedKey key name for tenant url
func (ot TenantURL) GetIsModifiedKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(IsModifiedKeyID, ot.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for tenant url
func (TenantURL) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for tenant url dependencies
func (ot TenantURL) GetDependencyKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{} // unused since we depend on the code to delete tenant urls first before tenant
}

// TTL returns the TTL for tenant url
func (ot TenantURL) TTL(c cache.TTLProvider) time.Duration {
	return c.TTL(TenantURLTTL)
}

// GetPrimaryKey returns the primary cache key name for session
func (et Session) GetPrimaryKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(SessionKeyID, et.ID)
}

// GetGlobalCollectionKey returns the global collection key name for session
func (et Session) GetGlobalCollectionKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameStatic(SessionCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for session
func (Session) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for session
}

// GetPerItemCollectionKey returns the per item collection key name for session
func (et Session) GetPerItemCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per session
}

// GetSecondaryKeys returns the secondary cache key names for session
func (et Session) GetSecondaryKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{} //unused
}

// GetDependenciesKey returns the dependencies key name for session
func (et Session) GetDependenciesKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since the depended object are always deleted first
}

// GetIsModifiedKey returns the isModifiedKey key name for session
func (et Session) GetIsModifiedKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(IsModifiedKeyID, et.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for session
func (Session) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for session dependencies
func (et Session) GetDependencyKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// TTL returns the TTL for session
func (et Session) TTL(c cache.TTLProvider) time.Duration {
	return c.TTL(SessionTTL)
}

// GetPrimaryKey returns the primary cache key name for company
func (cmp Company) GetPrimaryKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(CompanyKeyID, cmp.ID)
}

// GetGlobalCollectionKey returns the global collection key name for company
func (cmp Company) GetGlobalCollectionKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameStatic(CompanyCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for company
func (cmp Company) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for company
}

// GetPerItemCollectionKey returns the per item collection key name for company
func (cmp Company) GetPerItemCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per company
}

// GetSecondaryKeys returns the secondary cache key names for company
func (cmp Company) GetSecondaryKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{} // Unused since there nothing stored per company
}

// GetDependenciesKey returns the dependencies key name for company
func (cmp Company) GetDependenciesKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(dependencyKeyID, cmp.ID)
}

// GetIsModifiedKey returns the isModifiedKey key name for company
func (cmp Company) GetIsModifiedKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(IsModifiedKeyID, cmp.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for company
func (Company) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for company dependencies
func (cmp Company) GetDependencyKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// TTL returns the TTL for company
func (cmp Company) TTL(c cache.TTLProvider) time.Duration {
	return c.TTL(CompanyTTL)
}

// GetPrimaryKey returns the primary cache key name for tenant
func (t Tenant) GetPrimaryKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(TenantKeyID, t.ID)
}

// GetGlobalCollectionKey returns the global collection key name for tenant
func (t Tenant) GetGlobalCollectionKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameStatic(TenantCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for tenant
func (Tenant) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for tenant
}

// GetPerItemCollectionKey returns the per item collection key name for tenant
func (t Tenant) GetPerItemCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per tenant
}

// GetSecondaryKeys returns the secondary cache key names for tenant
func (t Tenant) GetSecondaryKeys(c cache.KeyNameProvider) []cache.Key {
	hostname := ""

	if t.TenantURL != "" {
		parsedURL, err := url.Parse(t.TenantURL)
		if err == nil {
			hostname = parsedURL.Host
		}
		if hostname != "" {
			return []cache.Key{c.GetKeyNameWithString(TenantHostnameKeyID, hostname)}
		}
	}

	return []cache.Key{}
}

// GetDependenciesKey returns the dependencies key name for tenant
func (t Tenant) GetDependenciesKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(dependencyKeyID, t.ID)
}

// GetIsModifiedKey returns the isModifiedKey key name for tenant
func (t Tenant) GetIsModifiedKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(IsModifiedKeyID, t.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for tenant
func (Tenant) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for tenant dependencies
func (t Tenant) GetDependencyKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{} // We depend on the code to delete tenants/tenantinternal first before company so we don't use this
}

// TTL returns the TTL for tenant
func (t Tenant) TTL(c cache.TTLProvider) time.Duration {
	return c.TTL(TenantTTL)
}

// GetPrimaryKey returns the primary cache key name for tenant internal
func (ti TenantInternal) GetPrimaryKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(TenantInternalKeyID, ti.ID)
}

// GetGlobalCollectionKey returns the global collection key name for tenant internal
func (ti TenantInternal) GetGlobalCollectionKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameStatic(TenantInternalCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for tenant internal
func (TenantInternal) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for tenant internal
}

// GetPerItemCollectionKey returns the per item collection key name for tenant internal
func (ti TenantInternal) GetPerItemCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per tenant internal
}

// GetSecondaryKeys returns the secondary cache key names for tenant internal
func (ti TenantInternal) GetSecondaryKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{} // Unused since there nothing stored per tenant internal
}

// GetDependenciesKey returns the dependencies key name for tenant internal
func (ti TenantInternal) GetDependenciesKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since the depended object are always deleted first
}

// GetIsModifiedKey returns the isModifiedKey key name for tenant internal
func (ti TenantInternal) GetIsModifiedKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(IsModifiedKeyID, ti.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for tenant internal
func (TenantInternal) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for tenant internal dependencies
func (ti TenantInternal) GetDependencyKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// TTL returns the TTL for tenant internal
func (ti TenantInternal) TTL(c cache.TTLProvider) time.Duration {
	return c.TTL(TenantInternalTTL)
}

// GetPrimaryKey returns the primary cache key name for SQLShimProxy
func (s SQLShimProxy) GetPrimaryKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(SQLShimProxyKeyID, s.ID)
}

// GetGlobalCollectionKey returns the global collection key name for SQLShimProxy
func (s SQLShimProxy) GetGlobalCollectionKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameStatic(SQLShimProxyCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for SQLShimProxy
func (s SQLShimProxy) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for SQLShimProxy
}

// GetPerItemCollectionKey returns the per item collection key name for SQLShimProxy
func (s SQLShimProxy) GetPerItemCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per SQLShimProxy
}

// GetSecondaryKeys returns the secondary cache key names for SQLShimProxy
func (s SQLShimProxy) GetSecondaryKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{} // Unused since there nothing stored per SQLShimProxy
}

// GetDependenciesKey returns the dependencies key name for SQLShimProxy
func (s SQLShimProxy) GetDependenciesKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(dependencyKeyID, s.ID)
}

// GetIsModifiedKey returns the isModifiedKey key name for SQLShimProxy
func (s SQLShimProxy) GetIsModifiedKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(IsModifiedKeyID, s.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for SQLShimProxy
func (SQLShimProxy) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for SQLShimProxy dependencies
func (s SQLShimProxy) GetDependencyKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// TTL returns the TTL for SQLShimProxy
func (s SQLShimProxy) TTL(c cache.TTLProvider) time.Duration {
	return c.TTL(SQLShimProxyTTL)
}

// companyConfigCacheTTLProvider implements the cache.CacheTTLProvider interface
type companyConfigCacheTTLProvider struct {
	tenantURLTTL      time.Duration
	sessionTTL        time.Duration
	companyTTL        time.Duration
	tenantTTL         time.Duration
	tenantInternalTTL time.Duration
	sqlShimProxyTTL   time.Duration
}

// newCompanyConfigCacheTTLProvider creates a new Configurablecache.CacheTTLProvider
func newCompanyConfigCacheTTLProvider(tenantURLTTL time.Duration, sessionTTL time.Duration, companyTTL time.Duration, tenantTTL time.Duration, tenantInternalTTL time.Duration, sqlShimProxyTTL time.Duration) *companyConfigCacheTTLProvider {
	return &companyConfigCacheTTLProvider{tenantURLTTL: tenantURLTTL, sessionTTL: sessionTTL, companyTTL: companyTTL, tenantTTL: tenantTTL, tenantInternalTTL: tenantInternalTTL, sqlShimProxyTTL: sqlShimProxyTTL}
}

const (
	// TenantURLTTL - TTL for tenantURL
	TenantURLTTL = "TENANT_URL_TTL"
	// SessionTTL - TTL for session
	SessionTTL = "SESSION_TTL"
	// CompanyTTL - TTL for session
	CompanyTTL = "COMPANY_TTL"
	// TenantTTL - TTL for company
	TenantTTL = "TENANT_TTL"
	// TenantInternalTTL - TTL for tenant internal
	TenantInternalTTL = "TENANT_INTERNAL_TTL"
	// SQLShimProxyTTL - TTL for sql shim proxy
	SQLShimProxyTTL = "SQLSHIM_PROXY_TTL"
)

// TTL returns the TTL for given type
func (c *companyConfigCacheTTLProvider) TTL(id cache.KeyTTLID) time.Duration {
	switch id {
	case TenantURLTTL:
		return c.tenantURLTTL
	case SessionTTL:
		return c.sessionTTL
	case CompanyTTL:
		return c.companyTTL
	case TenantTTL:
		return c.tenantTTL
	case TenantInternalTTL:
		return c.tenantInternalTTL
	case SQLShimProxyTTL:
		return c.sqlShimProxyTTL
	}
	return cache.SkipCacheTTL
}
