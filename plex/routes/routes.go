package routes

import (
	"context"
	"net/http"
	"os"

	"github.com/gofrs/uuid"
	"github.com/swaggest/openapi-go/openapi3"

	"userclouds.com/infra/acme"
	"userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/middleware"
	"userclouds.com/infra/service"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/ucreact"
	"userclouds.com/infra/uctypes/messaging/email"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/cors"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/oidcproviders"
	"userclouds.com/internal/security"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/testkeys"
	"userclouds.com/plex/internal"
	"userclouds.com/plex/internal/create"
	"userclouds.com/plex/internal/delegation"
	"userclouds.com/plex/internal/invite"
	"userclouds.com/plex/internal/loginapp"
	"userclouds.com/plex/internal/oidc"
	"userclouds.com/plex/internal/otp"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/resetpassword"
	"userclouds.com/plex/internal/saml"
	"userclouds.com/plex/internal/social"
	"userclouds.com/plex/internal/tenantconfig"
	"userclouds.com/plex/internal/wellknown"
	"userclouds.com/plex/manager"
	"userclouds.com/plex/serviceconfig"
)

func initAPIHandlers(
	ctx context.Context,
	m2mAuth jsonclient.Option,
	hb *builder.HandlerBuilder,
	cfg *serviceconfig.ServiceConfig,
	companyConfigStorage *companyconfig.Storage,
	jwtVerifier auth.Verifier,
	reqChecker security.ReqValidator,
	emailClient *email.Client,
	qc workerclient.Client,
	consoleTenantID uuid.UUID,
	handlerOpts ...internal.Option) {

	tenants := tenantmap.NewStateMap(companyConfigStorage, cfg.CacheConfig)

	consoleTenantInfo, err := companyConfigStorage.GetTenantInfo(ctx, consoleTenantID)
	if err != nil {
		uclog.Fatalf(ctx, "failed to get console tenant info: %v", err)
	}

	if _, err := tenants.InitializeConnections(ctx); err != nil {
		uclog.Errorf(ctx, "Failed to initialize tenant connections: %v", err)
	}

	// pre-load signing keys into local memory for oidc perf
	// NB: after InitializeConnections is called, the DB connections are established so cheap
	warmupTenants, err := companyConfigStorage.GetConnectOnStartupTenants(ctx)
	if err != nil {
		uclog.Fatalf(ctx, "failed to get warmup tenants: %v", err)
	}

	for _, tenant := range warmupTenants {
		ts, err := tenants.GetTenantStateForID(ctx, tenant.ID)
		if err != nil {
			uclog.Fatalf(ctx, "failed to get warmup tenant %v state: %v", tenant.ID, err)
		}
		mgr := manager.NewFromDB(ts.TenantDB, cfg.CacheConfig)
		tp, err := mgr.GetTenantPlex(ctx, tenant.ID)
		if err != nil {
			uclog.Fatalf(ctx, "failed to get warmup tenant %v plex: %v", tenant.ID, err)
		}
		_, err = tp.PlexConfig.Keys.PrivateKey.Resolve(ctx)
		if err != nil {
			uclog.Fatalf(ctx, "failed to resolve warmup tenant %v private key: %v", tenant.ID, err)
		}
		uclog.Debugf(ctx, "pre-loaded warmup tenant %v private key", tenant.ID)
	}

	tcCache := tenantconfig.NewCache(companyConfigStorage)

	consoleTenantState, err := tenants.GetTenantStateForID(ctx, consoleTenantID)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get console tenant state: %v", err)
	}
	mgr := manager.NewFromDB(consoleTenantState.TenantDB, consoleTenantState.CacheConfig)
	tp, err := mgr.GetTenantPlex(ctx, consoleTenantID)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to get console tenant state: %v", err)
	}
	consolePK, err := ucjwt.LoadRSAPublicKey([]byte(tp.PlexConfig.Keys.PublicKey))
	if err != nil {
		uclog.Fatalf(context.Background(), "failed to load console tenant public key: %v", err)
	}

	// NB: The shared multitenant.Middleware resolves the tenant from a request's Host header,
	// and then the tenantconfig.Middleware loads & caches the Plex-specific configs.
	perTenantMiddleware := middleware.Chain(
		service.BaseMiddleware,
		cors.Middleware(),
		security.Middleware(reqChecker),
		social.Middleware(),
		multitenant.Middleware(tenants),
		tenantconfig.Middleware(tcCache))
	var consoleEP *service.Endpoint
	if cfg.IsConsoleEndpointDefined() {
		consoleEP, err = cfg.GetConsoleEndpoint()
		if err != nil {
			uclog.Fatalf(ctx, "Failed to get console endpoint: %v", err)
		}
	}
	plexHandler, err := internal.NewHandler(companyConfigStorage, jwtVerifier, reqChecker, emailClient, tcCache, qc, m2mAuth, *consoleTenantInfo, consoleEP, consolePK, handlerOpts...)
	if err != nil {
		uclog.Fatalf(ctx, "Failed to create Plex handler: %v", err)
	}
	hb.Handle("/", perTenantMiddleware.Apply(plexHandler))

	wellKnownHandler := perTenantMiddleware.Apply(wellknown.NewHandler(cfg.ACME, qc))
	hb.Handle("/.well-known/", wellKnownHandler)

	// Static assets for new Plex UI (React app), but only if we're not running react dev server
	if os.Getenv("UC_PLEX_UI_DEV_PORT") == "" {
		hb.Handle(paths.PlexUIRoot, perTenantMiddleware.Apply(ucreact.NewHandler(cfg.StaticAssetsPath)))
	}
}

// Init initializes Plex routes and returns an http handler.
func Init(
	ctx context.Context,
	m2mAuth jsonclient.Option,
	companyConfigStorage *companyconfig.Storage,
	emailClient *email.Client,
	cfg *serviceconfig.ServiceConfig,
	reqChecker security.ReqValidator,
	consoleTenantID uuid.UUID,
	qc workerclient.Client) *builder.HandlerBuilder {

	hb := builder.NewHandlerBuilder()

	oidcProviderMap := oidcproviders.NewOIDCProviderMap()
	initAPIHandlers(ctx, m2mAuth, hb, cfg, companyConfigStorage, oidcProviderMap, reqChecker, emailClient, qc, consoleTenantID)

	// 404 for favicon.ico to avoid a bunch of log spew, redirects, etc.
	hb.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	return hb
}

// InitForTests is used by integration tests to attach test Plex services to an existing serve mux.
// Only API handlers are attached, not the service handlers which are not used by most tests.
func InitForTests(
	ctx context.Context,
	m2mAuth jsonclient.Option,
	hb *builder.HandlerBuilder,
	companyConfigStorage *companyconfig.Storage,
	jwtVerifier auth.Verifier,
	reqChecker security.ReqValidator,
	emailClient email.Client,
	factory provider.Factory,
	wc workerclient.Client,
	consoleTenantID uuid.UUID,
) {
	plexCfg := serviceconfig.ServiceConfig{
		CacheConfig: testhelpers.NewRedisConfigForTests(),
		ConsoleURL:  "http://fake-console-url-for-plex-in-tests.jerry.seinfeld.net",
		// just mute the thumbprint computation warnings in tests
		ACME: &acme.Config{PrivateKey: testkeys.Config.PrivateKey},
	}
	initAPIHandlers(ctx, m2mAuth, hb, &plexCfg, companyConfigStorage, jwtVerifier, reqChecker, &emailClient, wc, consoleTenantID, internal.Factory(factory))
}

// BuildOpenAPISpec is a passthrough to the generated BuildOpenAPISpec function.
func BuildOpenAPISpec(ctx context.Context, reflector *openapi3.Reflector) {
	create.BuildOpenAPISpec(ctx, reflector)
	delegation.BuildOpenAPISpec(ctx, reflector)
	invite.BuildOpenAPISpec(ctx, reflector)
	loginapp.BuildOpenAPISpec(ctx, reflector)
	oidc.BuildOpenAPISpec(ctx, reflector)
	otp.BuildOpenAPISpec(ctx, reflector)
	resetpassword.BuildOpenAPISpec(ctx, reflector)
	saml.BuildOpenAPISpec(ctx, reflector)
	social.BuildOpenAPISpec(ctx, reflector)
}
