package provider

import (
	"context"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/oidc"
	"userclouds.com/infra/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/messaging/email"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider/auth0"
	"userclouds.com/plex/internal/provider/cognito"
	"userclouds.com/plex/internal/provider/employee"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/provider/uc"
	"userclouds.com/plex/internal/tenantconfig"
)

// NewActiveClient returns a client for the active provider for a given
// Plex ClientID.
func NewActiveClient(ctx context.Context, f Factory, plexClientID string) (iface.Client, error) {
	pm := tenantconfig.MustGetPlexMap(ctx)
	_, policy, err := pm.FindAppForClientID(plexClientID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return NewClientForProviderID(ctx, f, plexClientID, policy.ActiveProviderID)
}

// NewFollowerClients returns an array of auth provider clients which should be
// updated when credentials, user profile data, etc. changes on the primary provider.
// For now, this just returns all inactive clients.
func NewFollowerClients(ctx context.Context, f Factory, plexClientID string) ([]iface.Client, error) {
	pm := tenantconfig.MustGetPlexMap(ctx)
	app, policy, err := pm.FindAppForClientID(plexClientID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	clients := []iface.Client{}
	for _, providerAppID := range app.ProviderAppIDs {
		provider, err := pm.FindProviderForAppID(providerAppID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		if provider.ID == policy.ActiveProviderID {
			// Skip the active provider
			continue
		}

		client, err := f.NewClient(ctx, *provider, plexClientID, providerAppID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		clients = append(clients, client)
	}

	return clients, nil
}

// NewClientForProviderID returns the client for a given Plex ClientID and Provider ID.
func NewClientForProviderID(ctx context.Context, f Factory, plexClientID string, providerID uuid.UUID) (iface.Client, error) {
	pm := tenantconfig.MustGetPlexMap(ctx)
	app, _, err := pm.FindAppForClientID(plexClientID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	for _, providerAppID := range app.ProviderAppIDs {
		provider, err := pm.FindProviderForAppID(providerAppID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		if provider.ID == providerID {
			client, err := f.NewClient(ctx, *provider, plexClientID, providerAppID)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			return client, nil
		}
	}

	return nil, ucerr.Errorf("no active provider app for provider ID: %v and plex client ID: %s", providerID, plexClientID)
}

// Factory is an interface for objects that can create provider clients
// from config. It primarily exists to allow for clean mocking/substitution,
// and production use cases always use the ProdFactory.
type Factory interface {
	NewClient(ctx context.Context, p tenantplex.Provider, plexClientID string, providerAppID uuid.UUID) (iface.Client, error)
	NewManagementClient(ctx context.Context, tc *tenantplex.TenantConfig, p tenantplex.Provider, appID uuid.UUID, appOrgID uuid.UUID) (iface.ManagementClient, error)
	NewOIDCAuthenticator(ctx context.Context, pt oidc.ProviderType, issuerURL string, cfg tenantplex.OIDCProviders, redirectURL *url.URL) (*oidc.Authenticator, error)
}

// ProdFactory is the normal, production-capable factory used to create
// IDP clients for Plex to talk to.
type ProdFactory struct {
	EmailClient *email.Client
	ConsoleEP   *service.Endpoint
}

// NewClient creates the appropriate type of IDP client based on the given config.
func (pf ProdFactory) NewClient(ctx context.Context, provider tenantplex.Provider, plexClientID string, providerAppID uuid.UUID) (iface.Client, error) {
	if err := provider.ValidateAppID(providerAppID); err != nil {
		return nil, ucerr.Wrap(err)
	}
	switch provider.Type {
	case tenantplex.ProviderTypeAuth0:
		return auth0.NewClient(provider.ID, provider.Name, *provider.Auth0, providerAppID)
	case tenantplex.ProviderTypeEmployee:
		if pf.ConsoleEP == nil {
			return nil, ucerr.New("Console Endpoint is required for employee provider")
		}
		return employee.NewClient(ctx, provider.ID, provider.Name, *pf.ConsoleEP)
	case tenantplex.ProviderTypeUC:
		return uc.NewClient(ctx, provider.ID, provider.Name, *provider.UC, providerAppID, plexClientID, pf.EmailClient)
	case tenantplex.ProviderTypeCognito:
		if pf.EmailClient == nil {
			return nil, ucerr.New("Email client is required for cognito provider")
		}
		return cognito.NewClient(ctx, provider.ID, provider.Name, provider.Cognito, providerAppID, plexClientID, *pf.EmailClient)
	default:
		return nil, ucerr.Errorf("unrecognized provider.Type %s", provider.Type)
	}
}

// NewManagementClient creates a new management-focused (non-app-specific) client for a provider
func (ProdFactory) NewManagementClient(ctx context.Context, tc *tenantplex.TenantConfig, p tenantplex.Provider, appID uuid.UUID, appOrgID uuid.UUID) (iface.ManagementClient, error) {
	// NB: requires tenantconfig middleware to be used here
	// tc := tenantconfig.MustGet(ctx)

	switch p.Type {
	case tenantplex.ProviderTypeAuth0:
		return auth0.NewManagementClient(ctx, *p.Auth0)
	case tenantplex.ProviderTypeEmployee:
		return employee.NewManagementClient(ctx, p.ID, p.Name)
	case tenantplex.ProviderTypeUC:
		return uc.NewManagementClient(ctx, tc, p.ID, p.Name, *p.UC, appID, appOrgID)
	case tenantplex.ProviderTypeCognito:
		return cognito.NewManagementClient(ctx, tc, p.ID, p.Name, p.Cognito, appID, appOrgID)
	default:
		return nil, ucerr.Errorf("unrecognized provider.Type %s", p.Type)
	}
}

// NewOIDCAuthenticator creates an authenticator that can talk to a given OIDC provider
func (ProdFactory) NewOIDCAuthenticator(ctx context.Context, pt oidc.ProviderType, issuerURL string, cfg tenantplex.OIDCProviders, redirectURL *url.URL) (*oidc.Authenticator, error) {
	p, err := cfg.GetProviderForIssuerURL(pt, issuerURL)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if !p.IsConfigured() {
		return nil, ucerr.Errorf("oidc provider named '%s' is not configured", p.GetName())
	}

	authr, err := p.CreateAuthenticator(redirectURL.String())
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return authr, nil
}

// NewActiveManagementClient returns a new management client for the active provider for the client ID
func NewActiveManagementClient(ctx context.Context, f Factory, plexClientID string) (iface.ManagementClient, error) {
	tc := tenantconfig.MustGet(ctx)
	pm := tenantconfig.MustGetPlexMap(ctx)
	app, policy, err := pm.FindAppForClientID(plexClientID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	for _, providerAppID := range app.ProviderAppIDs {
		provider, err := pm.FindProviderForAppID(providerAppID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		if provider.ID == policy.ActiveProviderID {
			return f.NewManagementClient(ctx, &tc, *provider, app.ID, app.OrganizationID)
		}
	}

	return nil, ucerr.Errorf("no provider matched active provider ID '%s'", policy.ActiveProviderID)
}

// NewFollowerManagementClients returns a possibly empty array of management clients for the non-active providers
func NewFollowerManagementClients(ctx context.Context, f Factory, clientID string) ([]iface.ManagementClient, error) {
	tc := tenantconfig.MustGet(ctx)
	pm := tenantconfig.MustGetPlexMap(ctx)

	app, _, err := pm.FindAppForClientID(clientID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var clients []iface.ManagementClient
	for _, prov := range pm.Providers {
		// skip the active automaticallyx
		if prov.ID == pm.Policy.ActiveProviderID {
			continue
		}

		// are any of this provider's apps listed as a client app in the plex app? app. :)
		var found bool
		for _, provAppID := range app.ProviderAppIDs {
			if prov.Type == tenantplex.ProviderTypeAuth0 {
				for _, provApp := range prov.Auth0.Apps {
					if provAppID == provApp.ID {
						found = true
						break
					}
				}
			} else if prov.Type == tenantplex.ProviderTypeUC {
				for _, provApp := range prov.UC.Apps {
					if provAppID == provApp.ID {
						found = true
						break
					}
				}
			}
		}
		if !found {
			continue
		}

		client, err := f.NewManagementClient(ctx, &tc, prov, app.ID, app.OrganizationID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		clients = append(clients, client)
	}

	return clients, nil
}
