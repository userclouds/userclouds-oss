package userstore_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/userstore"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/service"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
)

func TestSQLInjection(t *testing.T) {
	ctx := context.Background()

	cdbc, ldbc, s := testhelpers.NewTestStorage(t)
	_, ct, _ := testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, s, cdbc, ldbc)

	tm := tenantmap.NewStateMap(s, nil)
	m2mAuth, err := m2m.GetM2MTokenSource(ctx, ct.ID)
	assert.NoErr(t, err)
	consoleTenantInfo, err := s.GetTenantInfo(ctx, ct.ID)
	assert.NoErr(t, err)
	h, err := userstore.NewHandler(ctx, nil, nil, nil, m2mAuth, *consoleTenantInfo)
	assert.NoErr(t, err)

	srv := httptest.NewServer(service.BaseMiddleware.Apply(multitenant.Middleware(tm).Apply(h)))
	testhelpers.UpdateTenantURL(ctx, t, s, ct, srv)

	cl := jsonclient.New(srv.URL, jsonclient.HeaderHost(uctest.MustParseURL(ct.TenantURL).Host))
	var res idp.ListDataTypesResponse
	err = cl.Get(ctx, "/config/datatypes?sort_key=name%29%3BSELECT%20pg_sleep%2810%29--&sort_order=ascending", &res)
	assert.NotNil(t, err)
	var je jsonclient.Error
	assert.True(t, errors.As(err, &je))
	assert.Equal(t, je.StatusCode, http.StatusBadRequest)
}
