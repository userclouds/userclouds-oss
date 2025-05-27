package internal_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"

	"userclouds.com/authz"
	"userclouds.com/authz/internal"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/cache"
	cachehelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/featureflags"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/request"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucdb"
	dbMetrics "userclouds.com/infra/ucdb/metrics"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/tenantdb"
)

func initStorage(ctx context.Context, t *testing.T) *internal.Storage {
	tdb := testdb.New(t, migrate.NewTestSchema(tenantdb.Schema))
	return internal.NewStorage(ctx, uuid.Must(uuid.NewV4()), tdb, cachehelpers.NewCacheConfig())
}

func initInMemStorage(ctx context.Context, t *testing.T, tdb *ucdb.DB, cacheName string, tenantID uuid.UUID) (*internal.Storage, cache.Provider) {
	sharedCache := cachehelpers.NewInMemCache(ctx, t, cacheName, tenantID, cache.InvalidationTombstoneTTL)
	return internal.NewStorageForTests(ctx, tenantID, tdb, sharedCache, nil), sharedCache
}

func newObjectType(typeName string) authz.ObjectType {
	return authz.ObjectType{
		BaseModel: ucdb.NewBase(),
		TypeName:  typeName,
	}
}

func createObjectType(t *testing.T, ctx context.Context, s *internal.Storage, typeName string) authz.ObjectType {
	t.Helper()
	objType := newObjectType(typeName)
	err := s.SaveObjectType(ctx, &objType)
	assert.NoErr(t, err)
	return objType
}

func getObjectTypeID(t *testing.T, ctx context.Context, s *internal.Storage, typeName string) uuid.UUID {
	t.Helper()
	objType, err := s.GetObjectTypeForName(ctx, typeName)
	assert.NoErr(t, err)
	return objType.ID
}

func newEdgeType(t *testing.T, ctx context.Context, s *internal.Storage, typeName, sourceObjType, targetObjType string) authz.EdgeType {
	return authz.EdgeType{
		BaseModel:          ucdb.NewBase(),
		TypeName:           typeName,
		SourceObjectTypeID: getObjectTypeID(t, ctx, s, sourceObjType),
		TargetObjectTypeID: getObjectTypeID(t, ctx, s, targetObjType),
		Attributes:         []authz.Attribute{{Name: "read", Direct: true}},
	}
}

func createEdgeType(t *testing.T, ctx context.Context, s *internal.Storage, typeName, sourceObjType, targetObjType string) authz.EdgeType {
	t.Helper()
	edgeType := newEdgeType(t, ctx, s, typeName, sourceObjType, targetObjType)
	err := s.SaveEdgeType(ctx, &edgeType)
	assert.NoErr(t, err)
	return edgeType
}

func getEdgeTypeID(t *testing.T, ctx context.Context, s *internal.Storage, typeName string) uuid.UUID {
	t.Helper()
	edgeType, err := s.GetEdgeTypeForName(ctx, typeName)
	assert.NoErr(t, err)
	return edgeType.ID
}

func newObject(t *testing.T, ctx context.Context, s *internal.Storage, typeName, alias string) authz.Object {
	aliasP := &alias
	if alias == "" {
		aliasP = nil
	}
	return authz.Object{
		BaseModel: ucdb.NewBase(),
		Alias:     aliasP,
		TypeID:    getObjectTypeID(t, ctx, s, typeName),
	}
}

func createObject(t *testing.T, ctx context.Context, s *internal.Storage, typeName, alias string) authz.Object {
	t.Helper()
	obj := newObject(t, ctx, s, typeName, alias)
	err := s.SaveObject(ctx, &obj)
	assert.NoErr(t, err)
	return obj
}

func getObjectID(t *testing.T, ctx context.Context, s *internal.Storage, typeName, alias string) uuid.UUID {
	t.Helper()
	typeID := getObjectTypeID(t, ctx, s, typeName)
	obj, err := s.GetObjectForAlias(ctx, typeID, alias, uuid.Nil)
	assert.NoErr(t, err)
	return obj.ID
}

func newEdge(t *testing.T, ctx context.Context, s *internal.Storage, typeName, sourceObjName, targetObjName string) authz.Edge {
	t.Helper()
	return newEdgeWithID(t, ctx, s, uuid.Must(uuid.NewV4()), typeName, sourceObjName, targetObjName)
}

func newEdgeWithID(t *testing.T, ctx context.Context, s *internal.Storage, edgeID uuid.UUID, typeName, sourceObjName, targetObjName string) authz.Edge {
	t.Helper()
	edgeType, err := s.GetEdgeTypeForName(ctx, typeName)
	assert.NoErr(t, err)
	sourceObj, err := s.GetObjectForAlias(ctx, edgeType.SourceObjectTypeID, sourceObjName, uuid.Nil)
	assert.NoErr(t, err)
	_, err = s.GetObjectForAlias(ctx, edgeType.SourceObjectTypeID, sourceObjName, uuid.Must(uuid.NewV4()))
	assert.NotNil(t, err)
	targetObj, err := s.GetObjectForAlias(ctx, edgeType.TargetObjectTypeID, targetObjName, uuid.Nil)
	assert.NoErr(t, err)
	_, err = s.GetObjectForAlias(ctx, edgeType.TargetObjectTypeID, targetObjName, uuid.Must(uuid.NewV4()))
	assert.NotNil(t, err)
	return authz.Edge{
		BaseModel:      ucdb.NewBaseWithID(edgeID),
		EdgeTypeID:     edgeType.ID,
		SourceObjectID: sourceObj.ID,
		TargetObjectID: targetObj.ID,
	}
}

func createEdge(t *testing.T, ctx context.Context, s *internal.Storage, typeName, sourceObjName, targetObjName string) authz.Edge {
	t.Helper()
	edge := newEdge(t, ctx, s, typeName, sourceObjName, targetObjName)
	err := s.SaveEdge(ctx, &edge)
	assert.NoErr(t, err)
	return edge
}

func createEdgeIfNotExist(t *testing.T, ctx context.Context, s *internal.Storage, typeName, sourceObjName, targetObjName string) authz.Edge {
	t.Helper()
	edge := newEdge(t, ctx, s, typeName, sourceObjName, targetObjName)
	err := s.SaveEdge(ctx, &edge)
	assert.True(t, err == nil || ucdb.IsUniqueViolation(err))
	return edge
}

func createEdgeWithID(t *testing.T, ctx context.Context, s *internal.Storage, edgeID uuid.UUID, typeName, sourceObjName, targetObjName string) authz.Edge {
	t.Helper()
	edge := newEdgeWithID(t, ctx, s, edgeID, typeName, sourceObjName, targetObjName)
	err := s.SaveEdge(ctx, &edge)
	if err != nil {
		uclog.Verbosef(ctx, "Error: %v", err)
	}
	assert.NoErr(t, err)
	return edge
}

func equalEdgeCollections(t *testing.T, expected []authz.Edge, actual []authz.Edge) {
	t.Helper()
	assert.Equal(t, len(expected), len(actual))

	expMap := make(map[uuid.UUID]authz.Edge)
	for i := range expected {
		expMap[expected[i].ID] = expected[i]
	}
	for i := range expected {
		assert.True(t, cmp.Equal(expMap[actual[i].ID], actual[i]))
	}
}

func TestORMCacheInvalidation(t *testing.T) {
	// t.Parallel() not running in parallel for now to avoid conflict with featureflags.InvalidatingCache being turned off
	ctx := context.Background()
	cacheName := "AuthzTestCache" + uuid.Must(uuid.NewV4()).String()[0:8]
	tenantID := uuid.Must(uuid.NewV4())
	tdb := testdb.New(t, migrate.NewTestSchema(tenantdb.Schema))

	// Create two different storages, each with their own in memory cache.
	s1, _ := initInMemStorage(ctx, t, tdb, cacheName, tenantID)
	s2, _ := initInMemStorage(ctx, t, tdb, cacheName, tenantID)

	// Create an object type in storage 1, ensure it's visible in storage 2.
	objType := createObjectType(t, ctx, s1, "TestType")
	objType2, err := s2.GetObjectTypeForName(ctx, "TestType")
	assert.NoErr(t, err)
	assert.Equal(t, objType, *objType2)

	// Create object in storage 1, ensure it's visible in storage 2.
	obj := createObject(t, ctx, s1, "TestType", "TestObj")
	obj2, err := s2.GetObject(ctx, obj.ID)
	assert.NoErr(t, err)
	assert.Equal(t, obj, *obj2)
	obj3, err := s2.GetObjectForAlias(ctx, obj.TypeID, *obj.Alias, uuid.Nil)
	assert.NoErr(t, err)
	assert.Equal(t, obj, *obj3)

	// Delete the object in storage 1, ensure it's gone in storage 2.
	err = s1.DeleteObject(ctx, obj.ID)
	assert.NoErr(t, err)
	_, err = s2.GetObject(ctx, obj.ID)
	assert.NotNil(t, err)
	_, err = s2.GetObjectForAlias(ctx, obj.TypeID, *obj.Alias, uuid.Nil)
	assert.NotNil(t, err)

	// Create another object in storage 1, ensure it's visible in storage 2.
	obj = createObject(t, ctx, s1, "TestType", "TestObj2")
	obj2, err = s2.GetObject(ctx, obj.ID)
	assert.NoErr(t, err)
	assert.Equal(t, obj, *obj2)
	obj3, err = s2.GetObjectForAlias(ctx, obj.TypeID, *obj.Alias, uuid.Nil)
	assert.NoErr(t, err)
	assert.Equal(t, obj, *obj3)

	// Delete the object type in storage 1, ensure everything gone in storage 2 due to a flush.
	err = s1.DeleteObjectType(ctx, obj.TypeID)
	assert.NoErr(t, err)
	_, err = s2.GetObject(ctx, obj.ID)
	assert.NotNil(t, err)
	_, err = s2.GetObjectForAlias(ctx, obj.TypeID, *obj.Alias, uuid.Nil)
	assert.NotNil(t, err)

	// Create three objects and one edge
	objType = createObjectType(t, ctx, s1, "TestType")
	objType2, err = s2.GetObjectTypeForName(ctx, "TestType")
	assert.NoErr(t, err)
	assert.Equal(t, objType, *objType2)
	objT1 := createObject(t, ctx, s1, "TestType", "TestObjEdgeT1")
	obj2, err = s2.GetObject(ctx, objT1.ID)
	assert.NoErr(t, err)
	assert.Equal(t, objT1, *obj2)
	objT2 := createObject(t, ctx, s1, "TestType", "TestObjEdgeT2")
	obj2, err = s2.GetObject(ctx, objT2.ID)
	assert.NoErr(t, err)
	assert.Equal(t, objT2, *obj2)
	objT3 := createObject(t, ctx, s1, "TestType", "TestObjEdgeT3")
	obj2, err = s2.GetObject(ctx, objT3.ID)
	assert.NoErr(t, err)
	assert.Equal(t, objT3, *obj2)
	edgeType := createEdgeType(t, ctx, s1, "EdgeType1", "TestType", "TestType")
	assert.NoErr(t, err)
	edge1 := createEdge(t, ctx, s1, edgeType.TypeName, "TestObjEdgeT1", "TestObjEdgeT2")
	assert.NoErr(t, err)

	// Populate per object edge cache and obj2obj edge cache
	pager := getEdgePager(t, objT1.ID, objT2.ID)
	edges, _, err := s1.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objT1.ID, objT2.ID)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 1, assert.Must())
	assert.Equal(t, edges[0], edge1)
	edges, _, err = s2.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objT1.ID, objT2.ID)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 1, assert.Must())
	assert.Equal(t, edges[0], edge1)
	pager = getEdgePager(t, objT1.ID)
	edges, _, err = s1.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objT1.ID, uuid.Nil)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 1, assert.Must())
	assert.Equal(t, edges[0], edge1)
	edges, _, err = s2.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objT1.ID, uuid.Nil)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 1, assert.Must())
	assert.Equal(t, edges[0], edge1)
	pager = getEdgePager(t, objT1.ID, objT3.ID)
	edges, _, err = s1.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objT1.ID, objT3.ID)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 0, assert.Must())
	edges, _, err = s2.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objT1.ID, objT3.ID)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 0, assert.Must())
	pager = getEdgePager(t, objT3.ID)
	edges, _, err = s1.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objT3.ID, uuid.Nil)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 0, assert.Must())
	edges, _, err = s2.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objT3.ID, uuid.Nil)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 0, assert.Must())

	edge2 := createEdge(t, ctx, s1, edgeType.TypeName, "TestObjEdgeT1", "TestObjEdgeT3")
	assert.NoErr(t, err)

	expectedEdges := []authz.Edge{edge1, edge2}

	pager = getEdgePager(t, objT1.ID, objT2.ID)
	edges, _, err = s1.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objT1.ID, objT2.ID)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 1, assert.Must())
	assert.Equal(t, edges[0], edge1)
	edges, _, err = s2.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objT1.ID, objT2.ID)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 1, assert.Must())
	assert.Equal(t, edges[0], edge1)
	pager = getEdgePager(t, objT1.ID)
	edges, _, err = s1.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objT1.ID, uuid.Nil)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 2, assert.Must())
	equalEdgeCollections(t, expectedEdges, edges)
	edges, _, err = s2.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objT1.ID, uuid.Nil)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 2, assert.Must())
	equalEdgeCollections(t, expectedEdges, edges)
	pager = getEdgePager(t, objT1.ID, objT3.ID)
	edges, _, err = s1.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objT1.ID, objT3.ID)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 1, assert.Must())
	assert.Equal(t, edges[0], edge2)
	edges, _, err = s2.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objT1.ID, objT3.ID)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 1, assert.Must())
	assert.Equal(t, edges[0], edge2)
	pager = getEdgePager(t, objT3.ID)
	edges, _, err = s1.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objT3.ID, uuid.Nil)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 1, assert.Must())
	assert.Equal(t, edges[0], edge2)
	edges, _, err = s2.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objT3.ID, uuid.Nil)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 1, assert.Must())
	assert.Equal(t, edges[0], edge2)
}

func TestORMCacheNameCaching(t *testing.T) {
	// t.Parallel() not running in parallel for now to avoid conflict with featureflags.InvalidatingCache being turned off
	ctx := context.Background()
	cacheName := "AuthzTestCache" + uuid.Must(uuid.NewV4()).String()[0:8]
	tenantID := uuid.Must(uuid.NewV4())
	tdb := testdb.New(t, migrate.NewTestSchema(tenantdb.Schema))

	// Create two different storages, each with their own in memory cache.
	s1, cp1 := initInMemStorage(ctx, t, tdb, cacheName, tenantID)
	s2, cp2 := initInMemStorage(ctx, t, tdb, cacheName, tenantID)

	np := authz.NewCacheNameProviderForTenant(tenantID)
	ttlP := authz.NewCacheTTLProvider(time.Minute, time.Minute, time.Minute, time.Minute, time.Second)
	cm1 := cache.NewManager(cp1, np, ttlP)
	cm2 := cache.NewManager(cp2, np, ttlP)

	// Create an object type in storage 1, ensure it's visible in storage 2.
	objType := createObjectType(t, ctx, s1, "TestType")
	objType2, err := s2.GetObjectTypeForName(ctx, "TestType")
	assert.NoErr(t, err)
	assert.Equal(t, objType, *objType2)

	// ObjectType should be in the cache for s1 from the call to CreateObjectType
	cachehelpers.ValidateValueInCacheByID(ctx, t, cm1, authz.ObjectTypeKeyID, objType.ID, objType)

	// Create an edge type in storage 1, ensure it's visible in storage 2.
	edgeType := createEdgeType(t, ctx, s1, "TestEdgeType", "TestType", "TestType")
	edgeType2, err := s2.GetEdgeTypeForName(ctx, "TestEdgeType")
	assert.NoErr(t, err)
	assert.Equal(t, edgeType, *edgeType2)

	// EdgeType should be in the cache for s1 from the call to CreateEdgeType
	cachehelpers.ValidateValueInCacheByID(ctx, t, cm1, authz.EdgeTypeKeyID, edgeType.ID, edgeType)

	// Create object in storage 1, ensure it's visible in storage 2.
	obj := createObject(t, ctx, s1, "TestType", "TestObj")
	obj2, err := s2.GetObject(ctx, obj.ID)
	assert.NoErr(t, err)
	assert.Equal(t, obj, *obj2)
	obj3, err := s2.GetObjectForAlias(ctx, obj.TypeID, *obj.Alias, uuid.Nil)
	assert.NoErr(t, err)
	assert.Equal(t, obj, *obj3)

	// Object should be in the cache for s1 from the call to CreateObject
	cachehelpers.ValidateValueInCacheByID(ctx, t, cm1, authz.ObjectKeyID, obj.ID, obj)

	// At this point cache in s2  contains tombstones for objecttype, edgetype, and object (due to invalidation from create), so flush them first
	err = cp2.Flush(ctx, "", true)
	assert.NoErr(t, err)

	// Refetch everything by name to populate cache in s2
	objType2, err = s2.GetObjectTypeForName(ctx, "TestType")
	assert.NoErr(t, err)
	assert.Equal(t, objType, *objType2)
	edgeType2, err = s2.GetEdgeTypeForName(ctx, "TestEdgeType")
	assert.NoErr(t, err)
	assert.Equal(t, edgeType, *edgeType2)
	obj3, err = s2.GetObjectForAlias(ctx, obj.TypeID, *obj.Alias, uuid.Nil)
	assert.NoErr(t, err)
	assert.Equal(t, obj, *obj3)

	// Give time for async calls to finish
	time.Sleep(time.Millisecond * 100)

	// ObjectType should be in the cache for s2 from the call to GetObjectTypeForName
	cachehelpers.ValidateValueInCacheByID(ctx, t, cm2, authz.ObjectTypeKeyID, objType.ID, objType)

	// EdgeType should be in the cache for s1 from the call to CreateEdgeType
	cachehelpers.ValidateValueInCacheByID(ctx, t, cm2, authz.EdgeTypeKeyID, edgeType.ID, edgeType)

	// Object should be in the cache for s2 from the call to CreateObject
	cachehelpers.ValidateValueInCacheByID(ctx, t, cm2, authz.ObjectKeyID, obj.ID, obj)
}
func TestObjectType(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := initStorage(ctx, t)

	pager, err := authz.NewObjectTypePaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	assert.NoErr(t, err)

	objType := createObjectType(t, ctx, s, "TestType")

	objTypes, respFields, err := s.ListObjectTypesPaginated(ctx, *pager)
	assert.NoErr(t, err)
	assert.False(t, respFields.HasNext)
	assert.False(t, respFields.HasPrev)
	assert.Equal(t, len(objTypes), 1, assert.Must())
	assert.Equal(t, objTypes[0], objType)

	// Create a second type, make sure it doesn't
	// overwrite the first one or something dumb :)
	objType2 := createObjectType(t, ctx, s, "TestType2")
	objTypes, _, err = s.ListObjectTypesPaginated(ctx, *pager)
	assert.NoErr(t, err)
	assert.Equal(t, len(objTypes), 2)
	if cmp.Equal(objTypes[0], objType) {
		assert.Equal(t, objTypes[1], objType2)
	} else {
		assert.Equal(t, objTypes[0], objType2)
		assert.Equal(t, objTypes[1], objType)
	}
}

func TestObjectTypeNames(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := initStorage(ctx, t)

	objType := createObjectType(t, ctx, s, "TestType")
	objType2 := createObjectType(t, ctx, s, "TestType2")

	// Rename first object type & re-save it. Ensure ID is stable,
	// but we get a new type name.
	origID := objType.ID
	assert.Equal(t, getObjectTypeID(t, ctx, s, "TestType"), objType.ID)
	objType.TypeName = "NewTypeName"
	err := s.SaveObjectType(ctx, &objType)
	assert.NoErr(t, err)
	assert.Equal(t, objType.ID, origID)
	assert.Equal(t, getObjectTypeID(t, ctx, s, "NewTypeName"), objType.ID)

	// Ensure re-saving the object type did not create a 3rd type.
	pager, err := authz.NewObjectTypePaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	assert.NoErr(t, err)
	objTypes, _, err := s.ListObjectTypesPaginated(ctx, *pager)
	assert.NoErr(t, err)
	assert.Equal(t, len(objTypes), 2)

	// Try to rename first type over the name of the second type
	objType.TypeName = objType2.TypeName
	err = s.SaveObjectType(ctx, &objType)
	assert.NotNil(t, err, assert.Must())
	assert.True(t, ucdb.IsUniqueViolation(err))

	// Try to create a conflcting name but with a new ID
	objType3 := newObjectType("TestType2")
	err = s.SaveObjectType(ctx, &objType3)
	assert.NotNil(t, err, assert.Must())
	assert.True(t, ucdb.IsUniqueViolation(err))
}

func TestEdgeType(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := initStorage(ctx, t)

	createObjectType(t, ctx, s, "TestType1")
	createObjectType(t, ctx, s, "TestType2")
	createEdgeType(t, ctx, s, "EdgeType1", "TestType1", "TestType2")
	createEdgeType(t, ctx, s, "EdgeType2", "TestType2", "TestType1")
	createEdgeType(t, ctx, s, "EdgeType3", "TestType1", "TestType1")

	// Ensure the right count
	pager, err := authz.NewEdgeTypePaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	assert.NoErr(t, err)
	edgeTypes, respFields, err := s.ListEdgeTypesPaginated(ctx, *pager)
	assert.NoErr(t, err)
	assert.False(t, respFields.HasNext)
	assert.False(t, respFields.HasPrev)
	assert.Equal(t, len(edgeTypes), 3)

	// Don't allow invalid type IDs
	badSourceType := newEdgeType(t, ctx, s, "EdgeType4", "TestType1", "TestType1")
	badSourceType.SourceObjectTypeID = uuid.Must(uuid.NewV4())
	err = s.SaveEdgeType(ctx, &badSourceType)
	assert.NotNil(t, err)
	badTargetType := newEdgeType(t, ctx, s, "EdgeType5", "TestType1", "TestType1")
	badTargetType.TargetObjectTypeID = uuid.Must(uuid.NewV4())
	err = s.SaveEdgeType(ctx, &badTargetType)
	assert.NotNil(t, err)
}

func TestEdgeTypeNames(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := initStorage(ctx, t)

	createObjectType(t, ctx, s, "TestType1")
	createObjectType(t, ctx, s, "TestType2")
	edgeType1 := createEdgeType(t, ctx, s, "EdgeType1", "TestType1", "TestType2")

	// Not ok to create new type with existing name but new ID
	conflictEdgeType := newEdgeType(t, ctx, s, "EdgeType1", "TestType1", "TestType2")
	assert.NotEqual(t, edgeType1.ID, conflictEdgeType.ID)
	err := s.SaveEdgeType(ctx, &conflictEdgeType)
	assert.NotNil(t, err, assert.Must())
	assert.True(t, ucdb.IsUniqueViolation(err))

	// Rename first edge type & re-save it. Ensure ID is stable,
	// but we get a new type name.
	origID := edgeType1.ID
	edgeType1.TypeName = "EdgeType1_renamed"
	err = s.SaveEdgeType(ctx, &edgeType1)
	assert.NoErr(t, err)
	assert.Equal(t, origID, edgeType1.ID)
	assert.Equal(t, getEdgeTypeID(t, ctx, s, "EdgeType1_renamed"), edgeType1.ID)

	// Not ok to change types from TestType1 -> TestType2
	edgeType1.SourceObjectTypeID = getObjectTypeID(t, ctx, s, "TestType2")
	err = s.SaveEdgeType(ctx, &edgeType1)
	assert.NotNil(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestObject(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := initStorage(ctx, t)

	pager, err := authz.NewObjectPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	assert.NoErr(t, err)

	createObjectType(t, ctx, s, "TestType")
	obj := createObject(t, ctx, s, "TestType", "TestObj")

	objs, respFields, err := s.ListObjectsPaginated(ctx, *pager)
	assert.NoErr(t, err)
	assert.False(t, respFields.HasNext)
	assert.False(t, respFields.HasPrev)
	assert.Equal(t, len(objs), 1, assert.Must())
	assert.Equal(t, objs[0], obj)

	objCopy, err := s.GetObject(ctx, obj.ID)
	assert.NoErr(t, err)
	assert.Equal(t, *objCopy, obj)

	err = s.DeleteObject(ctx, obj.ID)
	assert.NoErr(t, err)
	objs, _, err = s.ListObjectsPaginated(ctx, *pager)
	assert.NoErr(t, err)
	assert.Equal(t, len(objs), 0)

	// Make sure invalid type IDs are not allowed
	invalidTypeObj := newObject(t, ctx, s, "TestType", "BadTypeObj")
	invalidTypeObj.TypeID = uuid.Must(uuid.NewV4())
	err = s.SaveObject(ctx, &invalidTypeObj)
	assert.NotNil(t, err)
}

func TestObjectNames(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := initStorage(ctx, t)

	createObjectType(t, ctx, s, "TestType1")
	createObjectType(t, ctx, s, "TestType2")
	obj := createObject(t, ctx, s, "TestType1", "TestObj")

	// Not ok to create new object with existing alias but new ID
	conflictObj := newObject(t, ctx, s, "TestType1", "TestObj")
	assert.NotEqual(t, obj.ID, conflictObj.ID)
	err := s.SaveObject(ctx, &conflictObj)
	assert.NotNil(t, err, assert.Must())
	assert.True(t, ucdb.IsUniqueViolation(err))

	// But ok to create object with same alias and different type.
	conflictObj = newObject(t, ctx, s, "TestType2", "TestObj")
	err = s.SaveObject(ctx, &conflictObj)
	assert.NoErr(t, err)
	assert.NotEqual(t, obj.ID, conflictObj.ID)

	// Rename first object & re-save it. Ensure ID is stable,
	// but we get a new alias.
	origID := obj.ID
	alias := "TestObj_renamed"
	obj.Alias = &alias
	err = s.SaveObject(ctx, &obj)
	assert.NoErr(t, err)
	assert.Equal(t, origID, obj.ID)
	assert.Equal(t, getObjectID(t, ctx, s, "TestType1", "TestObj_renamed"), obj.ID)

	// Not ok to change types from TestType1 -> TestType2
	obj.TypeID = getObjectTypeID(t, ctx, s, "TestType2")
	err = s.SaveObject(ctx, &obj)
	assert.NotNil(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestEdges(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := initStorage(ctx, t)

	createObjectType(t, ctx, s, "TestType1")
	createObjectType(t, ctx, s, "TestType2")
	obj1 := createObject(t, ctx, s, "TestType1", "TestObj1")
	obj2 := createObject(t, ctx, s, "TestType2", "TestObj2")
	createEdgeType(t, ctx, s, "EdgeType1", "TestType1", "TestType2")
	createEdgeType(t, ctx, s, "EdgeType2", "TestType2", "TestType1")
	createEdgeType(t, ctx, s, "EdgeType3", "TestType1", "TestType1")

	edge1 := createEdge(t, ctx, s, "EdgeType1", "TestObj1", "TestObj2")
	edge2 := createEdge(t, ctx, s, "EdgeType2", "TestObj2", "TestObj1")
	createEdge(t, ctx, s, "EdgeType3", "TestObj1", "TestObj1")
	pager := getEdgePager(t, obj1.ID)
	edges, _, err := s.ListEdgesPaginated(ctx, *pager)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 3)
	pager = getEdgePager(t, obj2.ID)
	edges, _, err = s.ListEdgesPaginated(ctx, *pager)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 2, assert.Must())
	if getEdgeTypeID(t, ctx, s, "EdgeType1") == edges[0].EdgeTypeID {
		assert.Equal(t, edges[0].ID, edge1.ID)
		assert.Equal(t, edges[1].ID, edge2.ID)
	} else {
		assert.Equal(t, edges[1].ID, edge1.ID)
		assert.Equal(t, edges[0].ID, edge2.ID)
	}
}

func getEdgeFilter(sourceObjectID uuid.UUID, targetObjectID uuid.UUID) pagination.Option {
	if targetObjectID.IsNil() {
		return pagination.Filter(
			fmt.Sprintf(
				"(('source_object_id',EQ,'%v'),OR,('target_object_id',EQ,'%v'))",
				sourceObjectID,
				sourceObjectID,
			),
		)
	}

	return pagination.Filter(
		fmt.Sprintf(
			"(('source_object_id',EQ,'%v'),AND,('target_object_id',EQ,'%v'))",
			sourceObjectID,
			targetObjectID,
		),
	)
}

func getEdgePager(
	t *testing.T,
	objectIDs ...uuid.UUID,
) *pagination.Paginator {
	t.Helper()
	switch len(objectIDs) {
	case 1:
		objectIDs = append(objectIDs, uuid.Nil)
	case 2:
	default:
		assert.Fail(t, "wrong number of object IDs: %d", len(objectIDs))
	}
	pager, err := authz.NewEdgePaginatorFromOptions(getEdgeFilter(objectIDs[0], objectIDs[1]))
	assert.NoErr(t, err)
	return pager
}

func TestEdgesBetweenObjects(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := initStorage(ctx, t)

	createObjectType(t, ctx, s, "TestType1")
	obj1 := createObject(t, ctx, s, "TestType1", "TestObj1")
	obj2 := createObject(t, ctx, s, "TestType1", "TestObj2")
	createEdgeType(t, ctx, s, "EdgeType1", "TestType1", "TestType1")
	createEdgeType(t, ctx, s, "EdgeType2", "TestType1", "TestType1")
	edge1 := createEdge(t, ctx, s, "EdgeType1", "TestObj1", "TestObj2")
	edge2 := createEdge(t, ctx, s, "EdgeType1", "TestObj2", "TestObj1")
	edge3 := createEdge(t, ctx, s, "EdgeType2", "TestObj1", "TestObj2")
	edge4 := createEdge(t, ctx, s, "EdgeType1", "TestObj2", "TestObj2")

	pager := getEdgePager(t, obj1.ID, obj2.ID)
	edges, _, err := s.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, obj1.ID, obj2.ID)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 2, assert.Must())
	if getEdgeTypeID(t, ctx, s, "EdgeType1") == edges[0].EdgeTypeID {
		assert.Equal(t, edges[0].ID, edge1.ID)
		assert.Equal(t, edges[1].ID, edge3.ID)
	} else {
		assert.Equal(t, edges[1].ID, edge1.ID)
		assert.Equal(t, edges[0].ID, edge3.ID)
	}

	pager = getEdgePager(t, obj2.ID, obj1.ID)
	edges, _, err = s.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, obj2.ID, obj1.ID)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 1, assert.Must())
	assert.Equal(t, edges[0].ID, edge2.ID)

	pager = getEdgePager(t, obj1.ID, obj1.ID)
	edges, _, err = s.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, obj1.ID, obj1.ID)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 0)

	pager = getEdgePager(t, obj2.ID, obj2.ID)
	edges, _, err = s.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, obj2.ID, obj2.ID)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 1, assert.Must())
	assert.Equal(t, edges[0].ID, edge4.ID)
}

func TestBadEdges(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := initStorage(ctx, t)

	createObjectType(t, ctx, s, "TestType1")
	createObjectType(t, ctx, s, "WrongType")
	testObj1 := createObject(t, ctx, s, "TestType1", "TestObj1")
	createObject(t, ctx, s, "TestType1", "TestObj2")
	wrongTypeObj := createObject(t, ctx, s, "WrongType", "WrongTypeObj")
	edgeType1 := createEdgeType(t, ctx, s, "EdgeType1", "TestType1", "TestType1")
	createEdgeType(t, ctx, s, "EdgeType2", "TestType1", "TestType1")

	// Invalid edge type IDs are not allowed
	invalidTypeEdge := newEdge(t, ctx, s, "EdgeType1", "TestObj1", "TestObj1")
	invalidTypeEdge.EdgeTypeID = uuid.Must(uuid.NewV4())
	err := s.SaveEdge(ctx, &invalidTypeEdge)
	assert.NotNil(t, err)

	// Don't allow edge with incorrect source or target types
	invalidSourceTypeEdge := authz.Edge{
		BaseModel:      ucdb.NewBase(),
		EdgeTypeID:     edgeType1.ID,
		SourceObjectID: wrongTypeObj.ID,
		TargetObjectID: testObj1.ID,
	}
	err = s.SaveEdge(ctx, &invalidSourceTypeEdge)
	assert.NotNil(t, err)
	invalidTargetTypeEdge := authz.Edge{
		BaseModel:      ucdb.NewBase(),
		EdgeTypeID:     edgeType1.ID,
		SourceObjectID: testObj1.ID,
		TargetObjectID: wrongTypeObj.ID,
	}
	err = s.SaveEdge(ctx, &invalidTargetTypeEdge)
	assert.NotNil(t, err)

	// Don't allow edges with undefined objects
	newEdge(t, ctx, s, "EdgeType1", "TestObj1", "TestObj1")
	err = s.SaveEdge(ctx, &invalidSourceTypeEdge)
	assert.NotNil(t, err)
	invalidTargetObjEdge := newEdge(t, ctx, s, "EdgeType1", "TestObj1", "TestObj1")
	invalidTargetObjEdge.TargetObjectID = uuid.Must(uuid.NewV4())
	err = s.SaveEdge(ctx, &invalidTargetObjEdge)
	assert.NotNil(t, err)

	// Don't allow duplicate edges (same type, source, target)
	createEdge(t, ctx, s, "EdgeType1", "TestObj1", "TestObj2")
	conflictEdge := newEdge(t, ctx, s, "EdgeType1", "TestObj1", "TestObj2")
	err = s.SaveEdge(ctx, &conflictEdge)
	assert.NotNil(t, err)
	assert.True(t, ucdb.IsUniqueViolation(err))

	// But allow edges with source & target flipped,
	// or with different edge type.
	createEdge(t, ctx, s, "EdgeType1", "TestObj2", "TestObj1")
	createEdge(t, ctx, s, "EdgeType2", "TestObj1", "TestObj2")
}

func TestDeleteEdges(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := initStorage(ctx, t)

	createObjectType(t, ctx, s, "TestType1")
	createEdgeType(t, ctx, s, "EdgeType1", "TestType1", "TestType1")

	obj1 := createObject(t, ctx, s, "TestType1", "TestObj1")
	obj2 := createObject(t, ctx, s, "TestType1", "TestObj2")
	edge := createEdge(t, ctx, s, "EdgeType1", "TestObj1", "TestObj2")

	// There should be 1 edge involving obj2
	pager := getEdgePager(t, obj2.ID, uuid.Nil)
	edges, _, err := s.ListEdgesPaginated(ctx, *pager)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 1)
	assert.Equal(t, edges[0].ID, edge.ID)

	// Deleting obj1 should delete the edge it shares with obj2
	err = s.DeleteObject(ctx, obj1.ID)
	assert.NoErr(t, err)
	pager = getEdgePager(t, obj2.ID, uuid.Nil)
	edges, _, err = s.ListEdgesPaginated(ctx, *pager)
	assert.NoErr(t, err)
	assert.Equal(t, len(edges), 0)
}

func TestSoftDeleteUniques(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := initStorage(ctx, t)

	ot := createObjectType(t, ctx, s, "ot")
	et := createEdgeType(t, ctx, s, "et", "ot", "ot")
	o := createObject(t, ctx, s, "ot", "o")
	e := createEdge(t, ctx, s, "et", "o", "o")

	// test each object delete-create in sequence to ensure we don't
	// just tear down IDs and only test the type cases (since those unique
	// constraints are string based, but edge-type is ID-based)
	assert.IsNil(t, s.DeleteEdge(ctx, e.ID))
	e2 := createEdge(t, ctx, s, "et", "o", "o")
	assert.NotEqual(t, e.ID, e2.ID)
	assert.IsNil(t, s.DeleteEdge(ctx, e2.ID))

	assert.IsNil(t, s.DeleteObject(ctx, o.ID))
	o2 := createObject(t, ctx, s, "ot", "o")
	assert.NotEqual(t, o.ID, o2.ID)
	assert.IsNil(t, s.DeleteObject(ctx, o2.ID))

	assert.IsNil(t, s.DeleteEdgeType(ctx, et.ID))
	et2 := createEdgeType(t, ctx, s, "et", "ot", "ot")
	assert.NotEqual(t, et.ID, et2.ID)
	assert.IsNil(t, s.DeleteEdgeType(ctx, et2.ID))

	assert.IsNil(t, s.DeleteObjectType(ctx, ot.ID))

	createObjectType(t, ctx, s, "ot")
	createEdgeType(t, ctx, s, "et", "ot", "ot")
	createObject(t, ctx, s, "ot", "o")
	createEdge(t, ctx, s, "et", "o", "o")
}

func countObjects(ctx context.Context, t *testing.T, s *internal.Storage) int {
	t.Helper()
	pager, err := authz.NewObjectPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	assert.NoErr(t, err)
	objs, _, err := s.ListObjectsPaginated(ctx, *pager)
	assert.NoErr(t, err)
	return len(objs)
}

func countObjectTypes(ctx context.Context, t *testing.T, s *internal.Storage) int {
	t.Helper()
	pager, err := authz.NewObjectTypePaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	assert.NoErr(t, err)
	objTypes, _, err := s.ListObjectTypesPaginated(ctx, *pager)
	assert.NoErr(t, err)
	return len(objTypes)
}

func countEdges(ctx context.Context, t *testing.T, s *internal.Storage) int {
	t.Helper()
	pager, err := authz.NewEdgePaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	assert.NoErr(t, err)
	edges, _, err := s.ListEdgesPaginated(ctx, *pager)
	assert.NoErr(t, err)
	return len(edges)
}

func countEdgeTypes(ctx context.Context, t *testing.T, s *internal.Storage) int {
	t.Helper()
	pager, err := authz.NewEdgeTypePaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	assert.NoErr(t, err)
	edgeTypes, _, err := s.ListEdgeTypesPaginated(ctx, *pager)
	assert.NoErr(t, err)
	return len(edgeTypes)
}

func TestCascadeDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := initStorage(ctx, t)

	// Create 2 object types and all combinations of edge types (plus an extra one).
	ot1 := createObjectType(t, ctx, s, "ot1")
	ot2 := createObjectType(t, ctx, s, "ot2")
	_ = createEdgeType(t, ctx, s, "et1", "ot1", "ot1")
	et2 := createEdgeType(t, ctx, s, "et2", "ot2", "ot2")
	_ = createEdgeType(t, ctx, s, "et3", "ot1", "ot2")
	_ = createEdgeType(t, ctx, s, "et4", "ot2", "ot1")
	et5 := createEdgeType(t, ctx, s, "et5", "ot2", "ot2")

	// We had a bug in this test masked by the fact # edges == # edge types and # objects == # object types,
	// so create dummy types to ensure counts are different
	_ = createObjectType(t, ctx, s, "unused")
	_ = createEdgeType(t, ctx, s, "unused", "ot2", "ot2")

	// Create 1 instance of each object & edge type.
	_ = createObject(t, ctx, s, "ot1", "o1")
	o2 := createObject(t, ctx, s, "ot2", "o2")
	_ = createEdge(t, ctx, s, "et1", "o1", "o1")
	e2 := createEdge(t, ctx, s, "et2", "o2", "o2")
	_ = createEdge(t, ctx, s, "et3", "o1", "o2")
	_ = createEdge(t, ctx, s, "et4", "o2", "o1")
	e5 := createEdge(t, ctx, s, "et5", "o2", "o2")

	assert.Equal(t, countObjects(ctx, t, s), 2)
	assert.Equal(t, countObjectTypes(ctx, t, s), 3)
	assert.Equal(t, countEdges(ctx, t, s), 5)
	assert.Equal(t, countEdgeTypes(ctx, t, s), 6)

	// Delete one edge type, ensure the edge type and any edges of that type get deleted,
	// but not other edges with the same type signature
	err := s.DeleteEdgeType(ctx, et2.ID)
	assert.NoErr(t, err)
	assert.Equal(t, countObjects(ctx, t, s), 2)
	assert.Equal(t, countObjectTypes(ctx, t, s), 3)
	assert.Equal(t, countEdges(ctx, t, s), 4)
	assert.Equal(t, countEdgeTypes(ctx, t, s), 5)
	_, err = s.GetEdge(ctx, e2.ID)
	assert.NotNil(t, err)
	_, err = s.GetEdgeType(ctx, et2.ID)
	assert.NotNil(t, err)

	// Delete one object type, ensure all associated edge types, edges, and objects get deleted.
	err = s.DeleteObjectType(ctx, ot1.ID)
	assert.NoErr(t, err)
	assert.Equal(t, countObjects(ctx, t, s), 1)
	assert.Equal(t, countObjectTypes(ctx, t, s), 2)
	assert.Equal(t, countEdges(ctx, t, s), 1)
	assert.Equal(t, countEdgeTypes(ctx, t, s), 2)
	_, err = s.GetEdge(ctx, e5.ID)
	assert.NoErr(t, err)
	_, err = s.GetEdgeType(ctx, et5.ID)
	assert.NoErr(t, err)
	_, err = s.GetObject(ctx, o2.ID)
	assert.NoErr(t, err)
	_, err = s.GetObjectType(ctx, ot2.ID)
	assert.NoErr(t, err)
}

func TestEdgePagination(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := initStorage(ctx, t)

	ot1 := createObjectType(t, ctx, s, "ot1")
	ot2 := createObjectType(t, ctx, s, "ot2")
	et := createEdgeType(t, ctx, s, "et1", ot1.TypeName, ot2.TypeName)

	objs := make([]authz.Object, 100)
	for i := range 50 {
		objs[i] = createObject(t, ctx, s, ot1.TypeName, fmt.Sprintf("o1_%d", i))
		objs[i+50] = createObject(t, ctx, s, ot2.TypeName, fmt.Sprintf("o2_%d", i))
	}

	for i := range 50 {
		_ = createEdge(t, ctx, s, et.TypeName, *objs[i].Alias, *objs[i+50].Alias)
	}

	time.Sleep(5 * time.Second)

	pager, err := authz.NewEdgePaginatorFromOptions(pagination.Limit(10))
	assert.NoErr(t, err)
	for {
		edges, pr, err := s.ListEdgesPaginated(ctx, *pager)
		assert.NoErr(t, err)
		assert.Equal(t, len(edges), 10)

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}

	pager, err = authz.NewEdgePaginatorFromOptions(pagination.Limit(10))
	assert.NoErr(t, err)

	ctx = dbMetrics.InitContext(ctx)
	dbm, err := dbMetrics.GetMetrics(ctx)
	assert.NoErr(t, err)

	callBefore := dbm.GetTotalCalls()
	for {
		edges, pr, err := s.ListEdgesPaginated(ctx, *pager)
		assert.NoErr(t, err)
		assert.Equal(t, len(edges), 10)

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}
	callsAfter := dbm.GetTotalCalls()
	// Verify that all the calls went against the cache
	assert.Equal(t, callsAfter-callBefore, 0)
}
func TestRecreateObject(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := initStorage(ctx, t)

	ot1 := createObjectType(t, ctx, s, "ot1")
	o1 := createObject(t, ctx, s, "ot1", "o1")

	// Delete object, ensure it's gone.
	err := s.DeleteObject(ctx, o1.ID)
	assert.NoErr(t, err)
	_, err = s.GetObject(ctx, o1.ID)
	assert.NotNil(t, err)

	// Create object of same type & ID in-place & validate it.
	err = s.SaveObject(ctx, &authz.Object{
		BaseModel: ucdb.NewBaseWithID(o1.ID),
		Alias:     o1.Alias,
		TypeID:    ot1.ID,
	})
	assert.NoErr(t, err)
	getO1, err := s.GetObject(ctx, o1.ID)
	assert.NoErr(t, err)
	assert.Equal(t, o1.ID, getO1.ID)

	// Delete it again.
	err = s.DeleteObject(ctx, o1.ID)
	assert.NoErr(t, err)
	_, err = s.GetObject(ctx, o1.ID)
	assert.NotNil(t, err)

	// Create object of different type but same ID in-place
	ot2 := createObjectType(t, ctx, s, "ot2")
	err = s.SaveObject(ctx, &authz.Object{
		BaseModel: ucdb.NewBaseWithID(o1.ID),
		Alias:     o1.Alias,
		TypeID:    ot2.ID,
	})
	assert.NoErr(t, err)
	getO1, err = s.GetObject(ctx, o1.ID)
	assert.NoErr(t, err)
	assert.Equal(t, o1.ID, getO1.ID)
}

func TestRecreateEdgeType(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := initStorage(ctx, t)

	_ = createObjectType(t, ctx, s, "ot1")
	ot2 := createObjectType(t, ctx, s, "ot2")
	et1 := createEdgeType(t, ctx, s, "et1", "ot1", "ot2")

	// Delete edge type, ensure it's gone.
	err := s.DeleteEdgeType(ctx, et1.ID)
	assert.NoErr(t, err)
	_, err = s.GetEdgeType(ctx, et1.ID)
	assert.NotNil(t, err)

	// Re-create edge type with same ID and object types
	err = s.SaveEdgeType(ctx, &authz.EdgeType{
		BaseModel:          ucdb.NewBaseWithID(et1.ID),
		TypeName:           et1.TypeName,
		SourceObjectTypeID: et1.SourceObjectTypeID,
		TargetObjectTypeID: et1.TargetObjectTypeID,
	})
	assert.NoErr(t, err)
	getET1, err := s.GetEdgeType(ctx, et1.ID)
	assert.NoErr(t, err)
	assert.Equal(t, et1.ID, getET1.ID)

	// Delete it again
	err = s.DeleteEdgeType(ctx, et1.ID)
	assert.NoErr(t, err)
	_, err = s.GetEdgeType(ctx, et1.ID)
	assert.NotNil(t, err)

	// Re-create edge type with same ID and different object type signature
	err = s.SaveEdgeType(ctx, &authz.EdgeType{
		BaseModel:          ucdb.NewBaseWithID(et1.ID),
		TypeName:           et1.TypeName,
		SourceObjectTypeID: ot2.ID,
		TargetObjectTypeID: ot2.ID,
	})
	assert.NoErr(t, err)
	getET1, err = s.GetEdgeType(ctx, et1.ID)
	assert.NoErr(t, err)
	assert.Equal(t, et1.ID, getET1.ID)
}

func TestKeyNames(t *testing.T) {
	np := authz.NewCacheNameProviderForTenant(uuid.Must(uuid.NewV4()))
	keyIDs := np.GetAllKeyIDs()
	for _, keyID := range keyIDs {
		assert.True(t, len(keyID) > 3)
		key := np.GetKeyName(cache.KeyNameID(keyID), []string{"a", "b", "c"})
		assert.True(t, len(key) > 3)
	}
}

func TestGlobalEdgesCacheInvalidation(t *testing.T) {
	ctx := context.Background()

	tenantID := uuid.Must(uuid.NewV4())
	tdb := testdb.New(t, migrate.NewTestSchema(tenantdb.Schema))

	t.Run("TestGlobalCacheInvalidation", func(t *testing.T) {
		s := internal.NewStorage(ctx, uuid.Must(uuid.NewV4()), tdb, cachehelpers.NewCacheConfig())
		runID := string(uuid.Must(uuid.NewV4()).String())
		ot1 := createObjectType(t, ctx, s, "ot1"+runID)
		ot2 := createObjectType(t, ctx, s, "ot2"+runID)
		et := createEdgeType(t, ctx, s, "et1"+runID, ot1.TypeName, ot2.TypeName)

		objectCount := 100
		objs := make([]authz.Object, objectCount)
		for i := range objectCount / 2 {
			objs[i] = createObject(t, ctx, s, ot1.TypeName, fmt.Sprintf("o1_%d_%v", i, runID))
			objs[i+objectCount/2] = createObject(t, ctx, s, ot2.TypeName, fmt.Sprintf("o2_%d_%v", i, runID))
		}

		for i := range objectCount / 2 {
			_ = createEdge(t, ctx, s, et.TypeName, *objs[i].Alias, *objs[i+objectCount/2].Alias)
		}
		// Test creating another edge, deleting it and creating it again
		eFlip := createEdge(t, ctx, s, et.TypeName, *objs[1].Alias, *objs[objectCount/2+2].Alias)
		validateEdges(t, ctx, s, nil, objs[1].ID, objs[objectCount/2+2].ID, true)
		err := s.DeleteEdge(ctx, eFlip.ID)
		assert.NoErr(t, err)
		eFlip = createEdge(t, ctx, s, et.TypeName, *objs[1].Alias, *objs[objectCount/2+2].Alias)
		validateEdges(t, ctx, s, nil, objs[1].ID, objs[objectCount/2+2].ID, true)
		err = s.DeleteEdge(ctx, eFlip.ID)
		assert.NoErr(t, err)

		// Creating the same edge (different EdgeID) from multiple threads and make sure the conflict doesn't cause invalidation not to fire
		wg := sync.WaitGroup{}
		for i := range 5 {
			wg.Add(1)

			go func(threadID int) {
				defer wg.Done()
				edge := authz.Edge{
					BaseModel:      ucdb.NewBase(),
					EdgeTypeID:     et.ID,
					SourceObjectID: objs[1].ID,
					TargetObjectID: objs[objectCount/2+2].ID,
				}
				err := s.InsertEdge(ctx, &edge)

				if err == nil {
					eFlip = edge
				}
			}(i)
		}
		wg.Wait()
		validateEdges(t, ctx, s, nil, objs[1].ID, objs[objectCount/2+2].ID, true)
		err = s.DeleteEdge(ctx, eFlip.ID)
		assert.NoErr(t, err)
		validateEdges(t, ctx, s, nil, objs[1].ID, objs[objectCount/2+2].ID, false)

		// Creating the same edge (same EdgeID) from multiple threads and make sure the conflict doesn't cause invalidation not to fire
		edgeID := uuid.Must(uuid.NewV4())
		for i := range 5 {
			wg.Add(1)

			go func(threadID int) {
				defer wg.Done()
				edge := authz.Edge{
					BaseModel:      ucdb.NewBaseWithID(edgeID),
					EdgeTypeID:     et.ID,
					SourceObjectID: objs[1].ID,
					TargetObjectID: objs[objectCount/2+2].ID,
				}
				err := s.InsertEdge(ctx, &edge)

				if err == nil {
					eFlip = edge
				}
			}(i)
		}
		wg.Wait()
		validateEdges(t, ctx, s, nil, objs[1].ID, objs[objectCount/2+2].ID, true)
		err = s.DeleteEdge(ctx, eFlip.ID)
		assert.NoErr(t, err)
		validateEdges(t, ctx, s, nil, objs[1].ID, objs[objectCount/2+2].ID, false)

		time.Sleep(5 * time.Second)

		ctx = dbMetrics.InitContext(ctx)
		dbm, err := dbMetrics.GetMetrics(ctx)
		assert.NoErr(t, err)

		// Test that cancelling a context mid population of the cache does not leave the cache in an inconsistent state
		startTimeout := time.Microsecond * 100
		for {
			withTimeoutCtx, cancel := context.WithTimeout(ctx, startTimeout)
			defer cancel()

			val, _, err := s.CheckAttribute(withTimeoutCtx, nil, objs[0].ID, objs[1].ID, "read")
			assert.Equal(t, (err == nil || errors.Is(err, context.DeadlineExceeded)), true)
			assert.Equal(t, val, false)
			// If we cancel the context in time check that the map is fully populated
			if err != nil {
				callBefore := dbm.GetTotalCalls()
				validateEdges(t, ctx, s, nil, objs[0].ID, objs[objectCount/2].ID, true)
				callsAfter := dbm.GetTotalCalls()
				assert.Equal(t, (callsAfter-callBefore) > 0, true)
				break
			} else {
				startTimeout = startTimeout / 2
			}
		}
		// Populate the global cache
		validateEdges(t, ctx, s, nil, objs[0].ID, objs[1].ID, false)
		// Validate that the in process global cache is populated and the read does hit redis

		callBefore := dbm.GetTotalCalls()
		validateEdges(t, ctx, s, nil, objs[1].ID, objs[2].ID, false)
		validateEdges(t, ctx, s, nil, objs[0].ID, objs[objectCount/2].ID, true)
		callsAfter := dbm.GetTotalCalls()
		// Verify that all the calls went against the cache
		assert.Equal(t, callsAfter-callBefore, 0)

		// Invalidate the edge global cache and test creating another edge, deleting it and creating it again
		_ = createEdge(t, ctx, s, et.TypeName, *objs[1].Alias, *objs[objectCount/2+2].Alias)

		time.Sleep(5 * time.Second)

		// Check the global cache is refreshed
		callBefore = dbm.GetTotalCalls()
		wg = sync.WaitGroup{}
		for i := range 5 {
			wg.Add(1)

			go func(threadID int) {
				defer wg.Done()
				validateEdges(t, ctx, s, nil, objs[threadID].ID, objs[threadID+1].ID, false)
				validateEdges(t, ctx, s, nil, objs[threadID].ID, objs[threadID+objectCount/2].ID, true)
			}(i)
		}
		wg.Wait()
		callsAfter = dbm.GetTotalCalls()
		// Verify that all the calls went against the cache
		expectedCalls := 1
		// TODO change this to one once we don't do double load on NoConflict
		if featureflags.IsEnabledForTenant(ctx, featureflags.OnMachineEdgesCacheVerify, tenantID) {
			expectedCalls = 2
		}
		assert.Equal(t, callsAfter-callBefore, expectedCalls)
		// Test that in a single threaded pattern, we store the cache of edges in the global cache against the tombstone value. Invalidate the cache to generate the tombstone
		_ = createEdge(t, ctx, s, et.TypeName, *objs[1].Alias, *objs[objectCount/2+3].Alias)

		callBefore = dbm.GetTotalCalls()
		validateEdges(t, ctx, s, nil, objs[1].ID, objs[2].ID, false)
		callsAfter = dbm.GetTotalCalls()
		// Verify that all the call went against DB
		assert.Equal(t, callsAfter-callBefore, 1)

		callBefore = dbm.GetTotalCalls()
		validateEdges(t, ctx, s, nil, objs[2].ID, objs[3].ID, false)
		callsAfter = dbm.GetTotalCalls()
		// Verify that all the call went against cache (note that we need the second call to occur before tombstone expires but we have 5 seconds)
		assert.Equal(t, callsAfter-callBefore, 0)

		// Test that in a multi threaded pattern only one of the threads hits the server to get the edges and rest read the cache. Invalidate the cache to generate the tombstone
		edgeTestVal := createEdge(t, ctx, s, et.TypeName, *objs[1].Alias, *objs[objectCount/2+4].Alias)
		err = s.DeleteEdge(ctx, edgeTestVal.ID)
		assert.NoErr(t, err)

		callBefore = dbm.GetTotalCalls()
		wg = sync.WaitGroup{}
		for i := range 5 {
			wg.Add(1)

			go func(threadID int) {
				defer wg.Done()
				validateEdges(t, ctx, s, nil, objs[1].ID, objs[objectCount/2+4].ID, false)
				validateEdges(t, ctx, s, nil, objs[threadID].ID, objs[threadID+1].ID, false)
				validateEdges(t, ctx, s, nil, objs[threadID+5].ID, objs[threadID+5+objectCount/2].ID, true)
			}(i)
		}
		wg.Wait()
		callsAfter = dbm.GetTotalCalls()
		// Verify that all but one of the calls went against the cache
		assert.Equal(t, callsAfter-callBefore, 1)

		// Test that in a multithread environment read after write consistency is maintained
		wg = sync.WaitGroup{}
		for i := 1; i < 10; i++ {
			wg.Add(1)

			go func(threadID int) {
				defer wg.Done()
				testEdgeVal := createEdge(t, ctx, s, et.TypeName, *objs[threadID].Alias, *objs[objectCount/2+5+threadID].Alias)
				validateEdges(t, ctx, s, nil, objs[threadID].ID, objs[objectCount/2+5+threadID].ID, true)

				err := s.DeleteEdge(ctx, testEdgeVal.ID)
				assert.NoErr(t, err)
				validateEdges(t, ctx, s, nil, objs[threadID].ID, objs[objectCount/2+5+threadID].ID, false)

				_ = createEdge(t, ctx, s, et.TypeName, *objs[threadID].Alias, *objs[objectCount/2+6+threadID].Alias)
				validateEdges(t, ctx, s, nil, objs[threadID].ID, objs[objectCount/2+6+threadID].ID, true)
			}(i)
		}
		wg.Wait()

		// Check that all the deleted edges are not present in the cache
		for i := 1; i < 10; i++ {
			validateEdges(t, ctx, s, nil, objs[i].ID, objs[objectCount/2+5+i].ID, false)
		}

		// Check that all the added edges are present in the cache
		for i := 1; i < 10; i++ {
			validateEdges(t, ctx, s, nil, objs[i].ID, objs[objectCount/2+6+i].ID, true)
		}

		for i := range objectCount / 2 {
			validateEdges(t, ctx, s, nil, objs[i].ID, objs[objectCount/2+i].ID, true)
		}

		err = s.FlushCacheForEdgeType(ctx, et.ID)
		assert.NoErr(t, err)

		callBefore = dbm.GetTotalCalls()

		validateEdges(t, ctx, s, nil, objs[2].ID, objs[3].ID, false)

		for i := range objectCount / 2 {
			validateEdges(t, ctx, s, nil, objs[i].ID, objs[objectCount/2+i].ID, true)
		}
		callsAfter = dbm.GetTotalCalls()
		// Verify that all the call went against cache (note that we need the second call to occur before tombstone expires but we have 5 seconds)
		assert.Equal(t, callsAfter-callBefore, 2)

		err = s.DeleteEdgeType(ctx, et.ID)
		assert.NoErr(t, err)

		callBefore = dbm.GetTotalCalls()

		for i := range objectCount / 2 {
			validateEdges(t, ctx, s, nil, objs[i].ID, objs[objectCount/2+i].ID, false)
		}

		callsAfter = dbm.GetTotalCalls()
		// Verify that there was one reload call after the edge type was deleted
		assert.True(t, (callsAfter-callBefore) == 51 || (callsAfter-callBefore) == 52) // 50 for the edge types collection and 1/2 for the edges
	})

	t.Run("TestGlobalCacheInvalidationCrossMachine", func(t *testing.T) {
		ctx := request.SetRequestIDIfNotSet(ctx, uuid.Must(uuid.NewV4()))

		_, err := internal.GetCacheManager(ctx, cachehelpers.NewCacheConfig(), tenantID)
		assert.NoErr(t, err)
		// Get the same redis cache as we are using in the storage
		sharedCache, err := cache.InitializeInvalidatingCacheFromConfig(
			ctx,
			cachehelpers.NewCacheConfig(),
			cache.RegionalRedisCacheName,
			authz.CachePrefix,
			cache.InvalidationHandlersLocalPublish(internal.AuthzHandlersInvalidations, []string{authz.EdgeCollectionKeyString}),
			// We don't use an invalidation delay in production but offbox SQL operations take much longer than on a devbox
			// to emulate the same behavior we set the delay to 500ns. Normally the time from from redis sent to receive is less < 150ns, so this just gives us a little buffer
			cache.InvalidationDelay(500*time.Nanosecond),
		)
		assert.NoErr(t, err)
		//sharedCache.P.SetTombstoneTTL(t, tombstoneTTL)

		// Create two different local on machine edge caches, each with its own invalidation worker thread recieving invalidation messages
		eC1 := &internal.EdgeCacheMapRecord{CacheTenantRecords: make(map[uuid.UUID]*internal.EdgeTenantRecord)}
		eC2 := &internal.EdgeCacheMapRecord{CacheTenantRecords: make(map[uuid.UUID]*internal.EdgeTenantRecord)}
		// Create two different storage instances wrapped around separate on machine edges caches
		s1 := internal.NewStorageForTests(ctx, tenantID, tdb, sharedCache.(*cache.InvalidationWrapper), eC1)
		s2 := internal.NewStorageForTests(ctx, tenantID, tdb, sharedCache.(*cache.InvalidationWrapper), eC2)

		// Create the test objects/edges through the two different storage instances
		runID := uuid.Must(uuid.NewV4()).String()
		ot1 := createObjectType(t, ctx, s1, "ot1"+runID)
		ot2 := createObjectType(t, ctx, s2, "ot2"+runID)
		et1 := createEdgeType(t, ctx, s1, "et1"+runID, ot1.TypeName, ot2.TypeName)
		et2 := createEdgeType(t, ctx, s1, "et2"+runID, ot2.TypeName, ot1.TypeName)

		objectCount := 100 // Set to 3100 for multipage test
		objs := make([]authz.Object, objectCount)
		for i := range objectCount / 2 {
			objs[i] = createObject(t, ctx, s1, ot1.TypeName, fmt.Sprintf("o1_%d_%v", i, runID))
			objs[i+objectCount/2] = createObject(t, ctx, s2, ot2.TypeName, fmt.Sprintf("o2_%d_%v", i, runID))
		}

		for i := range objectCount / 2 {
			createEdge(t, ctx, s1, et1.TypeName, *objs[i].Alias, *objs[i+objectCount/2].Alias)
			createEdge(t, ctx, s2, et2.TypeName, *objs[i+objectCount/2].Alias, *objs[i].Alias)
		}

		edgesToDelete := make([]authz.Edge, objectCount/4)
		for i := range objectCount / 4 {
			e := createEdge(t, ctx, s1, et1.TypeName, *objs[i].Alias, *objs[i+1+objectCount/2].Alias)
			edgesToDelete[i] = e
		}

		wgInitLoad := sync.WaitGroup{}
		wgInitLoad.Add(2)
		go func() {
			defer wgInitLoad.Done()
			// Do a basic check to ensure that the edges are present in both storage instances while deleting some edges at the same time
			for i := range objectCount / 2 {
				validateEdges(t, ctx, s1, s2, objs[i].ID, objs[objectCount/2+i].ID, true)
				validateEdges(t, ctx, s1, s2, objs[objectCount/2+i].ID, objs[i].ID, true)
			}
		}()

		go func() {
			defer wgInitLoad.Done()
			for i := range objectCount / 4 {
				err = s1.DeleteEdge(ctx, edgesToDelete[i].ID)
				assert.NoErr(t, err)
			}
		}()
		wgInitLoad.Wait()

		// Validate that none of the deletes were missed
		for i := range objectCount / 4 {
			validateEdges(t, ctx, s1, s2, objs[i].ID, objs[i+1+objectCount/2].ID, false)
		}

		// Let the MOD key tombstone expire
		time.Sleep(5 * time.Second)

		// Populate the steady state global cache (cache.NoConflictSentinel)
		path, _, err := s1.CheckAttribute(ctx, nil, objs[0].ID, objs[1].ID, "read")
		assert.NoErr(t, err)
		assert.False(t, path)

		path, _, err = s2.CheckAttribute(ctx, nil, objs[0].ID, objs[1].ID, "read")
		assert.NoErr(t, err)
		assert.False(t, path)

		for range 1 {
			threadCount := 10
			repsPerThread := 10
			// Test that in a multithread environment read after write consistency is maintained
			wg := sync.WaitGroup{}
			for i := 1; i < threadCount; i++ {
				wg.Add(1)

				// Use a different requestID to identify the worker threads in the logs
				ctxWorker := request.SetRequestIDIfNotSet(context.Background(), uuid.Must(uuid.NewV4()))

				go func(ctx context.Context, threadID int) {
					defer wg.Done()
					var err error

					for k := 1; k < repsPerThread; k++ {
						multiThreadVerify := false //threadID%2 == 0
						testEdgeVal := createEdge(t, ctx, s1, et1.TypeName, *objs[threadID].Alias, *objs[objectCount/2+1+threadID].Alias)
						uclog.Verbosef(ctx, "(iter %v) Created edge %v from %v to %v updated %v", k, testEdgeVal.ID, objs[threadID].ID, objs[objectCount/2+1+threadID].ID, testEdgeVal.Updated)
						// On even threads check the edge in both storage instances immediately and on odd threads check the edge in from a set of worker threads
						validateMultiThreaded(t, ctx, multiThreadVerify, threadCount/2, s1, s2, objs[threadID].ID, objs[objectCount/2+1+threadID].ID, true)

						err = s1.DeleteEdge(ctx, testEdgeVal.ID)
						assert.NoErr(t, err)
						uclog.Verbosef(ctx, "(iter %v) Deleted edge %v from %v to %v", k, testEdgeVal.ID, objs[threadID].ID, objs[objectCount/2+1+threadID].ID)
						validateMultiThreaded(t, ctx, multiThreadVerify, threadCount/2, s1, s2, objs[threadID].ID, objs[objectCount/2+1+threadID].ID, false)

						testEdgeVal = createEdgeWithID(t, ctx, s1, testEdgeVal.ID, et1.TypeName, *objs[threadID].Alias, *objs[objectCount/2+1+threadID].Alias)
						assert.NoErr(t, err)
						uclog.Verbosef(ctx, "(iter %v) Created edge %v from %v to %v", k, testEdgeVal.ID, objs[threadID].ID, objs[objectCount/2+1+threadID].ID)
						validateMultiThreaded(t, ctx, multiThreadVerify, threadCount/2, s1, s2, objs[threadID].ID, objs[objectCount/2+1+threadID].ID, true)

						err = s1.DeleteEdge(ctx, testEdgeVal.ID)
						assert.NoErr(t, err)
						uclog.Verbosef(ctx, "(iter %v) Deleted edge %v from %v to %v", k, testEdgeVal.ID, objs[threadID].ID, objs[objectCount/2+1+threadID].ID)
						validateMultiThreaded(t, ctx, multiThreadVerify, threadCount/2, s1, s2, objs[threadID].ID, objs[objectCount/2+1+threadID].ID, false)

						testEdgeVal = createEdgeIfNotExist(t, ctx, s1, et1.TypeName, *objs[threadID].Alias, *objs[objectCount/2+2+threadID].Alias)
						uclog.Verbosef(ctx, "(iter %v) Created edge %v from %v to %v updated %v", k, testEdgeVal.ID, objs[threadID].ID, objs[objectCount/2+2+threadID].ID, testEdgeVal.Updated)
						validateMultiThreaded(t, ctx, multiThreadVerify, threadCount/2, s1, s2, objs[threadID].ID, objs[objectCount/2+2+threadID].ID, true)
					}
					uclog.Verbosef(ctx, "Thread %v done", threadID)
				}(ctxWorker, i)
			}
			wg.Wait()

			// Check that all the deleted edges are not present in the cache
			for i := 1; i < threadCount; i++ {
				validateEdges(t, ctx, s1, s2, objs[i].ID, objs[objectCount/2+1+i].ID, false)
			}

			// Check that all the added edges are present in the cache
			for i := 1; i < threadCount; i++ {
				validateEdges(t, ctx, s1, s2, objs[i].ID, objs[objectCount/2+2+i].ID, true)
			}

			// Check that the original edges are still present in the cache
			for i := range objectCount / 2 {
				validateEdges(t, ctx, s1, s2, objs[i].ID, objs[objectCount/2+i].ID, true)
				validateEdges(t, ctx, s1, s2, objs[objectCount/2+i].ID, objs[i].ID, true)
			}
		}
		err = s1.DeleteEdgeType(ctx, et1.ID)
		assert.NoErr(t, err)

		for i := range objectCount / 2 {
			validateEdges(t, ctx, s1, s2, objs[i].ID, objs[objectCount/2+i].ID, false)
			validateEdges(t, ctx, s1, s2, objs[objectCount/2+i].ID, objs[i].ID, true)
		}

		for i := range objectCount / 4 {
			if i%2 == 0 {
				err = s2.DeleteObject(ctx, objs[i].ID)
				assert.NoErr(t, err)
			}
		}

		for i := range objectCount / 4 {
			validateEdges(t, ctx, s1, s2, objs[i].ID, objs[objectCount/2+i].ID, false)
			validateEdges(t, ctx, s1, s2, objs[objectCount/2+i].ID, objs[i].ID, i%2 != 0)
		}

		for i := objectCount / 4; i < objectCount/2; i++ {
			if i%2 == 0 {
				e, err := s1.FindEdge(ctx, et2.ID, objs[objectCount/2+i].ID, objs[i].ID)
				assert.NoErr(t, err)
				err = s2.DeleteEdge(ctx, e.ID)
				assert.NoErr(t, err)
			}
		}

		for i := objectCount / 4; i < objectCount/2; i++ {
			validateEdges(t, ctx, s1, s2, objs[i].ID, objs[objectCount/2+i].ID, false)
			validateEdges(t, ctx, s1, s2, objs[objectCount/2+i].ID, objs[i].ID, i%2 != 0)
		}

	})

	t.Run("TestGlobalCacheErrorDetectionWorker", func(t *testing.T) {
		// t.Parallel() - don't run in parallel as we need to make sure no other edges are being created/deleted
		_, err := internal.GetCacheManager(ctx, cachehelpers.NewCacheConfig(), tenantID)
		assert.NoErr(t, err)
		// Get the same redis cache as we are using in the storage
		sharedCache, err := cache.InitializeInvalidatingCacheFromConfig(
			ctx,
			cachehelpers.NewCacheConfig(),
			cache.RegionalRedisCacheName,
			authz.CachePrefix,
			cache.InvalidationHandlersLocalPublish(internal.AuthzHandlersInvalidations, []string{authz.EdgeCollectionKeyString}),
		)
		assert.NoErr(t, err)

		// Create storage instance with a local on machine edge cache that we have access to from the test
		eC1 := &internal.EdgeCacheMapRecord{CacheTenantRecords: make(map[uuid.UUID]*internal.EdgeTenantRecord)}
		s1 := internal.NewStorageForTests(ctx, tenantID, tdb, sharedCache.(*cache.InvalidationWrapper), eC1)

		// Create the test objects/edges
		runID := string(uuid.Must(uuid.NewV4()).String())
		ot1 := createObjectType(t, ctx, s1, "ot1"+runID)
		ot2 := createObjectType(t, ctx, s1, "ot2"+runID)
		et1 := createEdgeType(t, ctx, s1, "et1"+runID, ot1.TypeName, ot2.TypeName)
		et2 := createEdgeType(t, ctx, s1, "et2"+runID, ot2.TypeName, ot1.TypeName)

		objectCount := 8
		objs := make([]authz.Object, objectCount)
		for i := range objectCount / 2 {
			objs[i] = createObject(t, ctx, s1, ot1.TypeName, fmt.Sprintf("o1_%d_%v", i, runID))
			objs[i+objectCount/2] = createObject(t, ctx, s1, ot2.TypeName, fmt.Sprintf("o2_%d_%v", i, runID))
		}

		edgeIDForObj0 := uuid.Nil
		for i := range objectCount / 2 {
			edge := createEdge(t, ctx, s1, et1.TypeName, *objs[i].Alias, *objs[i+objectCount/2].Alias)
			if i == 0 {
				edgeIDForObj0 = edge.ID
			}
			_ = createEdge(t, ctx, s1, et2.TypeName, *objs[i+objectCount/2].Alias, *objs[i].Alias)
		}

		assert.False(t, internal.ErrorDetectionWorker(ctx, s1))
		time.Sleep(5 * time.Second)
		assert.False(t, internal.ErrorDetectionWorker(ctx, s1))

		// Do a basic check to ensure that the edges are present and populate the cache
		for i := range objectCount / 4 {
			validateEdges(t, ctx, s1, nil, objs[i].ID, objs[objectCount/2+i].ID, true)
			validateEdges(t, ctx, s1, nil, objs[objectCount/2+i].ID, objs[i].ID, true)
		}

		assert.NotNil(t, eC1.CacheTenantRecords[tenantID], assert.Must())
		assert.NotNil(t, eC1.CacheTenantRecords[tenantID].CacheRecords[string(cache.NoLockSentinel)], assert.Must())

		// Delete the source object from the cache and check that the error detection worker detects the error
		localCacheEdges := eC1.CacheTenantRecords[tenantID].CacheRecords[string(cache.NoLockSentinel)].EdgesMap
		obj1Edges := localCacheEdges[objs[0].ID]
		delete(localCacheEdges, objs[0].ID)
		assert.True(t, internal.ErrorDetectionWorker(ctx, s1))
		localCacheEdges = eC1.CacheTenantRecords[tenantID].CacheRecords[string(cache.NoLockSentinel)].EdgesMap
		assert.NotNil(t, localCacheEdges[objs[0].ID], assert.Must())
		assert.Equal(t, localCacheEdges[objs[0].ID][edgeIDForObj0], obj1Edges[edgeIDForObj0])
		assert.False(t, internal.ErrorDetectionWorker(ctx, s1))
		// Add a fake edge to the cache and check that the error detection worker detects the error
		fakeEdge := &authz.Edge{BaseModel: ucdb.NewBase(), SourceObjectID: objs[0].ID, TargetObjectID: objs[1].ID, EdgeTypeID: et1.ID}
		obj1EdgesWithFakeEdge := maps.Clone(obj1Edges)
		obj1EdgesWithFakeEdge[fakeEdge.ID] = fakeEdge
		localCacheEdges[objs[0].ID] = obj1EdgesWithFakeEdge
		assert.True(t, internal.ErrorDetectionWorker(ctx, s1))
		localCacheEdges = eC1.CacheTenantRecords[tenantID].CacheRecords[string(cache.NoLockSentinel)].EdgesMap
		assert.NotNil(t, localCacheEdges[objs[0].ID], assert.Must())
		assert.Equal(t, localCacheEdges[objs[0].ID][edgeIDForObj0], obj1Edges[edgeIDForObj0])
		assert.Equal(t, len(localCacheEdges[objs[0].ID]), 1)
		assert.False(t, internal.ErrorDetectionWorker(ctx, s1))
	})
}

func validateMultiThreaded(t *testing.T,
	ctx context.Context,
	runMulti bool,
	threadCount int,
	s1 *internal.Storage,
	s2 *internal.Storage,
	objID1 uuid.UUID,
	objID2 uuid.UUID,
	expectedVal bool) {

	t.Helper()

	if !runMulti {
		validateEdges(t, ctx, s1, s2, objID1, objID2, expectedVal)
	} else {
		wgInner := sync.WaitGroup{}
		defer wgInner.Wait()
		for j := 1; j < threadCount; j++ {
			wgInner.Add(1)
			go func(parentThreadID int) {
				t.Helper()
				defer wgInner.Done()
				validateEdges(t, ctx, s1, s2, objID1, objID2, expectedVal)
			}(j)
		}
	}
}

func validateEdges(t *testing.T, ctx context.Context, s1 *internal.Storage, s2 *internal.Storage, objID1 uuid.UUID, objID2 uuid.UUID, expectedVal bool) {
	t.Helper()

	uclog.Debugf(ctx, "Checking attribute for %v -> %v", objID1, objID2)
	val, _, err := s1.CheckAttribute(ctx, nil, objID1, objID2, "read")
	assert.NoErr(t, err)
	assert.Equal(t, val, expectedVal, assert.Errorf("Error for s1 in edge cache for %v -> %v expected %v returned %v", objID1, objID2, expectedVal, val))
	if val != expectedVal {
		uclog.Errorf(ctx, "Error for s1 in edge cache for %v -> %v expected %v returned %v", objID1, objID2, expectedVal, val)
	}

	if s2 == nil {
		return
	}

	val, _, err = s2.CheckAttribute(ctx, nil, objID1, objID2, "read")
	assert.NoErr(t, err)
	assert.Equal(t, val, expectedVal, assert.Errorf("Error for s2 in edge cache for %v -> %v expected %v returned %v", objID1, objID2, expectedVal, val))
}
