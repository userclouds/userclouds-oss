package oidc

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"

	"userclouds.com/infra/ucerr"
)

const linkedInDiscoveryURL = "https://www.linkedin.com/oauth"
const linkedInIssuerURL = "https://www.linkedin.com"

// LinkedInProvider defines configuration for a linkedin OIDC client and implements the Provider interface
type LinkedInProvider struct {
	baseProvider
}

var defaultLinkedInProviderConfig = ProviderConfig{
	Type:                    ProviderTypeLinkedIn,
	Name:                    ProviderTypeLinkedIn.String(),
	Description:             "LinkedIn",
	IssuerURL:               linkedInIssuerURL,
	CanUseLocalHostRedirect: false,
	DefaultScopes:           DefaultScopes,
	IsNative:                true,
}

// CreateAuthenticator is part of the Provider interface and creates an authenticator for linkedin
func (p LinkedInProvider) CreateAuthenticator(redirectURL string) (*Authenticator, error) {
	ctx := oidc.InsecureIssuerURLContext(context.Background(), p.GetIssuerURL())
	scopes := p.getCombinedScopes()
	return newAuthenticatorViaDiscovery(ctx, linkedInDiscoveryURL, p.config.ClientID, p.config.ClientSecret, redirectURL, scopes)
}

// GetDefaultSettings is part of the Provider interface and returns the default configuration for a linkedin provider
func (LinkedInProvider) GetDefaultSettings() ProviderConfig {
	return defaultLinkedInProviderConfig
}

// ValidateAdditionalScopes is part of the Provider interface and validates the additional scopes
func (p LinkedInProvider) ValidateAdditionalScopes() error {
	for _, scope := range SplitTokens(p.GetAdditionalScopes()) {
		if _, exists := validLinkedInScopes[scope]; !exists {
			return ucerr.Friendlyf(nil, "'%v' is not a valid LinkedIn oauth scope", scope)
		}
	}

	return nil
}

// additional scopes grabbed from https://www.linkedin.com/developers/apps/<app_id>/products/*/endpoints
var validLinkedInScopes = map[string]bool{
	"email":                    true,
	"profile":                  true,
	oidc.ScopeOpenID:           true,
	"r_1st_connections_size":   true,
	"r_ads":                    true,
	"r_ads_leadgen_automation": true,
	"r_ads_reporting":          true,
	"r_basicprofile":           true,
	"r_emailaddress":           true,
	"r_liteprofile":            true,
	"r_member_live":            true,
	"r_organization_admin":     true,
	"r_organization_live":      true,
	"r_organization_social":    true,
	"rw_ads":                   true,
	"rw_organization_admin":    true,
	"w_member_live":            true,
	"w_member_social":          true,
	"w_organization_live":      true,
	"w_organization_social":    true,
}
