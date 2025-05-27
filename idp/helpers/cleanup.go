package helpers

import (
	"context"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/tenantmap"
)

// CleanForTenant cleans up data for a tenant
func CleanForTenant(ctx context.Context, ts *tenantmap.TenantState, maxCandidates int, dryRun bool) (int, error) {
	s := storage.NewFromTenantState(ctx, ts)
	cm, err := storage.NewUserstoreColumnManager(ctx, s)
	if err != nil {
		return -1, ucerr.Wrap(err)
	}
	umrs := storage.NewUserMultiRegionStorage(ctx, ts.UserRegionDbMap, ts.ID)
	return umrs.CleanupUsers(ctx, cm, maxCandidates, dryRun)
}
