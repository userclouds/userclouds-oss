package storage

import (
	"context"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucerr"
)

// ClearCache clears the cache for all tenants
func ClearCache(ctx context.Context, useTestPrefix bool, cfg *cache.Config) error {
	cm, err := getCacheManager(ctx, useTestPrefix, cfg)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if cm == nil {
		return ucerr.Errorf("Cache manager is nil. Config %v", cfg)
	}
	return ucerr.Wrap(cm.Flush(ctx, "plexstorage"))
}

// LogCache logs the cache for all tenants
func LogCache(ctx context.Context, useTestPrefix bool, cfg *cache.Config) error {
	cm, err := getCacheManager(ctx, false, cfg)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if cm == nil {
		return ucerr.Errorf("Cache manager is nil. Config %v", cfg)
	}
	return ucerr.Wrap(cm.Provider.LogKeyValues(ctx, cm.N.GetPrefix()))
}
