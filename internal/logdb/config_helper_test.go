package logdb

import (
	"context"
	"testing"

	"userclouds.com/infra/assert"
)

func TestConfigLoader(t *testing.T) {
	sd, err := GetServiceData(context.Background())
	assert.NoErr(t, err)
	assert.NotNil(t, sd.DBCfg)
}
