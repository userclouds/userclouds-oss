package migrate

import (
	"context"
	"regexp"
	"time"

	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

const migrationDBTimeout = 60 * time.Second

var spaceRegex = regexp.MustCompile(`\s+`)

// Migration defines a single migration for a single database
//
//	Up defines the SQL from Version-1 -> Version, and
//	Down defines the SQL from Version -> Version-1
type Migration struct {
	Version int    `db:"version"`
	Table   string `db:"tbl"` // "table" is reserved in SQL
	Desc    string `db:"dsc"` // likewise, "desc" is reserved in SQL
	Up      string `db:"up"`
	Down    string `db:"down"`

	// These two exist to log old migrations that are safe, but no longer used
	// The specific initial use case for these is a 9/23 effort to make our
	// system safe to run on Postgres, but finding that we used some Cockroach-
	// specific annotations, that while not material, are throwing off our schema
	// checks during migration (we assert that the target database has the exact
	// same migrations as the code before migrating up, to protect against errors
	// like leaving old under-development migrations in place with conflicting version
	// numbers)
	DeprecatedUp   []string `db:"-"`
	DeprecatedDown []string `db:"-"`
}

// Equals returns true if the two migrations are equal
// We purposely ignore description & deprecated fields
func (m Migration) Equals(other Migration) bool {
	return m.Version == other.Version &&
		m.Table == other.Table &&
		m.Up == other.Up && m.Down == other.Down
}

// Migrations defines an ordered list of migrations for a single database.
type Migrations []Migration

// Validate implements Validateable
func (migs Migrations) Validate() error {
	if len(migs) == 0 {
		return nil
	}

	offset := migs[0].Version

	for i, v := range migs {
		if v.Version-offset != i {
			return ucerr.Errorf("migration[%d] has mismatched version (%d), desc: '%s'", i, v.Version, v.Desc)
		}
	}
	return nil
}

// GetMaxAvailable returns the highest indexed migration.
func (migs Migrations) GetMaxAvailable() int {
	maxAvail := -1
	for _, m := range migs {
		if m.Version > maxAvail {
			maxAvail = m.Version
		}
	}
	return maxAvail
}

// DoMigration applies migrations in the specified range to the given DB, and assumes
// the migration table already exists.
func (migs Migrations) DoMigration(ctx context.Context, db *ucdb.DB, current int, requested int) error {
	for current != requested {
		var m *Migration
		var newVersion int
		var sql string
		var err error
		if current < requested {
			// migrate up
			m, err = migs.Get(current + 1)
			if err != nil {
				return ucerr.Wrap(err)
			}
			sql = m.Up
			newVersion = m.Version
		} else {
			// note that we use current.Down here because we want to undo the current migration.
			m, err = migs.Get(current)
			if err != nil {
				return ucerr.Wrap(err)
			}
			sql = m.Down
			newVersion = m.Version - 1
		}

		// we adjust the timeout here and back just to be safe, eg when called from provisioning
		oldTimeout := db.Timeout()
		db.SetTimeout(migrationDBTimeout)
		uclog.Infof(ctx, "executing migration %d (%s)", m.Version, sql)
		if _, err := db.ExecContext(ctx, "DoMigration", sql); err != nil {
			db.SetTimeout(oldTimeout)
			return ucerr.Errorf("error executing migration %d (%s): %w", m.Version, sql, err)
		}
		db.SetTimeout(oldTimeout)

		// save this migration in the db in case we need to back it out later
		if newVersion > current {
			if err := SaveMigration(ctx, db, m); err != nil {
				return ucerr.Wrap(err)
			}
		} else {
			// clean up after ourselves by ensuring that this downgrade is
			// deleted from the database so we don't try to rerun them
			if err := DeleteMigration(ctx, db, m.Version); err != nil {
				return ucerr.Wrap(err)
			}
		}

		current = newVersion
	}

	return nil
}

// Get returns the migration at the given index.
func (migs Migrations) Get(v int) (*Migration, error) {
	for i := range migs {
		if migs[i].Version == v {
			m := migs[i]
			return &m, nil
		}
	}

	return nil, ucerr.Errorf("couldn't find migration with version %d", v)
}

// CompareMigrations compares a database migration to a list of code migrations
func CompareMigrations(dbMigration string, codeMigrations ...string) bool {
	// This is a bit of a hack, but it's the easiest way to compare the migrations without
	// throwing errors on whitespace, but also without erroring on descriptions
	normalizedBDMigration := spaceRegex.ReplaceAllString(dbMigration, " ")
	for _, codeMigration := range codeMigrations {
		if codeMigration == "" {
			continue
		}
		if spaceRegex.ReplaceAllString(codeMigration, " ") == normalizedBDMigration {
			return true
		}
	}
	return false
}
