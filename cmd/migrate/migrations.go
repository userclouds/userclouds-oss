package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"userclouds.com/infra/migrate"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/cmdline"
)

// is this a safe universe to auto-downgrade? or upgrading w/o prompting the user (if --no-prompt is passed)
func safeUniverse() bool {
	return !universe.Current().IsProdOrStaging()
}

// safeCurrent and safeRequest is a bit of a hack to make per-tenant migrations work easier
// If currentVersion & requestedVersion both match, we don't prompt for which version to migrate to
// suppressConfirmation is a similar system (also a hack) but it's to suppress the "are you sure?" prompt
// it's a pointer so that migrateDatabase can accept "all" and set it. Note that nil is safe (and == false)
func migrateDatabase(
	ctx context.Context,
	serviceName string,
	sd migrate.ServiceData,
	safeCurrent int,
	safeRequest int,
	suppressConfirmation *bool) (current, requested int) {
	// TODO: can we unify this with multitenant to manage DB connections etc?
	db, err := ucdb.New(ctx, sd.DBCfg, noopValidator{})
	if err != nil {
		uclog.Fatalf(ctx, "failed to init db %v: %v", sd.DBCfg.DBName, err)
	}

	if err := migrate.CreateMigrationsTable(ctx, db); err != nil {
		uclog.Fatalf(ctx, "failed to create migrations table: %v", err)
	}

	currentVersion, err := migrate.GetMaxVersion(ctx, db)
	if err != nil {
		uclog.Fatalf(ctx, "failed to get max version from migrations: %v", err)
	}

	maxAvail := sd.Migrations.GetMaxAvailable()
	requestedVersion := maxAvail
	if safeCurrent != currentVersion || requestedVersion != safeRequest {
		uclog.Infof(ctx, "Database %s for %s is currently at version: %d", sd.DBCfg.DBName, serviceName, currentVersion)
		uclog.Infof(ctx, "  Max available migration is: %d", maxAvail)

		// if we're running noDowngradePrompt (because we're running in make devsetup or make provision-dev),
		// we don't give the user the option to migrate down (they can run make migrate-dev for that if needed)
		if currentVersion == maxAvail && *noDowngradePrompt {
			requestedVersion = currentVersion
		} else if suppressConfirmation != nil && *suppressConfirmation && safeUniverse() {
			// if we're in dev and we said "all", just do all even if it's a downgrade
			requestedVersion = safeRequest
			uclog.Debugf(ctx, "requested 'all' in %v for %v, so migrating to %d", universe.Current(), serviceName, requestedVersion)
		} else {
			if rv := promptUser(ctx, fmt.Sprint(maxAvail), "What version would you like to migrate to? [%d] ", maxAvail); rv == "" {
				requestedVersion = maxAvail
			} else {
				requestedVersion, err = strconv.Atoi(rv)
				if err != nil {
					uclog.Fatalf(ctx, "invalid version %s: %v", rv, err)
				}
			}
		}
	}

	// this is safe, so don't prompt to confirm before exiting :)
	if requestedVersion == currentVersion {
		uclog.Infof(ctx, "DB: %v requested version (%d) matches existing version (%d) ... that was easy.", sd.DBCfg.DBName, requestedVersion, currentVersion)
		return currentVersion, requestedVersion
	}

	// if nil, there's no all option
	// NB: suppressing the confirmation prompt only works on the golden path, that is, expected
	// upgrade to max. downgrades will still be annoying (by design?), except in safeUniverses().
	// The safeCurrent check for -1 means we still allow you to select all on the first confirmation prompt
	if suppressConfirmation == nil || // if we haven't answered yet
		((safeCurrent != currentVersion || // if the current DB is in a different state from the last one
			(requestedVersion != maxAvail && // or we're not upgrading to max
				!safeUniverse())) && // unless we're in a safe universe, then allow auto-downgrades or partial-upgrades
			safeCurrent != -1) { // or if we're on the first confirmation prompt, hit the next else condition
		if !suppressPrompt() && !cmdline.Confirm(ctx, "Migrate %v from %d -> %d? [yN] ", serviceName, currentVersion, requestedVersion) {
			uclog.Fatalf(ctx, "ok, aborting migration")
		}
	} else if !*suppressConfirmation {
		// if not nil, and false, prompt and offer all option
		i := promptUser(ctx, "all", "Migrate %v from %d -> %d? [y | N | all] ", serviceName, currentVersion, requestedVersion)
		if strings.EqualFold(i, "all") {
			*suppressConfirmation = true
		} else if !strings.EqualFold(i, "y") {
			uclog.Fatalf(ctx, "ok, aborting migration")
		}
	}

	// ensure that extant migrations in the DB (if any) match what we think we've already applied
	existingMigs, err := migrate.SelectMigrations(ctx, db)
	if err != nil {
		uclog.Fatalf(ctx, "failed to select migrations from DB: %v", err)
	}

	// only check up to the requested version, since we might know about this and be trying to downgrade
	var codeMigrationIndex int
	for i := 0; i < len(existingMigs) && codeMigrationIndex < len(sd.Migrations) && i < requestedVersion; i++ {
		codeMig := sd.Migrations[codeMigrationIndex]
		dbMig := existingMigs[i]

		// since we truncated the first part of the code migrations on 3/21/23, we can't check
		// until we have matching versions
		if dbMig.Version < codeMig.Version {
			continue
		}

		codeMigrationIndex++
		// only check table, up, down
		if codeMig.Table == dbMig.Table && migrate.CompareMigrations(dbMig.Up, codeMig.Up) && migrate.CompareMigrations(dbMig.Down, codeMig.Down) {
			continue
		}

		// check deprecated up/downs if needed, as long as table matches
		// this logic is pretty convoluted, but we fall back to checking the relevant deprecated
		// queries for either/both up/down that didn't match
		if codeMig.Table == dbMig.Table {
			var deprecatedMatch bool

			// if up doesn't match, check for matches in old migrations but if up matches, default to true
			if !migrate.CompareMigrations(dbMig.Up, codeMig.Up) {
				deprecatedMatch = migrate.CompareMigrations(dbMig.Up, codeMig.DeprecatedUp...)
			} else {
				deprecatedMatch = true
			}

			// if the ups did match (whether through codeMig.Up == dbMig.Up, or a deprecatedup), check the downs
			// if the downs didn't match. No else clause necessary here because if codeMig.Down == dbMig.Down,
			// then the current value of deprecatedMatch is all we need
			if deprecatedMatch && !migrate.CompareMigrations(dbMig.Down, codeMig.Down) {
				// if the downs didn't match, reset to false unless we find one here
				deprecatedMatch = migrate.CompareMigrations(dbMig.Down, codeMig.DeprecatedDown...)
			}
			if deprecatedMatch {
				continue
			}
		}

		uclog.Warningf(ctx, "migration %d in DB don't match code: %v != %v", i, dbMig, codeMig)
		// never allow this in prod. Debug or even staging might get this, and dev all the time
		if universe.Current().IsProd() {
			uclog.Fatalf(ctx, "cannot continue, this is a production environment ... you'll need to fix this manually")
		}

		// if only the downs don't match, let's just update for you :)
		if codeMig.Table == dbMig.Table && migrate.CompareMigrations(dbMig.Up, codeMig.Up) {
			uclog.Debugf(ctx, "only down migrations don't match, so we can automatically fix that")
			if !cmdline.Confirm(ctx, "update down migration %d in DB to match code? [yN] ", i) {
				uclog.Warningf(ctx, "skipping update of down migration %d, you will continue to get warnings", i)
				continue
			}

			// we have to delete then re-save since we normally don't allow updating
			uclog.Infof(ctx, "updating down migration %d in DB to match code", i)
			if err := migrate.DeleteMigration(ctx, db, codeMig.Version); err != nil {
				uclog.Fatalf(ctx, "failed to delete migration %d: %v", i, err)
			}
			if err := migrate.SaveMigration(ctx, db, &codeMig); err != nil {
				uclog.Fatalf(ctx, "failed to update migration %d: %v", i, err)
			}
			continue
		}

		uclog.Warningf(ctx, "you probably want to downgrade below this migration and then upgrade again")
		if !cmdline.Confirm(ctx, "continue anyway? [yN] ") {
			uclog.Fatalf(ctx, "aborting migration")
		}

	}

	uclog.Infof(ctx, "migrating %v from %d to %d", sd.DBCfg.DBName, currentVersion, requestedVersion)
	start := time.Now().UTC()
	if requestedVersion > currentVersion {
		if currentVersion == -1 {
			if requestedVersion < sd.BaselineVersion {
				uclog.Fatalf(ctx, "cannot migrate to version %d, it is below the baseline version %d", requestedVersion, sd.BaselineVersion)
			}
			for _, sql := range sd.BaselineCreateStatements {
				if _, err := db.ExecContext(ctx, "migrateDatabase", sql); err != nil {
					uclog.Fatalf(ctx, "error executing sql (%s): %v", sql, err)
				}
			}

			currentVersion = sd.BaselineVersion
		}

		// upgrade path, where we use the migrations defined in code (since they haven't been
		// used in the database yet, by definition)
		err = sd.Migrations.DoMigration(ctx, db, currentVersion, requestedVersion)
	} else {
		// downgrade path, where we use the stored migrations from the database in case we
		// switched branches and don't have the later migrations in code (if they haven't been
		// merged yet, etc)

		checkSafeDowngrade(ctx, currentVersion, requestedVersion, sd.Migrations, existingMigs)

		if *codeMigrateFlag {
			checkCodeMigrate(ctx)
			err = sd.Migrations.DoMigration(ctx, db, currentVersion, requestedVersion)
		} else {
			err = existingMigs.DoMigration(ctx, db, currentVersion, requestedVersion)
		}
	}
	if err != nil {
		uclog.Fatalf(ctx, "could not migrate from %d -> %d: %v", currentVersion, requestedVersion, err)
	}
	took := time.Now().UTC().Sub(start)
	uclog.Infof(ctx, "migration of %v %v from %d -> %d finished successfully. took: %v", serviceName, sd.DBCfg.DBName, currentVersion, requestedVersion, took)
	return currentVersion, requestedVersion
}

func suppressPrompt() bool {
	return *noPrompt && safeUniverse()
}

func promptUser(ctx context.Context, defaultResponse, message string, args ...any) string {
	if suppressPrompt() {
		return defaultResponse
	}
	return cmdline.ReadConsole(ctx, message, args...)
}
