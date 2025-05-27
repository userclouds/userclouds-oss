package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"userclouds.com/infra/migrate"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/logdb"
)

func migrateTenants(ctx context.Context, uv universe.Universe) {
	serviceDataConsole, err := getDatabaseData(ctx, uv, "companyconfig")
	if err != nil {
		uclog.Fatalf(ctx, "error loading companyconfig db config: %v", err)
	}

	serviceDataTenantDB, err := getDatabaseData(ctx, uv, "tenantdb")
	if err != nil {
		uclog.Fatalf(ctx, "error loading companyconfig db config: %v", err)
	}

	db, err := ucdb.New(ctx, serviceDataConsole.DBCfg, noopValidator{})
	if err != nil {
		uclog.Fatalf(ctx, "error connecting to companyconfig db: %v", err)
	}

	storage, err := companyconfig.NewStorage(ctx, db, nil)
	if err != nil {
		uclog.Fatalf(ctx, "error creating companyconfig storage: %v", err)
	}

	cTDB := -1
	rTDB := -1
	cLDB := -1
	rLDB := -1
	var suppressTDB bool
	var suppressLDB bool

	pager, err := companyconfig.NewTenantPaginatorFromOptions(
		pagination.Limit(pagination.MaxLimit))
	if err != nil {
		uclog.Fatalf(ctx, "error initializing pagination options: %v", err)
	}

	for {
		tenants, respFields, err := storage.ListTenantsPaginated(ctx, *pager)
		if err != nil {
			uclog.Fatalf(ctx, "error listing tenants: %v", err)
		}

		// TODO: how important is it to get the count up front? This is not as useful now.
		uclog.Infof(ctx, "Migrating tenants: Got %d tenants to migrate...", len(tenants))

		for i, t := range tenants {
			ti, err := storage.GetTenantInternal(ctx, t.ID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					// We have a row in the tenants table but are missing a row in the tenant IDP table
					uclog.Infof(ctx, "Migrating %v: Couldn't find tenants_internal row for tenant (%s)", t.ID, t.Name)
					if promptDeleteTenant(ctx, storage, &t) {
						uclog.Infof(ctx, "Migrating %v: Deleted tenant instead of migrating", t.ID)
					} else {
						uclog.Infof(ctx, "Migrating %v: Skipped migration due to missing row in tenants_internal", t.ID)
					}
					continue
				} else {
					uclog.Fatalf(ctx, "error loading tenant %v db cfg: %v", t.ID, err)
				}
			}

			if err := ti.Validate(); err != nil {
				// We failed to validate the tenant config which means this tenant is malformed in some way
				uclog.Infof(ctx, "Migrating %v: Couldn't validate per tenant config for tenant (%s) - %v", t.ID, t.Name, err)
				if promptDeleteTenant(ctx, storage, &t) {
					uclog.Infof(ctx, "Migrating %v: Deleted tenant instead of migrating", t.ID)
				} else {
					uclog.Infof(ctx, "Migrating %v: Skipped migration due to invalid config in tenants_internal", t.ID)
				}
				continue
			}

			// we cycle current & requested so that if the previous database matched the same way,
			// you don't need to confirm again.
			tenantIDStr := fmt.Sprintf("tenant %v (%s) (%d/%d)", t.ID, t.Name, i+1, len(tenants))

			tenantDBData := migrate.ServiceData{
				DBCfg:                    &ti.TenantDBConfig,
				Migrations:               serviceDataTenantDB.Migrations,
				BaselineVersion:          serviceDataTenantDB.BaselineVersion,
				BaselineCreateStatements: serviceDataTenantDB.BaselineCreateStatements,
				PostgresOnlyExtensions:   serviceDataTenantDB.PostgresOnlyExtensions,
			}
			logDBData := migrate.ServiceData{
				DBCfg:                    &ti.LogConfig.LogDB,
				Migrations:               logdb.GetMigrations(),
				BaselineVersion:          -1,
				BaselineCreateStatements: []string{},
			}
			cTDB, rTDB = migrateDatabase(ctx, tenantIDStr, tenantDBData, cTDB, rTDB, &suppressTDB)
			for _, regionDB := range ti.RemoteUserRegionDBConfigs {
				regionDBData := migrate.ServiceData{
					DBCfg:                    &regionDB,
					Migrations:               serviceDataTenantDB.Migrations,
					BaselineVersion:          serviceDataTenantDB.BaselineVersion,
					BaselineCreateStatements: serviceDataTenantDB.BaselineCreateStatements,
					PostgresOnlyExtensions:   serviceDataTenantDB.PostgresOnlyExtensions,
				}
				cTDB, rTDB = migrateDatabase(ctx, tenantIDStr, regionDBData, cTDB, rTDB, &suppressTDB)
			}
			cLDB, rLDB = migrateDatabase(ctx, tenantIDStr, logDBData, cLDB, rLDB, &suppressLDB)
		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}
}

func promptDeleteTenant(ctx context.Context, storage *companyconfig.Storage, t *companyconfig.Tenant) bool {
	if cmdline.Confirm(ctx, "want to delete this tenant? [yN] ") &&
		cmdline.ProductionConfirm(ctx, "I know", "You're in prod. Type 'I know' to confirm: ") {
		if err := storage.DeleteTenant(ctx, t.ID); err != nil {
			uclog.Fatalf(ctx, "error deleting tenant %v: %v", t.ID, err)
		}
		return true
	}
	return false
}
