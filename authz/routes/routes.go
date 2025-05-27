package routes

import (
	"context"
	"net/http"
	"time"

	"github.com/go-http-utils/headers"
	"github.com/gofrs/uuid"
	"github.com/swaggest/openapi-go/openapi3"

	"userclouds.com/authz/config"
	"userclouds.com/authz/internal/api"
	"userclouds.com/infra/middleware"
	"userclouds.com/infra/service"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auditlog/handler"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/cors"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/oidcproviders"
	"userclouds.com/internal/resourcecheck"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/throttle"
)

const (
	// Number of simultaneous requests for a tenant that can be processed by the service
	activeRequests = 25
	// Number of requests for a tenant that can be queue by the service
	backlogRequests = 5000
	// Maximum number of requests that can be queued up before being rejected
	backlogLimit = 10000
	// Maximum time a request can be queued before being rejected
	backlogTimeout = time.Second * 30
)

func initAPIHandlers(ctx context.Context,
	hb *builder.HandlerBuilder,
	tenants *tenantmap.StateMap,
	storage *companyconfig.Storage,
	jwtVerifier auth.Verifier,
	consoleTenantID uuid.UUID,
	cfg config.Config) {

	perTenantMiddleware := middleware.Chain(
		service.BaseMiddleware,
		cors.Middleware(),
		multitenant.Middleware(tenants),
		throttle.LimitWithBacklogQueue(ctx, activeRequests, backlogRequests, backlogLimit, backlogTimeout),
		auth.Middleware(jwtVerifier, consoleTenantID))

	authzHandler := perTenantMiddleware.Apply(api.NewHandler(ctx, storage, cfg))
	hb.Handle("/authz/", authzHandler)

	auditLogHandler := perTenantMiddleware.Apply(handler.New())
	hb.Handle(auditlog.BasePathSegment+"/", auditLogHandler)
}

func initServiceHandlers(ctx context.Context, hb *builder.HandlerBuilder, cfg config.Config) {
	bldr := builder.NewHandlerBuilder()
	bldr = resourcecheck.AddResourceCheckEndpoint(bldr, cfg.CacheConfig, nil, nil)
	hb.Handle("/", service.BaseMiddleware.Apply(bldr.Build()))

	reflector := openapi3.Reflector{}
	reflector.Spec = &openapi3.Spec{Openapi: "3.0.3"}
	reflector.Spec.Info.
		WithTitle("AuthZ").
		WithDescription("AuthZ Service").
		WithVersion("1.0.0")
	BuildAuthZOpenAPISpec(ctx, &reflector)

	schemaYAML, err := reflector.Spec.MarshalYAML()
	if err != nil {
		uclog.Fatalf(ctx, "failed to marshal OpenAPI spec to YAML: %v", err)
	}
	hb.HandleFunc("/authz/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headers.ContentType, "application/yaml")
		w.Write(schemaYAML)
	})

	schemaJSON, err := reflector.Spec.MarshalJSON()
	if err != nil {
		uclog.Fatalf(ctx, "failed to marshal OpenAPI spec to JSON: %v", err)
	}
	hb.HandleFunc("/authz/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headers.ContentType, "application/json")
		w.Write(schemaJSON)
	})
}

// Init initializes AuthZ routes and returns an http handler.
func Init(ctx context.Context, tenants *tenantmap.StateMap, storage *companyconfig.Storage, cfg config.Config) *builder.HandlerBuilder {
	oidcProviderMap := oidcproviders.NewOIDCProviderMap()
	if err := oidcProviderMap.SetFallbackProviderToTenant(ctx, storage, cfg.ConsoleTenantID); err != nil {
		uclog.Fatalf(ctx, "failed to set up Console as fallback JWT verifier: %v", err)
	}

	hb := builder.NewHandlerBuilder()
	initAPIHandlers(ctx, hb, tenants, storage, oidcProviderMap, cfg.ConsoleTenantID, cfg)
	initServiceHandlers(ctx, hb, cfg)
	return hb
}

// InitForTests is used by integration tests to attach test AuthZ services to an existing serve mux.
// Only API handlers are attached, not the service handlers which are not used by most tests.
func InitForTests(hb *builder.HandlerBuilder, tenants *tenantmap.StateMap, storage *companyconfig.Storage, jwtVerifier auth.Verifier) {
	ctx := context.Background()
	initAPIHandlers(ctx, hb, tenants, storage, jwtVerifier, uuid.Nil, config.Config{})
}

// BuildAuthZOpenAPISpec is a passthrough to the generated BuildOpenAPISpec function.
func BuildAuthZOpenAPISpec(ctx context.Context, reflector *openapi3.Reflector) {
	api.BuildOpenAPISpec(ctx, reflector)
}
