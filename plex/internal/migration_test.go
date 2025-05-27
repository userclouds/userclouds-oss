package internal_test

import (
	"context"
	"testing"

	"userclouds.com/idp"
	"userclouds.com/infra/assert"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/test"
)

func TestMigrateUserOnLogin(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tcb, clientID := test.NewFollowerTenantConfigBuilder()
	tf := test.NewFixture(t, tcb.Build())

	username := test.GenUsername()
	password := test.GenPassword()

	// Create a user in the Active IDP, then login and ensure the Follower gets the account migrated.
	userID, err := tf.ActiveIDP.CreateUserWithPassword(ctx, username, password, iface.UserProfile{
		UserBaseProfile: idp.UserBaseProfile{Email: testEmail},
	})
	assert.NoErr(t, err)
	lr := createLoginRequest(t, tf.Storage, "unused", clientID, username, password)
	_, _, _, oauthErr, err := doLogin(t, tf.RequestFactory, tf.Handler, lr)
	assert.NoErr(t, err)
	assert.NotNil(t, oauthErr, assert.Must())
	assert.Equal(t, len(tf.ActiveIDP.Users), 1, assert.Must())
	assert.Equal(t, len(tf.FollowerIDP.Users), 1, assert.Must())
	assert.Equal(t, tf.FollowerIDP.Users[userID].Profile["email"], testEmail)
	assert.Equal(t, tf.FollowerIDP.Users[userID].Authn.AuthnType, idp.AuthnTypePassword)
	assert.Equal(t, tf.FollowerIDP.Users[userID].Authn.Username, username)
	assert.Equal(t, tf.FollowerIDP.Users[userID].Authn.Password, password)
	// Both profile and authn should get sync'd
	assert.Equal(t, tf.ActiveIDP.Users[userID].Profile, tf.FollowerIDP.Users[userID].Profile)
	assert.Equal(t, tf.ActiveIDP.Users[userID].Authn, tf.FollowerIDP.Users[userID].Authn)
}

func TestUpdateFollowerUserOnLogin(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tcb, clientID := test.NewFollowerTenantConfigBuilder()
	tf := test.NewFixture(t, tcb.Build())

	username := test.GenUsername()
	password := test.GenPassword()
	oldPassword := test.GenPassword()

	// Create a user in the Active IDP, then login and ensure the Follower gets the account migrated.
	userID, err := tf.ActiveIDP.CreateUserWithPassword(ctx, username, password, iface.UserProfile{
		UserBaseProfile: idp.UserBaseProfile{Email: testEmail,
			Name: "new name",
		}})
	assert.NoErr(t, err)
	// Create a user with the same username in the Follower IDP, but with different profile & older password
	_, err = tf.FollowerIDP.CreateUserWithPassword(ctx, username, oldPassword, iface.UserProfile{UserBaseProfile: idp.UserBaseProfile{
		Email: testEmail,
		Name:  "old name",
	}})
	assert.NoErr(t, err)
	lr := createLoginRequest(t, tf.Storage, "unused", clientID, username, password)
	_, _, _, oauthErr, err := doLogin(t, tf.RequestFactory, tf.Handler, lr)
	assert.NoErr(t, err)
	assert.NotNil(t, oauthErr, assert.Must())
	assert.Equal(t, len(tf.ActiveIDP.Users), 1, assert.Must())
	assert.Equal(t, len(tf.FollowerIDP.Users), 1, assert.Must())
	// Profile SHOULD NOT get sync'd
	assert.NotEqual(t, tf.ActiveIDP.Users[userID].Profile, tf.FollowerIDP.Users[userID].Profile)
	// AuthN SHOULD get sync'd
	assert.Equal(t, tf.ActiveIDP.Users[userID].Authn, tf.FollowerIDP.Users[userID].Authn)
	assert.Equal(t, tf.ActiveIDP.Users[userID].Profile["name"], "new name")
	assert.Equal(t, tf.FollowerIDP.Users[userID].Profile["name"], "old name")
}
