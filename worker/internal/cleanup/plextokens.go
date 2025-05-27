package cleanup

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/plex/helpers"
	"userclouds.com/worker"
)

// CleanPlexTokensForTenant cleans up plex tokens for a tenant
func CleanPlexTokensForTenant(ctx context.Context, tenantID uuid.UUID, tenantDB *ucdb.DB, cacheCfg *cache.Config, params worker.DataCleanupParams) error {
	uclog.Infof(ctx, "Cleaning plex tokens for tenant %v  max: %d dry run: %v", tenantID, params.MaxCandidates, params.DryRun)
	return ucerr.Wrap(helpers.CleanPlexTokensForTenant(ctx, tenantDB, cacheCfg, params.MaxCandidates, params.DryRun))
}
