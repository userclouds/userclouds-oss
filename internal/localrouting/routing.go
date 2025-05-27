package localrouting

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/go-http-utils/headers"

	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/routinghelper"
)

func stripPortFromHost(host string) string {
	parts := strings.SplitN(host, ":", 2)
	return parts[0]
}

// ProxiesMap is a map of process names to their proxy handler method
type ProxiesMap map[service.Service]*httputil.ReverseProxy

func addRoutes(ctx context.Context, routerName string, mux *http.ServeMux, proxies *ProxiesMap, routes []string, svc service.Service) error {
	proxy := proxies.mustGetProxy(ctx, svc)
	for _, route := range routes {
		uclog.Debugf(ctx, "%s routing %s -> %v", routerName, route, svc)
		mux.Handle(route, proxy)
	}
	return nil
}

func (proxies *ProxiesMap) mustGetProxy(ctx context.Context, service service.Service) *httputil.ReverseProxy {
	proxy, ok := (*proxies)[service]
	if !ok {
		uclog.Fatalf(ctx, "couldn't find proxy for service %s. proxies: %+v", service, *proxies)
	}
	return proxy
}

// NewProxiesMap creates a new ProxiesMap and adds the services/processed to it.
func NewProxiesMap(ctx context.Context, routeCfg routinghelper.RouteConfig, services []service.Service) (*ProxiesMap, error) {
	proxies := &ProxiesMap{}
	for _, svc := range services {
		if svc == service.Worker {
			continue
		}
		redirectPort, err := routeCfg.GetPortForService(svc)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		rewrite := func(r *httputil.ProxyRequest) {
			r.Out.URL.Scheme = "http"
			r.Out.URL.Host = fmt.Sprintf("%s:%v", stripPortFromHost(r.In.URL.Host), redirectPort)
			r.Out.Header[headers.XForwardedFor] = r.In.Header[headers.XForwardedFor]
			r.SetXForwarded()
		}
		(*proxies)[svc] = &httputil.ReverseProxy{Rewrite: rewrite}
	}
	return proxies, nil
}

// AddRulesToServer adds the rules from the given eb config to the given http mux
func (proxies *ProxiesMap) AddRulesToServer(ctx context.Context, routerName string, rules []routinghelper.Rule, mux *http.ServeMux) error {
	for _, rule := range rules {
		if len(rule.HostHeaders) > 0 {
			var routes []string
			for _, hostname := range rule.HostHeaders {
				for _, pathPrefix := range rule.PathPrefixes {
					routes = append(routes, hostname+pathPrefix)
				}
			}
			if err := addRoutes(ctx, routerName, mux, proxies, routes, rule.Service); err != nil {
				return ucerr.Wrap(err)
			}
		} else {
			if err := addRoutes(ctx, routerName, mux, proxies, rule.PathPrefixes, rule.Service); err != nil {
				return ucerr.Wrap(err)
			}
		}
	}
	return nil
}
