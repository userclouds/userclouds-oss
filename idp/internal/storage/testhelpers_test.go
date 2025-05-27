package storage_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/cache"
	cacheMetrics "userclouds.com/infra/cache/metrics"
	cacheTestHelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/request"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucdb"
	dbMetrics "userclouds.com/infra/ucdb/metrics"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantdb"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
)

type storageTestFixture struct {
	ctx    context.Context
	s      *storage.Storage
	db     *ucdb.DB
	tenant *companyconfig.Tenant
}

func newStorageForTests(t *testing.T) *storageTestFixture {
	t.Helper()
	ctx := context.Background()

	_, tenant, _, tdb, _, tm := testhelpers.CreateTestServer(ctx, t)
	req, err := http.NewRequest("GET", "/test", nil)
	assert.NoErr(t, err)

	jwt := uctest.CreateJWT(t, oidc.UCTokenClaims{}, tenant.TenantURL)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", jwt))
	ts, err := tm.GetTenantStateForID(ctx, tenant.ID)
	assert.NoErr(t, err)
	ctx = request.SetRequestData(multitenant.SetTenantState(ctx, ts), req, uuid.Nil)
	return &storageTestFixture{
		ctx:    ctx,
		s:      storage.New(ctx, tdb, tenant.ID, nil),
		db:     tdb,
		tenant: tenant,
	}
}

type clientCacheTestFixture struct {
	ctx       context.Context
	t         *testing.T
	storage   *storage.Storage
	tenantID  uuid.UUID
	keyPrefix string
	redis     *redis.Client
}

func (tf *clientCacheTestFixture) getTotalDBCalls() int {
	tf.t.Helper()
	dbm, err := dbMetrics.GetMetrics(tf.ctx)
	assert.NoErr(tf.t, err)
	return dbm.GetTotalCalls()
}

func (tf *clientCacheTestFixture) assertDBCallCount(expected int) {
	tf.t.Helper()
	assert.Equal(tf.t, tf.getTotalDBCalls(), expected)
}

func (tf *clientCacheTestFixture) cacheGet(key string) []byte {
	tf.t.Helper()
	value, err := tf.redis.Get(tf.ctx, key).Result()
	assert.NoErr(tf.t, err)
	assert.NotEqual(tf.t, value, "")
	return []byte(value)
}
func (tf *clientCacheTestFixture) cacheGetJSON(key string) map[string]any {
	tf.t.Helper()
	data := tf.cacheGet(key)
	var payload map[string]any
	err := json.Unmarshal(data, &payload)
	assert.NoErr(tf.t, err)
	return payload
}

func (tf *clientCacheTestFixture) deleteKey(key string) {
	tf.t.Helper()

	// Need to clear the key from in memory cache as well without changing the metrics :)
	cm, err := storage.GetCacheManager(tf.ctx, cacheTestHelpers.NewRedisConfigForTests(), tf.tenantID)
	assert.NoErr(tf.t, err)
	if cm.Provider.Layered(tf.ctx) {
		assert.NoErr(tf.t, cm.Provider.Flush(tf.ctx, key, true))
	}
	assert.NoErr(tf.t, tf.redis.Del(tf.ctx, key).Err())
}

func (tf clientCacheTestFixture) getNumFields(v reflect.Value) int {
	if v.Kind() != reflect.Struct {
		return 0
	}

	var numFields int
	for i := range v.NumField() {
		if v.Type().Field(i).Anonymous {
			numFields += tf.getNumFields(v.Field(i))
		} else {
			numFields++
		}
	}

	return numFields
}

func newTestFixtureForClientCache(t *testing.T, cacheKeyIDBase string) *clientCacheTestFixture {
	tenantID := uuid.Must(uuid.NewV4())
	ctx := cacheMetrics.InitContext(dbMetrics.InitContext(context.Background()))
	return &clientCacheTestFixture{
		storage:   newStorageWithRedisCache(ctx, t, tenantID),
		ctx:       ctx,
		tenantID:  tenantID,
		keyPrefix: fmt.Sprintf("userstore_%v_%v", tenantID, cacheKeyIDBase),
		t:         t,
		redis:     cache.NewLocalRedisClient(),
	}
}

func newStorageWithRedisCache(ctx context.Context, t testing.TB, tenantID uuid.UUID) *storage.Storage {
	tdb := testdb.New(t, migrate.NewTestSchema(tenantdb.Schema))
	return storage.New(ctx, tdb, tenantID, cacheTestHelpers.NewRedisConfigForTests())
}
