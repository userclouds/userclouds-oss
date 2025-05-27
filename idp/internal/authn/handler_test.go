package authn_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"

	"userclouds.com/idp"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/internal/authn"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/request"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/pageparameters/pagetype"
	"userclouds.com/internal/pageparameters/parameter"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
	"userclouds.com/plex/manager"
)

const testPassword string = "p@ssword"
const testEmail string = "foo@contoso.com"

type testFixture struct {
	ctx            context.Context
	t              *testing.T
	handler        http.Handler
	companyStorage *companyconfig.Storage
	tenants        *tenantmap.StateMap
	email          *uctest.EmailClient
	tenant         *companyconfig.Tenant
	tenantDB       *ucdb.DB
	clientID       string
	bearerToken    string
}

func (tf *testFixture) enableMFA(t *testing.T) {
	ctx := context.Background()

	mgr := manager.NewFromDB(tf.tenantDB, cachetesthelpers.NewCacheConfig())
	tp, err := mgr.GetTenantPlex(ctx, tf.tenant.ID)
	assert.NoErr(t, err)
	tp.PlexConfig.SetPageParameter(pagetype.EveryPage, parameter.MFAMethods, "email")
	err = mgr.SaveTenantPlex(ctx, tp)
	assert.NoErr(t, err)
}
func (tf *testFixture) request(method, path string, reqBody any) *httptest.ResponseRecorder {
	var reader io.Reader
	if reqBody != nil {
		reader = uctest.IOReaderFromJSONStruct(tf.t, reqBody)
	} else {
		reader = nil
	}
	req := httptest.NewRequest(method, tf.tenant.TenantURL+path, reader)
	req.Header.Add("Authorization", tf.bearerToken)
	w := httptest.NewRecorder()
	tf.handler.ServeHTTP(w, req)
	return w
}

func newTestFixture(t *testing.T) testFixture {
	ctx := context.Background()
	_, tenant, _, tenantDB, handler, tm := testhelpers.CreateTestServer(ctx, t)
	_, _, companyStorage := testhelpers.NewTestStorage(t)
	ts, err := tm.GetTenantStateForID(ctx, tenant.ID)
	assert.NoErr(t, err)

	mgr := manager.NewFromDB(tenantDB, cachetesthelpers.NewCacheConfig())
	tp, err := mgr.GetTenantPlex(ctx, tenant.ID)
	assert.NoErr(t, err)
	assert.Equal(t, len(tp.PlexConfig.PlexMap.Apps), 1)

	req, err := http.NewRequest("GET", "/test", nil)
	assert.NoErr(t, err)
	jwt := uctest.CreateJWT(t, oidc.UCTokenClaims{}, tenant.TenantURL)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", jwt))

	// Stub out email dispatch so we can retrieve MFA codes.
	email := &uctest.EmailClient{}
	ctx = request.SetRequestData(multitenant.SetTenantState(ctx, ts), req, uuid.Nil)
	return testFixture{
		t:              t,
		handler:        handler,
		companyStorage: companyStorage,
		tenants:        tm,
		email:          email,
		tenant:         tenant,
		tenantDB:       tenantDB,
		clientID:       tp.PlexConfig.PlexMap.Apps[0].ClientID,
		bearerToken:    fmt.Sprintf("Bearer %s", jwt),
		ctx:            ctx,
	}
}

func genUsername() string {
	// Generate unique username for this test to avoid conflict
	return "userfoo" + uuid.Must(uuid.NewV4()).String()
}

func createUserTest(tf testFixture, createReq idp.CreateUserAndAuthnRequest) (uuid.UUID, error) {
	w := tf.request(http.MethodPost, "/authn/users", createReq)
	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		return uuid.Nil, ucerr.Wrap(jsonclient.Error{StatusCode: resp.StatusCode})
	}
	var cur idp.UserResponse
	err := json.NewDecoder(resp.Body).Decode(&cur)
	return cur.ID, ucerr.Wrap(err)
}

func deleteUserTest(tf testFixture, userID uuid.UUID) error {
	w := tf.request(http.MethodDelete, fmt.Sprintf("/authn/users/%s", userID), nil)
	resp := w.Result()
	if resp.StatusCode != http.StatusNoContent {
		return ucerr.Wrap(jsonclient.Error{StatusCode: resp.StatusCode})
	}
	return nil
}

func createUserWithPassword(tf testFixture, username, password string) (uuid.UUID, error) {
	user, err := createUserWithPasswordAndProfile(tf, username, password, userstore.Record{"email": testEmail})
	return user, ucerr.Wrap(err)
}

func createUserWithPasswordAndProfile(tf testFixture, username, password string, profile userstore.Record) (uuid.UUID, error) {
	upReq := idp.CreateUserAndAuthnRequest{
		Profile:   profile,
		UserAuthn: idp.NewPasswordAuthn(username, password),
	}
	user, err := createUserTest(tf, upReq)
	return user, ucerr.Wrap(err)
}

func createUserWithOIDC(tf testFixture, provider oidc.ProviderType, issuerURL string, subject string, profile userstore.Record) (uuid.UUID, error) {
	upReq := idp.CreateUserAndAuthnRequest{
		Profile:   profile,
		UserAuthn: idp.NewOIDCAuthn(provider, issuerURL, subject),
	}
	user, err := createUserTest(tf, upReq)
	return user, ucerr.Wrap(err)
}

func createColumn(tf testFixture, column userstore.Column) error {
	createReq := idp.CreateColumnRequest{Column: column}
	w := tf.request(http.MethodPost, "/userstore/config/columns", createReq)
	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		return ucerr.Wrap(jsonclient.Error{StatusCode: resp.StatusCode})
	}
	return nil
}

func tryLogin(tf testFixture, username, password string) *http.Response {
	upReq := idp.UsernamePasswordLoginRequest{
		Username: username,
		Password: password,
		ClientID: tf.clientID,
	}
	w := tf.request(http.MethodPost, "/authn/uplogin", upReq)
	return w.Result()
}

func validateLoginResponse(t *testing.T, resp *http.Response) {
	t.Helper()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	var tr idp.LoginResponse
	err := json.NewDecoder(resp.Body).Decode(&tr)
	assert.NoErr(t, err)
	assert.Equal(t, tr.Status, idp.LoginStatusSuccess)
}

// Basic test to ensure we can retrieve the test tenant (every IDP handler
// method needs this to work).
// TODO: This test should probably move to multitenant or tenantdb?
// Originally this code was only used for IDP but then we created those pkgs so
// this logic could be shared with AuthZ.
func TestMultitenant(t *testing.T) {
	t.Parallel()
	tf := newTestFixture(t)
	// Ensure we can retrieve the tenant we just created.
	host, err := tenantmap.GetHostFromTenantURL(tf.tenant.TenantURL)
	assert.NoErr(t, err)
	tenant, err := tf.tenants.GetTenantStateForHostname(tf.ctx, host)
	assert.NoErr(t, err)
	assert.NotNil(t, tenant)
}

// Test to ensure secret resolution works when connecting to IDP/AuthZ database
// TODO: This test should probably move to multitenant or tenantdb?
// Originally this code was only used for IDP but then we created those packages so
// this logic could be shared with AuthZ.
func TestMultitenantSecretResolution(t *testing.T) {
	tf := newTestFixture(t)
	tenantInternal, err := tf.companyStorage.GetTenantInternal(tf.ctx, tf.tenant.ID)
	assert.NoErr(t, err)
	password, err := tenantInternal.TenantDBConfig.Password.Resolve(tf.ctx)
	assert.NoErr(t, err)
	sp, err := secret.NewString(tf.ctx, universe.ServiceName(), "unused-in-tests", password)
	assert.NoErr(t, err)
	tenantInternal.TenantDBConfig.Password = *sp
	assert.IsNil(t, tf.companyStorage.SaveTenantInternal(tf.ctx, tenantInternal))

	_, err = createUserWithPassword(tf, genUsername(), testPassword)
	assert.NoErr(t, err)
}

func TestUsernamePasswordLogin(t *testing.T) {
	t.Parallel()
	tf := newTestFixture(t)

	username := genUsername()

	// Try to log in with a user that doesn't exist.
	resp := tryLogin(tf, username, testPassword)
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/json")
	var oauthe ucerr.OAuthError
	err := json.NewDecoder(resp.Body).Decode(&oauthe)
	assert.NoErr(t, err)
	assert.Equal(t, oauthe.ErrorType, "invalid_grant")

	_, err = createUserWithPassword(tf, username, testPassword)
	assert.NoErr(t, err)

	// Try to log in with wrong password
	resp = tryLogin(tf, username, testPassword+"foo")
	assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
	err = json.NewDecoder(resp.Body).Decode(&oauthe)
	assert.NoErr(t, err)
	assert.Equal(t, oauthe.ErrorType, "invalid_grant")

	// Sign in with correct password
	resp = tryLogin(tf, username, testPassword)
	validateLoginResponse(t, resp)
}

func TestMFA(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tf := newTestFixture(t)
	tf.enableMFA(t)

	// Create user in system with MFA enabled.
	username := genUsername()
	userID, err := createUserWithPassword(tf, username, testPassword)
	assert.NoErr(t, err)
	host, err := tenantmap.GetHostFromTenantURL(tf.tenant.TenantURL)
	assert.NoErr(t, err)
	mgr, err := authn.GetManager(ctx, tf.tenants, host)
	assert.NoErr(t, err)
	assert.NoErr(t, mgr.CreateTestMFAEmailChannel(ctx, userID, testEmail))

	// Try to log in and ensure it succeeds with an MFA challenge.
	resp := tryLogin(tf, username, testPassword)
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	var lr idp.LoginResponse
	err = json.NewDecoder(resp.Body).Decode(&lr)
	assert.NoErr(t, err)
	assert.Equal(t, lr.Status, idp.LoginStatusMFARequired)
	assert.NotEqual(t, len(lr.MFAToken), 0)
	mfaToken, err := uuid.FromString(lr.MFAToken)
	assert.NoErr(t, err)

	// get an mfa code
	supportedMFAChannels, _, err := mgr.GetMFASettings(ctx, oidc.MFAChannelTypeSet{oidc.MFAEmailChannel: true}, userID)
	assert.NoErr(t, err)
	assert.Equal(t, len(supportedMFAChannels.Channels), 1)
	mfacr := idp.MFAChannelRequest{
		MFAToken:   mfaToken,
		MFAChannel: supportedMFAChannels.Channels[supportedMFAChannels.PrimaryChannelID],
	}
	w := tf.request(http.MethodPost, "/authn/mfacode", mfacr)
	resp = w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	var cr idp.MFACodeResponse
	err = json.NewDecoder(resp.Body).Decode(&cr)
	assert.NoErr(t, err)
	assert.Equal(t, mfacr.MFAToken, cr.MFAToken)
	assert.Equal(t, mfacr.MFAChannel.ID, cr.MFAChannel.ID)
	assert.Equal(t, mfacr.MFAChannel.ChannelType, cr.MFAChannel.ChannelType)
	assert.Equal(t, mfacr.MFAChannel.ChannelTypeID, cr.MFAChannel.ChannelTypeID)
	assert.Equal(t, mfacr.MFAChannel.Primary, cr.MFAChannel.Primary)
	assert.Equal(t, mfacr.MFAChannel.Verified, cr.MFAChannel.Verified)
	assert.Equal(t, mfacr.MFAChannel.LastVerified, cr.MFAChannel.LastVerified)
	assert.True(t, len(cr.MFACode) > 0)

	// log in with the code
	mfalr := idp.MFALoginRequest{
		MFAToken: mfaToken,
		MFACode:  cr.MFACode,
	}
	w = tf.request(http.MethodPost, "/authn/mfaresponse", mfalr)
	validateLoginResponse(t, w.Result())
}

func TestCreateUserWithPassword(t *testing.T) {
	t.Parallel()
	tf := newTestFixture(t)
	ctx := context.Background()
	username := genUsername()
	userID, err := createUserWithPassword(tf, username, "testme")
	assert.NoErr(t, err)

	// Check the credentials for the specific tenant.
	host, err := tenantmap.GetHostFromTenantURL(tf.tenant.TenantURL)
	assert.NoErr(t, err)
	mgr, err := authn.GetManager(ctx, tf.tenants, host)
	assert.NoErr(t, err)
	baseUser, err := mgr.CheckUsernamePassword(ctx, username, "testme")
	assert.NoErr(t, err)
	assert.Equal(t, userID, baseUser.ID)
	tenant, err := tf.tenants.GetTenantStateForHostname(ctx, host)
	assert.NoErr(t, err)
	tenantIDP := authn.NewAuthN(ctx, tenant)
	cm, err := storage.NewUserstoreColumnManager(ctx, tenantIDP.ConfigStorage)
	assert.NoErr(t, err)
	dtm, err := storage.NewDataTypeManager(ctx, tenantIDP.ConfigStorage)
	assert.NoErr(t, err)
	user, _, _, err := tenantIDP.UserMultiRegionStorage.GetUser(ctx, cm, dtm, userID, false)
	assert.NoErr(t, err)
	assert.Equal(t, user.Profile["email"], testEmail)

	// Can't create another user with duplicate username
	_, err = createUserWithPassword(tf, username, testPassword)
	var jsonClientErr jsonclient.Error
	assert.True(t, errors.As(err, &jsonClientErr), assert.Must())
	assert.Equal(t, jsonClientErr.StatusCode, http.StatusConflict)

	// But if we delete the user, we can
	err = deleteUserTest(tf, userID)
	assert.NoErr(t, err)
	_, err = createUserWithPassword(tf, username, testPassword)
	assert.NoErr(t, err)
}

func saveDefaultUserStoreSchema(tf testFixture) ([]userstore.Column, error) {
	columns := []userstore.Column{
		{
			ID:        uuid.Must(uuid.NewV4()),
			Table:     "users",
			Name:      "some_field",
			DataType:  datatype.String,
			IndexType: userstore.ColumnIndexTypeIndexed,
		},
		{
			ID:        uuid.Must(uuid.NewV4()),
			Table:     "users",
			Name:      "another_field",
			DataType:  datatype.Timestamp,
			IndexType: userstore.ColumnIndexTypeIndexed,
		},
	}

	for _, col := range columns {
		if err := createColumn(tf, col); err != nil {
			return []userstore.Column{}, err
		}
	}

	return columns, nil
}

func TestCreateUserWithOIDC(t *testing.T) {
	t.Parallel()
	tf := newTestFixture(t)
	ctx := context.Background()
	columns, err := saveDefaultUserStoreSchema(tf)
	assert.NoErr(t, err)

	email := fmt.Sprintf("someone%s@contoso.com", uuid.Must(uuid.NewV4()).String())
	profile := userstore.Record{
		"email":          email,
		"email_verified": false,
		"name":           "testname" + uuid.Must(uuid.NewV4()).String(),
		"nickname":       "testnickname" + uuid.Must(uuid.NewV4()).String(),
		"picture":        "testpicurl" + uuid.Must(uuid.NewV4()).String(),
		columns[0].Name:  "field value",
	}
	subject := "1234" + uuid.Must(uuid.NewV4()).String()

	userID, err := createUserWithOIDC(tf, oidc.ProviderTypeGoogle, oidc.ProviderTypeGoogle.GetDefaultIssuerURL(), subject, profile)
	assert.NoErr(t, err)

	host, err := tenantmap.GetHostFromTenantURL(tf.tenant.TenantURL)
	assert.NoErr(t, err)
	tenant, err := tf.tenants.GetTenantStateForHostname(ctx, host)
	assert.NoErr(t, err)
	tenantIDP := authn.NewAuthN(ctx, tenant)
	cm, err := storage.NewUserstoreColumnManager(ctx, tenantIDP.ConfigStorage)
	assert.NoErr(t, err)
	dtm, err := storage.NewDataTypeManager(ctx, tenantIDP.ConfigStorage)
	assert.NoErr(t, err)
	user, _, _, err := tenantIDP.UserMultiRegionStorage.GetUser(ctx, cm, dtm, userID, false)
	assert.NoErr(t, err)
	assert.Equal(t, user.Profile["email"], profile["email"])
	assert.Equal(t, user.Profile["email_verified"], profile["email_verified"])
	assert.Equal(t, user.Profile["name"], profile["name"])
	assert.Equal(t, user.Profile["nickname"], profile["nickname"])
	assert.Equal(t, user.Profile["picture"], profile["picture"])
	assert.Equal(t, user.Profile[columns[0].Name], profile[columns[0].Name])
	assert.Equal(t, user.Profile[columns[1].Name], nil)

	passwordAuthns, err := tenantIDP.ConfigStorage.ListPasswordAuthnsForUserID(ctx, user.ID)
	assert.NoErr(t, err)
	assert.Equal(t, len(passwordAuthns), 0, assert.Must())
	socialAuthns, err := tenantIDP.ConfigStorage.ListOIDCAuthnsForUserID(ctx, user.ID)
	assert.NoErr(t, err)
	assert.Equal(t, len(socialAuthns), 1, assert.Must())
	assert.Equal(t, socialAuthns[0].Type, oidc.ProviderTypeGoogle)
	assert.Equal(t, socialAuthns[0].OIDCSubject, subject)

	// Can't create another user with duplicate social type + subject
	_, err = createUserWithOIDC(tf, oidc.ProviderTypeGoogle, oidc.ProviderTypeGoogle.GetDefaultIssuerURL(), subject, profile)
	var jsonClientErr jsonclient.Error
	assert.True(t, errors.As(err, &jsonClientErr), assert.Must())
	assert.Equal(t, jsonClientErr.StatusCode, http.StatusConflict)

	// But if we delete the user, we can
	err = deleteUserTest(tf, userID)
	assert.NoErr(t, err)
	_, err = createUserWithOIDC(tf, oidc.ProviderTypeGoogle, oidc.ProviderTypeGoogle.GetDefaultIssuerURL(), subject, profile)
	assert.NoErr(t, err)
}

func TestCreateUserWithEmptyBody(t *testing.T) {
	t.Parallel()
	tf := newTestFixture(t)
	ctx := context.Background()
	_, err := saveDefaultUserStoreSchema(tf)
	assert.NoErr(t, err)
	w := tf.request(http.MethodPost, "/authn/users", nil)
	resp := w.Result()
	assert.Equal(t, resp.StatusCode, http.StatusCreated)
	var cur idp.UserResponse
	assert.NoErr(t, json.NewDecoder(resp.Body).Decode(&cur))
	userID := cur.ID

	host, err := tenantmap.GetHostFromTenantURL(tf.tenant.TenantURL)
	assert.NoErr(t, err)
	tenant, err := tf.tenants.GetTenantStateForHostname(ctx, host)
	assert.NoErr(t, err)
	tenantIDP := authn.NewAuthN(ctx, tenant)
	cm, err := storage.NewUserstoreColumnManager(ctx, tenantIDP.ConfigStorage)
	assert.NoErr(t, err)
	dtm, err := storage.NewDataTypeManager(ctx, tenantIDP.ConfigStorage)
	assert.NoErr(t, err)
	user, _, _, err := tenantIDP.UserMultiRegionStorage.GetUser(ctx, cm, dtm, userID, false)
	assert.NoErr(t, err)
	assert.Equal(t, user.Profile["email"], nil)
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	tf := newTestFixture(t)
	ctx := context.Background()
	columns, err := saveDefaultUserStoreSchema(tf)
	assert.NoErr(t, err)
	fieldName1 := columns[0].Name
	fieldName2 := columns[1].Name

	now := time.Now().UTC()
	profile := userstore.Record{
		"email":    testEmail, // Intentionally different from username
		"name":     "testname" + uuid.Must(uuid.NewV4()).String(),
		"nickname": "testnickname" + uuid.Must(uuid.NewV4()).String(),
		"picture":  "testpicurl" + uuid.Must(uuid.NewV4()).String(),
		fieldName1: "field1_original",
		fieldName2: now,
	}
	userID, err := createUserWithPasswordAndProfile(tf, "test@test.com", "testme", profile)
	assert.NoErr(t, err)

	host, err := tenantmap.GetHostFromTenantURL(tf.tenant.TenantURL)
	assert.NoErr(t, err)
	client, err := idp.NewClient(tf.tenant.TenantURL, idp.JSONClient(jsonclient.HeaderHost(host), jsonclient.HeaderAuth(tf.bearerToken)))
	assert.NoErr(t, err)
	resp, err := client.GetUser(ctx, userID)
	assert.NoErr(t, err)
	assert.Equal(t, resp.ID, userID)
	assert.Equal(t, resp.Profile["email"], profile["email"])
	assert.Equal(t, resp.Profile[fieldName1], "field1_original")
	// TODO: this is not awesome - how do we deal with client-side types in each language
	// in a sane manner? Do we need to get the schema client side and write per-language type casting?
	ts, err := time.Parse(time.RFC3339, resp.Profile[fieldName2].(string))
	assert.NoErr(t, err)
	assert.True(t, ts.Sub(now) < time.Second)

	updateResp, err := client.UpdateUser(ctx, userID, idp.UpdateUserRequest{
		Profile: userstore.Record{
			"email_verified": true,
			"name":           "", // intentionally set to empty string
			"nickname":       "newnickname",
			fieldName1:       "field1_updated",
		},
	})

	assert.NoErr(t, err)

	resp, err = client.GetUser(ctx, userID)
	assert.NoErr(t, err)
	assert.Equal(t, resp, updateResp)
	assert.Equal(t, resp.ID, userID)
	assert.Equal(t, resp.Profile["email"], testEmail)
	assert.Equal(t, resp.Profile["email_verified"], "true")
	assert.Equal(t, resp.Profile["name"], nil)
	assert.Equal(t, resp.Profile["nickname"], "newnickname")
	assert.Equal(t, resp.Profile["picture"], profile["picture"])
	assert.Equal(t, resp.Profile[fieldName1], "field1_updated")
	ts, err = time.Parse(time.RFC3339, resp.Profile[fieldName2].(string))
	assert.NoErr(t, err)
	assert.True(t, ts.Sub(now) < time.Second)

	// Also test ListUsers with and without authns
	usersResp, err := client.ListUsers(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, len(usersResp.Data), 1)
}

func TestSchema(t *testing.T) {
	t.Parallel()
	tf := newTestFixture(t)
	ctx := tf.ctx

	profile := userstore.Record{"email": "test@test.com"}

	// Test a create without the schema existing yet
	// NOTE: this test case revealed a bug where we cached the schema on first tenant access, and
	// did not update it on subsequent edits.
	badProfile := userstore.Record{"bad_field_name": "something"}

	_, err := createUserWithPasswordAndProfile(tf, "test@test.com", "testme", badProfile)
	var jsonClientErr jsonclient.Error
	assert.True(t, errors.As(err, &jsonClientErr), assert.Must())
	assert.Equal(t, jsonClientErr.StatusCode, http.StatusBadRequest)

	columns, err := saveDefaultUserStoreSchema(tf)
	assert.NoErr(t, err)
	fieldName1 := columns[0].Name
	fieldName2 := columns[1].Name

	// Test a couple bad creates
	badProfile1 := userstore.Record{
		fieldName2: "not_a_timestamp",
	}
	_, err = createUserWithPasswordAndProfile(tf, "test@test.com", "testme", badProfile1)
	assert.True(t, errors.As(err, &jsonClientErr), assert.Must())
	assert.Equal(t, jsonClientErr.StatusCode, http.StatusBadRequest)

	badProfile2 := userstore.Record{
		"bad_field_name": "not_a_valid_field",
	}
	_, err = createUserWithPasswordAndProfile(tf, "test@test.com", "testme", badProfile2)
	assert.True(t, errors.As(err, &jsonClientErr), assert.Must())
	assert.Equal(t, jsonClientErr.StatusCode, http.StatusBadRequest)

	// Test a good create
	goodProfile := profile
	goodProfile[fieldName1] = "valid_string"
	userID, err := createUserWithPasswordAndProfile(tf, "test@test.com", "testme", goodProfile)
	assert.NoErr(t, err)

	host, err := tenantmap.GetHostFromTenantURL(tf.tenant.TenantURL)
	assert.NoErr(t, err)
	client, err := idp.NewClient(tf.tenant.TenantURL, idp.JSONClient(jsonclient.HeaderHost(host), jsonclient.HeaderAuth(tf.bearerToken)))
	assert.NoErr(t, err)
	resp, err := client.GetUser(ctx, userID)
	assert.NoErr(t, err)
	assert.Equal(t, resp.Profile[fieldName1], "valid_string")

	// Test a couple bad updates
	_, err = client.UpdateUser(ctx, userID, idp.UpdateUserRequest{
		Profile: badProfile1,
	})
	assert.True(t, errors.As(err, &jsonClientErr), assert.Must())
	assert.Equal(t, jsonClientErr.StatusCode, http.StatusBadRequest)

	_, err = client.UpdateUser(ctx, userID, idp.UpdateUserRequest{
		Profile: badProfile2,
	})
	assert.True(t, errors.As(err, &jsonClientErr), assert.Must())
	assert.Equal(t, jsonClientErr.StatusCode, http.StatusBadRequest)

	// Test a valid update
	now := time.Now().UTC()
	goodProfile = profile
	goodProfile[fieldName2] = now

	updateResp, err := client.UpdateUser(ctx, userID, idp.UpdateUserRequest{
		Profile: goodProfile,
	})
	assert.NoErr(t, err)

	resp, err = client.GetUser(ctx, userID)
	assert.NoErr(t, err)
	assert.Equal(t, resp.Profile, updateResp.Profile)
	assert.Equal(t, resp.Profile[fieldName1], "valid_string")
	ts, err := time.Parse(time.RFC3339, resp.Profile[fieldName2].(string))
	assert.NoErr(t, err)
	assert.True(t, ts.Sub(now) < time.Second)

	// Test a null update
	_, err = client.UpdateUser(ctx, userID, idp.UpdateUserRequest{
		Profile: userstore.Record{fieldName1: nil},
	})
	assert.NoErr(t, err)
	resp, err = client.GetUser(ctx, userID)
	assert.NoErr(t, err)
	assert.Equal(t, resp.Profile[fieldName1], nil)

	// test default value
	newFieldID := uuid.Must(uuid.NewV4())
	defaultedFieldName := "a_new_column"
	err = createColumn(tf, userstore.Column{
		ID:           newFieldID,
		Table:        "users",
		Name:         defaultedFieldName,
		DataType:     datatype.String,
		IsArray:      false,
		DefaultValue: "new_column_default",
		IndexType:    userstore.ColumnIndexTypeIndexed,
	})
	assert.NoErr(t, err)
	userID, err = createUserWithPasswordAndProfile(tf, "testdefault@test.com", "testme", profile)
	assert.NoErr(t, err)

	resp, err = client.GetUser(ctx, userID)
	assert.NoErr(t, err)
	assert.Equal(t, resp.Profile[defaultedFieldName], "new_column_default")

	// create user with a different value specified for the column that has a default
	defaultOverrideProfile := profile
	defaultOverrideProfile[defaultedFieldName] = "non_default_value"
	defaultOverrideUserID, err := createUserWithPasswordAndProfile(tf, "testdefault2@test.com", "testdefault2", defaultOverrideProfile)
	assert.NoErr(t, err)
	resp, err = client.GetUser(ctx, defaultOverrideUserID)
	assert.NoErr(t, err)
	assert.Equal(t, resp.Profile[defaultedFieldName], "non_default_value")

	// test uniqueness
	newFieldID = uuid.Must(uuid.NewV4())
	uniqueFieldName := "unique_column"
	err = createColumn(tf, userstore.Column{
		ID:        newFieldID,
		Table:     "users",
		Name:      uniqueFieldName,
		DataType:  datatype.String,
		IsArray:   false,
		IndexType: userstore.ColumnIndexTypeUnique,
	})
	assert.NoErr(t, err)

	_, err = client.UpdateUser(ctx, userID, idp.UpdateUserRequest{
		Profile: userstore.Record{
			uniqueFieldName: "dup_value",
		},
	})
	assert.NoErr(t, err)
	resp, err = client.GetUser(ctx, userID)
	assert.NoErr(t, err)
	assert.Equal(t, resp.Profile[uniqueFieldName], "dup_value")
	assert.Equal(t, resp.Profile[defaultedFieldName], "new_column_default")

	userID2, err := createUserWithPasswordAndProfile(tf, "test2@test.com", "testme2", goodProfile)
	assert.NoErr(t, err)
	_, err = client.UpdateUser(ctx, userID2, idp.UpdateUserRequest{
		Profile: userstore.Record{
			uniqueFieldName: "dup_value",
		},
	})
	assert.True(t, errors.As(err, &jsonClientErr), assert.Must())

	mgmtClient, err := idp.NewManagementClient(tf.tenant.TenantURL, jsonclient.HeaderHost(host), jsonclient.HeaderAuth(tf.bearerToken))
	assert.NoErr(t, err)

	users, err := mgmtClient.ListUserBaseProfilesAndAuthNForEmail(ctx, profile["email"].(string), idp.AuthnTypeAll)
	assert.NoErr(t, err)
	assert.Equal(t, len(users), 4, assert.Must())

	// test uniqueness on create
	dupProfile := profile
	dupProfile[uniqueFieldName] = "dup_value"

	_, err = createUserWithPasswordAndProfile(tf, "test3@test.com", "testme3", dupProfile)
	assert.True(t, errors.As(err, &jsonClientErr), assert.Must())
	users, err = mgmtClient.ListUserBaseProfilesAndAuthNForEmail(ctx, profile["email"].(string), idp.AuthnTypeAll)
	assert.NoErr(t, err)
	assert.Equal(t, len(users), 4, assert.Must())
}

// TestUpdatePassword ensures that passwords can be updated properly for an existing user
func TestUpdatePassword(t *testing.T) {
	t.Parallel()
	tf := newTestFixture(t)
	ctx := context.Background()
	ts, err := tf.tenants.GetTenantStateForID(ctx, tf.tenant.ID)
	assert.NoErr(t, err)

	s := idptesthelpers.NewStorage(ctx, t, tf.tenantDB, tf.tenant.ID)
	umrs := storage.NewUserMultiRegionStorage(ctx, ts.UserRegionDbMap, tf.tenant.ID)
	m := authn.NewManager(s, umrs)

	username := genUsername()

	// Must create account before attempting to change password
	assert.NotNil(t, m.UpdateUsernamePassword(ctx, username, "pass"))

	_, err = createUserWithPasswordAndProfile(tf, username, "pass", userstore.Record{})
	assert.NoErr(t, err)

	a, err := s.GetPasswordAuthnForUsername(ctx, username)
	assert.NoErr(t, err)
	assert.Equal(t, a.Username, username)
	assert.IsNil(t, bcrypt.CompareHashAndPassword([]byte(a.Password), []byte("pass")))

	assert.IsNil(t, m.UpdateUsernamePassword(ctx, username, "newpass"))
	a, err = s.GetPasswordAuthnForUsername(ctx, username)
	assert.NoErr(t, err)
	assert.Equal(t, a.Username, username)
	assert.IsNil(t, bcrypt.CompareHashAndPassword([]byte(a.Password), []byte("newpass")))
}

// TestNoOverwritePassword ensures that a new user with the same username can't be created or
// overwrite the creds of an existing user.
func TestNoOverwritePassword(t *testing.T) {
	t.Parallel()
	tf := newTestFixture(t)
	ctx := tf.ctx
	s := idptesthelpers.NewStorage(ctx, t, tf.tenantDB, tf.tenant.ID)
	us := storage.NewUserStorage(ctx, tf.tenantDB, "", tf.tenant.ID)

	pager, err := storage.NewBaseUserPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	assert.NoErr(t, err)

	users, respFields, err := us.ListBaseUsersPaginated(ctx, *pager, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(users), 0)
	assert.False(t, respFields.HasNext)
	assert.False(t, respFields.HasPrev)

	username := genUsername()

	_, err = createUserWithPasswordAndProfile(tf, username, "pass", userstore.Record{})

	assert.NoErr(t, err)
	users, respFields, err = us.ListBaseUsersPaginated(ctx, *pager, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(users), 1)
	assert.False(t, respFields.HasNext)
	assert.False(t, respFields.HasPrev)

	_, err = createUserWithPasswordAndProfile(tf, username, "newpass", userstore.Record{})
	assert.NotNil(t, err, assert.Must())
	users, respFields, err = us.ListBaseUsersPaginated(ctx, *pager, false)
	assert.NoErr(t, err)
	assert.Equal(t, len(users), 1)
	assert.False(t, respFields.HasNext)
	assert.False(t, respFields.HasPrev)

	an, err := s.GetPasswordAuthnForUsername(ctx, username)
	assert.NoErr(t, err)
	assert.Equal(t, an.Username, username)
	assert.IsNil(t, bcrypt.CompareHashAndPassword([]byte(an.Password), []byte("pass")))
}

func TestMain(m *testing.M) {
	// Adjust working dir to match what our services expect.
	os.Chdir("../../..")

	os.Exit(m.Run())
}
