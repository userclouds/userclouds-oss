package start

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/config"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/routes"
	"userclouds.com/infra/logtransports"
	serviceNamespace "userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	_ "userclouds.com/internal/apiclient/routing" // turn on localhost routing for jsonclient
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/skeleton"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/ucopensearch"
)

func loadConfig(ctx context.Context) (*config.Config, error) {
	if routes.IsSQLShimEnabled() && universe.Current().IsCloud() {
		return config.LoadDBProxyConfig(ctx)
	}
	return config.LoadConfig(ctx)
}

// RunUserStore initializes and starts the User Store service.
func RunUserStore() {
	ctx := context.Background()
	cfg, err := loadConfig(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "failed to load IDP configuration: %v", err)
	}
	ts, err := m2m.GetM2MTokenSource(ctx, cfg.ConsoleTenantID)
	if err != nil {
		uclog.Fatalf(ctx, "failed to get m2m secret: %v", err)
	}

	if err := logtransports.InitLoggerAndTransportsForService(&cfg.Log, ts, serviceNamespace.IDP, service.GetMachineName()); err != nil {
		uclog.Fatalf(ctx, "failed to initialize logger and transports: %v", err)
	}
	defer logtransports.Close()
	server, err := skeleton.InitServer(ctx, skeleton.InitServerArgs{
		Service:           serviceNamespace.IDP,
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

	tenants := tenantmap.NewStateMap(companyConfigStorage, cfg.CacheConfig)
	warmupTenantIDs, err := tenants.InitializeConnections(ctx)
	if err != nil {
		uclog.Errorf(ctx, "failed to initialize tenant connections: %v", err.Error())
	}
	// Warm up the OS connections
	if err := ucopensearch.InitializeConnections(ctx, cfg.OpenSearchConfig,
		func(ctx context.Context) (map[uuid.UUID][]ucopensearch.QueryableIndex, error) {
			return getTenantIndexMap(ctx, tenants, warmupTenantIDs)
		}); err != nil {
		uclog.Errorf(ctx, "failed to initialize/warm up OS connections: %v", err.Error())
	}
	var workerClient workerclient.Client
	if cfg.WorkerClient != nil {
		if workerClient, err = workerclient.NewClientFromConfig(ctx, cfg.WorkerClient); err != nil {
			uclog.Fatalf(ctx, "failed to initialize worker client: %v", ucerr.Wrap(err))
		}
	}

	var searchUpdateConfig *config.SearchUpdateConfig
	if cfg.OpenSearchConfig != nil {
		searchUpdateConfig = &config.SearchUpdateConfig{
			SearchCfg:    cfg.OpenSearchConfig,
			WorkerClient: workerClient,
		}
	}
	idpRoutes, err := routes.Init(ctx, tenants, companyConfigStorage, workerClient, searchUpdateConfig, cfg)
	if err != nil {
		uclog.Fatalf(ctx, "failed to initialize routes: %v", err)
	}
	server.Run(ctx, skeleton.RunServerArgs{
		HandleBuilder:            idpRoutes,
		MountPoint:               cfg.MountPoint,
		InternalServerMountPoint: cfg.InternalServerMountPoint,
	})
}

func getTenantIndexMap(ctx context.Context, tenantMap *tenantmap.StateMap, tenantIDs []uuid.UUID) (map[uuid.UUID][]ucopensearch.QueryableIndex, error) {
	tenantIndexMap := make(map[uuid.UUID][]ucopensearch.QueryableIndex)
	for _, id := range tenantIDs {
		ts, err := tenantMap.GetTenantStateForID(ctx, id)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		indices, err := storage.GetQueryableIndicesForTenant(ctx, ts)
		uclog.Verbosef(ctx, "Got %d indices for tenant %v", len(indices), id)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		tenantIndexMap[id] = indices
	}
	return tenantIndexMap, nil
}
