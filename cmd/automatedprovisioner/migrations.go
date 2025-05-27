package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	tenantdb "userclouds.com/idp/migration"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/dbdata"
	"userclouds.com/internal/logdb"
)

type noopValidator struct{}

// Validate implements ucdb.Validate but does nothing, since we want to operate on non-current DBs
func (n noopValidator) Validate(_ context.Context, _ *ucdb.DB) error {
	return nil
}

func migrateDatabases(ctx context.Context, uv universe.Universe, tenantDBDownMigrate int) (map[string]*migrate.ServiceData, error) {
	var dbServiceNames = []string{"rootdb", "companyconfig", "rootdbstatus", "status"}
	// Bail out early if any config is missing or bad
	serviceData, err := loadServices(ctx, dbServiceNames)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	serviceDataTenantDB, err := getTenantDBData(ctx, uv)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	rootdbData := serviceData["rootdb"]
	rootdbstatusSD := serviceData["rootdbstatus"]
	companyConfigSD := serviceData["companyconfig"]
	if err := bootstrapDB(ctx, rootdbData.DBCfg, rootdbstatusSD.DBCfg); err != nil {
		return nil, ucerr.Wrap(err)
	}
	// Not supporting rootDB down migrations for now.
	if err := migrateDB(ctx, "rootDB", rootdbData, -1); err != nil {
		return nil, ucerr.Wrap(err)
	}
	// Not supporting non tenant DB down migrations for now.
	for _, service := range dbServiceNames {
		if err := migrateDB(ctx, service, serviceData[service], -1); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}
	if err := migrateTenants(ctx, companyConfigSD, serviceDataTenantDB, tenantDBDownMigrate); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return serviceData, nil
}

func loadServices(ctx context.Context, names []string) (map[string]*migrate.ServiceData, error) {
	serviceData := make(map[string]*migrate.ServiceData)
	for _, service := range names {
		sd, err := dbdata.GetDatabaseData(ctx, service)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		serviceData[service] = sd
	}
	return serviceData, nil
}

func cloneConfigForDB(cfg *ucdb.Config, dbName string) *ucdb.Config {
	return &ucdb.Config{
		User:      cfg.User,
		Password:  cfg.Password,
		Host:      cfg.Host,
		Port:      cfg.Port,
		DBName:    dbName,
		DBDriver:  cfg.DBDriver,
		DBProduct: cfg.DBProduct,
	}
}

func bootstrapDB(ctx context.Context, rootDBCfg, rootdbstatus *ucdb.Config) error {
	uclog.Infof(ctx, "bootstrapping %s", rootDBCfg.DBName)
	statments := []string{
		fmt.Sprintf(`GRANT ALL PRIVILEGES ON DATABASE %s TO %s`, rootDBCfg.DBName, rootDBCfg.User),
		`CREATE TABLE tmp (id UUID)`,
		`DROP TABLE tmp`,
	}
	pgDB, err := ucdb.New(ctx, cloneConfigForDB(rootDBCfg, "postgres"), noopValidator{})
	if err != nil {
		return ucerr.Wrap(err)
	}

	defer func() {
		if err := pgDB.Close(ctx); err != nil {
			uclog.Warningf(ctx, "failed to close db connection: %s", rootDBCfg.DBName)
		}
	}()
	if err := createDBIfNotExists(ctx, pgDB, rootDBCfg.DBName); err != nil {
		return ucerr.Wrap(err)
	}
	if err := createDBIfNotExists(ctx, pgDB, rootdbstatus.DBName); err != nil {
		return ucerr.Wrap(err)
	}
	defaultDB, err := ucdb.New(ctx, rootDBCfg, noopValidator{})
	if err != nil {
		return ucerr.Wrap(err)
	}

	defer func() {
		if err := defaultDB.Close(ctx); err != nil {
			uclog.Warningf(ctx, "failed to close db connection: %s", rootDBCfg.DBName)
		}
	}()
	for i, stmt := range statments {
		if _, err := defaultDB.ExecContext(ctx, fmt.Sprintf("BootstrapDB-%d", i), stmt); err != nil {
			return ucerr.Wrap(err)
		}
	}
	uclog.Infof(ctx, "DB %s bootstrapped successfully", rootDBCfg.DBName)
	return nil
}

func createDBIfNotExists(ctx context.Context, pgDB *ucdb.DB, dbName string) error {
	isDBExits := fmt.Sprintf(`
	/* bypass-known-table-check */
	SELECT FROM pg_database WHERE datname = '%s'`, dbName)
	res, err := pgDB.ExecContext(ctx, "CheckDBExists", isDBExits)
	if err != nil {
		return ucerr.Wrap(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return ucerr.Wrap(err)
	}
	if rows > 0 {
		uclog.Infof(ctx, "DB %s already exists", dbName)
		return nil
	}
	uclog.Infof(ctx, "DB %s doesn't exist. Creating", dbName)

	if _, err := pgDB.ExecContext(ctx, "CreateDefaultDB", fmt.Sprintf(`CREATE DATABASE %s`, dbName)); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

func migrateDB(ctx context.Context, name string, sd *migrate.ServiceData, downMigrateRequestedVersion int) error {
	uclog.Infof(ctx, "Migrating %v for %s", sd.DBCfg.DBName, name)
	db, err := ucdb.New(ctx, sd.DBCfg, noopValidator{})
	if err != nil {
		return ucerr.Wrap(err)
	}
	if err := migrate.CreateMigrationsTable(ctx, db); err != nil {
		return ucerr.Wrap(err)
	}
	currentVersion, err := migrate.GetMaxVersion(ctx, db)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if downMigrateRequestedVersion != -1 {
		return ucerr.Wrap(downgradeDB(ctx, db, sd, downMigrateRequestedVersion, currentVersion))
	}
	targetVersion := sd.Migrations.GetMaxAvailable()
	if currentVersion == targetVersion {
		uclog.Infof(ctx, "DB %v is already at max version %v", sd.DBCfg.DBName, currentVersion)
		return nil
	}
	if currentVersion > targetVersion {
		return ucerr.Errorf("DB %v is at version %v, which is newer than max available %v", sd.DBCfg.DBName, currentVersion, targetVersion)
	}
	if err := verifyMigrationsMatch(ctx, name, db, sd, targetVersion); err != nil {
		return ucerr.Wrap(err)
	}

	uclog.Infof(ctx, "migrating %v from %d to %d", sd.DBCfg.DBName, currentVersion, targetVersion)
	start := time.Now().UTC()
	if err := migrate.EnablePostgresExtensions(ctx, db, sd.PostgresOnlyExtensions); err != nil {
		return ucerr.Wrap(err)
	}
	if currentVersion == -1 {
		if targetVersion < sd.BaselineVersion {
			return ucerr.Errorf("cannot migrate to version %d, it is below the baseline version %d", targetVersion, sd.BaselineVersion)
		}
		for _, sql := range sd.BaselineCreateStatements {
			if _, err := db.ExecContext(ctx, "migrateDatabase", sql); err != nil {
				return ucerr.Wrap(err)
			}
		}
		currentVersion = sd.BaselineVersion
	}

	// upgrade path, where we use the migrations defined in code (since they haven't been
	// used in the database yet, by definition)
	if err := sd.Migrations.DoMigration(ctx, db, currentVersion, targetVersion); err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "migration of %v from %d -> %d finished successfully. took: %v", sd.DBCfg.DBName, currentVersion, targetVersion, time.Now().UTC().Sub(start))
	return nil
}

func downgradeDB(ctx context.Context, db *ucdb.DB, sd *migrate.ServiceData, downMigrateRequestedVersion, currentVersion int) error {
	if downMigrateRequestedVersion > currentVersion {
		return ucerr.Errorf("DB %v is at version %v, which is older than the requested downgrade version: %v", sd.DBCfg.DBName, currentVersion, downMigrateRequestedVersion)
	} else if downMigrateRequestedVersion == currentVersion {
		uclog.Infof(ctx, "DB %v is already at the requested downgrade version %v", sd.DBCfg.DBName, downMigrateRequestedVersion)
		return nil
	}
	uclog.Warningf(ctx, "DB: %v downgrading schema from %v to %v", sd.DBCfg.DBName, currentVersion, downMigrateRequestedVersion)
	return ucerr.Wrap(sd.Migrations.DoMigration(ctx, db, currentVersion, downMigrateRequestedVersion))
}

func verifyMigrationsMatch(ctx context.Context, service string, db *ucdb.DB, sd *migrate.ServiceData, maxAvail int) error {
	existingMigs, err := migrate.SelectMigrations(ctx, db)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// only check up to the requested version, since we might know about this and be trying to downgrade
	var codeMigrationIndex int
	for i := 0; i < len(existingMigs) && codeMigrationIndex < len(sd.Migrations) && i < maxAvail; i++ {
		codeMig := sd.Migrations[codeMigrationIndex]
		dbMig := existingMigs[i]

		// since we truncated the first part of the code migrations on 3/21/23, we can't check until we have matching versions
		if dbMig.Version < codeMig.Version {
			continue
		}
		codeMigrationIndex++
		// only check table, up, down
		if codeMig.Table != dbMig.Table {
			return ucerr.Errorf("DB %s@%s Migration %v doesn't match for between code & database: tables. Code: '%s', DB: '%s'",
				service, sd.DBCfg.DBName, codeMig.Version, codeMig.Table, dbMig.Table)
		}

		if !migrate.CompareMigrations(dbMig.Up, codeMig.Up) {
			return ucerr.Errorf("DB %s@%s Migration %v doesn't match for between code & database: Up SQL. DB: `%s`\nCode: `%s`",
				service, sd.DBCfg.DBName, codeMig.Version, dbMig.Up, codeMig.Up)
		}
		if !migrate.CompareMigrations(dbMig.Down, codeMig.Down) {
			return ucerr.Errorf("DB %s@%s Migration %v doesn't match for between code & database: Down SQL. DB: `%s`\nCode: `%s`",
				service, sd.DBCfg.DBName, codeMig.Version, dbMig.Down, codeMig.Down)
		}
	}
	return nil
}

func getTenantDBData(ctx context.Context, uv universe.Universe) (*migrate.ServiceData, error) {
	sd, err := tenantdb.GetServiceData(ctx, uv)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	if err := sd.Migrations.Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return sd, nil
}

func migrateTenants(ctx context.Context, companyConfigSD, serviceDataTenantDB *migrate.ServiceData, downgradeVersion int) error {
	db, err := ucdb.New(ctx, companyConfigSD.DBCfg, noopValidator{})
	if err != nil {
		return ucerr.Wrap(err)
	}

	storage, err := companyconfig.NewStorage(ctx, db, nil)
	if err != nil {
		return ucerr.Wrap(err)
	}

	pager, err := companyconfig.NewTenantPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return ucerr.Wrap(err)
	}

	for {
		tenants, respFields, err := storage.ListTenantsPaginated(ctx, *pager)
		if err != nil {
			return ucerr.Wrap(err)
		}

		// TODO: how important is it to get the count up front? This is not as useful now.
		uclog.Infof(ctx, "Migrating tenants: Got %d tenants to migrate...", len(tenants))

		for i, t := range tenants {
			ti, err := storage.GetTenantInternal(ctx, t.ID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					// We have a row in the tenants table but are missing a row in the tenant IDP table
					//TODO: let the caller know we had issues.
					uclog.Infof(ctx, "Migrating %v: Couldn't find tenants_internal row for tenant (%s), Skipped", t.ID, t.Name)
					continue
				} else {
					return ucerr.Wrap(err)
				}
			}

			if err := ti.Validate(); err != nil {
				// We failed to validate the tenant config which means this tenant is malformed in some way
				// We have a row in the tenants table but are missing a row in the tenant IDP table
				uclog.Infof(ctx, "Migrating %v: Couldn't validate per tenant config for tenant (%s) - %v, skipping", t.ID, t.Name, err)
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
			if err := migrateDB(ctx, tenantIDStr, &tenantDBData, downgradeVersion); err != nil {
				return ucerr.Wrap(err)
			}
			// Not supporting  tenant log DB down migrations for now.
			if err := migrateDB(ctx, tenantIDStr, &logDBData, -1); err != nil {
				return ucerr.Wrap(err)
			}
		}
		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}
	return nil
}
