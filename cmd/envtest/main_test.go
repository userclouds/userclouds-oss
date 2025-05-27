package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	authzconfig "userclouds.com/authz/config"
	authzRoutes "userclouds.com/authz/routes"
	"userclouds.com/idp/config"
	idpRoutes "userclouds.com/idp/routes"
	"userclouds.com/infra/acme"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/internal/routinghelper"
	"userclouds.com/internal/security"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/testkeys"
	"userclouds.com/plex/manager"
	"userclouds.com/plex/routes"
	"userclouds.com/plex/serviceconfig"
)

type router struct {
	Routes   []routinghelper.Rule
	Handlers map[service.Service]http.Handler
}

// we make our own devlb replica here to handle multiplexing services
func (rt router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, rule := range rt.Routes {
		if len(rule.HostHeaders) > 0 {
			continue // don't use console. as default
		}

		for _, prefix := range rule.PathPrefixes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				if rule.Service == service.LogServer {
					w.WriteHeader(http.StatusOK)
					return
				}

				rt.Handlers[rule.Service].ServeHTTP(w, r)
				return
			}
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

func TestEnvTest(t *testing.T) {
	assert.NoErr(t, os.Chdir("../.."))

	ctx := context.Background()

	cacheCfg := cachetesthelpers.NewCacheConfig()
	cdbc, ldbc, ccs := testhelpers.NewTestStorage(t)
	_, consoleTenant, _ := testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, ccs, cdbc, ldbc)

	routeCfg, err := routinghelper.ParseIngressConfig(ctx, universe.Current(), "unused", "unused", service.AllWebServices)
	assert.NoErr(t, err)
	r := router{
		Routes:   routeCfg.Rules,
		Handlers: make(map[service.Service]http.Handler),
	}
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)
	consoleTenant.TenantURL = testhelpers.UpdateTenantURLForTestTenant(t, consoleTenant.TenantURL, ts.URL)
	assert.NoErr(t, ccs.SaveTenant(ctx, consoleTenant))

	mgr, err := manager.NewFromCompanyConfig(ctx, ccs, consoleTenant.ID, cacheCfg)

	assert.NoErr(t, err)

	tp, err := mgr.GetTenantPlex(ctx, consoleTenant.ID)
	assert.NoErr(t, err)

	tsm := tenantmap.NewStateMap(ccs, cacheCfg)
	clientSecret, err := tp.PlexConfig.PlexMap.Apps[0].ClientSecret.Resolve(ctx)

	assert.NoErr(t, err)
	clientID := tp.PlexConfig.PlexMap.Apps[0].ClientID
	tokenSrc := jsonclient.ClientCredentialsTokenSource(consoleTenant.TenantURL+"/oidc/token", clientID, clientSecret, nil)
	idpConfig := &config.Config{ConsoleTenantID: consoleTenant.ID}
	authzCfg := authzconfig.Config{ConsoleTenantID: consoleTenant.ID}
	plexCfg := serviceconfig.ServiceConfig{ACME: &acme.Config{PrivateKey: testkeys.Config.PrivateKey}, ConsoleURL: "http://console.test.userclouds.tools"}
	r.Handlers[service.AuthZ] = authzRoutes.Init(ctx, tsm, ccs, authzCfg).Build()
	idpr, err := idpRoutes.Init(ctx, tsm, ccs, nil, nil, idpConfig)
	assert.NoErr(t, err)
	r.Handlers[service.IDP] = idpr.Build()
	r.Handlers[service.Plex] = routes.Init(ctx, tokenSrc, ccs, nil, &plexCfg, security.NewSecurityChecker(), consoleTenant.ID, nil).Build()

	var wg sync.WaitGroup
	wg.Add(3)
	assert.NoErr(t, runTest(ctx, "authz", consoleTenant.TenantURL, tokenSrc, &wg, true, 1))
	assert.NoErr(t, runTest(ctx, "userstore", consoleTenant.TenantURL, tokenSrc, &wg, true, 1))
	assert.NoErr(t, runTest(ctx, "tokenizer", consoleTenant.TenantURL, tokenSrc, &wg, true, 1))
	wg.Wait()

	assert.Equal(t, len(errs), 0, assert.Errorf("errors: %v", errs))
}
