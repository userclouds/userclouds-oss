package routinghelper

import (
	"context"
	"slices"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
)

var (
	universesUsingPorts = []universe.Universe{
		universe.Container,
		universe.Dev,
		universe.CI,
		universe.Test,
	}
)

func TestIngressParsing(t *testing.T) {
	ctx := context.Background()
	for _, uv := range universesUsingPorts {
		routeCfg, err := ParseIngressConfig(ctx, uv, "test.userclouds.tools", "mars", service.AllWebServices)
		assert.NoErr(t, err)
		assert.Equal(t, len(routeCfg.Rules), 7)
		ports := routeCfg.GetPorts()
		slices.Sort(ports)
		assert.Equal(t, ports, []int{5000, 5100, 5200, 5300, 5500})
	}
}

func TestIngressParsingHeadlessContainer(t *testing.T) {
	ctx := context.Background()
	routeCfg, err := ParseIngressConfig(ctx, universe.Container, "test.userclouds.tools", "mars", []service.Service{service.IDP, service.Plex, service.AuthZ})
	assert.NoErr(t, err)
	assert.Equal(t, len(routeCfg.Rules), 4)
	ports := routeCfg.GetPorts()
	slices.Sort(ports)
	assert.Equal(t, ports, []int{5000, 5100, 5200})
}

func TestGetPorts(t *testing.T) {
	ctx := context.Background()
	for _, uv := range universesUsingPorts {
		ports, err := getPortsForServices(ctx, uv, service.AllWebServices)
		assert.NoErr(t, err)
		assert.Equal(t, ports, []int{5000, 5100, 5200, 5300, 5500})
	}
}
