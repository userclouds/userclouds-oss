package config_test

import (
	"context"
	"os"
	"testing"

	"userclouds.com/idp/config"
	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/internal/testconfig"
)

func TestConfig(t *testing.T) {
	ctx := context.Background()
	var cfg config.Config
	testconfig.RunConfigTestForService(ctx, t, serviceNamespace.IDP, &cfg)

}

func TestDBProxyConfig(t *testing.T) {
	ctx := context.Background()
	var cfg config.Config
	excludedUniverses := make([]universe.Universe, 0, len(universe.AllUniverses()))
	for _, u := range universe.AllUniverses() {
		if !u.IsCloud() {
			excludedUniverses = append(excludedUniverses, u)
		}
	}
	// We use this only in k8s envs for debug,staging and prod. all other envs (and clouds envs not in k8s) use the regular IDP/Userstore config
	testconfig.RunConfigTestForAll(ctx, t, "dbproxy", &cfg, true, false, true, excludedUniverses...)
}

func TestMain(m *testing.M) {
	// Adjust working dir to match what our services expect.
	os.Chdir("../..")
	os.Exit(m.Run())
}
