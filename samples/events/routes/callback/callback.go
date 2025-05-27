package callback

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofrs/uuid"
	"github.com/userclouds/userclouds/samples/events/app"
	"github.com/userclouds/userclouds/samples/events/auth"
	"golang.org/x/oauth2"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
)

func validateUserInfoClaims(claims map[string]any) error {
	_, ok := claims["sub"].(string)
	if !ok {
		return fmt.Errorf("failed to get 'sub' from user info claims profile")
	}
	_, ok = claims["name"].(string)
	if !ok {
		claims["name"] = "<no name>"
	}
	return nil
}

func upsertUser(ctx context.Context, subject, name string) (string, error) {
	userID, err := uuid.FromString(subject)
	if err != nil {
		return "", err
	}
	// Get the RBAC user for a username and store the ID in our own user table.
	// TODO: Long term we probably don't need an Events app user table at all.
	user, err := app.GetAuthZClient().GetUser(ctx, userID)
	if err != nil {
		return "", err
	}

	if err := app.GetStorage().SaveUser(ctx, app.User{
		ID:   user.ID,
		Name: name}); err != nil {
		return "", err
	}
	return userID.String(), nil
}

// Handler handles the callback
func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	session, err := app.GetAuthSession(r)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}
	log.Printf("Callback method: %s", r.Method)
	log.Printf("URL from %s: %s", auth.GetMode(), r.URL)

	if r.URL.Query().Get("state") != session.Values["state"] {
		uchttp.Error(ctx, w, ucerr.New("Invalid state parameter"), http.StatusBadRequest)
		return
	}

	authenticator, err := auth.NewAuthenticator()
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	log.Printf("Code from %s: %s", auth.GetMode(), r.URL.Query().Get("code"))
	token, err := authenticator.Config.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("no token found: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		uchttp.Error(ctx, w, ucerr.New("No id_token field in oauth2 token."), http.StatusInternalServerError)
		return
	}

	log.Printf("raw ID token from %s: %s", auth.GetMode(), rawIDToken)

	oidcConfig := &oidc.Config{
		ClientID: authenticator.Config.ClientID,
	}

	idToken, err := authenticator.Provider.Verifier(oidcConfig).Verify(ctx, rawIDToken)
	if err != nil {
		altAuthenticator, err2 := auth.NewAltAuthenticator()
		if err2 != nil {
			uchttp.Error(ctx, w, ucerr.Errorf("Failed to verify ID Token: %w", err2), http.StatusInternalServerError)
			return
		}
		oidcConfig = &oidc.Config{
			ClientID: altAuthenticator.Config.ClientID,
		}
		idToken, err2 = altAuthenticator.Provider.Verifier(oidcConfig).Verify(ctx, rawIDToken)
		if err2 != nil {
			uchttp.Error(ctx, w, ucerr.Errorf("Failed to verify ID Token: [%v, %w]", err, err2), http.StatusInternalServerError)
			return
		}
	}

	tokenStr, _ := json.Marshal(idToken)
	log.Printf("Decoded id_token from %s: %s", auth.GetMode(), tokenStr)
	log.Printf("Access token from %s: %s", auth.GetMode(), token.AccessToken)

	// Getting now the userInfo
	var profile map[string]any
	if err := idToken.Claims(&profile); err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	// TODO: (#75) This isn't great - we should proxy this call through Plex
	// so the caller doesn't need to. And ideally we hit the correct endpoint
	// instead of having to try both.
	userInfo, err := authenticator.Provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		altAuthenticator, err2 := auth.NewAltAuthenticator()
		if err2 != nil {
			uchttp.Error(ctx, w, ucerr.Errorf("Failed to get userinfo: %w", err2), http.StatusInternalServerError)
			return
		}
		userInfo, err = altAuthenticator.Provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
		if err != nil {
			uchttp.Error(ctx, w, ucerr.Errorf("Failed to get userinfo: %w", err), http.StatusInternalServerError)
			return
		}
	}

	var userInfoProfile map[string]any
	err = userInfo.Claims(&userInfoProfile)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("Failed to get claims from userinfo: %w", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Userinfo: %+v (claims: %+v)", userInfo, userInfoProfile)

	err = validateUserInfoClaims(userInfoProfile)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("Failed to validate claims from userinfo: %w", err), http.StatusInternalServerError)
		return
	}

	// Create or update user, and get an app-specific "user_id"
	// to associate with the session.
	session.Values["user_id"], err = upsertUser(
		ctx, userInfoProfile["sub"].(string), userInfoProfile["name"].(string))
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	log.Printf("ID Token from %s: %s", auth.GetMode(), rawIDToken)
	log.Printf("Access Token from %s: %s", auth.GetMode(), token.AccessToken)
	log.Printf("Claims from token from %s: %s", auth.GetMode(), profile)

	session.Values["profile"] = profile

	// Since this uses `securecookie`` to sign the cookie, we can just store the profile and User ID.
	err = session.Save(r, w)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	// Redirect to logged in page
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}
