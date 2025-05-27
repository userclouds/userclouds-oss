package oidc

import (
	"fmt"

	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucerr"
)

type baseProvider struct {
	config *ProviderConfig
}

func (p baseProvider) getCombinedScopes() []string {
	return append(SplitTokens(p.config.DefaultScopes), SplitTokens(p.config.AdditionalScopes)...)
}

// GetAdditionalScopes is part of the Provider interface and returns the additional scopes
func (p baseProvider) GetAdditionalScopes() string {
	return p.config.AdditionalScopes
}

// GetDescription is part of the Provider interface
func (p baseProvider) GetDescription() string {
	return p.config.Description
}

// GetIssuerURL is part of the Provider interface
func (p baseProvider) GetIssuerURL() string {
	return p.config.IssuerURL
}

// GetLoginButtonDescription is part of the Provider interface
func (p baseProvider) GetLoginButtonDescription() string {
	return fmt.Sprintf("Sign in with %s", p.GetDescription())
}

// GetMergeButtonDescription is part of the Provider interface
func (p baseProvider) GetMergeButtonDescription() string {
	return fmt.Sprintf("Sign in to existing account with %s", p.GetDescription())
}

// GetName is part of the Provider interface
func (p baseProvider) GetName() string {
	return p.config.Name
}

// GetType is part of the Provider interface
func (p baseProvider) GetType() ProviderType {
	if err := p.config.Type.Validate(); err != nil {
		return ProviderTypeUnsupported
	}
	return p.config.Type
}

// IsConfigured is part of the Provider interface
func (p baseProvider) IsConfigured() bool {
	if err := p.config.Validate(); err != nil {
		return false
	}

	return p.config.ClientID != "" && p.config.ClientSecret != secret.String{}
}

// IsNative is part of the Provider interface
func (p baseProvider) IsNative() bool {
	return p.config.IsNative
}

// UseLocalHostRedirect is part of the Provider interface
func (p baseProvider) UseLocalHostRedirect() bool {
	return p.config.UseLocalHostRedirect
}

// ValidateCombinedScopes is part of the Provider interface and ensures the combined scopes are valid
func (p baseProvider) ValidateCombinedScopes() error {
	uniqueScopes := map[string]bool{}
	for _, scope := range p.getCombinedScopes() {
		if uniqueScopes[scope] {
			return ucerr.Friendlyf(nil, "scope '%s' already exists", scope)
		}
		uniqueScopes[scope] = true
	}

	return nil
}
