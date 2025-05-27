package start

import (
	"context"
	_ "net/http/pprof" // for profiling HTTP server

	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/migrate"
	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/infra/service"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/logdb"
	"userclouds.com/internal/logeventmetadata"
	"userclouds.com/internal/skeleton"
	"userclouds.com/logserver"
	"userclouds.com/logserver/internal/countersbackend"
	"userclouds.com/logserver/internal/instancebackend"
	"userclouds.com/logserver/internal/kinesisbackend"
	"userclouds.com/logserver/internal/storage"
	"userclouds.com/logserver/routes"
)

// RunLogServer initializes and starts the LogServer service.
func RunLogServer() {
	ctx := context.Background()
	cfg, err := logserver.LoadConfig(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "failed to load config: %v", err)
	}

	if err := logtransports.InitLoggerAndTransportsForService(&cfg.Log, nil, serviceNamespace.LogServer, service.GetMachineName()); err != nil {
		uclog.Fatalf(ctx, "failed to initialize logger and transports: %v", err)
	}
	defer logtransports.Close()
	server, err := skeleton.InitServer(ctx, skeleton.InitServerArgs{
		Service:           serviceNamespace.LogServer,
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

	globalEventMetadataStorage, err := logeventmetadata.NewStorageFromConfig(ctx, &cfg.CompanyDB)
	if err != nil {
		uclog.Fatalf(ctx, "failed to create global event metadata storage: %v", err)
	}

	// Initialize the backend services
	tenantCache, err := storage.NewTenantStorageCache(ctx,
		&cfg.DefaultLogDB,
		companyConfigStorage,
		globalEventMetadataStorage,
		cfg.ConsoleTenantID,
		migrate.SchemaValidator(logdb.Schema))
	if err != nil {
		uclog.Fatalf(ctx, "failed to create tenant cache: %v", err)
	}
	var kinesisBE *kinesisbackend.KinesisConnections
	if cfg.KinesisAWSRegion != "" {
		kinesisBE, err = kinesisbackend.NewKinesisConnections(ctx, cfg.KinesisAWSRegion)
		if err != nil {
			uclog.Fatalf(ctx, "failed to create kinesis backend: %v", err)
		}
	} else {
		uclog.Infof(ctx, "Kinesis backend not initialized, KinesisAWSRegion is empty")
	}

	countersBE, err := countersbackend.NewCounterStore(ctx, tenantCache)
	if err != nil {
		uclog.Fatalf(ctx, "failed to create counters backend: %v", err)
	}

	activityBE, err := instancebackend.NewInstanceActivityStore(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "failed to create instance activity backend: %v", err)
	}

	hb := routes.Init(companyConfigStorage, cfg.ConsoleTenantID, cfg, tenantCache, kinesisBE, countersBE, activityBE)
	server.Run(ctx, skeleton.RunServerArgs{
		HandleBuilder:            hb,
		MountPoint:               cfg.MountPoint,
		InternalServerMountPoint: cfg.InternalServerMountPoint,
	})
}
