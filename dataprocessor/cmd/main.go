package main

import (
	"context"
	"flag"

	"github.com/gofrs/uuid"

	"userclouds.com/dataprocessor/internal"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/yamlconfig"
)

var regionArg *string
var serviceArg *string
var forceArg *bool

func initFlags(ctx context.Context) {
	flag.Usage = func() {
		uclog.Infof(ctx, "usage: bin/dataprocessor [flags] [command] <tenantid>")
		uclog.Infof(ctx, "command: provision_kinesis - creates all the aws kinesis resources for a given tenant")
		uclog.Infof(ctx, "command: deprovision_kinesis - deletes all the aws kinesis resources for a given tenant")

		flag.VisitAll(func(f *flag.Flag) {
			uclog.Infof(ctx, "    %s: %v", f.Name, f.Usage)
		})
	}

	regionArg = flag.String("region", "aws-us-west-2", "UC region (like aws-us-west-2)")
	serviceArg = flag.String("service", "plex", "userclouds service (like plex, authz, etc)")
	forceArg = flag.Bool("force", false, "ignore errors")

	flag.Parse()
}

// takes universe from tne env var, and the service name on the command line
func main() {
	ctx := context.Background()

	logtransports.InitLoggerAndTransportsForTools(ctx, uclog.LogLevelDebug, uclog.LogLevelVerbose, "dataprocessor", logtransports.NoPrefix())
	defer logtransports.Close()

	var cfg internal.Config
	if err := yamlconfig.LoadToolConfig(ctx, "dataprocessor", &cfg); err != nil {
		uclog.Fatalf(ctx, "failed to load config: %v", err)
	}

	initFlags(ctx)

	if flag.NArg() != 2 {
		flag.Usage()
		uclog.Fatalf(ctx, "error: expected 2 non-flag args, got %d: %v", flag.NArg(), flag.Args())
	}

	// TODO implement background row compression for older rows

	commandName := flag.Arg(0)

	tenantID := uuid.Nil
	var err error
	if flag.NArg() == 2 {
		tenantID, err = uuid.FromString(flag.Arg(1))
		if err != nil {
			uclog.Fatalf(ctx, "error: couldn't parse specified tenant_id, got %s", flag.Arg(1))
		}
	}

	// Validate region and service
	awsRegion := "us-west-2"
	if regionArg != nil && *regionArg != "" {
		reg := region.MachineRegion(*regionArg)
		if err := reg.Validate(); err != nil {
			uclog.Fatalf(ctx, "invalid region %v -- must be one of %v", *regionArg, region.MachineRegionsForUniverse(universe.Current()))
		}
		awsRegion = region.GetAWSRegion(reg)
	}

	if serviceArg != nil && *serviceArg != "" && !service.IsValid(service.Service(*serviceArg)) {
		uclog.Fatalf(ctx, "invalid service %v -- must be one of %v", *serviceArg, service.AllServices)
	}
	svc := service.Service(*serviceArg)

	uclog.Debugf(ctx, "Dataprocessor started with command %s on %v", commandName, tenantID)

	switch commandName {
	case "provision_kinesis":
		err = internal.ProvisionKinesisRegionResourcesForService(ctx, &cfg, tenantID, awsRegion, svc, *forceArg)
	case "deprovision_kinesis":
		err = internal.DeProvisionKinesisRegionResourcesForService(ctx, &cfg, tenantID, awsRegion, svc, *forceArg)
	default:
		uclog.Fatalf(ctx, "error: unknown command - %s", commandName)
	}

	if err != nil {
		uclog.Fatalf(ctx, "error: command %s on %v failed %v", commandName, tenantID, err)
	}
}
