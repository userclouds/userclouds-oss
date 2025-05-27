package timestamparray

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/uctypes/timestamp"
)

func TestTimestampArray(t *testing.T) {
	tdb := testdb.New(t)
	ctx := context.Background()

	_, err := tdb.ExecContext(ctx, "TestTimestamps", `CREATE TABLE test (id UUID, one TIMESTAMP, many VARCHAR[])`)
	assert.NoErr(t, err)

	one := timestamp.Normalize(time.Now().UTC())
	two := time.Time{}
	many := TimestampArray{one, two}
	_, err = tdb.ExecContext(ctx, "TestTimestamps", `INSERT INTO test (id, one, many) VALUES ($1, $2, $3)`, uuid.Nil, &one, &many)
	assert.NoErr(t, err)

	type val struct {
		ID   uuid.UUID      `db:"id"`
		One  time.Time      `db:"one"`
		Many TimestampArray `db:"many"`
	}

	var vals []val
	assert.NoErr(t, tdb.SelectContext(ctx, "TestTimestamps", &vals, `SELECT * FROM test`))

	assert.Equal(t, 1, len(vals))
	assert.Equal(t, vals[0].ID, uuid.Nil)
	assert.Equal(t, vals[0].One, one)
	assert.Equal(t, 2, len(vals[0].Many))
	assert.Equal(t, vals[0].Many[0], one)
	assert.Equal(t, vals[0].Many[1], two)
}
