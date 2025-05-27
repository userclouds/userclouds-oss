package storage_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/cache"
	cachehelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/tenantdb"
	"userclouds.com/test/testlogtransport"
)

// this would normally be an unusual test to write (since it seemingly tests
// the ORM) but since we're using an unconventional PK setup for this table,
// it seems worthwhile.
func TestTokenSave(t *testing.T) {
	t.Parallel()

	fixture := newStorageForTests(t)
	s := fixture.s

	tr := &storage.TokenRecord{
		BaseModel:          ucdb.NewBase(),
		Data:               "foo",
		Token:              "bar",
		AccessPolicyID:     uuid.Must(uuid.NewV4()),
		TransformerID:      uuid.Must(uuid.NewV4()),
		TransformerVersion: 0,
	}
	assert.NoErr(t, s.SaveTokenRecord(fixture.ctx, tr))

	tr.ID = uuid.Must(uuid.NewV4())
	assert.NotNil(t, s.SaveTokenRecord(fixture.ctx, tr))
}

func TestGetTransformersMapWithCache(t *testing.T) {
	// NOTE: this test must run serially because it expects a specific number of error messages
	tf := newTestFixtureForClientCache(t, "TRANSFORMER")
	tf1 := newFakeTransformer("George")
	tf2 := newFakeTransformer("Costanza")

	assert.NoErr(t, tf.storage.SaveTransformer(tf.ctx, tf1))
	assert.NoErr(t, tf.storage.SaveTransformer(tf.ctx, tf2))
	validateValueInCacheByID(tf.ctx, t, tf.storage, storage.TransformerKeyID, tf1.ID, *tf1)
	validateValueInCacheByID(tf.ctx, t, tf.storage, storage.TransformerKeyID, tf2.ID, *tf2)
	// now delete them.
	cachehelpers.ForceDeleteCacheValueByID(tf.ctx, t, *tf.storage.CacheManager(), storage.TransformerKeyID, tf1.ID)
	cachehelpers.ForceDeleteCacheValueByID(tf.ctx, t, *tf.storage.CacheManager(), storage.TransformerKeyID, tf2.ID)
	expectedDBCalls := tf.getTotalDBCalls()

	// Properly handle empty list
	tfMap, err := tf.storage.GetTransformersMap(tf.ctx, []uuid.UUID{})
	assert.NoErr(t, err)
	assert.Equal(t, len(tfMap), 0)
	tf.assertDBCallCount(expectedDBCalls)

	expectedDBCalls += 2 // One DB roundtrip per transformer
	tfMap, err = tf.storage.GetTransformersMap(tf.ctx, []uuid.UUID{tf1.ID, tf2.ID})
	assert.NoErr(t, err)
	tf.assertDBCallCount(expectedDBCalls)
	assert.Equal(t, len(tfMap), 2)
	assert.Equal(t, *tfMap[tf1.ID], *tf1)
	assert.Equal(t, *tfMap[tf2.ID], *tf2)

	// Load from cache, no DB access (hence expectedDBCalls stays the same)
	tfMap, err = tf.storage.GetTransformersMap(tf.ctx, []uuid.UUID{tf1.ID, tf2.ID})
	assert.NoErr(t, err)
	tf.assertDBCallCount(expectedDBCalls)
	assert.Equal(t, len(tfMap), 2)
	assert.Equal(t, *tfMap[tf1.ID], *tf1)
	assert.Equal(t, *tfMap[tf2.ID], *tf2)

	tf1.Name = "Newman"
	expectedDBCalls++
	assert.IsNil(t, tf.storage.SaveTransformer(tf.ctx, tf1), assert.Must())
	validateValueInCacheByID(tf.ctx, t, tf.storage, storage.TransformerKeyID, tf1.ID, *tf1)
	tf.assertDBCallCount(expectedDBCalls)

	// Loaded, cache miss, so DB is hit
	expectedDBCalls++
	cachehelpers.ForceDeleteCacheValueByID(tf.ctx, t, *tf.storage.CacheManager(), storage.TransformerKeyID, tf1.ID)
	tfMap, err = tf.storage.GetTransformersMap(tf.ctx, []uuid.UUID{tf1.ID, tf2.ID})
	assert.NoErr(t, err)
	tf.assertDBCallCount(expectedDBCalls)
	assert.Equal(t, len(tfMap), 2)
	assert.Equal(t, *tfMap[tf1.ID], *tf1)
	assert.Equal(t, *tfMap[tf2.ID], *tf2)
}

func TestGetColumnsWithCache(t *testing.T) {
	t.Parallel()

	tf := newTestFixtureForClientCache(t, "COLUMN")
	col1 := &storage.Column{
		BaseModel:            ucdb.NewBase(),
		Table:                "users",
		Name:                 "jerry",
		DataTypeID:           datatype.String.ID,
		IsArray:              false,
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	}
	assert.IsNil(t, tf.storage.SaveColumn(tf.ctx, col1), assert.Must())
	col2 := &storage.Column{
		BaseModel:            ucdb.NewBase(),
		Table:                "users",
		Name:                 "kramer",
		DataTypeID:           datatype.Timestamp.ID,
		IsArray:              false,
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	}
	/*col3 := &Column{
		BaseModel: ucdb.NewBase(),
		Table:     "users",
		Name:      "Newman",
		DataTypeID: datatype.Integer.ID,
		IsArray:   false,
	}*/
	assert.IsNil(t, tf.storage.SaveColumn(tf.ctx, col2), assert.Must())

	/* TODO renenable by setting tomstone to 0
	tf.assertNoKeyInCache(tf.keyPrefix + "COL") // No column collection item in cache
	expectedDBCalls := tf.getTotalDBCalls() + 1 // only one DB calls is expected to populate the cache (batch load columns from the DB)
	cols, err := listColumnsNonPaginated(tf.ctx, tf.storage)
	assert.NoErr(t, err)
	tf.assertDBCallCount(expectedDBCalls)
	assert.Equal(t, len(cols), 2)
	assert.Equal(t, len(tf.cacheGetCollection()), 2) // 2 columns in the json object

	cols, err = tf.storage.ListColumnsNonPaginated(tf.ctx)
	assert.NoErr(t, err)
	tf.assertDBCallCount(expectedDBCalls) // didn't change, got it from cache
	assert.Equal(t, len(cols), 2)

	assert.Equal(t, findByID(t, cols, col1.ID), col1)
	assert.Equal(t, findByID(t, cols, col2.ID), col2)

	// Bust the cache
	assert.IsNil(t, tf.storage.SaveColumn(tf.ctx, col3), assert.Must())
	tf.assertNoKeyInCache(tf.keyPrefix + "COL") // Saving a single item should bust the collection cache key
	expectedDBCalls = tf.getTotalDBCalls() + 1  // only one DB calls is expected to populate the cache (batch load columns from the DB)
	cols, err = tf.storage.ListColumnsNonPaginated(tf.ctx)
	assert.NoErr(t, err)
	tf.assertDBCallCount(expectedDBCalls)
	assert.Equal(t, len(tf.cacheGetCollection()), 3) // 3 columns in the json object
	assert.Equal(t, len(cols), 3)
	assert.Equal(t, findByID(t, cols, col1.ID), col1)
	assert.Equal(t, findByID(t, cols, col2.ID), col2)
	assert.Equal(t, findByID(t, cols, col3.ID), col3)
	*/
}

/*
func findByID(t *testing.T, cols []Column, id uuid.UUID) *Column {
	for _, col := range cols {
		if col.ID == id {
			return &col
		}
	}
	assert.FailContinue(t, "Can't find column with ID: %v", id)
	return nil
}*/

func TestGetPurposeWithCache(t *testing.T) {
	t.Parallel()

	tf := newTestFixtureForClientCache(t, "PURPOSE")
	prp := &storage.Purpose{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
		Name:                     "ForJerry",
		Description:              "I have a sixth sense",
	}
	expectedNumFields := tf.getNumFields(reflect.ValueOf(*prp))

	expectedDBCalls := tf.getTotalDBCalls()
	assert.IsNil(t, tf.storage.SavePurpose(tf.ctx, prp), assert.Must())
	expectedKey := fmt.Sprintf("%s_%v", tf.keyPrefix, prp.ID)
	expectedDBCalls++
	tf.assertDBCallCount(expectedDBCalls)
	assert.Equal(t, len(tf.cacheGetJSON(expectedKey)), expectedNumFields)

	// Load from cache
	loadedPurpose, err := tf.storage.GetPurpose(tf.ctx, prp.ID)
	assert.NoErr(t, err)
	assert.Equal(t, loadedPurpose, prp)
	tf.assertDBCallCount(expectedDBCalls)

	// Load from DB and populate cache
	tf.deleteKey(expectedKey)
	expectedDBCalls++
	loadedPurpose, err = tf.storage.GetPurpose(tf.ctx, prp.ID)
	assert.NoErr(t, err)
	assert.Equal(t, loadedPurpose, prp)
	tf.assertDBCallCount(expectedDBCalls)

	prp.Description = "Cheapness is not a sense"
	expectedDBCalls++
	assert.IsNil(t, tf.storage.SavePurpose(tf.ctx, prp), assert.Must())
	tf.assertDBCallCount(expectedDBCalls)
	// Loaded from cache
	loadedPurpose, err = tf.storage.GetPurpose(tf.ctx, prp.ID)
	assert.NoErr(t, err)
	tf.assertDBCallCount(expectedDBCalls)
	assert.Equal(t, loadedPurpose, prp)
	assert.Equal(t, loadedPurpose.Description, "Cheapness is not a sense")
}

func TestGetMutatorWithCache(t *testing.T) {
	t.Parallel()

	tf := newTestFixtureForClientCache(t, "MUTATOR")
	mut := &storage.Mutator{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
		Name:                     "NewmanMutator",
		Description:              "The sea was angry that day my friends",
		Version:                  0,
		ColumnIDs:                []uuid.UUID{uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())},
		NormalizerIDs:            []uuid.UUID{uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())},
		AccessPolicyID:           uuid.Must(uuid.NewV4()),
		SelectorConfig:           userstore.UserSelectorConfig{WhereClause: "{newman} = ?"},
	}
	expectedNumFields := tf.getNumFields(reflect.ValueOf(*mut))

	expectedDBCalls := tf.getTotalDBCalls()
	assert.IsNil(t, tf.storage.SaveMutator(tf.ctx, mut), assert.Must())
	expectedKey := fmt.Sprintf("%s_%v", tf.keyPrefix, mut.ID)
	expectedDBCalls++
	tf.assertDBCallCount(expectedDBCalls)
	assert.Equal(t, len(tf.cacheGetJSON(expectedKey)), expectedNumFields)

	// Load from cache
	loadedMutator, err := tf.storage.GetLatestMutator(tf.ctx, mut.ID)
	assert.NoErr(t, err)
	assert.Equal(t, loadedMutator, mut)
	tf.assertDBCallCount(expectedDBCalls)

	// Load from DB and populate cache
	tf.deleteKey(expectedKey)
	expectedDBCalls++
	loadedMutator, err = tf.storage.GetLatestMutator(tf.ctx, mut.ID)
	assert.NoErr(t, err)
	assert.Equal(t, loadedMutator, mut)
	tf.assertDBCallCount(expectedDBCalls)

	mut.Description = "Cheapness is not a sense"
	expectedDBCalls++
	assert.IsNil(t, tf.storage.SaveMutator(tf.ctx, mut), assert.Must())
	tf.assertDBCallCount(expectedDBCalls)
	// Loaded from cache
	loadedMutator, err = tf.storage.GetLatestMutator(tf.ctx, mut.ID)
	assert.NoErr(t, err)
	tf.assertDBCallCount(expectedDBCalls)
	assert.Equal(t, loadedMutator, mut)
	assert.Equal(t, loadedMutator.Description, "Cheapness is not a sense")
}

func initInMemStorage(ctx context.Context, t *testing.T, tdb *ucdb.DB, cacheName string, tenantID uuid.UUID, tombstoneTTL time.Duration) *storage.Storage {
	sharedCache := cachehelpers.NewInMemCache(ctx, t, cacheName, tenantID, tombstoneTTL)
	return storage.NewWithCacheInvalidationWrapper(ctx, tdb, tenantID, sharedCache)

}

func initRedisStorage(ctx context.Context, t *testing.T, tdb *ucdb.DB, cacheName string, tenantID uuid.UUID, redisDB int, tombstoneTTL time.Duration) *storage.Storage {
	sharedCache := cachehelpers.NewRedisCache(ctx, t, cacheName, tenantID, redisDB, tombstoneTTL)
	return storage.NewWithCacheInvalidationWrapper(ctx, tdb, tenantID, sharedCache)
}

type testFixture struct {
	name string
	s1   *storage.Storage
	s2   *storage.Storage
}

func validateValueInCacheByID[item cache.SingleItem](ctx context.Context, t *testing.T, stg *storage.Storage, keyID cache.KeyNameID, id uuid.UUID, expectedValue item) {
	t.Helper()
	cachehelpers.ValidateValueInCacheByID[item](ctx, t, *stg.CacheManager(), keyID, id, expectedValue)
}

func validateCollection[item cache.SingleItem](ctx context.Context, t *testing.T, stg *storage.Storage, id cache.KeyNameID, expectedSize int) {
	t.Helper()
	ckey := stg.CacheManager().N.GetKeyNameStatic(id)
	collection, _, _, _, err := cache.GetItemsArrayFromCache[item](ctx, *stg.CacheManager(), ckey, false)
	assert.NoErr(t, err)
	if expectedSize == 0 {
		assert.IsNil(t, collection, assert.Errorf("Expected collection to be nil, got %+v", collection))
	} else {
		assert.NotNil(t, collection, assert.Must(), assert.Errorf("Expected collection to be non-nil (%d), got nil", expectedSize))
		assert.Equal(t, len(*collection), expectedSize)
	}
}

func validateCollectionWithIsModified[item cache.SingleItem](ctx context.Context, t *testing.T, stg *storage.Storage, id cache.KeyNameID, expectedSize int) {
	t.Helper()
	ckey := stg.CacheManager().N.GetKeyName(id, []string{"", "1500"})
	collection, _, _, _, err := cache.GetItemsArrayFromCache[item](ctx, *stg.CacheManager(), ckey, false)
	assert.NoErr(t, err)
	if expectedSize == 0 {
		assert.IsNil(t, collection, assert.Errorf("Expected collection to be nil, got %+v", collection))
	} else {
		assert.NotNil(t, collection, assert.Must(), assert.Errorf("Expected collection to be non-nil (%d), got nil", expectedSize))
		assert.Equal(t, len(*collection), expectedSize)
	}
}

func newTestFixtureInMemoryCache(ctx context.Context, t *testing.T, tombstoneTTL time.Duration) testFixture {
	tdb := testdb.New(t, migrate.NewTestSchema(tenantdb.Schema))
	cacheName := fmt.Sprintf("UserStoreTestCache-%s", uuid.Must(uuid.NewV4()).String()[:8])
	tenantID := uuid.Must(uuid.NewV4())
	// Create two different storages, each with their own in memory cache.
	return testFixture{
		name: "InMemory",
		s1:   initInMemStorage(ctx, t, tdb, cacheName, tenantID, tombstoneTTL),
		s2:   initInMemStorage(ctx, t, tdb, cacheName, tenantID, tombstoneTTL),
	}
}

func newTestFixtureRedisCache(ctx context.Context, t *testing.T, tombstoneTTL time.Duration) testFixture {
	tdb := testdb.New(t, migrate.NewTestSchema(tenantdb.Schema))
	cacheName := fmt.Sprintf("UserStoreTestCache-%s", uuid.Must(uuid.NewV4()).String()[:8])
	tenantID := uuid.Must(uuid.NewV4())
	// Create two different storages, each with their own redis connection cache.
	return testFixture{
		name: "Redis",
		s1:   initRedisStorage(ctx, t, tdb, cacheName, tenantID, 2, tombstoneTTL),
		s2:   initRedisStorage(ctx, t, tdb, cacheName, tenantID, 3, tombstoneTTL),
	}
}

func newFakeAccessPolicy(name string) *storage.AccessPolicy {
	return &storage.AccessPolicy{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
		Name:                     name,
		Description:              "Festivus for the rest of us",
		PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
	}
}

func newFakeAccessPolicyTemplate(name string) *storage.AccessPolicyTemplate {
	return &storage.AccessPolicyTemplate{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
		Name:                     name,
		Description:              "Hello, Newman",
		Function:                 fmt.Sprintf("function %v() {}", name),
	}
}

func newFakeMutator(name string) *storage.Mutator {
	return &storage.Mutator{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
		Name:                     name,
		Description:              "The sea was angry that day my friends",
		ColumnIDs:                []uuid.UUID{uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())},
		NormalizerIDs:            []uuid.UUID{uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())},
		AccessPolicyID:           uuid.Must(uuid.NewV4()),
		SelectorConfig:           userstore.UserSelectorConfig{WhereClause: "{newman} = ?"},
	}
}

func newFakeAccessor(name string) *storage.Accessor {
	return &storage.Accessor{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
		Name:                     name,
		DataLifeCycleState:       column.DataLifeCycleStateLive,
		Description:              "These pretzels are making me thirsty!",
		ColumnIDs:                []uuid.UUID{uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())},
		TransformerIDs:           []uuid.UUID{uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())},
		TokenAccessPolicyIDs:     []uuid.UUID{uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())},
		AccessPolicyID:           uuid.Must(uuid.NewV4()),
		PurposeIDs:               []uuid.UUID{uuid.Must(uuid.NewV4())},
		SelectorConfig:           userstore.UserSelectorConfig{WhereClause: "{kramer} = ?"},
	}
}

func newFakeColumn(name string) *storage.Column {
	return &storage.Column{
		BaseModel:            ucdb.NewBase(),
		Table:                "users",
		Name:                 name,
		IndexType:            storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeNone),
		DataTypeID:           datatype.String.ID,
		IsArray:              false,
		AccessPolicyID:       policy.AccessPolicyAllowAll.ID,
		DefaultTransformerID: policy.TransformerPassthrough.ID,
	}
}

func newFakeTransformer(name string) *storage.Transformer {
	return &storage.Transformer{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
		Name:                     name,
		InputDataTypeID:          datatype.String.ID,
		OutputDataTypeID:         datatype.Integer.ID,
		TransformType:            storage.InternalTransformTypeFromClient(policy.TransformTypeTransform),
		Function:                 fmt.Sprintf("function %v() {}", name),
		Parameters:               `["jerry", "seinfeld"]`,
	}
}

func TestORMCacheInvalidation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fixtures := []testFixture{
		newTestFixtureInMemoryCache(ctx, t, 0 /* tombstoneTTL */),
		newTestFixtureRedisCache(ctx, t, 0 /* tombstoneTTL */), // this goes into Redis DB 2(s1) and 3(s2)
	}
	testlogtransport.InitLoggerAndTransportsForTests(t)
	for _, fixture := range fixtures {
		t.Run("AccessPolicy-"+fixture.name, func(t *testing.T) {
			ap1 := newFakeAccessPolicy("Festivus")
			ap2 := newFakeAccessPolicy("Chicken")

			// Create an Access Policy in storage 1, ensure it's visible in storage 2.
			assert.NoErr(t, fixture.s1.SaveAccessPolicy(ctx, ap1))
			loadedAP1, err := fixture.s2.GetLatestAccessPolicy(ctx, ap1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, ap1, loadedAP1)

			ap1 = loadedAP1
			// update access policy in storage 1, ensure it's visible in storage 2.
			ap1.Description = "Jambalaya"
			assert.NoErr(t, fixture.s1.SaveAccessPolicy(ctx, ap1))
			loadedAP1, err = fixture.s2.GetLatestAccessPolicy(ctx, ap1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, ap1, loadedAP1)
			assert.Equal(t, loadedAP1.Description, "Jambalaya")
			// Access Policy should be in the cache for s1 from the call to SaveAccessPolicy
			validateValueInCacheByID(ctx, t, fixture.s1, storage.AccessPolicyKeyID, ap1.ID, *ap1)

			// Delete the Access Policy in storage 1, ensure it's gone in storage 2.
			assert.NoErr(t, fixture.s1.DeleteAllAccessPolicyVersions(ctx, ap1.ID))
			_, err = fixture.s2.GetLatestAccessPolicy(ctx, ap1.ID)
			assert.NotNil(t, err)
			_, err = fixture.s2.GetAccessPolicyByName(ctx, "Festivus")
			assert.NotNil(t, err)

			// Create another Access Policy in storage 1, ensure it's visible in storage 2.
			assert.NoErr(t, fixture.s1.SaveAccessPolicy(ctx, ap2))
			loadedAP2, err := fixture.s2.GetLatestAccessPolicy(ctx, ap2.ID)
			assert.NoErr(t, err)
			assert.Equal(t, ap2, loadedAP2)
			loadedAP2, err = fixture.s2.GetAccessPolicyByName(ctx, "Chicken")
			assert.NoErr(t, err)
			assert.Equal(t, ap2, loadedAP2)

			// Delete the Access Policy in storage 1, ensure everything gone in storage 2 due to a flush.
			assert.NoErr(t, fixture.s1.DeleteAllAccessPolicyVersions(ctx, ap2.ID))
			_, err = fixture.s2.GetLatestAccessPolicy(ctx, ap2.ID)
			assert.NotNil(t, err)
			_, err = fixture.s2.GetAccessPolicyByName(ctx, "Chicken")
			assert.NotNil(t, err)
			validateCollection[storage.AccessPolicy](ctx, t, fixture.s1, storage.AccessPolicyCollectionKeyID, 0)
			validateCollection[storage.AccessPolicy](ctx, t, fixture.s2, storage.AccessPolicyCollectionKeyID, 0)

			// // Create three access policies
			ap1 = newFakeAccessPolicy("Newman")
			ap2 = newFakeAccessPolicy("Jerry")
			ap3 := newFakeAccessPolicy("Kramer")
			assert.NoErr(t, fixture.s1.SaveAccessPolicy(ctx, ap1))
			assert.NoErr(t, fixture.s1.SaveAccessPolicy(ctx, ap2))
			validateCollection[storage.AccessPolicy](ctx, t, fixture.s1, storage.AccessPolicyCollectionKeyID, 0)
			validateCollection[storage.AccessPolicy](ctx, t, fixture.s2, storage.AccessPolicyCollectionKeyID, 0)
			loadedPolicies, err := fixture.s2.ListAccessPoliciesNonPaginated(ctx)
			validateCollection[storage.AccessPolicy](ctx, t, fixture.s2, storage.AccessPolicyCollectionKeyID, 2)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedPolicies), 2)
			assert.NoErr(t, fixture.s1.SaveAccessPolicy(ctx, ap3))
			validateCollection[storage.AccessPolicy](ctx, t, fixture.s1, storage.AccessPolicyCollectionKeyID, 0)
			validateCollection[storage.AccessPolicy](ctx, t, fixture.s2, storage.AccessPolicyCollectionKeyID, 0)

			loadedPolicies, err = fixture.s2.ListAccessPoliciesNonPaginated(ctx)
			validateCollection[storage.AccessPolicy](ctx, t, fixture.s2, storage.AccessPolicyCollectionKeyID, 3)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedPolicies), 3)
		})

		t.Run("AccessPolicyTemplate-"+fixture.name, func(t *testing.T) {
			apt1 := newFakeAccessPolicyTemplate("Jerry")
			apt2 := newFakeAccessPolicyTemplate("Kramer")

			// Create an Access Policy Template in storage 1, ensure it's visible in storage 2.
			assert.NoErr(t, fixture.s1.SaveAccessPolicyTemplate(ctx, apt1))
			loadedAPT1, err := fixture.s2.GetLatestAccessPolicyTemplate(ctx, apt1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, apt1, loadedAPT1)

			apt1 = loadedAPT1
			// update access policy template in storage 1, ensure it's visible in storage 2.
			apt1.Description = "Jambalaya"
			assert.NoErr(t, fixture.s1.SaveAccessPolicyTemplate(ctx, apt1))
			loadedAPT1, err = fixture.s2.GetLatestAccessPolicyTemplate(ctx, apt1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, apt1, loadedAPT1)
			assert.Equal(t, loadedAPT1.Description, "Jambalaya")
			// Access Policy Template should be in the cache for s1 from the call to SaveAccessPolicy
			validateValueInCacheByID(ctx, t, fixture.s1, storage.AccessPolicyTemplateKeyID, apt1.ID, *apt1)

			// Delete the Access Policy Template in storage 1, ensure it's gone in storage 2.
			assert.NoErr(t, fixture.s1.DeleteAllAccessPolicyTemplateVersions(ctx, apt1.ID))
			_, err = fixture.s2.GetLatestAccessPolicyTemplate(ctx, apt1.ID)
			assert.NotNil(t, err)
			_, err = fixture.s2.GetAccessPolicyTemplateByName(ctx, "Jerry")
			assert.NotNil(t, err)

			// Create another Access Policy Template in storage 1, ensure it's visible in storage 2.
			assert.NoErr(t, fixture.s1.SaveAccessPolicyTemplate(ctx, apt2))
			loadedAPT2, err := fixture.s2.GetLatestAccessPolicyTemplate(ctx, apt2.ID)
			assert.NoErr(t, err)
			assert.Equal(t, apt2, loadedAPT2)
			loadedAPT2, err = fixture.s2.GetAccessPolicyTemplateByName(ctx, "Kramer")
			assert.NoErr(t, err)
			assert.Equal(t, apt2, loadedAPT2)

			// Delete the Access Policy Template in storage 1, ensure everything gone in storage 2 due to a flush.
			assert.NoErr(t, fixture.s1.DeleteAllAccessPolicyTemplateVersions(ctx, apt2.ID))
			_, err = fixture.s2.GetLatestAccessPolicyTemplate(ctx, apt2.ID)
			assert.NotNil(t, err)
			_, err = fixture.s2.GetAccessPolicyTemplateByName(ctx, "Kramer")
			assert.NotNil(t, err)
			validateCollection[storage.AccessPolicyTemplate](ctx, t, fixture.s1, storage.AccessPolicyTemplateCollectionKeyID, 0)
			validateCollection[storage.AccessPolicyTemplate](ctx, t, fixture.s2, storage.AccessPolicyTemplateCollectionKeyID, 0)

			// Create three access policy templates
			apt1 = newFakeAccessPolicyTemplate("George")
			apt2 = newFakeAccessPolicyTemplate("Kenny")
			apt3 := newFakeAccessPolicyTemplate("Cosmo")
			assert.NoErr(t, fixture.s1.SaveAccessPolicyTemplate(ctx, apt1))
			assert.NoErr(t, fixture.s1.SaveAccessPolicyTemplate(ctx, apt2))
			validateCollection[storage.AccessPolicyTemplate](ctx, t, fixture.s1, storage.AccessPolicyTemplateCollectionKeyID, 0)
			validateCollection[storage.AccessPolicyTemplate](ctx, t, fixture.s2, storage.AccessPolicyTemplateCollectionKeyID, 0)
			loadedTemplates, err := fixture.s2.ListAccessPolicyTemplatesNonPaginated(ctx)
			validateCollection[storage.AccessPolicyTemplate](ctx, t, fixture.s2, storage.AccessPolicyTemplateCollectionKeyID, 2)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedTemplates), 2)
			assert.NoErr(t, fixture.s1.SaveAccessPolicyTemplate(ctx, apt3))
			validateCollection[storage.AccessPolicyTemplate](ctx, t, fixture.s1, storage.AccessPolicyTemplateCollectionKeyID, 0)
			validateCollection[storage.AccessPolicyTemplate](ctx, t, fixture.s2, storage.AccessPolicyTemplateCollectionKeyID, 0)

			loadedTemplates, err = fixture.s2.ListAccessPolicyTemplatesNonPaginated(ctx)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedTemplates), 3)
			validateCollection[storage.AccessPolicyTemplate](ctx, t, fixture.s2, storage.AccessPolicyTemplateCollectionKeyID, 3)
		})

		t.Run("Mutator-"+fixture.name, func(t *testing.T) {
			mt1 := newFakeMutator("Jerry")
			mt2 := newFakeMutator("Kramer")

			// Create an Mutator in storage 1, ensure it's visible in storage 2.
			assert.NoErr(t, fixture.s1.SaveMutator(ctx, mt1))
			loadedMutator1, err := fixture.s2.GetLatestMutator(ctx, mt1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, mt1, loadedMutator1)

			mt1 = loadedMutator1
			// update mutator in storage 1, ensure it's visible in storage 2.
			mt1.Description = "Jambalaya"
			assert.NoErr(t, fixture.s1.SaveMutator(ctx, mt1))
			loadedMutator1, err = fixture.s2.GetLatestMutator(ctx, mt1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, mt1, loadedMutator1)
			assert.Equal(t, loadedMutator1.Description, "Jambalaya")
			// Mutator should be in the cache for s1 from the call to SaveMutator
			validateValueInCacheByID(ctx, t, fixture.s1, storage.MutatorKeyID, mt1.ID, *mt1)

			// Delete the Mutator in storage 1, ensure it's gone in storage 2.
			assert.NoErr(t, fixture.s1.DeleteAllMutatorVersions(ctx, mt1.ID))
			_, err = fixture.s2.GetLatestMutator(ctx, mt1.ID)
			assert.NotNil(t, err)
			_, err = fixture.s2.GetMutatorByName(ctx, "Jerry")
			assert.NotNil(t, err)

			// Create another Mutator in storage 1, ensure it's visible in storage 2.
			assert.NoErr(t, fixture.s1.SaveMutator(ctx, mt2))
			loadedMutator2, err := fixture.s2.GetLatestMutator(ctx, mt2.ID)
			assert.NoErr(t, err)
			assert.Equal(t, mt2, loadedMutator2)
			loadedMutator2, err = fixture.s2.GetMutatorByName(ctx, "Kramer")
			assert.NoErr(t, err)
			assert.Equal(t, mt2, loadedMutator2)

			// Delete the Mutator in storage 1, ensure everything gone in storage 2 due to a flush.
			assert.NoErr(t, fixture.s1.DeleteAllMutatorVersions(ctx, mt2.ID))
			_, err = fixture.s2.GetLatestMutator(ctx, mt2.ID)
			assert.NotNil(t, err)
			_, err = fixture.s2.GetMutatorByName(ctx, "Kramer")
			assert.NotNil(t, err)
			validateCollection[storage.Mutator](ctx, t, fixture.s1, storage.MutatorCollectionKeyID, 0)
			validateCollection[storage.Mutator](ctx, t, fixture.s2, storage.MutatorCollectionKeyID, 0)

			// Create three mutators
			mt1 = newFakeMutator("George")
			mt2 = newFakeMutator("Kenny")
			mt3 := newFakeMutator("Cosmo")
			assert.NoErr(t, fixture.s1.SaveMutator(ctx, mt1))
			assert.NoErr(t, fixture.s1.SaveMutator(ctx, mt2))
			loadedMutators, err := fixture.s2.ListMutatorsNonPaginated(ctx)
			validateCollection[storage.Mutator](ctx, t, fixture.s2, storage.MutatorCollectionKeyID, 2)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedMutators), 2)
			assert.NoErr(t, fixture.s1.SaveMutator(ctx, mt3))
			validateCollection[storage.Mutator](ctx, t, fixture.s1, storage.MutatorCollectionKeyID, 0)
			validateCollection[storage.Mutator](ctx, t, fixture.s2, storage.MutatorCollectionKeyID, 0)

			loadedMutators, err = fixture.s2.ListMutatorsNonPaginated(ctx)
			validateCollection[storage.Mutator](ctx, t, fixture.s2, storage.MutatorCollectionKeyID, 3)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedMutators), 3)
		})

		t.Run("Accessor-"+fixture.name, func(t *testing.T) {
			ac1 := newFakeAccessor("Jerry")
			ac2 := newFakeAccessor("Kramer")

			// Create an Accessor in storage 1, ensure it's visible in storage 2.
			assert.NoErr(t, fixture.s1.SaveAccessor(ctx, ac1))
			loadedAccessor1, err := fixture.s2.GetLatestAccessor(ctx, ac1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, ac1, loadedAccessor1)

			ac1 = loadedAccessor1
			// update accessor in storage 1, ensure it's visible in storage 2.
			ac1.Description = "Jambalaya"
			assert.NoErr(t, fixture.s1.SaveAccessor(ctx, ac1))
			loadedAccessor1, err = fixture.s2.GetLatestAccessor(ctx, ac1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, ac1, loadedAccessor1)
			assert.Equal(t, loadedAccessor1.Description, "Jambalaya")
			// Accessor should be in the cache for s1 from the call to SaveAccessor
			validateValueInCacheByID(ctx, t, fixture.s1, storage.AccessorKeyID, ac1.ID, *ac1)

			// Delete the Accessor in storage 1, ensure it's gone in storage 2.
			assert.NoErr(t, fixture.s1.DeleteAllAccessorVersions(ctx, ac1.ID))
			_, err = fixture.s2.GetLatestAccessor(ctx, ac1.ID)
			assert.NotNil(t, err)
			_, err = fixture.s2.GetAccessorByName(ctx, "Jerry")
			assert.NotNil(t, err)

			// Create another Accessor in storage 1, ensure it's visible in storage 2.
			assert.NoErr(t, fixture.s1.SaveAccessor(ctx, ac2))
			loadedAccessor2, err := fixture.s2.GetLatestAccessor(ctx, ac2.ID)
			assert.NoErr(t, err)
			assert.Equal(t, ac2, loadedAccessor2)
			loadedAccessor2, err = fixture.s2.GetAccessorByName(ctx, "Kramer")
			assert.NoErr(t, err)
			assert.Equal(t, ac2, loadedAccessor2)

			// Delete the Accessor in storage 1, ensure everything gone in storage 2 due to a flush.
			assert.NoErr(t, fixture.s1.DeleteAllAccessorVersions(ctx, ac2.ID))
			_, err = fixture.s2.GetLatestAccessor(ctx, ac2.ID)
			assert.NotNil(t, err)
			_, err = fixture.s2.GetAccessorByName(ctx, "Kramer")
			assert.NotNil(t, err)
			validateCollection[storage.Accessor](ctx, t, fixture.s1, storage.AccessorCollectionKeyID, 0)
			validateCollection[storage.Accessor](ctx, t, fixture.s2, storage.AccessorCollectionKeyID, 0)

			// Create three accessors
			ac1 = newFakeAccessor("George")
			ac2 = newFakeAccessor("Kenny")
			ac3 := newFakeAccessor("Cosmo")
			assert.NoErr(t, fixture.s1.SaveAccessor(ctx, ac1))
			assert.NoErr(t, fixture.s1.SaveAccessor(ctx, ac2))
			loadedAccessors, err := fixture.s2.ListAccessorsNonPaginated(ctx)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedAccessors), 2)
			validateCollection[storage.Accessor](ctx, t, fixture.s2, storage.AccessorCollectionKeyID, 2)

			// Adding an accessor should bust the cache for the accessor collection
			assert.NoErr(t, fixture.s2.SaveAccessor(ctx, ac3))
			validateCollection[storage.Accessor](ctx, t, fixture.s2, storage.AccessorCollectionKeyID, 0)

			validateCollection[storage.Accessor](ctx, t, fixture.s1, storage.AccessorCollectionKeyID, 0) // No global collection cache in s1
			loadedAccessors, err = fixture.s1.ListAccessorsNonPaginated(ctx)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedAccessors), 3)
			validateCollection[storage.Accessor](ctx, t, fixture.s1, storage.AccessorCollectionKeyID, 3)
			ac2.Description = "Chicken"
			// Updating an accessor should bust the cache for the accessor collection
			assert.NoErr(t, fixture.s1.SaveAccessor(ctx, ac2))
			validateCollection[storage.Accessor](ctx, t, fixture.s1, storage.AccessorCollectionKeyID, 0)
		})

		t.Run("Column-"+fixture.name, func(t *testing.T) {
			col1 := newFakeColumn("Jerry")
			col2 := newFakeColumn("Kramer")

			// Create an Column in storage 1, ensure it's visible in storage 2.
			assert.NoErr(t, fixture.s1.SaveColumn(ctx, col1))
			loadedColumn1, err := fixture.s2.GetColumn(ctx, col1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, col1, loadedColumn1)

			col1 = loadedColumn1
			// update column in storage 1, ensure it's visible in storage 2.
			col1.IndexType = storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeIndexed)
			assert.NoErr(t, fixture.s1.SaveColumn(ctx, col1))
			loadedColumn1, err = fixture.s2.GetColumn(ctx, col1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, col1, loadedColumn1)
			assert.Equal(t, loadedColumn1.IndexType, storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeIndexed))
			// Column should be in the cache for s1 from the call to SaveColumn
			validateValueInCacheByID(ctx, t, fixture.s1, storage.ColumnKeyID, col1.ID, *col1)

			// Delete the Column in storage 1, ensure it's gone in storage 2.
			assert.NoErr(t, fixture.s1.DeleteColumn(ctx, col1.ID))
			_, err = fixture.s2.GetColumn(ctx, col1.ID)
			assert.NotNil(t, err)
			_, err = fixture.s2.GetUserColumnByName(ctx, "Jerry")
			assert.NotNil(t, err)

			// Create another Column in storage 1, ensure it's visible in storage 2.
			assert.NoErr(t, fixture.s1.SaveColumn(ctx, col2))
			loadedColumn2, err := fixture.s2.GetColumn(ctx, col2.ID)
			assert.NoErr(t, err)
			assert.Equal(t, col2, loadedColumn2)
			loadedColumn2a, err := fixture.s2.GetUserColumnByName(ctx, "Kramer")
			assert.NoErr(t, err)
			assert.Equal(t, col2, loadedColumn2a)

			// Delete the Column in storage 1, ensure everything gone in storage 2 due to a flush.
			assert.NoErr(t, fixture.s1.DeleteColumn(ctx, col2.ID))
			_, err = fixture.s2.GetColumn(ctx, col2.ID)
			assert.NotNil(t, err)
			_, err = fixture.s2.GetUserColumnByName(ctx, "Kramer")
			assert.NotNil(t, err)
			validateCollectionWithIsModified[storage.Column](ctx, t, fixture.s1, storage.ColumnCollectionPageKeyID, 0)
			validateCollectionWithIsModified[storage.Column](ctx, t, fixture.s2, storage.ColumnCollectionPageKeyID, 0)

			// Create three columns
			col1 = newFakeColumn("George")
			col2 = newFakeColumn("Kenny")
			col3 := newFakeColumn("Cosmo")
			assert.NoErr(t, fixture.s1.SaveColumn(ctx, col1))
			assert.NoErr(t, fixture.s1.SaveColumn(ctx, col2))
			validateCollectionWithIsModified[storage.Column](ctx, t, fixture.s1, storage.ColumnCollectionPageKeyID, 0)
			validateCollectionWithIsModified[storage.Column](ctx, t, fixture.s2, storage.ColumnCollectionPageKeyID, 0)
			loadedColumns, err := listColumnsNonPaginated(ctx, fixture.s2)
			validateCollectionWithIsModified[storage.Column](ctx, t, fixture.s2, storage.ColumnCollectionPageKeyID, 2)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedColumns), 2)
			assert.NoErr(t, fixture.s1.SaveColumn(ctx, col3))
			validateCollectionWithIsModified[storage.Column](ctx, t, fixture.s1, storage.ColumnCollectionPageKeyID, 0)
			validateCollectionWithIsModified[storage.Column](ctx, t, fixture.s2, storage.ColumnCollectionPageKeyID, 0)

			loadedColumns, err = listColumnsNonPaginated(ctx, fixture.s2)
			validateCollectionWithIsModified[storage.Column](ctx, t, fixture.s2, storage.ColumnCollectionPageKeyID, 3)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedColumns), 3)
		})

		t.Run("Transformer-"+fixture.name, func(t *testing.T) {
			tf1 := newFakeTransformer("Jerry")
			tf2 := newFakeTransformer("Kramer")

			// Create an Transformer in storage 1, ensure it's visible in storage 2.
			assert.NoErr(t, fixture.s1.SaveTransformer(ctx, tf1))
			loadedTransformer1, err := fixture.s2.GetLatestTransformer(ctx, tf1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, tf1, loadedTransformer1)

			tf1 = loadedTransformer1
			// update Transformer in storage 1, ensure it's visible in storage 2.
			tf1.Description = "Jambalaya"
			assert.NoErr(t, fixture.s1.SaveTransformer(ctx, tf1))
			loadedTransformer1, err = fixture.s2.GetLatestTransformer(ctx, tf1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, tf1, loadedTransformer1)
			assert.Equal(t, loadedTransformer1.Description, "Jambalaya")
			// Transformer should be in the cache for s1 from the call to SaveTransformer
			validateValueInCacheByID(ctx, t, fixture.s1, storage.TransformerKeyID, tf1.ID, *tf1)

			// Delete the Transformer in storage 1, ensure it's gone in storage 2.
			assert.NoErr(t, fixture.s1.DeleteAllTransformerVersions(ctx, tf1.ID))
			_, err = fixture.s2.GetLatestTransformer(ctx, tf1.ID)
			assert.NotNil(t, err)
			_, err = fixture.s2.GetTransformerByName(ctx, "Jerry")
			assert.NotNil(t, err)

			// Create another Transformer in storage 1, ensure it's visible in storage 2.
			assert.NoErr(t, fixture.s1.SaveTransformer(ctx, tf2))
			loadedTransformer2, err := fixture.s2.GetLatestTransformer(ctx, tf2.ID)
			assert.NoErr(t, err)
			assert.Equal(t, tf2, loadedTransformer2)
			loadedTransformer2, err = fixture.s2.GetTransformerByName(ctx, "Kramer")
			assert.NoErr(t, err)
			assert.Equal(t, tf2, loadedTransformer2)

			// Delete the Transformer in storage 1, ensure everything gone in storage 2 due to a flush.
			assert.NoErr(t, fixture.s1.DeleteAllTransformerVersions(ctx, tf2.ID))
			_, err = fixture.s2.GetLatestTransformer(ctx, tf2.ID)
			assert.NotNil(t, err)
			_, err = fixture.s2.GetTransformerByName(ctx, "Kramer")
			assert.NotNil(t, err)
			validateCollection[storage.Transformer](ctx, t, fixture.s1, storage.TransformerCollectionKeyID, 0)
			validateCollection[storage.Transformer](ctx, t, fixture.s2, storage.TransformerCollectionKeyID, 0)

			// Create three Transformers
			tf1 = newFakeTransformer("George")
			tf2 = newFakeTransformer("Kenny")
			ac3 := newFakeTransformer("Cosmo")
			assert.NoErr(t, fixture.s1.SaveTransformer(ctx, tf1))
			assert.NoErr(t, fixture.s1.SaveTransformer(ctx, tf2))
			loadedTransformers, err := fixture.s2.ListTransformersNonPaginated(ctx)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedTransformers), 2)
			validateCollection[storage.Transformer](ctx, t, fixture.s2, storage.TransformerCollectionKeyID, 2)

			// Adding an Transformer should bust the cache for the Transformer collection
			assert.NoErr(t, fixture.s2.SaveTransformer(ctx, ac3))
			validateCollection[storage.Transformer](ctx, t, fixture.s2, storage.TransformerCollectionKeyID, 0)

			validateCollection[storage.Transformer](ctx, t, fixture.s1, storage.TransformerCollectionKeyID, 0) // No global collection cache in s1
			loadedTransformers, err = fixture.s1.ListTransformersNonPaginated(ctx)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedTransformers), 3)
			validateCollection[storage.Transformer](ctx, t, fixture.s1, storage.TransformerCollectionKeyID, 3)
			tf2.Description = "Chicken"
			// Updating an Transformer should bust the cache for the Transformer collection
			assert.NoErr(t, fixture.s1.SaveTransformer(ctx, tf2))
			validateCollection[storage.Transformer](ctx, t, fixture.s1, storage.TransformerCollectionKeyID, 0)
		})
	}
}

func TestDependencies(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fixtures := []testFixture{
		newTestFixtureInMemoryCache(ctx, t, 0),
		newTestFixtureRedisCache(ctx, t, 0),
	}
	for _, fixture := range fixtures {
		t.Run("AccessorDependencies-"+fixture.name, func(t *testing.T) {
			col1 := newFakeColumn("Newman")
			col2 := newFakeColumn("Cosmo")
			assert.NoErr(t, fixture.s1.SaveColumn(ctx, col1))
			assert.NoErr(t, fixture.s1.SaveColumn(ctx, col2))
			tf1 := newFakeTransformer("Geroge")
			tf2 := newFakeTransformer("Costanza")
			assert.NoErr(t, fixture.s1.SaveTransformer(ctx, tf1))
			assert.NoErr(t, fixture.s1.SaveTransformer(ctx, tf2))
			apt := newFakeAccessPolicyTemplate("Jerry")
			assert.NoErr(t, fixture.s1.SaveAccessPolicyTemplate(ctx, apt))
			ap := newFakeAccessPolicy("Kramer")
			ap.ComponentIDs = []uuid.UUID{apt.ID}
			ap.ComponentParameters = []string{"{}"}
			ap.ComponentTypes = []int32{int32(storage.AccessPolicyComponentTypeTemplate)}
			assert.NoErr(t, fixture.s1.SaveAccessPolicy(ctx, ap))
			accessor := newFakeAccessor("Newman")
			accessor.AccessPolicyID = ap.ID
			accessor.ColumnIDs = []uuid.UUID{col1.ID, col2.ID}
			accessor.TransformerIDs = []uuid.UUID{tf1.ID, tf2.ID}
			accessor.TokenAccessPolicyIDs = []uuid.UUID{uuid.Nil, uuid.Nil}
			assert.NoErr(t, fixture.s1.SaveAccessor(ctx, accessor))

			// Try to delete objects, should fail due to dependencies
			assert.ErrorIs(t, fixture.s1.DeleteAllAccessPolicyTemplateVersions(ctx, apt.ID), storage.ErrStillInUse)
			assert.ErrorIs(t, fixture.s1.DeleteAllAccessPolicyVersions(ctx, apt.ID), storage.ErrStillInUse)
			// We don't actually have protections in our code for those, so they work we can delete those objects while they are referenced by the mutator
			// assert.ErrorIs(t, fixture.s1.DeleteColumn(ctx, col1.ID), ErrStillInUse)
			// assert.ErrorIs(t, fixture.s1.DeleteColumn(ctx, col2.ID), ErrStillInUse)
			// assert.ErrorIs(t, fixture.s1.DeleteTransformer(ctx, tf1.ID), ErrStillInUse)
			// assert.ErrorIs(t, fixture.s1.DeleteTransformer(ctx, tf2.ID), ErrStillInUse)

			// Delete Accessor
			assert.NoErr(t, fixture.s1.DeleteAllAccessorVersions(ctx, accessor.ID))
			// this should still fail
			assert.ErrorIs(t, fixture.s1.DeleteAllAccessPolicyTemplateVersions(ctx, apt.ID), storage.ErrStillInUse)
			// Delete objects
			assert.NoErr(t, fixture.s1.DeleteAllAccessPolicyVersions(ctx, ap.ID))
			assert.NoErr(t, fixture.s1.DeleteAllAccessPolicyTemplateVersions(ctx, apt.ID))
			assert.NoErr(t, fixture.s1.DeleteColumn(ctx, col1.ID))
			assert.NoErr(t, fixture.s1.DeleteColumn(ctx, col2.ID))
			assert.NoErr(t, fixture.s1.DeleteAllTransformerVersions(ctx, tf1.ID))
			assert.NoErr(t, fixture.s1.DeleteAllTransformerVersions(ctx, tf2.ID))
		})

		t.Run("MutatorDependencies-"+fixture.name, func(t *testing.T) {
			col1 := newFakeColumn("Kenny")
			col2 := newFakeColumn("Bania")
			assert.NoErr(t, fixture.s1.SaveColumn(ctx, col1))
			assert.NoErr(t, fixture.s1.SaveColumn(ctx, col2))
			apt := newFakeAccessPolicyTemplate("Seinfeld")
			assert.NoErr(t, fixture.s1.SaveAccessPolicyTemplate(ctx, apt))
			ap := newFakeAccessPolicy("Kramer")
			ap.ComponentIDs = []uuid.UUID{apt.ID}
			ap.ComponentParameters = []string{"{}"}
			ap.ComponentTypes = []int32{int32(storage.AccessPolicyComponentTypeTemplate)}
			assert.NoErr(t, fixture.s1.SaveAccessPolicy(ctx, ap))
			mutator := newFakeMutator("Joe")

			mutator.AccessPolicyID = ap.ID
			mutator.ColumnIDs = []uuid.UUID{col1.ID, col2.ID}
			mutator.NormalizerIDs = []uuid.UUID{policy.TransformerPassthrough.ID, policy.TransformerPassthrough.ID}
			assert.NoErr(t, fixture.s1.SaveMutator(ctx, mutator))

			// Try to delete objects, should fail due to dependencies
			assert.ErrorIs(t, fixture.s1.DeleteAllAccessPolicyTemplateVersions(ctx, apt.ID), storage.ErrStillInUse)
			assert.ErrorIs(t, fixture.s1.DeleteAllAccessPolicyVersions(ctx, apt.ID), storage.ErrStillInUse)
			// We don't actually have protections in our code for those, so they work we can delete those objects while they are referenced by the mutator
			// assert.ErrorIs(t, fixture.s1.DeleteColumn(ctx, col1.ID), ErrStillInUse)
			// assert.ErrorIs(t, fixture.s1.DeleteColumn(ctx, col2.ID), ErrStillInUse)

			// Delete Mutator
			assert.NoErr(t, fixture.s1.DeleteAllMutatorVersions(ctx, mutator.ID))
			// this should still fail
			assert.ErrorIs(t, fixture.s1.DeleteAllAccessPolicyTemplateVersions(ctx, apt.ID), storage.ErrStillInUse)
			// Delete objects
			assert.NoErr(t, fixture.s1.DeleteAllAccessPolicyVersions(ctx, ap.ID))
			assert.NoErr(t, fixture.s1.DeleteAllAccessPolicyTemplateVersions(ctx, apt.ID))
			assert.NoErr(t, fixture.s1.DeleteColumn(ctx, col1.ID))
			assert.NoErr(t, fixture.s1.DeleteColumn(ctx, col2.ID))
		})
	}
}

func TestGlobalCollectionPageCacheInvalidation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fixtures := []testFixture{
		newTestFixtureInMemoryCache(ctx, t, 1*time.Second /* tombstoneTTL */),
		newTestFixtureRedisCache(ctx, t, 1*time.Second /* tombstoneTTL */), // this goes into Redis DB 2(s1) and 3(s2)
	}
	for _, fixture := range fixtures {
		t.Run("Column-"+fixture.name, func(t *testing.T) {
			col1 := newFakeColumn("Jerry")
			col2 := newFakeColumn("Kramer")

			// Create an Column in storage 1, ensure it's visible in storage 2.
			assert.NoErr(t, fixture.s1.SaveColumn(ctx, col1))
			loadedColumn1, err := fixture.s2.GetColumn(ctx, col1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, col1, loadedColumn1)

			col1 = loadedColumn1
			// update column in storage 1, ensure it's visible in storage 2.
			col1.IndexType = storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeIndexed)
			assert.NoErr(t, fixture.s1.SaveColumn(ctx, col1))
			loadedColumn1, err = fixture.s2.GetColumn(ctx, col1.ID)
			assert.NoErr(t, err)
			assert.Equal(t, col1, loadedColumn1)
			assert.Equal(t, loadedColumn1.IndexType, storage.ColumnIndexTypeFromClient(userstore.ColumnIndexTypeIndexed))
			// Column should be in the cache for s1 from the call to SaveColumn
			validateValueInCacheByID(ctx, t, fixture.s1, storage.ColumnKeyID, col1.ID, *col1)

			// Create another Column in storage 1, ensure it's visible in storage 2.
			assert.NoErr(t, fixture.s1.SaveColumn(ctx, col2))
			loadedColumn2, err := fixture.s2.GetColumn(ctx, col2.ID)
			assert.NoErr(t, err)
			assert.Equal(t, col2, loadedColumn2)
			loadedColumn2a, err := fixture.s2.GetUserColumnByName(ctx, "Kramer")
			assert.NoErr(t, err)
			assert.Equal(t, col2, loadedColumn2a)

			// Baseline that cache is empty
			validateCollectionWithIsModified[storage.Column](ctx, t, fixture.s1, storage.ColumnCollectionPageKeyID, 0)
			validateCollectionWithIsModified[storage.Column](ctx, t, fixture.s2, storage.ColumnCollectionPageKeyID, 0)

			// Delete the Column in storage 1, ensure it's gone in storage 2.
			assert.NoErr(t, fixture.s1.DeleteColumn(ctx, col1.ID))
			_, err = fixture.s2.GetColumn(ctx, col1.ID)
			assert.NotNil(t, err)
			_, err = fixture.s2.GetUserColumnByName(ctx, "Jerry")
			assert.NotNil(t, err)

			loadedColumns1, err := listColumnsNonPaginated(ctx, fixture.s1)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedColumns1), 1)
			loadedColumns2, err := listColumnsNonPaginated(ctx, fixture.s2)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedColumns2), 1)
			// Shouldn't have cached the values due to the tombstone
			validateCollectionWithIsModified[storage.Column](ctx, t, fixture.s1, storage.ColumnCollectionPageKeyID, 0)
			validateCollectionWithIsModified[storage.Column](ctx, t, fixture.s2, storage.ColumnCollectionPageKeyID, 0)

			// Wait for the tombstone to expire
			time.Sleep(1 * time.Second)

			loadedColumns1, err = listColumnsNonPaginated(ctx, fixture.s1)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedColumns1), 1)
			loadedColumns2, err = listColumnsNonPaginated(ctx, fixture.s2)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedColumns2), 1)

			// Should be cached now
			validateCollectionWithIsModified[storage.Column](ctx, t, fixture.s1, storage.ColumnCollectionPageKeyID, 1)
			validateCollectionWithIsModified[storage.Column](ctx, t, fixture.s2, storage.ColumnCollectionPageKeyID, 1)

			// Create another two columns
			col1 = newFakeColumn("George")
			col2 = newFakeColumn("Kenny")
			assert.NoErr(t, fixture.s1.SaveColumn(ctx, col1))
			assert.NoErr(t, fixture.s1.SaveColumn(ctx, col2))

			loadedColumns1, err = listColumnsNonPaginated(ctx, fixture.s1)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedColumns1), 3)
			loadedColumns2, err = listColumnsNonPaginated(ctx, fixture.s2)
			assert.NoErr(t, err)
			assert.Equal(t, len(loadedColumns2), 3)
			// Should be cached without having to wait for tombstone to expire
			validateCollectionWithIsModified[storage.Column](ctx, t, fixture.s1, storage.ColumnCollectionPageKeyID, 3)
			validateCollectionWithIsModified[storage.Column](ctx, t, fixture.s2, storage.ColumnCollectionPageKeyID, 3)
		})
	}
}

// listColumnsNonPaginated is a test helper that loads columns using pagination
func listColumnsNonPaginated(ctx context.Context, s *storage.Storage) ([]storage.Column, error) {
	pager, err := storage.NewColumnPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	objs := make([]storage.Column, 0)
	pageCount := 0
	for {
		objRead, respFields, err := s.ListColumnsPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		objs = append(objs, objRead...)
		pageCount++
		if !pager.AdvanceCursor(*respFields) {
			break
		}
		if pageCount >= 10 {
			return nil, ucerr.Errorf("listColumnsNonPaginated exceeded max page count of 10")
		}
	}
	return objs, nil
}

func TestKeyNames(t *testing.T) {
	np := storage.NewCacheNameProviderForTenant(uuid.Must(uuid.NewV4()))
	keyIDs := np.GetAllKeyIDs()
	for _, keyID := range keyIDs {
		assert.True(t, len(keyID) > 3)
		key := np.GetKeyName(cache.KeyNameID(keyID), []string{"a", "b", "c"})
		assert.True(t, len(key) > 3)
	}
}
