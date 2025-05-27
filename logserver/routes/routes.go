package routes

import (
	"context"
	"net/http"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/middleware"
	"userclouds.com/infra/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/logserver/config"
	"userclouds.com/logserver/internal"
	"userclouds.com/logserver/internal/countersbackend"
	"userclouds.com/logserver/internal/instancebackend"
	"userclouds.com/logserver/internal/kinesisbackend"
	tmiddleware "userclouds.com/logserver/internal/middleware"
	"userclouds.com/logserver/internal/storage"
)

// defaultVerifier is a simple non per tenant verifier for backend only calls
type defaultVerifier struct {
	dProvider    *oidc.Provider
	dProviderURL string
	tc           *storage.TenantCache
	mutex        sync.Mutex
}

// VerifyAndDecode implements ucjwt.Verifier
func (v *defaultVerifier) VerifyAndDecode(ctx context.Context, rawJWT string) (*oidc.IDToken, error) {
	if err := v.initializeProvider(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	cfg := &oidc.Config{
		SkipClientIDCheck: true,
	}

	decodedJWT, err := v.dProvider.Verifier(cfg).Verify(ctx, rawJWT)
	if err != nil {
		if tenantID := tmiddleware.GetTenantID(ctx); tenantID != uuid.Nil {
			var provider *oidc.Provider
			if provider, err = v.tc.GetProviderForTenant(ctx, tenantID); err != nil {
				return nil, ucerr.Wrap(err)
			}

			// Disable the Client ID check in favor of the audience check further down.
			cfg := &oidc.Config{
				SkipClientIDCheck: true,
			}

			decodedJWT, err = provider.Verifier(cfg).Verify(ctx, rawJWT)
			if err != nil {
				uclog.Warningf(ctx, "error verifying JWT against both userclouds and per tenant %v provider: %+v", tenantID, err)
				return nil, ucerr.Wrap(err)
			}
		} else {
			uclog.Warningf(ctx, "error verifying JWT against both userclouds and per tenant %v provider: %+v", tenantID, err)
			return nil, ucerr.Wrap(err)
		}
	} else {
		// The call is not restricted to a tenant
		tmiddleware.ClearTenantID(ctx)
	}

	return decodedJWT, nil
}

// Init the JWT verification logic so that the given tenant is a "special" provider whose
// tokens can be trusted for any other tenant.
func (v *defaultVerifier) setProviderURL(ctx context.Context, storage *companyconfig.Storage, tenantID uuid.UUID) error {

	tenant, err := storage.GetTenant(ctx, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// just set the URL and we'll lazy-load it later to make startup more predictable
	v.dProviderURL = tenant.TenantURL

	return nil
}

// initializeProvider
func (v *defaultVerifier) initializeProvider() error {
	if v.dProvider != nil {
		return nil
	}

	// Need to initialize the provider
	v.mutex.Lock()
	defer v.mutex.Unlock()
	// Check if another thread initialized the provider while this thread was waiting for the lock
	if v.dProvider != nil {
		return nil
	}
	provider, err := oidc.NewProvider(context.Background(), v.dProviderURL)
	if err != nil {
		return ucerr.Wrap(err)
	}
	v.dProvider = provider

	return nil
}

func initAPIHandlers(hb *builder.HandlerBuilder, jwtVerifier auth.Verifier,
	cfg *config.Config, tenantCache *storage.TenantCache, kinesisBE *kinesisbackend.KinesisConnections,
	counterBE *countersbackend.CountersStore, activityBE *instancebackend.InstanceActivityStore) {

	authedMiddleware := middleware.Chain(
		service.BaseMiddleware,
		tmiddleware.TenantID(cfg.ConsoleTenantID),
		auth.Middleware(jwtVerifier, cfg.ConsoleTenantID))

	authedMux, openMux := internal.NewHandler(cfg, tenantCache, kinesisBE, counterBE, activityBE)
	eventMapHandler := service.BaseMiddleware.Apply(openMux)
	hb.Handle("/eventmetadata/", eventMapHandler)
	logServerHandler := authedMiddleware.Apply(authedMux)
	hb.Handle("/", logServerHandler)
}

func initServiceHandlers(hb *builder.HandlerBuilder, storage *companyconfig.Storage) {
	hb.Handle("/migrations", service.BaseMiddleware.Apply(
		http.HandlerFunc(internal.CreateMigrationHandler(storage))))
}

// Init initializes logserver routes and returns an http handler.
func Init(companyConfigStorage *companyconfig.Storage, consoleTenantID uuid.UUID,
	cfg *config.Config, tenantCache *storage.TenantCache, kinesisBE *kinesisbackend.KinesisConnections,
	countersBE *countersbackend.CountersStore, activityBE *instancebackend.InstanceActivityStore) *builder.HandlerBuilder {
	ctx := context.Background()

	var dVerifier = &defaultVerifier{tc: tenantCache}
	if err := dVerifier.setProviderURL(ctx, companyConfigStorage, consoleTenantID); err != nil {
		uclog.Fatalf(ctx, "failed to set up Console as fallback JWT verifier: %v", err)
	}

	hb := builder.NewHandlerBuilder()
	initAPIHandlers(hb, dVerifier, cfg, tenantCache, kinesisBE, countersBE, activityBE)
	initServiceHandlers(hb, companyConfigStorage)
	return hb
}

// InitForTests is used by integration tests to attach test logserver services to an existing serve mux.
// Only API handlers are attached, not the service handlers which are not used by most tests.
func InitForTests(hb *builder.HandlerBuilder, companyConfigStorage *companyconfig.Storage, jwtVerifier auth.Verifier,
	cfg *config.Config, tenantCache *storage.TenantCache, kinesisBE *kinesisbackend.KinesisConnections,
	countersBE *countersbackend.CountersStore, activityBE *instancebackend.InstanceActivityStore) {
	initAPIHandlers(hb, jwtVerifier, cfg, tenantCache, kinesisBE, countersBE, activityBE)
}
