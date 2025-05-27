package rootdbstatus

import (
	"context"
	"testing"

	"userclouds.com/infra/assert"
)

// TODO: this is replicated everywhere with migratehelper, find a better factoring
func TestMigrations(t *testing.T) {
	assert.Equal(t, GetMigrations(), Migrations)
	assert.NotEqual(t, GetMigrations().GetMaxAvailable(), -1)
}

func TestConfigLoader(t *testing.T) {
	sd, err := GetServiceData(context.Background())
	assert.NoErr(t, err)
	assert.NotNil(t, sd.DBCfg)
}
