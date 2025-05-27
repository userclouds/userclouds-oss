package storage

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/datamapping"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/infra/cache"
)

const (
	dependencyPrefix                                = "DEP"                                      // Share dependency key prefix among all items
	isModifiedPrefix                                = "MOD"                                      // Shared is modified key prefix among all items
	columnPrefix                                    = "COLUMN"                                   // Primary key for column
	columnCollectionKeyString                       = columnPrefix + "COL"                       // Global collection for column
	columnCollectionPagesPrefixString               = "PAGES"                                    // Pages prefix for column global collection
	accessorPrefix                                  = "ACCESSOR"                                 // Primary key for accessor
	accessorCollectionKeyString                     = accessorPrefix + "COL"                     // Global collection for accessor
	accessPolicyPrefix                              = "ACCESSPOLICY"                             // Primary key for access policy
	accessPoliciesCollectionKeyString               = accessPolicyPrefix + "COL"                 // Global collection for access policy
	accessPolicyRateLimitPrefix                     = "ACCESSPOLICYRATELIMIT"                    // Primary key for access policy rate limit
	accessPolicyTemplatePrefix                      = "ACCESSPOLICYTEMPLATE"                     // Primary key for access policy template
	accessPolicyTemplateCollectionKeyString         = accessPolicyTemplatePrefix + "COL"         // Global collection for access policy template
	purposePrefix                                   = "PURPOSE"                                  // Primary key for purpose
	purposeCollectionKeyString                      = purposePrefix + "COL"                      // Global collection for purpose
	mutatorPrefix                                   = "MUTATOR"                                  // primary key for Mutator
	mutatorCollectionKeyString                      = mutatorPrefix + "COL"                      // global collection for Mutator
	transformerPrefix                               = "TRANSFORMER"                              // primary key for transformer
	transformerCollectionKeyString                  = transformerPrefix + "COL"                  // global collection for transformer
	dataSourcePrefix                                = "DATASOURCE"                               // primary key for Data Source
	dataSourceCollectionKeyString                   = dataSourcePrefix + "COL"                   // global collection for Data Source
	dataSourceElementPrefix                         = "DATASOURCEELEMENT"                        // primary key for Data Source Element
	dataSourceElementCollectionKeyString            = dataSourceElementPrefix + "COL"            // global collection for Data Source Element
	dataTypePrefix                                  = "DATATYPE"                                 // primary key for DataType
	dataTypeCollectionKeyString                     = dataTypePrefix + "COL"                     // global collection for DataType
	columnValueRetentionDurationPrefix              = "COLUMNVALUERETENTIONDURATION"             // primary key for column value retention duration
	columnValueRetentionDurationCollectionKeyString = columnValueRetentionDurationPrefix + "COL" // global collection for column value retention duration
	accessorSearchIndexPrefix                       = "ACCESSORSEARCHINDEX"                      // primary key for AccessorSearchIndex
	accessorSearchIndexCollectionKeyString          = accessorSearchIndexPrefix + "COL"          // global collection for AccessorSearchIndex
	userSearchIndexPrefix                           = "USERSEARCHINDEX"                          // primary key for UserSearchIndex
	userSearchIndexCollectionKeyString              = userSearchIndexPrefix + "COL"              // global collection for UserSearchIndex
)

// TODO - I want to remove cacheNameProvider and cacheTTLProvider, I think hard coding constants in the methods will work better

// CacheNameProvider is the base implementation of the CacheNameProvider interface
type CacheNameProvider struct {
	basePrefix string // Base prefix for all keys TenantID_OrgID
}

// NewCacheNameProviderForTenant creates a new IDP CacheNameProvider for the given tenant
func NewCacheNameProviderForTenant(tenantID uuid.UUID) *CacheNameProvider {
	basePrefix := fmt.Sprintf("%s_%v", CachePrefix, tenantID)
	return &CacheNameProvider{basePrefix: basePrefix}
}

const (
	// CachePrefix is the prefix for all keys in userstore cache
	CachePrefix     = "userstore"
	dependencyKeyID = "DependencyKeyID"
	// IsModifiedKeyID is the key value indicating change in last TTL
	IsModifiedKeyID = "IsModifiedKeyID"

	// ColumnKeyID - primary key for Column
	ColumnKeyID = "ColumnKeyID"
	// ColumnNameKeyID - secondary key for Column
	ColumnNameKeyID = "ColumnNameID"
	// ColumnCollectionKeyID - global collection for Column
	ColumnCollectionKeyID = "ColumnsCollectionKeyID"
	// ColumnCollectionPagesKeyID - global collection pages for Column
	ColumnCollectionPagesKeyID = "ColumnsCollectionPagesKeyID"
	// ColumnCollectionPageKeyID is the key for each individual page in the global collection of edges
	ColumnCollectionPageKeyID = "ColumnCollectionPageKeyID"
	// IsModifiedCollectionKeyID - IsModified for global collection
	IsModifiedCollectionKeyID = "IsModifiedCollectionKeyID"

	// AccessorKeyID - primary key for Accessor
	AccessorKeyID = "AccessorKeyID"
	// AccessorNameKeyID - secondary key for Accessor
	AccessorNameKeyID = "AccessorNameID"
	// AccessorCollectionKeyID - global collection for Accessor
	AccessorCollectionKeyID = "AccessorCollectionKeyID"

	// AccessPolicyKeyID - primary key for Access Policy
	AccessPolicyKeyID = "AccessPolicyKeyID"
	// AccessPolicyNameKeyID - secondary key for Access Policy
	AccessPolicyNameKeyID = "AccessPolicyNameID"
	// AccessPolicyCollectionKeyID - global collection for Access Policy
	AccessPolicyCollectionKeyID = "AccessPolicyCollectionKeyID"

	// AccessPolicyRateLimitKeyID - primary key for AccessPolicyRateLimit
	AccessPolicyRateLimitKeyID = "AccessPolicyRateLimitKeyID"

	// AccessPolicyTemplateKeyID - primary key for Access Policy
	AccessPolicyTemplateKeyID = "AccessPolicyTemplateKeyID"
	// AccessPolicyTemplateCollectionKeyID - global collection for Access Policy
	AccessPolicyTemplateCollectionKeyID = "AccessPolicyTemplateCollectionKeyID"

	// PurposeKeyID - primary key for Purpose
	PurposeKeyID = "PurposeKeyID"
	// PurposeNameKeyID - secondary key for Purpose
	PurposeNameKeyID = "PurposeNameID"
	// PurposeCollectionKeyID - global collection for Purpose
	PurposeCollectionKeyID = "PurposeCollectionKeyID"

	// MutatorKeyID - primary key for Mutator
	MutatorKeyID = "MutatorKeyID"
	// MutatorNameKeyID - secondary key for Mutator
	MutatorNameKeyID = "MutatorNameID"
	// MutatorCollectionKeyID - global collection for Mutator
	MutatorCollectionKeyID = "MutatorCollectionKeyID"

	// TransformerKeyID - primary key for Transformer
	TransformerKeyID = "TransformerKeyID"
	// TransformerNameKeyID - secondary key for Transformer
	TransformerNameKeyID = "TransformerNameID"
	// TransformerCollectionKeyID - global collection for Transformer
	TransformerCollectionKeyID = "TransformerCollectionKeyID"

	// DataTypeKeyID - primary key for DataType
	// DataTypeKeyID = "DataTypeKeyID" (defined in column package)
	// DataTypeCollectionKeyID - global collection for DataType
	// DataTypeCollectionKeyID = "DataTypesCollectionKeyID" (defined in column package)

	// ColumnValueRetentionDurationKeyID - primary key for ColumnValueRetentionDuration
	ColumnValueRetentionDurationKeyID = "ColumnValueRetentionDurationKeyID"
	// ColumnValueRetentionDurationCollectionKeyID - global collection for ColumnValueRetentionDuration
	ColumnValueRetentionDurationCollectionKeyID = "ColumnValueRetentionDurationCollectionKeyID"

	// SQLShimDatabaseKeyID - primary key for SQLShimDatabase
	SQLShimDatabaseKeyID = "SQLShimDBKeyID"
	// SQLShimDatabaseNameKeyID - secondary key for SQLShimDatabase
	SQLShimDatabaseNameKeyID = "SQLShimDBNameID"
	// SQLShimDatabaseCollectionKeyID - global collection for SQLShimDatabase
	SQLShimDatabaseCollectionKeyID = "SQLShimDBCollectionKeyID"

	// ShimObjectStoreKeyID - primary key for ShimObjectStore
	ShimObjectStoreKeyID = "ShimObjectStoreKeyID"
	// ShimObjectStoreNameKeyID - secondary key for ShimObjectStore
	ShimObjectStoreNameKeyID = "ShimObjectStoreNameID"
	// ShimObjectStoreCollectionKeyID - global collection for ShimObjectStore
	ShimObjectStoreCollectionKeyID = "ShimObjectStoreCollectionKeyID"

	// AccessorSearchIndexKeyID - primary key for AccessorSearchIndex
	AccessorSearchIndexKeyID = "AccessorSearchIndexKeyID"
	// AccessorSearchIndexCollectionKeyID - global collection for AccessorSearchIndex
	AccessorSearchIndexCollectionKeyID = "AccessorSearchIndexCollectionKeyID"

	// UserSearchIndexKeyID - primary key for UserSearchIndex
	UserSearchIndexKeyID = "UserSearchIndexKeyID"
	// UserSearchIndexCollectionKeyID - global collection for UserSearchIndex
	UserSearchIndexCollectionKeyID = "UserSearchIndexCollectionKeyID"
)

// GetAllKeyIDs returns all the key name IDs
func (*CacheNameProvider) GetAllKeyIDs() []string {
	return []string{
		dependencyKeyID,
		IsModifiedKeyID,
		column.DataTypeKeyID,
		column.DataTypeCollectionKeyID,
		ColumnKeyID,
		ColumnNameKeyID,
		ColumnCollectionKeyID,
		ColumnCollectionPagesKeyID,
		ColumnCollectionPageKeyID,
		IsModifiedCollectionKeyID,
		AccessorKeyID,
		AccessorNameKeyID,
		AccessorCollectionKeyID,
		AccessPolicyKeyID,
		AccessPolicyNameKeyID,
		AccessPolicyCollectionKeyID,
		AccessPolicyRateLimitKeyID,
		AccessPolicyTemplateKeyID,
		AccessPolicyTemplateCollectionKeyID,
		PurposeKeyID,
		PurposeNameKeyID,
		PurposeCollectionKeyID,
		MutatorKeyID,
		MutatorNameKeyID,
		MutatorCollectionKeyID,
		TransformerKeyID,
		TransformerNameKeyID,
		TransformerCollectionKeyID,
		ColumnValueRetentionDurationKeyID,
		ColumnValueRetentionDurationCollectionKeyID,
		SQLShimDatabaseKeyID,
		SQLShimDatabaseCollectionKeyID,
		SQLShimDatabaseNameKeyID,
		ShimObjectStoreKeyID,
		ShimObjectStoreNameKeyID,
		ShimObjectStoreCollectionKeyID,
		AccessorSearchIndexKeyID,
		AccessorSearchIndexCollectionKeyID,
		UserSearchIndexKeyID,
		UserSearchIndexCollectionKeyID,
	}
}

// GetPrefix returns the base prefix for all keys
func (cnp *CacheNameProvider) GetPrefix() string {
	return cnp.basePrefix
}

// GetKeyNameStatic is a shortcut for GetKeyName with without components
func (cnp *CacheNameProvider) GetKeyNameStatic(id cache.KeyNameID) cache.Key {
	return cnp.GetKeyName(id, []string{})
}

// GetKeyNameWithID is a shortcut for GetKeyName with a single uuid ID component
func (cnp *CacheNameProvider) GetKeyNameWithID(id cache.KeyNameID, itemID uuid.UUID) cache.Key {
	return cnp.GetKeyName(id, []string{itemID.String()})
}

// GetKeyNameWithString is a shortcut for GetKeyName with a single string component
func (cnp *CacheNameProvider) GetKeyNameWithString(id cache.KeyNameID, itemName string) cache.Key {
	return cnp.GetKeyName(id, []string{itemName})
}

// GetKeyName gets the key name for the given key name ID and components
func (cnp *CacheNameProvider) GetKeyName(id cache.KeyNameID, components []string) cache.Key {
	switch id {
	case dependencyKeyID:
		return cnp.dependencyKey(components[0])
	case IsModifiedKeyID:
		return cnp.isModifiedKey(components[0])
	case IsModifiedCollectionKeyID:
		return cnp.isModifiedCollectionKey(components[0])

	case column.DataTypeKeyID:
		return cnp.dataTypeKey(components[0])
	case column.DataTypeCollectionKeyID:
		return cnp.dataTypeCollectionKey()

	case ColumnKeyID:
		return cnp.columnKey(components[0])
	case ColumnNameKeyID:
		return cnp.columnNameKey(components[0], components[1], components[2])
	case ColumnCollectionKeyID:
		return cnp.columnCollectionKey()
	case ColumnCollectionPagesKeyID:
		return cnp.columnCollectionPagesKey()
	case ColumnCollectionPageKeyID:
		return cnp.columnCollectionPageKey(components[0], components[1])

	case AccessorKeyID:
		return cnp.accessorKey(components[0])
	case AccessorNameKeyID:
		return cnp.accessorNameKey(components[0])
	case AccessorCollectionKeyID:
		return cnp.accessorCollectionKey()

	case AccessPolicyKeyID:
		return cnp.accessPolicyKey(components[0])
	case AccessPolicyNameKeyID:
		return cnp.accessPolicyNameKey(components[0])
	case AccessPolicyCollectionKeyID:
		return cnp.accessPolicyCollectionKey()

	case AccessPolicyRateLimitKeyID:
		return cache.Key(cnp.accessPolicyRateLimitKey(components[0]))

	case AccessPolicyTemplateKeyID:
		return cnp.accessPolicyTemplateKey(components[0])
	case AccessPolicyTemplateCollectionKeyID:
		return cnp.accessPolicyTemplateCollectionKey()

	case PurposeKeyID:
		return cnp.purposeKey(components[0])
	case PurposeNameKeyID:
		return cnp.purposeNameKey(components[0])
	case PurposeCollectionKeyID:
		return cnp.purposeCollectionKey()

	case MutatorKeyID:
		return cnp.mutatorKey(components[0])
	case MutatorNameKeyID:
		return cnp.mutatorNameKey(components[0])
	case MutatorCollectionKeyID:
		return cnp.mutatorCollectionKey()

	case TransformerKeyID:
		return cnp.transformerKey(components[0])
	case TransformerNameKeyID:
		return cnp.transformerNameKey(components[0])
	case TransformerCollectionKeyID:
		return cnp.transformerCollectionKey()

	case datamapping.DataSourceKeyID:
		return cnp.dataSourceKey(components[0])
	case datamapping.DataSourceCollectionKeyID:
		return cnp.dataSourceCollectionKey()

	case datamapping.DataSourceElementKeyID:
		return cnp.dataSourceElementKey(components[0])
	case datamapping.DataSourceElementCollectionKeyID:
		return cnp.dataSourceElementCollectionKey()

	case ColumnValueRetentionDurationKeyID:
		return cnp.columnValueRetentionDurationKey(components[0])
	case ColumnValueRetentionDurationCollectionKeyID:
		return cnp.columnValueRetentionDurationCollectionKey()

	case SQLShimDatabaseKeyID:
		return cnp.sqlShimDatabaseKey(components[0])
	case SQLShimDatabaseNameKeyID:
		return cnp.sqlShimDatabaseNameKey(components[0])
	case SQLShimDatabaseCollectionKeyID:
		return cnp.sqlShimDatabaseCollectionKey()

	case ShimObjectStoreKeyID:
		return cnp.shimObjectStoreKey(components[0])
	case ShimObjectStoreNameKeyID:
		return cnp.shimObjectStoreNameKey(components[0])
	case ShimObjectStoreCollectionKeyID:
		return cnp.shimObjectStoreCollectionKey()

	case AccessorSearchIndexKeyID:
		return cnp.accessorSearchIndexKey(components[0])
	case AccessorSearchIndexCollectionKeyID:
		return cnp.accessorSearchIndexCollectionKey()

	case UserSearchIndexKeyID:
		return cnp.userSearchIndexKey(components[0])
	case UserSearchIndexCollectionKeyID:
		return cnp.userSearchIndexCollectionKey()
	}
	return ""
}

// GetRateLimitKeyName gets the rate limit key name for the given key name ID and components
func (cnp *CacheNameProvider) GetRateLimitKeyName(id cache.KeyNameID, keySuffix string) cache.RateLimitKey {
	if id == AccessPolicyRateLimitKeyID {
		return cache.RateLimitKey(cnp.accessPolicyRateLimitKey(keySuffix))
	}

	return ""
}

func (cnp *CacheNameProvider) accessPolicyRateLimitKey(keySuffix string) string {
	return fmt.Sprintf(
		"%v_%v_%v",
		cnp.basePrefix,
		accessPolicyRateLimitPrefix,
		keySuffix,
	)
}

// dependencyKey returns key name for dependency keys
func (cnp *CacheNameProvider) dependencyKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, dependencyPrefix, id))
}

// isModifiedKey returns key name for isModified key
func (cnp *CacheNameProvider) isModifiedKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, isModifiedPrefix, id))
}

// isModifiedCollectionKey returns key name for isModified key
func (cnp *CacheNameProvider) isModifiedCollectionKey(colKey string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", colKey, isModifiedPrefix))
}

// dataTypeKey primary key for DataType
func (cnp *CacheNameProvider) dataTypeKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, dataTypePrefix, id))
}

// dataTypeCollectionKey returns key name for DataType collection
func (cnp *CacheNameProvider) dataTypeCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", cnp.basePrefix, dataTypeCollectionKeyString))
}

// columnKey primary key for column
func (cnp *CacheNameProvider) columnKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, columnPrefix, id))
}

// columnNameKey return secondary key for column (name based)
func (cnp *CacheNameProvider) columnNameKey(database, table, name string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v_%v_%v", cnp.basePrefix, columnPrefix, database, table, name))
}

// columnCollectionKey returns key name for column collection
func (cnp *CacheNameProvider) columnCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", cnp.basePrefix, columnCollectionKeyString))
}

// accessorKey primary key for accessor
func (cnp *CacheNameProvider) accessorKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, accessorPrefix, id))
}

// accessorKey secondary key for accessor (name based)
func (cnp *CacheNameProvider) accessorNameKey(name string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, accessorPrefix, name))
}

// accessorCollectionKey returns key name for accessor collection
func (cnp *CacheNameProvider) accessorCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", cnp.basePrefix, accessorCollectionKeyString))
}

// accessPolicyKey primary key for access policy
func (cnp *CacheNameProvider) accessPolicyKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, accessPolicyPrefix, id))
}

// accessPolicyKey secondary key for access policy (name based)
func (cnp *CacheNameProvider) accessPolicyNameKey(name string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, accessPolicyPrefix, name))
}

// accessPolicyCollectionKey returns key name for access policy collection
func (cnp *CacheNameProvider) accessPolicyCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", cnp.basePrefix, accessPoliciesCollectionKeyString))
}

// accessPolicyTemplateKey primary key for access policy
func (cnp *CacheNameProvider) accessPolicyTemplateKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, accessPolicyTemplatePrefix, id))
}

// accessPolicyTemplateCollectionKey returns key name for access policy collection
func (cnp *CacheNameProvider) accessPolicyTemplateCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", cnp.basePrefix, accessPolicyTemplateCollectionKeyString))
}

// purposeKey primary key for purpose
func (cnp *CacheNameProvider) purposeKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, purposePrefix, id))
}

// purposeKey secondary key for purpose
func (cnp *CacheNameProvider) purposeNameKey(name string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, purposePrefix, name))
}

// purposeCollectionKey returns key name for purpose collection
func (cnp *CacheNameProvider) purposeCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", cnp.basePrefix, purposeCollectionKeyString))
}

// mutatorKey primary key for mutator
func (cnp *CacheNameProvider) mutatorKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, mutatorPrefix, id))
}

// mutatorKey secondary key for mutator (name based)
func (cnp *CacheNameProvider) mutatorNameKey(name string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, mutatorPrefix, name))
}

// mutatorCollectionKey returns key name for mutator collection
func (cnp *CacheNameProvider) mutatorCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", cnp.basePrefix, mutatorCollectionKeyString))
}

// transformerKey primary key for transformer
func (cnp *CacheNameProvider) transformerKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, transformerPrefix, id))
}

// transformerNameKey secondary key for transformer (name based)
func (cnp *CacheNameProvider) transformerNameKey(name string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, transformerPrefix, name))
}

// transformerCollectionKey returns key name for transformer collection
func (cnp *CacheNameProvider) transformerCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", cnp.basePrefix, transformerCollectionKeyString))
}

func (cnp *CacheNameProvider) dataSourceKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, dataSourcePrefix, id))
}

func (cnp *CacheNameProvider) dataSourceCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", cnp.basePrefix, dataSourceCollectionKeyString))
}

func (cnp *CacheNameProvider) dataSourceElementKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, dataSourceElementPrefix, id))
}

func (cnp *CacheNameProvider) dataSourceElementCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", cnp.basePrefix, dataSourceElementCollectionKeyString))
}

// columnValueRetentionDurationKey primary key for columnValueRetentionDuration
func (cnp *CacheNameProvider) columnValueRetentionDurationKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, columnValueRetentionDurationPrefix, id))
}

// columnValueRetentionDurationCollectionKey returns key name for columnValueRetentionDuration collection
func (cnp *CacheNameProvider) columnValueRetentionDurationCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", cnp.basePrefix, columnValueRetentionDurationCollectionKeyString))
}

func (cnp *CacheNameProvider) sqlShimDatabaseKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, SQLShimDatabaseKeyID, id))
}

func (cnp *CacheNameProvider) sqlShimDatabaseNameKey(name string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, SQLShimDatabaseNameKeyID, name))
}

func (cnp *CacheNameProvider) sqlShimDatabaseCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", cnp.basePrefix, SQLShimDatabaseCollectionKeyID))
}

func (cnp *CacheNameProvider) shimObjectStoreKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, ShimObjectStoreKeyID, id))
}

func (cnp *CacheNameProvider) shimObjectStoreNameKey(name string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, ShimObjectStoreNameKeyID, name))
}

func (cnp *CacheNameProvider) shimObjectStoreCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", cnp.basePrefix, ShimObjectStoreCollectionKeyID))
}

func (cnp *CacheNameProvider) accessorSearchIndexKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, AccessorSearchIndexKeyID, id))
}

func (cnp *CacheNameProvider) accessorSearchIndexCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", cnp.basePrefix, AccessorSearchIndexCollectionKeyID))
}

func (cnp *CacheNameProvider) userSearchIndexKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, UserSearchIndexKeyID, id))
}

func (cnp *CacheNameProvider) userSearchIndexCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", cnp.basePrefix, UserSearchIndexCollectionKeyID))
}

// columnCollectionPagesKey returns key name for edge collection
func (cnp *CacheNameProvider) columnCollectionPagesKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", cnp.basePrefix, columnCollectionKeyString, columnCollectionPagesPrefixString))
}

// columnCollectionPageKey returns a key name for each individual page in the global collection of edges
func (cnp *CacheNameProvider) columnCollectionPageKey(cursor string, limit string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v_%v", cnp.basePrefix, columnCollectionKeyString, cursor, limit))
}

// GetPrimaryKey returns the primary cache key name for column
func (c Column) GetPrimaryKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(ColumnKeyID, c.ID)
}

// GetGlobalCollectionKey returns the global collection key name for column
func (Column) GetGlobalCollectionKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameStatic(ColumnCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for  column
func (c Column) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameStatic(ColumnCollectionPagesKeyID)
}

// GetSecondaryKeys returns the secondary cache key names for column
func (c Column) GetSecondaryKeys(knp cache.KeyNameProvider) []cache.Key {
	return []cache.Key{knp.GetKeyName(ColumnNameKeyID, []string{c.SQLShimDatabaseID.String(), strings.ToLower(c.Table), strings.ToLower(c.Name)})}
}

// GetPerItemCollectionKey returns the per item collection key name for column
func (Column) GetPerItemCollectionKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per object type, could store objects of this type in the future
}

// GetDependenciesKey returns the dependencies key name for column
func (Column) GetDependenciesKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused since we track dependencies explicitly
}

// GetIsModifiedKey returns the isModifiedKey key name for column
func (c Column) GetIsModifiedKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(IsModifiedKeyID, c.ID)
}

// GetIsModifiedCollectionKey returns the isModifiedCollectionKey key name for column
func (c Column) GetIsModifiedCollectionKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithString(IsModifiedCollectionKeyID, string(c.GetGlobalCollectionKey(knp)))
}

// GetDependencyKeys returns the list of keys for object type dependencies
func (Column) GetDependencyKeys(cache.KeyNameProvider) []cache.Key {
	return []cache.Key{} // ObjectTypes don't depend on anything
}

// TTL returns the TTL for object type
func (Column) TTL(ttlp cache.TTLProvider) time.Duration {
	return ttlp.TTL(ColumnTTL)
}

// GetPrimaryKey returns the primary cache key name for accessor
func (a Accessor) GetPrimaryKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(AccessorKeyID, a.ID)
}

// GetGlobalCollectionKey returns the global collection key name for accessor
func (Accessor) GetGlobalCollectionKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameStatic(AccessorCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for  accessor
func (a Accessor) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for accessor
}

// GetPerItemCollectionKey returns the per item collection key name for accessor
func (Accessor) GetPerItemCollectionKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per accessor, could store edges of this type in the future
}

// GetSecondaryKeys returns the secondary cache key names for accessor
func (a Accessor) GetSecondaryKeys(knp cache.KeyNameProvider) []cache.Key {
	return []cache.Key{knp.GetKeyNameWithString(AccessorNameKeyID, a.Name)}
}

// GetDependenciesKey returns the dependencies key name for accessor
func (Accessor) GetDependenciesKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused since we track dependencies explicitly
}

// GetIsModifiedKey returns the isModifiedKey key name for accessor
func (a Accessor) GetIsModifiedKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(IsModifiedKeyID, a.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for accessor
func (a Accessor) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for accessor dependencies
func (Accessor) GetDependencyKeys(cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// TTL returns the TTL for accessor
func (Accessor) TTL(ttlp cache.TTLProvider) time.Duration {
	return ttlp.TTL(AccessorTTL)
}

// GetPrimaryKey returns the primary cache key name for access policy
func (ap AccessPolicy) GetPrimaryKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(AccessPolicyKeyID, ap.ID)
}

// GetGlobalCollectionKey returns the global collection key name for access policy
func (AccessPolicy) GetGlobalCollectionKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameStatic(AccessPolicyCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for access policy
func (AccessPolicy) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for access policy
}

// GetPerItemCollectionKey returns the per item collection key name for access policy
func (AccessPolicy) GetPerItemCollectionKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per access policy, could store edges of this type in the future
}

// GetSecondaryKeys returns the secondary cache key names for access policy
func (ap AccessPolicy) GetSecondaryKeys(knp cache.KeyNameProvider) []cache.Key {
	return []cache.Key{knp.GetKeyNameWithString(AccessPolicyNameKeyID, ap.Name)}
}

// GetDependenciesKey returns the dependencies key name for access policy
func (AccessPolicy) GetDependenciesKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused as all dependent resources have to be deleted before AccessPolicy can be deleted
}

// GetIsModifiedKey returns the isModifiedKey key name for access policy
func (ap AccessPolicy) GetIsModifiedKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(IsModifiedKeyID, ap.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for access policy
func (ap AccessPolicy) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for access policy dependencies
func (AccessPolicy) GetDependencyKeys(cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// TTL returns the TTL for access policy
func (AccessPolicy) TTL(ttlp cache.TTLProvider) time.Duration {
	return ttlp.TTL(AccessPolicyTTL)
}

// GetPrimaryKey returns the primary cache key name for access policy template
func (apt AccessPolicyTemplate) GetPrimaryKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(AccessPolicyTemplateKeyID, apt.ID)
}

// GetGlobalCollectionKey returns the global collection key name for access policy template
func (AccessPolicyTemplate) GetGlobalCollectionKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameStatic(AccessPolicyTemplateCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for access policy template
func (AccessPolicyTemplate) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for access policy template
}

// GetPerItemCollectionKey returns the per item collection key name for access policy template
func (AccessPolicyTemplate) GetPerItemCollectionKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per access policy, could store edges of this type in the future
}

// GetSecondaryKeys returns the secondary cache key names for access policy template
func (AccessPolicyTemplate) GetSecondaryKeys(cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// GetDependenciesKey returns the dependencies key name for access policy template
func (AccessPolicyTemplate) GetDependenciesKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused as all dependent resources have to be deleted before AccessPolicyTemplate can be deleted
}

// GetIsModifiedKey returns the isModifiedKey key name for access policy template
func (apt AccessPolicyTemplate) GetIsModifiedKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(IsModifiedKeyID, apt.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for access policy template
func (apt AccessPolicyTemplate) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for access policy template dependencies
func (AccessPolicyTemplate) GetDependencyKeys(cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// TTL returns the TTL for access policy
func (AccessPolicyTemplate) TTL(ttlp cache.TTLProvider) time.Duration {
	return ttlp.TTL(AccessPolicyTemplateTTL)
}

// GetPrimaryKey returns the primary cache key name for purpose
func (p Purpose) GetPrimaryKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(PurposeKeyID, p.ID)
}

// GetGlobalCollectionKey returns the global collection key name for purpose
func (Purpose) GetGlobalCollectionKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameStatic(PurposeCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for purpose
func (Purpose) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for purpose
}

// GetPerItemCollectionKey returns the per item collection key name for purpose
func (Purpose) GetPerItemCollectionKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per purpose, could store edges of this type in the future
}

// GetSecondaryKeys returns the secondary cache key names for purpose
func (p Purpose) GetSecondaryKeys(knp cache.KeyNameProvider) []cache.Key {
	return []cache.Key{knp.GetKeyNameWithString(PurposeNameKeyID, strings.ToLower(p.Name))}
}

// GetDependenciesKey returns the dependencies key name for purpose
func (Purpose) GetDependenciesKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused as all dependent resources have to be deleted before Purpose can be deleted
}

// GetIsModifiedKey returns the isModifiedKey key name for purpose
func (p Purpose) GetIsModifiedKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(IsModifiedKeyID, p.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for purpose
func (p Purpose) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for purpose dependencies
func (Purpose) GetDependencyKeys(knp cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// TTL returns the TTL for purpose
func (Purpose) TTL(ttlp cache.TTLProvider) time.Duration {
	return ttlp.TTL(PurposeTTL)
}

// GetPrimaryKey returns the primary cache key name for mutator
func (m Mutator) GetPrimaryKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(MutatorKeyID, m.ID)
}

// GetGlobalCollectionKey returns the global collection key name for mutator
func (Mutator) GetGlobalCollectionKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameStatic(MutatorCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for mutator
func (Mutator) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for mutator
}

// GetPerItemCollectionKey returns the per item collection key name for mutator
func (Mutator) GetPerItemCollectionKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per mutator, could store edges of this type in the future
}

// GetSecondaryKeys returns the secondary cache key names for mutator
func (m Mutator) GetSecondaryKeys(knp cache.KeyNameProvider) []cache.Key {
	return []cache.Key{knp.GetKeyNameWithString(MutatorNameKeyID, m.Name)}
}

// GetDependenciesKey returns the dependencies key name for mutator
func (Mutator) GetDependenciesKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused as all dependent resources have to be deleted before Mutator can be deleted
}

// GetIsModifiedKey returns the isModifiedKey key name for mutator
func (m Mutator) GetIsModifiedKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(IsModifiedKeyID, m.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for mutator
func (m Mutator) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for mutator dependencies
func (Mutator) GetDependencyKeys(cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// TTL returns the TTL for mutator
func (Mutator) TTL(ttlp cache.TTLProvider) time.Duration {
	return ttlp.TTL(MutatorTTL)
}

// GetPrimaryKey returns the primary cache key name for transformer
func (t Transformer) GetPrimaryKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(TransformerKeyID, t.ID)
}

// GetGlobalCollectionKey returns the global collection key name for transformer
func (Transformer) GetGlobalCollectionKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameStatic(TransformerCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for transformer
func (Transformer) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for transformer
}

// GetPerItemCollectionKey returns the per item collection key name for transformer
func (Transformer) GetPerItemCollectionKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per transformer, could store edges of this type in the future
}

// GetSecondaryKeys returns the secondary cache key names for transformer
func (t Transformer) GetSecondaryKeys(knp cache.KeyNameProvider) []cache.Key {
	return []cache.Key{knp.GetKeyNameWithString(TransformerNameKeyID, t.Name)}
}

// GetDependenciesKey returns the dependencies key name for transformer
func (Transformer) GetDependenciesKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused as all dependent resources have to be deleted before Transformer can be deleted
}

// GetIsModifiedKey returns the isModifiedKey key name for transformer
func (t Transformer) GetIsModifiedKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(IsModifiedKeyID, t.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for transformer
func (t Transformer) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for transformer dependencies
func (Transformer) GetDependencyKeys(cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// TTL returns the TTL for transformer
func (Transformer) TTL(ttlp cache.TTLProvider) time.Duration {
	return ttlp.TTL(TransformerTTL)
}

// GetPrimaryKey returns the primary cache key name for column value retention duration
func (cvrd ColumnValueRetentionDuration) GetPrimaryKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(ColumnValueRetentionDurationKeyID, cvrd.ID)
}

// GetGlobalCollectionKey returns the global collection key name for column value retention duration
func (ColumnValueRetentionDuration) GetGlobalCollectionKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameStatic(ColumnValueRetentionDurationCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for column value retention duration
func (ColumnValueRetentionDuration) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for column value retention duration
}

// GetPerItemCollectionKey returns the per item collection key name for column value retention duration
func (ColumnValueRetentionDuration) GetPerItemCollectionKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per mutator, could store edges of this type in the future
}

// GetSecondaryKeys returns the secondary cache key names for column value retention duration
func (ColumnValueRetentionDuration) GetSecondaryKeys(cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// GetDependenciesKey returns the dependencies key name for column value retention duration
func (ColumnValueRetentionDuration) GetDependenciesKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused as all dependent resources have to be deleted before Mutator can be deleted
}

// GetIsModifiedKey returns the isModifiedKey key name for column value retention duration
func (cvrd ColumnValueRetentionDuration) GetIsModifiedKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(IsModifiedKeyID, cvrd.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for column value retention duration
func (ColumnValueRetentionDuration) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for mutator dependencies
func (ColumnValueRetentionDuration) GetDependencyKeys(cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// TTL returns the TTL for mutator
func (ColumnValueRetentionDuration) TTL(ttlp cache.TTLProvider) time.Duration {
	return ttlp.TTL(ColumnValueRetentionDurationTTL)
}

// GetPrimaryKey returns the primary cache key name for SQLShimDatabase
func (s SQLShimDatabase) GetPrimaryKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(SQLShimDatabaseKeyID, s.ID)
}

// GetGlobalCollectionKey returns the global collection key name for SQLShimDatabase
func (SQLShimDatabase) GetGlobalCollectionKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameStatic(SQLShimDatabaseCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for SQLShimDatabase
func (SQLShimDatabase) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for SQLShimDatabase
}

// GetPerItemCollectionKey returns the per item collection key name for SQLShimDatabase
func (SQLShimDatabase) GetPerItemCollectionKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per SQLShimDatabase, could store edges of this type in the future
}

// GetSecondaryKeys returns the secondary cache key names for SQLShimDatabase
func (s SQLShimDatabase) GetSecondaryKeys(knp cache.KeyNameProvider) []cache.Key {
	return []cache.Key{knp.GetKeyNameWithString(SQLShimDatabaseNameKeyID, s.Name)}
}

// GetDependenciesKey returns the dependencies key name for SQLShimDatabase
func (SQLShimDatabase) GetDependenciesKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused as all dependent resources have to be deleted before SQLShimDatabase can be deleted
}

// GetIsModifiedKey returns the isModifiedKey key name for SQLShimDatabase
func (s SQLShimDatabase) GetIsModifiedKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(IsModifiedKeyID, s.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for SQLShimDatabase
func (SQLShimDatabase) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for SQLShimDatabase dependencies
func (SQLShimDatabase) GetDependencyKeys(knp cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// TTL returns the TTL for SQLShimDatabase
func (SQLShimDatabase) TTL(ttlp cache.TTLProvider) time.Duration {
	return ttlp.TTL(SQLShimDatabaseTTL)
}

// GetPrimaryKey returns the primary cache key name for ShimObjectStore
func (s ShimObjectStore) GetPrimaryKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(ShimObjectStoreKeyID, s.ID)
}

// GetGlobalCollectionKey returns the global collection key name for ShimObjectStore
func (ShimObjectStore) GetGlobalCollectionKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameStatic(ShimObjectStoreCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for ShimObjectStore
func (ShimObjectStore) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for ShimObjectStore
}

// GetPerItemCollectionKey returns the per item collection key name for ShimObjectStore
func (ShimObjectStore) GetPerItemCollectionKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per ShimObjectStore, could store edges of this type in the future
}

// GetSecondaryKeys returns the secondary cache key names for ShimObjectStore
func (s ShimObjectStore) GetSecondaryKeys(knp cache.KeyNameProvider) []cache.Key {
	return []cache.Key{knp.GetKeyNameWithString(ShimObjectStoreNameKeyID, s.Name)}
}

// GetDependenciesKey returns the dependencies key name for ShimObjectStore
func (ShimObjectStore) GetDependenciesKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused as all dependent resources have to be deleted before ShimObjectStore can be deleted
}

// GetIsModifiedKey returns the isModifiedKey key name for ShimObjectStore
func (s ShimObjectStore) GetIsModifiedKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(IsModifiedKeyID, s.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for ShimObjectStore
func (ShimObjectStore) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for ShimObjectStore dependencies
func (ShimObjectStore) GetDependencyKeys(knp cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// TTL returns the TTL for ShimObjectStore
func (ShimObjectStore) TTL(ttlp cache.TTLProvider) time.Duration {
	return ttlp.TTL(ShimObjectStoreTTL)
}

// GetPrimaryKey returns the primary cache key name for accessor search index
func (asi AccessorSearchIndex) GetPrimaryKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(AccessorSearchIndexKeyID, asi.ID)
}

// GetGlobalCollectionKey returns the global collection key name for accessor search index
func (AccessorSearchIndex) GetGlobalCollectionKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameStatic(AccessorSearchIndexCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for accessor search index
func (AccessorSearchIndex) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for accessor search index
}

// GetPerItemCollectionKey returns the per item collection key name for accessor search index
func (AccessorSearchIndex) GetPerItemCollectionKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused
}

// GetSecondaryKeys returns the secondary cache key names for accessor search index
func (AccessorSearchIndex) GetSecondaryKeys(cache.KeyNameProvider) []cache.Key {
	return []cache.Key{} // Unused
}

// GetDependenciesKey returns the dependencies key name for acccessor search index
func (AccessorSearchIndex) GetDependenciesKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused
}

// GetIsModifiedKey returns the isModifiedKey key name for accessor search index
func (asi AccessorSearchIndex) GetIsModifiedKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(IsModifiedKeyID, asi.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for accessor search index
func (AccessorSearchIndex) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn on page caching
}

// GetDependencyKeys returns the list of keys for accessor search index dependencies
func (AccessorSearchIndex) GetDependencyKeys(cache.KeyNameProvider) []cache.Key {
	return []cache.Key{} // Unused
}

// TTL returns the TTL for accessor search index
func (AccessorSearchIndex) TTL(ttlp cache.TTLProvider) time.Duration {
	return ttlp.TTL(AccessorSearchIndexTTL)
}

// GetPrimaryKey returns the primary cache key name for user search index
func (usi UserSearchIndex) GetPrimaryKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(UserSearchIndexKeyID, usi.ID)
}

// GetGlobalCollectionKey returns the global collection key name for user search index
func (UserSearchIndex) GetGlobalCollectionKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameStatic(UserSearchIndexCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection pages key name for user search index
func (UserSearchIndex) GetGlobalCollectionPagesKey(knp cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for user search index
}

// GetPerItemCollectionKey returns the per item collection key name for user search index
func (UserSearchIndex) GetPerItemCollectionKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused
}

// GetSecondaryKeys returns the secondary cache key names for user search index
func (UserSearchIndex) GetSecondaryKeys(cache.KeyNameProvider) []cache.Key {
	return []cache.Key{} // Unused
}

// GetDependenciesKey returns the dependencies key name for user search index
func (UserSearchIndex) GetDependenciesKey(cache.KeyNameProvider) cache.Key {
	return "" // Unused
}

// GetIsModifiedKey returns the isModifiedKey key name for user search index
func (usi UserSearchIndex) GetIsModifiedKey(knp cache.KeyNameProvider) cache.Key {
	return knp.GetKeyNameWithID(IsModifiedKeyID, usi.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for user search index
func (UserSearchIndex) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn on page caching
}

// GetDependencyKeys returns the list of keys for user search index dependencies
func (UserSearchIndex) GetDependencyKeys(cache.KeyNameProvider) []cache.Key {
	return []cache.Key{} // Unused
}

// TTL returns the TTL for user search index
func (UserSearchIndex) TTL(ttlp cache.TTLProvider) time.Duration {
	return ttlp.TTL(UserSearchIndexTTL)
}

// idpCacheTTLProvider implements the cache.CacheTTLProvider interface
type idpCacheTTLProvider struct {
	dataTypeTTL                     time.Duration
	columnTTL                       time.Duration
	accessorTTL                     time.Duration
	accessPolicyTTL                 time.Duration
	accessPolicyRateLimitTTL        time.Duration
	AccessPolicyTemplateTTL         time.Duration
	PurposeTTL                      time.Duration
	MutatorTTL                      time.Duration
	TransformerTTL                  time.Duration
	ColumnValueRetentionDurationTTL time.Duration
	SQLShimDatabaseTTL              time.Duration
	ShimObjectStoreTTL              time.Duration
	AccessorSearchIndexTTL          time.Duration
	UserSearchIndexTTL              time.Duration
}

// newIDPCacheTTLProvider creates a new Configurablecache.CacheTTLProvider
func newIDPCacheTTLProvider(ttl time.Duration) *idpCacheTTLProvider {
	return &idpCacheTTLProvider{
		dataTypeTTL:                     ttl,
		columnTTL:                       ttl,
		accessorTTL:                     ttl,
		accessPolicyTTL:                 ttl,
		accessPolicyRateLimitTTL:        ttl,
		AccessPolicyTemplateTTL:         ttl,
		PurposeTTL:                      ttl,
		MutatorTTL:                      ttl,
		TransformerTTL:                  ttl,
		ColumnValueRetentionDurationTTL: ttl,
		SQLShimDatabaseTTL:              ttl,
		ShimObjectStoreTTL:              ttl,
		AccessorSearchIndexTTL:          ttl,
		UserSearchIndexTTL:              ttl,
	}
}

const (
	// ColumnTTL - TTL for Column
	ColumnTTL = "COLUMN_TTL"
	// AccessorTTL - TTL for Accessor
	AccessorTTL = "ACCESSOR_TTL"
	// AccessPolicyTTL - TTL for Access Policy
	AccessPolicyTTL = "ACCESS_POLICY_TTL"
	// AccessPolicyRateLimitTTL - TTL for Access Policy Rate Limit
	AccessPolicyRateLimitTTL = "ACCESS_POLICY_RATE_LIMIT_TTL"
	// AccessPolicyTemplateTTL - TTL for Access Policy Template
	AccessPolicyTemplateTTL = "ACCESS_POLICY_TEMPLATE_TTL"
	// PurposeTTL - TTL for Purpose
	PurposeTTL = "PURPOSE_TTL"
	// MutatorTTL - TTL for Mutator
	MutatorTTL = "MUTATOR_TTL"
	// TransformerTTL - TTL for Transformer
	TransformerTTL = "TRANSFORMER_TTL"
	// ColumnValueRetentionDurationTTL - TTL for ColumnValueRetentionDuration
	ColumnValueRetentionDurationTTL = "COLUMNVALUERETENTIONDURATION_TTL"
	// SQLShimDatabaseTTL - TTL for SQLShimDatabase
	SQLShimDatabaseTTL = "SQLSHIMDB_TTL"
	// ShimObjectStoreTTL - TTL for ShimObjectStore
	ShimObjectStoreTTL = "SHIMOBJECTSTORE_TTL"
	// AccessorSearchIndexTTL - TTL for AccessorSearchIndex
	AccessorSearchIndexTTL = "ACCESSORSEARCHINDEX_TTL"
	// UserSearchIndexTTL - TTL for UserSearchIndex
	UserSearchIndexTTL = "USERSEARCHINDEX_TTL"
)

// TTL returns the TTL for given type
func (p *idpCacheTTLProvider) TTL(id cache.KeyTTLID) time.Duration {
	switch id {
	case column.DataTypeTTL:
		return p.dataTypeTTL
	case ColumnTTL:
		return p.columnTTL
	case AccessorTTL:
		return p.accessorTTL
	case AccessPolicyTTL:
		return p.accessorTTL
	case AccessPolicyRateLimitTTL:
		return p.accessPolicyRateLimitTTL
	case AccessPolicyTemplateTTL:
		return p.AccessPolicyTemplateTTL
	case PurposeTTL:
		return p.PurposeTTL
	case MutatorTTL:
		return p.MutatorTTL
	case TransformerTTL:
		return p.TransformerTTL
	case ColumnValueRetentionDurationTTL:
		return p.ColumnValueRetentionDurationTTL
	case SQLShimDatabaseTTL:
		return p.SQLShimDatabaseTTL
	case ShimObjectStoreTTL:
		return p.ShimObjectStoreTTL
	case AccessorSearchIndexTTL:
		return p.AccessorSearchIndexTTL
	case UserSearchIndexTTL:
		return p.UserSearchIndexTTL
	}
	return cache.SkipCacheTTL
}
