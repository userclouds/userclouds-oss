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

// LogCache logs a given cache
func LogCache(ctx context.Context, cfg *cache.Config, companyConfigStorage *companyconfig.Storage, params worker.LogCacheParams) error {
	uclog.Infof(ctx, "Starting to log cache: %s for tenant ID: %s", params.CacheType, params.TenantID)
	if params.CacheType == worker.CacheTypeAuthZ || params.CacheType == worker.CacheTypeAll {
		return ucerr.Wrap(logAuthZCache(ctx, cfg, params.TenantID))
	}
	if params.CacheType == worker.CacheTypeUserStore || params.CacheType == worker.CacheTypeAll {
		return ucerr.Wrap(logUserstoreCache(ctx, cfg, params.TenantID))
	}
	if params.CacheType == worker.CacheTypePlex || params.CacheType == worker.CacheTypeAll {
		return ucerr.Wrap(logPlexCache(ctx, cfg))
	}
	if params.CacheType == worker.CacheTypeCompanyConfig || params.CacheType == worker.CacheTypeAll {
		return ucerr.Wrap(logCompanyConfigCache(ctx, companyConfigStorage))
	}
	return nil
}

func logAuthZCache(ctx context.Context, cfg *cache.Config, tenantID uuid.UUID) error {
	if tenantID.IsNil() {
		return ucerr.New("Log authz cache flush is called without tenant id")
	}

	return ucerr.Wrap(authzhelpers.LogCache(ctx, cfg, tenantID))
}

func logUserstoreCache(ctx context.Context, cfg *cache.Config, tenantID uuid.UUID) error {
	if tenantID.IsNil() {
		return ucerr.New("Log userstore cache flush is called without tenant id")
	}

	return ucerr.Wrap(idphelpers.LogCache(ctx, cfg, tenantID))
}

func logPlexCache(ctx context.Context, cfg *cache.Config) error {
	return ucerr.Wrap(plexstorage.LogCache(ctx, false, cfg))
}

func logCompanyConfigCache(ctx context.Context, companyConfigStorage *companyconfig.Storage) error {
	return ucerr.Wrap(companyConfigStorage.FlushCacheForCompany(ctx, uuid.Nil)) // Right now this will flush all keys in company cache
}
