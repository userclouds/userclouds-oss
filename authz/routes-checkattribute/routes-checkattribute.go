package routes

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/authz/config"
	"userclouds.com/authz/internal"
	"userclouds.com/authz/internal/checkattributehandler"
	"userclouds.com/infra/service"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/internal/resourcecheck"
	"userclouds.com/internal/tenantmap"
)

func initAPIHandlers(
	hb *builder.HandlerBuilder,
	tenantStateMap *tenantmap.StateMap,
	tenantsHandled []uuid.UUID,
) {
	checkAttributeHandler := service.BaseMiddleware.Apply(checkattributehandler.NewHandler(tenantStateMap, tenantsHandled))
	hb.Handle("/checkattribute", checkAttributeHandler)
}

func initServiceHandlers(hb *builder.HandlerBuilder, cfg config.Config) {
	bldr := builder.NewHandlerBuilder()
	bldr = resourcecheck.AddResourceCheckEndpoint(bldr, cfg.CacheConfig, nil, nil)
	hb.Handle("/", service.BaseMiddleware.Apply(bldr.Build()))
}

// Init initializes the routes for the checkattribute service
func Init(ctx context.Context, tenantStateMap *tenantmap.StateMap, cfg config.Config, tenantsHandled []uuid.UUID) (*builder.HandlerBuilder, error) {

	// warm up the edge cache for all handled tenants before accepting any requests
	for _, tenantID := range tenantsHandled {
		ts, err := tenantStateMap.GetTenantStateForID(ctx, tenantID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		s := internal.NewStorage(ctx, ts.ID, ts.TenantDB, ts.CacheConfig)
		_, err = s.GetBFSEdgeGlobalCache(ctx)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	hb := builder.NewHandlerBuilder()
	initAPIHandlers(hb, tenantStateMap, tenantsHandled)
	initServiceHandlers(hb, cfg)
	return hb, nil
}
