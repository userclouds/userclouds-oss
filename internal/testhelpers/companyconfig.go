package testhelpers

import (
	"context"
	"sync"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucdb"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/logdb"
	"userclouds.com/internal/tenantmap"
)

var cacheDBsOnce sync.Once
var cdbConfig *ucdb.Config
var ldbConfig *ucdb.Config
var ccs *companyconfig.Storage

// NewTestStorage creates a new CompanyConfig and LogServer DB, and returns
// a triple: (company config DB cfg, logserver DB cfg, company config storage).
func NewTestStorage(t *testing.T) (companyDBConfig *ucdb.Config, logDBConfig *ucdb.Config, storage *companyconfig.Storage) {
	cacheDBsOnce.Do(func() {
		cdbConfig, ldbConfig, ccs = NewUniqueTestStorage(t)
		_, err := ldbConfig.Password.Resolve(context.Background())
		assert.NoErr(t, err, assert.Errorf("failed to resolve password for logdb"))
	})

	return cdbConfig, ldbConfig, ccs
}

// NewUniqueTestStorage is an expensive way to create a totally isolated test DB setup
func NewUniqueTestStorage(t *testing.T) (*ucdb.Config, *ucdb.Config, *companyconfig.Storage) {
	ctx := context.Background()

	cdb := testdb.New(t, migrate.NewTestSchema(companyconfig.Schema))
	c := testdb.TestConfig(t, cdb)

	ccs, err := companyconfig.NewStorage(ctx, cdb, testhelpers.NewCacheConfig())
	assert.NoErr(t, err)

	ldb := testdb.New(t, migrate.NewTestMigrator(logdb.GetMigrations()))
	l := testdb.TestConfig(t, ldb)

	return &c, &l, ccs
}

// NewTestTenantStateMap returns a tenants state map to be used in tests
func NewTestTenantStateMap(companyConfigStorage *companyconfig.Storage) *tenantmap.StateMap {
	return tenantmap.NewStateMap(companyConfigStorage, testhelpers.NewRedisConfigForTests())
}
