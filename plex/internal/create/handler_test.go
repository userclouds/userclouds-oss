package create_test

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/crypto"
	"userclouds.com/internal/uctest"
	"userclouds.com/plex"
	"userclouds.com/plex/internal/otp"
	"userclouds.com/plex/internal/test"
)

func genCredentials() (email string, username string, password string) {
	rand := crypto.GenerateOpaqueAccessToken()
	email = fmt.Sprintf("user_%s@contoso.com", rand)
	username = fmt.Sprintf("user_%s", rand)
	password = fmt.Sprintf("password_%s", rand)
	return
}

// createUser issues a request to the test Plex to create a user and returns the http response
func createUser(t *testing.T, plexHandler http.Handler, rf *test.RequestFactory, clientID, email, username, password string) *httptest.ResponseRecorder {
	return createUserWithSession(t, plexHandler, rf, uuid.Nil, clientID, email, username, password)

}

func createUserWithSession(t *testing.T, plexHandler http.Handler, rf *test.RequestFactory, sessionID uuid.UUID, clientID, email, username, password string) *httptest.ResponseRecorder {
	req := plex.CreateUserRequest{
		SessionID: sessionID,
		ClientID:  clientID,
		Email:     email,
		Username:  username,
		Password:  password,
	}
	r := rf.NewRequest(http.MethodPost, "/create/submit", uctest.IOReaderFromJSONStruct(t, req))
	rr := httptest.NewRecorder()
	plexHandler.ServeHTTP(rr, r)
	return rr
}

func TestCreate(t *testing.T) {
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	tf := test.NewFixture(t, tcb.Build())

	email, username, password := genCredentials()
	rr := createUser(t, tf.Handler, tf.RequestFactory, clientID, email, username, password)
	assert.Equal(t, rr.Code, http.StatusCreated, assert.Must())

	// No verification email sent
	assert.Equal(t, len(tf.Email.Bodies), 0)
	assert.Equal(t, len(tf.Email.HTMLBodies), 0)
	// 1 user created
	assert.Equal(t, len(tf.ActiveIDP.Users), 1, assert.Must())
	for _, v := range tf.ActiveIDP.Users {
		assert.Equal(t, v, test.User{
			Authn: idp.NewPasswordAuthn(username, password),
			Profile: userstore.Record{
				"email":          email,
				"email_verified": false,
				"name":           "",
				"nickname":       "",
				"picture":        "",
			},
		})
	}
}

func TestCreateDisabled(t *testing.T) {
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	tcb.SetDisableSignUps(true)
	tf := test.NewFixture(t, tcb.Build())

	email, username, password := genCredentials()
	rr := createUser(t, tf.Handler, tf.RequestFactory, clientID, email, username, password)

	assert.Equal(t, rr.Code, http.StatusBadRequest)

	// But if we have an invite it's ok
	ctx := context.Background()
	sessionID, _, err := otp.CreateInviteSession(ctx, tf.Storage, clientID, email, "unusedstate", &url.URL{Scheme: "https://"}, otp.UseDefaultExpiry)
	assert.NoErr(t, err)

	rr = createUserWithSession(t, tf.Handler, tf.RequestFactory, sessionID, clientID, email, username, password)
	assert.Equal(t, rr.Code, http.StatusCreated)
}

func TestCreateEmailUsername(t *testing.T) {
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	tf := test.NewFixture(t, tcb.Build())

	email, _, password := genCredentials()
	rr := createUser(t, tf.Handler, tf.RequestFactory, clientID, email, email, password)
	assert.Equal(t, rr.Code, http.StatusCreated, assert.Must())

	// 1 user created
	assert.Equal(t, len(tf.ActiveIDP.Users), 1, assert.Must())
	for _, v := range tf.ActiveIDP.Users {
		assert.Equal(t, v, test.User{
			Authn: idp.NewPasswordAuthn(email, password),
			Profile: userstore.Record{
				"email":          email,
				"email_verified": false,
				"name":           "",
				"nickname":       "",
				"picture":        "",
			},
		})
	}
}

func TestCreateWithEmail(t *testing.T) {
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	tcb.SetVerifyEmails(true)
	tf := test.NewFixture(t, tcb.Build())

	email, username, password := genCredentials()
	rr := createUser(t, tf.Handler, tf.RequestFactory, clientID, email, username, password)
	assert.Equal(t, rr.Code, http.StatusCreated)
	// 1 user created, email not verified
	assert.Equal(t, len(tf.ActiveIDP.Users), 1, assert.Must())
	for _, v := range tf.ActiveIDP.Users {
		assert.Equal(t, v, test.User{
			Authn: idp.NewPasswordAuthn(username, password),
			Profile: userstore.Record{
				"email":          email,
				"email_verified": false,
				"name":           "",
				"nickname":       "",
				"picture":        "",
			},
		})
	}

	// Verification email sent, URL(s) valid
	assert.Equal(t, len(tf.Email.Bodies), 1, assert.Must())
	assert.Equal(t, len(tf.Email.HTMLBodies), 1, assert.Must())
	resetLinkURLFromHTML, err := uctest.ExtractURL(tf.Email.Bodies[0])
	assert.NoErr(t, err)
	resetLinkURL, err := uctest.ExtractURL(html.UnescapeString(tf.Email.HTMLBodies[0]))
	assert.NoErr(t, err)
	assert.Equal(t, resetLinkURLFromHTML, resetLinkURL)
}

func TestCreateWithEmailValidateOTP(t *testing.T) {
	tcb, clientID := test.NewBasicTenantConfigBuilder()
	tcb.SetVerifyEmails(true)
	tf := test.NewFixture(t, tcb.Build())

	email, username, password := genCredentials()
	rr := createUser(t, tf.Handler, tf.RequestFactory, clientID, email, username, password)
	assert.Equal(t, rr.Code, http.StatusCreated)
	// 1 user created, email not verified
	assert.Equal(t, len(tf.ActiveIDP.Users), 1, assert.Must())
	for _, v := range tf.ActiveIDP.Users {
		assert.Equal(t, v, test.User{
			Authn: idp.NewPasswordAuthn(username, password),
			Profile: userstore.Record{
				"email":          email,
				"email_verified": false,
				"name":           "",
				"nickname":       "",
				"picture":        "",
			},
		})
	}

	// Verification email sent, URL(s) valid
	assert.Equal(t, len(tf.Email.Bodies), 1, assert.Must())
	assert.Equal(t, len(tf.Email.HTMLBodies), 1, assert.Must())
	resetLinkURLFromHTML, err := uctest.ExtractURL(tf.Email.Bodies[0])
	assert.NoErr(t, err)
	resetLinkURL, err := uctest.ExtractURL(html.UnescapeString(tf.Email.HTMLBodies[0]))
	assert.NoErr(t, err)
	assert.Equal(t, resetLinkURLFromHTML, resetLinkURL)

	resetLinkURLString := strings.ReplaceAll(resetLinkURL.String(), "/plexui", "")
	req := tf.RequestFactory.NewRequest(http.MethodGet, resetLinkURLString, nil)
	rr = httptest.NewRecorder()
	tf.Handler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusOK)
	defer rr.Result().Body.Close()
	var msg string
	err = json.NewDecoder(rr.Result().Body).Decode(&msg)
	assert.NoErr(t, err)
	// TODO: probably too specific of a test for now, we should standardize on an error message
	// and put it in a constant
	assert.Contains(t, msg, "verified email")
	assert.Equal(t, len(tf.ActiveIDP.Users), 1, assert.Must())
	for _, v := range tf.ActiveIDP.Users {
		assert.Equal(t, v, test.User{
			Authn: idp.NewPasswordAuthn(username, password),
			Profile: userstore.Record{
				"email":          email,
				"email_verified": true,
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
