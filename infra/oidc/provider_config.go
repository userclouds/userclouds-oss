package oidc

import (
	"context"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucerr"
)

// ProviderConfig represents the configuration for an OIDC provider
type ProviderConfig struct {
	Type                    ProviderType  `json:"type" yaml:"type"`
	Name                    string        `json:"name" yaml:"name" validate:"notempty"`
	Description             string        `json:"description" yaml:"description" validate:"notempty"`
	IssuerURL               string        `json:"issuer_url" yaml:"issuer_url" validate:"notempty"`
	ClientID                string        `json:"client_id" yaml:"client_id"`
	ClientSecret            secret.String `json:"client_secret" yaml:"client_secret"`
	CanUseLocalHostRedirect bool          `json:"can_use_local_host_redirect" yaml:"can_use_local_host_redirect"`
	UseLocalHostRedirect    bool          `json:"use_local_host_redirect" yaml:"use_local_host_redirect"`
	DefaultScopes           string        `json:"default_scopes" yaml:"default_scopes" validate:"notempty"`
	AdditionalScopes        string        `json:"additional_scopes" yaml:"additional_scopes"`
	IsNative                bool          `json:"is_native" yaml:"is_native"`
}

func (pc *ProviderConfig) extraValidate() error {
	p, err := GetProvider(pc)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if err := pc.validateProvider(p); err != nil {
		return ucerr.Wrap(err)
	}

	if pc.Name == ProviderTypeNone.String() {
		return ucerr.Errorf("cannot use reserved name '%s'", pc.Name)
	}

	if pc.UseLocalHostRedirect {
		if !pc.CanUseLocalHostRedirect {
			return ucerr.New("cannot use local host redirect")
		}

		if u := universe.Current(); !u.IsDev() {
			return ucerr.Errorf("cannot use local host redirect in universe '%v'", u)
		}
	}

	return nil
}

func (pc *ProviderConfig) validateProvider(p Provider) error {
	defaultConfig := p.GetDefaultSettings()
	if pc.Type != defaultConfig.Type {
		return ucerr.Errorf("invalid type '%s'", pc.Type)
	}

	if defaultConfig.Name != "" {
		if pc.Name != defaultConfig.Name {
			return ucerr.Errorf("invalid name '%s'", pc.Name)
		}
	}

	if defaultConfig.Description != "" {
		if pc.Description != defaultConfig.Description {
			return ucerr.Errorf("invalid description '%s'", pc.Description)
		}
	}

	if defaultConfig.IssuerURL != "" {
		if pc.IssuerURL != defaultConfig.IssuerURL {
			return ucerr.Errorf("invalid issuer URL '%s'", pc.IssuerURL)
		}
	}

	if pc.CanUseLocalHostRedirect != defaultConfig.CanUseLocalHostRedirect {
		return ucerr.Errorf("can use local host redirect must be '%v'", defaultConfig.CanUseLocalHostRedirect)
	}

	if pc.IsNative != defaultConfig.IsNative {
		return ucerr.Errorf("is native cannot be '%v'", pc.IsNative)
	}

	if defaultConfig.DefaultScopes != "" {
		if pc.DefaultScopes != defaultConfig.DefaultScopes {
			return ucerr.Errorf("invalid default scopes '%s'", pc.DefaultScopes)
		}
	}

	if err := p.ValidateAdditionalScopes(); err != nil {
		return ucerr.Wrap(err)
	}

	if err := p.ValidateCombinedScopes(); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

//go:generate genvalidate ProviderConfig

// DecodeSecrets replaces secret values with placeholders for UIs
func (pc *ProviderConfig) DecodeSecrets(ctx context.Context) error {
	s, err := pc.ClientSecret.ResolveInsecurelyForUI(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}
	pc.ClientSecret = *s

	return nil
}

// EncodeSecrets encodes secret values on the basis of values from the UI
func (pc *ProviderConfig) EncodeSecrets(ctx context.Context, source []ProviderConfig) error {
	// easy case first
	if pc.ClientSecret == secret.EmptyString {
		return nil
	}

	// no changes, or source is nil so the whole provider is new
	if pc.ClientSecret == secret.UIPlaceholder {
		for _, sourceProvider := range source {
			// TODO (sgarrity 7/24): I'm not convinced type is the right comparison,
			// why don't these have IDs?
			if pc.Type == sourceProvider.Type {
				pc.ClientSecret = sourceProvider.ClientSecret
				return nil
			}
		}
	}

	// must be a new secret, or a new app (we didn't find the app ID in the source above)
	sec, err := pc.ClientSecret.MarshalText()
	if err != nil {
		return ucerr.Wrap(err)
	}

	ns, err := crypto.CreateClientSecret(ctx, pc.Type.String(), string(sec))
	if err != nil {
		return ucerr.Wrap(err)
	}
	pc.ClientSecret = *ns
	return nil
}

// GetDefaultNativeProviders returns the default settings for all native providers
func GetDefaultNativeProviders() []ProviderConfig {
	return []ProviderConfig{defaultFacebookProviderConfig, defaultGoogleProviderConfig, defaultLinkedInProviderConfig, defaultMicrosoftProviderConfig}
}

// GetDefaultCustomProvider returns the default settings for a custom provider
func GetDefaultCustomProvider() ProviderConfig {
	return defaultCustomProviderConfig
}
