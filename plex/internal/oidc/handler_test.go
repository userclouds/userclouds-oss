package oidc_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"

	"userclouds.com/idp"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/secret"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/testkeys"
	"userclouds.com/internal/uctest"
	plexOIDC "userclouds.com/plex/internal/oidc"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/tenantconfig"
	"userclouds.com/plex/manager"
)

func TestValidateClient(t *testing.T) {
	// TODO: there's probably a broader test case set here, but this is a specific regression

	// NB: the slashes are the important regression test here
	id := "foo/lorem"
	sec := "bar/baz"

	tc := &tenantplex.TenantConfig{
		PlexMap: tenantplex.PlexMap{
			Apps: []tenantplex.App{{
				ClientID:     id,
				ClientSecret: secret.NewTestString(sec),
			}},
		},
	}
	ctx := tenantconfig.TESTONLYSetTenantConfig(tc)

	// set up a request, the contents of which don't matter
	r := httptest.NewRequest(http.MethodPost, "/", nil)

	// the important point is that the id & secret are url-encoded per
	// https://www.rfc-editor.org/rfc/rfc6749#section-2.3.1
	token := fmt.Sprintf("%s:%s", url.QueryEscape(id), url.QueryEscape(sec))

	// per RFC 2617, the whole string is b64 encoded
	// https://www.rfc-editor.org/rfc/rfc2617#section-2
	token = base64.StdEncoding.EncodeToString([]byte(token))
	r.Header.Add("Authorization", fmt.Sprintf("Basic %s", token))

	pf := &url.Values{}
	_, err := plexOIDC.ValidateClient(ctx, r, pf)

	assert.NoErr(t, err)

	// set up a request that should fail to validate
	id = "baz"
	r = httptest.NewRequest(http.MethodPost, "/", nil)
	token = fmt.Sprintf("%s:%s", url.QueryEscape(id), url.QueryEscape(sec))
	token = base64.StdEncoding.EncodeToString([]byte(token))
	r.Header.Add("Authorization", fmt.Sprintf("Basic %s", token))
	pf = &url.Values{}
	_, err = plexOIDC.ValidateClient(ctx, r, pf)
	assert.NotNil(t, err)

	rr := httptest.NewRecorder()
	jsonapi.MarshalError(ctx, rr, err)
	assert.Equal(t, strings.TrimSpace(rr.Body.String()), `{"error":"invalid_client","error_description":"no plex app with Plex client ID 'baz' found"}`)
}

func TestGrantTypeRestrictions(t *testing.T) {
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
		"grant_type":    []string{"authorization_code"},
		"client_id":     []string{tp.PlexConfig.PlexMap.Apps[0].ClientID},
		"client_secret": []string{cs},
	}.Encode()

	ctx = tenantconfig.TESTONLYSetTenantConfig(&tenantplex.TenantConfig{
		PlexMap: tp.PlexConfig.PlexMap,
	})
	ctx = multitenant.SetTenantState(ctx, tenantmap.NewTenantState(ten, company, uctest.MustParseURL(ten.TenantURL), tdb, nil, nil, "", ccs, false, nil, nil))

	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(b)))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	r = r.WithContext(ctx)

	// this should all work
	rr := httptest.NewRecorder()
	h.TokenExchange(rr, r)
	assert.Equal(t, rr.Code, http.StatusBadRequest)
	bs, err := io.ReadAll(rr.Body)
	assert.NoErr(t, err)
	assert.Contains(t, string(bs), "required query parameter 'code' missing or malformed")

	// turn off authcode flow
	tp.PlexConfig.PlexMap.Apps[0].GrantTypes = tenantplex.GrantTypes{tenantplex.GrantTypeClientCredentials}
	assert.NoErr(t, mgr.SaveTenantPlex(ctx, tp))

	// this should fail
	rr = httptest.NewRecorder()
	h.TokenExchange(rr, r)
	assert.Equal(t, rr.Code, http.StatusBadRequest)
	bs, err = io.ReadAll(rr.Body)
	assert.NoErr(t, err)
	assert.Contains(t, string(bs), "Client is not authorized to use grant type authorization_code")
}

type passwordProvider struct {
	iface.Client
	provider.Factory
}

func (p *passwordProvider) NewClient(ctx context.Context, prov tenantplex.Provider, clientID string, provAppID uuid.UUID) (iface.Client, error) {
	return p, nil
}

func (p *passwordProvider) UsernamePasswordLogin(ctx context.Context, username, password string) (*iface.LoginResponseWithClaims, error) {
	return &iface.LoginResponseWithClaims{
		Status: idp.LoginStatusSuccess,
		Claims: jwt.MapClaims{
			"email": "me@foo.com",
			"sub":   "me@foo.com",
		}}, nil
}

func TestPasswordGrant(t *testing.T) {
	ctx := context.Background()
	cc, lc, ccs := testhelpers.NewTestStorage(t)
	company, ten, tdb := testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, ccs, cc, lc)

	mgr := manager.NewFromDB(tdb, cachetesthelpers.NewCacheConfig())
	tp, err := mgr.GetTenantPlex(ctx, ten.ID)
	assert.NoErr(t, err)

	h := plexOIDC.NewTestHandler(&passwordProvider{})

	cs, err := tp.PlexConfig.PlexMap.Apps[0].ClientSecret.Resolve(ctx)
	assert.NoErr(t, err)

	b := url.Values{
		"grant_type":    []string{"password"},
		"client_id":     []string{tp.PlexConfig.PlexMap.Apps[0].ClientID},
		"client_secret": []string{cs},
		"username":      []string{"foo"},
		"password":      []string{"bar"},
		// NB: scope is optional for password grant
	}.Encode()

	ctx = tenantconfig.TESTONLYSetTenantConfig(&tenantplex.TenantConfig{
		PlexMap: tp.PlexConfig.PlexMap,
		Keys:    testkeys.Config,
	})

	ctx = multitenant.SetTenantState(ctx, tenantmap.NewTenantState(ten, company, uctest.MustParseURL(ten.TenantURL), tdb, nil, nil, "", ccs, false, nil, nil))
	tp.PlexConfig.PlexMap.Apps[0].GrantTypes = tenantplex.GrantTypes{tenantplex.GrantTypePassword}
	assert.NoErr(t, mgr.SaveTenantPlex(ctx, tp))

	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(b)))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	r = r.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.TokenExchange(rr, r)
	assert.Equal(t, rr.Code, http.StatusOK, assert.Must())
	bs, err := io.ReadAll(rr.Body)
	assert.NoErr(t, err)

	var tokenResponse oidc.TokenResponse
	assert.NoErr(t, json.Unmarshal(bs, &tokenResponse))
	assert.Equal(t, tokenResponse.TokenType, "Bearer")
	assert.Equal(t, tokenResponse.IDToken, "") // no ID tokens for ROPC
}
