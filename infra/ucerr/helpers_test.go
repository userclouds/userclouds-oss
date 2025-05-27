package ucerr_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lib/pq"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucerr"
)

func TestIsContextCanceledError(t *testing.T) {
	ctx := context.Background()
	tdb := testdb.New(t)
	assert.False(t, ucerr.IsContextCanceledError(nil))
	assert.False(t, ucerr.IsContextCanceledError(ucerr.New("No Soup for you")))
	tdb.SetTimeout(time.Millisecond * 50)
	_, err := tdb.ExecContext(ctx, "TestIsContextCanceledError", "SELECT PG_SLEEP(1);")
	assert.NotNil(t, err)
	var pgErr *pq.Error
	assert.True(t, errors.As(err, &pgErr), assert.Errorf("Expected a *pq.Error, got %T: %v", err, err))
	assert.True(t, ucerr.IsContextCanceledError(err))
	assert.True(t, ucerr.IsContextCanceledError(context.Canceled))
}
