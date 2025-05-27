package testhelpers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"

	"userclouds.com/authz"
	authzroutes "userclouds.com/authz/routes"
	"userclouds.com/console/internal"
	"userclouds.com/console/internal/auth"
	"userclouds.com/console/routes"
	"userclouds.com/idp/idptesthelpers"
	idproutes "userclouds.com/idp/routes"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/oidcproviders"
	"userclouds.com/internal/provisioning"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/testkeys"
	"userclouds.com/internal/ucimage"
	"userclouds.com/internal/uctest"
	"userclouds.com/plex/manager"
	plextest "userclouds.com/plex/test"
)

// TestFixture is a test fixture for console tests
type TestFixture struct {
	CompanyConfigDBCfg     *ucdb.Config
	LogDBCfg               *ucdb.Config
	CompanyConfigStorage   *companyconfig.Storage
	ConsoleServerURL       string
	TenantServer           *httptest.Server
	TenantServerURL        string
	Email                  *uctest.EmailClient
	ucCompany              *companyconfig.Company
	UserCloudsCompanyID    uuid.UUID
	ConsoleTenantCompanyID uuid.UUID
	ConsoleTenantID        uuid.UUID
	ConsoleTenantDB        *ucdb.DB
	ClientID               string // this exists solely for user creation
	Sessions               *auth.SessionManager
	RBACClient             *authz.RBACClient
	t                      *testing.T
}

// These are used only for tenant creation
const tenantSubDomain = "not used"
const tenantProtocol = "not used"

func addRedirectURI(ctx context.Context, t *testing.T, companyConfigStorage *companyconfig.Storage, tenantID uuid.UUID, redirectURI string) {
	cacheCfg := cachetesthelpers.NewCacheConfig()
	mgr, err := manager.NewFromCompanyConfig(ctx, companyConfigStorage, tenantID, cacheCfg)
	assert.NoErr(t, err)
	tp, err := mgr.GetTenantPlex(ctx, tenantID)
	assert.NoErr(t, err)

	tp.PlexConfig.PlexMap.Apps[0].AllowedRedirectURIs = append(tp.PlexConfig.PlexMap.Apps[0].AllowedRedirectURIs, redirectURI)
	err = mgr.SaveTenantPlex(ctx, tp)
	assert.NoErr(t, err)
}

// TODO (sgarrity 10/23): this is a very trivial use of an option pattern, but we only need it (so far)
// in one place (of ~15 calls), and for one specific purpose. If you find this in the future and need
// to add more, you can easily implement a "normal" option pattern with an interface, an Apply(), etc
// and extend it with minimal disruption.

// ProvisionUniqueDatabase is an option for creating a unique database for each test
type ProvisionUniqueDatabase struct{}

// NewTestFixture creates a new test fixture
func NewTestFixture(t *testing.T, opts ...ProvisionUniqueDatabase) *TestFixture {
	return NewTestFixtureWithWorkerClient(t, workerclient.NewTestClient(), opts...)
}

// NewTestFixtureWithWorkerClient creates a new test fixture with the provided worker client
func NewTestFixtureWithWorkerClient(t *testing.T, wc workerclient.Client, opts ...ProvisionUniqueDatabase) *TestFixture {
	ctx := context.Background()

	// TODO: there is a lot of code here potentially shareable with plex/internal/e2e_test

	// Set up UC company & console tenant
	var companyConfigDBCfg, logDBCfg *ucdb.Config
	var companyConfigStorage *companyconfig.Storage
	if len(opts) != 0 {
		companyConfigDBCfg, logDBCfg, companyConfigStorage = testhelpers.NewUniqueTestStorage(t)
	} else {
		companyConfigDBCfg, logDBCfg, companyConfigStorage = testhelpers.NewTestStorage(t)
	}
	ucCompany, consoleTenant, consoleTenantDB := testhelpers.ProvisionConsoleCompanyAndTenant(ctx, t, companyConfigStorage, companyConfigDBCfg, logDBCfg)
	// Set up AuthZ, IDP, and Plex handlers on a single Tenant Server
	tenants := testhelpers.NewTestTenantStateMap(companyConfigStorage)
	oidcProviderMap := oidcproviders.NewOIDCProviderMap()
	tenantHB := builder.NewHandlerBuilder()
	authzroutes.InitForTests(tenantHB, tenants, companyConfigStorage, oidcProviderMap)
	idproutes.InitForTests(tenantHB, tenants, companyConfigStorage, consoleTenant.ID, oidcProviderMap)
	email := &uctest.EmailClient{}
	plextest.InitForExternalTests(ctx, t, tenantHB, companyConfigStorage, oidcProviderMap, email, consoleTenant.ID)

	// need to grab plex client ID for user creation later
	cacheCfg := cachetesthelpers.NewCacheConfig()
	mgr := manager.NewFromDB(consoleTenantDB, cacheCfg)
	tp, err := mgr.GetTenantPlex(ctx, consoleTenant.ID)
	assert.NoErr(t, err)
	clientID := tp.PlexConfig.PlexMap.Apps[0].ClientID

	// Create tenant server with uber handler
	tenantServer := httptest.NewServer(tenantHB.Build())
	t.Cleanup(tenantServer.Close)
	tenantServerURL := testhelpers.UpdateTenantURLForTestTenant(t, consoleTenant.TenantURL, tenantServer.URL)
	testhelpers.FixupTenantURL(t, companyConfigStorage, consoleTenant, tenantServerURL, consoleTenantDB)

	tokenSource, err := m2m.GetM2MTokenSource(ctx, consoleTenant.ID)
	assert.NoErr(t, err)
	authZClient, err := authz.NewClient(consoleTenant.TenantURL, authz.JSONClient(tokenSource))
	assert.NoErr(t, err)
	rbacClient := authz.NewRBACClient(authZClient)

	sessions := auth.NewSessionManager(companyConfigStorage)

	cfg := &internal.Config{
		ConsoleTenantID: consoleTenant.ID,
		TenantSubDomain: tenantSubDomain,
		TenantProtocol:  tenantProtocol,
		CompanyDB:       *companyConfigDBCfg,
		LogDB:           *logDBCfg,
		CacheConfig:     cacheCfg,
		Image:           &ucimage.Config{Host: "test_cloudfront_domain.cloudfront.net", S3Bucket: "test_s3_bucket"},
		WorkerClient:    workerclient.Config{Type: workerclient.TypeTest},
	}
	tf := &TestFixture{
		t:                      t,
		CompanyConfigDBCfg:     companyConfigDBCfg,
		LogDBCfg:               logDBCfg,
		CompanyConfigStorage:   companyConfigStorage,
		TenantServer:           tenantServer,
		TenantServerURL:        tenantServerURL,
		Email:                  email,
		UserCloudsCompanyID:    ucCompany.ID,
		ucCompany:              ucCompany,
		ConsoleTenantCompanyID: consoleTenant.CompanyID,
		ConsoleTenantID:        consoleTenant.ID,
		ConsoleTenantDB:        consoleTenantDB,
		ClientID:               clientID,
		Sessions:               sessions,
		RBACClient:             rbacClient,
	}

	// Set up Console API handler/server
	consoleHB := builder.NewHandlerBuilder()
	routes.InitForTests(ctx, consoleHB, cfg, tf.getConsoleURLCallback, companyConfigStorage, consoleTenantDB, wc, consoleTenant.ID)
	consoleServer := httptest.NewServer(consoleHB.Build())
	t.Cleanup(consoleServer.Close)
	tf.ConsoleServerURL = testhelpers.UpdateTenantURLForTestTenant(t, "http://console.test.userclouds.tools", consoleServer.URL)

	// Ensure the Console's auth callback & invite callback URLs are registered with Plex
	addRedirectURI(ctx, t, companyConfigStorage, consoleTenant.ID, tf.ConsoleServerURL+auth.AuthCallbackPath)
	addRedirectURI(ctx, t, companyConfigStorage, consoleTenant.ID, tf.ConsoleServerURL+auth.InviteCallbackPath)

	// At this point, the Console tenant's Plex instance should be up & running, so we can set up Console as a "fallback" JWT verifier for all tenants.
	assert.NoErr(t, oidcProviderMap.SetFallbackProviderToTenant(ctx, companyConfigStorage, consoleTenant.ID))
	return tf
}

func (tf *TestFixture) getConsoleURLCallback() *url.URL {
	return uctest.MustParseURL(tf.ConsoleServerURL)
}

// MakeUCAdmin creates a super admin user for the UC company and returns their user ID and auth session cookie
func (tf *TestFixture) MakeUCAdmin(ctx context.Context) (uuid.UUID, *http.Cookie, string) {
	// Create a "super admin" user (i.e. make them an admin of the UC company) & get their auth session cookie
	// so we can make API calls on their behalf
	ucCompanyOwnerUserID := uuid.Must(uuid.NewV4())
	_, email := idptesthelpers.CreateUser(tf.t, tf.ConsoleTenantDB, ucCompanyOwnerUserID, tf.UserCloudsCompanyID, uuid.Nil, tf.TenantServerURL)
	testhelpers.ProvisionTestCompany(ctx, tf.t, tf.CompanyConfigStorage, tf.ucCompany, tf.ConsoleTenantDB, tf.ConsoleTenantID, tf.ConsoleTenantCompanyID, provisioning.Owner(ucCompanyOwnerUserID))

	// Get the auth session cookie for the super admin
	cookie, err := tf.GetUserCookie(ucCompanyOwnerUserID, email)
	assert.NoErr(tf.t, err)
	return ucCompanyOwnerUserID, cookie, email
}

// GetUserCookie returns a cookie for the given user ID and email
func (tf *TestFixture) GetUserCookie(userID uuid.UUID, email string) (*http.Cookie, error) {
	ctx := context.Background()

	dummyRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	claims := oidc.UCTokenClaims{
		StandardClaims: oidc.StandardClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: userID.String()}},
		Email:          email,
	}
	idT, err := ucjwt.CreateToken(ctx,
		testkeys.GetPrivateKey(tf.t),
		testkeys.Config.KeyID,
		uuid.Must(uuid.NewV4()),
		claims,
		"testissuer",
		60*60 /* an hour should be ok for testing */)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	session, err := tf.Sessions.GetAuthSession(dummyRequest)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	session.IDToken = idT
	session.AccessToken = idT
	session.RefreshToken = idT
	if err := tf.Sessions.SaveSession(ctx, w, session); err != nil {
		return nil, ucerr.Wrap(err)
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		return nil, ucerr.Errorf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Name != auth.SessionCookieName {
		return nil, ucerr.Errorf("expected cookie name to be '%s', got '%s'", auth.SessionCookieName, cookies[0].Name)
	}
	return cookies[0], nil
}
