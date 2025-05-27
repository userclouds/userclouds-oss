package internal_test

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"
	"golang.org/x/oauth2"

	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/internal/uctest"
	"userclouds.com/plex"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/test"
	"userclouds.com/plex/manager"
)

func doImpersonateUser(tf *test.E2EFixture, accessToken, targetUserID string) (*plex.LoginResponse, error) {
	ctx := context.Background()
	plexClient := jsonclient.New(tf.TenantURL)
	req := &plex.ImpersonateUserRequest{
		AccessToken:  accessToken,
		TargetUserID: targetUserID,
	}

	var loginResponse plex.LoginResponse
	err := plexClient.Post(ctx, "/impersonateuser", req, &loginResponse)
	return &loginResponse, ucerr.Wrap(err)
}

func doPasswordChange(tf *test.E2EFixture, redirectURL, state, email, password string) error {
	ctx := context.Background()
	plexClient := jsonclient.New(tf.TenantURL)
	sessionID, err := storage.CreateOIDCLoginSession(
		ctx, tf.PlexStorage, tf.PlexClientID, []storage.ResponseType{storage.AuthorizationCodeResponseType},
		uctest.MustParseURL(redirectURL), state, "unusedscope")
	if err != nil {
		return ucerr.Wrap(err)
	}

	startReq := &plex.PasswordResetStartRequest{
		SessionID: sessionID,
		Email:     email,
	}

	emailBodies := len(tf.Email.Bodies)

	err = plexClient.Post(ctx, "/resetpassword/startsubmit", startReq, nil)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if len(tf.Email.Bodies) != emailBodies+1 {
		return ucerr.New("expected password reset flow to send an email")
	}

	resetLinkURL, err := uctest.ExtractURL(html.UnescapeString(tf.Email.Bodies[emailBodies]))
	if err != nil {
		return ucerr.Wrap(err)
	}

	otpCode := resetLinkURL.Query().Get("otp_code")

	submitReq := &plex.PasswordResetSubmitRequest{
		SessionID: sessionID,
		OTPCode:   otpCode,
		Password:  password,
	}

	err = plexClient.Post(ctx, "/resetpassword/resetsubmit", submitReq, nil)
	if err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

func TestUnified(t *testing.T) {
	t.Parallel()
	tf := test.NewE2EFixture(t)

	t.Run("TestUsernamePasswordLogin", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		username := test.GenUsername()
		password := test.GenPassword()
		name := fmt.Sprintf("Foo%s", crypto.MustRandomDigits(6))
		email := fmt.Sprintf("foo_%s@contoso.com", crypto.MustRandomDigits(6))
		redirectURL := fmt.Sprintf("contoso.com/redirect_%s", crypto.MustRandomDigits(6))
		state := crypto.GenerateOpaqueAccessToken()
		_, err := tf.IdpClient.CreateUserWithPassword(ctx, username, password, userstore.Record{
			"name":  name,
			"email": email,
		})
		assert.NoErr(t, err)

		loginResponse, err := tf.DoLogin(redirectURL, state, username, password)
		assert.NoErr(t, err)
		assert.Contains(t, loginResponse.RedirectTo, redirectURL)
		// Ensure 'code' is present and 'state' matches
		assert.True(t, len(uctest.MustParseURL(loginResponse.RedirectTo).Query().Get("code")) > 0)
		assert.Equal(t, uctest.MustParseURL(loginResponse.RedirectTo).Query().Get("state"), state)
	})

	// TestUpdateUsernamePasswordOnIDP ensures login works before/after changing
	// the user's password on the underlying IDP.
	t.Run("TestUpdateUsernamePassword", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		username := test.GenUsername()
		email := fmt.Sprintf("foo_%s@contoso.com", crypto.MustRandomDigits(6))
		oldPassword := test.GenPassword()
		newPassword := test.GenPassword()

		// Sanity check what the IDP client does with an invalid username
		// TODO: maybe not the best place for this check
		err := tf.IdpClient.UpdateUsernamePassword(ctx, username, newPassword)
		var jsonClientErr jsonclient.Error
		assert.True(t, errors.As(err, &jsonClientErr), assert.Must())
		assert.Equal(t, jsonClientErr.StatusCode, http.StatusNotFound)

		// Password change fails on unknown user.
		err = doPasswordChange(tf, "http://unused", "unused", email, newPassword)
		assert.NotNil(t, err, assert.Must())

		_, err = tf.IdpClient.CreateUserWithPassword(ctx, username, oldPassword, userstore.Record{
			"email": email,
		})
		assert.NoErr(t, err)

		// Login with old password successfully.
		_, err = tf.DoLogin("unused", "unused", username, oldPassword)
		assert.NoErr(t, err)

		// Change password, then ensure old password doesn't work but new password does.
		err = doPasswordChange(tf, "http://unused", "unused", email, newPassword)
		assert.NoErr(t, err)
		_, err = tf.DoLogin("unused", "unused", username, oldPassword)
		assert.NotNil(t, err, assert.Must())
		_, err = tf.DoLogin("unused", "unused", username, newPassword)
		assert.NoErr(t, err)
	})

	t.Run("TestImpersonateUser", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		// Set up a user that is not a company admin
		username := test.GenUsername()
		password := test.GenPassword()
		name := fmt.Sprintf("Foo%s", crypto.MustRandomDigits(6))
		email := fmt.Sprintf("foo_%s@contoso.com", crypto.MustRandomDigits(6))
		userID, err := tf.IdpClient.CreateUserWithPassword(ctx, username, password, userstore.Record{
			"name":  name,
			"email": email,
		})
		assert.NoErr(t, err)

		state := crypto.GenerateOpaqueAccessToken()
		loginResponse, err := tf.DoLogin(tf.RedirectURI, state, username, password)
		assert.NoErr(t, err)

		parsedURI := uctest.MustParseURL(loginResponse.RedirectTo)
		code := parsedURI.Query().Get("code")

		var oauthConfig = oauth2.Config{
			ClientID:     tf.PlexClientID,
			ClientSecret: tf.PlexSecret,
			Endpoint:     oauth2.Endpoint{AuthURL: tf.TenantURL + "/oidc/authorize", TokenURL: tf.TenantURL + "/oidc/token"},
			RedirectURL:  tf.RedirectURI,
			Scopes:       []string{"openid"},
		}

		// Retrieve an access token for the user
		token, err := oauthConfig.Exchange(ctx, code)
		assert.NoErr(t, err)

		// Set up a second user
		username2 := test.GenUsername()
		password2 := test.GenPassword()
		name2 := fmt.Sprintf("Foo%s", crypto.MustRandomDigits(6))
		email2 := fmt.Sprintf("foo_%s@contoso.com", crypto.MustRandomDigits(6))
		userID2, err := tf.IdpClient.CreateUserWithPassword(ctx, username2, password2, userstore.Record{
			"name":  name2,
			"email": email2,
		})
		assert.NoErr(t, err)

		// Attempt an impersonation, and verify that it fails
		_, err = doImpersonateUser(tf, token.AccessToken, userID2.String())
		assert.NotNil(t, err, assert.Must())

		// Make the original user a company admin
		_, err = tf.AuthzClient.CreateEdge(ctx, uuid.Must(uuid.NewV4()), userID, tf.CompanyID, ucauthz.AdminEdgeTypeID)
		assert.NoErr(t, err)

		// verify that as company admin, they can impersonate the user
		loginResponse, err = doImpersonateUser(tf, token.AccessToken, userID2.String())
		assert.NoErr(t, err)

		// check the expiration of the returned tokens
		code2 := uctest.MustParseURL(loginResponse.RedirectTo).Query().Get("code")

		// Retrieve an access token for the user
		token2, err := oauthConfig.Exchange(ctx, code2)
		assert.NoErr(t, err)

		// verify that the subject of the new token matches the user being impersonated
		tokenClaims, err := ucjwt.ParseUCClaimsUnverified(token2.AccessToken)
		assert.NoErr(t, err)
		assert.Equal(t, tokenClaims.Subject, userID2.String())
	})

	t.Run("TestRestrictedAccessLoginApp", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		tf := test.NewE2EFixture(t)

		// Remove and re-add the login app with restricted access
		mgr := manager.NewFromDB(tf.TenantDB, cachetesthelpers.NewCacheConfig())
		tp, err := mgr.GetTenantPlex(ctx, tf.TenantID)
		assert.NoErr(t, err)

		app := tp.PlexConfig.PlexMap.Apps[0]
		err = mgr.DeleteLoginApp(ctx, tf.TenantID, tf.AuthzClient, app.ID)
		assert.NoErr(t, err)

		app.RestrictedAccess = true
		err = mgr.AddLoginApp(ctx, tf.TenantID, tf.AuthzClient, app)
		assert.NoErr(t, err)

		// Create a user
		username := test.GenUsername()
		password := test.GenPassword()
		name := fmt.Sprintf("Foo%s", crypto.MustRandomDigits(6))
		email := fmt.Sprintf("foo_%s@contoso.com", crypto.MustRandomDigits(6))
		redirectURL := fmt.Sprintf("contoso.com/redirect_%s", crypto.MustRandomDigits(6))
		state := crypto.GenerateOpaqueAccessToken()
		userID, err := tf.IdpClient.CreateUserWithPassword(ctx, username, password, userstore.Record{
			"name":  name,
			"email": email,
		})
		assert.NoErr(t, err)

		// Attempt to login, and verify that it fails due to restricted access
		_, err = tf.DoLogin(redirectURL, state, username, password)
		assert.NotNil(t, err)

		// Add the CanLogin edge and verify that the login succeeds
		_, err = tf.AuthzClient.CreateEdge(ctx, uuid.Must(uuid.NewV4()), userID, tp.PlexConfig.PlexMap.Apps[0].ID, authz.CanLoginEdgeTypeID)
		assert.NoErr(t, err)

		_, err = tf.DoLogin(redirectURL, state, username, password)
		assert.NoErr(t, err)
	})
}
