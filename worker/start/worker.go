package start

import (
	"context"
	"net/http"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/logtransports"
	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/service"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/resourcecheck"
	"userclouds.com/internal/skeleton"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/worker/config"
	"userclouds.com/worker/internal"
	"userclouds.com/worker/internal/acme"
	"userclouds.com/worker/internal/cleanup"
	"userclouds.com/worker/internal/usersync"
	"userclouds.com/worker/internal/watchdog"
)

// RunWorker initializes and starts the Worker service.
func RunWorker() {
	ctx := context.Background()
	cfg, err := config.LoadConfig()
	if err != nil {
		uclog.Fatalf(ctx, "failed to load config: %v", err)
	}

	m2mAuth, err := m2m.GetM2MTokenSource(ctx, cfg.ConsoleTenantID)
	if err != nil {
		uclog.Fatalf(ctx, "failed to get m2m secret: %v", err)
	}
	if err := logtransports.InitLoggerAndTransportsForService(&cfg.Log, m2mAuth, serviceNamespace.Worker, service.GetMachineName()); err != nil {
		uclog.Fatalf(ctx, "failed to initialize logger and transports: %v", err)
	}
	defer logtransports.Close()
	server, err := skeleton.InitServer(ctx, skeleton.InitServerArgs{
		Service:           serviceNamespace.Worker,
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

	wc, err := workerclient.NewClientFromConfig(ctx, &cfg.WorkerClient)
	if err != nil {
		uclog.Fatalf(ctx, "failed to create worker client: %v", err)
	}
	consoleTenantInfo, err := companyConfigStorage.GetTenantInfo(ctx, cfg.ConsoleTenantID)
	if err != nil {
		uclog.Fatalf(ctx, "failed to get console tenant info: %v", err)
	}
	runningTasks := internal.NewRunningTasks()
	baseMiddleware := service.BaseMiddleware
	hb := resourcecheck.AddResourceCheckEndpoint(builder.NewHandlerBuilder(), cfg.CacheConfig, cfg.OpenSearchConfig, nil)

	tm := tenantmap.NewStateMap(companyConfigStorage, cfg.CacheConfig)
	if i, _ := cache.RunCrossRegionInvalidations(ctx, cfg.CacheConfig, cache.RegionalRedisCacheName, cache.GlobalRedisCacheName); i == nil {
		uclog.Warningf(ctx, "Not running cross region invalidations propagator")
	}
	// In Kubernetes/UC Cloud, we use SQS to poll for messages
	if universe.Current().IsCloud() {
		if err := internal.StartLongPollSQS(ctx, cfg, companyConfigStorage, m2mAuth, *consoleTenantInfo, tm, wc, runningTasks); err != nil {
			uclog.Fatalf(ctx, "failed to start SQS polling: %v", err)
		}
	} else {
		// In Dev, On Prem, we use HTTP to receive messages, since in those envs we don't have SQS, instead the worker client just does HTTP POST to this endpoint
		uclog.Infof(ctx, "Start HTTP Handler for worker messages")
		hb.Handle("/", baseMiddleware.Apply(internal.NewHTTPHandler(cfg, companyConfigStorage, m2mAuth, *consoleTenantInfo, tm, wc, runningTasks)))
	}

	addCronEndpoints(companyConfigStorage, tm, wc, hb)
	hb.Handle("/running", baseMiddleware.Apply(runningTasks.GetHTTPHandler()))
	server.Run(ctx, skeleton.RunServerArgs{
		HandleBuilder:            hb,
		MountPoint:               cfg.MountPoint,
		InternalServerMountPoint: cfg.InternalServerMountPoint,
	})
}

func addCronEndpoints(companyConfigStorage *companyconfig.Storage, tm *tenantmap.StateMap, wc workerclient.Client, hb *builder.HandlerBuilder) {
	// each of these corresponds to a separate cronjob in helm/userclouds/templates/worker-cronjobs.yaml
	addCronEndPoint(hb, "/syncall", usersync.All(companyConfigStorage, tm, wc))
	addCronEndPoint(hb, "/checkcnames", acme.CheckAllCNAMEsHandler(companyConfigStorage, tm, wc))
	addCronEndPoint(hb, "/watchdog/slowprov", watchdog.SlowProvisionWatchdog(companyConfigStorage))
	addCronEndPoint(hb, "/clean-userstore-data", cleanup.CleanUserStoreForAllTenantsHandler(companyConfigStorage, wc))
}

func addCronEndPoint(hb *builder.HandlerBuilder, endpoint string, handler http.Handler) {
	hb.Handle(endpoint, service.BaseMiddleware.Apply(handler))
}
