package routes

import (
	"context"
	"net/http"
	"net/url"
	"os"

	"github.com/gofrs/uuid"

	"userclouds.com/console/internal"
	"userclouds.com/console/internal/api"
	"userclouds.com/console/internal/auth"
	"userclouds.com/console/internal/tenantcache"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/middleware"
	"userclouds.com/infra/service"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/ucreact"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/companyconfig"
)

func initAPIHandlers(ctx context.Context, hb *builder.HandlerBuilder, cfg *internal.Config, getConsoleURLCallback auth.GetConsoleURLCallback, storage *companyconfig.Storage, consoleTenantDB *ucdb.DB, wc workerclient.Client, consoleTenantID uuid.UUID) {
	sessions := auth.NewSessionManager(storage)

	tc := tenantcache.NewCache(storage)

	// console uses auditlog.Middleware to add the auditlog storage to the request context
	// since we don't use multitenant.Middleware here
	baseMiddleware := middleware.Chain(service.BaseMiddleware,
		auditlog.Middleware(auditlog.NewStorage(consoleTenantDB)))

	authedAPIMiddleware := middleware.Chain(
		baseMiddleware,
		rejectNonConsoleHostMiddleware(getConsoleURLCallback),
		sessions.FailIfNotLoggedIn(),
	)

	// doesn't have authed Middleware otherwise you couldn't auth :)
	authHandler := baseMiddleware.Apply(auth.NewHandler(consoleTenantID, cfg.CacheConfig, getConsoleURLCallback, storage, sessions, tc))
	hb.Handle(auth.RootPath, authHandler)

	// CompanyConfig API handler
	apiHandler := authedAPIMiddleware.Apply(api.NewHandler(cfg, getConsoleURLCallback, storage, consoleTenantDB, tc, wc, consoleTenantID))
	hb.Handle("/api/", apiHandler)

	// Static assets for Console UI (React app), but only if we're not running React dev mode
	// (otherwise we serve two copies and it's confusing)
	// TODO: I wonder if there's a cleaner way to do this, probably once we don't serve
	// static assets through golang anyway?
	// we use _PORT (even though it's not used elsewhere) to be consistent with Plex
	if os.Getenv("UC_CONSOLE_UI_DEV_PORT") == "" {
		ucreact.MountStaticAssetsHandler(ctx, hb, cfg.StaticAssetsPath, "/")
	}
}

func initServiceHandlers(hb *builder.HandlerBuilder, db *ucdb.DB) {
	// Note: like plex, console binds to / already so we can't neatly wrap these in their own serviceMux
	hb.Handle("/migrations", service.BaseMiddleware.Apply(
		http.HandlerFunc(service.CreateMigrationVersionHandler(db)),
	))

	// TODO (sgarrity 10/23): remove after testing sentry?
	hb.Handle("/testerror", service.BaseMiddleware.Apply(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			jsonapi.MarshalError(ctx, w, ucerr.New("test error"))
		}),
	))
}

// Init initializes Console routes and returns an http handler.
func Init(ctx context.Context, cfg *internal.Config, db *ucdb.DB, storage *companyconfig.Storage, consoleTenantDB *ucdb.DB, wc workerclient.Client) (*builder.HandlerBuilder, error) {
	hb := builder.NewHandlerBuilder()
	consoleURL, err := url.Parse(cfg.ConsoleURL)
	if err != nil {
		// We check that the url is parseable in extraValidate, so this should never happen
		return nil, ucerr.Wrap(err)
	}
	cb := func() *url.URL { return consoleURL }
	initAPIHandlers(ctx, hb, cfg, cb, storage, consoleTenantDB, wc, cfg.ConsoleTenantID)
	initServiceHandlers(hb, db)

	return hb, nil
}

// InitForTests is used by integration tests to set up Console handlers.
func InitForTests(ctx context.Context,
	hb *builder.HandlerBuilder,
	cfg *internal.Config,
	getConsoleURLCallback auth.GetConsoleURLCallback,
	storage *companyconfig.Storage,
	consoleTenantDB *ucdb.DB,
	wc workerclient.Client,
	consoleTenantID uuid.UUID) {

	initAPIHandlers(ctx, hb, cfg, getConsoleURLCallback, storage, consoleTenantDB, wc, consoleTenantID)
}
