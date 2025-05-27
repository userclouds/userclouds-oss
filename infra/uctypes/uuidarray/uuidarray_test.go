package uuidarray_test

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/testdb"
	. "userclouds.com/infra/uctypes/uuidarray"
)

func TestNil(t *testing.T) {
	tdb := testdb.New(t)
	ctx := context.Background()

	_, err := tdb.ExecContext(ctx, "TestNil", `CREATE TABLE test (id UUID, uuids UUID[])`)
	assert.NoErr(t, err)

	_, err = tdb.ExecContext(ctx, "TestNil", `INSERT INTO test (id) VALUES ($1)`, uuid.Nil)
	assert.NoErr(t, err)

	type data struct {
		ID    uuid.UUID `db:"id"`
		UUIDs UUIDArray `db:"uuids"`
	}

	var vals []data
	assert.NoErr(t, tdb.SelectContext(ctx, "TestNil", &vals, `SELECT * FROM test`))

	assert.Equal(t, 1, len(vals))
	assert.Equal(t, vals[0].ID, uuid.Nil)
	assert.Equal(t, len(vals[0].UUIDs), 0)
}
