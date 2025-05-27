package social_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"
	"gopkg.in/square/go-jose.v2"

	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/internal/tenantplex/builder"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/testkeys"
	"userclouds.com/internal/uctest"
	"userclouds.com/plex"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/social"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/test"
	"userclouds.com/plex/internal/wellknown"
	"userclouds.com/plex/manager"
)

const inviterEmail = "earlyadopter@contoso.com"

func createLoginSession(tf *test.Fixture, state, redirectURL, clientID string, rts storage.ResponseTypes) (uuid.UUID, error) {
	ctx := context.Background()
	sessionID, err := storage.CreateOIDCLoginSession(
		ctx, tf.Storage, clientID, rts,
		uctest.MustParseURL(redirectURL), state, "unusedscope")
	if err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}
	return sessionID, nil
}

func makeSocialLoginURL(sessionID uuid.UUID, tenantURL string, provider oidc.ProviderType) string {
	socialLoginStartURL := uctest.MustParseURL(tenantURL)
	socialLoginStartURL.Path = "/social/login"
	socialLoginStartURL.RawQuery = url.Values{
		"session_id":    []string{sessionID.String()},
		"oidc_provider": []string{provider.String()},
	}.Encode()

	return socialLoginStartURL.String()
}

func startOIDCLogin(tf *test.Fixture, sessionID uuid.UUID, tenantURL string, provider oidc.ProviderType) *httptest.ResponseRecorder {
	socialLoginStartURL := makeSocialLoginURL(sessionID, tenantURL, provider)

	r := tf.RequestFactory.NewRequest(http.MethodGet, socialLoginStartURL, nil)
	rr := httptest.NewRecorder()

	tf.Handler.ServeHTTP(rr, r)
	return rr
}

func newCheckRedirectHTTPClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
}

// TODO: if we need this elsewhere we can refactor this into a standalone mock IDP?
// Or even set up a 2nd Plex server/tenant and use that as our mock IDP?
func mockSocialIDPHandler(t *testing.T, w http.ResponseWriter, r *http.Request,
	expectedClientID, expectedState, expectedRedirectURL, authCode string,
	expectedUserClaims iface.UserProfile, serverURL *string, lastJWT *string) {
	ctx := r.Context()
	claims := oidc.UCTokenClaims{
		Email: expectedUserClaims.Email,
		StandardClaims: oidc.StandardClaims{
			Audience:         []string{expectedClientID},
			RegisteredClaims: jwt.RegisteredClaims{Subject: expectedUserClaims.ID},
		},
	}

	switch r.URL.Path {
	case "/authorize":
		// Social login uses the Auth code flow from Plex -> Social IDP
		assert.Equal(t, r.URL.Query().Get("response_type"), "code")
		// Plex internally passes the session ID as 'state' to the underlying social IDP;
		// testing for it here is an internal implementation detail but it helps ensure we aren't
		// mixing up our states at least.
		assert.Equal(t, r.URL.Query().Get("state"), expectedState)
		assert.Equal(t, r.URL.Query().Get("client_id"), expectedClientID)
		assert.Equal(t, r.URL.Query().Get("redirect_uri"), expectedRedirectURL)
		redirectTo := uctest.MustParseURL(r.URL.Query().Get("redirect_uri"))
		redirectTo.RawQuery = url.Values{
			"state": []string{expectedState},
			"code":  []string{authCode},
		}.Encode()
		uchttp.Redirect(w, r, redirectTo.String(), http.StatusTemporaryRedirect)
	case "/.well-known/openid-configuration":
		providerJSON := wellknown.OpenIDProviderJSON{
			Issuer:       *serverURL,
			AuthURL:      *serverURL + "/authorize",
			TokenURL:     *serverURL + "/token",
			JWKSURL:      *serverURL + "/.well-known/jwks",
			UserInfoURL:  *serverURL + "/userinfo",
			Algorithms:   []string{"RS256"},
			SubjectTypes: []string{"public"},
			Scopes:       []string{"openid", "profile"},
			Claims:       []string{"iss", "sub", "aud", "exp", "iat", "auth_time", "nonce", "name", "email"},
		}
		jsonapi.Marshal(w, providerJSON)

	case "/.well-known/jwks":
		pubKey, err := ucjwt.LoadRSAPublicKey([]byte(testkeys.Config.PublicKey))
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "FailedToLoadPublicKey")
			return
		}
		keyset := &jose.JSONWebKeySet{
			Keys: []jose.JSONWebKey{
				{
					Algorithm: "RS256",
					Key:       pubKey,
					Use:       "sig",
					KeyID:     testkeys.Config.KeyID,
				},
			},
		}
		jsonapi.Marshal(w, keyset)
	case "/token":
		err := r.ParseForm()
		assert.NoErr(t, err)
		assert.Equal(t, r.PostForm.Get("code"), authCode)
		assert.Equal(t, r.PostForm.Get("redirect_uri"), expectedRedirectURL)

		jwt := uctest.CreateJWT(t, claims, *serverURL)
		*lastJWT = jwt
		jsonapi.Marshal(w, oidc.TokenResponse{
			AccessToken: crypto.GenerateOpaqueAccessToken(),
			TokenType:   "Bearer",
			IDToken:     jwt,
		})

	case "/userinfo":
		jsonapi.Marshal(w, claims)
	}
}

type testFixture struct {
	test.Fixture

	redirectURL                         string
	plexClientID, plexSocialCallbackURL string
	plexClientSecret                    string
	socialClientID, socialClientSecret  string
	authCode                            string
	expectedSocialState                 *string
	socialUserProfile                   *iface.UserProfile
	tcb                                 *builder.TenantConfigBuilder
	socialIDPServerURL                  string

	// TODO: ugh this is ugly, but I don't have a better solution yet and need to ship
	// this is a pointer because we create the string before we have this
	// object set up, and we need to be able to set it at runtime in mockSocialIDPHandler
	// That part is easy enough to fix but the "lastJWT" part seems especially janky.
	lastJWT *string
}

func newTestFixture(t *testing.T) *testFixture {
	t.Helper()
	ctx := context.Background()

	socialClientID := crypto.GenerateClientID()
	socialClientSecret, err := crypto.GenerateClientSecret(ctx, socialClientID)
	assert.NoErr(t, err)
	scs, err := socialClientSecret.Resolve(ctx)
	assert.NoErr(t, err)

	authCode := crypto.GenerateOpaqueAccessToken()
	socialUserProfile := &iface.UserProfile{
		ID:              fmt.Sprintf("socialuser%s", crypto.MustRandomDigits(6)),
		UserBaseProfile: idp.UserBaseProfile{Email: fmt.Sprintf("user%s@contoso.com", crypto.MustRandomDigits(6))},
	}
	redirectURL := fmt.Sprintf("http://contoso_%s.com/callback", crypto.MustRandomHex(4))
	tcb, plexClientID := test.NewBasicTenantConfigBuilder()
	tcb.SwitchToOIDCProvider(oidc.ProviderTypeGoogle.String()).
		SetClientID(socialClientID).
		SetClientSecret(*socialClientSecret)
	plexClientSecret := tcb.SwitchToApp(0).AddAllowedRedirectURI(redirectURL).ClientSecret()
	pcs, err := plexClientSecret.Resolve(ctx)
	assert.NoErr(t, err)
	tc := tcb.Build()
	tf := test.NewFixture(t, tc)

	// Start a real server with the Plex handler so that we can ensure redirect URLs can be followed as needed.
	plexServer := httptest.NewServer(tf.Handler)
	t.Cleanup(plexServer.Close)

	// Update tenant configs now that we have a real Plex URL
	testhelpers.UpdateTenantURL(ctx, t, tf.CompanyConfigStorage, tf.Tenant, plexServer)
	plexSocialCallbackURL := fmt.Sprintf("%s%s%s", tf.Tenant.TenantURL, paths.SocialRootPath, paths.SocialCallbackSubPath)

	expectedSocialState := new(string)

	// Make a fake Social IDP server. We don't know its URL til its running, but the server itself needs
	// to know its own URL for token signing purposes.
	socialIDPServerURL := new(string)
	var lastJWT string
	socialIDPServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockSocialIDPHandler(t, w, r, socialClientID, *expectedSocialState,
			plexSocialCallbackURL, authCode, *socialUserProfile, socialIDPServerURL, &lastJWT)
	}))
	t.Cleanup(socialIDPServer.Close)
	*socialIDPServerURL = socialIDPServer.URL

	// Hook in to the client that Plex uses to talk to the Google Social Provider and substitute our own
	// fake Social IDP Server.
	authr, err := oidc.NewAuthenticator(ctx, *socialIDPServerURL, socialClientID, *socialClientSecret, plexSocialCallbackURL)
	assert.NoErr(t, err)
	tf.OverrideOIDC = map[oidc.ProviderType]*oidc.Authenticator{oidc.ProviderTypeGoogle: authr}

	return &testFixture{
		Fixture:               *tf,
		redirectURL:           redirectURL,
		plexClientID:          plexClientID,
		plexSocialCallbackURL: plexSocialCallbackURL,
		plexClientSecret:      pcs,
		socialClientID:        socialClientID,
		socialClientSecret:    scs,
		expectedSocialState:   expectedSocialState,
		authCode:              authCode,
		socialUserProfile:     socialUserProfile,
		tcb:                   tcb,
		socialIDPServerURL:    *socialIDPServerURL,
		lastJWT:               &lastJWT,
	}
}

func (stf *testFixture) createLoginSession(t *testing.T, state string) uuid.UUID {
	t.Helper()
	sessionID, err := createLoginSession(&stf.Fixture, state, stf.redirectURL, stf.plexClientID,
		storage.ResponseTypes{storage.AuthorizationCodeResponseType, storage.TokenResponseType, storage.IDTokenResponseType})
	assert.NoErr(t, err)
	// Plex internally passes the session ID and tenant URL as 'state' to the underlying social IDP;
	// testing for it here is an internal implementation detail but it helps ensure we aren't
	// mixing up our states at least.
	*stf.expectedSocialState = social.EncodeState(sessionID, stf.Tenant.TenantURL)
	return sessionID
}

func (stf *testFixture) createInvitedLoginSession(t *testing.T, state string) uuid.UUID {
	ctx := context.Background()
	inviterUserID, err := stf.ActiveIDP.CreateUserWithPassword(ctx, inviterEmail, "pw", iface.UserProfile{UserBaseProfile: idp.UserBaseProfile{Email: inviterEmail}})
	assert.NoErr(t, err)
	n := len(stf.Email.Bodies)
	ts, err := jsonclient.ClientCredentialsForURL(stf.Tenant.TenantURL, stf.plexClientID, stf.plexClientSecret, nil)
	assert.NoErr(t, err)
	plexClient := plex.NewClient(stf.Tenant.TenantURL, ts)
	assert.NoErr(t, plexClient.SendInvite(ctx, plex.SendInviteRequest{
		InviteeEmail:  "doesntmatter@contoso.com",
		InviterUserID: inviterUserID,
		InviterName:   "Inviter Name",
		InviterEmail:  "inviter@contoso.com",
		ClientID:      stf.plexClientID,
		State:         state,
		RedirectURL:   stf.redirectURL,
	}))

	// Pull the session ID out of the emailed link
	assert.Equal(t, len(stf.Email.Bodies), n+1, assert.Errorf("expected plex client SendInvite to send an email"), assert.Must())
	magicLink, err := uctest.ExtractURL(stf.Email.HTMLBodies[n])
	assert.NoErr(t, err)
	sessionID, err := uuid.FromString(magicLink.Query().Get("session_id"))
	assert.NoErr(t, err)
	*stf.expectedSocialState = social.EncodeState(sessionID, stf.Tenant.TenantURL)
	return sessionID
}

func TestGoogleLogin(t *testing.T) {
	ctx := context.Background()

	googleClientID := crypto.GenerateClientID()
	googleClientSecret, err := crypto.GenerateClientSecret(ctx, oidc.ProviderTypeGoogle.String())
	assert.NoErr(t, err)
	state := crypto.GenerateOpaqueAccessToken()

	redirectURL := fmt.Sprintf("http://contoso_%s.com/callback", crypto.MustRandomHex(4))
	tcb, plexClientID := test.NewBasicTenantConfigBuilder()
	tcb.SwitchToOIDCProvider(oidc.ProviderTypeGoogle.String()).
		SetClientID(googleClientID).
		SetClientSecret(*googleClientSecret).
		SwitchToApp(0).AddAllowedRedirectURI(redirectURL)
	tc := tcb.Build()
	tf := test.NewFixture(t, tc)

	sessionID, err := createLoginSession(tf, state, redirectURL, plexClientID, storage.ResponseTypes{storage.IDTokenResponseType})
	assert.NoErr(t, err)

	// Initiate login with Google, ensure we get redirected to the right place with sane parameters.
	// TODO: didn't realize at the time but this makes a GET request the social IDP (specifically, to fetch
	// .well-known/openid-configuration) as part of initializing the `oidc.Authenticator` object even though
	// it's not actually redirecting/logging in. So this may make the test flaky in certain test environments.
	rr := startOIDCLogin(tf, sessionID, tf.Tenant.TenantURL, oidc.ProviderTypeGoogle)
	assert.Equal(t, rr.Result().StatusCode, http.StatusTemporaryRedirect)

	redirectTo, err := rr.Result().Location()
	assert.NoErr(t, err)
	assert.Contains(t, redirectTo.String(), oidc.ProviderTypeGoogle.GetDefaultIssuerURL())
	plexRedirect := url.QueryEscape(fmt.Sprintf("%s%s%s", tf.Tenant.TenantURL, paths.SocialRootPath, paths.SocialCallbackSubPath))
	assert.Contains(t, redirectTo.RawQuery, fmt.Sprintf("redirect_uri=%s", plexRedirect))
	assert.Contains(t, redirectTo.RawQuery, fmt.Sprintf("client_id=%s", googleClientID))
}

func TestSocialFlowCreateDisabled(t *testing.T) {
	stf := newTestFixture(t)
	ctx := context.Background()
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Stop following redirects at the last one
			if strings.Contains(req.URL.String(), stf.redirectURL) {
				return http.ErrUseLastResponse
			}
			return nil
		}}

	// Disable sign ups on tenant for this test.
	tc := stf.tcb.SetDisableSignUps(true).Build()
	mgr := manager.NewFromDB(stf.TenantDB, cachetesthelpers.NewCacheConfig())
	tp, err := mgr.GetTenantPlex(ctx, stf.Tenant.ID)
	assert.NoErr(t, err)
	tp.PlexConfig = tc
	assert.NoErr(t, mgr.SaveTenantPlex(ctx, tp))

	expectedState := social.EncodeState(uuid.Nil, stf.Tenant.TenantURL)

	// Social account creation not allowed without an invite
	sessionID := stf.createLoginSession(t, expectedState)
	socialLoginResponse, err := client.Get(makeSocialLoginURL(sessionID, stf.Tenant.TenantURL, oidc.ProviderTypeGoogle))
	assert.NoErr(t, err)
	assert.Equal(t, socialLoginResponse.StatusCode, http.StatusForbidden, assert.Must())

	// Social account creation IS allowed WITH an invite
	sessionID = stf.createInvitedLoginSession(t, expectedState)

	socialLoginResponse, err = client.Get(makeSocialLoginURL(sessionID, stf.Tenant.TenantURL, oidc.ProviderTypeGoogle))
	assert.NoErr(t, err)
	assert.Equal(t, socialLoginResponse.StatusCode, http.StatusSeeOther, assert.Must())

	// Login without an invite, but with an existing account, is allowed
	sessionID = stf.createLoginSession(t, expectedState)
	socialLoginResponse, err = client.Get(makeSocialLoginURL(sessionID, stf.Tenant.TenantURL, oidc.ProviderTypeGoogle))
	assert.NoErr(t, err)
	assert.Equal(t, socialLoginResponse.StatusCode, http.StatusSeeOther, assert.Must())

	// Login with a random new user, but without an invite, still fails
	stf.socialUserProfile.ID = fmt.Sprintf("anothersocialuser%s", crypto.MustRandomDigits(6))
	stf.socialUserProfile.Email = fmt.Sprintf("anotheruser%s@contoso.com", crypto.MustRandomDigits(6))
	sessionID = stf.createLoginSession(t, expectedState)
	socialLoginResponse, err = client.Get(makeSocialLoginURL(sessionID, stf.Tenant.TenantURL, oidc.ProviderTypeGoogle))
	assert.NoErr(t, err)
	assert.Equal(t, socialLoginResponse.StatusCode, http.StatusForbidden, assert.Must())
}

func TestSocialFlow(t *testing.T) {
	ctx := context.Background()
	stf := newTestFixture(t)
	client := newCheckRedirectHTTPClient()

	// Create a login session for a hypothetical client App to log into Plex.
	// Use the Hybrid flow (Code + Token + ID Token) to ensure we get all 3 values back.
	// Use a generated random plexState & client ID/secret value to ensure we don't cross the streams and mix up the Client -> Plex
	// plexState & client values with the Plex -> Social IDP plexState (there are 2 nested OIDC flows happening here).
	plexState := crypto.GenerateOpaqueAccessToken()
	sessionID := stf.createLoginSession(t, plexState)

	// Initiate social login on Plex (with Google as provider) and ensure we get redirected to the
	// right place with sane parameters.
	socialLoginResponse, err := client.Get(makeSocialLoginURL(sessionID, stf.Tenant.TenantURL, oidc.ProviderTypeGoogle))
	assert.NoErr(t, err)
	assert.Equal(t, socialLoginResponse.StatusCode, http.StatusTemporaryRedirect, assert.Must())

	// Plex should should redirect us to the mock social IDP server
	idpAuthorizeURL, err := socialLoginResponse.Location()
	assert.NoErr(t, err)
	assert.Contains(t, idpAuthorizeURL.String(), stf.socialIDPServerURL)
	assert.Contains(t, idpAuthorizeURL.Query().Get("state"), *stf.expectedSocialState)
	assert.Contains(t, idpAuthorizeURL.Query().Get("client_id"), stf.socialClientID)

	// Follow the redirect from Plex to hit the mock social IDP server's authorize endpoint.
	socialIDPResponse, err := client.Get(idpAuthorizeURL.String())
	assert.NoErr(t, err)
	assert.Equal(t, socialIDPResponse.StatusCode, http.StatusTemporaryRedirect)

	// The mock IDP should redirect us back to Plex (social callback handler) with an auth code.
	socialIDPRedirectURL, err := socialIDPResponse.Location()
	assert.NoErr(t, err)
	assert.Contains(t, socialIDPRedirectURL.String(), stf.plexSocialCallbackURL)
	assert.Equal(t, socialIDPRedirectURL.Query().Get("code"), stf.authCode)
	assert.Equal(t, socialIDPRedirectURL.Query().Get("state"), *stf.expectedSocialState)

	// Follow the redirect from Social IDP -> Plex, which should cause a bunch of things to happen:
	// 1. Plex should exchange the code from the mock IDP for a token.
	// 2. Plex should hit the user info endpoint.
	// 3. Plex should create tokens and ultimately return them to the client Application
	plexCallbackResponse, err := client.Get(socialIDPRedirectURL.String())
	assert.NoErr(t, err)
	assert.Equal(t, plexCallbackResponse.StatusCode, http.StatusSeeOther, assert.Must())
	plexCallbackRedirectURL, err := plexCallbackResponse.Location()
	assert.NoErr(t, err)
	assert.Contains(t, plexCallbackRedirectURL.String(), stf.redirectURL)
	plexAuthCode := plexCallbackRedirectURL.Query().Get("code")
	assert.True(t, len(plexAuthCode) > 0)
	assert.Equal(t, plexCallbackRedirectURL.Query().Get("state"), plexState)
	plexToken, err := stf.Storage.GetPlexTokenForAuthCode(ctx, plexAuthCode)
	assert.NoErr(t, err)
	assert.Equal(t, plexCallbackRedirectURL.Query().Get("token"), plexToken.AccessToken)
	assert.Equal(t, plexCallbackRedirectURL.Query().Get("id_token"), plexToken.IDToken)

	// TODO: there are many things wrong with this test, but it's a start.
	// 1) the way we get the "lastJWT" from the fixture is a total hack, and will
	// break as soon as we parallelize these tests.
	// 2) we don't validate the whole object because the oidc.TokenInfo object is
	// sort of built up inline in infra/oidc/auth.go and it's not easy to extract
	var ti oidc.TokenInfo
	assert.NoErr(t, json.Unmarshal([]byte(plexToken.UnderlyingToken), &ti))
	assert.Equal(t, ti.RawIDToken, *stf.lastJWT)

	c := jsonclient.New(stf.Tenant.TenantURL, jsonclient.HeaderAuthBearer(plexToken.AccessToken))
	assert.NoErr(t, c.Get(ctx, "/social/underlying", &ti))
	assert.Equal(t, ti.RawIDToken, *stf.lastJWT)

	// Ensure the token from Plex reflects the social user's data, and that our test IDP does too.
	tokenClaims, err := ucjwt.ParseUCClaimsUnverified(plexCallbackRedirectURL.Query().Get("id_token"))
	assert.NoErr(t, err)
	assert.Equal(t, tokenClaims.Email, stf.socialUserProfile.Email)
	assert.Equal(t, len(stf.ActiveIDP.Users), 1, assert.Must())
	for _, v := range stf.ActiveIDP.Users {
		assert.Equal(t, v, test.User{
			Authn: idp.NewOIDCAuthn(oidc.ProviderTypeGoogle, oidc.ProviderTypeGoogle.GetDefaultIssuerURL(), stf.socialUserProfile.ID),
			Profile: userstore.Record{
				"email":          stf.socialUserProfile.Email,
				"email_verified": false,
				"name":           "",
				"nickname":       "",
				"picture":        "",
			},
		})
	}
}

func TestMain(m *testing.M) {
	// Adjust working dir to match what our services expect.
	os.Chdir("../../..")
	os.Exit(m.Run())
}
