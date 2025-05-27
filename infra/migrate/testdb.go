package migrate

import (
	"context"

	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// TestSchema defines a struct that knows how to get to latest instantly
type TestSchema struct {
	Schema Schema
}

// NewTestSchema creates a new TestSchema for test DBs. DO NOT USE IN PROD CODE.
func NewTestSchema(s Schema) TestSchema {
	return TestSchema{s}
}

// Apply implements testdb.Option
func (s TestSchema) Apply(db *ucdb.DB, product ucdb.DBProduct) error {
	ctx := context.Background()
	return ucerr.Wrap(s.Schema.Apply(ctx, db, product))
}

// TestMigrator defines a struct that knows how to migrate (always to latest)
type TestMigrator struct {
	migs Migrations
}

// NewTestMigrator returns a new Migrator for test DBs. DO NOT USE IN PROD CODE.
// Note: this is much slower than using NewTestSchema, but necessary in some cases
// like to actually generate schemas, or when we haven't generated them (eg. for MySQL)
func NewTestMigrator(migs Migrations) TestMigrator {
	return TestMigrator{migs: migs}
}

// Apply implements testdb.Option
// TODO: we should actually unify the apply-and-save code too (between here and cmd/migrate)
// NB: product is unused here since this is a literal statement-by-statement migration,
// not a schema migration that has indices integrated or separate depending on product
func (m TestMigrator) Apply(db *ucdb.DB, _ ucdb.DBProduct) error {
	ctx := context.Background()
	if err := m.migs.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	if err := CreateMigrationsTable(ctx, db); err != nil {
		return ucerr.Wrap(err)
	}
	for _, mig := range m.migs {
		if _, err := db.Exec(mig.Up); err != nil {
			return ucerr.Errorf("error applying %d.Up: %w", mig.Version, err)
		}
		if err := SaveMigration(ctx, db, &mig); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
