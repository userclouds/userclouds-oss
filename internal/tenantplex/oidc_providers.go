package tenantplex

import (
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
)

// OIDCProviders represents the collection of OIDC providers for a tenant
type OIDCProviders struct {
	Providers []oidc.ProviderConfig `yaml:"providers" json:"providers"`
}

func (op *OIDCProviders) extraValidate() error {
	totalNativeProviders := 0
	uniqueNames := map[string]bool{}
	uniqueIssuerURLs := map[string]bool{}

	for _, p := range op.Providers {
		if uniqueNames[p.Name] {
			return ucerr.Friendlyf(nil, "OIDC provider name '%s' is not unique", p.Name)
		}
		uniqueNames[p.Name] = true

		if uniqueIssuerURLs[p.IssuerURL] {
			return ucerr.Friendlyf(nil, "OIDC provider issuer URL '%s' is not unique", p.IssuerURL)
		}
		uniqueIssuerURLs[p.IssuerURL] = true

		if p.IsNative {
			totalNativeProviders++
		}
	}

	if totalNativeProviders != len(oidc.NativeOIDCProviderTypes()) {
		return ucerr.Errorf("total number of native providers %d does not match expected %d", totalNativeProviders, len(oidc.NativeOIDCProviderTypes()))
	}

	return nil
}

//go:generate genvalidate OIDCProviders

// GetProviderForName returns the OIDC provider that matches the specified name
func (op OIDCProviders) GetProviderForName(name string) (oidc.Provider, error) {
	for _, p := range op.Providers {
		if p.Name == name {
			return oidc.GetProvider(&p)
		}
	}

	return nil, ucerr.Errorf("could not find OIDC provider for name '%s'", name)
}

// GetProviderForIssuerURL returns the OIDC provider that matches the specified type and issuer URL
func (op OIDCProviders) GetProviderForIssuerURL(provider oidc.ProviderType, issuerURL string) (oidc.Provider, error) {
	for _, p := range op.Providers {
		if p.Type == provider && p.IssuerURL == issuerURL {
			return oidc.GetProvider(&p)
		}
	}

	return nil, ucerr.Errorf("could not find OIDC provider for Type '%v' and IssuerURL '%s'", provider, issuerURL)
}
