package internal

import (
	"context"
	"maps"
	"sort"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"userclouds.com/authz"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/featureflags"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// TODO once we finish match the key prefix and finish testing client + server cache combo these TTLs should be shared between client and server in a UC only package
const (
	// serverObjTypeTTL specifies how long ObjectTypes remain in the cache by default.
	serverObjTypeTTL time.Duration = 60 * time.Minute
	// serverEdgeTypeTTL specifies how long EdgeTypes remain in the cache by default.
	serverEdgeTypeTTL time.Duration = 60 * time.Minute
	// serverObjTTL specifies how long Objects remain in the cache by default.
	serverObjTTL time.Duration = 60 * time.Minute
	// serverEdgeTTL specifies how long Edges remain in the cache by default. It is assumed that edges churn frequently so this number is set lower
	serverEdgeTTL time.Duration = 60 * time.Minute
	// serverExprWindow specifies expiration window for cache keys (added to the TTL), set to zero to expire immediately at TTL
	serverExprWindow time.Duration = 5 * time.Minute
	// validationInterval specifies how often the on machine edges cache should be validated against server state
	validationInterval time.Duration = 2 * time.Minute
	// AuthzHandlersInvalidations is the name of the cache invalidation channel that on machine edge handler use
	AuthzHandlersInvalidations = "authz_handlers_invalidations"
)

// PreSaveError is the error type returned when a pre-save hook fails, allowing for handlers to distinguish these errors from other storage errors
type PreSaveError struct {
	error
}

func (e PreSaveError) Unwrap() error {
	return e.error
}

// Storage manages access to authz data
type Storage struct {
	db        *ucdb.DB
	tenantID  uuid.UUID
	cm        *cache.Manager
	edgeCache *EdgeCacheMapRecord
}

var sharedCache cache.Provider
var sharedCacheOnce sync.Once
var sharedCacheConfig *cache.Config

// ErrInvalidUpsert is returned when an Upsert would violate consistency
// of the underlying object (i.e. changing fields that can't be changed).
var ErrInvalidUpsert = ucerr.New("some fields not allowed to change on upsert")

// NewStorage creates a Storage object backed by a DB
func NewStorage(ctx context.Context, tenantID uuid.UUID, db *ucdb.DB, cc *cache.Config) *Storage {
	s := &Storage{db: db, tenantID: tenantID}

	cm, err := GetCacheManager(ctx, cc, s.tenantID)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to create authz cache manager: %v", err)
	}

	s.cm = cm

	edgeCacheGlobalOnce.Do(func() {
		edgeCacheGlobal = &EdgeCacheMapRecord{CacheTenantRecords: make(map[uuid.UUID]*EdgeTenantRecord)}
	})

	s.edgeCache = edgeCacheGlobal
	return s
}

// NewStorageForTests creates a Storage object backed by a DB
func NewStorageForTests(ctx context.Context, tenantID uuid.UUID, db *ucdb.DB, cp *cache.InvalidationWrapper, edgesCache *EdgeCacheMapRecord) *Storage {
	s := &Storage{db: db, tenantID: tenantID}

	if cp != nil {
		np := authz.NewCacheNameProviderForTenant(s.tenantID)
		ttlP := authz.NewCacheTTLProvider(serverObjTypeTTL, serverEdgeTypeTTL, serverObjTTL, serverEdgeTTL, serverExprWindow)
		cm := cache.NewManager(cp, np, ttlP)

		s.cm = &cm
	}

	s.edgeCache = edgesCache

	if s.edgeCache == nil {
		edgeCacheGlobalOnce.Do(func() {
			edgeCacheGlobal = &EdgeCacheMapRecord{CacheTenantRecords: make(map[uuid.UUID]*EdgeTenantRecord)}
		})
		s.edgeCache = edgeCacheGlobal
	}

	return s
}

// GetCacheManager returns a cache manager for the storage layer (this method is not private to tooling and testing purposes)
func GetCacheManager(ctx context.Context, cc *cache.Config, tenantID uuid.UUID) (*cache.Manager, error) {
	if cc == nil || cc.RedisCacheConfig == nil {
		return nil, nil
	}

	sharedCacheOnce.Do(func() {
		var err error
		sharedCache, err = cache.InitializeInvalidatingCacheFromConfig(
			ctx,
			cc,
			cache.RegionalRedisCacheName,
			authz.CachePrefix,
			cache.InvalidationHandlersLocalPublish(AuthzHandlersInvalidations, []string{authz.EdgeCollectionKeyString}),
		)

		if err != nil {
			uclog.Errorf(ctx, "failed to create cache invalidation wrapper: %v", err)
			return
		}

		sharedCacheConfig = cc
	})

	if sharedCache == nil {
		return nil, nil
	}

	ttlP := authz.NewCacheTTLProvider(serverObjTypeTTL, serverEdgeTypeTTL, serverObjTTL, serverEdgeTTL, serverExprWindow)
	np := authz.NewCacheNameProviderForTenant(tenantID)
	cm := cache.NewManager(sharedCache, np, ttlP)

	return &cm, nil
}

//go:generate genorm --cache --followerreads --includeinsertonly --getbyname authz.ObjectType object_types tenantdb

// GetObjectTypeForName returns the definition of a single object type by its name.
func (s *Storage) GetObjectTypeForName(ctx context.Context, typeName string) (*authz.ObjectType, error) {
	var primaryKey cache.Key
	if s.cm != nil {
		primaryKey = s.cm.N.GetKeyNameWithString(authz.ObjectTypeNameKeyID, typeName)
	}
	return s.getObjectTypeByColumns(ctx, primaryKey, []string{"type_name"}, []any{typeName})
}

func (s *Storage) preDeleteObjectType(ctx context.Context, id uuid.UUID, wrappedDelete bool) error {
	// First get all edge types that use this object type and delete them,
	// which in turn will delete all edges of that type.
	// NOTE: this is not transactional, so we can get into a bad state;
	// probably want a reconciler to clean things up, OR use a transaction

	pager, err := authz.NewEdgeTypePaginatorFromOptions(
		pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return ucerr.Wrap(err)
	}

	for {
		edgeTypes, respFields, err := s.ListEdgeTypesPaginated(ctx, *pager)
		if err != nil {
			return ucerr.Wrap(err)
		}

		for _, edgeType := range edgeTypes {
			if edgeType.SourceObjectTypeID == id || edgeType.TargetObjectTypeID == id {
				if err := s.deleteInnerEdgeType(ctx, edgeType.ID, true); err != nil {
					return ucerr.Wrap(err)
				}
			}
		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	// Then delete all objects that use this object type.
	const deleteObjectQuery = `UPDATE objects SET deleted=NOW() WHERE type_id=$1 AND deleted='0001-01-01 00:00:00';`
	if _, err := s.db.ExecContext(ctx, "preDeleteObjectType", deleteObjectQuery, id); err != nil {
		return ucerr.Wrap(err)
	}

	if !wrappedDelete {
		// TODO move this post type delete
		if err := s.FlushCacheForObjectType(ctx, id); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

//go:generate genorm --cache --followerreads --includeinsertonly --getbyname authz.EdgeType edge_types tenantdb

// GetEdgeTypeForName returns the definition of a single edge type by its name.
func (s *Storage) GetEdgeTypeForName(ctx context.Context, typeName string) (*authz.EdgeType, error) {
	var primaryKey cache.Key
	if s.cm != nil {
		primaryKey = s.cm.N.GetKeyNameWithString(authz.EdgeTypeNameKeyID, typeName)
	}
	return s.getEdgeTypeByColumns(ctx, primaryKey, []string{"type_name"}, []any{typeName})
}

// Updating an existing edge type cannot change the source or target type IDs, because
// doing so would violate consistency of existing edges. Attempting to do so will
// result in no rows matching. Attempting to re-use an existing edge type name
// will result in a unique constraint violation.
// This is actually enforced by the generated ORM and "immutable" tags
func (s *Storage) preSaveEdgeType(ctx context.Context, edgeType *authz.EdgeType) error {
	// TODO: race condition? we can probably use the cache to get optimistic concurrency
	// Just ensure the type IDs are valid
	_, err := s.GetObjectType(ctx, edgeType.SourceObjectTypeID)
	if err != nil {
		return ucerr.Wrap(PreSaveError{ucerr.Friendlyf(err, "edge type %v source object type %v invalid", edgeType.ID, edgeType.SourceObjectTypeID)})
	}
	_, err = s.GetObjectType(ctx, edgeType.TargetObjectTypeID)
	if err != nil {
		return ucerr.Wrap(PreSaveError{ucerr.Friendlyf(err, "edge type %v target object type %v invalid", edgeType.ID, edgeType.TargetObjectTypeID)})
	}
	if edgeType.OrganizationID != uuid.Nil {
		if _, err := s.GetOrganization(ctx, edgeType.OrganizationID); err != nil {
			return ucerr.Wrap(PreSaveError{ucerr.Friendlyf(err, "edge type %v organization %v is invalid:", edgeType.ID, edgeType.OrganizationID)})
		}
	}

	return nil
}

// DeleteEdgeType soft-deletes the given edge type ID
func (s *Storage) preDeleteEdgeType(ctx context.Context, id uuid.UUID, wrappedDelete bool) error {
	// Delete all edges that use this edge type.
	// NOTE: this is not transactional, so we can get into a bad state;
	// probably want a reconciler to clean things up, OR use a transaction
	const deleteEdgeQuery = `UPDATE edges SET deleted=CLOCK_TIMESTAMP() WHERE edge_type_id=$1 AND deleted='0001-01-01 00:00:00';`
	if _, err := s.db.ExecContext(ctx, "preDeleteEdgeType", deleteEdgeQuery, id); err != nil {
		return ucerr.Wrap(err)
	}
	// TODO move this post type delete
	if !wrappedDelete {
		if err := s.FlushCacheForEdgeType(ctx, id); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}

//go:generate genorm --cache --includeinsertonly --getbyname --followerreads authz.Object objects tenantdb

// GetObjectForAlias retrieves the object with a given type and alias.
func (s *Storage) GetObjectForAlias(ctx context.Context, typeID uuid.UUID, alias string, orgID uuid.UUID) (*authz.Object, error) {
	var primaryKey cache.Key
	if s.cm != nil {
		primaryKey = s.cm.N.GetKeyName(authz.ObjAliasNameKeyID, []string{typeID.String(), alias, orgID.String()})
	}
	return s.getObjectByColumns(ctx, primaryKey, []string{"type_id", "alias", "organization_id"}, []any{typeID, alias, orgID})
}

// save inserts the given object if the ID does not exist, or allows renaming
// an existing object. However, in the latter case, the Type ID must not change because
// that would violate type consistency of existing edges.
// Attempting to update type ID will result in no rows matching, and attempting to re-use another alias
// would result in a unique constraint violation.
func (s *Storage) preSaveObject(ctx context.Context, object *authz.Object) error {
	// ID check prevents loop of SaveOrganization -> preSaveOrganization -> SaveObject -> preSaveObject -> GetOrganization
	if object.ID == object.OrganizationID {
		return nil
	}

	// TODO: there is a possible race condition here if the type ID is deleted
	// after validation. Should we validate the source/target types as part of
	// the SQL statement or just live with the race?
	if _, err := s.GetObjectType(ctx, object.TypeID); err != nil {
		return ucerr.Wrap(PreSaveError{ucerr.Friendlyf(err, "object %s is invalid, unable to load its object type (TypeID: %v)", object.ID, object.TypeID)})
	}

	if object.OrganizationID != uuid.Nil {
		if _, err := s.GetOrganization(ctx, object.OrganizationID); err != nil {
			return ucerr.Wrap(PreSaveError{ucerr.Friendlyf(err, "object %s is invalid, unable to load its organization (OrganizationID: %v)", object.ID, object.OrganizationID)})
		}
	}
	return nil
}

// NOTE: user objects cannot be deleted as their source of truth is in the user data store.
func (s *Storage) preDeleteObject(ctx context.Context, id uuid.UUID, wrappedDelete bool) error {
	// NOTE: this is not transactional, so we can get into a bad state;
	// probably want a reconciler to clean things up, OR use a transaction
	return ucerr.Wrap(s.DeleteEdgesFromObject(ctx, id))
}

//go:generate genorm --cache --includeinsertonly --followerreads --getbyname --cachepages authz.Edge edges tenantdb

// ListEdgesPaginatedAndUpdateCollectionCache populates Object_Edges collection as well as TargetObject_SourceObject collection
func (s *Storage) ListEdgesPaginatedAndUpdateCollectionCache(ctx context.Context,
	p pagination.Paginator,
	sourceObjectID uuid.UUID,
	targetObjectID uuid.UUID) ([]authz.Edge, *pagination.ResponseFields, error) {

	var obj authz.Object
	var sentinel cache.Sentinel
	var ckey cache.Key
	var err error

	cachable := false
	// TODO: make this more resilient to other callers
	if p.GetCursor() == "" {
		cachable = true
	}
	if s.cm != nil && cachable && (sourceObjectID != uuid.Nil || targetObjectID != uuid.Nil) {
		obj = authz.Object{BaseModel: ucdb.NewBaseWithID(sourceObjectID)}

		var edges *[]authz.Edge
		ckey = s.cm.N.GetKeyNameWithID(authz.ObjEdgesKeyID, sourceObjectID)
		edges, _, sentinel, _, err = cache.GetItemsArrayFromCache[authz.Edge](ctx, *s.cm, ckey, true)

		if err != nil {
			return nil, nil, ucerr.Wrap(err)
		}

		if edges != nil && len(*edges) <= p.GetLimit() {
			if targetObjectID != uuid.Nil {
				filteredEdges := make([]authz.Edge, 0)
				for _, edge := range *edges {
					if edge.TargetObjectID == targetObjectID {
						filteredEdges = append(filteredEdges, edge)
					}
				}
				edges = &filteredEdges
			}

			edges, respFields := pagination.ProcessResults(*edges, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())
			return edges, &respFields, nil
		}

		if targetObjectID != uuid.Nil {
			// Next try to read the edges between target object and source object. We could also try to read the edges from target object but in authz graph
			// it is rare to traverse in both directions so those collections would be less likely to be cached.
			ckey = s.cm.N.GetKeyName(authz.EdgesObjToObjID, []string{sourceObjectID.String(), targetObjectID.String()})

			edges, _, sentinel, _, err = cache.GetItemsArrayFromCache[authz.Edge](ctx, *s.cm, ckey, true)
			if err != nil {
				return nil, nil, ucerr.Wrap(err)
			}

			if edges != nil {
				edges, respFields := pagination.ProcessResults(*edges, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())
				return edges, &respFields, nil
			}
		}
		// Clear the lock in case of an error
		defer cache.ReleasePerItemCollectionLock(ctx, *s.cm, []cache.Key{ckey}, obj, sentinel)
	}

	edges, respFields, err := s.ListEdgesPaginated(ctx, p)

	if s.cm != nil && err == nil && cachable && !respFields.HasNext && !respFields.HasPrev {
		cache.SaveItemsToCollection(ctx, *s.cm, obj, edges, ckey, ckey, sentinel, false)
	}

	return edges, respFields, ucerr.Wrap(err)
}

// FindEdge gets the edge by its edge type ID and source & target object IDs.
func (s *Storage) FindEdge(ctx context.Context, edgeTypeID, sourceObjectID, targetObjectID uuid.UUID) (*authz.Edge, error) {
	var primaryKey cache.Key
	if s.cm != nil {
		var edge *authz.Edge
		var edges *[]authz.Edge
		var err error

		primaryKey = s.cm.N.GetKeyName(authz.EdgeFullKeyID, []string{sourceObjectID.String(), targetObjectID.String(), edgeTypeID.String()})

		// Try to fetch the individual edge first using secondary key  Source_Target_TypeID
		edge, _, _, err = cache.GetItemFromCache[authz.Edge](ctx, *s.cm, primaryKey, false)

		// Since we are not taking a lock we can ignore cache errors
		if err == nil && edge != nil {
			return edge, nil
		}
		// If the edges are in the cache by source->target - iterate over that set first
		edges, _, _, _, err = cache.GetItemsArrayFromCache[authz.Edge](ctx, *s.cm, s.cm.N.GetKeyName(authz.EdgesObjToObjID, []string{sourceObjectID.String(), targetObjectID.String()}), false)
		// Since we are not taking a lock we can ignore cache errors
		if err == nil && edges != nil {
			for _, edge := range *edges {
				if edge.EdgeTypeID == edgeTypeID {
					return &edge, nil
				}
			}
			// In theory we could return NotFound here but this is a rare enough case that it makes sense to try the server
		}
		// If there is a cache miss, try to get the edges from all in/out edges on the source object
		edges, _, _, _, err = cache.GetItemsArrayFromCache[authz.Edge](ctx, *s.cm, s.cm.N.GetKeyNameWithID(authz.ObjEdgesKeyID, sourceObjectID), false)
		// Since we are not taking a lock we can ignore cache errors
		if err == nil && edges != nil {
			for _, edge := range *edges {
				if edge.TargetObjectID == targetObjectID && edge.EdgeTypeID == edgeTypeID {
					return &edge, nil
				}
			}
			// In theory we could return NotFound here but this is a rare enough case that it makes sense to check the table
		}
		// We could also try all in/out edges from targetObjectID collection but that is less likely to be cached
	}

	// TODO We will repeat the call to get the primary edge key to figure out if we can use a stale read, this can be optimized
	return s.getEdgeByColumns(ctx, primaryKey, []string{"source_object_id", "target_object_id", "edge_type_id"}, []any{sourceObjectID, targetObjectID, edgeTypeID})
}

func (s *Storage) preSaveEdge(ctx context.Context, edge *authz.Edge) error {
	// TODO: storage layer caching?
	edgeType, err := s.GetEdgeType(ctx, edge.EdgeTypeID)
	if err != nil {
		return ucerr.Wrap(PreSaveError{err})
	}
	sourceObj, err := s.GetObject(ctx, edge.SourceObjectID)
	if err != nil {
		return ucerr.Wrap(PreSaveError{err})
	}
	targetObj, err := s.GetObject(ctx, edge.TargetObjectID)
	if err != nil {
		return ucerr.Wrap(PreSaveError{err})
	}
	if sourceObj.TypeID != edgeType.SourceObjectTypeID {
		return ucerr.Wrap(PreSaveError{ucerr.Friendlyf(nil, "source object (%s, type id: %s) doesn't match SourceObjectTypeID '%s'", sourceObj.ID, sourceObj.TypeID, edgeType.SourceObjectTypeID)})
	}
	if targetObj.TypeID != edgeType.TargetObjectTypeID {
		return ucerr.Wrap(PreSaveError{ucerr.Friendlyf(nil, "target object (%s, type id: %s) doesn't match TargetObjectTypeID '%s'", targetObj.ID, targetObj.TypeID, edgeType.TargetObjectTypeID)})
	}
	if edgeType.OrganizationID != uuid.Nil && (edgeType.OrganizationID != sourceObj.OrganizationID || edgeType.OrganizationID != targetObj.OrganizationID) {
		return ucerr.Wrap(PreSaveError{ucerr.Friendlyf(nil, "source and target objects must be in the same organization as edge type %s, org ID: %s", edgeType.TypeName, edgeType.OrganizationID)})
	}
	return nil
}

func (s *Storage) additionalSaveKeysForEdge(edge *authz.Edge) []cache.Key {
	if s.cm != nil {
		return []cache.Key{
			s.cm.N.GetKeyName(authz.EdgesObjToObjID, []string{edge.SourceObjectID.String(), edge.TargetObjectID.String()}),
			s.cm.N.GetKeyNameWithID(authz.ObjEdgesKeyID, edge.SourceObjectID),
			s.cm.N.GetKeyName(authz.EdgesObjToObjID, []string{edge.TargetObjectID.String(), edge.SourceObjectID.String()}),
			s.cm.N.GetKeyNameWithID(authz.ObjEdgesKeyID, edge.TargetObjectID),
		}
	}
	return []cache.Key{}
}

// DeleteEdgesFromObject removes the edges from/to given object
func (s *Storage) DeleteEdgesFromObject(ctx context.Context, objectID uuid.UUID) error {
	if s.cm != nil {
		obj := authz.Object{BaseModel: ucdb.NewBaseWithID(objectID)}

		// Taking a lock will delete all edges and paths that include this object as source or target. We intentionally tombstone the dependency key for the object to
		// prevent inflight reads of edge collection from object connected to this one from committing potentially stale results to the cache.
		sentinel, err := cache.TakePerItemCollectionLock(ctx, cache.Delete, *s.cm, nil, obj)

		if err != nil {
			return ucerr.Wrap(err)
		}
		defer cache.ReleasePerItemCollectionLock(ctx, *s.cm, nil, obj, sentinel)
	}

	const edgeQuery = `UPDATE edges SET deleted=NOW() WHERE (source_object_id=$1 OR target_object_id=$1) AND deleted='0001-01-01 00:00:00'; /* allow-multiple-target-use */`
	_, err := s.db.ExecContext(ctx, "DeleteEdgesFromObject", edgeQuery, objectID)
	if s.cm != nil {
		// We need to also reset the global edges collection and set the isModified flag
		edge := authz.Edge{BaseModel: ucdb.NewBase()}
		keys := []cache.Key{}

		// TODO find better packaging for this in api.go

		// Check if there is a default global collection for all items of this type and it is being used directly for follower reads
		if edge.GetGlobalCollectionKey(s.cm.N) != "" && edge.GetIsModifiedCollectionKey(s.cm.N) == "" {
			keys = append(keys, edge.GetGlobalCollectionKey(s.cm.N))
		}

		if edge.GetIsModifiedCollectionKey(s.cm.N) != "" {
			keys = append(keys, edge.GetIsModifiedCollectionKey(s.cm.N))
		}

		if err := s.cm.Provider.DeleteValue(ctx, keys, true, true /* force delete regardless of value */); err != nil {
			return ucerr.Wrap(err)
		}

		if edge.GetGlobalCollectionPagesKey(s.cm.N) != "" {
			if err := s.cm.Provider.ClearDependencies(ctx, edge.GetGlobalCollectionPagesKey(s.cm.N), false); err != nil {
				uclog.Errorf(ctx, "Error clearing pages of global collection %v from cache: %v", edge.GetGlobalCollectionPagesKey(s.cm.N), err)
			}
		}
	}
	return ucerr.Wrap(err)
}

func (s *Storage) preSaveOrganization(ctx context.Context, org *authz.Organization) error {
	obj := authz.Object{
		BaseModel:      ucdb.NewBaseWithID(org.ID),
		Alias:          &org.Name,
		TypeID:         authz.GroupObjectTypeID,
		OrganizationID: org.ID,
	}

	if err := s.SaveObject(ctx, &obj); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

func (s *Storage) preDeleteOrganization(ctx context.Context, id uuid.UUID, wrappedDelete bool) error {
	if err := s.DeleteObject(ctx, id); err != nil {
		return ucerr.Wrap(err)
	}

	// TODO move this post type delete
	if err := s.FlushCacheForOrganization(ctx, id); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

//go:generate genorm --cache --followerreads --includeinsertonly --getbyname authz.Organization organizations tenantdb

// GetOrganizationForName returns a single organization by its name.
func (s *Storage) GetOrganizationForName(ctx context.Context, orgName string) (*authz.Organization, error) {
	var primaryKey cache.Key
	if s.cm != nil {
		primaryKey = s.cm.N.GetKeyNameWithString(authz.OrganizationNameKeyID, orgName)
	}
	return s.getOrganizationByColumns(ctx, primaryKey, []string{"name"}, []any{orgName})
}

// CheckAttribute checks if the source object has the given attribute on the target object.
func (s *Storage) CheckAttribute(ctx context.Context, checkAttributeServiceName *string, sourceObjectID, targetObjectID uuid.UUID, attributeName string) (bool, []authz.AttributePathNode, error) {
	var ckey cache.Key
	var obj authz.Object
	sentinel := cache.NoLockSentinel
	if s.cm != nil {
		var path *[]authz.AttributePathNode
		var err error

		ckey = s.cm.N.GetKeyName(authz.AttributePathObjToObjID, []string{sourceObjectID.String(), targetObjectID.String(), attributeName})
		path, _, sentinel, _, err = cache.GetItemsArrayFromCache[authz.AttributePathNode](ctx, *s.cm, ckey, true)
		if err != nil {
			return false, nil, ucerr.Wrap(err)
		}

		if path != nil {
			return true, *path, nil
		}
		obj = authz.Object{BaseModel: ucdb.NewBaseWithID(sourceObjectID)}

		// Release the lock in case of error
		defer cache.ReleasePerItemCollectionLock(ctx, *s.cm, []cache.Key{ckey}, obj, sentinel)
	}

	var hasAttribute bool
	var path []authz.AttributePathNode
	var err error

	if featureflags.IsEnabledForTenant(ctx, featureflags.CheckAttributeViaService, s.tenantID) && checkAttributeServiceName != nil {
		hasAttribute, path, err = CheckAttributeViaService(ctx, *checkAttributeServiceName, s.tenantID, sourceObjectID, targetObjectID, attributeName)
	} else if !featureflags.IsEnabledForTenant(ctx, featureflags.OnMachineEdgesCacheCompareResults, s.tenantID) {
		hasAttribute, path, err = CheckAttributeBFS(ctx, s, s.tenantID, sourceObjectID, targetObjectID, attributeName, false)
	} else {
		if hasAttribute, path, err = CheckAttributeBFS(ctx, s, s.tenantID, sourceObjectID, targetObjectID, attributeName, true); err != nil {
			return false, nil, ucerr.Wrap(err)
		}

		hasAttributeC, pathC, err := CheckAttributeBFS(ctx, s, s.tenantID, sourceObjectID, targetObjectID, attributeName, false)

		if err != nil {
			uclog.Errorf(ctx, "Error checking attribute with cache %v for source %v and target %v: %v", attributeName, sourceObjectID, targetObjectID, err)
		} else if hasAttribute != hasAttributeC || !cmp.Equal(path, pathC, cmpopts.EquateEmpty()) {
			uclog.Errorf(ctx, "Error checking attribute with cache %v for source %v and target %v: mismatched results full %v inc %v full %v inc %v",
				attributeName, sourceObjectID, targetObjectID, hasAttribute, hasAttributeC, path, pathC)
		}
	}

	if s.cm != nil && hasAttribute && err == nil {
		// We can only cache positive responses, since we don't know when the path will be added to invalidate the negative result.
		cache.SaveItemsToCollection(ctx, *s.cm, obj, path, ckey, ckey, sentinel, false)
	}

	return hasAttribute, path, ucerr.Wrap(err)
}

// edgeBFSCacheGlobal is per process cache of edges for BFS traversal. It is stored as a map of per tenant
// ObjectID -> []edges (outgoing edges from that object). We don't use the normal layered cache approach for two reasons
// 1. We don't want to insert invalidation delay on each authz update/delete/create
// 2. We want to avoid re-marshalling, re-validating and reallocating memory for 15k edges
// This means that we use the cache similarly to follower reads in DB. If the MOD key is marked to indicate a change,
// we invalidate the in-process cache and re-populate whenever the MOD key is blank. To catch cases where we don't
// perform a read during the TTL of the MOD key we subscribe to invalidation events on that key as if we were in a different
// region. This a tradeoff between not taking an invalidation delay on each write and still having an in-process cache

// EdgeCacheRecord contains the edges cache for a particular point in time
type EdgeCacheRecord struct {
	// Map of ObjectID -> []edges out of that object
	EdgesMap map[uuid.UUID]map[uuid.UUID]*authz.Edge
	// True the cache is outdated and should be updated prior to use
	outdated bool
	// Latest validated time
	validatedTime time.Time
	// Latest updated time of any edge in the cache
	updatedTime time.Time
	// True the cache is in process of being updated updatedTime/edgesMap are not in valid state
	inProgress bool
	// Lock protecting the refresh/initialization of the edgesMap
	inProgressLock *sync.Mutex
}

// EdgeTenantRecord contains the cache records for a tenant
type EdgeTenantRecord struct {
	// Map of tombstone or cache.NoLockSentinel to EdgeCacheRecord
	CacheRecords           map[string]*EdgeCacheRecord
	invalidationRegistered bool
}

// EdgeCacheMapRecord contains the cache records for all tenants
type EdgeCacheMapRecord struct {
	// Map of per tenant cache records
	CacheTenantRecords map[uuid.UUID]*EdgeTenantRecord
	// We use a different cache provider for the invalidation subscription because we don't want the primary cache provider to run like on machine cache and
	// take the invalidation delay on each write, try to invalidate keys in region, etc. We never perform actual read/write operations on this cache provider
	invalidationCache     cache.Provider
	invalidationCacheOnce sync.Once
	// Lock to protect reads/writes to the map of per tenant cache records
	sync.RWMutex
	// Unique ID for this cache record used for debugging via logs
	id uuid.UUID
}

var edgeCacheGlobal *EdgeCacheMapRecord
var edgeCacheGlobalOnce sync.Once

// registerEdgeCollectionChangeHandler registers a handler for edge collection changes
func (s *Storage) registerEdgeCollectionChangeHandler(ctx context.Context) error {
	if s.edgeCache == nil {
		return ucerr.Errorf("sharedCache is not initialized")
	}
	s.edgeCache.invalidationCacheOnce.Do(func() {
		// Start the invalidation subscription thread to detect changes in edges collection for in-process edges cache
		s.edgeCache.invalidationCache = cache.RunInRegionLocalHandlersSubscriber(ctx, sharedCacheConfig, AuthzHandlersInvalidations)
		s.edgeCache.id = uuid.Must(uuid.NewV4())
	})

	mkey := s.cm.N.GetKeyNameStatic(authz.EdgeCollectionKeyID)
	// If there is IsModifiedCollectionKey use that instead of the GlobalCollectionKey
	if s.cm.N.GetKeyNameWithString(authz.IsModifiedCollectionKeyID, string(s.cm.N.GetKeyNameStatic(authz.EdgeCollectionKeyID))) != "" {
		mkey = s.cm.N.GetKeyNameWithString(authz.IsModifiedCollectionKeyID, string(s.cm.N.GetKeyNameStatic(authz.EdgeCollectionKeyID)))
	}

	// Make copies so that the handler doesn't keep the storage class alive
	tenantID := s.tenantID
	edgesCache := s.edgeCache

	handler := func(ctx context.Context, key cache.Key, flush bool) error {
		resetGlobalCacheForTenant(tenantID, edgesCache, flush)
		uclog.Verbosef(ctx, "getBFSGlobalCache: %v resetting edges global cache for %v", s.edgeCache.id, tenantID)
		return nil
	}

	return ucerr.Wrap(s.edgeCache.invalidationCache.RegisterInvalidationHandler(ctx, handler, mkey))
}

func resetGlobalCacheForTenant(tenantID uuid.UUID, edgesCache *EdgeCacheMapRecord, flush bool) {
	if edgesCache == nil {
		return
	}

	// Reset the map of per tenant edge caches
	edgesCache.Lock()
	defer edgesCache.Unlock()

	if _, ok := edgesCache.CacheTenantRecords[tenantID]; !ok {
		tenantRecord := &EdgeTenantRecord{CacheRecords: make(map[string]*EdgeCacheRecord), invalidationRegistered: false}
		edgesCache.CacheTenantRecords[tenantID] = tenantRecord
	}
	// if there is no entry for cache.NoLockSentinel that means that there is no in progress update
	if noConflictCache := edgesCache.CacheTenantRecords[tenantID].CacheRecords[string(cache.NoLockSentinel)]; noConflictCache != nil {
		noConflictCache.outdated = true
	}

	// if the cache was flushed we mark all records as outdated because edge type and object type deletions trigger a cache flush
	if flush {
		for _, r := range edgesCache.CacheTenantRecords[tenantID].CacheRecords {
			r.outdated = true
		}
	}
}

func (s *Storage) ensureRegistration(ctx context.Context) error {
	reg := false
	s.edgeCache.RLock()
	if r, ok := s.edgeCache.CacheTenantRecords[s.tenantID]; ok {
		reg = r.invalidationRegistered
	}
	s.edgeCache.RUnlock()

	if reg {
		return nil
	}

	s.edgeCache.Lock()
	defer s.edgeCache.Unlock()

	if _, ok := s.edgeCache.CacheTenantRecords[s.tenantID]; !ok {
		tenantRecord := &EdgeTenantRecord{CacheRecords: make(map[string]*EdgeCacheRecord), invalidationRegistered: false}
		s.edgeCache.CacheTenantRecords[s.tenantID] = tenantRecord
	}

	r := s.edgeCache.CacheTenantRecords[s.tenantID]

	if !r.invalidationRegistered {
		if err := s.registerEdgeCollectionChangeHandler(ctx); err != nil {
			return ucerr.Wrap(err)
		}
		r.invalidationRegistered = true
	}

	return nil
}

// GetBFSEdgeGlobalCache returns the global cache of edges for BFS traversal
func (s *Storage) GetBFSEdgeGlobalCache(ctx context.Context) (map[uuid.UUID]map[uuid.UUID]*authz.Edge, error) {
	if s.cm == nil || s.edgeCache == nil {
		return nil, nil
	}

	if featureflags.IsEnabledForTenant(ctx, featureflags.OnMachineEdgesCacheDisable, s.tenantID) {
		return nil, nil
	}

	if err := s.ensureRegistration(ctx); err != nil {
		return nil, ucerr.Wrap(err)
	}

	s.edgeCache.RLock()
	edgeCacheRecordsTenantMap := s.edgeCache.CacheTenantRecords[s.tenantID].CacheRecords
	s.edgeCache.RUnlock()
	if edgeCacheRecordsTenantMap == nil {
		return nil, ucerr.Errorf("cacheTenantRecords[%v] is nil. Expected to be initialized in ensureRegistration()", s.tenantID)
	}

	mkey := s.cm.N.GetKeyNameStatic(authz.EdgeCollectionKeyID)
	// If there is IsModifiedCollectionKey use that instead of the GlobalCollectionKey
	if s.cm.N.GetKeyNameWithString(authz.IsModifiedCollectionKeyID, string(s.cm.N.GetKeyNameStatic(authz.EdgeCollectionKeyID))) != "" {
		mkey = s.cm.N.GetKeyNameWithString(authz.IsModifiedCollectionKeyID, string(s.cm.N.GetKeyNameStatic(authz.EdgeCollectionKeyID)))
	}
	_, conflict, _, _, err := cache.GetItemsArrayFromCache[authz.Edge](ctx, *s.cm, mkey, false)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// Check if global cache for conflict==XX is already populated or if we need to populate it
	// We will only populate the cache once per conflict value
	var cachePopulated bool
	s.edgeCache.Lock()
	edgeCacheRecord := edgeCacheRecordsTenantMap[string(conflict)]
	var edgeCacheTenantRecordLatestBase *EdgeCacheRecord
	if edgeCacheRecord == nil {
		// TODO now that we don't hit the redis cache, we should reduce the load to one in progress record at a time
		edgeCacheRecord = &EdgeCacheRecord{EdgesMap: nil, inProgress: true, inProgressLock: &sync.Mutex{}, validatedTime: time.Now().UTC()}
		edgeCacheRecordsTenantMap[string(conflict)] = edgeCacheRecord

		// Try to find another entry to serve as a base if we don't have cache.NoLockSentinel entry in the map
		latestTime := time.Time{}
		for _, r := range edgeCacheRecordsTenantMap {
			if !r.inProgress && r.updatedTime.After(latestTime) {
				edgeCacheTenantRecordLatestBase = r
				latestTime = r.updatedTime
			}
		}

		edgeCacheRecord.inProgressLock.Lock() // Take the lock to cause other threads to wait for the population to complete before releasing the global lock
		defer edgeCacheRecord.inProgressLock.Unlock()
	} else {
		cachePopulated = true
	}
	s.edgeCache.Unlock()

	// If global cache for conflict==XX is populated or in progress of being populated, return it (possible after waiting for the population to complete)
	if cachePopulated {
		edgeCacheRecord.inProgressLock.Lock()
		defer edgeCacheRecord.inProgressLock.Unlock()

		s.edgeCache.RLock()
		// We read the value under lock here to prevent race detector from complaining but from correctness perspective we don't need it ie invalidation can come at this point
		cacheRecordOutdated := edgeCacheRecord.outdated
		s.edgeCache.RUnlock()

		if !cacheRecordOutdated { // lint: ignore
			uclog.Verbosef(ctx, "getBFSGlobalCache: %v returning global cache for conflict %s, %v edges time %v", s.edgeCache.id, conflict, getEdgeCountForCacheRecord(edgeCacheRecord), edgeCacheRecord.updatedTime)
			if conflict == cache.NoLockSentinel {
				// Check if we need to re-run validation, in which case kick off an async read of the server state
				if edgeCacheRecord.validatedTime.Add(validationInterval).Before(time.Now().UTC()) {
					edgeCacheRecord.validatedTime = time.Now().UTC()
					go func() {
						ErrorDetectionWorker(context.Background(), s)
					}()
				}
			}
			return edgeCacheRecord.EdgesMap, nil
		}
	}

	s.edgeCache.Lock()
	edgeCacheRecord.outdated = false
	edgeCacheRecord.inProgress = true // We are holding both global and in progress locks so we can safely set this flag
	s.edgeCache.Unlock()

	// If we hit an error we need to make sure that re-populate the edges map since we might have a partial map so flip outdated flag back to true
	// since deferred function in Go are called in last in first out order we will still be holding in progress lock when this function runs
	// The next caller or the next thread that was waiting for the in progress lock will re-populate the cache
	defer func() {
		if edgeCacheRecord.inProgress {
			s.edgeCache.Lock()
			edgeCacheRecord.inProgress = false
			edgeCacheRecord.outdated = true
			edgeCacheRecord.updatedTime = time.Time{} // Reset the updated time so that the entry is not used as valid baseline
			s.edgeCache.Unlock()
		}
	}()

	if !cachePopulated {
		// If MOD key is set to a tombstone check if we can start with old pre-update map value to avoid re-populating the whole map
		if edgeCacheTenantRecordLatestBase != nil && edgeCacheRecord.EdgesMap == nil && conflict != cache.NoLockSentinel {
			// copyCacheRecord will take edgeCacheTenantRecordLatestBase.inProgressLock
			copyCacheRecord(edgeCacheTenantRecordLatestBase, edgeCacheRecord)
		}
	}

	// Otherwise kick off the population of the global cache for this conflict value
	if err := readEdgesCacheFromServer(ctx, s, edgeCacheRecord, s.tenantID, string(conflict), false); err != nil {
		return nil, ucerr.Wrap(err)
	}

	// TODO temporarily always reload the cache on conflict = NoLockSentinel
	if conflict == cache.NoLockSentinel && featureflags.IsEnabledForTenant(ctx, featureflags.OnMachineEdgesCacheVerify, s.tenantID) {
		// Make a copy of the current cache record which was incrementally updated
		localSrcCacheRecordCopy := &EdgeCacheRecord{EdgesMap: nil, inProgressLock: &sync.Mutex{}}
		copyCacheRecordUnlocked(edgeCacheRecord, localSrcCacheRecordCopy)
		// do full reload from server
		if err := readEdgesCacheFromServer(ctx, s, edgeCacheRecord, s.tenantID, string(conflict), true); err != nil {
			return nil, ucerr.Wrap(err)
		}

		// Even if times are the same there is a small chance that the cache was updated between the time we did incremental load and full load
		if edgeCacheRecord.updatedTime == localSrcCacheRecordCopy.updatedTime {
			less := func(a, b *authz.Edge) bool { return a.Updated.Before(b.Updated) }
			if !cmp.Equal(edgeCacheRecord.EdgesMap, localSrcCacheRecordCopy.EdgesMap, cmpopts.SortSlices(less)) {
				uclog.Warningf(ctx, "getBFSGlobalCache: detected inconsistency in edges cache for %v. Diff '%v' Worker time %v Cache time %v \n",
					s.tenantID, cmp.Diff(edgeCacheRecord.EdgesMap, localSrcCacheRecordCopy.EdgesMap, cmpopts.SortSlices(less)), edgeCacheRecord.updatedTime, localSrcCacheRecordCopy.updatedTime)

			}
		} else {
			uclog.Verbosef(ctx, "getBFSGlobalCache: %v collection changed for %v between inc/full reload %v, time %v", s.edgeCache.id, s.tenantID, localSrcCacheRecordCopy.updatedTime, edgeCacheRecord.updatedTime)
		}
	}

	// Check if we need to clear caches calculated for conflict values if the tombstone has expired
	s.edgeCache.Lock()
	defer s.edgeCache.Unlock()
	edgeCacheRecord.inProgress = false // We haven't released the in-progress locks while holding the global lock, so we can safely reset this flag
	if conflict == cache.NoLockSentinel {
		s.edgeCache.CacheTenantRecords[s.tenantID].CacheRecords = make(map[string]*EdgeCacheRecord)
		s.edgeCache.CacheTenantRecords[s.tenantID].CacheRecords[string(conflict)] = edgeCacheRecord
	} else {
		for t, r := range edgeCacheRecordsTenantMap {
			// Trim older tombstone values
			if t != string(cache.NoLockSentinel) {
				if !r.inProgress && r.updatedTime.Add(300*time.Millisecond).Before(edgeCacheRecord.updatedTime) {
					uclog.Verbosef(ctx, "getBFSGlobalCache: %v deleting edges cache with %v source objects %v time %v", s.edgeCache.id, len(r.EdgesMap), t, r.updatedTime)
					delete(edgeCacheRecordsTenantMap, t)
				}
			}
		}
	}

	return edgeCacheRecord.EdgesMap, nil
}

func readEdgesCacheFromServer(ctx context.Context, s *Storage, edgeCacheTenantRecord *EdgeCacheRecord, tenantID uuid.UUID, conflict string, forceReload bool) error {
	fullLoad := forceReload
	if edgeCacheTenantRecord.EdgesMap == nil || edgeCacheTenantRecord.updatedTime.IsZero() {
		fullLoad = true // if the map is nil or updated time is zero, we need to load all edges
	}

	dirty := conflict != string(cache.NoLockSentinel)

	if fullLoad { // load all edges
		uclog.Verbosef(ctx, "getBFSGlobalCache: %v populating edges cache for %v conflict '%s'", s.edgeCache.id, tenantID, conflict)

		edgeCacheTenantRecord.EdgesMap = make(map[uuid.UUID]map[uuid.UUID]*authz.Edge) // reset the map since we are doing a full load
		edgeCacheTenantRecord.updatedTime = time.Time{}
		deletedTime := time.Time{}
		edgeCache := edgeCacheTenantRecord.EdgesMap

		// The invariant for the full load is that at the end there will be no edge that is not in the returned collection
		// for which updated < edgeCacheTenantRecord.updatedTime && deleted < edgeCacheTenantRecord.updatedTime
		// If that invariant is violated incremental reload will not pick up the edges so they will be missed
		for {
			edges, edgesRead, err := s.listEdgesForCachePaginated(ctx, edgeCacheTenantRecord.updatedTime, deletedTime, dirty)
			if err != nil {
				return ucerr.Wrap(err)
			}

			for i, edge := range edges {
				if edge.Deleted.IsZero() {
					if edgeCache[edge.SourceObjectID] == nil {
						edgeCache[edge.SourceObjectID] = make(map[uuid.UUID]*authz.Edge)
					}
					edgeCache[edge.SourceObjectID][edge.ID] = &edges[i]

					if edges[i].Updated.After(edgeCacheTenantRecord.updatedTime) {
						edgeCacheTenantRecord.updatedTime = edges[i].Updated
					}
				} else {
					// There were some edges deleted after the initial read so we need to process the tombstones
					if edgeCache[edge.SourceObjectID] != nil {
						delete(edgeCache[edge.SourceObjectID], edge.ID)
						if len(edgeCache[edge.SourceObjectID]) == 0 {
							delete(edgeCache, edge.SourceObjectID)
						}
					}
					if edges[i].Deleted.After(deletedTime) {
						deletedTime = edges[i].Deleted
					}
				}
			}

			if edgesRead < pagination.MaxLimit {
				break
			}
		}
		uclog.Verbosef(ctx, "getBFSGlobalCache: %v populated edges cache for %v with %v source objects %v edges %v time %v", s.edgeCache.id, tenantID, len(edgeCache), getEdgeCountForCacheRecord(edgeCacheTenantRecord), conflict, edgeCacheTenantRecord.updatedTime)
	} else { // do partial update from existing map state to latest updated time
		edgeCache := edgeCacheTenantRecord.EdgesMap

		uclog.Verbosef(ctx, "getBFSGlobalCache: %v updating edges cache for %v with %v source objects %v edges %v time %v", s.edgeCache.id, tenantID, len(edgeCache), getEdgeCountForCacheRecord(edgeCacheTenantRecord), conflict, edgeCacheTenantRecord.updatedTime)

		edges, err := s.listEdgesUpdated(ctx, edgeCacheTenantRecord.updatedTime, dirty)
		if err != nil {
			return ucerr.Wrap(err)
		}

		sort.Slice(edges, func(i, j int) bool {
			return edges[i].Deleted.IsZero() && (edges[i].Updated.Before(edges[j].Updated) || edges[i].Updated.Before(edges[j].Deleted)) || // not delete item
				!edges[i].Deleted.IsZero() && (edges[i].Deleted.Before(edges[j].Updated) || edges[i].Deleted.Before(edges[j].Deleted)) // deleted item
		})

		// We need to handle the case where we had edge with same id created, deleted and recreated with same updated time
		// In that case we will have a row for undeleted edge and 1 or more rows for deleted tombstones all with same ID
		edgesPreset := make(map[uuid.UUID]int, len(edges))
		edgesToSkip := make(map[int]bool)

		for i, edge := range edges {
			if _, ok := edgesPreset[edge.ID]; !ok {
				edgesPreset[edge.ID] = i
			} else {
				if edges[edgesPreset[edge.ID]].Deleted.IsZero() && edge.Deleted.IsZero() {
					uclog.Errorf(ctx, "getBFSGlobalCache: %v duplicate undeleted edges %v %v ", s.edgeCache.id, edge.BaseModel, edges[edgesPreset[edge.ID]].BaseModel)
				} else { // if we have duplicate entries for the same edge - just keep the latest one
					if edge.Deleted.IsZero() != edges[edgesPreset[edge.ID]].Deleted.IsZero() &&
						(edge.Updated.Equal(edges[edgesPreset[edge.ID]].Deleted) || edge.Deleted.Equal(edges[edgesPreset[edge.ID]].Updated)) {
						uclog.Errorf(ctx, "getBFSGlobalCache: %v deletion and creation occured at exactly same time %v %v ", s.edgeCache.id, edge.BaseModel, edges[edgesPreset[edge.ID]].BaseModel)
					}
					edgesToSkip[edgesPreset[edge.ID]] = true
					edgesPreset[edge.ID] = i
				}
			}
		}

		for i, edge := range edges {
			// Skip duplicate entries tombstones
			if edgesToSkip[i] {
				continue
			}
			if edge.Deleted.IsZero() {
				// Add the edge to the cache if this is not a tombstone
				if edgeCache[edge.SourceObjectID] == nil {
					edgeCache[edge.SourceObjectID] = make(map[uuid.UUID]*authz.Edge)
				}
				edgeCache[edge.SourceObjectID][edge.ID] = &edges[i]
			} else {
				// If we have a tombstone we need to remove the edge from the cache
				if edgeCache[edge.SourceObjectID] != nil {
					delete(edgeCache[edge.SourceObjectID], edge.ID)
					if len(edgeCache[edge.SourceObjectID]) == 0 {
						delete(edgeCache, edge.SourceObjectID)
					}
				}
			}

			if edges[i].Updated.After(edgeCacheTenantRecord.updatedTime) {
				edgeCacheTenantRecord.updatedTime = edges[i].Updated
			}
			if edges[i].Deleted.After(edgeCacheTenantRecord.updatedTime) {
				edgeCacheTenantRecord.updatedTime = edges[i].Deleted
			}
		}

		uclog.Verbosef(ctx, "getBFSGlobalCache: %v updated edges for %v cache with %v source objects %v edges %v time %v", s.edgeCache.id, tenantID, len(edgeCache), getEdgeCountForCacheRecord(edgeCacheTenantRecord),
			conflict, edgeCacheTenantRecord.updatedTime)
	}

	return nil
}

func (s *Storage) listEdgesUpdated(ctx context.Context, updatedTime time.Time, dirty bool) ([]authz.Edge, error) {
	// TODO add limit to this query and if exceeded fall back to full reload
	const q = "/* lint-sql-ok */ SELECT id, updated, deleted, edge_type_id, source_object_id, target_object_id, created FROM edges WHERE updated >= ($1 at time zone 'utc' - interval '10 milliseconds')::timestamp OR deleted >= ($1 at time zone 'utc' - interval '10 milliseconds')::timestamp;"

	var edges []authz.Edge
	if err := s.db.SelectContextWithDirty(ctx, "ListEdgesUpdated", &edges, q, dirty, updatedTime); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return edges, nil

}

func (s *Storage) listEdgesForCachePaginated(ctx context.Context, updatedTime time.Time, deletedTime time.Time, dirty bool) ([]authz.Edge, int, error) {
	const q = "/* lint-sql-ok */ SELECT id, updated, deleted, edge_type_id, source_object_id, target_object_id, created FROM edges WHERE updated >= $1 and deleted ='0001-01-01 00:00:00' ORDER by updated LIMIT $2;"

	// First get a set of undeleted edges ordered by updated time for pagination.MaxLimit edges with updated > updatedTime
	var edges []authz.Edge
	if err := s.db.SelectContextWithDirty(ctx, "ListEdgesForCachePaginated", &edges, q, dirty, updatedTime, pagination.MaxLimit); err != nil {
		return nil, 0, ucerr.Wrap(err)
	}

	edgesRead := len(edges)
	// Second pick up any deletes that potentially occurred during the previous calls. We could do this in a single query but that requires any extra index
	// to avoid full table scan
	if len(edges) > 0 {
		maxTime := edges[len(edges)-1].Updated
		if maxTime.After(deletedTime) {
			deletedTime = maxTime
		}
		const qd = "/* lint-sql-ok */ SELECT id, updated, deleted, edge_type_id, source_object_id, target_object_id, created FROM edges WHERE updated <= $1 AND deleted >= $2;"

		var edgesDeleted []authz.Edge
		if err := s.db.SelectContextWithDirty(ctx, "ListEdgesForCacheDeleted", &edgesDeleted, qd, dirty, maxTime, deletedTime); err != nil {
			return nil, 0, ucerr.Wrap(err)
		}
		edges = append(edges, edgesDeleted...)
	}

	return edges, edgesRead, nil

}

// copyCacheRecord assumes that the caller is holding the dst.inprogressLock or the single reference to dst
func copyCacheRecord(src *EdgeCacheRecord, dst *EdgeCacheRecord) {
	src.inProgressLock.Lock()
	defer src.inProgressLock.Unlock()
	copyCacheRecordUnlocked(src, dst)
}

// copyCacheRecordUnlocked assumes that the caller is holding the dst/src.inprogressLock or the single reference to dst/src
func copyCacheRecordUnlocked(src *EdgeCacheRecord, dst *EdgeCacheRecord) {
	dst.EdgesMap = make(map[uuid.UUID]map[uuid.UUID]*authz.Edge, len(src.EdgesMap))
	for k, v := range src.EdgesMap {
		dst.EdgesMap[k] = maps.Clone(v)
	}
	dst.updatedTime = src.updatedTime
}

func getEdgeCountForCacheRecord(cacheRecord *EdgeCacheRecord) int {
	count := 0
	for _, v := range cacheRecord.EdgesMap {
		count += len(v)
	}
	return count
}

// ErrorDetectionWorker is exported for testing purposes. It shouldn't be called directly.
func ErrorDetectionWorker(ctx context.Context, s *Storage) bool {
	errorsDetected := false

	// Make a local copy of the cache.NoLockSentinel cache records which are currently up to date (minimize the amound of time we need s.edgeCache.Lock)
	localNoConflictCacheRecords := make(map[uuid.UUID]*EdgeCacheRecord)
	localUpdatedTimes := make(map[uuid.UUID]time.Time) // keep a copy of the updated times to detect changes during the execution of the worker
	s.edgeCache.RLock()
	for tenantID, tenantRecord := range s.edgeCache.CacheTenantRecords {
		if tenantRecord.CacheRecords != nil && tenantRecord.CacheRecords[string(cache.NoLockSentinel)] != nil &&
			!tenantRecord.CacheRecords[string(cache.NoLockSentinel)].outdated &&
			!tenantRecord.CacheRecords[string(cache.NoLockSentinel)].inProgress &&
			// for now we run detectiomn on a single given tenant
			s.tenantID == tenantID {
			localNoConflictCacheRecords[tenantID] = tenantRecord.CacheRecords[string(cache.NoLockSentinel)]
			localUpdatedTimes[tenantID] = tenantRecord.CacheRecords[string(cache.NoLockSentinel)].updatedTime
		}
	}
	s.edgeCache.RUnlock()

	// For each update record, make a copy of it so we don't block incoming calls, retrieve the current state from the server,
	// compare the records to see if there are any differences, if there are differences and the current value is same as local copy -
	// swap new value into the map and log an error
	for tenantID, localSrcCacheRecord := range localNoConflictCacheRecords {
		// Make a copy of tenantRecord.cacheRecords[string(cache.NoLockSentinel)] (minimize time we are holding inprogressLock.Lock())
		localSrcCacheRecordCopy := &EdgeCacheRecord{EdgesMap: nil, inProgressLock: &sync.Mutex{}}
		if !localSrcCacheRecord.outdated && !localSrcCacheRecord.inProgress {
			// copyCacheRecord will lock localSrcCacheRecord
			copyCacheRecord(localSrcCacheRecord, localSrcCacheRecordCopy)
		} else {
			uclog.Verbosef(ctx, "errorDetectionWorker: skipping verification for %v, outdated %v inprogress %v updated %v", tenantID,
				localSrcCacheRecord.outdated, localSrcCacheRecord.inProgress, localUpdatedTimes[tenantID])
			continue
		}

		// Get current server state
		cmpCacheRecord := &EdgeCacheRecord{EdgesMap: nil, inProgressLock: &sync.Mutex{}}
		if err := readEdgesCacheFromServer(ctx, s, cmpCacheRecord, tenantID, string(cache.NoLockSentinel), true); err != nil {
			uclog.Errorf(ctx, "errorDetectionWorker: failed to get DB state for %v, error %v", tenantID, err)
			continue
		}

		// Compare the server state to the copy of the local state. We check for later updated time on the server side because the client may have seen later edges which have
		// since been deleted resulting in the latest undeleted edge being older than the latest edge seen by the client. This is not a problem as long as the edges are same in both
		// sets
		less := func(a, b *authz.Edge) bool { return a.Updated.Before(b.Updated) }
		if !cmp.Equal(cmpCacheRecord.EdgesMap, localSrcCacheRecordCopy.EdgesMap, cmpopts.SortSlices(less)) || cmpCacheRecord.updatedTime.After(localSrcCacheRecordCopy.updatedTime) {
			// If difference is detected and local state hasn't changed log an error and swap the new value into the map
			diffDetected := false
			s.edgeCache.RLock()
			// If local state hasn't changed swap the new value into the map
			if s.edgeCache.CacheTenantRecords[tenantID] != nil && // tenant record might have been deleted
				s.edgeCache.CacheTenantRecords[tenantID].CacheRecords[string(cache.NoLockSentinel)] == localSrcCacheRecord &&
				!localSrcCacheRecord.outdated && !localSrcCacheRecord.inProgress &&
				s.edgeCache.CacheTenantRecords[tenantID].CacheRecords[string(cache.NoLockSentinel)].updatedTime == localUpdatedTimes[tenantID] {
				s.edgeCache.CacheTenantRecords[tenantID].CacheRecords[string(cache.NoLockSentinel)] = cmpCacheRecord
				diffDetected = true
			}
			s.edgeCache.RUnlock()

			if diffDetected {
				uclog.Errorf(ctx, "errorDetectionWorker: detected inconsistency in edges cache for %v, swapping the cache. Diff '%v' Worker time %v Cache time %v",
					tenantID, cmp.Diff(cmpCacheRecord.EdgesMap, localSrcCacheRecordCopy.EdgesMap, cmpopts.SortSlices(less)), cmpCacheRecord.updatedTime, localSrcCacheRecordCopy.updatedTime)
				errorsDetected = true
			}
		}
	}
	return errorsDetected
}
