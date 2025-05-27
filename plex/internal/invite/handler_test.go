package invite_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/tenantplex"
	plexconfigtest "userclouds.com/internal/tenantplex/test"
	"userclouds.com/internal/uctest"
	"userclouds.com/plex"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/test"
)

const testState = "unused"
const inviterEmail = "earlyadopter@contoso.com"

// inviteUser issues a request to the test Plex to invite a user and returns the http response
func inviteUser(plexHandler http.Handler,
	rf *test.RequestFactory,
	jwt,
	redirectURL,
	clientID,
	inviteeEmail,
	inviterUserID,
	inviterName,
	inviterEmail string) (*httptest.ResponseRecorder, error) {

	req := plex.SendInviteRequest{
		ClientID:      clientID,
		InviteeEmail:  inviteeEmail,
		InviterName:   inviterName,
		InviterEmail:  inviterEmail,
		InviterUserID: inviterUserID,
		RedirectURL:   redirectURL,
		State:         testState,
	}

	bs, err := json.Marshal(req)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	r := rf.NewRequest(http.MethodPost, "/invite/send", bytes.NewReader(bs))
	if jwt != "" {
		r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))
	}

	rr := httptest.NewRecorder()

	plexHandler.ServeHTTP(rr, r)
	return rr, nil
}

// getRedirectInfoFromMagicLink follows the magic link URL and parses the expected redirect URL from the response
func getRedirectInfoFromMagicLink(tf *test.Fixture, magicLink *url.URL) (*url.URL, error) {
	w := httptest.NewRecorder()
	r := tf.RequestFactory.NewRequest(http.MethodGet, magicLink.String(), nil)
	tf.Handler.ServeHTTP(w, r)
	if w.Code != http.StatusTemporaryRedirect {
		return nil, ucerr.Errorf("expected code %d, got %d", http.StatusTemporaryRedirect, w.Code)
	}
	redirectURL, err := w.Result().Location()
	return redirectURL, ucerr.Wrap(err)
}

// validateSessionID ensures the URL has a "session_id" and sanity checks it against storage
func validateSessionID(ctx context.Context, tf *test.Fixture, plexURL *url.URL) error {
	sessionID, err := uuid.FromString(plexURL.Query().Get("session_id"))
	if err != nil {
		return ucerr.Wrap(err)
	}
	_, err = tf.Storage.GetOIDCLoginSession(ctx, sessionID)
	return ucerr.Wrap(err)
}

func newBasicTenantConfig() (tc tenantplex.TenantConfig, clientID, redirectURI string) {
	redirectURI = fmt.Sprintf("http://contoso_%s.com/callback", crypto.MustRandomHex(4))
	tcb := plexconfigtest.NewTenantConfigBuilder()
	clientID = tcb.AddProvider().MakeActive().MakeUC().AddUCApp().
		AddApp().AddAllowedRedirectURI(redirectURI).ClientID()
	tc = tcb.Build()
	return
}

func TestInviteRequiresAuth(t *testing.T) {
	tc, clientID, redirectURL := newBasicTenantConfig()
	tf := test.NewFixture(t, tc)
	ctx := context.Background()
	_, err := tf.ActiveIDP.CreateUserWithPassword(ctx, inviterEmail, "pw", iface.UserProfile{UserBaseProfile: idp.UserBaseProfile{Email: inviterEmail}})
	assert.NoErr(t, err)
	inviterProfile, err := tf.ActiveIDP.ListUsersForEmail(ctx, inviterEmail, idp.AuthnTypePassword)
	assert.NoErr(t, err)
	assert.Equal(t, len(inviterProfile), 1, assert.Must())

	// No Authorization header or valid token provided
	jwt := ""
	rr, err := inviteUser(tf.Handler, tf.RequestFactory, jwt, redirectURL, clientID, "slowloris@contoso.com", inviterProfile[0].ID, inviterProfile[0].Name, inviterProfile[0].Email)
	assert.NoErr(t, err)
	assert.Equal(t, rr.Code, http.StatusUnauthorized)
}

func TestInviteNewUser(t *testing.T) {
	tc, clientID, redirectURL := newBasicTenantConfig()
	tf := test.NewFixture(t, tc)
	ctx := context.Background()
	_, err := tf.ActiveIDP.CreateUserWithPassword(ctx, inviterEmail, "pw", iface.UserProfile{UserBaseProfile: idp.UserBaseProfile{Email: inviterEmail}})
	assert.NoErr(t, err)
	inviterProfile, err := tf.ActiveIDP.ListUsersForEmail(ctx, inviterEmail, idp.AuthnTypePassword)
	assert.NoErr(t, err)
	assert.Equal(t, len(inviterProfile), 1, assert.Must())

	jwt := uctest.CreateJWT(t, oidc.UCTokenClaims{SubjectType: authz.ObjectTypeLoginApp}, tf.Tenant.TenantURL)
	rr, err := inviteUser(tf.Handler, tf.RequestFactory, jwt, redirectURL, clientID, "slowloris@contoso.com", inviterProfile[0].ID, inviterProfile[0].Name, inviterProfile[0].Email)
	assert.NoErr(t, err)
	assert.Equal(t, rr.Code, http.StatusNoContent)
	assert.Equal(t, len(tf.Email.Bodies), 1, assert.Must())
	assert.Equal(t, len(tf.Email.HTMLBodies), 1, assert.Must())
	assert.Contains(t, tf.Email.Bodies[0], "to sign up")
	assert.Contains(t, tf.Email.HTMLBodies[0], "to sign up")

	// Parse HTML email for a link
	magicLink, err := uctest.ExtractURL(tf.Email.HTMLBodies[0])
	assert.NoErr(t, err)

	// We should get redirected to the Login page with a valid session ID
	magicLinkRedirectURL, err := getRedirectInfoFromMagicLink(tf, magicLink)
	assert.NoErr(t, err)
	assert.Contains(t, magicLinkRedirectURL.Path, paths.LoginUISubPath)
	err = validateSessionID(ctx, tf, magicLinkRedirectURL)
	assert.NoErr(t, err)

	// test we don't allow HTML injection in the name
	htmlInjectionString := "<script>alert('boo')</script>"
	tf.Email.Clear()
	jwt = uctest.CreateJWT(t, oidc.UCTokenClaims{SubjectType: authz.ObjectTypeLoginApp}, tf.Tenant.TenantURL)
	rr, err = inviteUser(tf.Handler, tf.RequestFactory, jwt, redirectURL, clientID, "slowloris@contoso.com", inviterProfile[0].ID, inviterProfile[0].Name+htmlInjectionString, inviterProfile[0].Email)
	assert.NoErr(t, err)
	assert.Equal(t, rr.Code, http.StatusNoContent)
	assert.Equal(t, len(tf.Email.Bodies), 1, assert.Must())
	assert.Equal(t, len(tf.Email.HTMLBodies), 1, assert.Must())
	assert.Contains(t, tf.Email.Bodies[0], "to sign up")
	assert.Contains(t, tf.Email.HTMLBodies[0], "to sign up")
	assert.DoesNotContain(t, tf.Email.HTMLBodies[0], htmlInjectionString)

}

func TestInviteExistingUser(t *testing.T) {
	tc, clientID, redirectURL := newBasicTenantConfig()
	tf := test.NewFixture(t, tc)
	ctx := context.Background()
	_, err := tf.ActiveIDP.CreateUserWithPassword(ctx, inviterEmail, "pw", iface.UserProfile{UserBaseProfile: idp.UserBaseProfile{Email: inviterEmail}})
	assert.NoErr(t, err)
	inviteeEmail := "existinguser@contoso.com"
	_, err = tf.ActiveIDP.CreateUserWithPassword(ctx, inviteeEmail, "pw", iface.UserProfile{UserBaseProfile: idp.UserBaseProfile{Email: inviteeEmail}})
	assert.NoErr(t, err)

	inviterProfile, err := tf.ActiveIDP.ListUsersForEmail(ctx, inviterEmail, idp.AuthnTypeAll)
	assert.NoErr(t, err)
	assert.Equal(t, len(inviterProfile), 1, assert.Must())

	inviteeProfile, err := tf.ActiveIDP.ListUsersForEmail(ctx, inviteeEmail, idp.AuthnTypeAll)
	assert.NoErr(t, err)
	assert.Equal(t, len(inviteeProfile), 1, assert.Must())

	jwt := uctest.CreateJWT(t, oidc.UCTokenClaims{SubjectType: authz.ObjectTypeLoginApp}, tf.Tenant.TenantURL)
	rr, err := inviteUser(tf.Handler, tf.RequestFactory, jwt, redirectURL, clientID, inviteeEmail, inviterProfile[0].ID, inviterProfile[0].Name, inviterProfile[0].Email)
	assert.NoErr(t, err)
	assert.Equal(t, rr.Code, http.StatusNoContent)
	assert.Equal(t, len(tf.Email.Bodies), 1, assert.Must())
	assert.Equal(t, len(tf.Email.HTMLBodies), 1)
	assert.Contains(t, tf.Email.Bodies[0], "There is an account associated with this email address")
	assert.Contains(t, tf.Email.HTMLBodies[0], "There is an account associated with this email address")

	// Parse Text email for a link
	magicLink, err := uctest.ExtractURL(tf.Email.Bodies[0])
	assert.NoErr(t, err)

	// We should get redirected to the Login UI page with a valid session ID
	magicLinkRedirectURL, err := getRedirectInfoFromMagicLink(tf, magicLink)
	assert.NoErr(t, err)
	assert.Contains(t, magicLinkRedirectURL.Path, paths.LoginUISubPath)
	err = validateSessionID(ctx, tf, magicLinkRedirectURL)
	assert.NoErr(t, err)
}

func TestMain(m *testing.M) {
	// Adjust working dir to match what our services expect.
	os.Chdir("../../..")
	os.Exit(m.Run())
}
