package ucdb_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lib/pq"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucdb"
)

type noopValidator struct{}

func (noopValidator) Validate(ctx context.Context, db *ucdb.DB) error {
	return nil
}

// we originally implemented session vars for cockroach to handle eg. sql_safe_updates
// but postgres doesn't support that. Rather than ripping out the code if we care in the future,
// this test just ensures that we can set session vars and read them back
func TestSessionVars(t *testing.T) {
	ctx := context.Background()

	tdb := testdb.New(t)

	// Create a table to test against
	_, err := tdb.ExecContext(ctx, "TestSessionVars", "CREATE TABLE tmp (id UUID PRIMARY KEY);")
	assert.NoErr(t, err)

	// try unsafe (unscoped) UPDATE
	cfg := testdb.TestConfig(t, tdb)
	sv := "SET my.var=1;"
	cfg.SessionVars = &sv
	unsafeDB, err := ucdb.New(ctx, &cfg, noopValidator{})
	assert.NoErr(t, err)
	var res string
	assert.NoErr(t, unsafeDB.GetContext(ctx, "TestSessionVars", &res, "SHOW my.var;"))
	assert.Equal(t, res, "1")
}

func TestTimeout(t *testing.T) {
	ctx := context.Background()

	tdb := testdb.New(t)

	var unused bool

	tdb.SetTimeout(time.Second)
	err := tdb.GetContext(ctx, "testsleep2", &unused, "SELECT PG_SLEEP(2);")
	assert.NotNil(t, err)
	var pe *pq.Error
	assert.True(t, errors.As(err, &pe))
	assert.Equal(t, pe.Code, pq.ErrorCode("57014"))

	tdb.SetTimeout(2 * time.Second)
	_, err = tdb.ExecContext(ctx, "testsleep", "SELECT PG_SLEEP(1);")
	assert.NoErr(t, err)
}

func TestValidationError(t *testing.T) {
	e := context.Canceled
	err := ucdb.TESTONLYNewValidationError(e)
	assert.ErrorIs(t, err, context.Canceled)
}
