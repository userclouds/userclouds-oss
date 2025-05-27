package rootdb

import (
	"context"
	"testing"

	"userclouds.com/infra/assert"
)

// TODO: this is replicated everywhere with migratehelper, find a better factoring
func TestMigrations(t *testing.T) {
	t.Run(("Test migrations for db"), func(t *testing.T) {
		returnedMigrations := GetMigrations()
		// Since postgres doesn't support CREATELOGIN, we need to remove it from the SQL command when running on postgres.
		// This test ensures that the migration is modified correctly.
		assert.Equal(t, len(returnedMigrations), len(migrations))
		assert.NotEqual(t, GetMigrations().GetMaxAvailable(), -1)
	})
}

func TestConfigLoader(t *testing.T) {
	sd, err := GetServiceData(context.Background())
	assert.NoErr(t, err)
	assert.NotNil(t, sd.DBCfg)
}
