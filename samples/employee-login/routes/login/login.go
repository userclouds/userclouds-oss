package login

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"

	"github.com/userclouds/userclouds/samples/employee-login/app"
	"github.com/userclouds/userclouds/samples/employee-login/auth"

	"userclouds.com/infra/uchttp"
)

// Handler handles login
func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Generate random state
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}
	state := base64.StdEncoding.EncodeToString(b)

	session, _ := app.GetAuthSession(r)
	// An error here likely means a bad cookie; treat as unauthenticated
	// and move on.
	//if err != nil {
	//uchttp.Error(ctx, w, err, http.StatusInternalServerError)
	//return
	//}

	session.Values["state"] = state
	err = session.Save(r, w)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	authenticator, err := auth.NewAuthenticator()
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	authCodeURL := authenticator.Config.AuthCodeURL(state)
	log.Printf("Redirecting to auth code URL: %s", authCodeURL)
	http.Redirect(w, r, authCodeURL, http.StatusTemporaryRedirect)
}
