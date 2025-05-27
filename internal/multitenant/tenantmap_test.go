package multitenant_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/middleware"
	"userclouds.com/infra/ucdb"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
)

func TestMultipleTenantURLs(t *testing.T) {
	ctx := context.Background()

	companyDBConfig, logDBConfig, companyStorage := testhelpers.NewTestStorage(t)
	company := testhelpers.ProvisionTestCompanyWithoutACL(ctx, t, companyStorage)
	tenant, _ := testhelpers.ProvisionTestTenant(ctx, t, companyStorage, companyDBConfig, logDBConfig, company.ID)
	turl := &companyconfig.TenantURL{BaseModel: ucdb.NewBase(), TenantID: tenant.ID, TenantURL: "http://foo.local", Validated: true}
	assert.NoErr(t, companyStorage.SaveTenantURL(ctx, turl))

	tm := testhelpers.NewTestTenantStateMap(companyStorage)
	ts, err := tm.GetTenantStateForHostname(ctx, tenant.GetHostName())
	assert.NoErr(t, err)
	assert.Equal(t, ts.ID, tenant.ID)

	ts, err = tm.GetTenantStateForHostname(ctx, "foo.local")
	assert.NoErr(t, err)
	assert.Equal(t, ts.ID, tenant.ID)
}

func TestInvalidTenantURLs(t *testing.T) {
	ctx := context.Background()

	companyDB, logDB, companyStorage := testhelpers.NewTestStorage(t)
	company := testhelpers.ProvisionTestCompanyWithoutACL(ctx, t, companyStorage)
	tenant, _ := testhelpers.ProvisionTestTenant(ctx, t, companyStorage, companyDB, logDB, company.ID)
	ts := testhelpers.NewTestTenantStateMap(companyStorage)

	mw := middleware.Chain(multitenant.Middleware(ts))

	var calls int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		jsonapi.Marshal(w, "test reply", jsonapi.Code(http.StatusCreated))
	})

	handlerWithMiddleware := mw.Apply(handler)

	srv := httptest.NewServer(handlerWithMiddleware)
	defer srv.Close()
	testhelpers.UpdateTenantURL(ctx, t, companyStorage, tenant, srv)
	hostname := tenant.GetHostName()
	_, err := ts.GetTenantStateForHostname(ctx, hostname)
	assert.NoErr(t, err)

	r := httptest.NewRequest(http.MethodPost, tenant.TenantURL, nil)
	r.Host = hostname
	rr := httptest.NewRecorder()
	handlerWithMiddleware.ServeHTTP(rr, r)

	assert.Equal(t, rr.Code, http.StatusCreated)
	assert.Equal(t, calls, 1)

	invalidURL := uctest.MustParseURL(tenant.TenantURL)
	invalidURL.Host = "invalid.local"

	r = httptest.NewRequest(http.MethodPost, invalidURL.String(), nil)
	rr = httptest.NewRecorder()
	handlerWithMiddleware.ServeHTTP(rr, r)

	assert.Equal(t, rr.Code, http.StatusNotFound)
	assert.Equal(t, calls, 1)
}
