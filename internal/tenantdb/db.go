package tenantdb

import (
	"context"
	"sync"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/migrate"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctrace"
	"userclouds.com/internal/companyconfig"
)

var tracer = uctrace.NewTracer("tenantdb")

// Connect connects to a per-tenant DB.
// Will return sql.ErrNoRows if the tenant is not found.
func Connect(ctx context.Context, storage *companyconfig.Storage, tenantID uuid.UUID) (*ucdb.DB, map[region.DataRegion]*ucdb.DB, region.DataRegion, error) {
	return uctrace.Wrap3(ctx, tracer, "tenantdb.Connect "+tenantID.String(), true, func(ctx context.Context) (*ucdb.DB, map[region.DataRegion]*ucdb.DB, region.DataRegion, error) {
		tenantInternal, err := storage.GetTenantInternal(ctx, tenantID)
		if err != nil {
			return nil, nil, "", ucerr.Wrap(err)
		}

		if err := tenantInternal.TenantDBConfig.Validate(); err != nil {
			return nil, nil, "", ucerr.Wrap(err)
		}

		userRegionDbMap := make(map[region.DataRegion]*ucdb.DB)
		errMap := make(map[region.DataRegion]error)
		var mu sync.Mutex
		var wg sync.WaitGroup

		// Connect to all remote region DBs in parallel
		for r, cfg := range tenantInternal.RemoteUserRegionDBConfigs {
			wg.Add(1)
			go func(r region.DataRegion, cfg ucdb.Config) {
				defer wg.Done()
				if err := cfg.Validate(); err != nil {
					uclog.Errorf(ctx, "failed to validate DB config for region %s: %v", r, err)
					mu.Lock()
					errMap[r] = err
					mu.Unlock()
					return
				}
				rDb, err := ConnectWithConfig(ctx, &cfg)
				if err != nil {
					uclog.Errorf(ctx, "failed to connect to DB for region %s: %v", r, err)
					mu.Lock()
					errMap[r] = err
					mu.Unlock()
					return
				}
				mu.Lock()
				userRegionDbMap[r] = rDb
				mu.Unlock()
			}(r, cfg)
		}

		// Connect to the primary user region DB on the main thread
		db, err := ConnectWithConfig(ctx, &tenantInternal.TenantDBConfig)
		mu.Lock()
		if err != nil {
			errMap[tenantInternal.PrimaryUserRegion] = err
		} else {
			userRegionDbMap[tenantInternal.PrimaryUserRegion] = db
		}
		mu.Unlock()

		wg.Wait()

		if len(errMap) > 0 {
			// if there were any errors, close any connections that were successfully opened
			for r, db := range userRegionDbMap {
				if err := db.Close(ctx); err != nil {
					uclog.Errorf(ctx, "failed to close DB for region %s: %v", r, err)
				}
			}

			// then return the first error
			for _, err := range errMap {
				return nil, nil, "", ucerr.Wrap(err)
			}
		}

		return db, userRegionDbMap, tenantInternal.PrimaryUserRegion, nil
	})
}

// ConnectWithConfig connects to a per-tenant DB via config.
func ConnectWithConfig(ctx context.Context, cfg *ucdb.Config) (*ucdb.DB, error) {
	db, err := ucdb.New(ctx, cfg, migrate.SchemaValidator(Schema))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return db, nil
}
