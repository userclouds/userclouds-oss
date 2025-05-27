package internal_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/oidc"
	"userclouds.com/internal/pageparameters/pagetype"
	param "userclouds.com/internal/pageparameters/parameter"
	"userclouds.com/internal/tenantplex"
	plextest "userclouds.com/internal/tenantplex/test"
	"userclouds.com/internal/uctest"
	"userclouds.com/plex/internal"
	"userclouds.com/plex/internal/test"
	"userclouds.com/plex/manager"
)

type testFixture struct {
	t         *testing.T
	tf        *test.Fixture
	sessionID uuid.UUID
	tenantID  uuid.UUID
	clientID  string
}

func newFixture(t *testing.T) testFixture {
	redirectURL := genRedirectURL()
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	cs, err := crypto.GenerateClientSecret(context.Background(), oidc.ProviderTypeGoogle.String())
	assert.NoErr(t, err)
	tcb.SwitchToOIDCProvider(oidc.ProviderTypeGoogle.String()).
		SetClientID(crypto.GenerateClientID()).
		SetClientSecret(*cs).
		SwitchToApp(0).SetName("Test App").
		AddAllowedRedirectURI(redirectURL)

	tc := tcb.Build()
	tf := test.NewFixture(t, tc)
	tenantID := tf.Tenant.ID
	sessionID := createOIDCLoginSession(t, tf.Storage, redirectURL, clientID)

	return testFixture{
		t:         t,
		tf:        tf,
		sessionID: sessionID,
		tenantID:  tenantID,
		clientID:  clientID,
	}
}

func (f *testFixture) loadTenantPlex() *tenantplex.TenantPlex {
	f.t.Helper()

	ctx := context.Background()

	mgr := manager.NewFromDB(f.tf.TenantDB, cachetesthelpers.NewCacheConfig())
	tp, err := mgr.GetTenantPlex(ctx, f.tenantID)
	assert.NoErr(f.t, err)
	return tp
}

func (f *testFixture) saveTenantPlex(tp *tenantplex.TenantPlex) *tenantplex.TenantPlex {
	f.t.Helper()

	ctx := context.Background()

	mgr := manager.NewFromDB(f.tf.TenantDB, cachetesthelpers.NewCacheConfig())
	err := mgr.SaveTenantPlex(ctx, tp)
	assert.NoErr(f.t, err)
	return f.loadTenantPlex()
}

func (f *testFixture) sendBadRequest(pt pagetype.Type, names []param.Name) {
	f.t.Helper()

	response := f.sendRequest(pt, names)
	assert.Equal(f.t, response.StatusCode, http.StatusBadRequest)
}

func (f *testFixture) sendGoodRequest(pt pagetype.Type, names []param.Name) internal.PageParametersResponse {
	f.t.Helper()

	response := f.sendRequest(pt, names)
	assert.Equal(f.t, response.StatusCode, http.StatusOK)
	var ppResponse internal.PageParametersResponse
	assert.NoErr(f.t, json.NewDecoder(response.Body).Decode(&ppResponse))
	assert.Equal(f.t, ppResponse.ClientID, f.clientID)
	return ppResponse
}

func (f *testFixture) sendRequest(pt pagetype.Type, names []param.Name) *http.Response {
	f.t.Helper()

	ppr := internal.PageParametersRequest{
		SessionID:      f.sessionID,
		PageType:       pt,
		ParameterNames: names,
	}
	request := f.tf.RequestFactory.NewRequest(http.MethodPost, "/login/pageparameters", uctest.IOReaderFromJSONStruct(f.t, ppr))
	w := httptest.NewRecorder()
	f.tf.Handler.ServeHTTP(w, request)
	return w.Result()
}

func (f *testFixture) newResponseVerifier(pt pagetype.Type, names []param.Name) *responseVerifier {
	return newResponseVerifier(f.t, pt, names).
		setExpectedValue(param.EnabledAuthenticationMethods, "password,passwordless,google").
		setExpectedValue(param.DisabledAuthenticationMethods, "facebook,linkedin,microsoft")
}

type responseVerifier struct {
	t                  *testing.T
	expectedParameters map[param.Name]param.Parameter
}

func newResponseVerifier(t *testing.T, pt pagetype.Type, names []param.Name) *responseVerifier {
	t.Helper()

	testParameters := pt.TestRenderParameters()
	expectedParameters := map[param.Name]param.Parameter{}
	for _, n := range names {
		p, found := testParameters[n]
		assert.True(t, found)
		expectedParameters[n] = p
	}

	return &responseVerifier{
		t:                  t,
		expectedParameters: expectedParameters,
	}
}

func (rv *responseVerifier) setExpectedValue(n param.Name, v string) *responseVerifier {
	rv.t.Helper()

	p, found := rv.expectedParameters[n]
	assert.True(rv.t, found)
	p.Value = v
	rv.expectedParameters[n] = p
	return rv
}

func (rv *responseVerifier) verifyResponse(response internal.PageParametersResponse) {
	rv.t.Helper()

	for _, rp := range response.PageParameters {
		ep, found := rv.expectedParameters[rp.Name]
		assert.True(rv.t, found)
		assert.Equal(rv.t, rp.Name, ep.Name)
		assert.Equal(rv.t, rp.Type, ep.Type)
		assert.Equal(rv.t, rp.Value, ep.Value)
	}
}

func TestDefaultEveryPageParameters(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	paramNames := pagetype.EveryPage.RenderParameterNames()
	response := f.sendGoodRequest(pagetype.EveryPage, paramNames)
	f.newResponseVerifier(pagetype.EveryPage, paramNames).verifyResponse(response)
}

func TestDefaultPlexCreateUserPageParameters(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	paramNames := pagetype.PlexCreateUserPage.RenderParameterNames()
	response := f.sendGoodRequest(pagetype.PlexCreateUserPage, paramNames)
	f.newResponseVerifier(pagetype.PlexCreateUserPage, paramNames).
		setExpectedValue(param.AllowCreation, "true").
		verifyResponse(response)
}

func TestDefaultPlexFinishResetPasswordPageParameters(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	paramNames := pagetype.PlexFinishResetPasswordPage.RenderParameterNames()
	response := f.sendGoodRequest(pagetype.PlexFinishResetPasswordPage, paramNames)
	f.newResponseVerifier(pagetype.PlexFinishResetPasswordPage, paramNames).verifyResponse(response)
}

func TestDefaultPlexLoginPageParameters(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	paramNames := pagetype.PlexLoginPage.RenderParameterNames()
	response := f.sendGoodRequest(pagetype.PlexLoginPage, paramNames)
	f.newResponseVerifier(pagetype.PlexLoginPage, paramNames).
		setExpectedValue(param.AllowCreation, "true").
		setExpectedValue(param.HeadingText, "Sign in to Test App").
		setExpectedValue(param.PasswordResetEnabled, "true").
		verifyResponse(response)
}

func TestDefaultPlexMfaChannelPageParameters(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	paramNames := pagetype.PlexMfaChannelPage.RenderParameterNames()
	response := f.sendGoodRequest(pagetype.PlexMfaChannelPage, paramNames)
	f.newResponseVerifier(pagetype.PlexMfaChannelPage, paramNames).verifyResponse(response)
}

func TestDefaultPlexMfaSubmitPageParameters(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	paramNames := pagetype.PlexMfaSubmitPage.RenderParameterNames()
	response := f.sendGoodRequest(pagetype.PlexMfaSubmitPage, paramNames)
	f.newResponseVerifier(pagetype.PlexMfaSubmitPage, paramNames).verifyResponse(response)
}

func TestDefaultPlexPasswordlessLoginPageParameters(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	paramNames := pagetype.PlexPasswordlessLoginPage.RenderParameterNames()
	response := f.sendGoodRequest(pagetype.PlexPasswordlessLoginPage, paramNames)
	f.newResponseVerifier(pagetype.PlexPasswordlessLoginPage, paramNames).
		setExpectedValue(param.HeadingText, "Sign in to Test App").
		verifyResponse(response)
}

func TestDefaultPlexStartResetPasswordPageParameters(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	paramNames := pagetype.PlexStartResetPasswordPage.RenderParameterNames()
	response := f.sendGoodRequest(pagetype.PlexStartResetPasswordPage, paramNames)
	f.newResponseVerifier(pagetype.PlexStartResetPasswordPage, paramNames).verifyResponse(response)
}

func TestOverrides(t *testing.T) {
	t.Parallel()

	paramNames := []param.Name{param.SubheadingText}
	verifier := newResponseVerifier(t, pagetype.PlexLoginPage, paramNames)

	// set a tenant specific override for login page
	f := newFixture(t)
	tp := f.loadTenantPlex()
	tp.PlexConfig = plextest.NewTenantConfigBuilderFromTenantConfig(tp.PlexConfig).
		SetTenantPageParameter(pagetype.PlexLoginPage, param.SubheadingText, "tenant subheading text").
		Build()
	f.saveTenantPlex(tp)

	response := f.sendGoodRequest(pagetype.PlexLoginPage, paramNames)
	verifier.setExpectedValue(param.SubheadingText, "tenant subheading text").verifyResponse(response)

	// set an app specific override for login page
	f = newFixture(t)
	tp = f.loadTenantPlex()
	tp.PlexConfig = plextest.NewTenantConfigBuilderFromTenantConfig(tp.PlexConfig).
		SetTenantPageParameter(pagetype.PlexLoginPage, param.SubheadingText, "tenant subheading text").
		SwitchToApp(0).SetAppPageParameter(pagetype.PlexLoginPage, param.SubheadingText, "app subheading text").
		Build()
	f.saveTenantPlex(tp)

	response = f.sendGoodRequest(pagetype.PlexLoginPage, paramNames)
	verifier.setExpectedValue(param.SubheadingText, "app subheading text").verifyResponse(response)

	// remove app login page override
	f = newFixture(t)
	tp = f.loadTenantPlex()
	tp.PlexConfig = plextest.NewTenantConfigBuilderFromTenantConfig(tp.PlexConfig).
		SetTenantPageParameter(pagetype.PlexLoginPage, param.SubheadingText, "tenant subheading text").
		SwitchToApp(0).SetAppPageParameter(pagetype.PlexLoginPage, param.SubheadingText, "app subheading text").
		DeleteAppPageParameter(pagetype.PlexLoginPage, param.SubheadingText).
		Build()
	f.saveTenantPlex(tp)

	response = f.sendGoodRequest(pagetype.PlexLoginPage, paramNames)
	verifier.setExpectedValue(param.SubheadingText, "tenant subheading text").verifyResponse(response)

	// remove tenant login page override
	f = newFixture(t)
	tp = f.loadTenantPlex()
	tp.PlexConfig = plextest.NewTenantConfigBuilderFromTenantConfig(tp.PlexConfig).
		SetTenantPageParameter(pagetype.PlexLoginPage, param.SubheadingText, "tenant subheading text").
		SwitchToApp(0).SetAppPageParameter(pagetype.PlexLoginPage, param.SubheadingText, "app subheading text").
		DeleteAppPageParameter(pagetype.PlexLoginPage, param.SubheadingText).
		DeleteTenantPageParameter(pagetype.PlexLoginPage, param.SubheadingText).
		Build()
	f.saveTenantPlex(tp)

	response = f.sendGoodRequest(pagetype.PlexLoginPage, paramNames)
	verifier.setExpectedValue(param.SubheadingText, "").verifyResponse(response)
}

func TestBadPageType(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	names := []param.Name{param.AllowCreation}
	f.sendBadRequest("Bad Page Type", names)
}

func TestBadParamaterName(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	names := []param.Name{"Bad Parameter Name"}
	f.sendBadRequest(pagetype.PlexLoginPage, names)
}

func TestDuplicateParamaterName(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	names := []param.Name{param.AllowCreation, param.AllowCreation}
	f.sendBadRequest(pagetype.PlexLoginPage, names)
}
