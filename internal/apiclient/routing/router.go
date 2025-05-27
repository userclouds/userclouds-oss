package routing

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/kubernetes"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/request"
	"userclouds.com/infra/uclog"
)

func init() {
	jsonclient.Router = serviceRouter{}
}

type serviceRouter struct{}

func getTargetServicePort(requestPath string) int {
	svc := getTargetService(requestPath)
	if svc.IsUndefined() {
		return -1
	}
	return serviceToPort[svc]
}

func getTargetService(requestPath string) service.Service {
	for _, rule := range nonHostRules {
		for _, pathPrefix := range rule.PathPrefixes {
			if strings.HasPrefix(requestPath, pathPrefix) {
				return rule.Service
			}
		}
	}
	return service.Undefined
}

// returns the kubernetes service name for a given request path
func getTargetKubernetesService(ctx context.Context, requestPath string) string {
	svc := getTargetService(requestPath)
	return kubernetes.GetHostForService(svc)
}

func updateRequest(ctx context.Context, req *http.Request, contextHostname string, scheme string, host string, port int) {
	uclog.Debugf(ctx, "servicerouter: replacing %s://%s with https://%s:%d", req.URL.Scheme, req.URL.Host, host, port)
	req.URL.Host = fmt.Sprintf("%s:%d", host, port)
	req.URL.Scheme = scheme
	req.Host = contextHostname // set this because multitenant.Middleware on the receiving side needs it
}

// Reroute implements jsonclient.RequestRouter
func (s serviceRouter) Reroute(ctx context.Context, req *http.Request) {
	// NB: this relies on all of our services to use request.Middleware, but that seems not-crazy?
	host := request.GetHostname(ctx)
	if host == "" {
		return // no hostname in context, not rerouting
	}

	if req.Host != host {
		return // don't reroute if we're going to a different host
	}

	uv := universe.Current()
	if uv.IsDev() {
		// on dev, everything goes through devlb
		updateRequest(ctx, req, host, "https", "dev.userclouds.tools", 3333)
	} else if uv.IsKubernetes() {
		if svcHost := getTargetKubernetesService(ctx, req.URL.Path); svcHost != "" {
			updateRequest(ctx, req, host, "http", svcHost, 80)
		}
	} else if port := getTargetServicePort(req.URL.Path); port != -1 {
		// This code path is mostly for containers and in tests.
		// since in those environments we run multiple binaries in the same container/machine so each service uses a different port
		updateRequest(ctx, req, host, "http", "localhost", port)
	} else {
		uclog.Verbosef(ctx, "servicerouter: port is empty, not rerouting %s/%s", req.URL.Host, req.URL.Path)
	}
}

//go:generate genrouting
