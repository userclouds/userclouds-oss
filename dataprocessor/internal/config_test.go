package internal_test

import (
	"context"
	"testing"

	"userclouds.com/dataprocessor/internal"
	"userclouds.com/internal/testconfig"
)

func TestConfig(t *testing.T) {
	ctx := context.Background()
	var cfg internal.Config
	testconfig.RunConfigTestTool(ctx, t, "dataprocessor", &cfg, false)
}
