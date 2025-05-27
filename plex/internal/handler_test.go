package internal_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/pageparameters/pagetype"
	"userclouds.com/internal/pageparameters/parameter"
	"userclouds.com/internal/tenantmap"
	plexconfigtest "userclouds.com/internal/tenantplex/test"
	"userclouds.com/internal/uctest"
	"userclouds.com/plex"
	"userclouds.com/plex/internal"
	"userclouds.com/plex/internal/otp"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
	"userclouds.com/plex/internal/test"
	"userclouds.com/plex/internal/token"
	"userclouds.com/plex/manager"
)

const testState = "teststate1234"
const testScope = "openid profile email"
const testName = "test name"
const testNickname = "test nickname"
const testPicture = "http://picture.com/1234"
const testEmail = "test@contoso.com"
const otpSubmitPath = otp.RootPath + otp.SubmitSubPath
const testCacheDuration = 10 * time.Millisecond

func init() {
	tenantconfig.TESTONLYSetTenantConfigCacheDuration(testCacheDuration)
}

func createOIDCLoginSession(t *testing.T, s *storage.Storage, redirectURL, clientID string) (sessionID uuid.UUID) {
	sessionID, err :=
		storage.CreateOIDCLoginSession(context.Background(), s, clientID,
			[]storage.ResponseType{storage.AuthorizationCodeResponseType},
			uctest.MustParseURL(redirectURL), testState, "unusedscope")
	assert.NoErr(t, err)
	return sessionID
}

func createLoginRequest(t *testing.T, s *storage.Storage, redirectURL, clientID, username, password string) plex.LoginRequest {
	return plex.LoginRequest{
		Username:  username,
		Password:  password,
		SessionID: createOIDCLoginSession(t, s, redirectURL, clientID),
	}
}

func parseLoginResponse(t *testing.T, resp *http.Response) *url.URL {
	var lresp plex.LoginResponse
	err := json.NewDecoder(resp.Body).Decode(&lresp)
	assert.NoErr(t, err)
	return uctest.MustParseURL(lresp.RedirectTo)
}

// if non-empty, clientID & secret here will be used for BasicAuth (an alternative OAuth client auth system)
// this isn't the best factoring but it seems ok for now?
func tokenExchange(tf *test.Fixture, query url.Values, clientID, clientSecret string) (*oidc.TokenResponse, error) {
	// both or neither client ID/secret required
	if (clientID == "") != (clientSecret == "") {
		return nil, ucerr.New("can't pass only one of clientID and clientSecret")
	}

	req := tf.RequestFactory.NewRequest(http.MethodPost, "/oidc/token", strings.NewReader(query.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	if clientID != "" {
		// per RFC 6749 section 2.3.1, client ID and secret need to be form-urlencoded :/
		cid := url.QueryEscape(clientID)
		csec := url.QueryEscape(clientSecret)
		req.SetBasicAuth(cid, csec)
	}

	w := httptest.NewRecorder()
	tf.Handler.ServeHTTP(w, req)
	resp := w.Result()
	if resp.StatusCode >= http.StatusBadRequest {
		assert.Equal(tf.Testing, resp.Header.Get("Content-Type"), "application/json")
		var oauthe ucerr.OAuthError
		// OAuth standard requires us to return a body with error descriptions
		// in many cases, so try to decode response but ignore the error if it fails.
		innerErr := json.NewDecoder(resp.Body).Decode(&oauthe)
		if innerErr == nil {
			oauthe.Code = resp.StatusCode
			return nil, oauthe
		} else {
			return nil, ucerr.Errorf("/token endpoint returned unexpected status code %d", resp.StatusCode)
		}
	}

	var tresp oidc.TokenResponse
	err := json.NewDecoder(resp.Body).Decode(&tresp)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &tresp, ucerr.Wrap(err)
}

func buildAuthCodeTokenExchangeQuery(redirectURL, code, codeVerifier string) url.Values {
	query := url.Values{}
	query.Add("grant_type", "authorization_code")
	query.Add("code", code)
	if len(codeVerifier) > 0 {
		query.Add("code_verifier", codeVerifier)
	}
	query.Add("redirect_uri", redirectURL)
	return query
}

func authCodeTokenExchange(tf *test.Fixture, redirectURL, clientID, clientSecret, code, codeVerifier string) (*oidc.UCTokenClaims, *oidc.TokenResponse, error) {
	query := buildAuthCodeTokenExchangeQuery(redirectURL, code, codeVerifier)
	query.Add("client_id", clientID)
	query.Add("client_secret", clientSecret)
	tokenResponse, err := tokenExchange(tf, query, "", "")
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	tokenClaims, err := ucjwt.ParseUCClaimsVerified(tokenResponse.IDToken, tf.PublicKey)
	return tokenClaims, tokenResponse, ucerr.Wrap(err)
}

func authCodeTokenExchangeWithBasicAuth(tf *test.Fixture, redirectURL, clientID, clientSecret, code, codeVerifier string) (*oidc.UCTokenClaims, *oidc.TokenResponse, error) {
	query := buildAuthCodeTokenExchangeQuery(redirectURL, code, codeVerifier)
	tokenResponse, err := tokenExchange(tf, query, clientID, clientSecret)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	tokenClaims, err := ucjwt.ParseUCClaimsVerified(tokenResponse.IDToken, tf.PublicKey)
	return tokenClaims, tokenResponse, ucerr.Wrap(err)
}

func clientCredentialsTokenExchange(tf *test.Fixture, clientID, clientSecret string, subjectJWT string, extraAudiences ...string) (*oidc.UCTokenClaims, *oidc.TokenResponse, error) {
	query := url.Values{}
	query.Add("grant_type", "client_credentials")
	query.Add("client_id", clientID)
	query.Add("client_secret", clientSecret)
	for _, v := range extraAudiences {
		query.Add("audience", v)
	}
	if subjectJWT != "" {
		query.Add("subject_jwt", subjectJWT)
	}
	tokenResponse, err := tokenExchange(tf, query, "", "")
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	tokenClaims, err := ucjwt.ParseUCClaimsVerified(tokenResponse.AccessToken, tf.PublicKey)
	return tokenClaims, tokenResponse, ucerr.Wrap(err)
}

func clientCredentialsTokenExchangeWithBasicAuth(tf *test.Fixture, clientID, clientSecret string, subjectJWT string, extraAudiences ...string) (*oidc.UCTokenClaims, *oidc.TokenResponse, error) {
	query := url.Values{}
	query.Add("grant_type", "client_credentials")
	for _, v := range extraAudiences {
		query.Add("audience", v)
	}
	if subjectJWT != "" {
		query.Add("subject_jwt", subjectJWT)
	}
	tokenResponse, err := tokenExchange(tf, query, clientID, clientSecret)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	tokenClaims, err := ucjwt.ParseUCClaimsVerified(tokenResponse.AccessToken, tf.PublicKey)
	return tokenClaims, tokenResponse, ucerr.Wrap(err)
}

func refreshTokenTokenExchange(tf *test.Fixture, clientID, clientSecret, refreshToken string) (*oidc.UCTokenClaims, *oidc.TokenResponse, error) {
	query := url.Values{}
	query.Add("grant_type", "refresh_token")
	query.Add("client_id", clientID)
	query.Add("client_secret", clientSecret)
	query.Add("refresh_token", refreshToken)
	tokenResponse, err := tokenExchange(tf, query, "", "")
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	tokenClaims, err := ucjwt.ParseUCClaimsVerified(tokenResponse.AccessToken, tf.PublicKey)
	return tokenClaims, tokenResponse, ucerr.Wrap(err)
}

func validateTokenResponse(t *testing.T, tokenResponse *oidc.TokenResponse) {
	t.Helper()
	assert.Equal(t, tokenResponse.TokenType, "Bearer")
	assert.NotEqual(t, len(tokenResponse.AccessToken), 0)
	assert.NotEqual(t, len(tokenResponse.IDToken), 0)
}

func genRedirectURL() string {
	return fmt.Sprintf("http://contoso_%s.com/callback", crypto.MustRandomHex(4))
}

func doLogin(t *testing.T, rf *test.RequestFactory, plexHandler http.Handler, lr plex.LoginRequest) (redirectURL *url.URL, code string, state string, oauthe ucerr.OAuthError, err error) {
	t.Helper()
	req := rf.NewRequest(http.MethodPost, "/login", uctest.IOReaderFromJSONStruct(t, lr))
	w := httptest.NewRecorder()
	plexHandler.ServeHTTP(w, req)

	resp := w.Result()
	oauthe.Code = resp.StatusCode

	if resp.StatusCode != http.StatusOK {
		err = json.NewDecoder(resp.Body).Decode(&oauthe)
		return
	}

	redirectURL = parseLoginResponse(t, resp)
	code = redirectURL.Query().Get("code")
	state = redirectURL.Query().Get("state")
	return
}

func TestLoginHandlerInvalidUsername(t *testing.T) {
	t.Parallel()

	redirectURL := genRedirectURL()
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	tcb.SwitchToApp(0).AddAllowedRedirectURI(redirectURL)
	tf := test.NewFixture(t, tcb.Build())

	lr := createLoginRequest(t, tf.Storage, redirectURL, clientID, "invaliduser", "invalidpass")
	_, _, _, oauthe, err := doLogin(t, tf.RequestFactory, tf.Handler, lr)
	assert.NoErr(t, err)
	assert.Equal(t, oauthe.Code, http.StatusBadRequest)
	assert.Equal(t, oauthe.ErrorType, ucerr.ErrIncorrectUsernamePassword.ErrorType)
	assert.Equal(t, oauthe.ErrorDesc, ucerr.ErrIncorrectUsernamePassword.ErrorDesc)
}

func TestLoginHandlerInvalidPassword(t *testing.T) {
	t.Parallel()

	redirectURL := genRedirectURL()
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	tcb.SwitchToApp(0).AddAllowedRedirectURI(redirectURL)
	tf := test.NewFixture(t, tcb.Build())

	_, err := tf.ActiveIDP.CreateUserWithPassword(context.Background(), "validuser", "validpass", iface.UserProfile{UserBaseProfile: idp.UserBaseProfile{Email: testEmail}})
	assert.NoErr(t, err)
	lr := createLoginRequest(t, tf.Storage, redirectURL, clientID, "validuser", "invalidpass")
	_, _, _, oauthe, err := doLogin(t, tf.RequestFactory, tf.Handler, lr)
	assert.NoErr(t, err)
	assert.Equal(t, oauthe.Code, http.StatusBadRequest)
	assert.Equal(t, oauthe.ErrorType, ucerr.ErrIncorrectUsernamePassword.ErrorType)
	assert.Equal(t, oauthe.ErrorDesc, ucerr.ErrIncorrectUsernamePassword.ErrorDesc)
}

func TestLoginHandlerValid(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	redirectURL := genRedirectURL()
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	cs := tcb.SwitchToApp(0).AddAllowedRedirectURI(redirectURL).ClientSecret()
	clientSecret, err := cs.Resolve(ctx)
	assert.NoErr(t, err)
	tf := test.NewFixture(t, tcb.Build())

	_, err = tf.ActiveIDP.CreateUserWithPassword(context.Background(), "validuser", "validpass", iface.UserProfile{UserBaseProfile: idp.UserBaseProfile{Email: testEmail}})
	assert.NoErr(t, err)
	lr := createLoginRequest(t, tf.Storage, redirectURL, clientID, "validuser", "validpass")
	loginResponseRedirectURL, code, state, oauthe, err := doLogin(t, tf.RequestFactory, tf.Handler, lr)
	assert.NoErr(t, err)
	assert.Equal(t, oauthe.Code, http.StatusOK, assert.Must())
	assert.Contains(t, loginResponseRedirectURL.String(), redirectURL)
	assert.NotEqual(t, len(code), 0)
	assert.Equal(t, state, testState)

	// Try a bad client secret
	_, _, err = authCodeTokenExchange(tf, redirectURL, clientID, "badsecret", code, "")
	assert.NotNil(t, err, assert.Must())
	assert.True(t, errors.As(err, &oauthe), assert.Must())
	assert.Equal(t, oauthe.Code, http.StatusUnauthorized)
	assert.Equal(t, oauthe.ErrorType, ucerr.ErrInvalidClientSecret.ErrorType)
	assert.Equal(t, oauthe.ErrorDesc, ucerr.ErrInvalidClientSecret.ErrorDesc)

	// Try a bad redirect URI
	_, _, err = authCodeTokenExchange(tf, "http://contoso_invalid.com/callback", clientID, clientSecret, code, "")
	assert.NotNil(t, err, assert.Must())
	assert.True(t, errors.As(err, &oauthe), assert.Must())
	assert.Equal(t, oauthe.Code, http.StatusBadRequest)
	assert.Equal(t, oauthe.ErrorType, "invalid_request")
	// Not ideal to check for literals in user-facing errors but better than not testing this I think?
	assert.Contains(t, oauthe.ErrorDesc, "redirect URI")

	// Try a bad code
	_, _, err = authCodeTokenExchange(tf, redirectURL, clientID, clientSecret, code+"foo", "")
	assert.NotNil(t, err, assert.Must())
	assert.True(t, errors.As(err, &oauthe), assert.Must())
	assert.Equal(t, oauthe.Code, http.StatusBadRequest)
	assert.Equal(t, oauthe.ErrorType, ucerr.ErrInvalidAuthorizationCode.ErrorType)
	assert.Equal(t, oauthe.ErrorDesc, ucerr.ErrInvalidAuthorizationCode.ErrorDesc)

	// Try the right code
	_, tokenResponse, err := authCodeTokenExchange(tf, redirectURL, clientID, clientSecret, code, "")
	assert.NoErr(t, err)
	validateTokenResponse(t, tokenResponse)
}

func TestLoginHandlerValidWithBasicAuth(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	redirectURL := genRedirectURL()
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	cs := tcb.SwitchToApp(0).AddAllowedRedirectURI(redirectURL).ClientSecret()
	clientSecret, err := cs.Resolve(ctx)
	assert.NoErr(t, err)
	tf := test.NewFixture(t, tcb.Build())

	_, err = tf.ActiveIDP.CreateUserWithPassword(context.Background(), "validuser", "validpass", iface.UserProfile{UserBaseProfile: idp.UserBaseProfile{Email: testEmail}})
	assert.NoErr(t, err)
	lr := createLoginRequest(t, tf.Storage, redirectURL, clientID, "validuser", "validpass")
	loginResponseRedirectURL, code, state, oauthe, err := doLogin(t, tf.RequestFactory, tf.Handler, lr)
	assert.NoErr(t, err)
	assert.Equal(t, oauthe.Code, http.StatusOK)
	assert.Contains(t, loginResponseRedirectURL.String(), redirectURL)
	assert.NotEqual(t, len(code), 0)
	assert.Equal(t, state, testState)

	// Try a bad client secret
	_, _, err = authCodeTokenExchangeWithBasicAuth(tf, redirectURL, clientID, "badsecret", code, "")
	assert.NotNil(t, err, assert.Must())
	assert.True(t, errors.As(err, &oauthe), assert.Must())
	assert.Equal(t, oauthe.Code, http.StatusUnauthorized)
	assert.Equal(t, oauthe.ErrorType, ucerr.ErrInvalidClientSecret.ErrorType)
	assert.Equal(t, oauthe.ErrorDesc, ucerr.ErrInvalidClientSecret.ErrorDesc)

	// Try the right code
	_, tokenResponse, err := authCodeTokenExchangeWithBasicAuth(tf, redirectURL, clientID, clientSecret, code, "")
	assert.NoErr(t, err)
	validateTokenResponse(t, tokenResponse)
}

func TestLogout(t *testing.T) {
	t.Parallel()

	logoutURI := "http://contoso.com/logout"
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	tcb.SwitchToApp(0).AddAllowedLogoutURI(logoutURI)
	tf := test.NewFixture(t, tcb.Build())

	logoutCount := tf.ActiveIDP.LogoutCount
	query := url.Values{}
	query.Add("client_id", clientID)
	query.Add("redirect_url", logoutURI)
	req := tf.RequestFactory.NewRequest(http.MethodGet, fmt.Sprintf("/logout?%s", query.Encode()), nil)
	w := httptest.NewRecorder()
	tf.Handler.ServeHTTP(w, req)

	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusFound)
	assert.Equal(t, logoutCount+1, tf.ActiveIDP.LogoutCount)

	// Test invalid logout URI
	badQuery := url.Values{}
	badQuery.Add("client_id", clientID)
	badQuery.Add("redirect_url", "http://evilsite.com/logout")
	req = tf.RequestFactory.NewRequest(http.MethodGet, fmt.Sprintf("/logout?%s", badQuery.Encode()), nil)
	w = httptest.NewRecorder()
	tf.Handler.ServeHTTP(w, req)

	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
	// No increase to logout count
	assert.Equal(t, logoutCount+1, tf.ActiveIDP.LogoutCount)
}

func TestMFA(t *testing.T) {
	t.Parallel()

	redirectURL := genRedirectURL()
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	tcb.SwitchToApp(0).AddAllowedRedirectURI(redirectURL).SetTenantPageParameter(pagetype.EveryPage, parameter.MFAMethods, "email")
	tf := test.NewFixture(t, tcb.Build())

	userID, err := tf.ActiveIDP.CreateUserWithPassword(context.Background(), "validuser", "validpass", iface.UserProfile{UserBaseProfile: idp.UserBaseProfile{Email: testEmail}})
	assert.NoErr(t, err)
	// TODO: expose a way to change MFA settings on provider interface
	user := tf.ActiveIDP.Users[userID]
	user.EnableMFA()
	tf.ActiveIDP.Users[userID] = user

	lreq := createLoginRequest(t, tf.Storage, redirectURL, clientID, "validuser", "validpass")
	req := tf.RequestFactory.NewRequest(http.MethodPost, "/login", uctest.IOReaderFromJSONStruct(t, lreq))
	w := httptest.NewRecorder()
	tf.Handler.ServeHTTP(w, req)

	// Ensure we got a successful response that would redirect the user agent
	// from the Plex Login UI to the MFA prompt UI
	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	loginResponseRedirectURL := parseLoginResponse(t, resp)
	assert.Contains(t, loginResponseRedirectURL.Path, paths.PlexUIRoot+paths.MFACodeUISubPath)
	sessionID, err := uuid.FromString(loginResponseRedirectURL.Query().Get("session_id"))
	assert.NoErr(t, err)
	assert.Equal(t, lreq.SessionID, sessionID)

	// Respond with the wrong code
	mfaReq := internal.MFASubmitRequest{
		MFACode:   "bad_mfa_code",
		SessionID: sessionID,
	}
	req = tf.RequestFactory.NewRequest(http.MethodPost, "/mfa/submit", uctest.IOReaderFromJSONStruct(t, mfaReq))
	w = httptest.NewRecorder()
	tf.Handler.ServeHTTP(w, req)
	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	loginResponseRedirectURL = parseLoginResponse(t, resp)
	assert.Contains(t, loginResponseRedirectURL.Path, paths.PlexUIRoot+paths.MFACodeUISubPath)
	sessionID, err = uuid.FromString(loginResponseRedirectURL.Query().Get("session_id"))
	assert.NoErr(t, err)
	assert.Equal(t, lreq.SessionID, sessionID)

	// Respond with the right code
	mfaReq = internal.MFASubmitRequest{
		MFACode:   tf.ActiveIDP.Users[userID].MFACode,
		SessionID: sessionID,
	}
	req = tf.RequestFactory.NewRequest(http.MethodPost, "/mfa/submit", uctest.IOReaderFromJSONStruct(t, mfaReq))
	w = httptest.NewRecorder()
	tf.Handler.ServeHTTP(w, req)
	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func makeCodeAuthorizeRequest(rf *test.RequestFactory, redirectURL, clientID string, extraValues url.Values) *http.Request {
	query := extraValues
	query.Add("scope", testScope)
	query.Add("response_type", string(storage.AuthorizationCodeResponseType))
	query.Add("client_id", clientID)
	query.Add("state", testState)
	query.Add("redirect_uri", redirectURL)
	return rf.NewRequest(http.MethodGet, fmt.Sprintf("/oidc/authorize?%s", query.Encode()), nil)
}

func getSessionIDFromRedirect(resp *http.Response) (uuid.UUID, error) {
	redirectURL, locErr := resp.Location()
	if resp.StatusCode != http.StatusFound {
		location := "[none]"
		if locErr == nil {
			location = redirectURL.String()
		}
		return uuid.Nil, ucerr.Errorf("unexpected status code %d; expected %d. Location header: %s", resp.StatusCode, http.StatusFound, location)
	}
	if locErr != nil {
		return uuid.Nil, ucerr.Wrap(locErr)
	}
	sessionID, err := uuid.FromString(redirectURL.Query().Get("session_id"))
	return sessionID, ucerr.Wrap(err)
}

func TestAuthorize(t *testing.T) {
	t.Parallel()

	redirectURL := genRedirectURL()
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	tcb.SwitchToApp(0).AddAllowedRedirectURI(redirectURL)
	tf := test.NewFixture(t, tcb.Build())

	// Ensure that the /authorize endpoint creates an OIDC request state object and
	// redirects us somewhere sane.
	req := makeCodeAuthorizeRequest(tf.RequestFactory, redirectURL, clientID, url.Values{})
	w := httptest.NewRecorder()
	tf.Handler.ServeHTTP(w, req)

	// White box test the result to ensure the OIDC login request is created properly.
	sessionID, err := getSessionIDFromRedirect(w.Result())
	assert.NoErr(t, err)
	session, err := tf.Storage.GetOIDCLoginSession(context.Background(), sessionID)
	assert.NoErr(t, err)
	assert.Equal(t, session.Scopes, testScope)
	rts, err := storage.NewResponseTypes(session.ResponseTypes)
	assert.NoErr(t, err)
	assert.Equal(t, len(rts), 1, assert.Must())
	assert.Equal(t, rts[0], storage.AuthorizationCodeResponseType)
	assert.Equal(t, session.ClientID, clientID)
	assert.Equal(t, session.State, testState)
	assert.Equal(t, session.RedirectURI, redirectURL)

	// test client ID validation
	req = makeCodeAuthorizeRequest(tf.RequestFactory, redirectURL, "bad_client_id", url.Values{})
	w = httptest.NewRecorder()
	tf.Handler.ServeHTTP(w, req)

	assert.Equal(t, w.Code, http.StatusBadRequest)
	_, err = getSessionIDFromRedirect(w.Result())
	assert.NotNil(t, err)

	// test redirect URL validation
	req = makeCodeAuthorizeRequest(tf.RequestFactory, "http://contoso_invalid.com/callback", clientID, url.Values{})
	w = httptest.NewRecorder()
	tf.Handler.ServeHTTP(w, req)

	assert.Equal(t, w.Code, http.StatusBadRequest)
	_, err = getSessionIDFromRedirect(w.Result())
	assert.NotNil(t, err)
}

func TestPasswordless(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	redirectURL := genRedirectURL()
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	cs := tcb.SwitchToApp(0).AddAllowedRedirectURI(redirectURL).ClientSecret()
	clientSecret, err := cs.Resolve(ctx)
	assert.NoErr(t, err)

	tc := tcb.Build()
	tf := test.NewFixture(t, tc)

	userID, err := tf.ActiveIDP.CreateUserWithPassword(context.Background(), "validuser", "validpass", iface.UserProfile{UserBaseProfile: idp.UserBaseProfile{
		Email:         testEmail,
		EmailVerified: false,
		Name:          testName,
		Nickname:      testNickname,
		Picture:       testPicture,
	}})
	assert.NoErr(t, err)

	sessionID, err := storage.CreateOIDCLoginSession(
		ctx, tf.Storage, clientID, []storage.ResponseType{storage.AuthorizationCodeResponseType},
		uctest.MustParseURL(redirectURL), testState, "unusedscope")
	assert.NoErr(t, err)

	startRequest := internal.PasswordlessLoginRequest{
		SessionID: sessionID,
		Email:     testEmail,
	}

	// It's POST only but at least ensure GET doens't work
	t.Run("GetNotAllowed", func(t *testing.T) {
		t.Parallel()

		req := tf.RequestFactory.NewRequest(http.MethodGet, "/passwordless/start", uctest.IOReaderFromJSONStruct(t, startRequest))
		w := httptest.NewRecorder()
		tf.Handler.ServeHTTP(w, req)
		assert.Equal(t, w.Result().StatusCode, http.StatusMethodNotAllowed)
	})

	t.Run("OTPCodeLogin", func(t *testing.T) {
		t.Parallel()

		bodyCount := len(tf.Email.Bodies)
		htmlBodyCount := len(tf.Email.HTMLBodies)
		req := tf.RequestFactory.NewRequest(http.MethodPost, "/passwordless/start", uctest.IOReaderFromJSONStruct(t, startRequest))
		w := httptest.NewRecorder()
		tf.Handler.ServeHTTP(w, req)
		assert.Equal(t, w.Result().StatusCode, http.StatusNoContent)
		assert.Equal(t, len(tf.Email.Bodies), bodyCount+1, assert.Must())
		assert.Equal(t, len(tf.Email.HTMLBodies), htmlBodyCount+1, assert.Must())

		emailBody := tf.Email.Bodies[len(tf.Email.Bodies)-1]
		// Expect whitespace around the code - maybe not a fair long term assumption but it works for the current template
		codeRegexp := regexp.MustCompile(`\s[0-9][0-9][0-9][0-9][0-9][0-9]\s`)
		otpCode := strings.TrimSpace(codeRegexp.FindString(emailBody))
		assert.Equal(t, len(otpCode), 6, assert.Must())

		t.Run("WrongCode", func(t *testing.T) {
			t.Parallel()

			badCode := ""
			// Change one digit
			if otpCode[0] == '0' {
				badCode = "1" + otpCode[1:]
			} else {
				badCode = "0" + otpCode[1:]
			}
			submitRequest := otp.SubmitRequest{
				SessionID: sessionID,
				Email:     testEmail,
				OTPCode:   badCode,
			}
			req := tf.RequestFactory.NewRequest(http.MethodPost, otpSubmitPath, uctest.IOReaderFromJSONStruct(t, submitRequest))
			w := httptest.NewRecorder()
			tf.Handler.ServeHTTP(w, req)
			assert.Equal(t, w.Result().StatusCode, http.StatusBadRequest)

			// TODO: can we check anything else (like body) to make sure we hit the right codepath?
			// we were failing on another 400 before. I didn't add a friendly error yet in case it's a security leak
		})

		t.Run("WrongEmail", func(t *testing.T) {
			t.Parallel()

			submitRequest := otp.SubmitRequest{
				SessionID: sessionID,
				Email:     "bademail@contoso.com",
				OTPCode:   otpCode,
			}
			req := tf.RequestFactory.NewRequest(http.MethodPost, otpSubmitPath, uctest.IOReaderFromJSONStruct(t, submitRequest))
			w := httptest.NewRecorder()
			tf.Handler.ServeHTTP(w, req)
			assert.Equal(t, w.Result().StatusCode, http.StatusBadRequest)

			// TODO: can we check anything else (like body) to make sure we hit the right codepath?
			// we were failing on another 400 before. I didn't add a friendly error yet in case it's a security leak
		})

		t.Run("SuccessfulLogin", func(t *testing.T) {
			t.Parallel()

			submitRequest := otp.SubmitRequest{
				SessionID: sessionID,
				Email:     testEmail,
				OTPCode:   otpCode,
			}
			// POST simulates UI form submit with code
			req := tf.RequestFactory.NewRequest(http.MethodPost, otpSubmitPath, uctest.IOReaderFromJSONStruct(t, submitRequest))
			w := httptest.NewRecorder()
			tf.Handler.ServeHTTP(w, req)
			resp := w.Result()
			assert.Equal(t, resp.StatusCode, http.StatusOK)
			loginResponseRedirectURL := parseLoginResponse(t, resp)
			assert.Contains(t, loginResponseRedirectURL.String(), redirectURL)
			code := loginResponseRedirectURL.Query().Get("code")
			assert.NotEqual(t, len(code), 0)
			state := loginResponseRedirectURL.Query().Get("state")
			assert.Equal(t, state, testState)

			tokenClaims, tokenResponse, err := authCodeTokenExchange(tf, redirectURL, clientID, clientSecret, code, "")
			assert.NoErr(t, err)
			validateTokenResponse(t, tokenResponse)
			assert.Equal(t, tokenClaims.Issuer, tf.Tenant.TenantURL)
			assert.Equal(t, tokenClaims.Audience, []string{clientID, tf.Tenant.TenantURL})
			assert.Equal(t, tokenClaims.Subject, userID)
			assert.Equal(t, tokenClaims.Name, testName)
			assert.Equal(t, tokenClaims.Nickname, testNickname)
			assert.Equal(t, tokenClaims.Picture, testPicture)
			assert.Equal(t, tokenClaims.Email, testEmail)

			// Make sure token ID matches storage
			plexToken, err := tf.Storage.GetPlexToken(ctx, uuid.FromStringOrNil(tokenClaims.ID))
			assert.NoErr(t, err)
			assert.Equal(t, plexToken.AccessToken, tokenResponse.AccessToken)
			assert.Equal(t, plexToken.IDToken, tokenResponse.IDToken)

			// Can't re-use token
			req = tf.RequestFactory.NewRequest(http.MethodPost, otpSubmitPath, uctest.IOReaderFromJSONStruct(t, submitRequest))
			w = httptest.NewRecorder()
			tf.Handler.ServeHTTP(w, req)
			assert.Equal(t, w.Result().StatusCode, http.StatusBadRequest)
		})
	})
}

// TODO: Test creation of account through magic link login, not just login to existing account.
func TestMagicLinkLogin(t *testing.T) {
	t.Parallel()

	redirectURL := genRedirectURL()
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	tcb.SwitchToApp(0).AddAllowedRedirectURI(redirectURL)
	tf := test.NewFixture(t, tcb.Build())

	_, err := tf.ActiveIDP.CreateUserWithPassword(context.Background(), "validuser", "validpass", iface.UserProfile{
		UserBaseProfile: idp.UserBaseProfile{Email: testEmail},
	})
	assert.NoErr(t, err)

	sessionID, err := storage.CreateOIDCLoginSession(
		context.Background(), tf.Storage, clientID, []storage.ResponseType{storage.AuthorizationCodeResponseType},
		uctest.MustParseURL(redirectURL), testState, "unusedscope")
	assert.NoErr(t, err)

	startRequest := internal.PasswordlessLoginRequest{
		SessionID: sessionID,
		Email:     testEmail,
	}

	bodyCount := len(tf.Email.Bodies)
	htmlBodyCount := len(tf.Email.HTMLBodies)
	req := tf.RequestFactory.NewRequest(http.MethodPost, "/passwordless/start", uctest.IOReaderFromJSONStruct(t, startRequest))
	w := httptest.NewRecorder()
	tf.Handler.ServeHTTP(w, req)
	assert.Equal(t, w.Result().StatusCode, http.StatusNoContent)
	assert.Equal(t, len(tf.Email.Bodies), bodyCount+1, assert.Must())
	assert.Equal(t, len(tf.Email.HTMLBodies), htmlBodyCount+1, assert.Must())

	magicLinkURL, err := uctest.ExtractURL(tf.Email.HTMLBodies[htmlBodyCount])
	assert.NoErr(t, err)

	req = tf.RequestFactory.NewRequest(http.MethodGet, magicLinkURL.String(), nil)
	w = httptest.NewRecorder()
	tf.Handler.ServeHTTP(w, req)
	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusSeeOther)
	loginResponseRedirectURL, err := resp.Location()
	assert.NoErr(t, err)
	assert.Contains(t, loginResponseRedirectURL.String(), redirectURL)
	assert.Equal(t, loginResponseRedirectURL.Query().Get("state"), testState)
}

func TestRefreshToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	cs := tcb.SwitchToApp(0).ClientSecret()
	clientSecret, err := cs.Resolve(ctx)
	assert.NoErr(t, err)

	tc := tcb.Build()
	tf := test.NewFixture(t, tc)

	t.Run("BadClientID", func(t *testing.T) {
		t.Parallel()

		_, _, err := refreshTokenTokenExchange(tf, "bad_client_id", clientSecret, "")
		assert.NotNil(t, err, assert.Must())
		var oauthe ucerr.OAuthError
		assert.True(t, errors.As(err, &oauthe), assert.Must())
		assert.Equal(t, oauthe.Code, http.StatusUnauthorized)
		assert.Equal(t, oauthe.ErrorType, "invalid_client")
	})

	t.Run("BadSecret", func(t *testing.T) {
		t.Parallel()

		_, _, err := refreshTokenTokenExchange(tf, clientID, "bad_secret", "")
		assert.NotNil(t, err, assert.Must())
		var oauthe ucerr.OAuthError
		assert.True(t, errors.As(err, &oauthe), assert.Must())
		assert.Equal(t, oauthe.Code, http.StatusUnauthorized)
		assert.Equal(t, oauthe.ErrorType, "invalid_grant")
	})

	t.Run("NoRefreshToken", func(t *testing.T) {
		t.Parallel()

		_, _, err := refreshTokenTokenExchange(tf, clientID, clientSecret, "")
		assert.NotNil(t, err, assert.Must())
	})

	t.Run("InvalidRefreshToken", func(t *testing.T) {
		t.Parallel()

		_, _, err := refreshTokenTokenExchange(tf, clientID, clientSecret, "badtoken")
		assert.NotNil(t, err, assert.Must())
	})

	t.Run("ValidRefreshToken", func(t *testing.T) {
		t.Parallel()

		subject := "validuser"
		audience := []string{tf.Tenant.TenantURL}
		tokenID, err := uuid.NewV4()
		assert.NoErr(t, err)

		ctx := multitenant.SetTenantState(ctx, tenantmap.NewTenantState(tf.Tenant, tf.Company, uctest.MustParseURL(tf.Tenant.TenantURL), nil, nil, nil, "", nil, false, nil, nil))
		refreshToken, err := token.CreateRefreshTokenJWT(ctx, &tc, tokenID, subject, "", "", audience, ucjwt.DefaultValidityRefresh)
		assert.NoErr(t, err)

		refreshTokenClaims, err := ucjwt.ParseUCClaimsVerified(refreshToken, tf.PublicKey)
		assert.NoErr(t, err)
		// Refresh tokens should not have an audience.
		assert.True(t, len(refreshTokenClaims.Audience) == 0, assert.Must())

		claims, _, err := refreshTokenTokenExchange(tf, clientID, clientSecret, refreshToken)
		assert.NoErr(t, err)
		assert.Equal(t, claims.Subject, subject)
		assert.Equal(t, claims.Audience, audience)
	})
}

func TestClientCredentials(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	cs := tcb.SwitchToApp(0).ClientSecret()
	clientSecret, err := cs.Resolve(ctx)
	assert.NoErr(t, err)

	appID := tcb.SwitchToApp(0).ID()
	tc := tcb.Build()
	tf := test.NewFixture(t, tc)

	t.Run("BadClientID", func(t *testing.T) {
		t.Parallel()

		_, _, err := clientCredentialsTokenExchange(tf, "bad_client_id", clientSecret, "")
		assert.NotNil(t, err, assert.Must())
		var oauthe ucerr.OAuthError
		assert.True(t, errors.As(err, &oauthe), assert.Must())
		assert.Equal(t, oauthe.Code, http.StatusUnauthorized)
		assert.Equal(t, oauthe.ErrorType, "invalid_client")
	})

	t.Run("BadSecret", func(t *testing.T) {
		t.Parallel()

		_, _, err := clientCredentialsTokenExchange(tf, clientID, "bad_secret", "")
		assert.NotNil(t, err, assert.Must())
		var oauthe ucerr.OAuthError
		assert.True(t, errors.As(err, &oauthe), assert.Must())
		assert.Equal(t, oauthe.Code, http.StatusUnauthorized)
		assert.Equal(t, oauthe.ErrorType, "invalid_grant")
	})

	t.Run("BadSecretBasicAuth", func(t *testing.T) {
		t.Parallel()

		_, _, err := clientCredentialsTokenExchangeWithBasicAuth(tf, clientID, "bad_secret", "")
		assert.NotNil(t, err, assert.Must())
		var oauthe ucerr.OAuthError
		assert.True(t, errors.As(err, &oauthe), assert.Must())
		assert.Equal(t, oauthe.Code, http.StatusUnauthorized)
		assert.Equal(t, oauthe.ErrorType, "invalid_grant")
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		tokenClaims, tokenResponse, err := clientCredentialsTokenExchange(tf, clientID, clientSecret, "")
		assert.NoErr(t, err)
		assert.Equal(t, tokenResponse.TokenType, "Bearer")
		assert.NotEqual(t, len(tokenResponse.AccessToken), 0)
		// ID token doesn't make sense for CCF
		assert.Equal(t, len(tokenResponse.IDToken), 0)
		// Audience is the tenant URL with no extra audience specified
		assert.Equal(t, tokenClaims.Audience, []string{tf.Tenant.TenantURL})
		assert.Equal(t, tokenClaims.Subject, appID.String())

		// Make sure token ID matches storage
		plexToken, err := tf.Storage.GetPlexToken(ctx, uuid.FromStringOrNil(tokenClaims.ID))
		assert.NoErr(t, err)
		assert.Equal(t, plexToken.AccessToken, tokenResponse.AccessToken)
		assert.Equal(t, plexToken.IDToken, tokenResponse.IDToken)
	})

	t.Run("SuccessBasicAuth", func(t *testing.T) {
		t.Parallel()

		tokenClaims, tokenResponse, err := clientCredentialsTokenExchangeWithBasicAuth(tf, clientID, clientSecret, "")
		assert.NoErr(t, err)
		assert.Equal(t, tokenResponse.TokenType, "Bearer")
		assert.NotEqual(t, len(tokenResponse.AccessToken), 0)
		// ID token doesn't make sense for CCF
		assert.Equal(t, len(tokenResponse.IDToken), 0)
		// Audience is the tenant URL with no extra audience specified
		assert.Equal(t, tokenClaims.Audience, []string{tf.Tenant.TenantURL})
		assert.Equal(t, tokenClaims.Subject, appID.String())

		// Make sure token ID matches storage
		plexToken, err := tf.Storage.GetPlexToken(ctx, uuid.FromStringOrNil(tokenClaims.ID))
		assert.NoErr(t, err)
		assert.Equal(t, plexToken.AccessToken, tokenResponse.AccessToken)
		assert.Equal(t, plexToken.IDToken, tokenResponse.IDToken)
	})
	t.Run("SuccessExtraAudiences", func(t *testing.T) {
		t.Parallel()

		tokenClaims, _, err := clientCredentialsTokenExchange(tf, clientID, clientSecret, "", "http://example.com/custom_audience", "foobar")
		assert.NoErr(t, err)

		// Don't care about order; sort before comparing
		expectedAudiences := []string{tf.Tenant.TenantURL, "http://example.com/custom_audience", "foobar"}
		sort.Strings(expectedAudiences)
		sort.Strings(tokenClaims.Audience)

		assert.Equal(t, tokenClaims.Audience, expectedAudiences)
		assert.Equal(t, tokenClaims.Subject, appID.String())
	})
}

func assertRedirectError(t *testing.T, resp *http.Response, expectedError string) {
	t.Helper()
	assert.Equal(t, resp.StatusCode, http.StatusFound, assert.Must())
	redirectURL, err := resp.Location()
	assert.NoErr(t, err)
	assert.Equal(t, redirectURL.Query().Get("error"), expectedError)
}

func makeCodeChallengeRequest(h http.Handler, rf *test.RequestFactory, redirectURL, clientID, method, challenge string) *http.Response {
	u := url.Values{}
	if len(method) > 0 {
		u.Add("code_challenge_method", method)
	}
	if len(challenge) > 0 {
		u.Add("code_challenge", challenge)
	}
	req := makeCodeAuthorizeRequest(rf, redirectURL, clientID, u)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w.Result()
}

func loginWithPKCE(t *testing.T, rf *test.RequestFactory, h http.Handler, redirectURL, clientID, codeChallenge string) (string, error) {
	t.Helper()
	resp := makeCodeChallengeRequest(h, rf, redirectURL, clientID, "S256", codeChallenge)
	sessionID, err := getSessionIDFromRedirect(resp)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	lr := plex.LoginRequest{
		Username:  "validuser",
		Password:  "validpass",
		SessionID: sessionID,
	}
	_, code, _, oauthe, err := doLogin(t, rf, h, lr)
	if oauthe.Code != http.StatusOK {
		return "", oauthe
	}
	return code, err
}

func assertInvalidCodeVerifier(t *testing.T, err error) {
	t.Helper()
	assert.NotNil(t, err, assert.Must())
	var oauthe ucerr.OAuthError
	assert.True(t, errors.As(err, &oauthe), assert.Must())
	assert.Equal(t, oauthe.Code, http.StatusBadRequest)
	assert.Equal(t, oauthe.ErrorType, ucerr.ErrInvalidCodeVerifier.ErrorType)
	assert.Equal(t, oauthe.ErrorDesc, ucerr.ErrInvalidCodeVerifier.ErrorDesc)
}

func TestPKCE(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	redirectURL := genRedirectURL()
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	cs := tcb.SwitchToApp(0).AddAllowedRedirectURI(redirectURL).ClientSecret()
	clientSecret, err := cs.Resolve(ctx)
	assert.NoErr(t, err)

	tf := test.NewFixture(t, tcb.Build())

	codeVerifier := crypto.NewCodeVerifier()
	validCodeChallenge, err := codeVerifier.GetCodeChallenge(crypto.CodeChallengeMethodS256)
	assert.NoErr(t, err)
	_, err = tf.ActiveIDP.CreateUserWithPassword(context.Background(), "validuser", "validpass", iface.UserProfile{UserBaseProfile: idp.UserBaseProfile{Email: testEmail}})
	assert.NoErr(t, err)

	t.Run("UnsupportedMethod", func(t *testing.T) {
		t.Parallel()

		assertRedirectError(t, makeCodeChallengeRequest(tf.Handler, tf.RequestFactory, redirectURL, clientID, "plain", validCodeChallenge), "invalid_request")
	})
	t.Run("CodeChallengeTooShort", func(t *testing.T) {
		t.Parallel()

		// Trim 1 character off
		assertRedirectError(t, makeCodeChallengeRequest(tf.Handler, tf.RequestFactory, redirectURL, clientID, "S256", validCodeChallenge[:len(validCodeChallenge)-1]), "invalid_request")
	})
	t.Run("CodeChallengeTooLong", func(t *testing.T) {
		t.Parallel()

		// Add a character
		assertRedirectError(t, makeCodeChallengeRequest(tf.Handler, tf.RequestFactory, redirectURL, clientID, "S256", validCodeChallenge+"a"), "invalid_request")
	})
	t.Run("MissingCodeChallengeMethod", func(t *testing.T) {
		t.Parallel()

		assertRedirectError(t, makeCodeChallengeRequest(tf.Handler, tf.RequestFactory, redirectURL, clientID, "", validCodeChallenge+"a"), "invalid_request")
	})
	t.Run("MissingCodeChallenge", func(t *testing.T) {
		t.Parallel()

		assertRedirectError(t, makeCodeChallengeRequest(tf.Handler, tf.RequestFactory, redirectURL, clientID, "S256", ""), "invalid_request")
	})

	t.Run("SuccessfulAuthorize", func(t *testing.T) {
		t.Parallel()

		code, err := loginWithPKCE(t, tf.RequestFactory, tf.Handler, redirectURL, clientID, validCodeChallenge)
		assert.NoErr(t, err)
		assert.NotEqual(t, len(code), 0)

		_, tokenResponse, err := authCodeTokenExchange(tf, redirectURL, clientID, clientSecret, code, string(codeVerifier))
		assert.NoErr(t, err)
		validateTokenResponse(t, tokenResponse)

		// Can't re-use it
		_, _, err = authCodeTokenExchange(tf, redirectURL, clientID, clientSecret, code, string(codeVerifier))
		assertInvalidCodeVerifier(t, err)
	})

	t.Run("MissingClientIDOnExchange", func(t *testing.T) {
		t.Parallel()

		code, err := loginWithPKCE(t, tf.RequestFactory, tf.Handler, redirectURL, clientID, validCodeChallenge)
		assert.NoErr(t, err)
		assert.NotEqual(t, len(code), 0)

		_, _, err = authCodeTokenExchange(tf, redirectURL, "", clientSecret, code, string(codeVerifier))
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "no plex app with Plex client ID")
	})

	t.Run("BadClientSecretOnExchange", func(t *testing.T) {
		t.Parallel()

		code, err := loginWithPKCE(t, tf.RequestFactory, tf.Handler, redirectURL, clientID, validCodeChallenge)
		assert.NoErr(t, err)
		assert.NotEqual(t, len(code), 0)

		_, _, err = authCodeTokenExchange(tf, redirectURL, clientID, "bad_secret", code, string(codeVerifier))
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "invalid client secret")
	})

	t.Run("FailExchangeAfterAuthorize", func(t *testing.T) {
		t.Parallel()

		code, err := loginWithPKCE(t, tf.RequestFactory, tf.Handler, redirectURL, clientID, validCodeChallenge)
		assert.NoErr(t, err)
		assert.NotEqual(t, len(code), 0)

		// Pass wrong verifier in
		_, _, err = authCodeTokenExchange(tf, redirectURL, clientID, clientSecret, code, string(crypto.NewCodeVerifier()))
		assertInvalidCodeVerifier(t, err)

		// Pass right verifier in, but expect an error
		_, _, err = authCodeTokenExchange(tf, redirectURL, clientID, clientSecret, code, string(codeVerifier))
		assertInvalidCodeVerifier(t, err)
	})
}

func TestTenantConfigsUpdate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	redirectURL := genRedirectURL()
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	tcb.SwitchToApp(0).AddAllowedRedirectURI(redirectURL)
	tc := tcb.Build()
	tf := test.NewFixture(t, tc)

	// make sure our config is set up right for the original client ID
	req := makeCodeAuthorizeRequest(tf.RequestFactory, redirectURL, clientID, url.Values{})
	w := httptest.NewRecorder()
	tf.Handler.ServeHTTP(w, req)
	sessionID, err := getSessionIDFromRedirect(w.Result())
	assert.NoErr(t, err)
	session, err := tf.Storage.GetOIDCLoginSession(ctx, sessionID)
	assert.NoErr(t, err)
	assert.Equal(t, session.ClientID, clientID)

	updatedClientID := "foo"
	tcb = plexconfigtest.NewTenantConfigBuilderFromTenantConfig(tc)
	tcb.SwitchToApp(0).SetClientID(updatedClientID)
	tcNew := tcb.Build()
	mgr := manager.NewFromDB(tf.TenantDB, cachetesthelpers.NewCacheConfig())
	tp, err := mgr.GetTenantPlex(ctx, tf.Tenant.ID)
	assert.NoErr(t, err)
	tp.PlexConfig = tcNew
	err = mgr.SaveTenantPlex(ctx, tp)
	assert.NoErr(t, err)

	// Let plex's cache timeout
	time.Sleep(testCacheDuration * 2)

	// try again with the wrong client ID
	req = makeCodeAuthorizeRequest(tf.RequestFactory, redirectURL, clientID, url.Values{})
	w = httptest.NewRecorder()
	tf.Handler.ServeHTTP(w, req)
	_, err = getSessionIDFromRedirect(w.Result())
	assert.NotNil(t, err, assert.Must())
	assert.Equal(t, w.Code, http.StatusBadRequest)

	// and now try the right client ID
	req = makeCodeAuthorizeRequest(tf.RequestFactory, redirectURL, updatedClientID, url.Values{})
	w = httptest.NewRecorder()
	tf.Handler.ServeHTTP(w, req)
	sessionID, err = getSessionIDFromRedirect(w.Result())
	assert.NoErr(t, err)
	session, err = tf.Storage.GetOIDCLoginSession(ctx, sessionID)
	assert.NoErr(t, err)
	assert.Equal(t, session.ClientID, updatedClientID)
}
