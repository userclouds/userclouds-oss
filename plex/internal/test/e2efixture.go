package test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	authzroutes "userclouds.com/authz/routes"
	"userclouds.com/idp"
	idproutes "userclouds.com/idp/routes"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/oidcproviders"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/tenantplex"
	plexconfigtest "userclouds.com/internal/tenantplex/test"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
	"userclouds.com/plex"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/token"
	"userclouds.com/plex/manager"
	plexroutes "userclouds.com/plex/routes"
)

// E2EFixture is a test fixture for end-to-end tests.
type E2EFixture struct {
	TenantURL            string
	TenantDB             *ucdb.DB
	TenantID             uuid.UUID
	CompanyConfigStorage *companyconfig.Storage
	PlexStorage          *storage.Storage
	PlexClientID         string
	PlexSecret           string
	IdpClient            *idp.ManagementClient
	AuthzClient          *authz.Client
	Email                *uctest.EmailClient
	CompanyID            uuid.UUID
	RedirectURI          string
}

const testUCApp = "testucapp"
const testPlexApp = "testplexapp"

func newTenantConfig(t *testing.T, plexClientID, plexSecret, redirectURI, tenantURL string, appID uuid.UUID) (tc tenantplex.TenantConfig) {
	tc = plexconfigtest.NewTenantConfigBuilder().
		AddProvider().SetName("testprov").MakeActive().MakeUC().SetIDPURL(tenantURL).AddUCAppWithName(testUCApp).
		AddApp().SetID(appID).SetName(testPlexApp).SetClientID(plexClientID).SetClientSecret(secret.NewTestString(plexSecret)).AddAllowedRedirectURI(redirectURI).Build()
	assert.IsNil(t, tc.Validate(), assert.Must())
	return
}

const testUsername = "userfoo"
const testPassword = "p@ssword"

// GenUsername generates a unique username for this test to avoid conflict
func GenUsername() string {
	// Generate unique username for this test to avoid conflict
	return testUsername + uuid.Must(uuid.NewV4()).String()
}

// GenPassword generates a unique password for this test to avoid conflict
func GenPassword() string {
	// Generate unique password for this test to avoid conflict
	return testPassword + uuid.Must(uuid.NewV4()).String()
}

// DoLogin performs a login request to the test server.
func (tf *E2EFixture) DoLogin(redirectURL, state, username, password string) (*plex.LoginResponse, error) {
	ctx := context.Background()
	plexClient := jsonclient.New(tf.TenantURL)
	sessionID, err := storage.CreateOIDCLoginSession(
		ctx, tf.PlexStorage, tf.PlexClientID, []storage.ResponseType{storage.AuthorizationCodeResponseType},
		uctest.MustParseURL(redirectURL), state, "unusedscope")
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	lr := &plex.LoginRequest{
		Username:  username,
		Password:  password,
		SessionID: sessionID,
	}

	var loginResponse plex.LoginResponse
	err = plexClient.Post(ctx, "/login", lr, &loginResponse)
	return &loginResponse, ucerr.Wrap(err)
}

// NewE2EFixture creates a new E2E test fixture.
func NewE2EFixture(t *testing.T) *E2EFixture {
	ctx := context.Background()
	email := &uctest.EmailClient{}

	// Provision base UC company & Console tenant, then provision a test tenant.
	// TODO: is there an easier way to shortcut this? I think the IDP multitenancy
	// logic won't work without this since it directly reads from the DB, though
	// we could refactor it to be like Plex and have an extra layer of indirection.
	companyConfigDBCfg, logDBCfg, companyConfigStorage := testhelpers.NewTestStorage(t)
	_, consoleTenant, consoleTenantDB := testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, companyConfigStorage, companyConfigDBCfg, logDBCfg)
	company := testhelpers.NewCompanyForTest(ctx, t, companyConfigStorage, consoleTenantDB, consoleTenant.ID, consoleTenant.CompanyID)
	tenant, tenantDB := testhelpers.ProvisionTestTenant(ctx, t, companyConfigStorage, companyConfigDBCfg, logDBCfg, company.ID)

	// Run IDP server pointing to company config DB
	tenants := testhelpers.NewTestTenantStateMap(companyConfigStorage)
	jwtVerifier := oidcproviders.NewOIDCProviderMap()
	hb := builder.NewHandlerBuilder()
	idproutes.InitForTests(hb, tenants, companyConfigStorage, consoleTenant.ID, jwtVerifier)
	m2mAuth, err := m2m.GetM2MTokenSource(ctx, consoleTenant.ID)
	assert.NoErr(t, err)
	// Because this is an e2e test, we use the prod factory to talk to an actual UC IDP, not a mock IDP.
	plexroutes.InitForTests(ctx, m2mAuth, hb, companyConfigStorage, jwtVerifier, &testhelpers.SecurityValidator{}, email, provider.ProdFactory{}, nil, consoleTenant.ID)
	authzroutes.InitForTests(hb, tenants, companyConfigStorage, jwtVerifier)
	tenantServer := httptest.NewServer(hb.Build())
	t.Cleanup(tenantServer.Close)

	// Update tenant URL for IDP so we don't have to do Host header override shenanigans
	tenant.TenantURL = testhelpers.UpdateTenantURLForTestTenant(t, tenant.TenantURL, tenantServer.URL)
	assert.NoErr(t, companyConfigStorage.SaveTenant(ctx, tenant))

	plexClientID := crypto.GenerateClientID()
	plexSecret := crypto.MustRandomBase64(32) // we don't ever use the secret.String version here
	redirectURI := "https://example.com/redirect"
	cacheCfg := cachetesthelpers.NewCacheConfig()
	mgr := manager.NewFromDB(tenantDB, cacheCfg)
	tp, err := mgr.GetTenantPlex(ctx, tenant.ID)
	assert.NoErr(t, err)

	appID := tp.PlexConfig.PlexMap.Apps[0].ID
	tp.PlexConfig = newTenantConfig(t, plexClientID, plexSecret, redirectURI, tenant.TenantURL, appID)
	assert.NoErr(t, mgr.SaveTenantPlex(ctx, tp))

	ctx = multitenant.SetTenantState(ctx, tenantmap.NewTenantState(tenant, company, uctest.MustParseURL(tenant.TenantURL), tenantDB, nil, nil, "", companyConfigStorage, false, nil, nil))
	jwt, err := token.CreateAccessTokenJWT(ctx,
		&tp.PlexConfig,
		uuid.Must(uuid.NewV4()),
		"",
		"",
		"",
		tenant.TenantURL,
		[]string{tenant.TenantURL},
		ucjwt.DefaultValidityAccess,
	)
	assert.NoErr(t, err)
	idpClient, err := idp.NewManagementClient(tenant.TenantURL, jsonclient.HeaderAuthBearer(jwt))
	assert.NoErr(t, err)
	authzClient, err := authz.NewClient(tenant.TenantURL, authz.JSONClient(jsonclient.HeaderAuthBearer(jwt)))
	assert.NoErr(t, err)

	return &E2EFixture{
		TenantURL:            tenant.TenantURL,
		TenantDB:             tenantDB,
		TenantID:             tenant.ID,
		CompanyConfigStorage: companyConfigStorage,
		PlexStorage:          storage.New(ctx, tenantDB, nil),
		PlexClientID:         plexClientID,
		PlexSecret:           plexSecret,
		RedirectURI:          redirectURI,
		IdpClient:            idpClient,
		AuthzClient:          authzClient,
		Email:                email,
		CompanyID:            company.ID,
	}
}
