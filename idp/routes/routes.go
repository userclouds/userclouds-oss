package routes

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/go-http-utils/headers"
	"github.com/gofrs/uuid"
	"github.com/swaggest/openapi-go/openapi3"

	"userclouds.com/idp/config"
	"userclouds.com/idp/internal/authn"
	"userclouds.com/idp/internal/datamapping"
	"userclouds.com/idp/internal/s3shim"
	"userclouds.com/idp/internal/sqlshim"
	"userclouds.com/idp/internal/sqlshim/msqlshim"
	"userclouds.com/idp/internal/sqlshim/psqlshim"
	"userclouds.com/idp/internal/tokenizer"
	"userclouds.com/idp/internal/userstore"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/middleware"
	"userclouds.com/infra/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/cors"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/oidcproviders"
	"userclouds.com/internal/resourcecheck"
	"userclouds.com/internal/security"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/throttle"
	logServerClient "userclouds.com/logserver/client"
	uehandler "userclouds.com/userevent/handler"
)

const (
	// see: helm/userclouds/templates/userstore.yaml
	envKeyDisableSQLShim = "UC_DISABLE_SQL_SHIM"

	// Number of simultaneous requests for a tenant that can be processed by the service
	activeRequests = 12
	// Number of requests for a single tenant that can be queued by the service
	backlogRequests = 5000
	// Maximum number of requests across all tenants that can be queued up before being rejected
	backlogLimit = 10000
	// Maximum time a request can be queued before being rejected
	backlogTimeout = time.Second * 30
)

func initAPIHandlers(ctx context.Context, hb *builder.HandlerBuilder,
	tenants *tenantmap.StateMap,
	companyConfigStorage *companyconfig.Storage,
	jwtVerifier auth.Verifier,
	localWorkerClient workerclient.Client,
	searchUpdateConfig *config.SearchUpdateConfig,
	cfg *config.Config,
	consoleTenantID uuid.UUID,
	enableLogServer bool,

) error {
	m2mAuth, err := m2m.GetM2MTokenSource(ctx, consoleTenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	consoleTenantInfo, err := companyConfigStorage.GetTenantInfo(ctx, consoleTenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	perTenantMiddleware := middleware.Chain(
		service.BaseMiddleware,
		cors.Middleware(),
		multitenant.Middleware(tenants),
		throttle.LimitWithBacklogQueue(ctx, activeRequests, backlogRequests, backlogLimit, backlogTimeout),
		auth.Middleware(jwtVerifier, consoleTenantID))

	authNHandler := perTenantMiddleware.Apply(authn.NewHandler(cfg, searchUpdateConfig, companyConfigStorage))
	hb.Handle("/authn/", authNHandler)

	hUserStore, err := userstore.NewHandler(ctx, cfg, searchUpdateConfig, localWorkerClient, m2mAuth, *consoleTenantInfo)
	if err != nil {
		return ucerr.Wrap(err)
	}
	userStoreHandler := perTenantMiddleware.Apply(hUserStore)
	hb.Handle("/userstore/", userStoreHandler)

	userEventHandler := perTenantMiddleware.Apply(uehandler.New())
	hb.Handle("/userevent/", userEventHandler)

	hTokenizer, err := tokenizer.NewHandler(m2mAuth, *consoleTenantInfo, enableLogServer)
	if err != nil {
		return ucerr.Wrap(err)
	}
	tokenizerHandler := perTenantMiddleware.Apply(hTokenizer)
	hb.Handle("/tokenizer/", tokenizerHandler)

	hDatamapping, err := datamapping.NewHandler(m2mAuth, *consoleTenantInfo)
	if err != nil {
		return ucerr.Wrap(err)
	}
	datamappingHandler := perTenantMiddleware.Apply(hDatamapping)
	hb.Handle("/userstore/datamapping/", datamappingHandler)

	var cacheConfig *cache.Config
	if cfg != nil {
		cacheConfig = cfg.CacheConfig
	}
	s3shimHandler := middleware.Chain(
		service.BaseMiddleware,
		cors.Middleware(),
		multitenant.Middleware(tenants)).Apply(s3shim.NewProxy(tenants, cacheConfig, jwtVerifier, companyConfigStorage))
	hb.Handle("/s3shim/", s3shimHandler)

	return nil
}

func initOpenAPIHandler(ctx context.Context, hb *builder.HandlerBuilder, path string, serviceName string, build func(ctx context.Context, reflector *openapi3.Reflector)) error {
	reflector := openapi3.Reflector{}
	reflector.Spec = &openapi3.Spec{Openapi: "3.0.3"}
	reflector.Spec.Info.
		WithTitle(serviceName).
		WithDescription(serviceName + " Service").
		WithVersion("1.0.0")
	build(ctx, &reflector)

	schemaYAML, err := reflector.Spec.MarshalYAML()
	if err != nil {
		return ucerr.Wrap(err)
	}
	hb.HandleFunc(path+"/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headers.ContentType, "application/yaml")
		w.Write(schemaYAML)
	})

	schemaJSON, err := reflector.Spec.MarshalJSON()
	if err != nil {
		return ucerr.Wrap(err)
	}
	hb.HandleFunc(path+"/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headers.ContentType, "application/json")
		w.Write(schemaJSON)
	})
	return nil
}

func initServiceHandlers(ctx context.Context, hb *builder.HandlerBuilder, companyConfigStorage *companyconfig.Storage, cfg *config.Config, workerClient workerclient.Client) error {
	bldr := builder.NewHandlerBuilder()
	bldr = resourcecheck.AddResourceCheckEndpoint(bldr, cfg.CacheConfig, cfg.OpenSearchConfig, workerClient)
	bldr = bldr.HandleFunc("/migrations", authn.CreateMigrationHandler(companyConfigStorage))
	hb.Handle("/", service.BaseMiddleware.Apply(bldr.Build()))
	if err := initOpenAPIHandler(ctx, hb, "/tokenizer", "Tokenizer", tokenizer.BuildOpenAPISpec); err != nil {
		return ucerr.Wrap(err)
	}
	if err := initOpenAPIHandler(ctx, hb, "/userstore", "Userstore", userstore.BuildOpenAPISpec); err != nil {
		return ucerr.Wrap(err)
	}
	if err := initOpenAPIHandler(ctx, hb, "/authn", "AuthN", authn.BuildOpenAPISpec); err != nil {
		return ucerr.Wrap(err)
	}
	//initOpenAPIHandler(ctx, hb, "/datamapping", "Datamapping", datamapping.BuildOpenAPISpec) // TODO: this is not ready for public yet, so leave this commented out
	return nil
}

// IsSQLShimEnabled checks if the SQL shim is enabled.
func IsSQLShimEnabled() bool {
	disableShim := os.Getenv(envKeyDisableSQLShim)
	return disableShim == "" || disableShim == "false"
}

// Init initializes IDP routes and returns an http handler.
func Init(ctx context.Context, tenants *tenantmap.StateMap, companyConfigStorage *companyconfig.Storage,
	workerClient workerclient.Client,
	searchUpdateConfig *config.SearchUpdateConfig, cfg *config.Config) (*builder.HandlerBuilder, error) {
	oidcProviderMap := oidcproviders.NewOIDCProviderMap()
	if err := oidcProviderMap.SetFallbackProviderToTenant(ctx, companyConfigStorage, cfg.ConsoleTenantID); err != nil {
		return nil, ucerr.Wrap(err)
	}
	if cfg.SQLShimConfig != nil && IsSQLShimEnabled() {
		uclog.Infof(ctx, "Starting sqlshim proxy on ports mysql %v, postgres %v", cfg.SQLShimConfig.MySQLPorts, cfg.SQLShimConfig.PostgresPorts)
		if err := runSQLShimProxy(ctx, *cfg.SQLShimConfig, cfg.CacheConfig, workerClient, cfg.ConsoleTenantID, tenants, oidcProviderMap, companyConfigStorage); err != nil {
			return nil, ucerr.Wrap(err)
		}
	} else {
		uclog.Infof(ctx, "SQLShim disable. flag: %v cfg: %v", IsSQLShimEnabled(), cfg.SQLShimConfig)
	}
	hb := builder.NewHandlerBuilder()
	if err := initAPIHandlers(ctx, hb, tenants, companyConfigStorage, oidcProviderMap, workerClient, searchUpdateConfig, cfg, cfg.ConsoleTenantID, true); err != nil {
		return nil, ucerr.Wrap(err)
	}
	if err := initServiceHandlers(ctx, hb, companyConfigStorage, cfg, workerClient); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return hb, nil
}

// InitForTests is used by integration tests to attach test IDP services to an existing serve mux.
// Only API handlers are attached, not the service handlers which are not used by most tests.
func InitForTests(hb *builder.HandlerBuilder, tenants *tenantmap.StateMap, companyConfigStorage *companyconfig.Storage, consoleTenantID uuid.UUID, jwtVerifier auth.Verifier) {
	ctx := context.Background()
	if err := initAPIHandlers(ctx, hb, tenants, companyConfigStorage, jwtVerifier, nil, nil, nil, consoleTenantID, false); err != nil {
		uclog.Fatalf(ctx, "failed to init API handlers: %v", err)
	}
}

// BuildTokenizerOpenAPISpec is a passthrough to Tokenizer's generated BuildOpenAPISpec function.
func BuildTokenizerOpenAPISpec(ctx context.Context, reflector *openapi3.Reflector) {
	tokenizer.BuildOpenAPISpec(ctx, reflector)
}

// BuildUserstoreOpenAPISpec is a passthrough to Userstore's generated BuildOpenAPISpec function.
func BuildUserstoreOpenAPISpec(ctx context.Context, reflector *openapi3.Reflector) {
	userstore.BuildOpenAPISpec(ctx, reflector)
}

// BuildAuthNOpenAPISpec is a passthrough to AuthN's generated BuildOpenAPISpec function.
func BuildAuthNOpenAPISpec(ctx context.Context, reflector *openapi3.Reflector) {
	authn.BuildOpenAPISpec(ctx, reflector)
}

func runSQLShimProxy(ctx context.Context,
	config config.SQLShimConfig,
	cacheConfig *cache.Config,
	workerClient workerclient.Client,
	consoleTenantID uuid.UUID,
	tm *tenantmap.StateMap,
	jwtVerifier auth.Verifier,
	ccs *companyconfig.Storage) error {
	m2mAuth, err := m2m.GetM2MTokenSource(ctx, consoleTenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	consoleTenantInfo, err := ccs.GetTenantInfo(ctx, consoleTenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	lgsc, err := logServerClient.NewClientForTenantAuth(consoleTenantInfo.TenantURL, consoleTenantInfo.TenantID, m2mAuth, security.PassXForwardedFor())
	if err != nil {
		return ucerr.Wrap(err)
	}

	for _, p := range config.MySQLPorts {
		uclog.Infof(ctx, "Starting MySQL proxy on port %d", p)
		proxy := sqlshim.NewProxy(p,
			tm,
			cacheConfig,
			jwtVerifier,
			ccs,
			workerClient,
			lgsc,
			msqlshim.ConnectionFactory{},
			userstore.ProxyHandlerFactory{})

		if err := proxy.Start(ctx); err != nil {
			return ucerr.Wrap(err)
		}
	}

	for _, p := range config.PostgresPorts {
		uclog.Infof(ctx, "Starting Postgres proxy on port %d", p)
		proxy := sqlshim.NewProxy(p,
			tm,
			cacheConfig,
			jwtVerifier,
			ccs,
			workerClient,
			lgsc,
			psqlshim.ConnectionFactory{},
			userstore.ProxyHandlerFactory{})

		if err := proxy.Start(ctx); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if config.HealthCheckPort != nil {
		uclog.Infof(ctx, "Starting HealthCheck proxy on port %d", *config.HealthCheckPort)
		proxy := sqlshim.NewHealthCheckProxy(*config.HealthCheckPort)
		if err := proxy.Start(ctx); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
