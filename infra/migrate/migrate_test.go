package migrate_test

// package is migrate_test so we don't cause a circular import loading eg. idp.Migrations
// this is the same reason we can't reuse the per-service code from cmd/migrate (without adding
// yet another package, which we'll do one of these days)

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/logdb"
	"userclouds.com/internal/rootdb"
	"userclouds.com/internal/tenantdb"
)

type serviceData struct {
	migs migrate.Migrations
	cols map[string][]string
}

func TestMigrations(t *testing.T) {
	assertNoMigrationsTable(t, tenantdb.SchemaBaseline)
	assertNoMigrationsTable(t, companyconfig.SchemaBaseline)
	ms := make(map[string]serviceData)
	ms["tenantdb"] = serviceData{tenantdb.GetMigrations(), tenantdb.UsedColumns}
	ms["companyconfig"] = serviceData{companyconfig.GetMigrations(), companyconfig.UsedColumns}
	ms["logserver"] = serviceData{logdb.GetMigrations(), logdb.UsedColumns}

	// The rootdb migration test is the only one that creates & deletes databases with
	// fixed names, which means multiple instances can't run in parallel on the same cluster
	// (unlike every other test which generates test DBs with UUIDs in names, and unlike
	// local tests which use an ephemeral cluster per test).
	// TODO: this isn't the best possible solution.
	// TODO: we should also test rootdbstatus migrations here, but need parallel postgres testdb infra
	if universe.Current() != universe.CI {
		ms["rootdb"] = serviceData{rootdb.GetMigrations(), nil}
	}

	for service, sd := range ms {
		// capture loop vars
		service := service
		sd := sd
		t.Run(service, func(t *testing.T) {
			t.Parallel()
			var tdb *ucdb.DB
			if service == "tenantdb" {
				tdb = testdb.New(t, migrate.NewTestSchema(tenantdb.SchemaBaseline))
			} else if service == "companyconfig" {
				tdb = testdb.New(t, migrate.NewTestSchema(companyconfig.SchemaBaseline))
			} else {
				tdb = testdb.New(t)
			}
			testServiceMigrations(t, sd, tdb)
		})
	}
}

func assertNoMigrationsTable(t *testing.T, s migrate.Schema) {
	re := regexp.MustCompile(`CREATE TABLE.*migrations`)
	for _, createStatement := range s.CreateStatements {
		assert.False(t, re.MatchString(createStatement), assert.Errorf("migrations table found in schema: %s", createStatement))
	}
}

type migrationSchemas struct {
	Before string // represents the table schema before migration applied
	After  string // represents the table schema after migration applied
}

// TODO: remove this + checks before we actually launch as table renames will cause downtime
var reRenameTable = regexp.MustCompile("^ALTER TABLE ([a-zA-Z_]+) RENAME TO ([a-zA-Z_]+);$")

func strInList(s string, l []string) bool {
	return slices.Contains(l, s)
}

func testServiceMigrations(t *testing.T, sd serviceData, tdb *ucdb.DB) {
	ctx := context.Background()

	// we have to create this because we operate on it (once at least in companyconfig)
	assert.IsNil(t, migrate.CreateMigrationsTable(ctx, tdb))

	// first figure out how many different tables we're operating on
	tables := make(map[string]any)
	for _, m := range sd.migs {
		tables[m.Table] = true
	}

	var migrationResults []map[string]migrationSchemas
	for _, m := range sd.migs {
		up := m.Up
		// these would fail later in the test anyway, but easier to be explicit here
		assert.NotEqual(t, up, "", assert.Errorf("mig %d Up can't be empty", m.Version))
		assert.NotEqual(t, m.Down, "", assert.Errorf("mig %d Down can't be empty", m.Version))

		// save before schemas
		schemas := make(map[string]migrationSchemas, len(tables))
		assert.IsNil(t, saveBeforeSchemas(t, tdb, schemas, tables))
		beforeTables, err := migrate.SelectTables(ctx, tdb)
		assert.NoErr(t, err)

		// run the up migration
		_, err = tdb.Exec(up)
		assert.NoErr(t, err, assert.Errorf("error running Up for %d", m.Version))

		// save after schemas
		assert.IsNil(t, saveAfterSchemas(t, tdb, schemas, tables))
		afterUpTables, err := migrate.SelectTables(ctx, tdb)
		assert.NoErr(t, err)

		// ignore changes to the following tables
		tablesToIgnore := []string{m.Table}

		// handle table renames - identify both before & after table name
		renameTableMatches := reRenameTable.FindSubmatch([]byte(up))
		if renameTableMatches != nil {
			// if the RE matches there should be 3 results: the entire expression that matched and the 2
			// parenthesized capture groups (original table name, new table name).
			assert.Equal(t, len(renameTableMatches), 3)
			assert.Equal(t, m.Table, string(renameTableMatches[1]), assert.Errorf("migration `Table` in a rename statement should be original name"))
			tablesToIgnore = append(tablesToIgnore, string(renameTableMatches[2]))
		}

		// check that only the expected tables changed
		for tbl := range tables {
			if strInList(tbl, tablesToIgnore) {
				continue // don't assert that this table's schema *did* change in case of a data migration
			}
			// these schemas should be unaffected
			assert.Equal(t, schemas[tbl].Before, schemas[tbl].After,
				assert.Errorf("unexpected table %s changed on mig %d up", tbl, m.Version))
		}

		if renameTableMatches == nil {
			// check that no new "hidden" tables were created, but ignore the one we operated on in both maps
			// so we don't have to understand create/alter/delete semantics (yet? :) )
			// TODO: be smarter about CREATE/ALTER/DELETE
			delete(beforeTables, m.Table)
			delete(afterUpTables, m.Table)
		} else {
			delete(beforeTables, string(renameTableMatches[1]))
			delete(afterUpTables, string(renameTableMatches[2]))
		}
		assert.Equal(t, afterUpTables, beforeTables, assert.Errorf("migration %d up created or deleted an extra table", m.Version))

		// save these schema results for when we downgrade
		migrationResults = append(migrationResults, schemas)
	}

	// while we're here -- at current max(migrations), ensure our validator works
	// for rootdbs we don't validate columns since it's just the migration table
	if sd.cols != nil {
		sv := migrate.SchemaValidator(migrate.Schema{Columns: sd.cols})
		assert.NoErr(t, sv.Validate(ctx, tdb))
	}

	for i := len(sd.migs) - 1; i >= 0; i-- {
		down := sd.migs[i].Down
		schemas := migrationResults[i]
		// make sure we're in the state we expect
		for tbl := range tables {
			s, err := getSchema(t, tdb, tbl)
			assert.NoErr(t, err)
			assert.Equal(t, s, schemas[tbl].After,
				assert.Errorf("schemas don't match for table %s before down migration %d", tbl, sd.migs[i].Version),
				assert.Must())
		}

		beforeTables, err := migrate.SelectTables(ctx, tdb)
		assert.NoErr(t, err)

		// run the down migration
		_, err = tdb.Exec(down)
		assert.NoErr(t, err, assert.Errorf("down migration %d failed", sd.migs[i].Version))

		// capture the table list again
		afterDownTables, err := migrate.SelectTables(ctx, tdb)
		assert.NoErr(t, err)

		// ignore changes to the following tables
		tablesToIgnore := []string{sd.migs[i].Table}

		// handle table renames - identify both before & after table name
		renameTableMatches := reRenameTable.FindSubmatch([]byte(down))
		if renameTableMatches != nil {
			// if the RE matches there should be 3 results: the entire expression that matched and the 2
			// parenthesized capture groups (original table name, new table name).
			assert.Equal(t, len(renameTableMatches), 3)
			assert.Equal(t, sd.migs[i].Table, string(renameTableMatches[2]), assert.Errorf("migration `Table` in a rename statement should be original name"))
			tablesToIgnore = append(tablesToIgnore, string(renameTableMatches[1]))
		}

		// make sure we're still where we expect
		for tbl := range tables {
			s, err := getSchema(t, tdb, tbl)
			assert.NoErr(t, err)
			assert.Equal(t, s, schemas[tbl].Before,
				assert.Errorf("schemas don't match for table %s after down migration %d", tbl, sd.migs[i].Version),
				assert.Must())

			// if it wasn't the one we messed with, make sure it didn't change in the downgrade either
			if strInList(tbl, tablesToIgnore) {
				continue
			}

			assert.Equal(t, schemas[tbl].Before, schemas[tbl].After,
				assert.Errorf("unexpected table %s changed in mig %d down", tbl, sd.migs[i].Version))
		}

		if renameTableMatches == nil {
			// check that no new "hidden" tables were created, but ignore the one we operated on in both maps
			// so we don't have to understand create/alter/delete semantics (yet? :) )
			// TODO: be smarter about CREATE/ALTER/DELETE
			delete(beforeTables, sd.migs[i].Table)
			delete(afterDownTables, sd.migs[i].Table)
		} else {
			delete(beforeTables, string(renameTableMatches[1]))
			delete(afterDownTables, string(renameTableMatches[2]))
		}
		assert.Equal(t, afterDownTables, beforeTables, assert.Errorf("migration %d down created or deleted extra tables", sd.migs[i].Version))
	}
}

func saveBeforeSchemas(t *testing.T, db *ucdb.DB, schemas map[string]migrationSchemas, tables map[string]any) error {
	return saveSchemas(t, db, schemas, tables, func(sch *migrationSchemas, s string) { sch.Before = s })
}

func saveAfterSchemas(t *testing.T, db *ucdb.DB, schemas map[string]migrationSchemas, tables map[string]any) error {
	return saveSchemas(t, db, schemas, tables, func(sch *migrationSchemas, s string) { sch.After = s })
}

func saveSchemas(t *testing.T, db *ucdb.DB, schemas map[string]migrationSchemas, tables map[string]any, update func(*migrationSchemas, string)) error {
	for tbl := range tables {
		s, err := getSchema(t, db, tbl)
		if err != nil {
			return ucerr.Wrap(err)
		}
		sch, ok := schemas[tbl]
		if !ok {
			sch = migrationSchemas{}
		}
		update(&sch, s)
		schemas[tbl] = sch
	}
	return nil
}

func getSchema(t *testing.T, db *ucdb.DB, tblName string) (string, error) {
	ct := migrate.GetTableSchema(context.Background(), t, db, tblName)
	if ct == "table does not exist" {
		return "table does not exist", nil
	}
	return normalizeSchema(ct)
}

var familyRE = regexp.MustCompile(`([^\)]+) \((.*)\)`)

// because SQL doesn't guarantee ordering on create table syntaxes, we need
// to normalize them ... I don't love messing with SQL like this, but it's only
// for testing so seems ok. If this gets too hairy, we could change our testing
// strategy to eg. make ordering changes less common, it's just slightly less complete
func normalizeSchema(schema string) (string, error) {
	// we rely on the fact that CDB at least returns CREATE statements that look like this:
	//
	// CREATE TABLE foo (
	//   id UUID NOT NULL,
	//   name VARCHAR
	// );
	//
	// which means we can split on newlines rather than trying to parse (tricky when you have eg.
	// multi-column indices that themselves include commas)
	lines := strings.Split(schema, "\n")

	// ignore the first and last lines - CREATE and close paren
	fields := lines[1 : len(lines)-1]

	// trim all the indenting off (since we're going to put them in a single line for
	// easier-to-read diffs), and drop trailing comma so we can add it ourselves in case
	// last line isn't last alphabetically
	for i, f := range fields {
		f = strings.TrimSpace(f)
		f = strings.TrimSuffix(f, ",")

		// of course family can include a list of fields which need to be alphabetized
		if strings.HasPrefix(f, "FAMILY") || strings.HasPrefix(f, "CONSTRAINT") || strings.HasPrefix(f, "UNIQUE") {
			matches := familyRE.FindAllStringSubmatch(f, -1)
			if len(matches) != 1 || len(matches[0]) != 3 {
				return "", ucerr.Errorf("family line didn't match regex: %s", f)
			}
			subs := strings.Split(matches[0][2], ",")
			for i := range subs {
				subs[i] = strings.TrimSpace(subs[i])
			}
			sort.Strings(subs)
			f = fmt.Sprintf("%s (%s)", matches[0][1], strings.Join(subs, ", "))
		}

		fields[i] = f
	}

	// sort them so we have a stable ordering, the whole point of this exercise
	// NB: this might often put things like CONSTRAINT and INDEX in the middle of the columns,
	// which isn't awesome but won't hurt us here.
	sort.Strings(fields)

	// put them back together
	fs := strings.Join(fields, ", ")

	// make them look like SQL again
	return fmt.Sprintf("%s %s)", lines[0], fs), nil
}
