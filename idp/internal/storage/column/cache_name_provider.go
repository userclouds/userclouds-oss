package column

import (
	"time"

	"userclouds.com/infra/cache"
)

const (
	// IsModifiedKeyID is the key value indicating change in last TTL (is redefined here to avoid circular dependency with between columns and storage)
	IsModifiedKeyID = "IsModifiedKeyID"

	// DataTypeKeyID - primary key for DataType
	DataTypeKeyID = "DataTypeKeyID"

	// DataTypeCollectionKeyID - global collection for DataType
	DataTypeCollectionKeyID = "DataTypesCollectionKeyID"

	// DataTypeTTL - TTL for DataType
	DataTypeTTL = "DATA_TYPE_TTL"
)

// GetPrimaryKey returns the primary cache key name for DataType
func (dt DataType) GetPrimaryKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(DataTypeKeyID, dt.ID)
}

// GetGlobalCollectionKey returns the global collection key name for DataType
func (DataType) GetGlobalCollectionKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameStatic(DataTypeCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for DataType
func (DataType) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for DataType
}

// GetSecondaryKeys returns the secondary cache key names for DataType
func (dt DataType) GetSecondaryKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// GetPerItemCollectionKey returns the per item collection key name for DataType
func (DataType) GetPerItemCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per DataType, could store objects of this type in the future
}

// GetDependenciesKey returns the dependencies key name for DataType
func (DataType) GetDependenciesKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since we track dependencies explicitly
}

// GetIsModifiedKey returns the isModifiedKey key name for DataType
func (dt DataType) GetIsModifiedKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(IsModifiedKeyID, dt.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for DataType
func (DataType) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for DataType dependencies
func (DataType) GetDependencyKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{} // ObjectTypes don't depend on anything
}

// TTL returns the TTL for DataType
func (DataType) TTL(c cache.TTLProvider) time.Duration {
	return c.TTL(DataTypeTTL)
}
