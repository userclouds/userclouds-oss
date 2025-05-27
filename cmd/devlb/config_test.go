package main

import (
	"context"
	"os"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/internal/routinghelper"
)

// This is a pretty trivial test, but it would have could an EB parsing
// bug I introduced a few weeks ago so we'll start here
// I don't want to check that X maps to Y since it could/should change,
// but this function should never panic if things are ok, and devlb
// isn't exercised in the commit / test path right now
func TestConfigParsing(t *testing.T) {
	os.Chdir("../..") // parseEBConfig expects to be called from the repo root

	ctx := context.Background()
	routeCfg, err := routinghelper.ParseIngressConfig(ctx, universe.Dev, "dev.userclouds.tools", "dev-region", service.AllWebServices)
	assert.NoErr(t, err)
	assert.NotEqual(t, len(routeCfg.Rules), 0)
	assert.NotEqual(t, len(routeCfg.Ports), 0)
}
