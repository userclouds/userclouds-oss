package test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uctypes/messaging/email"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/security"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/routes"
)

// InitForExternalTests initializes the plex handler for use in e2e/integration tests outside of Plex itself,
// where access to internal structures is not necessary.
func InitForExternalTests(ctx context.Context, t *testing.T, hb *builder.HandlerBuilder, companyConfigStorage *companyconfig.Storage, jwtVerifier auth.Verifier, email email.Client, consoleTenantID uuid.UUID) {
	// Because this is used for e2e/integration tests, we use the ProdFactory
	// to talk to an actual UC IDP, not a mock IDP.
	m2mAuth, err := m2m.GetM2MTokenSource(ctx, consoleTenantID)
	assert.NoErr(t, err)
	routes.InitForTests(ctx, m2mAuth, hb, companyConfigStorage, jwtVerifier, &SecurityValidator{}, email, provider.ProdFactory{}, nil, consoleTenantID)
}

// SecurityValidator implements security.ReqValidator and is a test-only validator for Plex tests
type SecurityValidator struct {
}

// ValidateRequest implements security.ReqValidator
func (*SecurityValidator) ValidateRequest(r *http.Request) (security.Status, error) {
	return security.Status{}, nil
}

// FinishedRequest implements security.ReqValidator
func (*SecurityValidator) FinishedRequest(status security.Status, r *http.Request, success bool) {
	// Do nothing
}

// IsCallBlocked implements security.ReqValidator
func (*SecurityValidator) IsCallBlocked(ctx context.Context, username string) bool {
	return false
}
