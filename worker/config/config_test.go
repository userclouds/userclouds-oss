package config_test

import (
	"context"
	"os"
	"testing"

	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/internal/testconfig"
	"userclouds.com/worker/config"
)

func TestConfig(t *testing.T) {
	ctx := context.Background()
	var cfg config.Config
	testconfig.RunConfigTestForService(ctx, t, serviceNamespace.Worker, &cfg)

}

func TestMain(m *testing.M) {
	// Adjust working dir to match what our services expect.
	os.Chdir("../..")
	os.Exit(m.Run())
}
