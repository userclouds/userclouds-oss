package main

import (
	"context"
	"os"

	"sigs.k8s.io/yaml"

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/routinghelper"
)

const (
	loopbackDomain = "test.userclouds.tools"
)

func main() {
	ctx := context.Background()
	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "exportrouting")
	defer logtransports.Close()
	if len(os.Args) < 2 {
		uclog.Debugf(ctx, "Usage: exportrouting [filename] [list of services]")
		uclog.Fatalf(ctx, "Expected a filename, instead got %d: %v", len(os.Args), os.Args)
	}

	filename := os.Args[1]
	includeServicesStrings := os.Args[2:]
	if len(includeServicesStrings) > 0 {
		uclog.Infof(ctx, "Exporting rules for: %v", includeServicesStrings)
	} else {
		uclog.Infof(ctx, "Exporting rules all for services")
	}
	includeServices := make([]service.Service, 0, len(includeServicesStrings))
	for _, svcName := range includeServicesStrings {
		svc := service.Service(svcName)
		if !service.IsValid(svc) {
			uclog.Fatalf(ctx, "Invalid service: %v", svc)
		}
		includeServices = append(includeServices, svc)
	}

	routeCfg, err := routinghelper.ParseIngressConfig(ctx, universe.Current(), loopbackDomain, "mars", includeServices)
	if err != nil {
		uclog.Fatalf(ctx, "failed to parse routing config: %v", err)
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		uclog.Fatalf(ctx, "failed to open file: %v", err)
	}
	defer file.Close()
	uclog.Infof(ctx, "Exporting %v rules into: %v", len(routeCfg.Rules), filename)
	data, err := yaml.Marshal(routeCfg)
	if err != nil {
		uclog.Fatalf(ctx, "failed to encode: %v", err)
	}
	if _, err := file.Write(data); err != nil {
		uclog.Fatalf(ctx, "failed to write output: %v", err)
	}
}
