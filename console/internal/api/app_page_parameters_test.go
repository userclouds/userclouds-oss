package api_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/console/internal/api"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/oidc"
	pageparams "userclouds.com/internal/pageparameters"
	"userclouds.com/internal/pageparameters/pagetype"
	param "userclouds.com/internal/pageparameters/parameter"
	"userclouds.com/internal/tenantplex"
	plextest "userclouds.com/internal/tenantplex/test"
)

func appPageParametersURL(tenantID uuid.UUID, appID uuid.UUID) string {
	return fmt.Sprintf("/api/tenants/%v/apppageparameters/%v", tenantID, appID)
}

func appPageParametersURLFromTenantConfig(tenantID uuid.UUID, tc tenantplex.TenantConfig) string {
	return appPageParametersURL(tenantID, tc.PlexMap.Apps[0].ID)
}

func newSaveAppPageParametersRequestFromTenantConfig(t *testing.T, tc tenantplex.TenantConfig) api.SaveAppPageParametersRequest {
	t.Helper()

	getters, cd := tenantplex.MakeParameterRetrievalTools(&tc, &tc.PlexMap.Apps[0])
	sappr := api.SaveAppPageParametersRequest{PageTypeParameterChanges: api.ParameterChangeByNameByPageType{}}
	for _, pt := range pagetype.Types() {
		paramsByName := api.ParameterChangeByName{}
		for _, pn := range pt.ParameterNames() {
			currentValue, _, err := pageparams.GetParameterValues(pt, pn, getters, cd)
			assert.NoErr(t, err)
			paramsByName[pn] = api.ParameterChange{Name: pn, NewValue: currentValue}
		}
		if len(paramsByName) > 0 {
			sappr.PageTypeParameterChanges[pt] = paramsByName
		}
	}
	return sappr
}

func verifyAppPageParametersResponse(t *testing.T, tenantID uuid.UUID, tc tenantplex.TenantConfig, appr api.AppPageParametersResponse) {
	t.Helper()

	assert.IsNil(t, appr.Validate(), assert.Must())
	assert.Equal(t, tenantID, appr.TenantID)
	assert.Equal(t, tc.PlexMap.Apps[0].ID, appr.AppID)
	assert.Equal(t, len(appr.PageTypeParameters), len(pagetype.Types()))
	for pt, paramsByName := range appr.PageTypeParameters {
		for pn, p := range paramsByName {
			assert.Equal(t, pn, p.Name)
			assert.True(t, pt.SupportsParameterName(pn))
		}
	}
}

func verifyDefault(t *testing.T, pt pagetype.Type, p api.Parameter, tc tenantplex.TenantConfig) {
	t.Helper()

	defaultParam, found := pagetype.DefaultParameterGetter(pt, p.Name)
	assert.True(t, found)
	defaultParam, err := defaultParam.ApplyClientData(tenantplex.MakeParameterClientData(&tc, &tc.PlexMap.Apps[0]))
	assert.NoErr(t, err)
	assert.Equal(t, defaultParam.Value, p.DefaultValue)
	assert.Equal(t, fmt.Sprintf("%s.%s", defaultParam.Name, defaultParam.Value), fmt.Sprintf("%s.%s", p.Name, p.DefaultValue))
}

func TestDefaultAppPageParametersTenantPlex(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tf, c := newAPIClient(ctx, t)
	// load tenant config
	tc := loadTenantConfig(t, tf)

	t.Run("TestDefaultAppPageParametersTenantPlex", func(t *testing.T) {
		// verify that we have exactly one app and no custom page parameter settings by default
		// so we don't have to check in our other tests
		assert.Equal(t, len(tc.PlexMap.Apps), 1, assert.Must())
		assert.Equal(t, len(tc.PlexMap.Apps[0].PageParameters), 0, assert.Must())
		assert.Equal(t, len(tc.PageParameters), 0, assert.Must())
	})

	t.Run("TestDefaultAppPageParametersTenantPlex", func(t *testing.T) {
		var appr api.AppPageParametersResponse
		assert.IsNil(t, c.Get(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), &appr), assert.Must())
		verifyAppPageParametersResponse(t, tf.ConsoleTenantID, tc, appr)
		for pt, paramsByName := range appr.PageTypeParameters {
			for _, p := range paramsByName {
				verifyDefault(t, pt, p, tc)
				assert.Equal(t, p.CurrentValue, p.DefaultValue)
			}
		}
	})

	t.Run("TestCustomizeAppPageParameters", func(t *testing.T) {
		// add a page parameter customization
		tc := plextest.NewTenantConfigBuilderFromTenantConfig(loadTenantConfig(t, tf)).
			SwitchToApp(0).
			SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "password").
			SetAppPageParameter(pagetype.PlexCreateUserPage, param.EmailLabel, "new label").
			SetAppPageParameter(pagetype.PlexFinishResetPasswordPage, param.NewPasswordLabel, "new label").
			SetAppPageParameter(pagetype.PlexLoginPage, param.CreateAccountText, "new text").
			SetAppPageParameter(pagetype.PlexMfaSubmitPage, param.MFACodeLabel, "new label").
			SetAppPageParameter(pagetype.PlexPasswordlessLoginPage, param.EmailLabel, "new label").
			SetAppPageParameter(pagetype.PlexStartResetPasswordPage, param.HeadingText, "new text").
			Build()

		sappr := newSaveAppPageParametersRequestFromTenantConfig(t, tc)

		// send a put update and validate response
		var putResponse api.AppPageParametersResponse
		assert.IsNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &putResponse), assert.Must())
		verifyAppPageParametersResponse(t, tf.ConsoleTenantID, tc, putResponse)
		assert.Equal(t, putResponse.PageTypeParameters[pagetype.EveryPage][param.AuthenticationMethods].CurrentValue, "password")
		assert.Equal(t, putResponse.PageTypeParameters[pagetype.PlexCreateUserPage][param.EmailLabel].CurrentValue, "new label")
		assert.Equal(t, putResponse.PageTypeParameters[pagetype.PlexFinishResetPasswordPage][param.NewPasswordLabel].CurrentValue, "new label")
		assert.Equal(t, putResponse.PageTypeParameters[pagetype.PlexLoginPage][param.CreateAccountText].CurrentValue, "new text")
		assert.Equal(t, putResponse.PageTypeParameters[pagetype.PlexMfaSubmitPage][param.MFACodeLabel].CurrentValue, "new label")
		assert.Equal(t, putResponse.PageTypeParameters[pagetype.PlexPasswordlessLoginPage][param.EmailLabel].CurrentValue, "new label")
		assert.Equal(t, putResponse.PageTypeParameters[pagetype.PlexStartResetPasswordPage][param.HeadingText].CurrentValue, "new text")
		verifyDefault(t, pagetype.EveryPage, putResponse.PageTypeParameters[pagetype.EveryPage][param.AuthenticationMethods], tc)
		verifyDefault(t, pagetype.PlexCreateUserPage, putResponse.PageTypeParameters[pagetype.PlexCreateUserPage][param.EmailLabel], tc)
		verifyDefault(t, pagetype.PlexFinishResetPasswordPage, putResponse.PageTypeParameters[pagetype.PlexFinishResetPasswordPage][param.NewPasswordLabel], tc)
		verifyDefault(t, pagetype.PlexLoginPage, putResponse.PageTypeParameters[pagetype.PlexLoginPage][param.CreateAccountText], tc)
		verifyDefault(t, pagetype.PlexMfaSubmitPage, putResponse.PageTypeParameters[pagetype.PlexMfaSubmitPage][param.MFACodeLabel], tc)
		verifyDefault(t, pagetype.PlexPasswordlessLoginPage, putResponse.PageTypeParameters[pagetype.PlexPasswordlessLoginPage][param.EmailLabel], tc)
		verifyDefault(t, pagetype.PlexStartResetPasswordPage, putResponse.PageTypeParameters[pagetype.PlexStartResetPasswordPage][param.HeadingText], tc)

		// send a get request and ensure response is identical
		var getResponse api.AppPageParametersResponse
		assert.IsNil(t, c.Get(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), &getResponse), assert.Must())
		assert.True(t, reflect.DeepEqual(getResponse, putResponse))
	})

	t.Run("TestSetDefaultAppPageParameters", func(t *testing.T) {
		modifiedTC := plextest.NewTenantConfigBuilderFromTenantConfig(tc).
			SwitchToApp(0).
			SetAppPageParameter(pagetype.PlexCreateUserPage, param.EmailLabel, "new label").
			Build()

		sappr := newSaveAppPageParametersRequestFromTenantConfig(t, modifiedTC)

		var appr api.AppPageParametersResponse
		assert.IsNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, modifiedTC), sappr, &appr), assert.Must())
		verifyAppPageParametersResponse(t, tf.ConsoleTenantID, modifiedTC, appr)
		p := appr.PageTypeParameters[pagetype.PlexCreateUserPage][param.EmailLabel]
		verifyDefault(t, pagetype.PlexCreateUserPage, p, modifiedTC)
		assert.Equal(t, p.CurrentValue, "new label")

		sappr.PageTypeParameterChanges[pagetype.PlexCreateUserPage][param.EmailLabel] = api.ParameterChange{Name: param.EmailLabel, NewValue: p.DefaultValue}
		assert.IsNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, modifiedTC), sappr, &appr), assert.Must())
		p = appr.PageTypeParameters[pagetype.PlexCreateUserPage][param.EmailLabel]
		verifyDefault(t, pagetype.PlexCreateUserPage, p, modifiedTC)
		assert.Equal(t, p.CurrentValue, p.DefaultValue)
		finalTC := loadTenantConfig(t, tf)
		_, found := finalTC.PlexMap.Apps[0].PageParameters[pagetype.PlexCreateUserPage][param.EmailLabel]
		assert.False(t, found)
	})

	t.Run("TestCustomizeAppPageParameters", func(t *testing.T) {
		tc := plextest.NewTenantConfigBuilderFromTenantConfig(loadTenantConfig(t, tf)).
			SwitchToApp(0).
			SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "passwordless").
			Build()

		sappr := newSaveAppPageParametersRequestFromTenantConfig(t, tc)

		var appr api.AppPageParametersResponse
		assert.IsNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr), assert.Must())
		verifyAppPageParametersResponse(t, tf.ConsoleTenantID, tc, appr)
		p := appr.PageTypeParameters[pagetype.EveryPage][param.AuthenticationMethods]
		verifyDefault(t, pagetype.EveryPage, p, tc)
		assert.Equal(t, p.CurrentValue, "passwordless")

		tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
			SwitchToApp(0).
			SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "google").
			Build()
		sappr = newSaveAppPageParametersRequestFromTenantConfig(t, tc)
		assert.NotNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr), assert.Must())

		tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
			SwitchToApp(0).
			SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "facebook").
			Build()
		sappr = newSaveAppPageParametersRequestFromTenantConfig(t, tc)
		assert.NotNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr), assert.Must())

		tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
			SwitchToApp(0).
			SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "google,password,passwordless").
			Build()
		sappr = newSaveAppPageParametersRequestFromTenantConfig(t, tc)
		assert.NotNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr), assert.Must())

		// set up google as a social provider
		cs, err := crypto.GenerateClientSecret(ctx, oidc.ProviderTypeGoogle.String())
		assert.IsNil(t, err)
		tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
			SwitchToOIDCProvider(oidc.ProviderTypeGoogle.String()).
			SetClientID(crypto.GenerateClientID()).
			SetClientSecret(*cs).
			SwitchToApp(0).
			SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "password,passwordless").
			Build()

		// first save the tenant config setting
		saveTenantConfigRequest := fmt.Sprintf("/api/tenants/%v/plexconfig", tf.ConsoleTenantID)
		tcRequest := api.SaveTenantPlexConfigRequest{TenantConfig: tc}

		var tcResponse api.SaveTenantPlexConfigResponse
		assert.NoErr(t, tcRequest.TenantConfig.UpdateUISettings(ctx))
		assert.IsNil(t, c.Post(ctx, saveTenantConfigRequest, tcRequest, &tcResponse), assert.Must())

		// now select google as an authentication method, which should succeed
		tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
			SwitchToApp(0).
			SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "google,password,passwordless").
			Build()

		sappr = newSaveAppPageParametersRequestFromTenantConfig(t, tc)
		assert.IsNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr), assert.Must())
		verifyAppPageParametersResponse(t, tf.ConsoleTenantID, tc, appr)
		p = appr.PageTypeParameters[pagetype.EveryPage][param.AuthenticationMethods]
		verifyDefault(t, pagetype.EveryPage, p, tc)
		assert.Equal(t, p.CurrentValue, "google,password,passwordless")

		// try updating the tenant config after clearing the social provider - this should now fail
		noSocialTC := plextest.NewTenantConfigBuilderFromTenantConfig(tc).
			SwitchToOIDCProvider(oidc.ProviderTypeGoogle.String()).SetClientID("").Build()
		tcRequest = api.SaveTenantPlexConfigRequest{TenantConfig: noSocialTC}
		assert.NotNil(t, c.Post(ctx, saveTenantConfigRequest, tcRequest, &tcResponse))

		// remove google as an authentication method, then successfully clear the social provider
		tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
			SwitchToApp(0).
			SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "password,passwordless").
			Build()

		sappr = newSaveAppPageParametersRequestFromTenantConfig(t, tc)
		assert.IsNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr), assert.Must())
		verifyAppPageParametersResponse(t, tf.ConsoleTenantID, tc, appr)
		p = appr.PageTypeParameters[pagetype.EveryPage][param.AuthenticationMethods]
		verifyDefault(t, pagetype.EveryPage, p, tc)
		assert.Equal(t, p.CurrentValue, "password,passwordless")

		tcRequest = api.SaveTenantPlexConfigRequest{TenantConfig: noSocialTC}
		assert.IsNil(t, c.Post(ctx, saveTenantConfigRequest, tcRequest, &tcResponse))

		// set up facebook as a social provider
		cs, err = crypto.GenerateClientSecret(ctx, oidc.ProviderTypeFacebook.String())
		assert.IsNil(t, err)
		tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
			SwitchToOIDCProvider(oidc.ProviderTypeFacebook.String()).
			SetClientID(crypto.GenerateClientID()).
			SetClientSecret(*cs).
			SwitchToApp(0).
			SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "password,passwordless").
			Build()

		tcRequest = api.SaveTenantPlexConfigRequest{TenantConfig: tc}
		assert.IsNil(t, c.Post(ctx, saveTenantConfigRequest, tcRequest, &tcResponse), assert.Must())

		// successfully add facebook as a social provider

		tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
			SwitchToApp(0).
			SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "facebook,password,passwordless").
			Build()

		sappr = newSaveAppPageParametersRequestFromTenantConfig(t, tc)
		assert.IsNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr), assert.Must())
		verifyAppPageParametersResponse(t, tf.ConsoleTenantID, tc, appr)
		p = appr.PageTypeParameters[pagetype.EveryPage][param.AuthenticationMethods]
		verifyDefault(t, pagetype.EveryPage, p, tc)
		assert.Equal(t, p.CurrentValue, "facebook,password,passwordless")

		// Clear all providers
		tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
			SwitchToApp(0).
			SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "password").
			Build()

		sappr = newSaveAppPageParametersRequestFromTenantConfig(t, tc)
		assert.IsNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr), assert.Must())
		verifyAppPageParametersResponse(t, tf.ConsoleTenantID, tc, appr)
		p = appr.PageTypeParameters[pagetype.EveryPage][param.AuthenticationMethods]
		verifyDefault(t, pagetype.EveryPage, p, tc)
		assert.Equal(t, p.CurrentValue, "password")

	})

	t.Run("TestBadGetAppPageParametersRequests", func(t *testing.T) {
		var appr api.AppPageParametersResponse

		// request with bad tenant id
		assert.NotNil(t, c.Get(ctx, appPageParametersURL(uuid.Nil, uuid.Nil), &appr), assert.Must())

		// request with bad app id
		assert.NotNil(t, c.Get(ctx, appPageParametersURL(tf.ConsoleTenantID, uuid.Nil), &appr), assert.Must())
	})

	t.Run("TestBadSaveAppPageParametersRequests", func(t *testing.T) {
		var appr api.AppPageParametersResponse
		tc := plextest.NewTenantConfigBuilderFromTenantConfig(loadTenantConfig(t, tf)).
			SwitchToApp(0).SetAppPageParameter(pagetype.PlexLoginPage, param.CreateAccountText, "app login page").
			Build()
		sappr := newSaveAppPageParametersRequestFromTenantConfig(t, tc)

		// put request with bad tenant id
		assert.NotNil(t, c.Put(ctx, appPageParametersURL(uuid.Nil, uuid.Nil), sappr, &appr))

		// put request with bad app id
		assert.NotNil(t, c.Put(ctx, appPageParametersURL(tf.ConsoleTenantID, uuid.Nil), sappr, &appr))

		// put request with no page types
		sappr = api.SaveAppPageParametersRequest{PageTypeParameterChanges: api.ParameterChangeByNameByPageType{}}
		assert.NotNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr))

		// put request with empty page type
		sappr = api.SaveAppPageParametersRequest{PageTypeParameterChanges: api.ParameterChangeByNameByPageType{}}
		sappr.PageTypeParameterChanges[pagetype.EveryPage] = api.ParameterChangeByName{}
		assert.NotNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr))

		// put request with bad page type
		sappr = api.SaveAppPageParametersRequest{PageTypeParameterChanges: api.ParameterChangeByNameByPageType{}}
		sappr.PageTypeParameterChanges["bad page type"] = api.ParameterChangeByName{}
		sappr.PageTypeParameterChanges["bad page type"][param.CreateAccountText] = api.ParameterChange{Name: param.CreateAccountText, NewValue: "new value"}
		assert.NotNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr))

		// put request with invalid parameter name
		sappr = api.SaveAppPageParametersRequest{PageTypeParameterChanges: api.ParameterChangeByNameByPageType{}}
		sappr.PageTypeParameterChanges[pagetype.PlexMfaSubmitPage] = api.ParameterChangeByName{}
		sappr.PageTypeParameterChanges[pagetype.PlexMfaSubmitPage]["bad param name"] = api.ParameterChange{Name: "bad param name", NewValue: "new value"}
		assert.NotNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr))

		// put request with unsupported parameter name
		sappr = api.SaveAppPageParametersRequest{PageTypeParameterChanges: api.ParameterChangeByNameByPageType{}}
		sappr.PageTypeParameterChanges[pagetype.PlexMfaSubmitPage] = api.ParameterChangeByName{}
		sappr.PageTypeParameterChanges[pagetype.PlexMfaSubmitPage][param.AuthenticationMethods] = api.ParameterChange{Name: param.AuthenticationMethods, NewValue: "password"}
		assert.NotNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr))

		// put request with mismatched parameter name in key and value
		sappr = api.SaveAppPageParametersRequest{PageTypeParameterChanges: api.ParameterChangeByNameByPageType{}}
		sappr.PageTypeParameterChanges[pagetype.PlexMfaSubmitPage] = api.ParameterChangeByName{}
		sappr.PageTypeParameterChanges[pagetype.PlexMfaSubmitPage][param.HeadingText] = api.ParameterChange{Name: param.MFACodeLabel, NewValue: "new value"}
		assert.NotNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr))

		// put request with invalid parameter value
		sappr = api.SaveAppPageParametersRequest{PageTypeParameterChanges: api.ParameterChangeByNameByPageType{}}
		sappr.PageTypeParameterChanges[pagetype.PlexLoginPage] = api.ParameterChangeByName{}
		sappr.PageTypeParameterChanges[pagetype.PlexLoginPage][param.PasswordResetEnabled] = api.ParameterChange{Name: param.PasswordResetEnabled, NewValue: "bad value"}
		assert.NotNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr))

	})
}

func TestAuthenticationSettings(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tf, c := newAPIClient(ctx, t)

	// default tenant config does not have any social providers set up;
	// should be able to set password or passwordless as authentication
	// options, but not anything else

	tc := plextest.NewTenantConfigBuilderFromTenantConfig(loadTenantConfig(t, tf)).
		SwitchToApp(0).
		SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "passwordless").
		Build()

	sappr := newSaveAppPageParametersRequestFromTenantConfig(t, tc)

	var appr api.AppPageParametersResponse
	assert.IsNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr), assert.Must())
	verifyAppPageParametersResponse(t, tf.ConsoleTenantID, tc, appr)
	p := appr.PageTypeParameters[pagetype.EveryPage][param.AuthenticationMethods]
	verifyDefault(t, pagetype.EveryPage, p, tc)
	assert.Equal(t, p.CurrentValue, "passwordless")

	tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
		SwitchToApp(0).
		SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "google").
		Build()
	sappr = newSaveAppPageParametersRequestFromTenantConfig(t, tc)
	assert.NotNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr), assert.Must())

	tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
		SwitchToApp(0).
		SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "facebook").
		Build()
	sappr = newSaveAppPageParametersRequestFromTenantConfig(t, tc)
	assert.NotNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr), assert.Must())

	tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
		SwitchToApp(0).
		SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "google,password,passwordless").
		Build()
	sappr = newSaveAppPageParametersRequestFromTenantConfig(t, tc)
	assert.NotNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr), assert.Must())

	// set up google as a social provider
	cs, err := crypto.GenerateClientSecret(ctx, oidc.ProviderTypeGoogle.String())
	assert.IsNil(t, err)
	tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
		SwitchToOIDCProvider(oidc.ProviderTypeGoogle.String()).
		SetClientID(crypto.GenerateClientID()).
		SetClientSecret(*cs).
		SwitchToApp(0).
		SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "password,passwordless").
		Build()

	// first save the tenant config setting
	saveTenantConfigRequest := fmt.Sprintf("/api/tenants/%v/plexconfig", tf.ConsoleTenantID)
	tcRequest := api.SaveTenantPlexConfigRequest{TenantConfig: tc}

	var tcResponse api.SaveTenantPlexConfigResponse
	assert.NoErr(t, tcRequest.TenantConfig.UpdateUISettings(ctx))
	assert.IsNil(t, c.Post(ctx, saveTenantConfigRequest, tcRequest, &tcResponse), assert.Must())

	// now select google as an authentication method, which should succeed
	tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
		SwitchToApp(0).
		SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "google,password,passwordless").
		Build()

	sappr = newSaveAppPageParametersRequestFromTenantConfig(t, tc)
	assert.IsNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr), assert.Must())
	verifyAppPageParametersResponse(t, tf.ConsoleTenantID, tc, appr)
	p = appr.PageTypeParameters[pagetype.EveryPage][param.AuthenticationMethods]
	verifyDefault(t, pagetype.EveryPage, p, tc)
	assert.Equal(t, p.CurrentValue, "google,password,passwordless")

	// try updating the tenant config after clearing the social provider - this should now fail
	noSocialTC := plextest.NewTenantConfigBuilderFromTenantConfig(tc).
		SwitchToOIDCProvider(oidc.ProviderTypeGoogle.String()).SetClientID("").Build()
	tcRequest = api.SaveTenantPlexConfigRequest{TenantConfig: noSocialTC}
	assert.NoErr(t, tcRequest.TenantConfig.UpdateUISettings(ctx))
	assert.NotNil(t, c.Post(ctx, saveTenantConfigRequest, tcRequest, &tcResponse))

	// remove google as an authentication method, then successfully clear the social provider
	tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
		SwitchToApp(0).
		SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "password,passwordless").
		Build()

	sappr = newSaveAppPageParametersRequestFromTenantConfig(t, tc)
	assert.IsNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr), assert.Must())
	verifyAppPageParametersResponse(t, tf.ConsoleTenantID, tc, appr)
	p = appr.PageTypeParameters[pagetype.EveryPage][param.AuthenticationMethods]
	verifyDefault(t, pagetype.EveryPage, p, tc)
	assert.Equal(t, p.CurrentValue, "password,passwordless")

	tcRequest = api.SaveTenantPlexConfigRequest{TenantConfig: noSocialTC}
	assert.NoErr(t, tcRequest.TenantConfig.UpdateUISettings(ctx))
	assert.IsNil(t, c.Post(ctx, saveTenantConfigRequest, tcRequest, &tcResponse))

	// set up facebook as a social provider
	cs, err = crypto.GenerateClientSecret(ctx, oidc.ProviderTypeFacebook.String())
	assert.IsNil(t, err)
	tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
		SwitchToOIDCProvider(oidc.ProviderTypeFacebook.String()).
		SetClientID(crypto.GenerateClientID()).
		SetClientSecret(*cs).
		SwitchToApp(0).
		SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "password,passwordless").
		Build()

	tcRequest = api.SaveTenantPlexConfigRequest{TenantConfig: tc}
	assert.NoErr(t, tcRequest.TenantConfig.UpdateUISettings(ctx))
	assert.IsNil(t, c.Post(ctx, saveTenantConfigRequest, tcRequest, &tcResponse), assert.Must())

	// successfully add facebook as a social provider

	tc = plextest.NewTenantConfigBuilderFromTenantConfig(tc).
		SwitchToApp(0).
		SetAppPageParameter(pagetype.EveryPage, param.AuthenticationMethods, "facebook,password,passwordless").
		Build()

	sappr = newSaveAppPageParametersRequestFromTenantConfig(t, tc)
	assert.IsNil(t, c.Put(ctx, appPageParametersURLFromTenantConfig(tf.ConsoleTenantID, tc), sappr, &appr), assert.Must())
	verifyAppPageParametersResponse(t, tf.ConsoleTenantID, tc, appr)
	p = appr.PageTypeParameters[pagetype.EveryPage][param.AuthenticationMethods]
	verifyDefault(t, pagetype.EveryPage, p, tc)
	assert.Equal(t, p.CurrentValue, "facebook,password,passwordless")
}
