package routinghelper

import (
	"slices"
	"strings"

	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucerr"
)

// Rule stores info for an ALB rule parsed from EB config
type Rule struct {
	PathPrefixes    []string `yaml:"path_prefixes" json:"path_prefixes"`
	HostHeaders     []string `yaml:"host_headers,omitempty" json:"host_headers,omitempty"`
	service.Service `yaml:"service" json:"service"`
}

// RouteConfig stores routing rules for services and service ports parsed from our config files svc_listener objects
type RouteConfig struct {
	Rules []Rule                  `yaml:"rules" json:"rules"`
	Ports map[service.Service]int `yaml:"process_ports" json:"process_ports"`
}

// GetPorts returns the ports used by services
func (cfg *RouteConfig) GetPorts() []int {
	ports := make([]int, 0, len(cfg.Ports))
	for _, port := range cfg.Ports {
		ports = append(ports, port)
	}
	return ports
}

func newRouteConfig(portsMap map[service.Service]int) RouteConfig {
	return RouteConfig{
		Ports: portsMap,
	}
}

// GetNonHostRules returns the rules that don't have host headers
func (cfg *RouteConfig) GetNonHostRules() []Rule {
	nonHostRules := make([]Rule, 0, len(cfg.Rules))
	for _, rule := range cfg.Rules {
		if len(rule.HostHeaders) == 0 {
			nonHostRules = append(nonHostRules, rule)
		}
	}
	return nonHostRules
}

// GetPortForService returns the port for a given service
func (cfg *RouteConfig) GetPortForService(svc service.Service) (int, error) {
	port, ok := cfg.Ports[svc]
	if !ok {
		return -1, ucerr.Errorf("couldn't find port for service %s", svc)
	}
	return port, nil
}

func (cfg *RouteConfig) addRule(service service.Service, pathPrefixes []string, hostHeaders ...string) {
	cfg.Rules = append(cfg.Rules, Rule{
		PathPrefixes: pathPrefixes,
		HostHeaders:  hostHeaders,
		Service:      service,
	})
}

func (cfg *RouteConfig) findRuleService(svc service.Service) *Rule {
	for i := range cfg.Rules {
		if cfg.Rules[i].Service == svc {
			return &cfg.Rules[i] // Return a pointer to the struct in the slice
		}
	}
	return nil
}

func (cfg *RouteConfig) findByHost(host string) *Rule {
	for i := range cfg.Rules {
		if cfg.Rules[i].isHostMatch(host) {
			return &cfg.Rules[i]
		}
	}
	return nil
}

func (rule *Rule) isHostMatch(host string) bool {
	if rule.isTenantHost() {
		return isTenantHost(host)
	}
	return strings.Split(rule.HostHeaders[0], ".")[0] == strings.Split(host, ".")[0]
}

func (rule *Rule) updateRule(host, path string) {
	if !slices.Contains(rule.PathPrefixes, path) {
		rule.PathPrefixes = append(rule.PathPrefixes, path)
	}
	if !slices.Contains(rule.HostHeaders, host) {
		rule.HostHeaders = append(rule.HostHeaders, host)
	}
}

func (rule *Rule) isTenantHost() bool {
	return isTenantHost(rule.HostHeaders[0])
}

func isTenantHost(host string) bool {
	return strings.HasPrefix(host, "*.")
}
