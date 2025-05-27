package auth

import (
	"context"
	"log"
	"net/url"
	"os"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Authenticator abstracts different types of Authenticators
type Authenticator struct {
	Provider *oidc.Provider
	Config   oauth2.Config
}

// Const string enum for which IDP to use in the sample employee login app.
const (
	ModePlex = "Plex"
)

// GetMode returns one of the defined modes to indicate which IDP the
// sample employee login app is using.
func GetMode() string {
	return ModePlex
}

// NewAuthenticator returns an Authenticator based on env vars
func NewAuthenticator() (*Authenticator, error) {
	return newUserCloudsAuthenticator()
}

// GetLogoutURL returns the (mode-dependent) URL to redirect to in order to log the user out.
func GetLogoutURL(clientID, redirectURL string) string {
	query := url.Values{}
	query.Add("client_id", clientID)
	query.Add("redirect_url", redirectURL)
	return os.Getenv("UC_TENANT_BASE_URL") + "/logout?" + query.Encode()
}

// GetClientID returns the (mode-dependent) client ID for the IDP.
func GetClientID() string {
	return os.Getenv("PLEX_CLIENT_ID")
}

func newUserCloudsAuthenticator() (*Authenticator, error) {
	provider, err := oidc.NewProvider(context.TODO(), os.Getenv("UC_TENANT_BASE_URL"))
	if err != nil {
		log.Printf("failed to get provider: %v", err)
		return nil, err
	}

	conf := oauth2.Config{
		ClientID:     os.Getenv("PLEX_CLIENT_ID"),
		ClientSecret: os.Getenv("PLEX_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("OIDC_CALLBACK_URL"),
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return &Authenticator{
		Provider: provider,
		Config:   conf,
	}, nil
}
