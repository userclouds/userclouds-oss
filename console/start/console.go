package start

import (
	"context"
	"errors"

	"userclouds.com/console/internal"
	"userclouds.com/console/routes"
	"userclouds.com/infra/logtransports"
	"userclouds.com/infra/migrate"
	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/infra/service"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/skeleton"
	"userclouds.com/internal/tenantdb"
)

func initRoutes(cfg *internal.Config) (*builder.HandlerBuilder, error) {
	ctx := context.Background()

	qc, err := workerclient.NewClientFromConfig(ctx, &cfg.WorkerClient)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	db, err := ucdb.New(ctx, &cfg.CompanyDB, migrate.SchemaValidator(companyconfig.Schema))
	if err != nil {
		var ve ucdb.ValidationError
		if errors.As(err, &ve) {
			uclog.Errorf(context.Background(), "failed to validate db schemas, loading skeleton service: %v", err)
			return skeleton.Router(ctx, &cfg.CompanyDB)
		}
		return nil, ucerr.Wrap(err)
	}

	storage, err := companyconfig.NewStorageFromConfig(ctx, &cfg.CompanyDB, cfg.CacheConfig)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	consoleTenantDB, _, _, err := tenantdb.Connect(ctx, storage, cfg.ConsoleTenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	hb, err := routes.Init(ctx, cfg, db, storage, consoleTenantDB, qc)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return hb, nil
}

// RunConsole initializes and starts the Console service.
func RunConsole() {
	ctx := context.Background()
	cfg, err := internal.LoadConfig(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "failed to load config: %v", err)
	}

	ts, err := m2m.GetM2MTokenSource(ctx, cfg.ConsoleTenantID)
	if err != nil {
		uclog.Fatalf(ctx, "failed to get m2m secret: %v", err)
	}

	if err := logtransports.InitLoggerAndTransportsForService(&cfg.Log, ts, serviceNamespace.Console, service.GetMachineName()); err != nil {
		uclog.Fatalf(ctx, "failed to initialize logger and transports: %v", err)
	}
	defer logtransports.Close()
	server, err := skeleton.InitServer(ctx, skeleton.InitServerArgs{
		Service:           serviceNamespace.Console,
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

	r, err := initRoutes(cfg)
	if err != nil {
		uclog.Fatalf(ctx, "could not init console: %v", err)
	}
	server.Run(ctx, skeleton.RunServerArgs{
		HandleBuilder:            r,
		MountPoint:               cfg.MountPoint,
		InternalServerMountPoint: cfg.InternalServerMountPoint,
	})
}
