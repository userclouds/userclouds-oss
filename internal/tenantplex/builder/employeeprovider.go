package builder

import (
	"github.com/gofrs/uuid"

	"userclouds.com/internal/tenantplex"
)

// EmployeeProviderBuilder introduces Provider-specific building methods for an
// EmployeeProvider; it can only be used when a client has added an EmployeeProvider
// to a TenantConfig via the TenantConfigBuilder
type EmployeeProviderBuilder struct {
	TenantConfigBuilder
}

// AddEmployeeProvider will initialize the EmployeeProvider
func (tcb *TenantConfigBuilder) AddEmployeeProvider() *EmployeeProviderBuilder {
	// an EmployeeProvider is always of type UC, and has exactly
	// one UCProvider App
	appID := (*tcb.defaults)().GenDefaultID()
	tcb.plexMap.EmployeeProvider =
		&tenantplex.Provider{
			ID:   (*tcb.defaults)().GenDefaultID(),
			Name: (*tcb.defaults)().GenDefaultAppName(),
			Type: tenantplex.ProviderTypeEmployee,
			UC: &tenantplex.UCProvider{
				Apps: []tenantplex.UCApp{
					{
						ID:   appID,
						Name: (*tcb.defaults)().GenDefaultName(),
					},
				},
			},
		}
	if tcb.plexMap.EmployeeApp != nil {
		tcb.plexMap.EmployeeApp.ProviderAppIDs = []uuid.UUID{appID}
	}
	return &EmployeeProviderBuilder{*tcb}
}

// SetID will set the id of the EmployeeProvider
func (epb *EmployeeProviderBuilder) SetID(id uuid.UUID) *EmployeeProviderBuilder {
	epb.plexMap.EmployeeProvider.ID = id
	return epb
}

// SetName will set the name of the EmployeeProvider
func (epb *EmployeeProviderBuilder) SetName(name string) *EmployeeProviderBuilder {
	epb.plexMap.EmployeeProvider.Name = name
	return epb
}

// SetAppName will set the name of the UC App for the EmployeeProvider
func (epb *EmployeeProviderBuilder) SetAppName(name string) *EmployeeProviderBuilder {
	epb.plexMap.EmployeeProvider.UC.Apps[0].Name = name
	return epb
}
