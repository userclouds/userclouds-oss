package test

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"

	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/logdb"
	"userclouds.com/internal/tenantdb"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/internal/testhelpers"
	"userclouds.com/internal/uctest"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/manager"
	"userclouds.com/plex/routes"
)

// User is a basic/mock user object
type User struct {
	Authn   idp.UserAuthn
	Profile userstore.Record

	SupportedMFAChannels oidc.MFAChannels

	// Transient values stored for testing
	MFACode string
}

// EnableMFA creates a supported email MFA challenge with the user's email
func (u *User) EnableMFA() {
	email := u.Profile.StringValue("email")
	channel := oidc.NewMFAChannel(oidc.MFAEmailChannel, email, email)
	channel.Verified = true
	channel.Primary = true
	u.SupportedMFAChannels = oidc.MFAChannels{
		ChannelTypes:     oidc.MFAChannelTypeSet{oidc.MFAEmailChannel: true},
		Channels:         oidc.MFAChannelsByID{channel.ID: channel},
		PrimaryChannelID: channel.ID,
	}
}

// IDP is a mock IDP used by the plex handler in lieu of a real IDP
// TODO: implement iface.Client methods to move more plex tests over to use this fixture
type IDP struct {
	iface.ManagementClient
	iface.Client

	ProviderID uuid.UUID
	Active     bool

	// Map of User ID to user
	Users map[string]User

	// Map of User ID to supported MFA Channels
	SupportedMFAChannels map[string]oidc.MFAChannels

	// FailNextUUPRequest causes the next UpdateUsernamePassword call to fail if true, then resets to false
	// TODO: we can invest in better failure injection when more tests need it.
	FailNextUUPRequest bool

	// Track total number of logouts.
	// TODO: invest in more general test hooks/callbacks when more tests need it.
	LogoutCount int
}

// Fixture encapsulates the test setup/state necessary to test plex handlers,
// and contains mocks to evaluate state before/after operations.
type Fixture struct {
	provider.Factory

	CompanyConfigStorage *companyconfig.Storage
	Storage              *storage.Storage
	Handler              http.Handler
	Email                *uctest.EmailClient
	ActiveIDP            *IDP
	FollowerIDP          *IDP
	PublicKey            *rsa.PublicKey
	Company              *companyconfig.Company
	Tenant               *companyconfig.Tenant
	TenantDB             *ucdb.DB
	Testing              *testing.T

	RequestFactory *RequestFactory
	OverrideOIDC   map[oidc.ProviderType]*oidc.Authenticator
}

const mfaTokenPrefix = "mfa_token_for: "
const userIDSuffix = "_userid"

func mfaTokenFromUsername(username string) string {
	return mfaTokenPrefix + username
}

func usernameFromMFAToken(mfaToken string) string {
	return strings.TrimPrefix(mfaToken, mfaTokenPrefix)
}

func userIDFromEmail(email string) string {
	return email + userIDSuffix
}

func (tf *Fixture) getIDP(providerID uuid.UUID) (*IDP, error) {
	followerID := uuid.Nil
	if providerID == tf.ActiveIDP.ProviderID {
		return tf.ActiveIDP, nil
	} else if tf.FollowerIDP != nil {
		followerID = tf.FollowerIDP.ProviderID
		if providerID == followerID && providerID != uuid.Nil {
			return tf.FollowerIDP, nil
		}
	}
	return nil, ucerr.Errorf("provider %s does not match active (%s) or follower (%s) provider ID", providerID, tf.ActiveIDP.ProviderID, followerID)
}

// NewClient implements provider.Factory
func (tf *Fixture) NewClient(ctx context.Context, p tenantplex.Provider, plexClientID string, appID uuid.UUID) (iface.Client, error) {
	i, err := tf.getIDP(p.ID)
	return i, ucerr.Wrap(err)
}

// NewManagementClient implements provider.Factory
func (tf *Fixture) NewManagementClient(ctx context.Context, _ *tenantplex.TenantConfig, p tenantplex.Provider, _, _ uuid.UUID) (iface.ManagementClient, error) {
	i, err := tf.getIDP(p.ID)
	return i, ucerr.Wrap(err)
}

// NewOIDCAuthenticator implements provider.Factory and allows selective overriding of oidc providers.
func (tf *Fixture) NewOIDCAuthenticator(ctx context.Context, pt oidc.ProviderType, issuerURL string, cfg tenantplex.OIDCProviders, redirectURL *url.URL) (*oidc.Authenticator, error) {
	if authr, ok := tf.OverrideOIDC[pt]; ok {
		return authr, nil
	}
	var pf provider.ProdFactory
	authr, err := pf.NewOIDCAuthenticator(ctx, pt, issuerURL, cfg, redirectURL)
	return authr, ucerr.Wrap(err)
}

func (i *IDP) claimsResponse(user *User) *iface.LoginResponseWithClaims {
	return &iface.LoginResponseWithClaims{
		// Handle the 2 special claims required by oidc.ExtractClaims
		Claims: jwt.MapClaims{
			"sub":   userIDFromEmail(user.Profile.StringValue("email")),
			"email": user.Profile.StringValue("email"),
		},
		Status: idp.LoginStatusSuccess,
	}
}

// UsernamePasswordLogin implements iface.Client
func (i *IDP) UsernamePasswordLogin(ctx context.Context, username, password string) (*iface.LoginResponseWithClaims, error) {
	var user *User

	for _, u := range i.Users {
		if u.Authn.AuthnType == idp.AuthnTypePassword && u.Authn.Username == username {
			user = &u
			break
		}
	}
	if user == nil || user.Authn.Password != password {
		return nil, ucerr.Wrap(ucerr.ErrIncorrectUsernamePassword)
	}

	claimsResponse := i.claimsResponse(user)

	if user.SupportedMFAChannels.HasPrimaryChannel() {
		claimsResponse.Status = idp.LoginStatusMFARequired
		claimsResponse.MFAToken = mfaTokenFromUsername(username)
		claimsResponse.MFAProvider = i.ProviderID
		claimsResponse.SupportedMFAChannels = user.SupportedMFAChannels
	}

	return claimsResponse, nil
}

// MFAChallenge implements iface.Client
func (i *IDP) MFAChallenge(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*oidc.MFAChannel, error) {
	username := usernameFromMFAToken(mfaToken)
	var user *User
	for _, u := range i.Users {
		if u.Authn.AuthnType == idp.AuthnTypePassword && u.Authn.Username == username {
			user = &u
			break
		}
	}

	// Set current MFA code
	user.MFACode = crypto.MustRandomDigits(6)

	return &channel, nil
}

// MFALogin implements iface.Client
func (i *IDP) MFALogin(ctx context.Context, mfaToken, mfaCode string, mfaChannel oidc.MFAChannel) (*iface.LoginResponseWithClaims, error) {
	username := usernameFromMFAToken(mfaToken)
	var user *User
	for _, u := range i.Users {
		if u.Authn.AuthnType == idp.AuthnTypePassword && u.Authn.Username == username {
			user = &u
			break
		}
	}

	if mfaCode != user.MFACode {
		return &iface.LoginResponseWithClaims{Status: idp.LoginStatusMFACodeInvalid}, nil
	}

	return i.claimsResponse(user), nil
}

// LoginURL implements iface.Client
func (i *IDP) LoginURL(ctx context.Context, sessionID uuid.UUID, _ *tenantplex.App) (*url.URL, error) {
	return paths.LoginURL(ctx, sessionID)
}

// Logout implements iface.Client
func (i *IDP) Logout(ctx context.Context, redirectURL string) (string, error) {
	i.LogoutCount++
	return "", nil
}

// CreateUserWithPassword implements iface.ManagementClient
func (i *IDP) CreateUserWithPassword(_ context.Context, username, password string, profile iface.UserProfile) (string, error) {
	userID := userIDFromEmail(profile.Email)
	i.Users[userID] = User{
		Authn:   idp.NewPasswordAuthn(username, password),
		Profile: profile.ToUserstoreRecord(),
	}
	return userID, nil
}

// CreateUserWithOIDC implements iface.ManagementClient
func (i *IDP) CreateUserWithOIDC(ctx context.Context, provider oidc.ProviderType, issuerURL string, oidcSubject string, profile iface.UserProfile) (string, error) {
	userID := userIDFromEmail(profile.Email)
	i.Users[userID] = User{
		Authn:   idp.NewOIDCAuthn(provider, issuerURL, oidcSubject),
		Profile: profile.ToUserstoreRecord(),
	}
	return userID, nil
}

// GetUser implements iface.ManagementClient
func (i *IDP) GetUser(ctx context.Context, userID string) (*iface.UserProfile, error) {
	user, ok := i.Users[userID]
	if !ok {
		return nil, ucerr.Errorf("user ID '%s' not found", userID)
	}
	return &iface.UserProfile{
		ID: userID,
		UserBaseProfile: idp.UserBaseProfile{
			Name:          user.Profile.StringValue("name"),
			Nickname:      user.Profile.StringValue("nickname"),
			Email:         user.Profile.StringValue("email"),
			EmailVerified: user.Profile.BoolValue("email_verified"),
			Picture:       user.Profile.StringValue("picture"),
		},
		Authns: []idp.UserAuthn{user.Authn},
	}, nil
}

// GetUserForOIDC implements iface.ManagementClient
func (i *IDP) GetUserForOIDC(ctx context.Context, provider oidc.ProviderType, issuerURL string, oidcSubject string, _ string) (*iface.UserProfile, error) {
	for userID, user := range i.Users {
		if user.Authn.AuthnType == idp.AuthnTypeOIDC && user.Authn.OIDCProvider == provider && user.Authn.OIDCIssuerURL == issuerURL && user.Authn.OIDCSubject == oidcSubject {
			return &iface.UserProfile{
				ID: userID,
				UserBaseProfile: idp.UserBaseProfile{
					Name:          user.Profile.StringValue("name"),
					Nickname:      user.Profile.StringValue("nickname"),
					Email:         user.Profile.StringValue("email"),
					EmailVerified: user.Profile.BoolValue("email_verified"),
					Picture:       user.Profile.StringValue("picture"),
				},
				Authns: []idp.UserAuthn{user.Authn},
			}, nil
		}
	}
	return nil, ucerr.Wrap(iface.ErrUserNotFound)
}

// ListUsersForEmail implements iface.ManagementClient
func (i *IDP) ListUsersForEmail(ctx context.Context, email string, authnType idp.AuthnType) ([]iface.UserProfile, error) {
	userID := userIDFromEmail(email)
	user, err := i.GetUser(ctx, userID)
	if err != nil {
		return nil, nil
	}
	return []iface.UserProfile{*user}, nil
}

// SetEmailVerified implements iface.ManagementClient
func (i *IDP) SetEmailVerified(_ context.Context, userID string, verified bool) error {
	u, ok := i.Users[userID]
	if !ok {
		return ucerr.Errorf("user ID '%s' not found", userID)
	}
	u.Profile["email_verified"] = verified
	i.Users[userID] = u
	return nil
}

// UpdateUsernamePassword implements iface.ManagementClient
func (i *IDP) UpdateUsernamePassword(ctx context.Context, username, password string) error {
	if i.FailNextUUPRequest {
		i.FailNextUUPRequest = false
		return ucerr.New("failed to update username & password")
	}
	for id, user := range i.Users {
		if user.Authn.Username == username {
			user.Authn.Password = password
			i.Users[id] = user
			return nil
		}
	}
	return ucerr.Wrap(iface.ErrUserNotFound)
}

func (i IDP) String() string {
	if i.Active {
		return fmt.Sprintf("active IDP, provider ID: %s", i.ProviderID)
	}
	return fmt.Sprintf("follower IDP, provider ID: %s", i.ProviderID)
}

// NewIDP creates a new mock IDP
func NewIDP() *IDP {
	return &IDP{
		Users: map[string]User{},
	}
}

// NewFixture creates a new Plex integration test fixture which
// includes a handler, mock email client, and mock user table.
// TODO: is it worth supporting multiple tenants in this fixture?
func NewFixture(t *testing.T, tc tenantplex.TenantConfig) *Fixture {
	t.Helper()
	ctx := context.Background()

	tdb := testdb.New(t, migrate.NewTestSchema(tenantdb.Schema))
	tenantDBCfg := testdb.TestConfig(t, tdb)
	logDB := testdb.New(t, migrate.NewTestMigrator(logdb.GetMigrations()))
	logDBCfg := testdb.TestConfig(t, logDB)

	_, _, companyConfigStorage := testhelpers.NewTestStorage(t)
	tf := &Fixture{
		CompanyConfigStorage: companyConfigStorage,
		Storage:              storage.New(ctx, tdb, nil),
		Email:                &uctest.EmailClient{},
		ActiveIDP:            NewIDP(),
		RequestFactory: &RequestFactory{
			hostName: fmt.Sprintf("%v.%v", uuid.Must(uuid.NewV4()), testhelpers.TestTenantSubDomain),
			tcs:      tenantplex.TenantConfigs{tc},
		},
		Testing:  t,
		TenantDB: tdb,
	}

	// We assume only 1 tenant and exactly 1 active + 1 (optional) follower
	followerProviderID := uuid.Nil
	assert.True(t, len(tc.PlexMap.Providers) == 1 || len(tc.PlexMap.Providers) == 2, assert.Must())
	if len(tc.PlexMap.Providers) == 1 || tc.PlexMap.Providers[0].ID == tc.PlexMap.Policy.ActiveProviderID {
		assert.Equal(t, tc.PlexMap.Policy.ActiveProviderID, tc.PlexMap.Providers[0].ID, assert.Must())
		if len(tc.PlexMap.Providers) == 2 {
			followerProviderID = tc.PlexMap.Providers[1].ID
		}
	} else {
		assert.Equal(t, tc.PlexMap.Policy.ActiveProviderID, tc.PlexMap.Providers[1].ID, assert.Must())
		followerProviderID = tc.PlexMap.Providers[0].ID
	}
	tf.ActiveIDP.ProviderID = tc.PlexMap.Policy.ActiveProviderID
	tf.ActiveIDP.Active = true
	if len(tc.PlexMap.Providers) == 2 {
		tf.FollowerIDP = NewIDP()
		tf.FollowerIDP.ProviderID = followerProviderID
		tf.FollowerIDP.Active = false
	}
	company := companyconfig.NewCompany("Kramer Inc.", companyconfig.CompanyTypeCustomer)
	assert.NoErr(t, tf.CompanyConfigStorage.SaveCompany(ctx, &company))
	// Create underlying DB representation for Tenant based on tenant config, since
	// multitenant middleware in Plex uses the DB as source of truth.
	tf.Tenant = &companyconfig.Tenant{
		BaseModel: ucdb.NewBase(),
		Name:      "unused",
		CompanyID: company.ID,
		TenantURL: fmt.Sprintf("http://%v", tf.RequestFactory.hostName),
	}
	tf.Company = &company
	assert.NoErr(t, tf.CompanyConfigStorage.SaveTenant(ctx, tf.Tenant))
	tp := tenantplex.TenantPlex{
		VersionBaseModel: ucdb.NewVersionBaseWithID(tf.Tenant.ID),
		PlexConfig:       tc,
	}
	mgr := manager.NewFromDB(tf.TenantDB, cachetesthelpers.NewCacheConfig())
	assert.NoErr(t, mgr.SaveTenantPlex(ctx, &tp))

	ti := companyconfig.TenantInternal{
		BaseModel:         ucdb.NewBaseWithID(tf.Tenant.ID),
		TenantDBConfig:    tenantDBCfg,
		PrimaryUserRegion: region.DefaultUserDataRegionForUniverse(universe.Test),
		LogConfig:         companyconfig.TenantLogConfig{LogDB: logDBCfg},
	}
	assert.NoErr(t, tf.CompanyConfigStorage.SaveTenantInternal(ctx, &ti))
	hb := builder.NewHandlerBuilder()
	m2mAuth, err := m2m.GetM2MTokenSource(ctx, tf.Tenant.ID)
	assert.NoErr(t, err)
	routes.InitForTests(ctx, m2mAuth, hb, companyConfigStorage, uctest.JWTVerifier{}, &testhelpers.SecurityValidator{}, tf.Email, tf, nil, tf.Tenant.ID)
	tf.Handler = hb.Build()
	if tc.Keys.PublicKey != "" {
		key, err := ucjwt.LoadRSAPublicKey([]byte(tc.Keys.PublicKey))
		assert.NoErr(t, err)
		tf.PublicKey = key
	}
	return tf
}
