package userstore_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"userclouds.com/idp/internal/userstore"
	publicUserstore "userclouds.com/idp/userstore"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/service"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
)

func TestErrorStackLeak(t *testing.T) {
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
	// NB: we use anonymous structs here to avoid triggering the client-side validation logic
	col := struct{ Column any }{
		Column: struct {
			Name      string
			Type      string
			IndexType publicUserstore.ColumnIndexType
			IsArray   bool
		}{
			Name:      "test",
			Type:      "blaze",
			IndexType: publicUserstore.ColumnIndexTypeIndexed,
			IsArray:   false,
		},
	}
	var res publicUserstore.Column

	err = cl.Post(ctx, "/config/columns", col, &res)
	assert.NotNil(t, err)
	var je jsonclient.Error
	assert.True(t, errors.As(err, &je))
	assert.Equal(t, je.StatusCode, http.StatusBadRequest)
	assert.DoesNotContain(t, je.Body, ".go")
}
