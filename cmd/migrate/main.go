package main

import (
	"context"
	"flag"
	"fmt"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
)

type noopValidator struct{}

var codeMigrateFlag *bool
var checkDeployedVersionFlag *bool
var flagLogfile *string
var noDowngradePrompt *bool
var noPrompt *bool
var noUnsafeWarnInDev *bool
var verbose *bool

// Validate implements ucdb.Validate but does nothing, since we want to operate on non-current DBs
func (n noopValidator) Validate(_ context.Context, _ *ucdb.DB) error {
	return nil
}

func initFlags(ctx context.Context) {
	flag.Usage = func() {
		uclog.Infof(ctx, "usage: bin/migrate [flags] <database>")

		flag.VisitAll(func(f *flag.Flag) {
			uclog.Infof(ctx, "    %s: %v", f.Name, f.Usage) // f.Name, f.Value
		})
	}
	codeMigrateFlag = flag.Bool("code", false, "use down migrations from code as source of truth (instead of DB)")

	// running this inside the migrate binary makes sense for now
	// because it already understands per-service local migrations, etc.
	checkDeployedVersionFlag = flag.Bool("checkDeployed", false, fmt.Sprintf("check that '%s' universe's migrations are up to date with local code", universe.Current()))
	flagLogfile = flag.String("logfile", "", "logfile name for debug output")
	noDowngradePrompt = flag.Bool("noDowngradePrompt", false, "don't prompt to migrate if db==code, useful for devsetup etc")
	noPrompt = flag.Bool("noPrompt", false, "don't prompt user (non prod/staging only), implies -noDowngradePrompt")
	noUnsafeWarnInDev = flag.Bool("noUnsafeWarnInDev", false, "don't warn about unsafe migrations in dev")
	verbose = flag.Bool("verbose", false, "enable verbose output")

	flag.Parse()
	if *noPrompt {
		*noDowngradePrompt = true
	}
}

// takes universe from the env var, and the database name on the command line
func main() {
	ctx := context.Background()
	initFlags(ctx)
	var screenLogLevel uclog.LogLevel = uclog.LogLevelInfo
	if *verbose {
		screenLogLevel = uclog.LogLevelVerbose
	}

	logtransports.InitLoggerAndTransportsForTools(ctx, screenLogLevel, uclog.LogLevelVerbose, "migrate", logtransports.Filename(*flagLogfile))
	defer logtransports.Close()

	if flag.NArg() < 1 {
		flag.Usage()
		uclog.Fatalf(ctx, "error: expected at least one database name to be specified, got %d: %v", flag.NArg(), flag.Args())
	}
	uv := universe.Current()
	if *codeMigrateFlag && !uv.IsDev() {
		uclog.Fatalf(ctx, "error: -code flag can only be used in 'dev' universe")
	} else if *codeMigrateFlag {
		uclog.Infof(ctx, "Using code (instead of DB) as source of truth for DB migrations")
	}

	if *noUnsafeWarnInDev && !uv.IsDev() {
		uclog.Fatalf(ctx, "error: -noUnsafeWarnInDev flag can only be used in 'dev' universe")
	}
	if *noPrompt && !safeUniverse() {
		uclog.Fatalf(ctx, "error: -noPrompt flag can only be used in in a safe universe (not prod or staging)")
	}

	for _, dbName := range flag.Args() {
		uclog.Infof(ctx, "Migrating Database %s", dbName)
		dbData, err := getDatabaseData(ctx, uv, dbName)
		if err != nil {
			uclog.Fatalf(ctx, "couldn't get service data: %v", err)
		}
		if *checkDeployedVersionFlag {
			migrationURL, err := getMigrationURL(ctx, uv, dbName)
			if err != nil {
				uclog.Fatalf(ctx, "couldn't get migration URL for %s: %v", dbName, err)
			}
			verifyDeployedVersion(ctx, dbName, dbData, migrationURL)
		} else {
			if dbName == "tenantdb" {
				migrateTenants(ctx, uv)
			} else {
				sd := migrate.ServiceData{
					DBCfg:                    dbData.DBCfg,
					Migrations:               dbData.Migrations,
					BaselineVersion:          dbData.BaselineVersion,
					BaselineCreateStatements: dbData.BaselineCreateStatements,
				}
				migrateDatabase(ctx, dbName, sd, -1, -1, nil)
			}
		}
	}
}

func verifyDeployedVersion(ctx context.Context, dbName string, dbData *migrate.ServiceData, migrationURL string) {
	if dbName != "tenantdb" && dbName != "companyconfig" && dbName != "status" {
		uclog.Fatalf(ctx, "service/DB '%s' doesn't support --checkDeployed flag", dbName)
	} else if err := checkDeployedVersion(ctx, dbName, *dbData, migrationURL); err != nil {
		uclog.Fatalf(ctx, "'%s' universe migration check: %v", universe.Current(), err)
	}
}

// this function fatals if the user aborts
func checkSafeDowngrade(ctx context.Context, currentVersion, requestedVersion int, migsCode, migsDB migrate.Migrations) {
	safe := true
	for i := currentVersion; i >= requestedVersion; i-- {
		c, err := migsCode.Get(i)
		if err != nil {
			safe = false
			uclog.Infof(ctx, "migration %d isn't in your current codebase, so we'll rely on what's in the DB", i)
			uclog.Infof(ctx, "  if these two are somehow out of sync, you might have an issue")
			continue
		}
		d, err := migsDB.Get(i)
		if err != nil {
			safe = false // not needed with fatal, but in case of future change
			uclog.Fatalf(ctx, "couldn't get necessary migration %d from the database: %v", i, err)
		}
		if c != nil && d != nil && !c.Equals(*d) {
			safe = false
			uclog.Infof(ctx, "migration %d doesn't match between code & database", i)
			uclog.Infof(ctx, "code: %#v", c)
			uclog.Infof(ctx, "  db: %#v", d)
		}
	}
	if !safe && !*noUnsafeWarnInDev {
		if !cmdline.Confirm(ctx, "This downgrade could be unsafe ... proceed? [yN] ") {
			uclog.Fatalf(ctx, "ok, aborting migration")
		}
	}
}

const codeMigrateWarning = `This downgrade is overriding the migrations from the DB with those defined in code.
This may be dangerous because the code may have diverged unsafely since the DB was migrated.
Use this flag if you think the DB's migrations table is wrong and know what you're doing.
Proceed? [yN] `

func checkCodeMigrate(ctx context.Context) {
	if !cmdline.Confirm(ctx, codeMigrateWarning) {
		uclog.Fatalf(ctx, "ok, aborting migration")
	}
}
