package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/console/internal"
	"userclouds.com/console/internal/auth"
	"userclouds.com/console/internal/tenantcache"
	"userclouds.com/idp"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/security"
	"userclouds.com/internal/ucimage"
	logServerClient "userclouds.com/logserver/client"
	"userclouds.com/userevent"
)

type provisionDBInfo struct {
	companyDB ucdb.Config
	logDB     ucdb.Config
}
type handler struct {
	provisionDBInfo        provisionDBInfo
	cacheConfig            *cache.Config
	tenantsSubDomain       string
	tenantsProtocol        string
	consoleTenant          companyconfig.TenantInfo
	getConsoleURLCallback  auth.GetConsoleURLCallback
	storage                *companyconfig.Storage
	consoleTenantDB        *ucdb.DB
	tenantCache            *tenantcache.Cache
	consoleRBACClient      *authz.RBACClient
	consoleIDPClient       *idp.ManagementClient
	consoleUserEventClient *userevent.Client
	consoleLogServerClient *logServerClient.Client
	consoleAuditLogClient  *auditlog.Client
	consoleImageClient     *ucimage.Client
	workerClient           workerclient.Client
	onPremSQLShimHost      string
	onPremSQLShimPorts     []int
}

func (h *handler) createClients(ctx context.Context, imgCfg *ucimage.Config, cacheConfig *cache.Config, cti companyconfig.TenantInfo, s *companyconfig.Storage, wc workerclient.Client) error {
	h.workerClient = wc

	tokenSource, err := m2m.GetM2MTokenSource(ctx, cti.TenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	consoleAuthZClient, err := apiclient.NewAuthzClientWithTokenSource(ctx, s,
		cti.TenantID, cti.TenantURL, tokenSource, apiclient.ClientCacheConfig(cacheConfig))
	if err != nil {
		return ucerr.Wrap(err)
	}

	h.consoleRBACClient = authz.NewRBACClient(consoleAuthZClient)

	h.consoleIDPClient, err = idp.NewManagementClient(cti.TenantURL, tokenSource, security.PassXForwardedFor())
	if err != nil {
		return ucerr.Wrap(err)
	}

	h.consoleUserEventClient, err = userevent.NewClient(cti.TenantURL, tokenSource, security.PassXForwardedFor())
	if err != nil {
		return ucerr.Wrap(err)
	}

	h.consoleLogServerClient, err = logServerClient.NewClient(cti.TenantURL, cti.TenantID, tokenSource, security.PassXForwardedFor())
	if err != nil {
		return ucerr.Wrap(err)
	}

	h.consoleAuditLogClient, err = auditlog.NewClient(cti.TenantURL, tokenSource)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if imgCfg != nil {
		h.consoleImageClient, err = ucimage.NewClient(*imgCfg)
		if err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}

// NewHandler returns a new console API handler
// TODO: we probably shouldn't leave ucdb.Config lying around here, can go away
// once provisioning is refactored to its own place
func NewHandler(cfg *internal.Config, getConsoleURLCallback auth.GetConsoleURLCallback, storage *companyconfig.Storage, consoleTenantDB *ucdb.DB, tc *tenantcache.Cache, wc workerclient.Client, consoleTenantID uuid.UUID) http.Handler {
	ctx := context.Background()
	cti, err := storage.GetTenantInfo(ctx, consoleTenantID)
	if err != nil {
		uclog.Fatalf(ctx, "error getting console tenant info: %v", err)
	}
	h := &handler{
		consoleTenant:         *cti,
		provisionDBInfo:       provisionDBInfo{companyDB: cfg.CompanyDB, logDB: cfg.LogDB},
		tenantsSubDomain:      cfg.TenantSubDomain,
		tenantsProtocol:       cfg.TenantProtocol,
		cacheConfig:           cfg.CacheConfig,
		getConsoleURLCallback: getConsoleURLCallback,
		storage:               storage,
		consoleTenantDB:       consoleTenantDB,
		tenantCache:           tc,
	}

	if universe.Current().IsOnPrem() {
		h.onPremSQLShimHost = fmt.Sprintf("mysql-proxy.%s", strings.TrimPrefix(cfg.TenantSubDomain, "tenant."))
		h.onPremSQLShimPorts = cfg.OnPremSQLShimPorts
	}

	if err := h.createClients(ctx, cfg.Image, cfg.CacheConfig, *cti, storage, wc); err != nil {
		uclog.Fatalf(ctx, "error setting up console clients: %v", err)
	}

	topLevel := builder.NewHandlerBuilder()

	// /allcompanies
	// TODO: there is an auth check in this method but we should probably still reconsider its existence.
	topLevel.MethodHandler("/allcompanies").Get(h.listAllCompanies)

	// /serviceinfo
	topLevel.HandleFunc("/serviceinfo", h.serviceInfoHandler)

	// /companies
	companies := topLevel.CollectionHandler("/companies").
		Delete(h.deleteCompany).
		GetAll(h.listCompanies).
		Post(h.createCompany).
		Put(h.updateCompany).
		WithAuthorizer(h.newCompanyAuthorizer())

	// /companies/<uuid>/actions/*
	companies.NestedMethodHandler("/actions/inviteuser").Post(h.inviteUserToCompany)
	companies.NestedMethodHandler("/actions/listinvites").Get(h.listInvites)

	// /companies/<uuid>/employeeroles/*
	companies.NestedCollectionHandler("/employeeroles").
		GetAll(h.listCompanyRoleForEmployees).
		Delete(h.deleteEmployeeFromCompany).
		Put(h.updateCompanyRoleForEmployee).
		WithAuthorizer(h.newCompanyRolesAuthorizer())

	// /tenants
	companies.NestedCollectionHandler("/tenants").
		Delete(h.deleteTenant).
		GetAll(h.listTenants).
		GetOne(h.getTenant).
		Post(h.createTenant).
		Put(h.updateTenant).
		WithAuthorizer(h.newTenantNestedAuthorizer())

	tenants := companies.CollectionHandler("/tenants").
		WithAuthorizer(h.newTenantAuthorizer())

	// /tenants/<tenant uuid>/employeeroles/<user uuid>
	tenants.NestedCollectionHandler("/employeeroles").
		GetOne(h.getTenantRolesForEmployee).
		GetAll(h.listTenantRolesForEmployees).
		Put(h.updateTenantRolesForEmployee).
		WithAuthorizer(h.newTenantRolesAuthorizer())

	// /tenants/<uuid>/apppageparameters
	tenants.NestedCollectionHandler("/apppageparameters").
		GetOne(h.getAppPageParameters).
		Put(h.saveAppPageParameters).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	// /tenants/<uuid>/loginapps
	tenants.NestedCollectionHandler("/loginapps").
		GetOne(h.getLoginApp).
		Post(h.addLoginApp).
		Delete(h.deleteLoginApp).
		Put(h.updateLoginApp).
		GetAll(h.listLoginApps).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	// TODO: need to support /actions URLs off of double-nested collections
	tenants.NestedMethodHandler("/loginapps/actions/samlidp").
		Post(h.enableSAMLIDP)

	// /tenants/<uuid>/auditlog/entries
	tenants.NestedCollectionHandler("/auditlog/entries").
		GetAll(h.listAuditLogEntries).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	// /tenants/<uuid>/auditlog/entries
	tenants.NestedCollectionHandler("/dataaccesslog/entries").
		GetAll(h.listDataAccessLogEntries).
		GetOne(h.getDataAccessLogEntry).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	// /tenants/<uuid>/runs
	tenants.NestedCollectionHandler("/runs").
		GetAll(h.listRuns).
		GetOne(h.getRun).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	// /tenants/<uuid>/authz/*

	// TODO: Technically this should be nested under the Objects accessor, like "/authz/objects/<uuid>/edges",
	// but that requires adding support for a nested handler to `NestedCollectionHandler` as well as some
	// kind of `DoubleNestedMethodHandler`. For now we'll just go with this and require a query param to filter on object.
	tenants.NestedCollectionHandler("/authz/edges").
		GetAll(h.listEdges).
		GetOne(h.getEdge).
		Post(h.createEdge).
		Delete(h.deleteEdge).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	tenants.NestedMethodHandler("/authz/checkattribute").Get(h.checkAttribute)

	// TODO: with a bit of re-working, we could issue a limited-purpose token that the client SPA
	// could use to talk directly to the AuthZ service with, which would remove the need to proxy these APIs
	// through the orgconfig/console service.
	tenants.NestedCollectionHandler("/authz/edgetypes").
		GetAll(h.listEdgeTypes).
		GetOne(h.getEdgeType).
		Delete(h.deleteEdgeType).
		Post(h.createEdgeType).
		Put(h.updateEdgeType).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	tenants.NestedCollectionHandler("/authz/objects").
		GetAll(h.listObjects).
		GetOne(h.getObject).
		Post(h.createObject).
		Delete(h.deleteObject).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	tenants.NestedCollectionHandler("/authz/objecttypes").
		GetAll(h.listObjectTypes).
		GetOne(h.getObjectType).
		Post(h.createObjectType).
		Delete(h.deleteObjectType).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	// /tenants/<uuid>/counters/*

	// Proxy calls to logserver for Console UI. Posts are not proxied since UI doesn't use it - only reads
	tenants.NestedMethodHandler("/counters/activity").Get(h.listCounterRecords)

	tenants.NestedMethodHandler("/counters/charts").Post(h.getCharts)

	tenants.NestedMethodHandler("/counters/query").Post(h.getCounts)

	tenants.NestedMethodHandler("/counters/sources").Get(h.listCounterSources)

	// /tenants/<uuid>/emailelements
	tenants.NestedMethodHandler("/emailelements").
		Get(h.getTenantAppEmailElements).
		Post(h.saveTenantAppEmailElements)

	// /tenants/<uuid>/smselements
	tenants.NestedMethodHandler("/smselements").
		Get(h.getTenantAppSMSElements).
		Post(h.saveTenantAppSMSElements)

	// /tenants/<uuid>/keys/*

	// TODO: not totally happy with /keys and /keys/private but
	// a collection of keys with UUIDs doesn't really make sense either
	tenants.NestedMethodHandler("/keys").Get(h.listTenantPublicKeys)

	tenants.NestedMethodHandler("/keys/actions/rotate").Put(h.rotateKeys)

	tenants.NestedMethodHandler("/keys/private").Get(h.getTenantPrivateKey)

	// /tenants/<uuid>/oidcproviders/*

	tenants.NestedMethodHandler("/oidcproviders/create").
		Post(h.createOIDCProviderHandler)

	tenants.NestedMethodHandler("/oidcproviders/delete").
		Post(h.deleteOIDCProviderHandler)

	// /tenants/<uuid>/plexconfig
	tenants.NestedMethodHandler("/plexconfig").
		Get(h.getTenantPlexConfig).
		Post(h.saveTenantPlexConfig)

	tenants.NestedMethodHandler("/plexconfig/providers/actions/import").
		Post(h.importProviderAppHandler)

	// /tenants/<uuid>/policies/permissions/*
	tenants.NestedCollectionHandler("/policies/permissions").
		GetAll(h.listGlobalPolicyPermissions).
		GetOne(h.listPolicyPermissions).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

		// /tenants/<uuid>/policies/*
	tenants.NestedCollectionHandler("/policies/templates").
		Delete(h.deleteAccessPolicyTemplate).
		GetAll(h.listAccessPolicyTemplates).
		GetOne(h.getAccessPolicyTemplate).
		Post(h.createAccessPolicyTemplate).
		Put(h.updateAccessPolicyTemplate).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	tenants.NestedMethodHandler("/policies/templates/actions/test").Post(h.testRunAccessPolicyTemplate)

	// /tenants/<uuid>/policies/*
	tenants.NestedCollectionHandler("/policies/access").
		Delete(h.deleteAccessPolicy).
		GetAll(h.listAccessPolicies).
		GetOne(h.getAccessPolicy).
		Post(h.createAccessPolicy).
		Put(h.updateAccessPolicy).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	tenants.NestedMethodHandler("/policies/access/actions/test").Post(h.testRunAccessPolicy)

	tenants.NestedCollectionHandler("/policies/transformation").
		Delete(h.deleteTransformer).
		GetAll(h.listTransformers).
		GetOne(h.getTransformer).
		Post(h.createTransformer).
		Put(h.updateTransformer).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	tenants.NestedMethodHandler("/policies/transformation/actions/test").Post(h.testRunTransformer)

	tenants.NestedCollectionHandler("/policies/secrets").
		Delete(h.deleteSecret).
		GetAll(h.listSecrets).
		Post(h.createSecret).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	// /tenants/<uuid>/uploadlogo
	tenants.NestedMethodHandler("/uploadlogo").Post(h.uploadLogo)

	// /tenants/<uuid>/urls
	tenants.NestedCollectionHandler("/urls").
		GetAll(h.listTenantURLs).
		Post(h.createTenantURL).
		Put(h.updateTenantURL).
		Delete(h.deleteTenantURL).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	tenants.NestedMethodHandler("/urls/actions/validate").
		Post(h.validateTenantURL)

	// /tenants/<uuid>/users/*
	tenants.NestedCollectionHandler("/users").
		Delete(h.deleteUser).
		GetAll(h.listTenantUsers).
		GetOne(h.getTenantUser).
		Put(h.updateUser).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	// tenants/<uuid>/userevents/*
	tenants.NestedCollectionHandler("/userevents").
		GetAll(h.listTenantUserEvents).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	// tenants/<uuid>/consentedpurposesforuser/*
	tenants.NestedCollectionHandler("/consentedpurposesforuser").
		GetOne(h.getTenantUserConsentedPurposes).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	tenants.NestedCollectionHandler("/userstore/databases").
		GetAll(h.listTenantUserStoreDatabases).
		GetOne(h.getTenantUserStoreDatabase).
		Post(h.saveTenantUserStoreDatabase).
		Delete(h.deleteTenantUserStoreDatabase).
		Put(h.updateTenantUserStoreDatabase).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	tenants.NestedMethodHandler("/userstore/update_database_proxy_ports").
		Post(h.updateTenantUserStoreDatabaseProxyPorts)

	tenants.NestedMethodHandler("/userstore/test_database").
		Post(h.testTenantUserStoreDatabase)

	tenants.NestedCollectionHandler("/userstore/objectstores").
		GetAll(h.listTenantUserStoreObjectStores).
		GetOne(h.getTenantUserStoreObjectStore).
		Post(h.saveTenantUserStoreObjectStore).
		Delete(h.deleteTenantUserStoreObjectStore).
		Put(h.updateTenantUserStoreObjectStore).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	tenants.NestedCollectionHandler("/userstore/datatypes").
		GetAll(h.listTenantUserStoreDataTypes).
		GetOne(h.getTenantUserStoreDataType).
		Post(h.saveTenantUserStoreDataType).
		Delete(h.deleteTenantUserStoreDataType).
		Put(h.updateTenantUserStoreDataType).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	tenants.NestedCollectionHandler("/userstore/columns").
		GetAll(h.listTenantUserStoreColumns).
		GetOne(h.getTenantUserStoreColumn).
		Post(h.saveTenantUserStoreColumn).
		Delete(h.deleteTenantUserStoreColumn).
		Put(h.updateTenantUserStoreColumn).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	tenants.NestedMethodHandler("/userstore/columns/retentiondurations/actions/get").Post(h.getUserStoreColumnRetentionDurations)
	tenants.NestedMethodHandler("/userstore/columns/retentiondurations/actions/update").Post(h.updateUserStoreColumnRetentionDurations)

	// tenants/<uuid>/userstore/accessors
	tenants.NestedCollectionHandler("/userstore/accessors").
		GetAll(h.listTenantAccessors).
		GetOne(h.getTenantAccessor).
		Post(h.createTenantAccessor).
		Put(h.updateTenantAccessor).
		Delete(h.deleteTenantAccessor).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	tenants.NestedMethodHandler("/userstore/executeaccessor").Post(h.executeTenantAccessor)

	// tenants/<uuid>/userstore/mutators
	tenants.NestedCollectionHandler("/userstore/mutators").
		GetAll(h.listTenantMutators).
		GetOne(h.getTenantMutator).
		Post(h.createTenantMutator).
		Put(h.updateTenantMutator).
		Delete(h.deleteTenantMutator).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	// tenants/<uuid>/userstore/purposes
	tenants.NestedCollectionHandler("/userstore/purposes").
		GetAll(h.listTenantPurposes).
		GetOne(h.getTenantPurpose).
		Post(h.createTenantPurpose).
		Put(h.updateTenantPurpose).
		Delete(h.deleteTenantPurpose).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	// tenants/<uuid>/datamapping/datasources
	tenants.NestedCollectionHandler("/datamapping/datasources").
		GetAll(h.listDataSources).
		GetOne(h.getDataSource).
		Post(h.createDataSource).
		Put(h.updateDataSource).
		Delete(h.deleteDataSource).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	// tenants/<uuid>/datamapping/elements
	tenants.NestedCollectionHandler("/datamapping/elements").
		GetAll(h.listDataSourceElements).
		GetOne(h.getDataSourceElement).
		Post(h.createDataSourceElement).
		Put(h.updateDataSourceElement).
		Delete(h.deleteDataSourceElement).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	tenants.NestedMethodHandler("/userstore/codegensdk.go").Get(h.getCodegenGolangSDK)
	tenants.NestedMethodHandler("/userstore/codegensdk.py").Get(h.getCodegenPythonSDK)
	tenants.NestedMethodHandler("/userstore/codegensdk.ts").Get(h.getCodegenTypescriptSDK)

	// tenants/<uuid>/organizations/*
	organizations := tenants.NestedCollectionHandler("/organizations").
		GetOne(h.getOrganization).
		GetAll(h.listOrganizations).
		Post(h.createOrganization).
		Put(h.updateOrganization).
		WithAuthorizer(uchttp.NewNestedAllowAllAuthorizer())

	return organizations.Build()
}

type serviceInfoHandler struct {
	Environment    string      `json:"environment"`
	IsProduction   bool        `json:"is_production"`
	UCAdmin        bool        `json:"uc_admin"`
	AdminCompanies []uuid.UUID `json:"company_admin"`
}

func (h *handler) serviceInfoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	adminCompanies, err := h.getAdminCompaniesForUser(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	uv := universe.Current()
	jsonapi.Marshal(w, serviceInfoHandler{
		Environment:    fmt.Sprintf("%s %s", uv, region.Current()),
		IsProduction:   uv.IsProd(),
		UCAdmin:        (h.ensureUCAdmin(r) == nil),
		AdminCompanies: adminCompanies,
	})
}
