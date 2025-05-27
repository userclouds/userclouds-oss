package oidc

import (
	"context"
	"net/http"
	"strings"

	"golang.org/x/oauth2"

	"userclouds.com/infra/ucerr"
)

// GoogleProvider defines configuration for a google OIDC client and implements the Provider interface
type GoogleProvider struct {
	baseProvider
}

var defaultGoogleProviderConfig = ProviderConfig{
	Type:                    ProviderTypeGoogle,
	Name:                    ProviderTypeGoogle.String(),
	Description:             "Google",
	IssuerURL:               googleIssuerURL,
	CanUseLocalHostRedirect: false,
	DefaultScopes:           DefaultScopes,
	IsNative:                true,
}

const googleIssuerURL = "https://accounts.google.com"

// CreateAuthenticator is part of the Provider interface and creates an authenticator for google
func (p GoogleProvider) CreateAuthenticator(redirectURL string) (*Authenticator, error) {
	scopes := p.getCombinedScopes()
	a, err := newAuthenticatorViaDiscovery(context.Background(), p.GetIssuerURL(), p.config.ClientID, p.config.ClientSecret, redirectURL, scopes)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// google accepts a "login_hint" parameter with an email address to choose the
	// correct account to log into
	a.AuthCodeOptionGetter = func(r *http.Request) []oauth2.AuthCodeOption {
		if r.URL.Query().Has("email") {
			email := r.URL.Query().Get("email")
			return []oauth2.AuthCodeOption{oauth2.SetAuthURLParam("login_hint", email)}
		}

		return nil
	}
	return a, nil
}

// GetDefaultSettings is part of the Provider interface and returns the default settings for a google provider
func (GoogleProvider) GetDefaultSettings() ProviderConfig {
	return defaultGoogleProviderConfig
}

// ValidateAdditionalScopes is part of the Provider interface and validates the additional scopes
func (p GoogleProvider) ValidateAdditionalScopes() error {
	for _, scope := range SplitTokens(p.GetAdditionalScopes()) {
		if !strings.HasPrefix(scope, "https://www.googleapis.com/auth/") {
			return ucerr.Friendlyf(nil, "\"%v\" is not a valid Google oauth scope", scope)
		}
	}

	return nil
}
