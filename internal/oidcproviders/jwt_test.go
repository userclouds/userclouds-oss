package oidcproviders

import (
	"context"
	"testing"

	gooidc "github.com/coreos/go-oidc/v3/oidc"

	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantdb"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/tenantplex"
	tenantplexstorage "userclouds.com/internal/tenantplex/storage"
	plexconfigtest "userclouds.com/internal/tenantplex/test"
	"userclouds.com/internal/uctest"
)

type mockProvider struct {
	providerURL string
}

func (p mockProvider) Verifier(*gooidc.Config) Verifier {
	return p
}

func (p mockProvider) Verify(ctx context.Context, rawJWT string) (*gooidc.IDToken, error) {
	tc, err := ucjwt.ParseUCClaimsUnverified(rawJWT)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	if tc.Issuer != p.providerURL {
		return nil, ucerr.Errorf("invalid issuer: expected %s, got %s", p.providerURL, tc.Issuer)
	}
	// NB: we only pass through the stuff we know we need right now
	return &gooidc.IDToken{Audience: tc.Audience}, nil
}

func createJWT(t *testing.T, tenantURL string, audience ...string) string {
	return uctest.CreateJWT(t, oidc.UCTokenClaims{StandardClaims: oidc.StandardClaims{Audience: audience}}, tenantURL)
}

func TestMultipleProviders(t *testing.T) {
	ctx := context.Background()
	cacheCfg := cachetesthelpers.NewCacheConfig()
	ccdb := testdb.New(t, migrate.NewTestSchema(companyconfig.Schema))
	tdb := testdb.New(t, migrate.NewTestSchema(tenantdb.Schema))
	companyConfigStorage, err := companyconfig.NewStorage(ctx, ccdb, cacheCfg)
	assert.NoErr(t, err)

	company := &companyconfig.Company{BaseModel: ucdb.NewBase(), Name: "Jerry Enterprises", Type: companyconfig.CompanyTypeCustomer}
	assert.NoErr(t, companyConfigStorage.SaveCompany(ctx, company))
	tenant := &companyconfig.Tenant{BaseModel: ucdb.NewBase(), CompanyID: company.ID, TenantURL: "https://contoso.com", Name: "Contoso"}
	assert.NoErr(t, companyConfigStorage.SaveTenant(ctx, tenant))

	tp := tenantplex.TenantPlex{
		VersionBaseModel: ucdb.NewVersionBaseWithID(tenant.ID),
		PlexConfig: plexconfigtest.NewTenantConfigBuilder().
			AddProvider().SetName("active").MakeActive().MakeUC().
			Build(),
	}
	tps := tenantplexstorage.New(ctx, tdb, cacheCfg)
	err = tps.SaveTenantPlex(ctx, &tp)
	assert.NoErr(t, err)

	assert.NoErr(t, companyConfigStorage.SaveTenantURL(ctx, &companyconfig.TenantURL{
		BaseModel: ucdb.NewBase(),
		TenantID:  tenant.ID,
		TenantURL: "https://auth.contoso.com",
		Validated: true,
	}))

	newProvider = func(providerURL string) (Provider, error) {
		return &mockProvider{providerURL}, nil
	}

	mainJWT := createJWT(t, "https://contoso.com", "https://contoso.com")
	ctx = multitenant.SetTenantState(ctx, tenantmap.NewTenantState(tenant, company, uctest.MustParseURL(tenant.TenantURL), tdb, nil, nil, "", companyConfigStorage, false, nil, cacheCfg))
	pm := NewOIDCProviderMap()
	_, err = pm.VerifyAndDecode(ctx, mainJWT)
	assert.NoErr(t, err)

	// token issued to a non-primary tenant URL should still be valid on the tenant
	subJWT := createJWT(t, "https://auth.contoso.com", "https://contoso.com", "https://auth.contoso.com")
	_, err = pm.VerifyAndDecode(ctx, subJWT)
	assert.NoErr(t, err)

	externalIssuer := "https://www.okta.com"
	tp.PlexConfig.ExternalOIDCIssuers = []string{externalIssuer}
	err = tps.SaveTenantPlex(ctx, &tp)
	assert.NoErr(t, err)

	// test external issuer
	extJWT := createJWT(t, externalIssuer, "https://contoso.com", "https://auth.contoso.com")
	pm = NewOIDCProviderMap()
	_, err = pm.VerifyAndDecode(ctx, extJWT)
	assert.NoErr(t, err)
}
