package normalizedstring_test

import (
	"context"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/uctypes/normalizedstring"
)

func TestNormalizedString(t *testing.T) {
	tdb := testdb.New(t)
	ctx := context.Background()

	str := "Foo"
	normalizedStr := normalizedstring.String(str)

	type data struct {
		ID            int                      `db:"id"`
		Str           *string                  `db:"str"`
		NormalizedStr *normalizedstring.String `db:"normalized_str"`
	}

	_, err := tdb.ExecContext(ctx, "TestNormalizedString", `CREATE TABLE test (id INT, str VARCHAR, normalized_str VARCHAR)`)
	assert.NoErr(t, err)

	d := data{ID: 1}
	_, err = tdb.ExecContext(ctx, "TestNormalizedString", `INSERT INTO test (id, str, normalized_str) VALUES ($1, $2, $3)`, d.ID, d.Str, d.NormalizedStr)
	assert.NoErr(t, err)

	d = data{ID: 2, Str: &str, NormalizedStr: &normalizedStr}
	_, err = tdb.ExecContext(ctx, "TestNormalizedString", `INSERT INTO test (id, str, normalized_str) VALUES ($1, $2, $3)`, d.ID, d.Str, d.NormalizedStr)
	assert.NoErr(t, err)

	var vals []data
	assert.NoErr(t, tdb.SelectContext(ctx, "TestNormalizedString", &vals, `SELECT * FROM test`))

	assert.Equal(t, 2, len(vals))
	assert.Equal(t, vals[0].ID, 1)
	assert.IsNil(t, vals[0].Str)
	assert.IsNil(t, vals[0].NormalizedStr)
	assert.Equal(t, vals[1].ID, 2)
	assert.Equal(t, *vals[1].Str, "Foo")
	assert.Equal(t, *vals[1].NormalizedStr, normalizedstring.String("foo"))

	var strExact string = "foo"
	assert.NoErr(t, tdb.SelectContext(ctx, "TestNormalizedString", &vals, `SELECT * FROM test WHERE str = $1`, strExact))
	assert.Equal(t, 0, len(vals))

	var normalizedExact normalizedstring.String = "Foo"
	assert.NoErr(t, tdb.SelectContext(ctx, "TestNormalizedString", &vals, `SELECT * FROM test WHERE normalized_str = $1`, normalizedExact))
	assert.Equal(t, 1, len(vals))

	var strPattern string = "fo%"
	assert.NoErr(t, tdb.SelectContext(ctx, "TestNormalizedString", &vals, `SELECT * FROM test WHERE str LIKE $1`, strPattern))
	assert.Equal(t, 0, len(vals))

	var normalizedPattern normalizedstring.String = "Fo%"
	assert.NoErr(t, tdb.SelectContext(ctx, "TestNormalizedString", &vals, `SELECT * FROM test WHERE normalized_str LIKE $1`, normalizedPattern))
	assert.Equal(t, 1, len(vals))
}
