package routinghelper

import (
	"context"
	"slices"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
)

func TestParseIngressConfig(t *testing.T) {
	ctx := context.Background()
	baseURL := "example.com"
	region := "us-west-2"

	t.Run("AllServicesIncluded", func(t *testing.T) {
		routeCfg, err := ParseIngressConfig(ctx, universe.Container, baseURL, region, service.AllWebServices)
		assert.NoErr(t, err)

		assert.Equal(t, len(routeCfg.Rules), 7)

		assert.NotNil(t, routeCfg.findRuleService(service.IDP), assert.Must())
		assert.NotNil(t, routeCfg.findRuleService(service.AuthZ), assert.Must())
		assert.NotNil(t, routeCfg.findRuleService(service.Plex), assert.Must())
		assert.NotNil(t, routeCfg.findRuleService(service.Console), assert.Must())
		assert.NotNil(t, routeCfg.findRuleService(service.LogServer), assert.Must())

		authzRule := routeCfg.findRuleService(service.AuthZ)
		assert.NotNil(t, authzRule, assert.Must())
		assert.True(t, slices.Contains(authzRule.PathPrefixes, "/authz/"))
		assert.True(t, slices.Contains(authzRule.PathPrefixes, "/auditlog/"))

		idpRule := routeCfg.findRuleService(service.IDP)
		assert.NotNil(t, idpRule, assert.Must())
		assert.True(t, slices.Contains(idpRule.PathPrefixes, "/authn/"))
		assert.True(t, slices.Contains(idpRule.PathPrefixes, "/userevent/"))
		assert.True(t, slices.Contains(idpRule.PathPrefixes, "/userstore/"))
		assert.True(t, slices.Contains(idpRule.PathPrefixes, "/tokenizer/"))
		assert.True(t, slices.Contains(idpRule.PathPrefixes, "/s3shim/"))

		consoleRule := routeCfg.findRuleService(service.Console)
		assert.NotNil(t, consoleRule, assert.Must())
		assert.True(t, slices.Contains(consoleRule.HostHeaders, "console.example.com"))
		assert.True(t, slices.Contains(consoleRule.HostHeaders, "console.aws-us-west-2.example.com"))
	})

	t.Run("LimitedServicesIncluded", func(t *testing.T) {
		includeServices := []service.Service{service.IDP, service.Plex}
		routeCfg, err := ParseIngressConfig(ctx, universe.Container, baseURL, region, includeServices)
		assert.NoErr(t, err)

		assert.NotNil(t, routeCfg.findRuleService(service.IDP), assert.Must())
		assert.NotNil(t, routeCfg.findRuleService(service.Plex), assert.Must())

		assert.IsNil(t, routeCfg.findRuleService(service.AuthZ))
		assert.IsNil(t, routeCfg.findRuleService(service.Console))
	})

	t.Run("SingleServiceIncluded", func(t *testing.T) {
		includeServices := []service.Service{service.AuthZ}
		routeCfg, err := ParseIngressConfig(ctx, universe.Container, baseURL, region, includeServices)
		assert.NoErr(t, err)

		assert.NotNil(t, routeCfg.findRuleService(service.AuthZ), assert.Must())

		assert.IsNil(t, routeCfg.findRuleService(service.IDP))
		assert.IsNil(t, routeCfg.findRuleService(service.Console))
		assert.IsNil(t, routeCfg.findRuleService(service.LogServer))
	})

	t.Run("EmptyServicesIncluded", func(t *testing.T) {
		includeServices := []service.Service{}
		routeCfg, err := ParseIngressConfig(ctx, universe.Container, baseURL, region, includeServices)
		assert.NoErr(t, err)

		assert.IsNil(t, routeCfg.findRuleService(service.IDP))
		assert.IsNil(t, routeCfg.findRuleService(service.AuthZ))
		assert.IsNil(t, routeCfg.findRuleService(service.Console))
		assert.IsNil(t, routeCfg.findRuleService(service.LogServer))
	})

	t.Run("HostHeaderProcessing", func(t *testing.T) {
		routeCfg, err := ParseIngressConfig(ctx, universe.Container, baseURL, region, []service.Service{service.IDP})
		assert.NoErr(t, err)

		var idpRuleWithHost *Rule
		for _, rule := range routeCfg.Rules {
			if rule.Service == service.IDP && len(rule.HostHeaders) > 0 {
				idpRuleWithHost = &rule
				break
			}
		}
		assert.NotNil(t, idpRuleWithHost, assert.Must())

		assert.True(t, slices.Contains(idpRuleWithHost.HostHeaders, "idp.example.com"))
	})

	t.Run("DifferentRegion", func(t *testing.T) {
		routeCfg, err := ParseIngressConfig(ctx, universe.Container, baseURL, "eu-west-1", []service.Service{service.IDP})
		assert.NoErr(t, err)

		var idpRuleWithHost *Rule
		for _, rule := range routeCfg.Rules {
			if rule.Service == service.IDP && len(rule.HostHeaders) > 0 {
				idpRuleWithHost = &rule
				break
			}
		}
		assert.NotNil(t, idpRuleWithHost, assert.Must())

		assert.True(t, slices.Contains(idpRuleWithHost.HostHeaders, "idp.example.com"))
	})
}
