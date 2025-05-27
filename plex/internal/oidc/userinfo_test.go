package oidc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/oidc"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
	plexOIDC "userclouds.com/plex/internal/oidc"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/tenantconfig"
	"userclouds.com/plex/manager"
)

func TestUserInfo(t *testing.T) {
	ctx := context.Background()
	cc, lc, ccs := testhelpers.NewTestStorage(t)
	company, ten, tdb := testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, ccs, cc, lc)

	mgr := manager.NewFromDB(tdb, cachetesthelpers.NewCacheConfig())
	tp, err := mgr.GetTenantPlex(context.Background(), ten.ID)
	assert.NoErr(t, err)

	h := plexOIDC.NewTestHandler(provider.ProdFactory{})
	cs, err := tp.PlexConfig.PlexMap.Apps[0].ClientSecret.Resolve(ctx)
	assert.NoErr(t, err)

	b := url.Values{
		"grant_type":    []string{"client_credentials"},
		"client_id":     []string{tp.PlexConfig.PlexMap.Apps[0].ClientID},
		"client_secret": []string{cs},
	}.Encode()

	ctx = tenantconfig.TESTONLYSetTenantConfig(&tenantplex.TenantConfig{
		PlexMap: tp.PlexConfig.PlexMap,
		Keys:    tp.PlexConfig.Keys,
	})
	ts := tenantmap.NewTenantState(ten, company, uctest.MustParseURL(ten.TenantURL), tdb, nil, nil, "", ccs, false, nil, nil)
	ctx = multitenant.SetTenantState(ctx, ts)
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(b)))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	r = r.WithContext(ctx)

	// this should all work
	rr := httptest.NewRecorder()
	h.TokenExchange(rr, r)
	assert.Equal(t, rr.Code, http.StatusOK)

	var tr oidc.TokenResponse
	assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &tr))

	r = httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+tr.AccessToken)
	r = r.WithContext(ctx)

	rr = httptest.NewRecorder()
	h.UserInfoHandler(rr, r)
	assert.Equal(t, rr.Code, http.StatusOK)

	var uir oidc.UCTokenClaims
	assert.NoErr(t, json.Unmarshal(rr.Body.Bytes(), &uir))
	assert.Equal(t, uir.Subject, tp.PlexConfig.PlexMap.Apps[0].ClientID)
	assert.Equal(t, uir.SubjectType, "client")
	assert.Equal(t, uir.Name, tp.PlexConfig.PlexMap.Apps[0].Name)

	// test failure case where the login app was changed underneath a valid token
	tp.PlexConfig.PlexMap.Apps[0].ClientID = "newclientid"
	ctx = tenantconfig.TESTONLYSetTenantConfig(&tenantplex.TenantConfig{
		PlexMap: tp.PlexConfig.PlexMap,
		Keys:    tp.PlexConfig.Keys,
	})
	// we need to reset this too because tenantconfig.TESTONLYSetTenantConfig doesn't take a
	// context to start with, but forces context.Background()
	ctx = multitenant.SetTenantState(ctx, ts)

	r = httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+tr.AccessToken)
	r = r.WithContext(ctx)

	rr = httptest.NewRecorder()
	h.UserInfoHandler(rr, r)
	assert.Equal(t, rr.Code, http.StatusBadRequest)
}
