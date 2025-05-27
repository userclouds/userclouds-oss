package builder

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/secret"
	"userclouds.com/internal/tenantplex"
)

// TenantConfigDefaults provides default values used by the TenantConfigBuilder
type TenantConfigDefaults interface {
	GenDefaultAppName() string

	GenDefaultClientID() string

	GenDefaultClientSecret(context.Context, string) (*secret.String, error)

	GenDefaultID() uuid.UUID

	GenDefaultName() string

	GenDefaultKeys() tenantplex.Keys

	GenDefaultTenantURL() string

	GenDefaultTokenValidity() tenantplex.TokenValidity

	GenDefaultImpersonatedUserConfig() tenantplex.ImpersonateUserConfig
}

// TenantConfigDefaultsGetter is a function that returns a TenantConfigDefaults interface implementation
type TenantConfigDefaultsGetter func() TenantConfigDefaults
