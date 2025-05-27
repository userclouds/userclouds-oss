package provisioning

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/provisioning/types"
)

const (
	// ServiceNameForTenantDBSecret is the service name used when creating a secret to store the tenant DB password.
	// We want to use the same scope (which is service in most cases) regardless of the
	// actual scope/service that actually provisions the tenant DB.
	ServiceNameForTenantDBSecret = "console"
)

type noopValidator struct{}

// Validate implements ucdb.Validate but does nothing, since we want to
// connect to an existing DB just to create a new, separate DB.
// We explicitly don't support this in a more global scope to discourage
// accidental "temporary" use.
func (n noopValidator) Validate(_ context.Context, _ *ucdb.DB) error {
	return nil
}

// ProvisionableDB manages the provisioning lifecycle of a database + cres
type ProvisionableDB struct {
	types.Named
	types.NoopClose
	types.Parallelizable
	bootstrapDBCfg   *ucdb.Config
	ProvisionedDBCfg *ucdb.Config

	schema         migrate.Schema
	baselineSchema *migrate.Schema

	// save this for later callers, and it becomes their responsbility to close it
	DB *ucdb.DB
}

// Because provisioning connections are not reused (instead multitenant cache will open new ones), we need to close them so they don't
// hang around either during running "provision" or in a long running console process
func closeDBConnection(ctx context.Context, db *ucdb.DB) {
	if err := db.Close(ctx); err != nil {
		uclog.Errorf(ctx, "Failed to close provisioning db connection %v", *db)
	}
}

// NewProvisionableDBFromExistingConfigs sets up a ProvisionableDB from existing configs
func NewProvisionableDBFromExistingConfigs(name string,
	bootstrapDBCfg, provisionedDBCfg, overrideBootstrapDBCfg *ucdb.Config,
	schema migrate.Schema, baselineSchema *migrate.Schema) *ProvisionableDB {

	// if we are explicitly overriding the bootstrap DB, we do that here
	// TODO: this whole system is a bit hacky, but this is particularly objectionable since
	// we use the "automatically generated" DBName, username, and password, but we overwrite the
	// cluster (and because of cdb DB-on-cluster naming conventions, we mess with DBName "sort of")
	localProvDBCfg := *provisionedDBCfg
	if overrideBootstrapDBCfg != nil {
		localProvDBCfg.Host = overrideBootstrapDBCfg.Host
		localProvDBCfg.Port = overrideBootstrapDBCfg.Port
		localProvDBCfg.DBDriver = overrideBootstrapDBCfg.DBDriver
		localProvDBCfg.DBProduct = overrideBootstrapDBCfg.DBProduct
		localProvDBCfg.DBName = fmt.Sprintf("%s%s", getClusterPrefixFromFullName(overrideBootstrapDBCfg.DBName), getDBNameFromFullName(provisionedDBCfg.DBName))

		// we also need to override bootstrapDBCfg so we create the user etc on the right cluster
		bootstrapDBCfg = overrideBootstrapDBCfg
	}

	return &ProvisionableDB{
		Named:            types.NewNamed(name),
		Parallelizable:   types.NewParallelizable(),
		bootstrapDBCfg:   bootstrapDBCfg,
		ProvisionedDBCfg: &localProvDBCfg,
		schema:           schema,
		baselineSchema:   baselineSchema,
	}
}

// GetTenantDBPasswordSecretName returns the name of the secret that stores the password for a tenant DB
func GetTenantDBPasswordSecretName(dbName string) string {
	return fmt.Sprintf("%s-dbpassword", dbName)
}

// NewProvisionableDB sets up a ProvisionableDB
func NewProvisionableDB(ctx context.Context,
	friendlyName string,
	bootstrapDBCfg *ucdb.Config,
	userName, dbName string,
	schema migrate.Schema,
	baselineSchema *migrate.Schema) (*ProvisionableDB, error) {

	provisionedDBCfg := *bootstrapDBCfg

	provisionedDBCfg.User = userName
	password, err := provisionedDBCfg.Password.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if !provisionedDBCfg.IsLocal() && password != "" {
		s, err := secret.NewString(ctx, ServiceNameForTenantDBSecret, GetTenantDBPasswordSecretName(dbName), crypto.MustRandomHex(24))
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		provisionedDBCfg.Password = *s
	}

	// If there's a cluster name prepended to the DB name, e.g.
	// "honey-badger-123.foo", we'll need to prepend it to the new DB Name in the
	// config, e.g. "honey-badger-123.bar".
	provisionedDBCfg.DBName = fmt.Sprintf("%s%s", getClusterPrefixFromFullName(provisionedDBCfg.DBName), dbName)
	return NewProvisionableDBFromExistingConfigs(friendlyName,
		bootstrapDBCfg,
		&provisionedDBCfg,
		nil, // there are no overrides for new DBs (`cmd/provision` uses the existing config path from files)
		schema,
		baselineSchema), nil
}

func getDBNameFromFullName(fullDBName string) string {
	// Extract DB name if there is a cluster name before it (used for hosted Cockroach clusters).
	if strings.Contains(fullDBName, ".") {
		parts := strings.Split(fullDBName, ".")
		return parts[1]
	}
	return fullDBName
}

func getClusterPrefixFromFullName(fullDBName string) string {
	// Extract cluster name if one is present (used for hosted Cockroach clusters).
	if strings.Contains(fullDBName, ".") {
		parts := strings.Split(fullDBName, ".")
		return parts[0] + "."
	}
	return ""
}

func validateDriver(db *ucdb.DB) error {
	if db.DriverName() != ucdb.PostgresDriver {
		return ucerr.Errorf("unrecognized DB driver: %s", db.DriverName())
	}
	return nil

}

// Provision implements Provisionable
func (pdb *ProvisionableDB) Provision(ctx context.Context) error {
	uclog.Infof(ctx, "[%s] provisioning new DB...", pdb.Name())

	bootstrapDB, err := ucdb.New(ctx, pdb.bootstrapDBCfg, noopValidator{})
	if err != nil {
		uclog.Errorf(ctx, "[%s] failed to connect to bootstrap DB '%s@%s': %v", pdb.Name(), pdb.bootstrapDBCfg.User, pdb.bootstrapDBCfg.DBName, err)
		return ucerr.Wrap(err)
	}
	defer closeDBConnection(ctx, bootstrapDB)
	if err := validateDriver(bootstrapDB); err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "[%s] connected to bootstrap DB '%s@%s', type '%s'", pdb.Name(), pdb.bootstrapDBCfg.User, pdb.bootstrapDBCfg.DBName, bootstrapDB.DriverName())

	// Check if the user already exists
	qU := `/* lint-sql-ok */ SELECT COUNT(usename) FROM pg_user WHERE usename=$1;`
	var uCount int
	err = bootstrapDB.GetContext(ctx, "Provision", &uCount, qU, pdb.ProvisionedDBCfg.User)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		uclog.Errorf(ctx, "[%s] failed to check existence of user '%s': %v", pdb.Name(), pdb.ProvisionedDBCfg.User, err)
		return ucerr.Wrap(err)
	} else if err == nil && uCount == 0 {
		provisionedDBPassword, err := pdb.ProvisionedDBCfg.Password.Resolve(ctx)
		if err != nil {
			return ucerr.Wrap(err)
		}
		// Create a user (quotes needed around user names because log DB users are all digits).
		q := fmt.Sprintf(`CREATE USER "%s"`, pdb.ProvisionedDBCfg.User)
		if provisionedDBPassword != "" {
			uclog.Infof(ctx, "[%s] creating new user '%s' with password", pdb.Name(), pdb.ProvisionedDBCfg.User)
			// Explicitly use the un-obfuscated password here.
			q = fmt.Sprintf("%s LOGIN PASSWORD '%s';", q, provisionedDBPassword)
		} else {
			uclog.Infof(ctx, "[%s] creating user '%s' without password", pdb.Name(), pdb.ProvisionedDBCfg.User)
		}

		if _, err := bootstrapDB.ExecContext(ctx, "Provision", q); err != nil {
			uclog.Errorf(ctx, "[%s] failed to create new user '%s': %v", pdb.Name(), pdb.ProvisionedDBCfg.User, err)
			return ucerr.Wrap(err)
		}
	} else {
		uclog.Infof(ctx, "[%s] User already exists, so not re-creating '%s'", pdb.Name(), pdb.ProvisionedDBCfg.User)
	}

	dbName := getDBNameFromFullName(pdb.ProvisionedDBCfg.DBName)

	// Check  existence of the database and create if needed
	qD := `/* lint-sql-ok */ SELECT COUNT(datname) FROM pg_database WHERE datname=$1`
	var dCount int
	err = bootstrapDB.GetContext(ctx, "Provision", &dCount, qD, dbName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		uclog.Errorf(ctx, "[%s] failed to check existence of database '%s': %v", pdb.Name(), pdb.ProvisionedDBCfg.DBName, err)
		return ucerr.Wrap(err)
	} else if err == nil && dCount == 0 {

		uclog.Infof(ctx, "[%s] creating database '%s'", pdb.Name(), pdb.ProvisionedDBCfg.DBName)
		additionalDBParams := ""
		if pdb.ProvisionedDBCfg.DBProduct == ucdb.AWSAuroraPostgres {
			additionalDBParams = " WITH ENCODING UTF8 LC_COLLATE 'C' LC_CTYPE 'C' TEMPLATE template0"
		}
		s := fmt.Sprintf(`CREATE DATABASE %s %s;`, dbName, additionalDBParams)
		if _, err := bootstrapDB.ExecContext(ctx, "Provision", s); err != nil {
			uclog.Errorf(ctx, "[%s] failed to create new database '%s': %v", pdb.Name(), pdb.ProvisionedDBCfg.DBName, err)
			return ucerr.Wrap(err)
		}
	} else {
		uclog.Infof(ctx, "[%s] Database already exists, so not re-creating '%s'", pdb.Name(), pdb.ProvisionedDBCfg.DBName)
	}

	uclog.Infof(ctx, "[%s] granting privileges to user '%s' on database '%s'", pdb.Name(), pdb.ProvisionedDBCfg.User, pdb.ProvisionedDBCfg.DBName)
	// TODO: can we scope these permissions down?
	p := fmt.Sprintf(`GRANT ALL PRIVILEGES ON DATABASE %s TO "%s"`, dbName, pdb.ProvisionedDBCfg.User)
	if _, err := bootstrapDB.ExecContext(ctx, "Provision", p); err != nil {
		return ucerr.Wrap(err)
	}

	// We use == postgres instead of .IsProductPostgres() because this doesn't apply to eg. Aurora
	if pdb.bootstrapDBCfg.DBProduct == ucdb.Postgres {
		cfg := *pdb.bootstrapDBCfg
		cfg.DBName = dbName
		tempDB, err := ucdb.New(ctx, &cfg, noopValidator{})
		if err != nil {
			return ucerr.Wrap(err)
		}
		p = fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO "%s"; GRANT CREATE ON SCHEMA public TO "%s";`, pdb.ProvisionedDBCfg.User, pdb.ProvisionedDBCfg.User)
		if _, err := tempDB.ExecContext(ctx, "Provision", p); err != nil {
			return ucerr.Wrap(err)
		}
		pdb.ProvisionedDBCfg.DBProduct = ucdb.Postgres
	}

	if pdb.ProvisionedDBCfg.DBProduct == ucdb.AWSAuroraPostgres {
		// We use Aurora Postgres 15.4, and PostgreSQL 15 revokes the CREATE permission from all users except a database owner from the public (or default) schema
		// see https://www.postgresql.org/about/news/postgresql-15-released-2526/
		p = fmt.Sprintf(`GRANT "%s" TO "%s"`, pdb.ProvisionedDBCfg.User, pdb.bootstrapDBCfg.User)
		if _, err := bootstrapDB.ExecContext(ctx, "Provision", p); err != nil {
			return ucerr.Wrap(err)
		}

		p = fmt.Sprintf(`ALTER DATABASE %s OWNER TO "%s"`, dbName, pdb.ProvisionedDBCfg.User)
		if _, err := bootstrapDB.ExecContext(ctx, "Provision", p); err != nil {
			return ucerr.Wrap(err)
		}

		p = fmt.Sprintf(`REVOKE "%s" FROM "%s"`, pdb.ProvisionedDBCfg.User, pdb.bootstrapDBCfg.User)
		if _, err := bootstrapDB.ExecContext(ctx, "Provision", p); err != nil {
			return ucerr.Wrap(err)
		}
	}

	time.Sleep(5 * time.Second) // give the DB a moment to propagate

	// Don't validate because we haven't migrated yet
	uclog.Debugf(ctx, "[%s] connecting to new DB '%s'", pdb.Name(), pdb.ProvisionedDBCfg.DBName)
	newDB, err := ucdb.New(ctx, pdb.ProvisionedDBCfg, noopValidator{})
	if err != nil {
		uclog.Errorf(ctx, "[%s] failed to connect to new DB '%s@%s': %v", pdb.Name(), pdb.ProvisionedDBCfg.User, pdb.ProvisionedDBCfg.DBName, err)
		return ucerr.Wrap(err)
	}
	// defer closeDBConnection(ctx, newDB)
	pdb.DB = newDB

	uclog.Infof(ctx, "[%s] creating migrations table on DB '%s'", pdb.Name(), pdb.ProvisionedDBCfg.DBName)
	if err := migrate.CreateMigrationsTable(ctx, newDB); err != nil {
		return ucerr.Wrap(err)
	}

	if pdb.ProvisionedDBCfg.IsProductPostgres() {
		if err := migrate.EnablePostgresExtensions(ctx, newDB, pdb.schema.PostgresOnlyExtensions); err != nil {
			return ucerr.Wrap(err)
		}
	}

	currentVersion, err := migrate.GetMaxVersion(ctx, newDB)
	if err != nil {
		uclog.Infof(ctx, "[%s] failed to get current migration version for DB '%s': %v", pdb.Name(), pdb.ProvisionedDBCfg.DBName, err)
		return ucerr.Wrap(err)
	}

	// short circuit migrations if we're starting from scratch, much faster
	// keep the migrate path in case we happen to call provision on a not-fully-migrated DB
	// TODO once we migrate logdb to Schemas, we can remove this nil check and the Schema{} creations in logdb.go
	if currentVersion == -1 {
		if pdb.schema.CreateStatements != nil && !types.UseBaselineSchema {
			uclog.Infof(ctx, "[%s] creating tables on DB from schema", pdb.Name())
			if err := pdb.schema.Apply(ctx, newDB, pdb.ProvisionedDBCfg.DBProduct); err != nil {
				return ucerr.Wrap(err)
			}
		} else if types.UseBaselineSchema && pdb.baselineSchema != nil {
			uclog.Infof(ctx, "[%s] creating tables on DB from baseline schema", pdb.Name())
			// we use this to test postgres compatibility in CI of the most recent migrations
			if err := pdb.baselineSchema.Apply(ctx, newDB, pdb.ProvisionedDBCfg.DBProduct); err != nil {
				return ucerr.Wrap(err)
			}
		} else {
			uclog.Infof(ctx, "[%s] no schema to apply, will create DB by running all migrations", pdb.Name())
		}
	}

	// always run this because it's "free" if we created all the way to schema, and we need it for the baseline schema
	// and for partially provisioned tenants
	currentVersion, err = migrate.GetMaxVersion(ctx, newDB)
	if err != nil {
		uclog.Infof(ctx, "[%s] failed to get current migration version for DB '%s': %v", pdb.Name(), pdb.ProvisionedDBCfg.DBName, err)
		return ucerr.Wrap(err)
	}
	maxAvail := pdb.schema.Migrations.GetMaxAvailable()
	uclog.Infof(ctx, "[%s] database '%s' current migration version %d, migrating to %d", pdb.Name(), pdb.ProvisionedDBCfg.DBName, currentVersion, maxAvail)
	if err := pdb.schema.Migrations.DoMigration(ctx, newDB, currentVersion, maxAvail); err != nil {
		return ucerr.Wrap(err)
	}

	uclog.Infof(ctx, "[%s] successfully provisioned new DB!", pdb.Name())
	return nil
}

// Validate implements Provisionable and Validateable
func (pdb *ProvisionableDB) Validate(ctx context.Context) error {
	if err := pdb.ProvisionedDBCfg.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	db, err := ucdb.New(ctx, pdb.bootstrapDBCfg, noopValidator{})
	if err != nil {
		return ucerr.Wrap(err)
	}
	defer closeDBConnection(ctx, db)
	if err := validateDriver(db); err != nil {
		return ucerr.Wrap(err)
	}
	// NB: we don't check for user existence here because
	// a) the bootstrap cfg user doesn't currently have permissions on the system.users table, and
	// b) the privileges check does check user existence for us as well
	var dbNames []string
	const showDBQueryPostgres = `/* bypass-known-table-check */ SELECT datname FROM pg_database WHERE datistemplate = false;`
	if err := db.SelectContext(ctx, "Validate", &dbNames, showDBQueryPostgres); err != nil {
		return ucerr.Errorf("failed to list databases on %s at %s: %w", pdb.bootstrapDBCfg.DBName, pdb.bootstrapDBCfg.Host, err)
	}

	dbName := getDBNameFromFullName(pdb.ProvisionedDBCfg.DBName)
	if !slices.Contains(dbNames, dbName) {
		return ucerr.Errorf("database '%s' not found", pdb.ProvisionedDBCfg.DBName)
	}
	return nil
}

// Cleanup implements Provisionable
func (pdb *ProvisionableDB) Cleanup(ctx context.Context) error {
	// TODO: implement soft-deletion for databases. For now, just orphan them.
	return nil
}

// Nuke will hard-delete all resources.
func (pdb *ProvisionableDB) Nuke(ctx context.Context) error {
	// Extra sanity check - don't nuke DBs outside of Dev. This is already checked in the CLI
	// but since this is the most destructive operation, let's check again here.
	uv := universe.Current()
	if !uv.IsDev() {
		return ucerr.Errorf("cannot nuke resources except in Dev universe: %v", uv)
	}

	// TODO: call Cleanup() depending on what goes in it?

	uclog.Infof(ctx, "[%s] nuking DB...", pdb.Name())
	db, err := ucdb.New(ctx, pdb.bootstrapDBCfg, noopValidator{})
	if err != nil {
		uclog.Errorf(ctx, "[%s] failed to connect to bootstrap DB '%s@%s': %v", pdb.Name(), pdb.bootstrapDBCfg.User, pdb.bootstrapDBCfg.DBName, err)
		return ucerr.Wrap(err)
	}
	defer closeDBConnection(ctx, db)
	if err := validateDriver(db); err != nil {
		return ucerr.Wrap(err)
	}

	dbName := getDBNameFromFullName(pdb.ProvisionedDBCfg.DBName)

	// One weird thing in cockroach (maybe postgres?) is that the DB owner ('userclouds') doesn't own
	// the tables inside, and can't drop the database as a result.
	// https://www.cockroachlabs.com/docs/v21.1/grant#grant-privileges-on-all-tables-in-a-database-or-schema
	if pdb.ProvisionedDBCfg.Validate() == nil {
		uclog.Debugf(ctx, "[%s] granting ALL privileges to bootstrap user '%s' on provisioned database '%s' so it can be dropped", pdb.Name(), pdb.bootstrapDBCfg.User, pdb.ProvisionedDBCfg.DBName)
		if dbByOwner, err := ucdb.New(ctx, pdb.ProvisionedDBCfg, noopValidator{}); err == nil {
			defer closeDBConnection(ctx, dbByOwner)
			p := fmt.Sprintf(`GRANT ALL PRIVILEGES ON TABLE %s.* TO "%s"`, dbName, pdb.bootstrapDBCfg.User)
			if _, err := dbByOwner.ExecContext(ctx, "Provision", p); err != nil {
				return ucerr.Wrap(err)
			}
		}
	}

	uclog.Infof(ctx, "[%s] attempting to drop database '%s'", pdb.Name(), pdb.ProvisionedDBCfg.DBName)
	dropDatabaseQ := fmt.Sprintf(`DROP DATABASE IF EXISTS %s;`, dbName)
	if _, err := db.ExecContext(ctx, "Provision", dropDatabaseQ); err != nil {
		uclog.Errorf(ctx, "[%s] failed to drop database '%s': %v", pdb.Name(), pdb.ProvisionedDBCfg.DBName, err)
		return ucerr.Wrap(err)
	}

	uclog.Infof(ctx, "[%s] attempting to drop database user '%s'", pdb.Name(), pdb.ProvisionedDBCfg.User)
	dropUserQ := fmt.Sprintf(`DROP USER IF EXISTS "%s"`, pdb.ProvisionedDBCfg.User)
	if _, err := db.ExecContext(ctx, "Provision", dropUserQ); err != nil {
		uclog.Errorf(ctx, "[%s] failed to drop database user '%s': %v", pdb.Name(), pdb.ProvisionedDBCfg.User, err)
		return ucerr.Wrap(err)
	}

	uclog.Infof(ctx, "[%s] successfully nuked DB!", pdb.Name())
	return nil
}
