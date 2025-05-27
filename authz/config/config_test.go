package config_test

import (
	"context"
	"os"
	"testing"

	"userclouds.com/authz/config"
	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/internal/testconfig"
)

func TestAuthZConfig(t *testing.T) {
	ctx := context.Background()
	var cfg config.Config
	testconfig.RunConfigTestForService(ctx, t, serviceNamespace.AuthZ, &cfg)
}

func TestCheckAttributeConfig(t *testing.T) {
	ctx := context.Background()
	var cfg config.Config
	testconfig.RunConfigTestForService(ctx, t, serviceNamespace.CheckAttribute, &cfg)
}

func TestMain(m *testing.M) {
	// Adjust working dir to match what our services expect.
	os.Chdir("../..")
	os.Exit(m.Run())
}
