package helpers

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/authz/internal"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucerr"
)

// ClearCache clears the cache for the given tenant
func ClearCache(ctx context.Context, cfg *cache.Config, tenantID uuid.UUID) error {
	cm, err := internal.GetCacheManager(ctx, cfg, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if cm == nil {
		return ucerr.Errorf("Cache manager is nil. Config %v", cfg)
	}
	return ucerr.Wrap(cm.Flush(ctx, "AuthZCache"))
}

// LogCache logs the cache for the given tenant
func LogCache(ctx context.Context, cfg *cache.Config, tenantID uuid.UUID) error {
	cm, err := internal.GetCacheManager(ctx, cfg, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if cm == nil {
		return ucerr.Errorf("Cache manager is nil. Config %v", cfg)
	}
	return ucerr.Wrap(cm.Provider.LogKeyValues(ctx, cm.N.GetPrefix()))
}
