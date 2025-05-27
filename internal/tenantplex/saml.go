package tenantplex

import (
	"userclouds.com/infra/secret"
	"userclouds.com/internal/tenantplex/samlconfig"
)

// SAMLIDP represents the config for any Login App (fka Plex App) that can serve
// as a SAML IDP
type SAMLIDP struct {
	Certificate string        `json:"certificate" validate:"notempty"`
	PrivateKey  secret.String `json:"private_key"`

	MetadataURL string `json:"metadata_url" validate:"notempty"`
	SSOURL      string `json:"sso_url" validate:"notempty"`

	TrustedServiceProviders []samlconfig.EntityDescriptor `json:"trusted_service_providers,omitempty" validate:"skip"`
}

//go:generate genvalidate SAMLIDP
