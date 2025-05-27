package builder

import (
	"fmt"

	"userclouds.com/infra/oidc"
	"userclouds.com/infra/secret"
)

// OIDCProviderBuilder introduces OIDCProvider-specific building methods
type OIDCProviderBuilder struct {
	TenantConfigBuilder
}

// AddOIDCProvider adds an OIDCProvider to the underlying TenantConfig, returning an OIDCProviderBuilder
// so other OIDCProvider-specific changes can be made
func (tcb *TenantConfigBuilder) AddOIDCProvider(name string) *OIDCProviderBuilder {
	tcb.tenantConfig.OIDCProviders.Providers =
		append(tcb.tenantConfig.OIDCProviders.Providers,
			oidc.GetDefaultCustomProvider())
	tcb.currOIDCProvider = &tcb.tenantConfig.OIDCProviders.Providers[len(tcb.tenantConfig.OIDCProviders.Providers)-1]
	tcb.currOIDCProvider.Name = name
	return &OIDCProviderBuilder{*tcb}
}

// SwitchToOIDCProvider switches to the OIDCProvider with the specified name. A panic will be issued
// if the name cannot be found.
func (tcb *TenantConfigBuilder) SwitchToOIDCProvider(name string) *OIDCProviderBuilder {
	for i := range tcb.tenantConfig.OIDCProviders.Providers {
		if tcb.tenantConfig.OIDCProviders.Providers[i].Name == name {
			tcb.currOIDCProvider = &tcb.tenantConfig.OIDCProviders.Providers[i]
			return &OIDCProviderBuilder{*tcb}
		}
	}

	// this would be a programmer error - ok to panic here as this
	// will be discovered during test development
	panic(fmt.Sprintf("unrecognized OIDCProvider name '%s'", name))
}

// SetAdditionalScopes sets additional scopes for the OIDC provider
func (opb *OIDCProviderBuilder) SetAdditionalScopes(scopes string) *OIDCProviderBuilder {
	opb.currOIDCProvider.AdditionalScopes = scopes
	return opb
}

// SetClientID sets the client ID for the OIDC provider
func (opb *OIDCProviderBuilder) SetClientID(clientID string) *OIDCProviderBuilder {
	opb.currOIDCProvider.ClientID = clientID
	return opb
}

// SetClientSecret sets the client secret for the OIDC provider
func (opb *OIDCProviderBuilder) SetClientSecret(clientSecret secret.String) *OIDCProviderBuilder {
	opb.currOIDCProvider.ClientSecret = clientSecret
	return opb
}

// SetDescription sets the description of the OIDC provider
func (opb *OIDCProviderBuilder) SetDescription(description string) *OIDCProviderBuilder {
	opb.currOIDCProvider.Description = description
	return opb
}

// SetIssuerURL sets the issuer URL for the OIDC provider
func (opb *OIDCProviderBuilder) SetIssuerURL(issuerURL string) *OIDCProviderBuilder {
	opb.currOIDCProvider.IssuerURL = issuerURL
	return opb
}

// SetUseLocalHostRedirect sets the flag for the OIDC provider
func (opb *OIDCProviderBuilder) SetUseLocalHostRedirect(ulhr bool) *OIDCProviderBuilder {
	opb.currOIDCProvider.UseLocalHostRedirect = ulhr
	return opb
}
