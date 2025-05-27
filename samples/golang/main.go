package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Sample code showing the simplest possible (or close) OIDC login flow with golang
// This sample explicitly has a hard-coded login endpoint and shared public credentials
// to make getting up and running as simple as possible, but obviously you should never
// do this with your own credentials. These are locked down to do nothing other than
// allow login to this sample tenant.

// The code style is also intentionally simplified to make this one (short) page
// If you're adapting this, get rid of the globals, the hardcoded strings, and the like,
// and make sure to actually generate, store, and check the state on a per-request basis.

func getCfg(ctx context.Context) (*oauth2.Config, *oidc.Provider) {
	var prov, err = oidc.NewProvider(ctx, "https://sample.tenant.userclouds.com")
	if err != nil {
		panic(err)
	}

	return &oauth2.Config{
		ClientID:     "5f107e226353791560f93164a09f7e0f",
		ClientSecret: "2ftQe4RU7aR/iStpcFf3gfiUjnsbWGFY0C9aWkPzDqT14eyp23ysKuI6iBbOAW/O",
		Endpoint:     prov.Endpoint(),
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"openid"},
	}, prov
}

func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "<html>Hello, world. <a href=\"/login\">Sign in</a></html>")
}

func login(w http.ResponseWriter, r *http.Request) {
	cfg, _ := getCfg(r.Context())
	http.Redirect(w, r, cfg.AuthCodeURL("GENERATE_ME"), http.StatusTemporaryRedirect)
}

func callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cfg, prov := getCfg(ctx)

	// confirm state matches
	if state := r.URL.Query().Get("state"); state != "GENERATE_ME" {
		http.Error(w, "mismatched state in OIDC login", http.StatusBadRequest)
		return
	}

	// exchange code for a token
	code := r.URL.Query().Get("code")
	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// validate the token
	idt := token.Extra("id_token").(string)
	idToken, err := prov.Verifier(&oidc.Config{ClientID: cfg.ClientID}).Verify(ctx, idt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "logged in as %s\n\n", idToken.Subject)

	// display all the claims in the token
	claims := make(map[string]any)
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for k, v := range claims {
		fmt.Fprintf(w, "%s: %v\n", k, v)
	}
}

const hostAndPort = "localhost:8080"

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", home)
	mux.HandleFunc("/login", login)
	mux.HandleFunc("/callback", callback)

	fmt.Printf("listening on http://%s\n", hostAndPort)
	log.Fatal(http.ListenAndServe(hostAndPort, mux))
}
