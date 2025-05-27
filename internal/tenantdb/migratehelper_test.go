package tenantdb_test

import (
	"os"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/internal/tenantdb"
)

// TODO: this is replicated everywhere with migratehelper, find a better factoring
func TestMigrations(t *testing.T) {
	assert.Equal(t, tenantdb.GetMigrations(), tenantdb.Migrations[tenantdb.BaselineSchemaVersion+1:])
	assert.NotEqual(t, tenantdb.GetMigrations().GetMaxAvailable(), -1)
}

func TestMain(m *testing.M) {
	// Adjust working dir to match what our services expect.
	os.Chdir("../..")
	os.Exit(m.Run())
}
