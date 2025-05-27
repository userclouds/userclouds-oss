package builder

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/secret"
	message "userclouds.com/internal/messageelements"
	"userclouds.com/internal/pageparameters/pagetype"
	"userclouds.com/internal/pageparameters/parameter"
	"userclouds.com/internal/tenantplex"
)

// AppBuilder introduces App-specific building methods; it can only be used
// when a client has added an App to a TenantConfig via the TenantConfigBuilder
type AppBuilder struct {
	TenantConfigBuilder
}

// AddApp adds a plex App to the underlying TenantConfig, and returns a pointer
// to an AppBuilder builder so that other App-specific changes can be made
func (tcb *TenantConfigBuilder) AddApp() *AppBuilder {
	id := (*tcb.defaults)().GenDefaultID()
	cs, err := (*tcb.defaults)().GenDefaultClientSecret(context.Background(), id.String())
	if err != nil {
		// TODO (sgarrity 6/24): this shouldn't panic, but I think the required fix
		// with the builder pattern would be to delay all of this until `Build()`,
		// which right now feels like a bigger refactor than I can bite off
		panic(fmt.Sprintf("failed to save app client secret to SM: %v", err))
	}

	tcb.plexMap.Apps =
		append(tcb.plexMap.Apps,
			tenantplex.App{
				ID:                    id,
				Name:                  (*tcb.defaults)().GenDefaultAppName(),
				ClientID:              (*tcb.defaults)().GenDefaultClientID(),
				ClientSecret:          *cs,
				ProviderAppIDs:        *tcb.providerAppIDs,
				GrantTypes:            tenantplex.SupportedGrantTypes,
				TokenValidity:         (*tcb.defaults)().GenDefaultTokenValidity(),
				ImpersonateUserConfig: (*tcb.defaults)().GenDefaultImpersonatedUserConfig(),
			})
	tcb.currApp = &tcb.plexMap.Apps[len(tcb.plexMap.Apps)-1]
	return &AppBuilder{*tcb}
}

// SwitchToApp sets the current App to the specified index, returning a pointer
// to an AppBuilder. An out of bounds index will panic as this should be caught
// by the programmer during development.
func (tcb *TenantConfigBuilder) SwitchToApp(appNum int) *AppBuilder {
	if appNum < 0 || appNum >= len(tcb.plexMap.Apps) {
		// this would be a programmer error - ok to panic here since
		// this will be discovered during test development
		panic(fmt.Sprintf("invalid index %d into Apps of length %d",
			appNum, len(tcb.plexMap.Apps)))
	}

	tcb.currApp = &tcb.plexMap.Apps[appNum]
	return &AppBuilder{*tcb}
}

// ID returns the ID for the current App
func (ab AppBuilder) ID() uuid.UUID {
	return ab.currApp.ID
}

// ClientID returns the client ID for the current App
func (ab AppBuilder) ClientID() string {
	return ab.currApp.ClientID
}

// ClientSecret returns the client secret for the current App
func (ab AppBuilder) ClientSecret() secret.String {
	return ab.currApp.ClientSecret
}

// Name returns the name of the current App
func (ab AppBuilder) Name() string {
	return ab.currApp.Name
}

// SetID sets the ID for the current App
func (ab *AppBuilder) SetID(id uuid.UUID) *AppBuilder {
	ab.currApp.ID = id
	return ab
}

// SetClientID sets the client ID for the current App
func (ab *AppBuilder) SetClientID(id string) *AppBuilder {
	ab.currApp.ClientID = id
	return ab
}

// SetClientSecret sets the client secret for the current App
func (ab *AppBuilder) SetClientSecret(secret secret.String) *AppBuilder {
	ab.currApp.ClientSecret = secret
	return ab
}

// SetName sets the name for the current App
func (ab *AppBuilder) SetName(name string) *AppBuilder {
	ab.currApp.Name = name
	return ab
}

// SetOrganizationID sets the organizationID for the current App
func (ab *AppBuilder) SetOrganizationID(orgID uuid.UUID) *AppBuilder {
	ab.currApp.OrganizationID = orgID
	return ab
}

// AddAllowedLogoutURI adds a URI to the current App
func (ab *AppBuilder) AddAllowedLogoutURI(uri string) *AppBuilder {
	ab.currApp.AllowedLogoutURIs = append(ab.currApp.AllowedLogoutURIs, uri)
	return ab
}

// AddAllowedRedirectURI adds a URI to the current App
func (ab *AppBuilder) AddAllowedRedirectURI(uri string) *AppBuilder {
	ab.currApp.AllowedRedirectURIs = append(ab.currApp.AllowedRedirectURIs, uri)
	return ab
}

// CustomizeMessageElement adds a message element customization to the current App
func (ab *AppBuilder) CustomizeMessageElement(mt message.MessageType, elt message.ElementType, s string) *AppBuilder {
	ab.currApp.CustomizeMessageElement(mt, elt, s)
	return ab
}

// DeleteAppPageParameter will delete a parameter value override for the given page type for an app
func (ab *AppBuilder) DeleteAppPageParameter(t pagetype.Type, n parameter.Name) *AppBuilder {
	ab.currApp.DeletePageParameter(t, n)
	return ab
}

// SetAppPageParameter will set a parameter value override for the given page type for an app
func (ab *AppBuilder) SetAppPageParameter(t pagetype.Type, n parameter.Name, value string) *AppBuilder {
	ab.currApp.SetPageParameter(t, n, value)
	return ab
}
