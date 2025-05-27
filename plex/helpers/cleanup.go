package helpers

import (
	"context"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/plex/internal/storage"
)

// CleanPlexTokensForTenant cleans up plex tokens for a tenant
func CleanPlexTokensForTenant(ctx context.Context, tenantDB *ucdb.DB, cacheCfg *cache.Config, maxCandidates int, dryRun bool) error {
	s := storage.New(ctx, tenantDB, cacheCfg)
	return ucerr.Wrap(s.CleanPlexTokens(ctx, maxCandidates, dryRun))
}
