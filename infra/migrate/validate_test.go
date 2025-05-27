package migrate_test

import (
	"context"
	"maps"
	"os"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/testdb"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/tools/generate"
	"userclouds.com/tools/generate/genorm"
)

func TestSchemaValidator(t *testing.T) {
	ctx := context.Background()
	tdb := testdb.New(t, migrate.NewTestSchema(companyconfig.Schema))

	// normal case
	sv := migrate.SchemaValidator(companyconfig.Schema)
	assert.NoErr(t, sv.Validate(ctx, tdb))

	// and we should be able to codegen queries against this safely
	// NB: this part of the test case is pretty fragile to the way we generate queries,
	// but it also serves a purpose to ensure we generate those queries mostly correctly
	// (eg. avoid using SELECT *, which will break on unexpected fields)
	tmpDir, err := os.MkdirTemp("", "validate_test")
	assert.NoErr(t, err)
	p := generate.GetPackageForPath("../../internal/companyconfig", true)
	genorm.Run(ctx, p, tmpDir, "genorm" /* ignored */, "Company", "companies", "companyconfig")
	gbs, err := os.ReadFile(tmpDir + "/company_orm_generated.go")
	assert.NoErr(t, err)
	var q string
	for line := range strings.SplitSeq(string(gbs), "\n") {
		if strings.Contains(line, "SELECT") && strings.Contains(line, "FROM companies WHERE id=$1 AND deleted=") {
			q = strings.Split(line, `"`)[1]
			break
		}
	}
	oid := uuid.Must(uuid.NewV4())
	const saveQ = `INSERT INTO companies (id, name) VALUES ($1, 'foo'); /* lint-deleted-ok */`
	_, err = tdb.ExecContext(ctx, "TestSchemaValidator", saveQ, oid)
	assert.NoErr(t, err)
	var company companyconfig.Company
	assert.NoErr(t, tdb.GetContext(ctx, "TestSchemaValidator", &company, q, oid))
	// Note: if anything above fails, it's probably because our genorm "interface" changed slightly

	// if we add a column to the database, we should still validate
	_, err = tdb.ExecContext(ctx, "TestSchemaValidator", "ALTER TABLE companies ADD COLUMN validation_testing VARCHAR;")
	assert.NoErr(t, err)
	assert.NoErr(t, sv.Validate(ctx, tdb))

	// and our codegen'd query should still work
	assert.NoErr(t, tdb.GetContext(ctx, "TestSchemaValidator", &company, q, oid))

	// and if we add a whole table, that's ok too
	_, err = tdb.ExecContext(ctx, "TestSchemaValidator", "CREATE TABLE extraneous (id UUID);")
	assert.NoErr(t, err)
	assert.NoErr(t, sv.Validate(ctx, tdb))

	// if we delete a required column from the database, we should fail
	_, err = tdb.ExecContext(ctx, "TestSchemaValidator", "ALTER TABLE companies DROP COLUMN updated;")
	assert.NoErr(t, err)
	assert.NotNil(t, sv.Validate(ctx, tdb))
	_, err = tdb.ExecContext(ctx, "TestSchemaValidator", "ALTER TABLE companies ADD COLUMN updated TIMESTAMP;")
	assert.NoErr(t, err)
	assert.NoErr(t, sv.Validate(ctx, tdb)) // make sure we're back in a good state

	// likewise if we delete a whole required table, we should fail

	// save the create table statement for later
	ct := migrate.GetTableSchema(ctx, t, tdb, "companies")

	_, err = tdb.ExecContext(ctx, "TestSchemaValidator", "DROP TABLE companies")
	assert.NoErr(t, err)
	assert.NotNil(t, sv.Validate(ctx, tdb))
	_, err = tdb.ExecContext(ctx, "TestSchemaValidator", ct)
	assert.NoErr(t, err)
	assert.NoErr(t, sv.Validate(ctx, tdb)) // make sure we're back in a good state

	// but if we add a new column in the code that doesn't exist in the DB yet, fail
	newUC := make(map[string][]string)
	newS := migrate.Schema{Columns: newUC}
	maps.Copy(newUC, companyconfig.UsedColumns)
	newUC["companies"] = append(newUC["companies"], "new_feature_column")
	sv = migrate.SchemaValidator(newS)
	assert.NotNil(t, sv.Validate(ctx, tdb))
	newUC["companies"] = companyconfig.UsedColumns["companies"] // reset to normal
	sv = migrate.SchemaValidator(newS)
	assert.NoErr(t, sv.Validate(ctx, tdb))

	// adding a new table in code (but not DB) should also fail
	newUC["new_feature_table"] = []string{"id"}
	sv = migrate.SchemaValidator(newS)
	assert.NotNil(t, sv.Validate(ctx, tdb))
	delete(newUC, "new_feature_table") // reset to normal
	sv = migrate.SchemaValidator(newS)
	assert.NoErr(t, sv.Validate(ctx, tdb))

	// removing a column from the code but not the DB should be ok
	cCompanies := companyconfig.UsedColumns["companies"]
	newUC["companies"] = cCompanies[:len(cCompanies)-1]
	sv = migrate.SchemaValidator(newS)
	assert.NoErr(t, sv.Validate(ctx, tdb))
	newUC["companies"] = companyconfig.UsedColumns["companies"] // reset to normal
	sv = migrate.SchemaValidator(newS)
	assert.NoErr(t, sv.Validate(ctx, tdb))

	// and same for removing a table from code but not DB
	delete(newUC, "companies")
	sv = migrate.SchemaValidator(newS)
	assert.NoErr(t, sv.Validate(ctx, tdb))
	newUC["companies"] = companyconfig.UsedColumns["companies"] // reset to normal (in case future tests added below)
	sv = migrate.SchemaValidator(newS)
	assert.NoErr(t, sv.Validate(ctx, tdb))
}
