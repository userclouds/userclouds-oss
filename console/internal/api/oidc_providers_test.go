package api_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"

	"userclouds.com/console/internal/api"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/secret"
	"userclouds.com/internal/pageparameters/pagetype"
	param "userclouds.com/internal/pageparameters/parameter"
	plextest "userclouds.com/internal/tenantplex/test"
)

func createProviderURL(tenantID uuid.UUID) string {
	return fmt.Sprintf("/api/tenants/%v/oidcproviders/create", tenantID)
}

func deleteProviderURL(tenantID uuid.UUID) string {
	return fmt.Sprintf("/api/tenants/%v/oidcproviders/delete", tenantID)
}

func TestOIDCProviderCreateAndDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tf, c := newAPIClient(ctx, t)

	nativeProviders := oidc.GetDefaultNativeProviders()
	assert.True(t, len(nativeProviders) == 4)
	for _, p := range nativeProviders {
		assert.NoErr(t, p.Validate())
	}

	var resp oidc.ProviderConfig

	// cannot create a native provider
	provider := nativeProviders[0]
	createReq := api.CreateOIDCProviderRequest{provider}
	assert.NotNil(t, c.Post(ctx, createProviderURL(tf.ConsoleTenantID), createReq, &resp))

	// cannot create a provider with a name that exists
	provider = oidc.GetDefaultCustomProvider()
	provider.Name = nativeProviders[0].Name
	provider.Description = "Foo"
	provider.IssuerURL = "https://foo.com"
	assert.NoErr(t, provider.Validate())
	createReq = api.CreateOIDCProviderRequest{provider}
	assert.NotNil(t, c.Post(ctx, createProviderURL(tf.ConsoleTenantID), createReq, &resp))

	// cannot create a provider with an issuer URL that exists
	provider = oidc.GetDefaultCustomProvider()
	provider.Name = "foo"
	provider.Description = "Foo"
	provider.IssuerURL = nativeProviders[0].IssuerURL
	assert.NoErr(t, provider.Validate())
	createReq = api.CreateOIDCProviderRequest{provider}
	assert.NotNil(t, c.Post(ctx, createProviderURL(tf.ConsoleTenantID), createReq, &resp))

	// cannot create a provider that is invalid
	provider = oidc.GetDefaultCustomProvider()
	provider.Name = "foo"
	provider.Description = "Foo"
	provider.IssuerURL = "https://foo.com"
	provider.IsNative = true
	assert.NotNil(t, provider.Validate())
	createReq = api.CreateOIDCProviderRequest{provider}
	assert.NotNil(t, c.Post(ctx, createProviderURL(tf.ConsoleTenantID), createReq, &resp))

	// successfully create a provider that is fully configured
	provider = oidc.GetDefaultCustomProvider()
	provider.Name = "foo"
	provider.Description = "Foo"
	provider.IssuerURL = "https://foo.com"
	provider.ClientID = "foo"
	provider.ClientSecret = secret.NewTestString("foo")
	assert.NoErr(t, provider.Validate())
	p, err := oidc.GetProvider(&provider)
	assert.NoErr(t, err)
	assert.True(t, p.IsConfigured())
	createReq = api.CreateOIDCProviderRequest{provider}
	assert.NoErr(t, c.Post(ctx, createProviderURL(tf.ConsoleTenantID), createReq, &resp))
	assert.Equal(t, provider, resp, assert.CmpOpt(cmp.AllowUnexported(secret.String{})))

	// cannot create the same provider twice
	assert.NotNil(t, c.Post(ctx, createProviderURL(tf.ConsoleTenantID), createReq, &resp))

	// can add a second distinct provider
	anotherProvider := oidc.GetDefaultCustomProvider()
	anotherProvider.Name = "bar"
	anotherProvider.Description = "Bar"
	anotherProvider.IssuerURL = "https://bar.com"
	assert.NoErr(t, anotherProvider.Validate())
	createReq = api.CreateOIDCProviderRequest{anotherProvider}
	assert.NoErr(t, c.Post(ctx, createProviderURL(tf.ConsoleTenantID), createReq, &resp))
	assert.Equal(t, anotherProvider, resp, assert.CmpOpt(cmp.AllowUnexported(secret.String{})))

	// check that expected providers exist
	tc := loadTenantConfig(t, tf)
	assert.True(t, len(tc.OIDCProviders.Providers) == 6)
	assert.Equal(t, tc.OIDCProviders.Providers[0], nativeProviders[0], assert.CmpOpt(cmp.AllowUnexported(secret.String{})))
	assert.Equal(t, tc.OIDCProviders.Providers[1], nativeProviders[1], assert.CmpOpt(cmp.AllowUnexported(secret.String{})))
	assert.Equal(t, tc.OIDCProviders.Providers[2], nativeProviders[2], assert.CmpOpt(cmp.AllowUnexported(secret.String{})))
	assert.Equal(t, tc.OIDCProviders.Providers[4], provider, assert.CmpOpt(cmp.AllowUnexported(secret.String{})))
	assert.Equal(t, tc.OIDCProviders.Providers[5], anotherProvider, assert.CmpOpt(cmp.AllowUnexported(secret.String{})))

	// cannot delete a native provider
	deleteReq := api.DeleteOIDCProviderRequest{nativeProviders[0].Name}
	assert.NotNil(t, c.Post(ctx, deleteProviderURL(tf.ConsoleTenantID), deleteReq, nil))

	// cannot delete a non-existing provider
	deleteReq = api.DeleteOIDCProviderRequest{"baz"}
	assert.NotNil(t, c.Post(ctx, deleteProviderURL(tf.ConsoleTenantID), deleteReq, nil))

	// cannot delete a used provider
	tc = plextest.NewTenantConfigBuilderFromTenantConfig(loadTenantConfig(t, tf)).
		SwitchToApp(0).
		SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, provider.Name).
		Build()
	sappr := newSaveAppPageParametersRequestFromTenantConfig(t, tc)
	var putResponse api.AppPageParametersResponse
	assert.NoErr(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &putResponse))

	deleteReq = api.DeleteOIDCProviderRequest{provider.Name}
	assert.NotNil(t, c.Post(ctx, deleteProviderURL(tf.ConsoleTenantID), deleteReq, nil))

	// successfully delete a providers

	deleteReq = api.DeleteOIDCProviderRequest{anotherProvider.Name}
	assert.NoErr(t, c.Post(ctx, deleteProviderURL(tf.ConsoleTenantID), deleteReq, nil))

	// check that expected providers exist
	tc = loadTenantConfig(t, tf)
	assert.True(t, len(tc.OIDCProviders.Providers) == 5)
	assert.Equal(t, tc.OIDCProviders.Providers[0], nativeProviders[0], assert.CmpOpt(cmp.AllowUnexported(secret.String{})))
	assert.Equal(t, tc.OIDCProviders.Providers[1], nativeProviders[1], assert.CmpOpt(cmp.AllowUnexported(secret.String{})))
	assert.Equal(t, tc.OIDCProviders.Providers[2], nativeProviders[2], assert.CmpOpt(cmp.AllowUnexported(secret.String{})))
	assert.Equal(t, tc.OIDCProviders.Providers[4], provider, assert.CmpOpt(cmp.AllowUnexported(secret.String{})))
}
