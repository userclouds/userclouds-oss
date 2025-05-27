package worker

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/config"
	"userclouds.com/idp/internal/userstore"
	"userclouds.com/internal/tenantmap"
)

// ExecuteMutator is a pass-through function to internal function userstore.ExecuteMutator
func ExecuteMutator(
	ctx context.Context,
	req idp.ExecuteMutatorRequest,
	ts *tenantmap.TenantState,
	searchUpdateCfg *config.SearchUpdateConfig,
) ([]uuid.UUID, int, error) {
	return userstore.ExecuteMutator(ctx, req, ts.ID, nil, searchUpdateCfg)
}
