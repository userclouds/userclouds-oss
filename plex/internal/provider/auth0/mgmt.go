package auth0

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"userclouds.com/idp"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/sdkclient"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/messaging/email/emailaddress"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider/iface"
)

// MgmtClient represents an auth0 management client
type MgmtClient struct {
	iface.ManagementClient
	// TODO: supply per-request extra headers so we can re-use a single json client
	cfg    tenantplex.Auth0Provider
	client *sdkclient.Client
}

// NewManagementClient returns an auth0 client that talks to the management API
// NB: this returns a *MgmtClient and not an iface.ManagementClient because we hang
// some extra app sync methods on it that are Auth0 specific
func NewManagementClient(ctx context.Context, cfg tenantplex.Auth0Provider) (*MgmtClient, error) {
	baseURL := &url.URL{
		Scheme: "https",
		Host:   cfg.Domain,
	}
	jc := sdkclient.New(baseURL.String(), "plex-management")

	// NOTE: we generally [re-]create management clients per request, instead of keeping them
	// alive indefinitely. The management tokens we get here will expire at some point (hours, by default)
	// which is fine for now. If we keep clients alive longer we need a codepath to refresh the token.
	token, err := getManagementToken(ctx, cfg, jc)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	jc.Apply(jsonclient.HeaderAuthBearer(token.AccessToken))

	return &MgmtClient{
		cfg:    cfg,
		client: jc,
	}, nil
}

type createUserRequest struct {
	Connection    string `json:"connection"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Username      string `json:"username,omitempty"`
	Password      string `json:"password" validate:"notempty"`
	Name          string `json:"name,omitempty"`
	Nickname      string `json:"nickname,omitempty"`
	Picture       string `json:"picture,omitempty"`
}

//go:generate genvalidate createUserRequest

func (cur createUserRequest) extraValidate() error {
	a := emailaddress.Address(cur.Email)
	if err := a.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

type createUserResponse struct {
	UserID string `json:"user_id"`
}

const auth0DefaultConnection = "Username-Password-Authentication" // TODO: from config?

// CreateUserWithPassword implements provider.iface.ManagementClient
func (mc MgmtClient) CreateUserWithPassword(ctx context.Context, username, password string, profile iface.UserProfile) (string, error) {
	// TODO: this is a very minimal struct, Auth0 supports many more fields:
	//       https://auth0.com/docs/api/management/v2#!/Users/post_users
	// TODO: Auth0 allows non-email usernames, but you MUST set the "requires_username" flag on the Database Connection
	// in the Auth0 settings (see https://auth0.com/docs/authenticate/database-connections/require-username for details).
	// Until/unless we query that flag via the Management API (documentation doens't show how to query 'requires_username',
	// but it probably involves calling https://auth0.com/docs/api/management/v2/#!/Connections/get_connections_by_id),
	// we can't support usernames that aren't email addresses.
	if profile.Email != username {
		return "", ucerr.Errorf("support for non-email usernames ('%s') in Auth0 not yet implemented; email in profile '%s' must match username", username, profile.Email)
	}
	req := createUserRequest{
		Connection:    auth0DefaultConnection,
		Password:      password,
		Name:          profile.Name,
		Nickname:      profile.Nickname,
		Email:         profile.Email,
		EmailVerified: profile.EmailVerified,
		Picture:       profile.Picture,
	}
	if err := req.Validate(); err != nil {
		return "", ucerr.Wrap(err)
	}

	var resp createUserResponse
	if err := mc.client.Post(ctx, "/api/v2/users", req, &resp); err != nil {
		uclog.Debugf(ctx, "auth0 create user error: %+v", err)
		return "", ucerr.Wrap(err)
	}

	return resp.UserID, nil
}

type userIdentity struct {
	Provider string `json:"provider"`
	IsSocial bool   `json:"isSocial"`
	UserID   string `json:"user_id"`
}

func (ui userIdentity) getOIDCProviderInfo() (provider oidc.ProviderType, issuerURL string) {
	if !ui.IsSocial {
		return oidc.ProviderTypeNone, ""
	}

	// NOTE: this should be updated with any social providers that both Auth0 and we support
	if ui.Provider == "google-oauth2" {
		return oidc.ProviderTypeGoogle, oidc.ProviderTypeGoogle.GetDefaultIssuerURL()
	}
	// TODO: gint - 5/9/23 - add Facebook and LinkedIn support for Auth0

	return oidc.ProviderTypeUnsupported, ""
}

type userResponse struct {
	UserID        string         `json:"user_id"`
	Email         string         `json:"email"`
	EmailVerified bool           `json:"email_verified"`
	Name          string         `json:"name"`
	Nickname      string         `json:"nickname"`
	Picture       string         `json:"picture"`
	UpdatedAt     string         `json:"updated_at"` // NOTE: Auth0 is non-OIDC compliant and treats this as a string instead of integer seconds from Unix epoch
	Username      string         `json:"username"`   // May be specified if not equal to email
	Identities    []userIdentity `json:"identities" validate:"skip"`
	Multifactor   []string       `json:"multifactor" validate:"skip"`
}

//go:generate genvalidate userResponse

func (ur userResponse) extraValidate() error {
	a := emailaddress.Address(ur.Email)
	if err := a.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

func newUserProfile(auth0User *userResponse) *iface.UserProfile {
	// Technically the timestamp is ISO8601 and RFC3339 is a subset(?) but this works in practice.
	updatedAt, err := time.Parse(time.RFC3339, auth0User.UpdatedAt)
	if err != nil {
		updatedAt = time.Unix(0, 0)
	}
	authns := []idp.UserAuthn{}
	for _, identity := range auth0User.Identities {
		if identity.IsSocial {
			provider, issuerURL := identity.getOIDCProviderInfo()
			authns = append(authns,
				idp.UserAuthn{
					AuthnType:     idp.AuthnTypeOIDC,
					OIDCProvider:  provider,
					OIDCIssuerURL: issuerURL,
					OIDCSubject:   identity.UserID,
				})
		} else {
			// From inspection of return values I am pretty sure this is right,
			// though with SMS login we may need to add another special case?
			username := auth0User.Username
			if username == "" {
				username = auth0User.Email
			}
			authns = append(authns, idp.UserAuthn{
				AuthnType: idp.AuthnTypePassword,
				Username:  username,
			})

		}

	}
	mfaChannels := []idp.UserMFAChannel{}
	for _, multifactor := range auth0User.Multifactor {
		// TODO: gint - 5/10/23 - need to figure out what is returned in the Multifactor array
		mfaChannels = append(mfaChannels,
			idp.UserMFAChannel{
				ChannelType:        oidc.MFAAuth0EmailChannel,
				ChannelDescription: multifactor,
			})
	}

	return &iface.UserProfile{
		ID:        auth0User.UserID,
		UpdatedAt: updatedAt.Unix(),
		UserBaseProfile: idp.UserBaseProfile{
			Email:         auth0User.Email,
			EmailVerified: auth0User.EmailVerified,
			Name:          auth0User.Name,
			Nickname:      auth0User.Nickname,
			Picture:       auth0User.Picture,
		},
		// TODO: I'm not sure 100% this is right, I spot checked a few accounts but we should
		// probably test more scenarios.
		Authns:      authns,
		MFAChannels: mfaChannels,
	}
}

// GetUser implements provider.iface.ManagementClient
func (mc MgmtClient) GetUser(ctx context.Context, userID string) (*iface.UserProfile, error) {
	pathURL := &url.URL{
		Path: fmt.Sprintf("/api/v2/users/%s", userID),
	}
	var resp userResponse
	if err := mc.client.Get(ctx, pathURL.String(), &resp); err != nil {
		return nil, ucerr.Wrap(iface.ClassifyGetUserError(err))
	}
	return newUserProfile(&resp), nil
}

// ListUsersForEmail implements provider.iface.ManagementClient
func (mc MgmtClient) ListUsersForEmail(ctx context.Context, email string, authnType idp.AuthnType) ([]iface.UserProfile, error) {
	pathURL := &url.URL{
		Path:     "/api/v2/users",
		RawQuery: url.Values{"q": []string{fmt.Sprintf("email:\"%s\"", email)}}.Encode(),
	}

	var resp []userResponse
	if err := mc.client.Get(ctx, pathURL.String(), &resp); err != nil {
		uclog.Debugf(ctx, "auth0 get users error: %+v", err)
		return nil, ucerr.Wrap(err)
	}

	users := []iface.UserProfile{}
	for _, user := range resp {
		var match bool
		for _, identity := range user.Identities {
			if identity.Provider == "auth0" &&
				(authnType == idp.AuthnTypePassword || authnType == idp.AuthnTypeAll) {
				match = true
			} else if identity.IsSocial &&
				(authnType == idp.AuthnTypeOIDC || authnType == idp.AuthnTypeAll) {
				match = true
			}
		}
		if match {
			users = append(users, *newUserProfile(&user))
		}
	}

	return users, nil
}

func (mc MgmtClient) updateUser(ctx context.Context, userID string, body any) error {
	pathURL := &url.URL{
		Path: fmt.Sprintf("/api/v2/users/%s", userID),
	}

	if err := mc.client.Patch(ctx, pathURL.String(), body, nil); err != nil {
		uclog.Debugf(ctx, "auth0 updateUser error: %+v", err)
		return ucerr.Wrap(err)
	}

	return nil
}

// SetEmailVerified implements provider.iface.ManagementClient
func (mc MgmtClient) SetEmailVerified(ctx context.Context, userID string, verified bool) error {
	type req struct {
		EmailVerified bool `json:"email_verified"`
	}

	return ucerr.Wrap(mc.updateUser(ctx, userID, &req{true}))
}

// UpdateUsernamePassword implements provider.iface.ManagementClient
func (mc MgmtClient) UpdateUsernamePassword(ctx context.Context, username, password string) error {
	// TODO: email != username but for now that's all we support
	users, err := mc.ListUsersForEmail(ctx, username, idp.AuthnTypePassword)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if len(users) != 1 {
		// Wrap the UserNotFound error here with extra context
		return ucerr.Errorf("expected 1 user with username '%s', got %d: %w", username, len(users), iface.ErrUserNotFound)
	}

	type req struct {
		Connection string `json:"connection"`
		Password   string `json:"password"`
	}

	r := &req{
		Connection: auth0DefaultConnection,
		Password:   password,
	}
	return ucerr.Wrap(iface.ClassifyGetUserError(mc.updateUser(ctx, users[0].ID, r)))
}

// TODO: switch over to use oidc.ClientCredentialsParameters + GetToken()
func getManagementToken(ctx context.Context, cfg tenantplex.Auth0Provider, jc *sdkclient.Client) (*oidc.TokenResponse, error) {
	// TODO: this probably belonds in oidc.types?
	type req struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Audience     string `json:"audience"`
		GrantType    string `json:"grant_type"`
	}

	cs, err := cfg.Management.ClientSecret.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	payload := req{
		ClientID:     cfg.Management.ClientID,
		ClientSecret: cs,
		Audience:     cfg.Management.Audience,
		GrantType:    "client_credentials",
	}

	var res oidc.TokenResponse

	uclog.Debugf(ctx, "requesting auth0 mgmt token")
	if err := jc.Post(ctx, "/oauth/token", payload, &res); err != nil {
		uclog.Debugf(ctx, "mgmt token error response: %v", err)
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// String implements Stringer
func (mc MgmtClient) String() string {
	// NOTE: non-pointer receiver required for this to work on both pointer & non-pointer types
	return fmt.Sprintf("type '%s', auth0 domain: '%s', client_id: '%s'",
		tenantplex.ProviderTypeAuth0, mc.cfg.Domain, mc.cfg.Management.ClientID)
}

const maxPages = 20                          // https://auth0.com/docs/manage-users/user-search/retrieve-users-with-get-users-endpoint
const pageSize = 50                          // https://auth0.com/docs/pagination
var perRequestDelay = time.Millisecond * 500 // this exists so we can override it in tests

// ListUsersUpdatedDuring implements provider.iface.ManagementClient
func (mc MgmtClient) ListUsersUpdatedDuring(ctx context.Context, since, until time.Time) ([]iface.UserProfile, error) {
	users := []iface.UserProfile{}
	page := 0

	for {
		vals := url.Values{
			// Auth0 doesn't seem to support Lucene > queries on updated_at, so we're using this janky workaround
			"q":        []string{fmt.Sprintf(`updated_at:["%s" TO "%s"]`, since.Format(time.RFC3339), until.Format(time.RFC3339))},
			"per_page": []string{strconv.Itoa(pageSize)}, // Auth0 max, just to be explicit
			"page":     []string{strconv.Itoa(page)},
		}

		pathURL := &url.URL{
			Path:     "/api/v2/users",
			RawQuery: vals.Encode(),
		}

		var resp []userResponse
		if err := mc.client.Get(ctx, pathURL.String(), &resp); err != nil {
			uclog.Debugf(ctx, "auth0 list users error: %+v", err)
			return nil, ucerr.Wrap(err)
		}

		for _, user := range resp {
			users = append(users, *newUserProfile(&user))
		}

		if len(resp) == pageSize && page < (maxPages-1) {
			// try another page and see?
			time.Sleep(perRequestDelay) // TODO: this is super naive and assumes only one user of this API
			page++
			continue
		}

		break
	}

	return users, nil
}
