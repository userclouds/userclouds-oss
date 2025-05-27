package builder

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/internal/tenantplex"
)

type prodTenantConfigDefaults struct {
}

// GenDefaultAppName produces a default app name
func (prodTenantConfigDefaults) GenDefaultAppName() string {
	return ""
}

// GenDefaultClientID produces a default client ID
func (prodTenantConfigDefaults) GenDefaultClientID() string {
	return crypto.GenerateClientID()
}

// GenDefaultClientSecret produces a default client secret
func (prodTenantConfigDefaults) GenDefaultClientSecret(ctx context.Context, clientID string) (*secret.String, error) {
	return crypto.GenerateClientSecret(ctx, clientID)
}

// GenDefaultTokenValidity produces a default token validity
func (prodTenantConfigDefaults) GenDefaultTokenValidity() tenantplex.TokenValidity {
	return tenantplex.TokenValidity{
		Access:          ucjwt.DefaultValidityAccess,
		Refresh:         ucjwt.DefaultValidityRefresh,
		ImpersonateUser: ucjwt.DefaultValidityImpersonateUser,
	}
}

// GenDefaultImpersonatedUserConfig produces a default impersonated user config
func (prodTenantConfigDefaults) GenDefaultImpersonatedUserConfig() tenantplex.ImpersonateUserConfig {
	return tenantplex.ImpersonateUserConfig{
		CheckAttribute:          "",
		BypassCompanyAdminCheck: false,
	}
}

// GenDefaultID produces a default test id
func (prodTenantConfigDefaults) GenDefaultID() uuid.UUID {
	return uuid.Must(uuid.NewV4())
}

// GenDefaultName produces a default test name
func (prodTenantConfigDefaults) GenDefaultName() string {
	return ""
}

// GenDefaultKeys produces default test JWT keys
func (prodTenantConfigDefaults) GenDefaultKeys() tenantplex.Keys {
	return tenantplex.Keys{}
}

// GenDefaultTenantURL produces a default tenant URL
func (prodTenantConfigDefaults) GenDefaultTenantURL() string {
	return ""
}

// NewProdTenantConfigBuilder creates a new TenantConfigBuilder, initializing
// Tenant-Specific attributes of the associated TenantConfig
func NewProdTenantConfigBuilder() *TenantConfigBuilder {
	defaults := func() TenantConfigDefaults {
		return prodTenantConfigDefaults{}
	}
	return NewTenantConfigBuilder(defaults)
}

// NewProdTenantConfigBuilderFromTenantConfig creates a new TenantConfigBuilder from
// the passed in TenantConfig
func NewProdTenantConfigBuilderFromTenantConfig(tc tenantplex.TenantConfig) *TenantConfigBuilder {
	defaults := func() TenantConfigDefaults {
		return prodTenantConfigDefaults{}
	}
	return NewTenantConfigBuilderFromTenantConfig(defaults, tc)
}
