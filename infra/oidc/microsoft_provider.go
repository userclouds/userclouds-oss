package oidc

import (
	"context"
	"regexp"

	gooidc "github.com/coreos/go-oidc/v3/oidc"

	"userclouds.com/infra/ucerr"
)

// MicrosoftProvider defines configuration for a Microsoft Live/Office365/Azure/ActiveDirectory/MSN/Clippy OIDC client and implements the Provider interface
type MicrosoftProvider struct {
	baseProvider
}

// https://learn.microsoft.com/en-us/entra/identity-platform/howto-convert-app-to-be-multi-tenant

var defaultMicrosoftProviderConfig = ProviderConfig{
	Type:                    ProviderTypeMicrosoft,
	Name:                    ProviderTypeMicrosoft.String(),
	Description:             "Microsoft",
	IssuerURL:               MicrosoftIssuerURL,
	CanUseLocalHostRedirect: false,
	DefaultScopes:           DefaultScopes,
	IsNative:                true,
}

// MicrosoftIssuerURL is the default issuer URL for Microsoft
const MicrosoftIssuerURL = "https://login.microsoftonline.com/common/v2.0"

// CreateAuthenticator is part of the Provider interface and creates a custom autheticator
func (p MicrosoftProvider) CreateAuthenticator(redirectURL string) (*Authenticator, error) {
	scopes := p.getCombinedScopes()

	// this piece of heaven brought to you by "embrace and extend", episode 207,
	// wherein Microsoft decided that OIDC standards compliance was for wimps :)
	// https://github.com/MicrosoftDocs/azure-docs/issues/38427
	overrideCtx := gooidc.InsecureIssuerURLContext(context.Background(), "https://sts.windows.net/{tenantid}/")
	authr, err := newAuthenticatorViaDiscovery(overrideCtx, p.GetIssuerURL(), p.config.ClientID, p.config.ClientSecret, redirectURL, scopes)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// override the issuer validation for the same reason as above with InsecureIssuerURLContext
	authr.OverrideValidateIssuer = validateIssuer
	return authr, nil
}

// GetDefaultSettings is part of the Provider interface and returns the default settings for a custom provider
func (MicrosoftProvider) GetDefaultSettings() ProviderConfig {
	return defaultMicrosoftProviderConfig
}

// ValidateAdditionalScopes is part of the Provider interface and validates the additional scopes
func (p MicrosoftProvider) ValidateAdditionalScopes() error {
	if p.config.AdditionalScopes != "" {
		return ucerr.New("additional scopes are not supported for Microsoft yet")
	}
	return nil
}

var microsoftIssuerRE = regexp.MustCompile(`^https://login\.microsoftonline\.com/([a-zA-Z0-9-]+)/v2.0$`)

// ValidateIssuer is part of the Provider interface and validates the issuer in cases
// where the library can't do it automatically (eg. MSFT)
// https://github.com/MicrosoftDocs/azure-docs/issues/38427
func validateIssuer(iss string) error {
	if !microsoftIssuerRE.MatchString(iss) {
		return ucerr.Errorf("invalid issuer '%s'", iss)
	}

	return nil
}
