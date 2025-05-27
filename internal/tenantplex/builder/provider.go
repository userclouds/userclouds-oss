package builder

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/internal/tenantplex"
)

// ProviderBuilder introduces Provider-specific building methods; it can only be used
// when a client has added a Provider to a TenantConfig via the TenantConfigBuilder
type ProviderBuilder struct {
	TenantConfigBuilder
}

// Auth0ProviderBuilder introduces Auth0Provider-specific building methods; it can
// only be accessed when a client has converted a Provider that it is building into
// an Auth0Provider
type Auth0ProviderBuilder struct {
	ProviderBuilder
}

// UCProviderBuilder introduces UCProvider-specific building methods; it can
// only be accessed when a client has converted a Provider that it is building into
// a UCProvider
type UCProviderBuilder struct {
	ProviderBuilder
}

// AddProvider adds a Provider to the underlying TenantConfig, and returns a pointer
// to a ProviderBuilder builder so that other Provider-specific changes can be made
func (tcb *TenantConfigBuilder) AddProvider() *ProviderBuilder {
	tcb.plexMap.Providers =
		append(tcb.plexMap.Providers,
			tenantplex.Provider{
				ID:   (*tcb.defaults)().GenDefaultID(),
				Name: (*tcb.defaults)().GenDefaultName(),
			})
	tcb.currProvider = &tcb.plexMap.Providers[len(tcb.plexMap.Providers)-1]
	return &ProviderBuilder{*tcb}
}

// SwitchToProvider sets the current Provider to the specified index, returning a
// pointer to a ProviderBuilder. An out of bounds index will panic as this should
// be caught by the programmer during development.
func (tcb *TenantConfigBuilder) SwitchToProvider(providerNum int) *ProviderBuilder {
	if providerNum < 0 || providerNum >= len(tcb.plexMap.Providers) {
		// this would be a programmer error - ok to panic here since
		// this will be discovered during test development
		panic(fmt.Sprintf("invalid index %d into Providers of length %d",
			providerNum, len(tcb.plexMap.Providers)))
	}

	tcb.currProvider = &tcb.plexMap.Providers[providerNum]
	return &ProviderBuilder{*tcb}
}

func (pb *ProviderBuilder) addProviderAppID(id uuid.UUID) {
	*pb.providerAppIDs = append(*pb.providerAppIDs, id)
	// make sure that any previously added plex Apps are associated with this provider app id
	for i := range pb.plexMap.Apps {
		pb.plexMap.Apps[i].ProviderAppIDs = append(pb.plexMap.Apps[i].ProviderAppIDs, id)
	}
}

// MakeActive will mark the current Provider as active in the Tenant
func (pb *ProviderBuilder) MakeActive() *ProviderBuilder {
	pb.plexMap.Policy =
		tenantplex.Policy{
			ActiveProviderID: pb.currProvider.ID,
		}
	return pb
}

// SetName will set the name of the current Provider
func (pb *ProviderBuilder) SetName(name string) *ProviderBuilder {
	pb.currProvider.Name = name
	return pb
}

// MakeUC will change the current Provider into a UCProvider - if the
// Provider was already a UCProvider, we simply return a UCProviderBuilder,
// otherwise we perform some initialization
func (pb *ProviderBuilder) MakeUC() *UCProviderBuilder {
	if pb.currProvider.Type != tenantplex.ProviderTypeUC {
		pb.currProvider.Type = tenantplex.ProviderTypeUC
		pb.currProvider.UC = &tenantplex.UCProvider{
			IDPURL: (*pb.defaults)().GenDefaultName(),
			Apps:   []tenantplex.UCApp{},
		}
	}
	return &UCProviderBuilder{*pb}
}

// AddUCApp will add a UCApp with a default name to the current UCProvider
func (pb *UCProviderBuilder) AddUCApp() *UCProviderBuilder {
	return pb.AddUCAppWithName((*pb.defaults)().GenDefaultName())
}

// AddUCAppWithName will add a UCApp with the specified name to the current UCProvider
func (pb *UCProviderBuilder) AddUCAppWithName(name string) *UCProviderBuilder {
	appID := (*pb.defaults)().GenDefaultID()
	pb.currProvider.UC.Apps = append(pb.currProvider.UC.Apps, tenantplex.UCApp{ID: appID, Name: name})
	pb.addProviderAppID(appID)
	return pb
}

// SetIDPURL will set the IDP URL of the current UCProvider
func (pb *UCProviderBuilder) SetIDPURL(url string) *UCProviderBuilder {
	pb.currProvider.UC.IDPURL = url
	return pb
}

// MakeAuth0 will change the current Provider into an Auth0Provider - if the
// Provider was already an Auth0Provider, we simply return an Auth0ProviderBuilder,
// otherwise we perform some initialization
func (pb *ProviderBuilder) MakeAuth0() *Auth0ProviderBuilder {
	if pb.currProvider.Type != tenantplex.ProviderTypeAuth0 {
		pb.currProvider.Type = tenantplex.ProviderTypeAuth0
		cs, err := (*pb.defaults)().GenDefaultClientSecret(context.Background(), "auth0mgmt")
		if err != nil {
			// TODO (sgarrity 6/24): this shouldn't panic, but I think the required fix
			// with the builder pattern would be to delay all of this until `Build()`,
			// which right now feels like a bigger refactor than I can bite off
			panic(fmt.Sprintf("failed to save app client secret to SM: %v", err))
		}

		pb.currProvider.Auth0 = &tenantplex.Auth0Provider{
			Domain: (*pb.defaults)().GenDefaultName(),
			Apps:   []tenantplex.Auth0App{},
			Management: tenantplex.Auth0Management{
				ClientID:     (*pb.defaults)().GenDefaultClientID(),
				ClientSecret: *cs,
				Audience:     (*pb.defaults)().GenDefaultName(),
			},
		}
	}
	return &Auth0ProviderBuilder{*pb}
}

// AddAuth0App will add an Auth0App with a default name to the current Auth0Provider
func (pb *Auth0ProviderBuilder) AddAuth0App() *Auth0ProviderBuilder {
	return pb.AddAuth0AppWithName((*pb.defaults)().GenDefaultName())
}

// AddAuth0AppWithName will add an Auth0App with the specified name to the current Auth0Provider
func (pb *Auth0ProviderBuilder) AddAuth0AppWithName(name string) *Auth0ProviderBuilder {
	appID := (*pb.defaults)().GenDefaultID()
	cs, err := (*pb.defaults)().GenDefaultClientSecret(context.Background(), appID.String())
	if err != nil {
		// TODO (sgarrity 6/24): this shouldn't panic, but I think the required fix
		// with the builder pattern would be to delay all of this until `Build()`,
		// which right now feels like a bigger refactor than I can bite off
		panic(fmt.Sprintf("failed to save app client secret to SM: %v", err))
	}

	pb.currProvider.Auth0.Apps =
		append(pb.currProvider.Auth0.Apps,
			tenantplex.Auth0App{
				ID:           appID,
				Name:         name,
				ClientID:     (*pb.defaults)().GenDefaultClientID(),
				ClientSecret: *cs,
			})
	pb.addProviderAppID(appID)
	return pb
}

// SetDomain will set the domain for the current Auth0Provider
func (pb *Auth0ProviderBuilder) SetDomain(domain string) *Auth0ProviderBuilder {
	pb.currProvider.Auth0.Domain = domain
	return pb
}

// SetManagementAudience will set the management audience for the current Auth0Provider
func (pb *Auth0ProviderBuilder) SetManagementAudience(audience string) *Auth0ProviderBuilder {
	pb.currProvider.Auth0.Management.Audience = audience
	return pb
}

// SetRedirect will set the redirect for the current Auth0Provider
func (pb *Auth0ProviderBuilder) SetRedirect(redirect bool) *Auth0ProviderBuilder {
	pb.currProvider.Auth0.Redirect = redirect
	return pb
}
