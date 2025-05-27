package start

import (
	"context"
	"fmt"
	"os"

	"github.com/gofrs/uuid"

	"userclouds.com/authz/config"
	routes "userclouds.com/authz/routes-checkattribute"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/namespace/region"
	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/infra/service"
	"userclouds.com/infra/uclog"
	_ "userclouds.com/internal/apiclient/routing" // turn on localhost routing for jsonclient
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/skeleton"
	"userclouds.com/internal/tenantmap"
)

const (
	// see: helm/userclouds/templates/checkattribute.yaml
	envServiceInstanceName = "SERVICE_INSTANCE_NAME"
)

// RunCheckAttribute initializes and starts the Check Attribute service.
func RunCheckAttribute() {
	ctx := context.Background()

	cfg, err := config.LoadCheckAttributeConfig(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "LoadCheckAttributeConfig failed: %v", err)
	}

	ts, err := m2m.GetM2MTokenSource(ctx, cfg.ConsoleTenantID)
	if err != nil {
		uclog.Fatalf(ctx, "failed to get m2m secret: %v", err)
	}
	if err := logtransports.InitLoggerAndTransportsForService(&cfg.Log, ts, serviceNamespace.CheckAttribute, service.GetMachineName()); err != nil {
		uclog.Fatalf(ctx, "failed to initialize logger and transports: %v", err)
	}
	defer logtransports.Close()

	serviceInstanceName := os.Getenv(envServiceInstanceName) // k8s/cloud only
	server, err := skeleton.InitServer(ctx, skeleton.InitServerArgs{
		Service:             serviceNamespace.CheckAttribute,
		FeatureFlagConfig:   cfg.FeatureFlagConfig,
		SentryConfig:        cfg.Sentry,
		TracingConfig:       cfg.Tracing,
		CacheConfig:         cfg.CacheConfig,
		ConsoleTenantID:     cfg.ConsoleTenantID,
		ServiceConfig:       cfg,
		ServiceInstanceName: serviceInstanceName,
	})
	if err != nil {
		uclog.Fatalf(ctx, "Server initialization failed: %v", err)
	}

	companyConfigStorage, err := companyconfig.NewStorageFromConfig(ctx, &cfg.CompanyDB, cfg.CacheConfig)
	if err != nil {
		uclog.Fatalf(ctx, "failed to create company config storage: %v", err)
	}
	tenants := tenantmap.NewStateMap(companyConfigStorage, cfg.CacheConfig)
	tenantsHandled, err := getHandledTenants(ctx, cfg, serviceInstanceName, region.Current())
	if err != nil {
		uclog.Fatalf(ctx, "failed to get handled tenants: %v", err)
	}
	hb, err := routes.Init(ctx, tenants, *cfg, tenantsHandled)
	if err != nil {
		uclog.Fatalf(ctx, "failed to initialize routes: %v", err)
	}

	server.Run(ctx, skeleton.RunServerArgs{
		HandleBuilder:            hb,
		MountPoint:               cfg.MountPoint,
		InternalServerMountPoint: cfg.InternalServerMountPoint,
	})
}

func getHandledTenants(ctx context.Context, cfg *config.Config, instanceName string, rg region.MachineRegion) ([]uuid.UUID, error) {
	var serviceName string
	if instanceName == "" {
		serviceName = string(serviceNamespace.CheckAttribute)
	} else {
		serviceName = fmt.Sprintf("%v-%s", serviceNamespace.CheckAttribute, instanceName)
	}
	tenantsHandled := cfg.GetHandledTenants(serviceName, rg)
	uclog.Infof(ctx, "[%s/%v] handled %d tenants from config: %v", serviceName, rg, len(tenantsHandled), tenantsHandled)
	return tenantsHandled, nil
}
