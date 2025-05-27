package cachetool

import (
	"context"

	"github.com/gofrs/uuid"

	authzhelpers "userclouds.com/authz/helpers"
	idphelpers "userclouds.com/idp/helpers"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	plexstorage "userclouds.com/internal/tenantplex/storage"
	"userclouds.com/worker"
)

// ClearCache clears a given cache
func ClearCache(ctx context.Context, cfg *cache.Config, companyConfigStorage *companyconfig.Storage, params worker.ClearCacheParams) error {
	uclog.Infof(ctx, "Starting to clear cache: %s for tenant ID: %s", params.CacheType, params.TenantID)
	if params.CacheType == worker.CacheTypeAuthZ || params.CacheType == worker.CacheTypeAll {
		return ucerr.Wrap(clearAuthZCache(ctx, cfg, params.TenantID))
	}
	if params.CacheType == worker.CacheTypeUserStore || params.CacheType == worker.CacheTypeAll {
		return ucerr.Wrap(clearUserstoreCache(ctx, cfg, params.TenantID))
	}
	if params.CacheType == worker.CacheTypePlex || params.CacheType == worker.CacheTypeAll {
		return ucerr.Wrap(clearPlexCache(ctx, cfg))
	}
	if params.CacheType == worker.CacheTypeCompanyConfig || params.CacheType == worker.CacheTypeAll {
		return ucerr.Wrap(clearCompanyConfigCache(ctx, companyConfigStorage))
	}
	return nil
}

func clearAuthZCache(ctx context.Context, cfg *cache.Config, tenantID uuid.UUID) error {
	if tenantID.IsNil() {
		return ucerr.New("Clear authz cache flush is called without tenant id")
	}

	return ucerr.Wrap(authzhelpers.ClearCache(ctx, cfg, tenantID))
}

func clearUserstoreCache(ctx context.Context, cfg *cache.Config, tenantID uuid.UUID) error {
	if tenantID.IsNil() {
		return ucerr.New("Clear userstore cache flush is called without tenant id")
	}

	return ucerr.Wrap(idphelpers.ClearCache(ctx, cfg, tenantID))
}

func clearPlexCache(ctx context.Context, cfg *cache.Config) error {
	return ucerr.Wrap(plexstorage.ClearCache(ctx, false, cfg))
}

func clearCompanyConfigCache(ctx context.Context, companyConfigStorage *companyconfig.Storage) error {
	return ucerr.Wrap(companyConfigStorage.FlushCacheForCompany(ctx, uuid.Nil)) // Right now this will flush all keys in company cache
}
