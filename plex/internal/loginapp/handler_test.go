package loginapp_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"golang.org/x/oauth2"

	"userclouds.com/authz/ucauthz"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/internal/uctest"
	"userclouds.com/plex"
	"userclouds.com/plex/internal/test"
)

func TestLoginApp(t *testing.T) {
	ctx := context.Background()
	tf := test.NewE2EFixture(t)

	// Set up a user that is initially not a company admin
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
	code := uctest.MustParseURL(loginResponse.RedirectTo).Query().Get("code")

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

	// Attempt to create a login app, expecting it to fail
	plexClient := plex.NewClient(tf.TenantURL, jsonclient.HeaderAuthBearer(token.AccessToken))
	_, err = plexClient.CreateLoginApp(ctx, &plex.LoginAppRequest{
		ClientName: "newapp",
	})
	assert.NotNil(t, err)

	// Make the user an admin and try again
	_, err = tf.AuthzClient.CreateEdge(ctx, uuid.Must(uuid.NewV4()), userID, tf.CompanyID, ucauthz.AdminEdgeTypeID)
	assert.NoErr(t, err)
	resp, err := plexClient.CreateLoginApp(ctx, &plex.LoginAppRequest{
		ClientName: "newapp",
	})
	assert.NoErr(t, err)
	assert.True(t, resp.ClientID != "", assert.Must())
	assert.Equal(t, resp.Metadata.ClientName, "newapp", assert.Must())

	// Verify that the app exists
	resp, err = plexClient.GetLoginApp(ctx, resp.AppID)
	assert.NoErr(t, err)
	assert.Equal(t, resp.Metadata.ClientName, "newapp", assert.Must())

	// List the login apps and verify that it is present
	apps, err := plexClient.ListLoginApps(ctx, uuid.Nil)
	assert.NoErr(t, err)
	found := false
	for _, app := range apps {
		if app.AppID == resp.AppID {
			found = true
			break
		}
	}
	assert.True(t, found)

	// Update the login app
	req := resp.Metadata
	req.ClientName = "newapp_renamed"
	resp, err = plexClient.UpdateLoginApp(ctx, &req, resp.AppID)
	assert.NoErr(t, err)
	assert.Equal(t, resp.Metadata.ClientName, "newapp_renamed", assert.Must())

	// Verify that invalid grant types aren't accepted
	req.GrantTypes = []string{"invalid_grant_type"}
	_, err = plexClient.UpdateLoginApp(ctx, &req, resp.AppID)
	assert.NotNil(t, err, assert.Must())

	// Delete the login app
	err = plexClient.DeleteLoginApp(ctx, resp.AppID)
	assert.NoErr(t, err)

	// Verify that the app is gone
	_, err = plexClient.GetLoginApp(ctx, resp.AppID)
	assert.NotNil(t, err, assert.Must())
}
