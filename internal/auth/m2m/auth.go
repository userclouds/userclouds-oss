package m2m

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/companyconfig"
)

// SubjectTypeM2M is the subject type for M2M tokens
const SubjectTypeM2M = "m2m"

// ValidateM2MSecret returns an error if the tenant ID does not match the secret
func ValidateM2MSecret(ctx context.Context, tenantID uuid.UUID, token string) error {
	correctToken, err := GetM2MSecret(ctx, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if correctToken != token {
		return ucerr.Errorf("invalid access token")
	}

	return nil
}

// GetM2MSecret returns the M2M access token for the tenant
// Note that this gives you service access to the tenant, so the secret itself is sensitive
// But this code always runs in the context of a service that has database creds for the tenant,
// and so can look up the CCF creds for the tenant and get the same access ... this just makes
// it faster / easier (and better on prem)
func GetM2MSecret(ctx context.Context, tenantID uuid.UUID) (string, error) {
	secretName := getTenantSecretLocation(tenantID)
	s := secret.FromLocation(secretName)
	return s.Resolve(ctx)
}

// GetM2MSecretAuthHeader returns authorization header value for M2M authentication
func GetM2MSecretAuthHeader(ctx context.Context, tenantID uuid.UUID) (string, error) {
	token, err := GetM2MSecret(ctx, tenantID)
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	return fmt.Sprintf("AccessToken %s", token), nil
}

// GetM2MTokenSource is a jsonclient.Option that adds the M2M secret header to a request
func GetM2MTokenSource(ctx context.Context, tenantID uuid.UUID) (jsonclient.Option, error) {
	value, err := GetM2MSecretAuthHeader(ctx, tenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return jsonclient.HeaderAuth(value), nil
}

func getTenantSecretLocation(tenantID uuid.UUID) string {
	return secret.LocationFromName(string(service.Console), getTenantSecretName(tenantID))
}

func getTenantSecretName(tenantID uuid.UUID) string {
	return fmt.Sprintf("cross_service_auth_token/%s", tenantID)
}

// CreateSecret creates a new M2M secret for the tenant (in the future this can add to a list)
func CreateSecret(ctx context.Context, tenant *companyconfig.Tenant) error {
	val := crypto.MustRandomHex(32)
	_, err := secret.NewString(ctx, string(service.Console), getTenantSecretName(tenant.ID), val)
	return ucerr.Wrap(err)
}
