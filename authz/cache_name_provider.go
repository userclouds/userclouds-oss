package authz

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/uclog"
)

const (
	// CachePrefix is the prefix for all keys in authz cache
	CachePrefix                 = "authz"
	objTypePrefix               = "OBJTYPE"      // Primary key for object type
	objTypeCollectionKeyString  = "OBJTYPE_COL"  // Global collection for object type
	edgeTypePrefix              = "EDGETYPE"     // Primary key for edge type
	edgeTypeCollectionKeyString = "EDGETYPE_COL" // Global collection for edge type
	objPrefix                   = "OBJ"          // Primary key for object
	objCollectionKeyString      = "OBJ_COL"      // Global collection for object
	objEdgeCollection           = "OBJEDGES"     // Per object collection of all in/out edges
	perObjectEdgesPrefix        = "E"            // Per object collection of source/target edges
	perObjectPathPrefix         = "P"            // Per object collection containing path for a particular source/target/attribute
	edgePrefix                  = "EDGE"         // Primary key for edge
	// EdgeCollectionKeyString is the string identifier for the global collection of edges and collections MOD key
	EdgeCollectionKeyString         = "EDGE_COL" // Global collection for edge
	edgeCollectionPagesPrefixString = "PAGES"    // Pages making up global collection of edges
	orgPrefix                       = "ORG"      // Primary key for organization
	orgCollectionKeyString          = "ORG_COL"  // Global collection for organizations
	dependencyPrefix                = "DEP"      // Shared dependency key prefix among all items
	isModifiedPrefix                = "MOD"      // Shared is modified key prefix among all items
)

// CacheNameProvider is the base implementation of the CacheNameProvider interface
type CacheNameProvider struct {
	cache.NoRateLimitKeyNameProvider
	basePrefix string // Base prefix for all keys TenantID_OrgID
}

// NewCacheNameProviderForTenant creates a new authz CacheNameProvider for a tenant
func NewCacheNameProviderForTenant(tenantID uuid.UUID) *CacheNameProvider {
	return NewCacheNameProvider(fmt.Sprintf("%v_%v", CachePrefix, tenantID))
}

// NewCacheNameProvider creates a new BasesCacheNameProvider
func NewCacheNameProvider(basePrefix string) *CacheNameProvider {
	return &CacheNameProvider{basePrefix: basePrefix}
}

const (
	// ObjectTypeKeyID is the primary key for object type
	ObjectTypeKeyID = "ObjTypeKeyID"
	// EdgeTypeKeyID is the primary key for edge type
	EdgeTypeKeyID = "EdgeTypeKeyID"
	// ObjectKeyID is the primary key for object
	ObjectKeyID = "ObjectKeyID"
	// EdgeKeyID is the primary key for edge
	EdgeKeyID = "EdgeKeyID"
	// OrganizationKeyID is the primary key for organization
	OrganizationKeyID = "OrgKeyID"
	// EdgeFullKeyID is the secondary key for edge
	EdgeFullKeyID = "EdgeFullKeyNameID"
	// ObjectTypeNameKeyID is the secondary key for object type
	ObjectTypeNameKeyID = "ObjectTypeKeyNameID"
	// ObjEdgesKeyID is the key for collection of edges of an object
	ObjEdgesKeyID = "ObjectEdgesKeyID"
	// EdgeTypeNameKeyID is the secondary key for edge type
	EdgeTypeNameKeyID = "EdgeTypeKeyNameID"
	// ObjAliasNameKeyID is the secondary key for object
	ObjAliasNameKeyID = "ObjAliasKeyNameID"
	// OrganizationNameKeyID is the secondary key for organization
	OrganizationNameKeyID = "OrgCollectionKeyNameID"
	// EdgesObjToObjID is the key for collection of edges between two objects
	EdgesObjToObjID = "EdgesObjToObjID"
	// DependencyKeyID is the key for list of dependencies
	DependencyKeyID = "DependencyKeyID"
	// IsModifiedKeyID is the key value indicating change in last TTL
	IsModifiedKeyID = "IsModifiedKeyID"
	// IsModifiedCollectionKeyID is the key value indicating change for global colleciton in last TTL
	IsModifiedCollectionKeyID = "IsModifiedCollectionKeyID"
	// ObjectTypeCollectionKeyID is the key for global collection of object types
	ObjectTypeCollectionKeyID = "ObjTypeCollectionKeyID"
	// EdgeTypeCollectionKeyID is the key for global collection of edge types
	EdgeTypeCollectionKeyID = "EdgeTypeCollectionKeyID"
	// ObjectCollectionKeyID is the key for global collection of objects
	ObjectCollectionKeyID = "ObjCollectionKeyID"
	// EdgeCollectionKeyID is the key for global collection of edges
	EdgeCollectionKeyID = "EdgeCollectionKeyID"
	// EdgeCollectionPagesKeyID is the key for pages making up global collection of edges
	EdgeCollectionPagesKeyID = "EdgeCollectionPagesKeyID"
	// EdgeCollectionPageKeyID is the key for each individual page in the global collection of edges
	EdgeCollectionPageKeyID = "EdgeCollectionPageKeyID"
	// OrganizationCollectionKeyID is the key for global collection of organizations
	OrganizationCollectionKeyID = "OrgCollectionKeyID"
	// AttributePathObjToObjID is the primary key for attribute path
	AttributePathObjToObjID = "AttributePathObjToObjID"
)

// GetPrefix returns the base prefix for all keys
func (c *CacheNameProvider) GetPrefix() string {
	return c.basePrefix
}

// GetAllKeyIDs returns all the key IDs
func (c *CacheNameProvider) GetAllKeyIDs() []string {
	return []string{
		ObjectTypeKeyID,
		EdgeTypeKeyID,
		ObjectKeyID,
		EdgeKeyID,
		OrganizationKeyID,
		EdgeFullKeyID,
		ObjectTypeNameKeyID,
		ObjEdgesKeyID,
		EdgeTypeNameKeyID,
		ObjAliasNameKeyID,
		OrganizationNameKeyID,
		EdgesObjToObjID,
		DependencyKeyID,
		IsModifiedKeyID,
		IsModifiedCollectionKeyID,
		ObjectTypeCollectionKeyID,
		EdgeTypeCollectionKeyID,
		ObjectCollectionKeyID,
		EdgeCollectionKeyID,
		EdgeCollectionPagesKeyID,
		EdgeCollectionPageKeyID,
		OrganizationCollectionKeyID,
		AttributePathObjToObjID,
	}
}

// GetKeyNameStatic is a shortcut for GetKeyName with without components
func (c *CacheNameProvider) GetKeyNameStatic(id cache.KeyNameID) cache.Key {
	return c.GetKeyName(id, []string{})
}

// GetKeyNameWithID is a shortcut for GetKeyName with a single uuid ID component
func (c *CacheNameProvider) GetKeyNameWithID(id cache.KeyNameID, itemID uuid.UUID) cache.Key {
	return c.GetKeyName(id, []string{itemID.String()})
}

// GetKeyNameWithString is a shortcut for GetKeyName with a single string component
func (c *CacheNameProvider) GetKeyNameWithString(id cache.KeyNameID, itemName string) cache.Key {
	return c.GetKeyName(id, []string{itemName})
}

// GetKeyName gets the key name for the given key name ID and components
func (c *CacheNameProvider) GetKeyName(id cache.KeyNameID, components []string) cache.Key {
	switch id {
	case ObjectTypeKeyID:
		return c.objectTypeKey(components[0])
	case EdgeTypeKeyID:
		return c.edgeTypeKey(components[0])
	case ObjectKeyID:
		return c.objectKey(components[0])
	case EdgeKeyID:
		return c.edgeKey(components[0])
	case OrganizationKeyID:
		return c.orgKey(components[0])
	case ObjectTypeNameKeyID:
		return c.objectTypeKeyName(components[0])
	case EdgeTypeNameKeyID:
		return c.edgeTypeKeyName(components[0])
	case ObjAliasNameKeyID:
		return c.objAliasKeyName(components[0], components[1], components[2])
	case OrganizationNameKeyID:
		return c.orgKeyName(components[0])
	case EdgesObjToObjID:
		return c.edgesObjToObj(components[0], components[1])

	case ObjectTypeCollectionKeyID:
		return c.objTypeCollectionKey()
	case EdgeTypeCollectionKeyID:
		return c.edgeTypeCollectionKey()
	case ObjectCollectionKeyID:
		return c.objCollectionKey()
	case EdgeCollectionKeyID:
		return c.edgeCollectionKey()
	case EdgeCollectionPagesKeyID:
		return c.edgeCollectionPagesKey()
	case EdgeCollectionPageKeyID:
		return c.edgeCollectionPageKey(components[0], components[1])
	case OrganizationCollectionKeyID:
		return c.orgCollectionKey()
	case ObjEdgesKeyID:
		return c.objectEdgesKey(components[0])
	case DependencyKeyID:
		return c.dependencyKey(components[0])
	case IsModifiedKeyID:
		return c.isModifiedKey(components[0])
	case IsModifiedCollectionKeyID:
		return c.isModifiedCollectionKey(components[0])
	case EdgeFullKeyID:
		return c.edgeFullKeyNameFromIDs(components[0], components[1], components[2])
	case AttributePathObjToObjID:
		return c.attributePathObjToObj(components[0], components[1], components[2])
	}
	return ""
}

// objectTypeKey primary key for object type
func (c *CacheNameProvider) objectTypeKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, objTypePrefix, id))
}

// edgeTypeKey primary key for edge type
func (c *CacheNameProvider) edgeTypeKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, edgeTypePrefix, id))
}

// objectKey primary key for object
func (c *CacheNameProvider) objectKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, objPrefix, id))
}

// edgeKey primary key for edge
func (c *CacheNameProvider) edgeKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, edgePrefix, id))
}

// orgKey primary key for edge
func (c *CacheNameProvider) orgKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, orgPrefix, id))
}

// objectTypeKeyName returns secondary key name for [objTypePrefix + TypeName] -> [ObjType] mapping
func (c *CacheNameProvider) objectTypeKeyName(typeName string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, objTypePrefix, typeName))
}

// objectEdgesKey returns key name for per object edges collection
func (c *CacheNameProvider) objectEdgesKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, objEdgeCollection, id))
}

// edgeTypeKeyName returns secondary key name for [edgeTypePrefix + TypeName] -> [EdgeType] mapping
func (c *CacheNameProvider) edgeTypeKeyName(typeName string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, edgeTypePrefix, typeName))
}

// objAliasKeyName returns key name for [TypeID + Alias] -> [Object] mapping
func (c *CacheNameProvider) objAliasKeyName(typeID string, alias string, orgID string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v_%v_%v", c.basePrefix, objPrefix, typeID, alias, orgID))
}

// edgesObjToObj returns key name for [SourceObjID _ TargetObjID] -> [Edge [] ] mapping
func (c *CacheNameProvider) edgesObjToObj(sourceObjID string, targetObjID string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v_%v_%v", c.basePrefix, objPrefix, sourceObjID, perObjectEdgesPrefix, targetObjID))
}

// edgeFullKeyNameFromIDs returns key name for [SourceObjID _ TargetObjID _ EdgeTypeID] -> [Edge] mapping
func (c *CacheNameProvider) edgeFullKeyNameFromIDs(sourceID string, targetID string, typeID string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v_%v_%v", c.basePrefix, edgePrefix, sourceID, targetID, typeID))
}

// orgKeyName returns secondary key name for [orgPrefix + Name] -> [Organization] mapping
func (c *CacheNameProvider) orgKeyName(orgName string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, orgPrefix, orgName))
}

// dependencyKey returns key name for dependency keys
func (c *CacheNameProvider) dependencyKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, dependencyPrefix, id))
}

// isModifiedKey returns key name for isModified key
func (c *CacheNameProvider) isModifiedKey(id string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, isModifiedPrefix, id))
}

// objTypeCollectionKey returns key name for object type collection
func (c *CacheNameProvider) objTypeCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", c.basePrefix, objTypeCollectionKeyString))
}

// edgeTypeCollectionKey returns key name for edge type collection
func (c *CacheNameProvider) edgeTypeCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", c.basePrefix, edgeTypeCollectionKeyString))
}

// objCollectionKey returns key name for object collection
func (c *CacheNameProvider) objCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", c.basePrefix, objCollectionKeyString))
}

// edgeCollectionKey returns key name for edge collection
func (c *CacheNameProvider) edgeCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", c.basePrefix, EdgeCollectionKeyString))
}

// edgeCollectionModifiedKey returns key name for edge collection
func (c *CacheNameProvider) isModifiedCollectionKey(colKey string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", colKey, isModifiedPrefix))
}

// edgeCollectionKey returns key name for edge collection
func (c *CacheNameProvider) edgeCollectionPagesKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v", c.basePrefix, EdgeCollectionKeyString, edgeCollectionPagesPrefixString))
}

func (c *CacheNameProvider) edgeCollectionPageKey(cursor string, limit string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v_%v", c.basePrefix, EdgeCollectionKeyString, cursor, limit))
}

// orgCollectionKey returns key name for edge collection
func (c *CacheNameProvider) orgCollectionKey() cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v", c.basePrefix, orgCollectionKeyString))
}

// attributePathObjToObj returns key name for attribute path
func (c *CacheNameProvider) attributePathObjToObj(sourceID string, targetID string, attributeName string) cache.Key {
	return cache.Key(fmt.Sprintf("%v_%v_%v_%v_%v_%v", c.basePrefix, objPrefix, sourceID, perObjectPathPrefix, targetID, attributeName))
}

// GetPrimaryKey returns the primary cache key name for object type
func (ot ObjectType) GetPrimaryKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(ObjectTypeKeyID, ot.ID)
}

// GetGlobalCollectionKey returns the global collection key name for object type
func (ot ObjectType) GetGlobalCollectionKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameStatic(ObjectTypeCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection key name for object type
func (ot ObjectType) GetGlobalCollectionPagesKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for object types
}

// GetSecondaryKeys returns the secondary cache key names for object type
func (ot ObjectType) GetSecondaryKeys(c cache.KeyNameProvider) []cache.Key {
	if ot.TypeName != "" {
		return []cache.Key{c.GetKeyNameWithString(ObjectTypeNameKeyID, ot.TypeName)}
	}
	uclog.Verbosef(context.Background(), "ObjectType %v has no name. Dropping secondary key", ot.ID)
	return []cache.Key{}
}

// GetPerItemCollectionKey returns the per item collection key name for object type
func (ot ObjectType) GetPerItemCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per object type, could store objects of this type in the future
}

// GetDependenciesKey returns the dependencies key name for object type
func (ot ObjectType) GetDependenciesKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since the whole cache is flushed on delete
}

// GetIsModifiedKey returns the isModifiedKey key name for object type
func (ot ObjectType) GetIsModifiedKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(IsModifiedKeyID, ot.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for object type
func (ot ObjectType) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for object type dependencies
func (ot ObjectType) GetDependencyKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{} // ObjectTypes don't depend on anything
}

// TTL returns the TTL for object type
func (ot ObjectType) TTL(c cache.TTLProvider) time.Duration {
	return c.TTL(ObjectTypeTTL)
}

// GetPrimaryKey returns the primary cache key name for edge type
func (et EdgeType) GetPrimaryKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(EdgeTypeKeyID, et.ID)
}

// GetGlobalCollectionKey returns the global collection key name for edge type
func (et EdgeType) GetGlobalCollectionKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameStatic(EdgeTypeCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection key name for edge type
func (et EdgeType) GetGlobalCollectionPagesKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for edge types
}

// GetPerItemCollectionKey returns the per item collection key name for edge type
func (et EdgeType) GetPerItemCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there nothing stored per edge type, could store edges of this type in the future
}

// GetSecondaryKeys returns the secondary cache key names for edge type
func (et EdgeType) GetSecondaryKeys(c cache.KeyNameProvider) []cache.Key {
	if et.TypeName != "" {
		return []cache.Key{c.GetKeyNameWithString(EdgeTypeNameKeyID, et.TypeName)}
	}
	uclog.Verbosef(context.Background(), "EdgeType %v has no name. Dropping secondary key", et.ID)
	return []cache.Key{}
}

// GetDependenciesKey returns the dependencies key name for edge type
func (et EdgeType) GetDependenciesKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since the whole cache is flushed on delete
}

// GetIsModifiedKey returns the isModifiedKey key name for edge type
func (et EdgeType) GetIsModifiedKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(IsModifiedKeyID, et.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for edge type
func (et EdgeType) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for edge type dependencies
func (et EdgeType) GetDependencyKeys(c cache.KeyNameProvider) []cache.Key {
	// EdgeTypes depend on source/target object types but we don't store that dependency because we currently flush the whole cache on object type delete
	return []cache.Key{}
}

// TTL returns the TTL for edge type
func (et EdgeType) TTL(c cache.TTLProvider) time.Duration {
	return c.TTL(EdgeTypeTTL)
}

// GetPrimaryKey returns the primary cache key name for object
func (o Object) GetPrimaryKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(ObjectKeyID, o.ID)
}

// GetSecondaryKeys returns the secondary cache key names for object
func (o Object) GetSecondaryKeys(c cache.KeyNameProvider) []cache.Key {
	if o.Alias != nil {
		return []cache.Key{c.GetKeyName(ObjAliasNameKeyID, []string{o.TypeID.String(), *o.Alias, o.OrganizationID.String()})}
	}
	return []cache.Key{}
}

// GetGlobalCollectionKey returns the global collection key name for object
func (o Object) GetGlobalCollectionKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameStatic(ObjectCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection key name for objects
func (o Object) GetGlobalCollectionPagesKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for objects
}

// GetPerItemCollectionKey returns the per item collection key name for object
func (o Object) GetPerItemCollectionKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(ObjEdgesKeyID, o.ID)
}

// GetDependenciesKey return dependencies cache key name for object
func (o Object) GetDependenciesKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(DependencyKeyID, o.ID)
}

// GetIsModifiedKey returns the isModifiedKey key name for object
func (o Object) GetIsModifiedKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(IsModifiedKeyID, o.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for object
func (o Object) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for object dependencies
func (o Object) GetDependencyKeys(c cache.KeyNameProvider) []cache.Key {
	// Objects depend on object types but we don't store that dependency because we currently flush the whole cache on object type delete
	return []cache.Key{}
}

// TTL returns the TTL for object
func (o Object) TTL(c cache.TTLProvider) time.Duration {
	return c.TTL(ObjectTTL)
}

// GetPrimaryKey returns the primary cache key name for edge
func (e Edge) GetPrimaryKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(EdgeKeyID, e.ID)
}

// GetGlobalCollectionKey returns the global collection cache key names for edge
func (e Edge) GetGlobalCollectionKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameStatic(EdgeCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection key name for edge
func (e Edge) GetGlobalCollectionPagesKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameStatic(EdgeCollectionPagesKeyID)
}

// GetPerItemCollectionKey returns the per item collection key name for edge
func (e Edge) GetPerItemCollectionKey(c cache.KeyNameProvider) cache.Key {
	return ""
}

// GetDependenciesKey return  dependencies cache key name for edge
func (e Edge) GetDependenciesKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(DependencyKeyID, e.ID)
}

// GetIsModifiedKey returns the isModifiedKey key name for edge
func (e Edge) GetIsModifiedKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(IsModifiedKeyID, e.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for edge
func (e Edge) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithString(IsModifiedCollectionKeyID, string(e.GetGlobalCollectionKey(c)))
}

// GetDependencyKeys returns the list of keys for edge dependencies
func (e Edge) GetDependencyKeys(c cache.KeyNameProvider) []cache.Key {
	// Edges depend on objects and edge types. We don't store edgetype dependency because we currently flush the whole cache on edge type delete
	return []cache.Key{c.GetKeyNameWithID(DependencyKeyID, e.SourceObjectID), c.GetKeyNameWithID(DependencyKeyID, e.TargetObjectID)}
}

// GetSecondaryKeys returns the secondary cache key names for edge
func (e Edge) GetSecondaryKeys(c cache.KeyNameProvider) []cache.Key {
	if !e.SourceObjectID.IsNil() || !e.TargetObjectID.IsNil() || !e.EdgeTypeID.IsNil() {
		return []cache.Key{c.GetKeyName(EdgeFullKeyID, []string{e.SourceObjectID.String(), e.TargetObjectID.String(), e.EdgeTypeID.String()})}
	}
	uclog.Verbosef(context.Background(), "Edge %v has no source, target or edge type. Dropping secondary key", e.ID)
	return []cache.Key{}
}

// TTL returns the TTL for edge
func (e Edge) TTL(c cache.TTLProvider) time.Duration {
	return c.TTL(EdgeTTL)
}

// GetPrimaryKey returns the primary cache key name for path node
func (e AttributePathNode) GetPrimaryKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since  AttributePathNode is not stored in cache directly
}

// GetGlobalCollectionKey returns the global collection cache key names for  path node
func (e AttributePathNode) GetGlobalCollectionKey(c cache.KeyNameProvider) cache.Key {
	return ""
}

// GetGlobalCollectionPagesKey returns the global collection key name for path node
func (e AttributePathNode) GetGlobalCollectionPagesKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for path node
}

// GetPerItemCollectionKey returns the per item collection key name for  path node
func (e AttributePathNode) GetPerItemCollectionKey(c cache.KeyNameProvider) cache.Key {
	return ""
}

// GetDependenciesKey return  dependencies cache key name for  path node
func (e AttributePathNode) GetDependenciesKey(c cache.KeyNameProvider) cache.Key {
	return ""
}

// GetIsModifiedKey returns the isModifiedKey key name for attribute path
func (e AttributePathNode) GetIsModifiedKey(c cache.KeyNameProvider) cache.Key {
	return ""
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for attribute path
func (e AttributePathNode) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for path node dependencies
func (e AttributePathNode) GetDependencyKeys(c cache.KeyNameProvider) []cache.Key {
	//  Path node depend on objects and edges.
	if e.EdgeID != uuid.Nil {
		return []cache.Key{c.GetKeyNameWithID(DependencyKeyID, e.EdgeID), c.GetKeyNameWithID(DependencyKeyID, e.ObjectID)}
	}
	return []cache.Key{c.GetKeyNameWithID(DependencyKeyID, e.ObjectID)}
}

// GetSecondaryKeys returns the secondary cache key names for path node
func (e AttributePathNode) GetSecondaryKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// TTL returns the TTL for  path node
func (e AttributePathNode) TTL(c cache.TTLProvider) time.Duration {
	return c.TTL(EdgeTTL) // Same TTL as edge
}

// GetPrimaryKey returns the primary cache key name for organization
func (o Organization) GetPrimaryKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(OrganizationKeyID, o.ID)
}

// GetGlobalCollectionKey returns the global collection cache key names for organization
func (o Organization) GetGlobalCollectionKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameStatic(OrganizationCollectionKeyID)
}

// GetGlobalCollectionPagesKey returns the global collection key name for organization
func (o Organization) GetGlobalCollectionPagesKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused since there is no pagination for organization
}

// GetPerItemCollectionKey returns the per item collection key name for organization (none)
func (o Organization) GetPerItemCollectionKey(c cache.KeyNameProvider) cache.Key {
	return ""
}

// GetDependenciesKey return  dependencies cache key name for organization
func (o Organization) GetDependenciesKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(DependencyKeyID, o.ID)
}

// GetIsModifiedKey returns the isModifiedKey key name for organization
func (o Organization) GetIsModifiedKey(c cache.KeyNameProvider) cache.Key {
	return c.GetKeyNameWithID(IsModifiedKeyID, o.ID)
}

// GetIsModifiedCollectionKey returns the IsModifiedCollectionKeyID key name for organization
func (o Organization) GetIsModifiedCollectionKey(c cache.KeyNameProvider) cache.Key {
	return "" // Unused until we turn one page caching
}

// GetDependencyKeys returns the list of keys for organization dependencies
func (o Organization) GetDependencyKeys(c cache.KeyNameProvider) []cache.Key {
	return []cache.Key{}
}

// GetSecondaryKeys returns the secondary cache key names for organization (none)
func (o Organization) GetSecondaryKeys(c cache.KeyNameProvider) []cache.Key {
	if o.Name != "" {
		return []cache.Key{c.GetKeyNameWithString(OrganizationNameKeyID, o.Name)}
	}
	uclog.Verbosef(context.Background(), "Organization %v has no name. Dropping secondary key", o.ID)
	return []cache.Key{}
}

// TTL returns the TTL for edge
func (o Organization) TTL(c cache.TTLProvider) time.Duration {
	return c.TTL(OrganizationTTL)
}
