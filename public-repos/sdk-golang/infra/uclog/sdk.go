package uclog

import (
	"context"

	"github.com/gofrs/uuid"
)

// GetTenantID always returns uuid.Nil since SDK relies on the tenantID specified at initialization time
func GetTenantID(ctx context.Context) uuid.UUID {
	return uuid.Nil
}

// validateHandlerMap does perform any validation in the SDK
func validateHandlerMap(ctx context.Context) {
}
