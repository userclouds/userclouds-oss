package logdb

import (
	"testing"

	"userclouds.com/infra/assert"
)

func TestMigrations(t *testing.T) {
	assert.Equal(t, GetMigrations(), Migrations)
	assert.NotEqual(t, GetMigrations().GetMaxAvailable(), -1)
}
