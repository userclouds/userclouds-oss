package internal_test

import (
	"context"
	"testing"

	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/internal/testconfig"
	"userclouds.com/logserver/config"
)

func TestConfig(t *testing.T) {
	ctx := context.Background()
	var cfg config.Config
	testconfig.RunConfigTestForService(ctx, t, serviceNamespace.LogServer, &cfg)
}
