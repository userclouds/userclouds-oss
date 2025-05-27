package oidc

import (
	"context"
)

// CustomProvider defines configuration for a custom OIDC client and implements the Provider interface
type CustomProvider struct {
	baseProvider
}

var defaultCustomProviderConfig = ProviderConfig{
	Type:                    ProviderTypeCustom,
	CanUseLocalHostRedirect: false,
	DefaultScopes:           DefaultScopes,
	IsNative:                false,
}

// CreateAuthenticator is part of the Provider interface and creates a custom autheticator
func (p CustomProvider) CreateAuthenticator(redirectURL string) (*Authenticator, error) {
	scopes := p.getCombinedScopes()
	return newAuthenticatorViaDiscovery(context.Background(), p.GetIssuerURL(), p.config.ClientID, p.config.ClientSecret, redirectURL, scopes)
}

// GetDefaultSettings is part of the Provider interface and returns the default settings for a custom provider
func (CustomProvider) GetDefaultSettings() ProviderConfig {
	return defaultCustomProviderConfig
}

// ValidateAdditionalScopes is part of the Provider interface and validates the additional scopes
func (p CustomProvider) ValidateAdditionalScopes() error {
	return nil
}
