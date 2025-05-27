package userstore

// TODO: better package name to not conflict with idp/userstore?

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/config"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/security"
	"userclouds.com/internal/tenantplex/storage"
	logServerClient "userclouds.com/logserver/client"
)

type handler struct {
	logServerClient    *logServerClient.Client
	dataImportConfig   *config.DataImportConfig
	searchUpdateConfig *config.SearchUpdateConfig
	workerClient       workerclient.Client
}

// NewHandler returns a new http.Handler for the userstore service.
func NewHandler(ctx context.Context, cfg *config.Config, searchUpdateConfig *config.SearchUpdateConfig, workerClient workerclient.Client, m2mAuth jsonclient.Option, consoleTenantInfo companyconfig.TenantInfo) (http.Handler, error) {
	h := &handler{
		workerClient:       workerClient,
		searchUpdateConfig: searchUpdateConfig,
	}
	// cfg is nil in most tests and when running w/o log server, and don't need data import & worker client
	if cfg != nil {
		lsc, err := logServerClient.NewClientForTenantAuth(consoleTenantInfo.TenantURL, consoleTenantInfo.TenantID, m2mAuth, security.PassXForwardedFor())
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		h.logServerClient = lsc
		h.dataImportConfig = cfg.DataImportConfig
	}
	hb := builder.NewHandlerBuilder()
	handlerBuilder(hb, h)
	hb.MethodHandler("/download/codegensdk.go").Get(h.getCodegenGolangSDK)
	hb.MethodHandler("/download/codegensdk.py").Get(h.getCodegenPythonSDK)
	hb.MethodHandler("/download/codegensdk.ts").Get(h.getCodegenTypescriptSDK)
	hb.MethodHandler("/oidcissuers").
		Put(h.updateOIDCIssuersList).
		Get(h.getOIDCIssuersList)
	hb.CollectionHandler("/upload/dataimport").
		GetAll(h.dataImportInitialize).
		GetOne(h.dataImportStatus).
		WithAuthorizer(h.newRoleBasedAuthorizer())
	hb.CollectionHandler("/config/databases").
		GetAll(h.listDatabases).
		GetOne(h.getDatabase).
		Post(h.createDatabase).
		Put(h.updateDatabase).
		Delete(h.deleteDatabase).
		WithAuthorizer(h.newRoleBasedAuthorizer())
	hb.MethodHandler("/config/databases/test").
		Post(h.testDatabaseConnection)
	hb.MethodHandler("/config/searchindices/accessor/remove").
		Post(h.removeAccessorUserSearchIndex)
	hb.MethodHandler("/config/searchindices/accessor/set").
		Post(h.setAccessorUserSearchIndex)
	hb.CollectionHandler("/config/objectstores").
		GetAll(h.listObjectStores).
		GetOne(h.getObjectStore).
		Post(h.createObjectStore).
		Put(h.updateObjectStore).
		Delete(h.deleteObjectStore).
		WithAuthorizer(h.newRoleBasedAuthorizer())
	hb.MethodHandler("/config/regions").
		Get(h.listUserRegions)
	return hb.Build(), nil
}

//go:generate genhandler /userstore POST,executeAccessorHandler,/api/accessors POST,executeMutatorHandler,/api/mutators POST,getConsentedPurposesForUser,/api/consentedpurposes collection,UserstoreUser,h.newRoleBasedAuthorizer(),/api/users collection,DataType,h.newRoleBasedAuthorizer(),/config/datatypes collection,Column,h.newRoleBasedAuthorizer(),/config/columns collection,Accessor,h.newRoleBasedAuthorizer(),/config/accessors collection,Mutator,h.newRoleBasedAuthorizer(),/config/mutators collection,Purpose,h.newRoleBasedAuthorizer(),/config/purposes collection,UserSearchIndex,h.newRoleBasedAuthorizer(),/config/searchindices collection,SoftDeletedRetentionDuration,h.newRoleBasedAuthorizer(),/config/softdeletedretentiondurations collection,LiveRetentionDuration,h.newRoleBasedAuthorizer(),/config/liveretentiondurations nestedcollection,SoftDeletedRetentionDuration,h.newNestedRoleBasedAuthorizer(),/softdeletedretentiondurations,Purpose nestedcollection,LiveRetentionDuration,h.newNestedRoleBasedAuthorizer(),/liveretentiondurations,Purpose nestedcollection,SoftDeletedRetentionDuration,h.newNestedRoleBasedAuthorizer(),/softdeletedretentiondurations,Column nestedcollection,LiveRetentionDuration,h.newNestedRoleBasedAuthorizer(),/liveretentiondurations,Column

func (h *handler) newRoleBasedAuthorizer() uchttp.CollectionAuthorizer {
	return &uchttp.MethodAuthorizer{
		GetAllF: func(r *http.Request) error {
			return ucerr.Wrap(h.ensureTenantMember(false))
		},
		GetOneF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(false))
		},
		PostF: func(r *http.Request) error {
			return ucerr.Wrap(h.ensureTenantMember(true))
		},
		PutF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(true))
		},
		DeleteF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(true))
		},
		DeleteAllF: func(r *http.Request) error {
			return ucerr.Wrap(h.ensureTenantMember(true))
		},
		NestedF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(false))
		},
	}
}

func (h *handler) newNestedRoleBasedAuthorizer() uchttp.NestedCollectionAuthorizer {
	return &uchttp.NestedMethodAuthorizer{
		GetAllF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(false))
		},
		GetOneF: func(r *http.Request, _, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(false))
		},
		PostF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(true))
		},
		PutF: func(r *http.Request, _, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(true))
		},
		DeleteF: func(r *http.Request, _, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(true))
		},
		DeleteAllF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(true))
		},
	}
}

func (h *handler) ensureTenantMember(_ /*adminOnly*/ bool) error {
	return nil // TODO: figure out how to do this w/o calling authz service
}

func (h *handler) getOIDCIssuersList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ts := multitenant.MustGetTenantState(ctx)
	s := storage.New(ctx, ts.TenantDB, ts.CacheConfig)
	tenantPlex, err := s.GetTenantPlex(ctx, ts.ID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	jsonapi.Marshal(w, tenantPlex.PlexConfig.ExternalOIDCIssuers)
}

func (h *handler) updateOIDCIssuersList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ts := multitenant.MustGetTenantState(ctx)

	var req []string
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	s := storage.New(ctx, ts.TenantDB, ts.CacheConfig)
	tenantPlex, err := s.GetTenantPlex(ctx, ts.ID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	tenantPlex.PlexConfig.ExternalOIDCIssuers = req
	if err := s.SaveTenantPlex(ctx, tenantPlex); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, req, jsonapi.Code(http.StatusOK))
}

func (h *handler) listUserRegions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ts := multitenant.MustGetTenantState(ctx)
	regions := []region.DataRegion{}
	for k := range ts.UserRegionDbMap {
		regions = append(regions, k)
	}
	jsonapi.Marshal(w, regions)
}
