package migrate

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// Schema defines a database schema with the migrations to get there,
// as well as a "cached" copy of the CREATE TABLE statements in a
// fully-migrated state, and a list of required columns per-table
// (for validation)
type Schema struct {
	Migrations             Migrations
	CreateStatements       []string
	Columns                map[string][]string
	PostgresOnlyExtensions []string
}

// Validate implements Validateable
func (s Schema) Validate() error {
	if err := s.Migrations.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if len(s.CreateStatements) == 0 {
		return ucerr.Errorf("can't use Schema without create statements")
	}
	return nil
}

// TODO This is used to limit concurrent CREATE INDEX calls to db
// Need better solution for this
var globalSchemaLock sync.Mutex

// EnablePostgresExtensions enables the provided list of extensions on the postgres DB
func EnablePostgresExtensions(ctx context.Context, db *ucdb.DB, extensions []string) error {
	for _, ext := range extensions {
		uclog.Infof(ctx, "Enable %s extension on postgres DB", ext)
		if _, err := db.ExecContext(ctx, fmt.Sprintf("EnableExtension-%s", ext), fmt.Sprintf(`CREATE EXTENSION IF NOT EXISTS %s;`, ext)); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}

// Apply applies a schema to the database
func (s Schema) Apply(ctx context.Context, db *ucdb.DB, product ucdb.DBProduct) error {
	if err := s.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	if db.DBProduct == ucdb.Postgres {
		if err := EnablePostgresExtensions(ctx, db, s.PostgresOnlyExtensions); err != nil {
			return ucerr.Wrap(err)
		}
	}

	globalSchemaLock.Lock()
	defer globalSchemaLock.Unlock()

	// we actually need to save the migrations so Connect will Validate
	if err := CreateMigrationsTable(ctx, db); err != nil {
		return ucerr.Wrap(err)
	}

	creates := s.CreateStatements
	uclog.Infof(ctx, "Applying %d create statements", len(creates))
	for _, stmt := range creates {
		// don't try to recreate the migrations table since we just did that
		// (we don't reorder this because during provisioning, we want to check
		// MaxVersion to decide on using Schema or Migrations)
		if strings.Contains(stmt, "public.migrations") {
			continue
		}

		if _, err := db.ExecContext(ctx, "Schema.Apply", stmt); err != nil {
			return ucerr.Wrap(err)
		}
	}
	uclog.Infof(ctx, "Updating migrations table %d migrations", len(s.Migrations))
	if err := SaveMigrations(ctx, db, s.Migrations); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
