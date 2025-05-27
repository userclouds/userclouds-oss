package uctrace_test

import (
	"context"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/uctrace"
)

func TestTracingInit(t *testing.T) {
	// Tests that the init process works even with a non-existent collector
	// We no longer expect an error for transient failures as they are logged as warnings
	ctx := context.Background()
	cfg := uctrace.Config{CollectorHost: "localhost:6666"}
	shutdown, err := uctrace.Init(ctx, "authz", "festivus", &cfg)
	assert.IsNil(t, err)       // No error expected as transient failures are now logged as warnings
	assert.NotNil(t, shutdown) // Shutdown function should be returned

	// Clean up
	shutdown(ctx)
}
