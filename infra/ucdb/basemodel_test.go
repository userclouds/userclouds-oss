package ucdb_test

import (
	"testing"
	"time"

	"userclouds.com/infra/assert"
	. "userclouds.com/infra/ucdb"
)

func TestDeleted(t *testing.T) {
	type s struct {
		BaseModel
	}

	obj := s{
		BaseModel: NewBase(),
	}

	assert.True(t, obj.Alive())

	obj.Deleted = time.Now().UTC()
	assert.False(t, obj.Alive())
}
