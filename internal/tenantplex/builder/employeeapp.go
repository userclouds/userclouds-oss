package builder

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/secret"
	"userclouds.com/internal/tenantplex"
)

// EmployeeAppBuilder introduces App-specific building methods for an EmployeeApp;
// it can only be used when a client has added an EmployeeApp to a TenantConfig
// via the TenantConfigBuilder
type EmployeeAppBuilder struct {
	TenantConfigBuilder
}

// AddEmployeeApp will initialize the EmployeeApp
func (tcb *TenantConfigBuilder) AddEmployeeApp() *EmployeeAppBuilder {
	id := (*tcb.defaults)().GenDefaultID()
	cs, err := (*tcb.defaults)().GenDefaultClientSecret(context.Background(), id.String())
	if err != nil {
		// TODO (sgarrity 6/24): this shouldn't panic, but I think the required fix
		// with the builder pattern would be to delay all of this until `Build()`,
		// which right now feels like a bigger refactor than I can bite off
		panic(fmt.Sprintf("failed to save app client secret to SM: %v", err))
	}

	tcb.plexMap.EmployeeApp =
		&tenantplex.App{
			ID:                    id,
			Name:                  (*tcb.defaults)().GenDefaultAppName(),
			ClientID:              (*tcb.defaults)().GenDefaultClientID(),
			ClientSecret:          *cs,
			GrantTypes:            tenantplex.SupportedGrantTypes,
			TokenValidity:         (*tcb.defaults)().GenDefaultTokenValidity(),
			ImpersonateUserConfig: (*tcb.defaults)().GenDefaultImpersonatedUserConfig(),
		}
	if tcb.plexMap.EmployeeProvider != nil {
		tcb.plexMap.EmployeeApp.ProviderAppIDs = []uuid.UUID{tcb.plexMap.EmployeeProvider.UC.Apps[0].ID}
	}
	return &EmployeeAppBuilder{*tcb}
}

// SetID will set the id of the EmployeeApp
func (eab *EmployeeAppBuilder) SetID(id uuid.UUID) *EmployeeAppBuilder {
	eab.plexMap.EmployeeApp.ID = id
	return eab
}

// AddAllowedRedirectURI adds a URI to the EmployeeApp
func (eab *EmployeeAppBuilder) AddAllowedRedirectURI(uri string) *EmployeeAppBuilder {
	eab.plexMap.EmployeeApp.AllowedRedirectURIs = append(eab.plexMap.EmployeeApp.AllowedRedirectURIs, uri)
	return eab
}

// SetClientID will set the client id of the EmployeeApp
func (eab *EmployeeAppBuilder) SetClientID(clientID string) *EmployeeAppBuilder {
	eab.plexMap.EmployeeApp.ClientID = clientID
	return eab
}

// SetClientSecret will set the client secret of the EmployeeApp
func (eab *EmployeeAppBuilder) SetClientSecret(clientSecret secret.String) *EmployeeAppBuilder {
	eab.plexMap.EmployeeApp.ClientSecret = clientSecret
	return eab
}

// SetName will set the name of the EmployeeApp
func (eab *EmployeeAppBuilder) SetName(name string) *EmployeeAppBuilder {
	eab.plexMap.EmployeeApp.Name = name
	return eab
}

// SetOrganizationID will set the name of the EmployeeApp
func (eab *EmployeeAppBuilder) SetOrganizationID(orgID uuid.UUID) *EmployeeAppBuilder {
	eab.plexMap.EmployeeApp.OrganizationID = orgID
	return eab
}
