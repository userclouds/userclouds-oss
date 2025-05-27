package internal_test

import (
	"context"
	"os"
	"testing"

	"userclouds.com/cmd/devlb/internal"
	"userclouds.com/internal/testconfig"
)

func TestConfig(t *testing.T) {
	ctx := context.Background()
	var cfg internal.Config
	testconfig.RunConfigTestTool(ctx, t, "devlb", &cfg, true)
}

func TestMain(m *testing.M) {
	// Adjust working dir to match what our services expect.
	os.Chdir("../../..")
	os.Exit(m.Run())
}
