package helpers

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucerr"
)

// GetCacheNameProviderForTenantID returns the cache name provider for the given tenant ID. This helps in cases where we need to access the name provider but not allowed to import internal packages.
func GetCacheNameProviderForTenantID(tenantID uuid.UUID) cache.KeyNameProvider {
	return storage.NewCacheNameProviderForTenant(tenantID)
}

// ClearCache clears the cache for the given tenant
func ClearCache(ctx context.Context, cfg *cache.Config, tenantID uuid.UUID) error {
	cm, err := storage.GetCacheManager(ctx, cfg, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if cm == nil {
		return ucerr.Errorf("Cache manager is nil. Config %v", cfg)
	}
	return ucerr.Wrap(cm.Flush(ctx, "userstore"))
}

// LogCache logs the cache for the given tenant
func LogCache(ctx context.Context, cfg *cache.Config, tenantID uuid.UUID) error {
	cm, err := storage.GetCacheManager(ctx, cfg, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if cm == nil {
		return ucerr.Errorf("Cache manager is nil. Config %v", cfg)
	}
	return ucerr.Wrap(cm.Provider.LogKeyValues(ctx, cm.N.GetPrefix()))
}
