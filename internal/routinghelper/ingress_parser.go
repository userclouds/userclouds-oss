package routinghelper

import (
	"context"
	"fmt"
	"slices"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"

	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/uchelm"
)

// ParseIngressConfig parses the ingress configuration from a Helm chart Ingress object and returns a RouteConfig.
func ParseIngressConfig(ctx context.Context, uv universe.Universe, baseURL string, region string, includeServices []service.Service) (RouteConfig, error) {
	pm, err := getPortsForServicesMap(ctx, uv, includeServices)
	if err != nil {
		return RouteConfig{}, ucerr.Wrap(err)
	}
	cfg := newRouteConfig(pm)
	// We must render the ingress for the prod setup (the ingress is rendered the same for all cloud envs) and not for the current universe which cloud be a non-cloud universe.
	// We only need the routing rules from the ingress. and that part is the same for all cloud universes.
	releaseManifest, err := uchelm.RenderUCChartForUniverse(universe.Prod)
	if err != nil {
		return cfg, ucerr.Wrap(err)
	}
	var ingress networkingv1.Ingress
	if err := uchelm.GetFirstManifest(releaseManifest, "Ingress", &ingress); err != nil {
		return cfg, ucerr.Wrap(err)
	}
	if err := addRulesFromIngress(ingress, &cfg, includeServices); err != nil {
		return cfg, ucerr.Wrap(err)
	}

	for i := range cfg.Rules {
		hostHeaders := make([]string, 0, 2)
		if !cfg.Rules[i].isTenantHost() {
			part := strings.Split(cfg.Rules[i].HostHeaders[0], ".")[0]
			hostHeaders = []string{
				fmt.Sprintf("%s.%s", part, baseURL),
			}
			// Add regional host names, only if there is more than one host name defined
			// the additional host names defined in the ingress are the regional ones.
			if len(cfg.Rules[i].HostHeaders) > 1 {
				regionalHost := fmt.Sprintf("%s.aws-%s.%s", part, region, baseURL)
				hostHeaders = append(hostHeaders, regionalHost)
			}
		}
		cfg.Rules[i].HostHeaders = hostHeaders
	}
	defaultSvc := ingress.Spec.DefaultBackend.Service.Name
	cfg.addRule(service.Service(defaultSvc), []string{"/"})
	return cfg, nil
}

func addRulesFromIngress(ingress networkingv1.Ingress, cfg *RouteConfig, includeServices []service.Service) error {
	for _, rule := range ingress.Spec.Rules {
		for _, path := range rule.HTTP.Paths {
			if path.PathType == nil {
				return ucerr.Errorf("path type is nil for '%s'", path.Path)
			}
			var svc service.Service
			if path.Backend.Service.Name == "userstore" {
				svc = service.IDP
			} else {
				svc = service.Service(path.Backend.Service.Name)
			}
			if !slices.Contains(includeServices, svc) {
				continue
			}
			if *path.PathType != networkingv1.PathTypePrefix {
				continue
			}
			if path.Path == "" {
				continue
			}
			if path.Path[0] != '/' {
				path.Path = "/" + path.Path
			}
			if existingRule := cfg.findRuleService(svc); existingRule != nil {
				if existingRule.isHostMatch(rule.Host) {
					// Add the path to the existing rule's PathPrefixes
					existingRule.updateRule(rule.Host, path.Path)
					continue
				} else if existingRule := cfg.findByHost(rule.Host); existingRule != nil {
					existingRule.updateRule(rule.Host, path.Path)
					continue
				}
			}
			cfg.addRule(svc, []string{path.Path}, rule.Host)
		}
	}
	return nil
}
