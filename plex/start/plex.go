package start

import (
	"context"
	_ "net/http/pprof" // for profiling HTTP server

	"userclouds.com/infra/logtransports"
	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/service"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/messaging/email"
	"userclouds.com/infra/workerclient"
	"userclouds.com/infra/yamlconfig"
	_ "userclouds.com/internal/apiclient/routing" // turn on localhost routing for jsonclient
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/security"
	"userclouds.com/internal/skeleton"
	"userclouds.com/plex/routes"
	"userclouds.com/plex/serviceconfig"
)

// RunPlex initializes and starts the Plex service.
func RunPlex() {
	ctx := context.Background()
	var cfg serviceconfig.ServiceConfig
	if err := yamlconfig.LoadServiceConfig(ctx, serviceNamespace.Plex, &cfg); err != nil {
		uclog.Fatalf(ctx, "failed to load config: %v", err)
	}

	ts, err := m2m.GetM2MTokenSource(ctx, cfg.ConsoleTenantID)
	if err != nil {
		uclog.Fatalf(ctx, "failed to get m2m secret: %v", err)
	}
	if err := logtransports.InitLoggerAndTransportsForService(&cfg.Log, ts, serviceNamespace.Plex, service.GetMachineName()); err != nil {
		uclog.Fatalf(ctx, "failed to initialize logger and transports: %v", err)
	}
	defer logtransports.Close()
	server, err := skeleton.InitServer(ctx, skeleton.InitServerArgs{
		Service:           serviceNamespace.Plex,
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
		uclog.Fatalf(ctx, "failed to initialize companyconfig storage: %v", err.Error())
	}

	reqChecker := security.NewSecurityChecker()
	var emailClient *email.Client
	if !cfg.DisableEmail {
		e, err := email.NewClient(ctx)
		if err != nil {
			uclog.Errorf(ctx, "failed to initialize email client: %v", err)
		}
		emailClient = &e
	} else {
		uclog.Infof(ctx, "email client is disabled")
		emailClient = nil
	}

	var wc workerclient.Client
	if cfg.WorkerClient != nil {
		qc, err := workerclient.NewClientFromConfig(ctx, cfg.WorkerClient)
		if err != nil {
			uclog.Fatalf(ctx, "failed to initialize worker client: %v", err)
		}
		wc = qc
	} else if universe.Current().IsCloud() {
		uclog.Fatalf(ctx, "worker client is required in cloud environment")
	}
	hb := routes.Init(ctx, ts, companyConfigStorage, emailClient, &cfg, reqChecker, cfg.ConsoleTenantID, wc)
	server.Run(ctx, skeleton.RunServerArgs{
		HandleBuilder:            hb,
		MountPoint:               cfg.MountPoint,
		InternalServerMountPoint: cfg.InternalServerMountPoint,
	})
}
