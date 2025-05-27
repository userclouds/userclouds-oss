package companyconfig_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp/cmpopts"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/cache"
	cachehelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucdb"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/testhelpers"
)

func TestTenantInvalidName(t *testing.T) {
	ctx := context.Background()
	_, _, store := testhelpers.NewTestStorage(t)
	err := store.SaveTenant(ctx, testhelpers.NewNamedTenant("j"))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Tenant.Name length has to be between 2 and 30 (length: 1)")
}

func TestTI(t *testing.T) {
	fakeDBConfig := ucdb.Config{
		User:      "fake",
		DBName:    "fake",
		DBDriver:  ucdb.PostgresDriver,
		DBProduct: ucdb.Postgres,
		Host:      "anytown",
		Port:      "1234",
	}
	ctx := context.Background()
	_, _, store := testhelpers.NewTestStorage(t)

	ti := &companyconfig.TenantInternal{
		BaseModel:         ucdb.NewBase(),
		TenantDBConfig:    fakeDBConfig,
		PrimaryUserRegion: region.DefaultUserDataRegionForUniverse(universe.Test),
		LogConfig: companyconfig.TenantLogConfig{
			LogDB: fakeDBConfig,
		},
	}

	assert.IsNil(t, store.SaveTenantInternal(ctx, ti), assert.Must())

	gotTIC, err := store.GetTenantInternal(ctx, ti.ID)
	assert.NoErr(t, err)
	assert.Equal(t, gotTIC, ti, assert.CmpOpt(cmpopts.IgnoreUnexported(secret.String{})))
}

func TestTenantDomainSoftDeleteUnique(t *testing.T) {
	ctx := context.Background()
	od, ld, store := testhelpers.NewTestStorage(t)
	company := testhelpers.ProvisionTestCompanyWithoutACL(ctx, t, store)
	te, _ := testhelpers.ProvisionTestTenant(ctx, t, store, od, ld, company.ID)
	assert.IsNil(t, store.SaveTenant(ctx, te))
	assert.IsNil(t, store.DeleteTenant(ctx, te.ID))
	assert.IsNil(t, store.SaveTenant(ctx, te))
}

func TestHasTenants(t *testing.T) {
	ctx := context.Background()
	occ, lcc, s := testhelpers.NewUniqueTestStorage(t)

	has, err := s.HasAnyTenants(ctx)
	assert.NoErr(t, err)
	assert.False(t, has)

	testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, s, occ, lcc)

	has, err = s.HasAnyTenants(ctx)
	assert.NoErr(t, err)
	assert.True(t, has)
}

func TestDeleteTenantURLs(t *testing.T) {
	ctx := context.Background()
	_, _, s := testhelpers.NewTestStorage(t)

	ten := testhelpers.NewTenant()
	assert.NoErr(t, s.SaveTenant(ctx, ten))
	tu := companyconfig.TenantURL{
		BaseModel: ucdb.NewBase(),
		TenantID:  ten.ID,
		TenantURL: "https://test.com",
	}

	assert.NoErr(t, s.SaveTenantURL(ctx, &tu))
	tuc, err := s.GetTenantURL(ctx, tu.ID)
	assert.NoErr(t, err)
	assert.Equal(t, tuc, &tu)
	// Hit the cache
	tuc, err = s.GetTenantURL(ctx, tu.ID)
	assert.NoErr(t, err)
	assert.Equal(t, tuc, &tu)

	assert.NoErr(t, s.DeleteTenant(ctx, ten.ID))
	got, err := s.GetTenantURL(ctx, tu.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.IsNil(t, got)
}

func TestInvalidRedisConfig(t *testing.T) {
	ctx := context.Background()
	companyDB := testdb.New(t, migrate.NewTestSchema(companyconfig.Schema))
	dbCFg := testdb.TestConfig(t, companyDB)
	ocStorage, err := companyconfig.NewStorageFromConfig(ctx, &dbCFg,
		&cache.Config{
			RedisCacheConfig: []cache.RegionalRedisConfig{
				{
					RedisConfig: cache.RedisConfig{
						Host:   "fakehost",
						Port:   6379,
						DBName: 0,
					},
					Region: region.Current(),
				}}})
	assert.NoErr(t, err)
	assert.NotNil(t, ocStorage)
}

// this test exists here just because we need a place to test genorm logic
// without having to codegen a full fake ORM for a test
func TestDelete(t *testing.T) {
	ctx := context.Background()
	_, _, s := testhelpers.NewTestStorage(t)
	ten := &companyconfig.Tenant{
		BaseModel: ucdb.NewBase(),
		Name:      "tt",
		CompanyID: uuid.Must(uuid.NewV4()),
		TenantURL: "https://test.com",
	}
	assert.NoErr(t, s.SaveTenant(ctx, ten))

	// first delete should work
	assert.NoErr(t, s.DeleteTenant(ctx, ten.ID))

	got, err := s.GetTenantSoftDeleted(ctx, ten.ID)
	assert.NoErr(t, err)
	ten.Deleted = got.Deleted // need to set this, obviously, before we compare them
	assert.Equal(t, got, ten)

	// should fail to delete after already deleted
	assert.NotNil(t, s.DeleteTenant(ctx, ten.ID))
}

func initInMemStorage(ctx context.Context, tdb *ucdb.DB) (*companyconfig.Storage, error) {
	return companyconfig.NewStorage(ctx, tdb, cachehelpers.NewCacheConfig())
}

type IDAble interface {
	GetID() uuid.UUID
}

func GetID(i any) uuid.UUID {
	if t, ok := i.(*companyconfig.Tenant); ok {
		return t.ID
	} else if url, ok := i.(*companyconfig.TenantURL); ok {
		return url.ID
	}

	return uuid.Nil
}

func validateExpectedItemsCollection[item any](t *testing.T, tenants []item, expected []item) {
	assert.Equal(t, len(tenants), len(expected))
	resultMap := map[uuid.UUID]item{}
	for i := range tenants {
		resultMap[GetID(&tenants[i])] = tenants[i]
	}
	for i := range expected {
		assert.Equal(t, resultMap[GetID(&expected[i])], expected[i])
	}
}

func validateExpectedTenants(t *testing.T, ctx context.Context, s1 *companyconfig.Storage, s2 *companyconfig.Storage, company *companyconfig.Company, expected []companyconfig.Tenant) {
	tenants, err := s1.ListTenantsForCompany(ctx, company.ID)
	assert.NoErr(t, err)
	validateExpectedItemsCollection(t, tenants, expected)
	tenants, err = s2.ListTenantsForCompany(ctx, company.ID)
	assert.NoErr(t, err)
	validateExpectedItemsCollection(t, tenants, expected)
	pager, err := companyconfig.NewTenantPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	assert.NoErr(t, err)
	tenants, _, err = s1.ListTenantsPaginated(ctx, *pager)
	assert.NoErr(t, err)
	validateExpectedItemsCollection(t, tenants, expected)
	tenants, _, err = s2.ListTenantsPaginated(ctx, *pager)
	assert.NoErr(t, err)
	validateExpectedItemsCollection(t, tenants, expected)
}

func validateExpectedTenantsURLs(t *testing.T, ctx context.Context, s1 *companyconfig.Storage, s2 *companyconfig.Storage, tenant *companyconfig.Tenant, expected []companyconfig.TenantURL) {
	tenantURLs, err := s1.ListTenantURLsForTenant(ctx, tenant.ID)
	assert.NoErr(t, err)
	validateExpectedItemsCollection(t, tenantURLs, expected)
	tenantURLs, err = s2.ListTenantURLsForTenant(ctx, tenant.ID)
	assert.NoErr(t, err)
	validateExpectedItemsCollection(t, tenantURLs, expected)
	pager, err := companyconfig.NewTenantURLPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	assert.NoErr(t, err)
	tenantURLs, _, err = s1.ListTenantURLsPaginated(ctx, *pager)
	assert.NoErr(t, err)
	validateExpectedItemsCollection(t, tenantURLs, expected)
	tenantURLs, _, err = s2.ListTenantURLsPaginated(ctx, *pager)
	assert.NoErr(t, err)
	validateExpectedItemsCollection(t, tenantURLs, expected)
}

func TestORMCacheInvalidation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tdb := testdb.New(t, migrate.NewTestSchema(companyconfig.Schema))

	// Create two different storages, each with their own in memory cache.
	s1, err := initInMemStorage(ctx, tdb)
	assert.NoErr(t, err)
	s2, err := initInMemStorage(ctx, tdb)
	assert.NoErr(t, err)

	// Create a company and two tenants
	company := companyconfig.NewCompany(uuid.Must(uuid.NewV4()).String(), companyconfig.CompanyTypeCustomer)
	err = s1.SaveCompany(ctx, &company)
	assert.NoErr(t, err)
	ten := &companyconfig.Tenant{
		BaseModel: ucdb.NewBase(),
		Name:      uuid.Must(uuid.NewV4()).String()[0:10],
		CompanyID: company.ID,
		TenantURL: "https://test" + uuid.Must(uuid.NewV4()).String() + ".com",
	}
	err = s2.SaveTenant(ctx, ten)
	assert.NoErr(t, err)
	validateExpectedTenants(t, ctx, s1, s2, &company, []companyconfig.Tenant{*ten})

	outT, err := s1.GetTenant(ctx, ten.ID)
	assert.NoErr(t, err)
	assert.Equal(t, outT, ten)
	outT, err = s1.GetTenantByHost(ctx, ten.GetHostName())
	assert.NoErr(t, err)
	assert.Equal(t, outT, ten)

	// Delete the tenant and validate that all collections are cleared
	err = s1.DeleteTenant(ctx, ten.ID)
	assert.NoErr(t, err)
	validateExpectedTenants(t, ctx, s1, s2, &company, []companyconfig.Tenant{})

	_, err = s1.GetTenant(ctx, ten.ID)
	assert.NotNil(t, err)
	_, err = s1.GetTenantByHost(ctx, ten.GetHostName())
	assert.NotNil(t, err)

	// Create another tenant and validation the collections are updated
	ten = &companyconfig.Tenant{
		BaseModel: ucdb.NewBase(),
		Name:      uuid.Must(uuid.NewV4()).String()[0:10],
		CompanyID: company.ID,
		TenantURL: "https://test" + uuid.Must(uuid.NewV4()).String() + ".com",
	}
	err = s1.SaveTenant(ctx, ten)
	assert.NoErr(t, err)
	validateExpectedTenants(t, ctx, s1, s2, &company, []companyconfig.Tenant{*ten})

	// Now try a multithreaded test
	threadCount := 10
	opCount := 2
	tenantsExpected := []companyconfig.Tenant{}
	tenantsExpected = append(tenantsExpected, *ten)
	tenantLock := sync.Mutex{}
	wg := sync.WaitGroup{}
	// 	Have threadCount threads try to release key with non matching sentinel
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()

			localTenants := []companyconfig.Tenant{}
			var localerr error
			for j := range opCount {
				// Create another tenant and validation the collections are updated
				localTen := &companyconfig.Tenant{
					BaseModel: ucdb.NewBase(),
					Name:      uuid.Must(uuid.NewV4()).String()[0:10],
					CompanyID: company.ID,
					TenantURL: "https://test" + uuid.Must(uuid.NewV4()).String() + ".com",
				}
				if j%2 == 0 {
					localerr = s1.SaveTenant(ctx, localTen)
				} else {
					localerr = s2.SaveTenant(ctx, localTen)
				}
				assert.NoErr(t, localerr)
				localTenants = append(localTenants, *localTen)
			}

			localerr = s1.DeleteTenant(ctx, localTenants[1].ID)
			assert.NoErr(t, localerr)

			tenantLock.Lock()
			tenantsExpected = append(tenantsExpected, localTenants[0])
			tenantLock.Unlock()
		}(i)
	}
	wg.Wait()

	validateExpectedTenants(t, ctx, s1, s2, &company, tenantsExpected)

	// Test TenantURL collections
	url := &companyconfig.TenantURL{
		BaseModel: ucdb.NewBase(),
		TenantID:  ten.ID,
		TenantURL: "https://test" + uuid.Must(uuid.NewV4()).String() + ".com",
	}
	err = s1.SaveTenantURL(ctx, url)
	assert.NoErr(t, err)
	validateExpectedTenantsURLs(t, ctx, s1, s2, ten, []companyconfig.TenantURL{*url})

	outU, err := s1.GetTenantURL(ctx, url.ID)
	assert.NoErr(t, err)
	assert.Equal(t, outU, url)
	outU, err = s1.GetTenantURLByURL(ctx, url.TenantURL)
	assert.NoErr(t, err)
	assert.Equal(t, outU, url)

	err = s2.DeleteTenantURL(ctx, url.ID)
	assert.NoErr(t, err)
	validateExpectedTenantsURLs(t, ctx, s1, s2, ten, []companyconfig.TenantURL{})

	_, err = s1.GetTenantURL(ctx, url.ID)
	assert.NotNil(t, err)
	_, err = s1.GetTenantURLByURL(ctx, url.TenantURL)
	assert.NotNil(t, err)

	url = &companyconfig.TenantURL{
		BaseModel: ucdb.NewBase(),
		TenantID:  ten.ID,
		TenantURL: "https://test" + uuid.Must(uuid.NewV4()).String() + ".com",
	}
	err = s1.SaveTenantURL(ctx, url)
	assert.NoErr(t, err)
	validateExpectedTenantsURLs(t, ctx, s1, s2, ten, []companyconfig.TenantURL{*url})

	// Now try a multithreaded test
	threadCount = 10
	opCount = 10
	tenantsURLsExpected := []companyconfig.TenantURL{}
	tenantsURLsExpected = append(tenantsURLsExpected, *url)
	// 	Have threadCount threads try to release key with non matching sentinel
	for i := range threadCount {
		wg.Add(1)

		go func(threadID int) {
			defer wg.Done()

			localTenantURLs := []companyconfig.TenantURL{}
			var localerr error
			for j := range opCount {
				// Create another tenant and validation the collections are updated
				localTenURL := &companyconfig.TenantURL{
					BaseModel: ucdb.NewBase(),
					TenantID:  ten.ID,
					TenantURL: "https://test" + uuid.Must(uuid.NewV4()).String() + ".com",
				}
				if j%2 == 0 {
					localerr = s1.SaveTenantURL(ctx, localTenURL)
				} else {
					localerr = s2.SaveTenantURL(ctx, localTenURL)
				}
				assert.NoErr(t, localerr)
				localTenantURLs = append(localTenantURLs, *localTenURL)
			}

			localerr = s1.DeleteTenantURL(ctx, localTenantURLs[0].ID)
			assert.NoErr(t, localerr)

			tenantLock.Lock()
			tenantsURLsExpected = append(tenantsURLsExpected, localTenantURLs[1:]...)
			tenantLock.Unlock()
		}(i)
	}
	wg.Wait()

	validateExpectedTenantsURLs(t, ctx, s1, s2, ten, tenantsURLsExpected)
}

func TestKeyNames(t *testing.T) {
	np := companyconfig.NewCompanyConfigCacheNameProvider()
	keyIDs := np.GetAllKeyIDs()
	for _, keyID := range keyIDs {
		assert.True(t, len(keyID) > 3)
		key := np.GetKeyName(cache.KeyNameID(keyID), []string{"a", "b", "c"})
		assert.True(t, len(key) > 3)
	}
}
