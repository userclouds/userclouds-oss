package start

import (
	"context"

	"userclouds.com/authz/config"
	"userclouds.com/authz/routes"
	"userclouds.com/infra/logtransports"
	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/infra/service"
	"userclouds.com/infra/uclog"
	_ "userclouds.com/internal/apiclient/routing" // turn on localhost routing for jsonclient
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/skeleton"
	"userclouds.com/internal/tenantmap"
)

// RunAuthZ initializes and starts the AuthZ service.
func RunAuthZ() {
	ctx := context.Background()
	cfg, err := config.LoadAuthzConfig(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "LoadAuthzConfig failed: %v", err)
	}

	ts, err := m2m.GetM2MTokenSource(ctx, cfg.ConsoleTenantID)
	if err != nil {
		uclog.Fatalf(ctx, "failed to get m2m secret: %v", err)
	}
	if err := logtransports.InitLoggerAndTransportsForService(&cfg.Log, ts, serviceNamespace.AuthZ, service.GetMachineName()); err != nil {
		uclog.Fatalf(ctx, "failed to initialize logger and transports: %v", err)
	}
	defer logtransports.Close()
	server, err := skeleton.InitServer(ctx, skeleton.InitServerArgs{
		Service:           serviceNamespace.AuthZ,
		FeatureFlagConfig: cfg.FeatureFlagConfig,
		SentryConfig:      cfg.Sentry,
		TracingConfig:     cfg.Tracing,
		CacheConfig:       cfg.CacheConfig,
		ConsoleTenantID:   cfg.ConsoleTenantID,
		ServiceConfig:     cfg,
	})
	if err != nil {
		uclog.Fatalf(ctx, "Server initialization failed: %v", err)
	}

	companyConfigStorage, err := companyconfig.NewStorageFromConfig(ctx, &cfg.CompanyDB, cfg.CacheConfig)
	if err != nil {
		uclog.Fatalf(ctx, "failed to create company config storage: %v", err)
	}

	tenants := tenantmap.NewStateMap(companyConfigStorage, cfg.CacheConfig)
	if _, err := tenants.InitializeConnections(ctx); err != nil {
		uclog.Errorf(ctx, "failed to initialize tenant connections: %v", err.Error())
	}
	hb := routes.Init(ctx, tenants, companyConfigStorage, *cfg)
	server.Run(ctx, skeleton.RunServerArgs{
		HandleBuilder:            hb,
		MountPoint:               cfg.MountPoint,
		InternalServerMountPoint: cfg.InternalServerMountPoint,
	})
}
