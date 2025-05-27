package internal_test

import (
	"context"
	"testing"

	"userclouds.com/console/internal"
	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/internal/testconfig"
)

func TestConfig(t *testing.T) {
	ctx := context.Background()
	var cfg internal.Config
	testconfig.RunConfigTestForService(ctx, t, serviceNamespace.Console, &cfg)
}
