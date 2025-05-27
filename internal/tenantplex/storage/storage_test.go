package storage_test

import (
	"context"
	"sync"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucdb"
	"userclouds.com/internal/companyconfig"
	tenantProvisioning "userclouds.com/internal/provisioning/tenant"
	"userclouds.com/internal/tenantdb"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/internal/tenantplex/storage"
	"userclouds.com/internal/testkeys"
)

func initRedisStorage(ctx context.Context, tdb *ucdb.DB) *storage.Storage {
	return storage.NewForTests(ctx, tdb, testhelpers.NewCacheConfig())
}

func createTenantPlex(t *testing.T, ctx context.Context, tenant *companyconfig.Tenant) *tenantplex.TenantPlex {
	_, tc, err := tenantProvisioning.CreatePlexConfig(ctx, tenant, tenantProvisioning.UseKeys(&testkeys.Config))
	assert.NoErr(t, err)
	tc.Keys = testkeys.Config
	tenantPlex := &tenantplex.TenantPlex{
		VersionBaseModel: ucdb.NewVersionBaseWithID(tenant.ID),
		PlexConfig:       *tc,
	}
	return tenantPlex
}

func TestORMCacheInvalidation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tdb := testdb.New(t, migrate.NewTestSchema(tenantdb.Schema))

	// Create two different storages, each with their own in memory cache.
	s1 := initRedisStorage(ctx, tdb)
	s2 := initRedisStorage(ctx, tdb)
	ten := &companyconfig.Tenant{
		BaseModel: ucdb.NewBase(),
		Name:      uuid.Must(uuid.NewV4()).String()[0:10],
		CompanyID: uuid.Must(uuid.NewV4()),
		TenantURL: "https://test" + uuid.Must(uuid.NewV4()).String() + ".com",
	}

	// Create a tenantPlex and two tenants
	tenantPlex := createTenantPlex(t, ctx, ten)
	assert.NoErr(t, s1.SaveTenantPlex(ctx, tenantPlex))

	outT, err := s1.GetTenantPlex(ctx, tenantPlex.ID)
	assert.NoErr(t, err)
	assert.Equal(t, cmp.Equal(outT, tenantPlex, cmp.AllowUnexported(secret.String{})), true)
	outT, err = s2.GetTenantPlex(ctx, tenantPlex.ID)
	assert.NoErr(t, err)
	assert.Equal(t, cmp.Equal(outT, tenantPlex, cmp.AllowUnexported(secret.String{})), true)

	// Delete the tenant and validate that all collections are cleared
	assert.NoErr(t, s1.DeleteTenantPlex(ctx, tenantPlex.ID))

	_, err = s1.GetTenantPlex(ctx, tenantPlex.ID)
	assert.NotNil(t, err)

	// Create another tenant and validation the collections are updated
	tenantPlex = createTenantPlex(t, ctx, ten)
	assert.NoErr(t, s1.SaveTenantPlex(ctx, tenantPlex))

	outT, err = s1.GetTenantPlex(ctx, tenantPlex.ID)
	assert.NoErr(t, err)
	assert.Equal(t, cmp.Equal(outT, tenantPlex, cmp.AllowUnexported(secret.String{})), true)
	outT, err = s2.GetTenantPlex(ctx, tenantPlex.ID)
	assert.NoErr(t, err)
	assert.Equal(t, cmp.Equal(outT, tenantPlex, cmp.AllowUnexported(secret.String{})), true)

	// Now try a multithreaded test
	threadCount := 10
	opCount := 2
	wg := sync.WaitGroup{}
	// 	Have threadCount threads try to release key with non matching sentinel
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()

			localTenants := []tenantplex.TenantPlex{}
			var localerr error
			for j := range opCount {
				// Create another tenant and validation the collections are updated
				loclTen := &companyconfig.Tenant{
					BaseModel: ucdb.NewBase(),
					Name:      uuid.Must(uuid.NewV4()).String()[0:10],
					CompanyID: uuid.Must(uuid.NewV4()),
					TenantURL: "https://test" + uuid.Must(uuid.NewV4()).String() + ".com",
				}

				localTenPlex := createTenantPlex(t, ctx, loclTen)
				if j%2 == 0 {
					localerr = s1.SaveTenantPlex(ctx, localTenPlex)
				} else {
					localerr = s2.SaveTenantPlex(ctx, localTenPlex)
				}
				assert.NoErr(t, localerr)
				localTenants = append(localTenants, *localTenPlex)
			}

			localerr = s1.DeleteTenantPlex(ctx, localTenants[1].ID)
			assert.NoErr(t, localerr)

			outTLocal, err := s1.GetTenantPlex(ctx, localTenants[0].ID)
			assert.NoErr(t, err)

			assert.Equal(t, cmp.Equal(*outTLocal, localTenants[0], cmp.AllowUnexported(secret.String{})), true)
			outTLocal, err = s2.GetTenantPlex(ctx, localTenants[0].ID)
			assert.NoErr(t, err)
			assert.Equal(t, cmp.Equal(*outTLocal, localTenants[0], cmp.AllowUnexported(secret.String{})), true)
		}(i)
	}
	wg.Wait()
}
