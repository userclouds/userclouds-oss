package test

import (
	"context"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/oidc"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider/iface"
)

// ProviderFactory is a test provider factory
type ProviderFactory struct{}

// NewClient is a test client factory
func (ProviderFactory) NewClient(ctx context.Context, p tenantplex.Provider, plexClientID string, providerAppID uuid.UUID) (iface.Client, error) {
	return nil, nil
}

// NewManagementClient is a test management client factory
func (ProviderFactory) NewManagementClient(ctx context.Context, tc *tenantplex.TenantConfig, p tenantplex.Provider, appID uuid.UUID, appOrgID uuid.UUID) (iface.ManagementClient, error) {
	return nil, nil
}

// NewOIDCAuthenticator is a test OIDC authenticator factory
func (ProviderFactory) NewOIDCAuthenticator(ctx context.Context, pt oidc.ProviderType, issuerURL string, cfg tenantplex.OIDCProviders, redirectURL *url.URL) (*oidc.Authenticator, error) {
	return nil, nil
}
