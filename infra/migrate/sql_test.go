package migrate_test

import (
	"context"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/testdb"
	"userclouds.com/internal/companyconfig"
)

func TestGetMaxVersion(t *testing.T) {
	ctx := context.Background()

	tdb := testdb.New(t)
	assert.NoErr(t, migrate.CreateMigrationsTable(ctx, tdb))

	mv, err := migrate.GetMaxVersion(ctx, tdb)
	assert.NoErr(t, err)
	assert.Equal(t, mv, -1)

	m := companyconfig.Migrations[0]
	assert.NoErr(t, migrate.SaveMigration(ctx, tdb, &m))
	mv, err = migrate.GetMaxVersion(ctx, tdb)
	assert.NoErr(t, err)
	assert.Equal(t, mv, m.Version)
}
