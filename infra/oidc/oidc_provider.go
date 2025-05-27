package oidc

import "userclouds.com/infra/ucerr"

// Provider defines the backend interface for an OIDC provider
type Provider interface {
	GetType() ProviderType
	GetName() string
	GetDescription() string
	GetIssuerURL() string
	GetLoginButtonDescription() string
	GetMergeButtonDescription() string
	GetDefaultSettings() ProviderConfig
	IsConfigured() bool
	IsNative() bool
	ValidateAdditionalScopes() error
	ValidateCombinedScopes() error

	CreateAuthenticator(redirectURL string) (*Authenticator, error)
	UseLocalHostRedirect() bool
}

// GetProvider returns the appropriate OIDC Provider implementation for the provider config
func GetProvider(pc *ProviderConfig) (Provider, error) {
	if pc == nil {
		return nil, ucerr.New("ProviderConfig cannot be nil")
	}

	switch pc.Type {
	case ProviderTypeCustom:
		return CustomProvider{baseProvider{config: pc}}, nil
	case ProviderTypeGoogle:
		return GoogleProvider{baseProvider{config: pc}}, nil
	case ProviderTypeFacebook:
		return FacebookProvider{baseProvider{config: pc}}, nil
	case ProviderTypeLinkedIn:
		return LinkedInProvider{baseProvider{config: pc}}, nil
	case ProviderTypeMicrosoft:
		return MicrosoftProvider{baseProvider{config: pc}}, nil
	default:
		return nil, ucerr.Errorf("ProviderConfig type is unsupported: '%v'", pc.Type)
	}
}
