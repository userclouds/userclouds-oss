package companyconfig_test

import (
	"context"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/internal/companyconfig"
)

func TestMigrations(t *testing.T) {
	assert.Equal(t, companyconfig.GetMigrations(), companyconfig.Migrations[companyconfig.BaselineSchemaVersion+1:])
	assert.NotEqual(t, companyconfig.GetMigrations().GetMaxAvailable(), -1)
}

func TestConfigLoader(t *testing.T) {
	sd, err := companyconfig.GetServiceData(context.Background())
	assert.NoErr(t, err)
	assert.NotNil(t, sd.DBCfg)
}
