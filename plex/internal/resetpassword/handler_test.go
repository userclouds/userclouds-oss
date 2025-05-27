package resetpassword_test

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/crypto"
	"userclouds.com/internal/uctest"
	"userclouds.com/plex"
	"userclouds.com/plex/internal/otp"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/test"
	"userclouds.com/test/testlogtransport"
)

const testState = "teststate1234"

var testRedirectURL = uctest.MustParseURL("http://contoso.com/callback")

func createUserWithPassword(t *testing.T, tf *test.Fixture, password string) (string, string, string) {
	ctx := context.Background()
	t.Helper()
	hash := crypto.GenerateOpaqueAccessToken()
	username := fmt.Sprintf("user_%s", hash)
	email := fmt.Sprintf("testuser_%s@contoso.com", hash)
	userIDActive, err := tf.ActiveIDP.CreateUserWithPassword(ctx, username, password, iface.UserProfile{UserBaseProfile: idp.UserBaseProfile{Email: email}})
	assert.NoErr(t, err)
	var userIDFollower string
	if tf.FollowerIDP != nil {
		userIDFollower, err = tf.FollowerIDP.CreateUserWithPassword(ctx, username, password, iface.UserProfile{UserBaseProfile: idp.UserBaseProfile{Email: email}})
		assert.NoErr(t, err)
	}
	return email, userIDActive, userIDFollower
}

func TestStartScreen(t *testing.T) {
	tcb, testClientID := test.NewFollowerTenantConfigBuilder()
	tf := test.NewFixture(t, tcb.Build())

	t.Run("NoSessionID", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := tf.RequestFactory.NewRequest(http.MethodGet, "/resetpassword/start", nil)
		tf.Handler.ServeHTTP(w, req)
		assert.Equal(t, w.Result().StatusCode, http.StatusBadRequest)
	})

	t.Run("GoodSessionID", func(t *testing.T) {
		sessionID, err := storage.CreateOIDCLoginSession(
			context.Background(), tf.Storage, testClientID, []storage.ResponseType{storage.AuthorizationCodeResponseType},
			testRedirectURL, testState, "unusedscope")
		assert.NoErr(t, err)
		w := httptest.NewRecorder()
		req := tf.RequestFactory.NewRequest(http.MethodGet, fmt.Sprintf("/resetpassword/start?session_id=%s", sessionID), nil)
		tf.Handler.ServeHTTP(w, req)
		resp := w.Result()
		// Expect to be redirected to the actual UI
		assert.Equal(t, resp.StatusCode, http.StatusSeeOther)
		assert.Contains(t, resp.Header.Get("Location"), fmt.Sprintf("%s%s", paths.PlexUIRoot, paths.StartResetPasswordUISubPath))
	})
}

func submitResetRequest(t *testing.T, tf *test.Fixture, clientID, email string, sessionID uuid.UUID) (*url.URL, string) {
	// NOTE: errors in here may be annoying to debug due to 'Helper' but we use this logic
	// in enough places it seems worthwhile to avoid a lot of duplication/boilerplate.
	t.Helper()

	// Capture pre-existing count
	emailCount := len(tf.Email.Bodies)
	htmlBodyCount := len(tf.Email.HTMLBodies)

	// POST request to start the password reset flow.
	startReqBody := plex.PasswordResetStartRequest{
		SessionID: sessionID,
		Email:     email,
	}
	w := httptest.NewRecorder()
	req := tf.RequestFactory.NewRequest(http.MethodPost, "/resetpassword/startsubmit", uctest.IOReaderFromJSONStruct(t, startReqBody))
	tf.Handler.ServeHTTP(w, req)
	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusNoContent)

	// Ensure 1 new email was sent with a valid link (and no errors)
	assert.Equal(t, len(tf.Email.Bodies), emailCount+1, assert.Must())
	assert.Equal(t, len(tf.Email.HTMLBodies), htmlBodyCount+1, assert.Must())
	resetLinkURLFromHTML, err := uctest.ExtractURL(tf.Email.Bodies[emailCount])
	assert.NoErr(t, err)
	resetLinkURL, err := uctest.ExtractURL(html.UnescapeString(tf.Email.HTMLBodies[htmlBodyCount]))
	assert.NoErr(t, err)
	assert.Equal(t, resetLinkURLFromHTML, resetLinkURL)
	otpCode := resetLinkURL.Query().Get("otp_code")

	return resetLinkURL, otpCode
}

// TestReset is an integration test of the whole password reset flow from reset request
// to changed password.
func TestReset(t *testing.T) {
	tcb, testClientID := test.NewFollowerTenantConfigBuilder()
	tc := tcb.Build()
	tf := test.NewFixture(t, tc)

	sessionID, err := storage.CreateOIDCLoginSession(
		context.Background(), tf.Storage, testClientID, []storage.ResponseType{storage.AuthorizationCodeResponseType},
		testRedirectURL, testState, "unusedscope")
	assert.NoErr(t, err)

	email, userIDActive, userIDFollower := createUserWithPassword(t, tf, "oldpassword")

	resetURL, otpCode := submitResetRequest(t, tf, testClientID, email, sessionID)
	expectedURL := fmt.Sprintf("%s%s%s", tf.Tenant.TenantURL, otp.RootPath, otp.SubmitSubPath)
	assert.Contains(t, resetURL.String(), expectedURL)
	// Ensure there is a code of some reasonable length
	assert.True(t, len(otpCode) >= 32)

	// Invoke the reset handler via a GET to the reset link we got in email.
	w := httptest.NewRecorder()
	req := tf.RequestFactory.NewRequest(http.MethodGet, resetURL.String(), nil)
	tf.Handler.ServeHTTP(w, req)
	resp := w.Result()
	// Expect to be redirected to the actual UI
	assert.Equal(t, resp.StatusCode, http.StatusSeeOther, assert.Must())
	assert.Contains(t, resp.Header.Get("Location"), fmt.Sprintf("%s%s", paths.PlexUIRoot, paths.FinishResetPasswordUISubPath))

	// Ensure no password changes have occurred (client(s) have been created as a result of
	// invoking the reset handlers)
	userActive, ok := tf.ActiveIDP.Users[userIDActive]
	assert.True(t, ok, assert.Must())
	assert.Equal(t, userActive.Authn.Password, "oldpassword")
	userFollower, ok := tf.FollowerIDP.Users[userIDFollower]
	assert.True(t, ok, assert.Must())
	assert.Equal(t, userFollower.Authn.Password, "oldpassword")

	newPassword := fmt.Sprintf("newpassword_%s", crypto.GenerateOpaqueAccessToken())
	// POST to finish the password change operation.
	submitReqBody := plex.PasswordResetSubmitRequest{
		SessionID: sessionID,
		OTPCode:   otpCode,
		Password:  newPassword,
	}
	w = httptest.NewRecorder()
	req = tf.RequestFactory.NewRequest(http.MethodPost, "/resetpassword/resetsubmit", uctest.IOReaderFromJSONStruct(t, submitReqBody))
	tf.Handler.ServeHTTP(w, req)
	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusNoContent, assert.Must())

	userActive, ok = tf.ActiveIDP.Users[userIDActive]
	assert.True(t, ok, assert.Must())
	assert.Equal(t, userActive.Authn.Password, newPassword)
	userFollower, ok = tf.FollowerIDP.Users[userIDFollower]
	assert.True(t, ok, assert.Must())
	assert.Equal(t, userFollower.Authn.Password, newPassword)
}

func TestInvalidResetLink(t *testing.T) {
	tcb, testClientID := test.NewFollowerTenantConfigBuilder()
	tf := test.NewFixture(t, tcb.Build())

	email, _, _ := createUserWithPassword(t, tf, "oldpassword")

	sessionID, err := storage.CreateOIDCLoginSession(
		context.Background(), tf.Storage, testClientID, []storage.ResponseType{storage.AuthorizationCodeResponseType},
		testRedirectURL, testState, "unusedscope")
	assert.NoErr(t, err)
	resetURL, otpCode := submitResetRequest(t, tf, testClientID, email, sessionID)

	// Mess with the code
	q := resetURL.Query()
	q.Set("otp_code", otpCode+"garbage")
	resetURL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	req := tf.RequestFactory.NewRequest(http.MethodGet, resetURL.String(), nil)
	tf.Handler.ServeHTTP(w, req)
	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
}

func TestInvalidResetCode(t *testing.T) {
	tcb, testClientID := test.NewFollowerTenantConfigBuilder()
	tf := test.NewFixture(t, tcb.Build())

	email, _, _ := createUserWithPassword(t, tf, "oldpassword")

	sessionID, err := storage.CreateOIDCLoginSession(
		context.Background(), tf.Storage, testClientID, []storage.ResponseType{storage.AuthorizationCodeResponseType},
		testRedirectURL, testState, "unusedscope")
	assert.NoErr(t, err)
	_, otpCode := submitResetRequest(t, tf, testClientID, email, sessionID)

	// Mess with the code
	submitReqBody := plex.PasswordResetSubmitRequest{
		SessionID: sessionID,
		OTPCode:   otpCode + "garbage",
		Password:  "unused",
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/resetpassword/resetsubmit", uctest.IOReaderFromJSONStruct(t, submitReqBody))
	tf.Handler.ServeHTTP(w, req)
	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusNotFound)
}

func TestFailedReset(t *testing.T) {
	tcb, testClientID := test.NewFollowerTenantConfigBuilder()
	tf := test.NewFixture(t, tcb.Build())

	type testCase struct {
		name        string
		idpToFail   *test.IDP
		errorString string
	}
	testCases := []testCase{
		{
			name:        "ActiveFailed",
			idpToFail:   tf.ActiveIDP,
			errorString: "failed to update username & password",
		},
		{
			name:        "FollowerFailed",
			idpToFail:   tf.FollowerIDP,
			errorString: "error updating password on follower client",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tt := testlogtransport.InitLoggerAndTransportsForTests(t)
			tc.idpToFail.FailNextUUPRequest = true

			//tf.FollowerIDP.FailNextUUPRequest = true
			tf.ActiveIDP.Users = map[string]test.User{}
			tf.FollowerIDP.Users = map[string]test.User{}

			email, _, _ := createUserWithPassword(t, tf, "oldpassword")

			sessionID, err := storage.CreateOIDCLoginSession(
				context.Background(), tf.Storage, testClientID, []storage.ResponseType{storage.AuthorizationCodeResponseType},
				testRedirectURL, testState, "unusedscope")
			assert.NoErr(t, err)
			_, otpCode := submitResetRequest(t, tf, testClientID, email, sessionID)

			submitReqBody := plex.PasswordResetSubmitRequest{
				SessionID: sessionID,
				OTPCode:   otpCode,
				Password:  crypto.GenerateOpaqueAccessToken(),
			}
			w := httptest.NewRecorder()
			req := tf.RequestFactory.NewRequest(http.MethodPost, "/resetpassword/resetsubmit", uctest.IOReaderFromJSONStruct(t, submitReqBody))
			tf.Handler.ServeHTTP(w, req)
			resp := w.Result()
			assert.Equal(t, resp.StatusCode, http.StatusInternalServerError)
			var errResponse struct {
				Error string `json:"error"`
			}
			defer resp.Body.Close()
			err = json.NewDecoder(resp.Body).Decode(&errResponse)
			assert.NoErr(t, err)

			// we check the logs for the error message to ensure we're hitting the right codepath,
			// but the response itself will be sanitized
			tt.AssertLogsContainString(tc.errorString)
		})
	}
}

func TestRateLimit(t *testing.T) {
	tcb, testClientID := test.NewBasicTenantConfigBuilder()
	tf := test.NewFixture(t, tcb.Build())

	email, _, _ := createUserWithPassword(t, tf, "oldpassword")

	for i := range 6 {
		sessionID, err := storage.CreateOIDCLoginSession(
			context.Background(), tf.Storage, testClientID, []storage.ResponseType{storage.AuthorizationCodeResponseType},
			testRedirectURL, testState, "unusedscope")
		assert.NoErr(t, err)

		// POST request to start the password reset flow.
		startReqBody := plex.PasswordResetStartRequest{
			SessionID: sessionID,
			Email:     email,
		}
		w := httptest.NewRecorder()

		req := tf.RequestFactory.NewRequest(http.MethodPost, "/resetpassword/startsubmit", uctest.IOReaderFromJSONStruct(t, startReqBody))
		tf.Handler.ServeHTTP(w, req)
		resp := w.Result()

		if i != 5 {
			assert.Equal(t, resp.StatusCode, http.StatusNoContent)
		} else {
			assert.Equal(t, resp.StatusCode, http.StatusTooManyRequests)
		}
	}
}

func TestResetDoenstLeakData(t *testing.T) {
	tcb, testClientID := test.NewBasicTenantConfigBuilder()
	tf := test.NewFixture(t, tcb.Build())

	email, _, _ := createUserWithPassword(t, tf, "oldpassword")

	tryReset := func(email string) *http.Response {
		sessionID, err := storage.CreateOIDCLoginSession(
			context.Background(), tf.Storage, testClientID, []storage.ResponseType{storage.AuthorizationCodeResponseType},
			testRedirectURL, testState, "unusedscope")
		assert.NoErr(t, err)

		// POST request to start the password reset flow.
		startReqBody := plex.PasswordResetStartRequest{
			SessionID: sessionID,
			Email:     email,
		}
		w := httptest.NewRecorder()

		req := tf.RequestFactory.NewRequest(http.MethodPost, "/resetpassword/startsubmit", uctest.IOReaderFromJSONStruct(t, startReqBody))
		tf.Handler.ServeHTTP(w, req)
		return w.Result()
	}

	good := tryReset(email)
	bad := tryReset("notauser@contoso.com")

	assert.Equal(t, good.StatusCode, bad.StatusCode)

	// remove request ID header from requests since they are not going to be the same.
	good.Header.Del("X-Request-Id")
	bad.Header.Del("X-Request-Id")
	assert.Equal(t, good.Header, bad.Header)

	gb, err := io.ReadAll(good.Body)
	assert.NoErr(t, err)
	bb, err := io.ReadAll(bad.Body)
	assert.NoErr(t, err)

	assert.Equal(t, string(gb), string(bb)) // string to make debugging easier
}

func TestMain(m *testing.M) {
	// Adjust working dir to match what our services expect.
	os.Chdir("../../..")
	os.Exit(m.Run())
}
