package datamapping

import (
	"time"

	"userclouds.com/infra/cache"
)

const (
	// IsModifiedKeyID is the key value indicating change in last TTL
	IsModifiedKeyID = "IsModifiedKeyID"

	// DataSourceKeyID - primary key for Data Source
	DataSourceKeyID = "DataSourceKeyID"
	// DataSourceNameKeyID - secondary key for Data Source
	DataSourceNameKeyID = "DataSourceNameKeyID"
	// DataSourceCollectionKeyID - global collection for Data Source
	DataSourceCollectionKeyID = "DataSourceCollectionKeyID"
	// DataSourceTTL - TTL for Data Source
	DataSourceTTL = "DATA_SOURCE_TTL"
	// DataSourceElementKeyID - primary key for Data Source Element
	DataSourceElementKeyID = "DataSourceElementKeyID"
	// DataSourceElementNameKeyID - secondary key for Data Source Element
	DataSourceElementNameKeyID = "DataSourceElementNameKeyID"
	// DataSourceElementCollectionKeyID - global collection for Data Source Element
	DataSourceElementCollectionKeyID = "DataSourceElementCollectionKeyID"
	// DataSourceElementTTL - TTL for Data Source Element
	DataSourceElementTTL = "DATA_SOURCE_ELEMENT_TTL"
)

// GetPrimaryKey returns the primary cache key name for data source element
func (ds DataSource) GetPrimaryKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(DataSourceKeyID, ds.ID)
}

// GetGlobalCollectionKey returns the global collection key name for data source element
func (ds DataSource) GetGlobalCollectionKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameStatic(DataSourceCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for  data source
func (ds DataSource) GetGlobalCollectionPagesKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for  data source
}

// GetPerItemCollectionKey returns the per item collection key name for data source element
func (ds DataSource) GetPerItemCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per access policy, could store edges of this type in the future
}

// GetSecondaryKeys returns the secondary cache key names for data source element
func (ds DataSource) GetSecondaryKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{c.GetKeyNameWithString(DataSourceNameKeyID, ds.Name)}
}

// GetDependenciesKey returns the dependencies key name for data source element
func (ds DataSource) GetDependenciesKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused as all dependent resources have to be deleted before DataSource can be deleted
}

// GetIsModifiedKey returns the isModifiedKey key name for data source element
func (ds DataSource) GetIsModifiedKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(IsModifiedKeyID, ds.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for data source element
func (ds DataSource) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for data source element dependencies
func (ds DataSource) GetDependencyKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// TTL returns the TTL for access policy
func (ds DataSource) TTL(c cache.TTLProvider) time.Duration {
	return c.TTL(DataSourceTTL)
}

// GetPrimaryKey returns the primary cache key name for data source element
func (dse DataSourceElement) GetPrimaryKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(DataSourceElementKeyID, dse.ID)
}

// GetGlobalCollectionKey returns the global collection key name for data source element
func (dse DataSourceElement) GetGlobalCollectionKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameStatic(DataSourceElementCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for  data source element
func (dse DataSourceElement) GetGlobalCollectionPagesKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for  data source element
}

// GetPerItemCollectionKey returns the per item collection key name for data source element
func (dse DataSourceElement) GetPerItemCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per access policy, could store edges of this type in the future
}

// GetSecondaryKeys returns the secondary cache key names for data source element
func (dse DataSourceElement) GetSecondaryKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{c.GetKeyNameWithString(DataSourceElementNameKeyID, dse.Path)}
}

// GetDependenciesKey returns the dependencies key name for data source element
func (dse DataSourceElement) GetDependenciesKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused as all dependent resources have to be deleted before DataSourceElement can be deleted
}

// GetIsModifiedKey returns the isModifiedKey key name for data source element
func (dse DataSourceElement) GetIsModifiedKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(IsModifiedKeyID, dse.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for data source element
func (dse DataSourceElement) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for data source element dependencies
func (dse DataSourceElement) GetDependencyKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// TTL returns the TTL for access policy
func (dse DataSourceElement) TTL(c cache.TTLProvider) time.Duration {
	return c.TTL(DataSourceElementTTL)
}
