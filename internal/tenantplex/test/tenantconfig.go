package test

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/internal/tenantplex/builder"
	"userclouds.com/internal/testkeys"
)

type testTenantConfigDefaults struct {
}

// GenDefaultAppName produces a default test app name
func (testTenantConfigDefaults) GenDefaultAppName() string {
	return "App_" + crypto.MustRandomDigits(6)
}

// GenDefaultClientID produces a default test client ID
func (testTenantConfigDefaults) GenDefaultClientID() string {
	return crypto.GenerateClientID()
}

// GenDefaultClientSecret produces a default test client secret
func (testTenantConfigDefaults) GenDefaultClientSecret(ctx context.Context, clientID string) (*secret.String, error) {
	return crypto.GenerateClientSecret(ctx, clientID)
}

// GenDefaultTokenValidity produces a default test token validity
func (testTenantConfigDefaults) GenDefaultTokenValidity() tenantplex.TokenValidity {
	return tenantplex.TokenValidity{
		Access:          ucjwt.DefaultValidityAccess,
		Refresh:         ucjwt.DefaultValidityRefresh,
		ImpersonateUser: ucjwt.DefaultValidityImpersonateUser,
	}
}

// GenDefaultImpersonatedUserConfig produces a default test impersonated user config
func (testTenantConfigDefaults) GenDefaultImpersonatedUserConfig() tenantplex.ImpersonateUserConfig {
	return tenantplex.ImpersonateUserConfig{
		CheckAttribute:          "",
		BypassCompanyAdminCheck: false,
	}
}

// GenDefaultID produces a default test id
func (testTenantConfigDefaults) GenDefaultID() uuid.UUID {
	return uuid.Must(uuid.NewV4())
}

// GenDefaultName produces a default test name
func (testTenantConfigDefaults) GenDefaultName() string {
	return "unused"
}

// GenDefaultKeys produces default test JWT keys
func (testTenantConfigDefaults) GenDefaultKeys() tenantplex.Keys {
	return testkeys.Config
}

// GenDefaultTenantURL produces a default test tenant URL
func (testTenantConfigDefaults) GenDefaultTenantURL() string {
	return fmt.Sprintf("http://tenant_%s.uc.com", crypto.MustRandomDigits(6))
}

// NewTenantConfigBuilder creates a new TenantConfigBuilder, initializing
// Tenant-Specific attributes of the associated TenantConfig
func NewTenantConfigBuilder() *builder.TenantConfigBuilder {
	defaults := func() builder.TenantConfigDefaults {
		return testTenantConfigDefaults{}
	}
	return builder.NewTenantConfigBuilder(defaults)
}

// NewTenantConfigBuilderFromTenantConfig creates a new TenantConfigBuilder from
// the passed in TenantConfig
func NewTenantConfigBuilderFromTenantConfig(tc tenantplex.TenantConfig) *builder.TenantConfigBuilder {
	defaults := func() builder.TenantConfigDefaults {
		return testTenantConfigDefaults{}
	}
	return builder.NewTenantConfigBuilderFromTenantConfig(defaults, tc)
}
