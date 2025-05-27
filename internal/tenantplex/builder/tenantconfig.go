package builder

import (
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/oidc"
	"userclouds.com/infra/uctypes/messaging/telephony"
	"userclouds.com/internal/pageparameters/pagetype"
	"userclouds.com/internal/pageparameters/parameter"
	"userclouds.com/internal/tenantplex"
)

// TenantConfigBuilder is used to build up a test TenantConfig structure
//
// The TenantConfigBuilder hierarchy follows the Builder pattern, which allows you
// to construct the complex object that is TenantConfig in an easy to understand
// step-by-step process.
//
// In order to facilitate this chaining of methods, each setter method returns
// a pointer to the builder object. If a setter method creates a sub-object (e.g., AddApp()),
// the pointer for the builder for that particular sub-object is returned. This relies
// on composition, since we embed the base builder class in the sub-object builder, and
// works because all state is maintained within that base builder.
//
// As an example, to create a TenantConfig containing a UCProvider and an App,
// one would do:
//
// NewTenantConfigBuilder().AddProvider().MakeActive().MakeUC().AddUCApp("name").AddApp().Build()
//
// The call to AddProvider above returns the ProviderBuilder, and then the all to
// MakeUC returns the UCProviderBuilder. Note that the call to AddApp is defined on
// the base TenantConfigBuilder, which is accessible because ProviderBuilder is embedded
// within UCProviderBuilder, and in turn TenantConfigBuilder is embedded in ProviderBuilder.
//
// Once AddApp is called, any app-specific builder methods apply to this last created
// app. Similarly, a call to AddProvider will result in any app-specific calls to be applied
// against this last created provider. To provide flexibility, SwitchToApp and SwitchToProvider
// allow a client of TenantConfigBuilder to switch the context of the builder to the
// specified App or Provider at any time in the building process.
type TenantConfigBuilder struct {
	defaults         *TenantConfigDefaultsGetter
	tenantConfig     *tenantplex.TenantConfig
	plexMap          *tenantplex.PlexMap
	currApp          *tenantplex.App
	currOIDCProvider *oidc.ProviderConfig
	currProvider     *tenantplex.Provider
	providerAppIDs   *[]uuid.UUID
}

// NewTenantConfigBuilder creates a new TenantConfigBuilder with the specified
// TenantConfigDefaultsGetter, initializing Tenant-Specific attributes of the
// associated TenantConfig
func NewTenantConfigBuilder(defaults TenantConfigDefaultsGetter) *TenantConfigBuilder {
	return NewTenantConfigBuilderFromTenantConfig(
		defaults,
		tenantplex.TenantConfig{
			Keys:          defaults().GenDefaultKeys(),
			OIDCProviders: tenantplex.OIDCProviders{Providers: oidc.GetDefaultNativeProviders()},
		})
}

// NewTenantConfigBuilderFromTenantConfig creates a new TenantConfigBuilder from
// the specified TenantConfigDefaultsGetter and TenantConfig
func NewTenantConfigBuilderFromTenantConfig(defaults TenantConfigDefaultsGetter, tc tenantplex.TenantConfig) *TenantConfigBuilder {
	tenantConfig := &tc
	plexMap := &tenantConfig.PlexMap

	var currApp *tenantplex.App
	if len(plexMap.Apps) > 0 {
		currApp = &plexMap.Apps[len(plexMap.Apps)-1]
	}

	var currOIDCProvider *oidc.ProviderConfig
	if len(tenantConfig.OIDCProviders.Providers) > 0 {
		currOIDCProvider = &tenantConfig.OIDCProviders.Providers[len(tenantConfig.OIDCProviders.Providers)-1]
	}

	var currProvider *tenantplex.Provider
	providerAppIDs := []uuid.UUID{}
	if len(plexMap.Providers) > 0 {
		currProvider = &plexMap.Providers[len(plexMap.Providers)-1]
		for _, p := range plexMap.Providers {
			if p.Type == tenantplex.ProviderTypeAuth0 {
				for _, a := range p.Auth0.Apps {
					providerAppIDs = append(providerAppIDs, a.ID)
				}
			} else if p.Type == tenantplex.ProviderTypeUC {
				for _, a := range p.UC.Apps {
					providerAppIDs = append(providerAppIDs, a.ID)
				}
			}
		}
	}

	return &TenantConfigBuilder{
		defaults:         &defaults,
		tenantConfig:     tenantConfig,
		plexMap:          plexMap,
		currApp:          currApp,
		currOIDCProvider: currOIDCProvider,
		currProvider:     currProvider,
		providerAppIDs:   &providerAppIDs,
	}
}

// String will output a pretty-print representation of the underlying
// TenantConfig, including indentation and the keys and associated values,
// which is useful for testing
func (tcb TenantConfigBuilder) String() string {
	j, err := json.MarshalIndent(*tcb.tenantConfig, "", "  ")
	if err != nil {
		return fmt.Sprintf("conversion failed with: %v", err)
	}
	return string(j)

}

// SetDisableSignUps will set the associated flag in TenantConfig
func (tcb *TenantConfigBuilder) SetDisableSignUps(b bool) *TenantConfigBuilder {
	tcb.tenantConfig.DisableSignUps = b
	return tcb
}

// DeleteTenantPageParameter will delete a parameter value override for the given page type for a tenant
func (tcb *TenantConfigBuilder) DeleteTenantPageParameter(t pagetype.Type, n parameter.Name) *TenantConfigBuilder {
	tcb.tenantConfig.DeletePageParameter(t, n)
	return tcb
}

// SetTenantPageParameter will set a parameter value override for the given page type for a tenant
func (tcb *TenantConfigBuilder) SetTenantPageParameter(t pagetype.Type, n parameter.Name, value string) *TenantConfigBuilder {
	tcb.tenantConfig.SetPageParameter(t, n, value)
	return tcb
}

// SetKeys will set the JWT Keys in TenantConfig
func (tcb *TenantConfigBuilder) SetKeys(k tenantplex.Keys) *TenantConfigBuilder {
	tcb.tenantConfig.Keys = k
	return tcb
}

// SetVerifyEmails will set the associated flag in TenantConfig
func (tcb *TenantConfigBuilder) SetVerifyEmails(b bool) *TenantConfigBuilder {
	tcb.tenantConfig.VerifyEmails = b
	return tcb
}

// ResetTelephonyProvider will reset the telephony provider in TenantConfig
func (tcb *TenantConfigBuilder) ResetTelephonyProvider() *TenantConfigBuilder {
	tcb.plexMap.TelephonyProvider = telephony.ProviderConfig{
		Type:       telephony.ProviderTypeNone,
		Properties: map[telephony.PropertyKey]string{},
	}
	return tcb
}

// Build actually emits the built TenantConfig
func (tcb *TenantConfigBuilder) Build() tenantplex.TenantConfig {
	return *tcb.tenantConfig
}
