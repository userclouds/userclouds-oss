package serviceconfig_test

import (
	"context"
	"testing"

	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/internal/testconfig"
	config "userclouds.com/plex/serviceconfig"
)

func TestConfig(t *testing.T) {
	ctx := context.Background()
	var cfg config.ServiceConfig
	testconfig.RunConfigTestForService(ctx, t, serviceNamespace.Plex, &cfg)
}
