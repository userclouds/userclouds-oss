package middleware

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
)

func TestTenantMatchesOrIsNil(t *testing.T) {
	ctx := context.Background()

	// setup (usually from Middleware init)
	consoleTenantID = uuid.Must(uuid.NewV4())

	tenantID := uuid.Must(uuid.NewV4())

	ctx = setTenantID(ctx, tenantID)
	assert.True(t, TenantMatchesOrIsConsole(ctx, tenantID))
	assert.False(t, TenantMatchesOrIsConsole(ctx, uuid.Nil))
	assert.False(t, TenantMatchesOrIsConsole(ctx, uuid.Must(uuid.NewV4())))

	// this allows all requests (need a better solution and name some day)
	ClearTenantID(ctx)
	assert.True(t, TenantMatchesOrIsConsole(ctx, tenantID))
	assert.True(t, TenantMatchesOrIsConsole(ctx, uuid.Nil))
	assert.True(t, TenantMatchesOrIsConsole(ctx, uuid.Must(uuid.NewV4())))
}
