package main

import (
	"context"
	"net/http"
	"path/filepath"

	"userclouds.com/cmd/devlb/internal"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/yamlconfig"
	"userclouds.com/internal/localrouting"
	"userclouds.com/internal/repopath"
	"userclouds.com/internal/routinghelper"
)

func main() {
	ctx := context.Background()

	var cfg internal.Config
	if err := yamlconfig.LoadToolConfig(ctx, "devlb", &cfg); err != nil {
		uclog.Fatalf(ctx, "failed to load config: %v", err)
	}

	routeCfg, err := routinghelper.ParseIngressConfig(ctx, universe.Dev, "dev.userclouds.tools", "dev-region", service.AllWebServices)
	if err != nil {
		uclog.Fatalf(ctx, "failed to parse routing config: %v", err)
	}
	if err := logtransports.InitLoggerAndTransportsForService(&cfg.Log, nil, "devlb", "localdev"); err != nil {
		uclog.Fatalf(ctx, "failed to initialize logger and transports: %v", err)
	}
	defer logtransports.Close()

	http.DefaultTransport.(*http.Transport).MaxIdleConns = 10000
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 10000

	// set up our proxies keyed to the service names in as defined helm chart Ingress object
	proxies, err := localrouting.NewProxiesMap(ctx, routeCfg, service.AllWebServices)
	if err != nil {
		uclog.Fatalf(ctx, "failed to create proxies: %v", err)
	}

	// NOTE: we intentionally do not use uchttp.NewServeMux here, since the uchttp-wrapped
	// handler code strips the initial prefix from the route
	mux := http.NewServeMux()
	if err := proxies.AddRulesToServer(ctx, "devlb", routeCfg.Rules, mux); err != nil {
		uclog.Fatalf(ctx, "failed to add rules to server: %v", err)
	}

	http.Handle("/", mux)
	uclog.Debugf(ctx, "Dev Load Balancer listening on %s", cfg.BaseURL())
	rootPath := repopath.BaseDir()
	uclog.Fatalf(ctx, "%v", http.ListenAndServeTLS(cfg.HostAndPort(), filepath.Join(rootPath, "cert/devlb.crt"), filepath.Join(rootPath, "cert/devlb.key"), nil))
}
