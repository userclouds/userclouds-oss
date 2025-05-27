package migrate

import (
	"context"
	"strings"

	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// SaveMigration saves a migration that was just executed to the DB
func SaveMigration(ctx context.Context, db *ucdb.DB, m *Migration) error {
	const q = `INSERT INTO migrations (tbl, version, dsc, up, down) VALUES ($1, $2, $3, $4, $5); /* lint-deleted */`
	if _, err := db.ExecContext(ctx, "SaveMigration", q, m.Table, m.Version, m.Desc, m.Up, m.Down); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// SaveMigrations does a bulk insert
// TODO: logdb doesn't use this yet
func SaveMigrations(ctx context.Context, db *ucdb.DB, ms []Migration) error {
	const q = `INSERT INTO migrations (tbl, version, dsc, up, down) VALUES (:tbl, :version, :dsc, :up, :down); /* lint-deleted */`
	if _, err := db.NamedExecContext(ctx, q, ms); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// GetMaxVersion gets the latest applied migration to this DB
func GetMaxVersion(ctx context.Context, db *ucdb.DB) (int, error) {
	const q = `SELECT MAX(version) FROM migrations; /* lint-deleted */`
	var m *int
	if err := db.GetContext(ctx, "GetMaxVersion", &m, q); err != nil {
		return -2, ucerr.Wrap(err)
	}

	if m == nil {
		return -1, nil
	}

	return *m, nil
}

// SelectMigrations loads all the migrations that have been applied to this DB
func SelectMigrations(ctx context.Context, db *ucdb.DB) (Migrations, error) {
	const q = `SELECT tbl, version, dsc, up, down FROM migrations ORDER BY version ASC; /* lint-deleted */`

	var ms Migrations
	if err := db.SelectContext(ctx, "SelectMigrations", &ms, q); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return ms, nil
}

// CreateMigrationsTable creates the migrations table. Lives here because useful in tests (migrate.NewTestMigrator()) too
func CreateMigrationsTable(ctx context.Context, db *ucdb.DB) error {
	const q = `CREATE TABLE IF NOT EXISTS migrations (tbl VARCHAR(1000) NOT NULL, version INT NOT NULL, dsc VARCHAR(1000), up VARCHAR(10000) NOT NULL, down VARCHAR(1000), PRIMARY KEY(version));`

	_, err := db.ExecContext(ctx, "CreateMigrationsTable", q)
	return ucerr.Wrap(err)
}

// DeleteMigration removes the migration (after a downgrade)
// TODO: we should probably soft-delete for safety here :)
func DeleteMigration(ctx context.Context, db *ucdb.DB, version int) error {
	const q = `DELETE FROM migrations WHERE version=$1;`
	_, err := db.ExecContext(ctx, "DeleteMigration", q, version)
	return ucerr.Wrap(err)
}

type table struct {
	TableName string `db:"table_name"`
}

// SelectTables loads all the tables currently in the database
// This is factored out for use in genschemas
func SelectTables(ctx context.Context, db *ucdb.DB) (map[string]struct{}, error) {
	const q = `SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'; /* lint: deleted-safe lint-system-table */`
	var tbls []table
	if err := db.SelectContext(ctx, "SelectTables", &tbls, q); err != nil {
		return nil, ucerr.Wrap(err)
	}

	// TODO: we should just implement a stringset type
	m := make(map[string]struct{})
	for _, t := range tbls {
		m[t.TableName] = struct{}{}
	}

	return m, nil
}

type columnSchema struct {
	TableName  string `db:"table_name"`
	ColumnName string `db:"column_name"`
}

// SelectColumns returns a list of the columns in a table
// Note that similar code is used in migrate_test and genschemas but they're
// all slightly different and not worth refactoring right now
func SelectColumns(ctx context.Context, db *ucdb.DB) (map[string][]string, error) {
	const q = `SELECT table_name, column_name FROM information_schema.columns ORDER BY column_name ASC; /* lint: deleted-safe lint-system-table */`

	var table []columnSchema
	if err := db.SelectContext(ctx, "SelectColumns", &table, q); err != nil {
		if strings.HasSuffix(err.Error(), "does not exist") {
			return nil, ucerr.New("table does not exist")
		}
		return nil, ucerr.Wrap(err)
	}

	cols := make(map[string][]string)
	for _, c := range table {
		cols[c.TableName] = append(cols[c.TableName], c.ColumnName)
	}

	return cols, nil
}
