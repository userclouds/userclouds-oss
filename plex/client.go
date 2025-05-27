package plex

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/sdkclient"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/messaging/email/emailaddress"
	"userclouds.com/plex/paths"
)

// Client represents a client to talk to the Userclouds IDP
type Client struct {
	client *sdkclient.Client
}

// NewClient constructs a new Plex client
func NewClient(url string, opts ...jsonclient.Option) *Client {
	return &Client{
		client: sdkclient.New(url, "plex", opts...),
	}
}

// SendInviteRequest is the request struct for triggering an email invite
// TODO: when we support SMS, do we use a different struct/handler or just augment this?
type SendInviteRequest struct {
	InviteeEmail string `json:"invitee_email"`

	// TODO: when we require auth we can either derive this value from the token or, if used
	// as a 'management api', we can trust the caller.
	InviterUserID string `json:"inviter_user_id"`
	InviterName   string `json:"inviter_name"`
	InviterEmail  string `json:"inviter_email"`

	// ClientID is used to determine which app in the tenant the user is being invited to (for email template purposes).
	// TODO: in the future we can have per-client settings, defaults, templates, etc. for invited users.
	ClientID string `json:"client_id"`

	// State is an OAuth/OIDC state string that the client app can supply which will be supplied back.
	State string `json:"state"`

	// RedirectURL is where the user should be redirected accepting the invite and signing in.
	RedirectURL string `json:"redirect_url"`

	// InviteText is included in the body of the invite to provide additional context.
	InviteText string `json:"invite_text"`

	// Expires is the UTC time for when the invite expires.
	Expires time.Time `json:"expires"`

	// TODO: future options may include:
	// 1. Lock invite to Email or Domain (i.e. don't allow recipient to change address).
	// 2. Expiration time
	// 3. Apply authz permissions, relationships/memberships, etc. on accept.
	// 4. Show/hide sender of invite.
}

// Validate implements Validateable
func (req SendInviteRequest) Validate() error {
	a := emailaddress.Address(req.InviteeEmail)
	if err := a.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if req.State == "" {
		return ucerr.New("SendInviteRequest.State can't be empty")
	}
	if req.RedirectURL == "" {
		return ucerr.New("SendInviteRequest.RedirectURL can't be empty")
	}
	return nil
}

// SendInvite sends an email invitation to a user.
// TODO: require bearer token; either management token for the client or user-specific token.
func (c *Client) SendInvite(ctx context.Context, req SendInviteRequest) error {
	if err := req.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	if err := c.client.Post(ctx, paths.SendInvitePath, req, nil); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// CreateUserRequest is the request structure to create a user.
type CreateUserRequest struct {
	// SessionID is optional; if an account is created as part of a login session, it should be specified
	// to ensure that certain features (e.g. invites) work. If a tenant disables new user sign ups, for example,
	// this must be specified to validate the invite.
	SessionID uuid.UUID `json:"session_id"`

	ClientID string `json:"client_id" validate:"notempty"`
	Email    string `json:"email"`
	Username string `json:"username" validate:"notempty"`
	Password string `json:"password" validate:"notempty"`
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
	// TODO: add validation of picture URL and likely also integrate into image upload
	Picture string `json:"picture"`
}

//go:generate genvalidate CreateUserRequest

func (cur CreateUserRequest) extraValidate() error {
	a := emailaddress.Address(cur.Email)
	if err := a.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// CreateUserResponse is the response struct from creating a user.
type CreateUserResponse struct {
	UserID string `json:"user_id"`
}

// CreateUser creates a user.
func (c *Client) CreateUser(ctx context.Context, req CreateUserRequest) (*CreateUserResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	var resp CreateUserResponse
	if err := c.client.Post(ctx, paths.CreateUserPath, req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// LoginRequest defines the JSON request to the Login handler.
type LoginRequest struct {
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	SessionID uuid.UUID `json:"session_id"`
}

// LoginResponse is returned as JSON by Plex's login handlers to
// tell the login page where to redirect to on successful login.
type LoginResponse struct {
	RedirectTo string `json:"redirect_to"`
}

// UsernamePasswordLogin logs a user in with username + password.
func (c *Client) UsernamePasswordLogin(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	var resp LoginResponse
	if err := c.client.Post(ctx, paths.LoginPath, req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// ImpersonateUserRequest defines the JSON request to the ImpersonateUser handler.
// AccessToken is a valid access token issued by Plex for the user who is impersonating the target user.
// TargetUserID is the ID of the user to impersonate.
type ImpersonateUserRequest struct {
	AccessToken  string `json:"access_token"`
	TargetUserID string `json:"target_user_id"`
}

// ImpersonateUser returns Plex tokens for another user, enabling the user to impersonate the target user. There are
// configurable authz-based checks on if this impersonation is permitted, as well as the expiration of the
// returned tokens for the impersonated user.
func (c *Client) ImpersonateUser(ctx context.Context, accessToken string, targetUserID string) (*LoginResponse, error) {
	req := ImpersonateUserRequest{AccessToken: accessToken, TargetUserID: targetUserID}
	var resp LoginResponse
	if err := c.client.Post(ctx, paths.ImpersonateUserPath, req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// PasswordResetStartRequest defines the POST request to trigger sending a password reset email to a user.
// TODO: require a nonce and/or cooldown to limit DOS/brute force attacks.
type PasswordResetStartRequest struct {
	SessionID uuid.UUID `json:"session_id" validate:"notnil"`
	Email     string    `json:"email"`
}

//go:generate genvalidate PasswordResetStartRequest

func (prsr PasswordResetStartRequest) extraValidate() error {
	a := emailaddress.Address(prsr.Email)
	if err := a.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// PasswordResetSubmitRequest defines the POST request to finalize a password reset using the code emailed to a user.
// TODO: require a nonce and/or cooldown to limit DOS/brute force attacks.
type PasswordResetSubmitRequest struct {
	SessionID uuid.UUID `json:"session_id" validate:"notnil"`
	OTPCode   string    `json:"otp_code" validate:"notempty"`
	Password  string    `json:"password" validate:"notempty"`
}

//go:generate genvalidate PasswordResetSubmitRequest

// LoginAppRequest is based on https://github.com/golang/oauth2/pull/417/files, which
// implements the client side of https://datatracker.ietf.org/doc/html/rfc7591
type LoginAppRequest struct {
	// RedirectURIs specifies redirection URI strings for use in
	// redirect-based flows such as the "authorization code" and "implicit".
	RedirectURIs []string `json:"redirect_uris,omitempty"`

	// TokenEndpointAuthMethod specifies indicator of the requested authentication
	// method for the token endpoint
	// Possible values are:
	// "none": The client is a public client and does not have a client secret.
	// "client_secret_post": The client uses the HTTP POST parameters
	// "client_secret_basic": The client uses HTTP Basic
	// Additional values can be defined or absolute URIs can also be used
	// as values for this parameter without being registered.
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method,omitempty"`

	// GrantTypes specifies grant type strings that the client can use at the token endpoint
	// Possible values are:
	// "authorization_code": The authorization code grant type
	// "implicit": The implicit grant type
	// "password": The resource owner password credentials grant type
	// "client_credentials": The client credentials grant type
	// "refresh_token": The refresh token grant type
	// "urn:ietf:params:oauth:grant-type:jwt-bearer": The JWT Bearer Token Grant Type
	// "urn:ietf:params:oauth:grant-type:saml2-bearer": The SAML 2.0 Bearer Assertion Grant
	GrantTypes []string `json:"grant_types,omitempty"`

	// ResponseTypes specifies response type strings that the client can
	// use at the authorization endpoint.
	// Possible values are:
	// "code": The "authorization code" response
	// "token": The "implicit" response
	ResponseTypes []string `json:"response_types,omitempty"`

	// ClientName specifies Human-readable string name of the client
	// to be presented to the end-user during authorization
	ClientName string `json:"client_name,omitempty"`

	// ClientURI specifies URL of a web page providing information about the client.
	ClientURI string `json:"client_uri,omitempty"`

	// LogoURI specifies URL of a logo of the client
	LogoURI string `json:"logo_uri,omitempty"`

	// Scope specifies wire-level scopes representation
	Scope string `json:"scope,omitempty"`

	// Contacts specifies ways to contact people responsible for this client,
	// typically email addresses.
	Contacts []string `json:"contacts,omitempty"`

	// TermsOfServiceURI specifies URL of a human-readable terms of service
	// document for the client
	TermsOfServiceURI string `json:"tos_uri,omitempty"`

	// PolicyURI specifies URL of a human-readable privacy policy document
	PolicyURI string `json:"policy_uri,omitempty"`

	// JWKSURI specifies URL referencing the client's JWK Set [RFC7517] document,
	// which contains the client's public keys.
	JWKSURI string `json:"jwks_uri,omitempty"`

	// JWKS specifies the client's JWK Set [RFC7517] document, which contains
	// the client's public keys.  The value of this field MUST be a JSON
	// containing a valid JWK Set.
	JWKS string `json:"jwks,omitempty"`

	// SoftwareID specifies UUID assigned by the client developer or software publisher
	// used by registration endpoints to identify the client software.
	SoftwareID string `json:"software_id,omitempty"`

	// SoftwareVersion specifies version of the client software
	SoftwareVersion string `json:"software_version,omitempty"`

	// SoftwareStatement specifies client metadata values about the client software
	// as claims.  This is a string value containing the entire signed JWT.
	SoftwareStatement string `json:"software_statement,omitempty"`
}

// LoginAppResponse describes Client Information Response as specified in Section 3.2.1 of RFC 7591
type LoginAppResponse struct {
	// AppID specifies the UUID for this login app.
	AppID uuid.UUID `json:"app_id"`

	// OrganizationID specifies the UUID for the organization that this app belongs to.
	OrganizationID uuid.UUID `json:"organization_id"`

	// ClientID specifies client identifier string.
	ClientID string `json:"client_id"`

	// ClientSecret specifies client secret string.
	ClientSecret string `json:"client_secret"`

	// ClientIDIssuedAt specifies time at which the client identifier was issued.
	ClientIDIssuedAt time.Time `json:"client_id_issued_at"`

	// ClientSecretExpiresAt specifies time at which the client	secret will expire or 0 if it will not expire.
	ClientSecretExpiresAt time.Time `json:"client_secret_expires_at"`

	// Additionally, the authorization server MUST return all registered metadata about this client
	Metadata LoginAppRequest `json:",inline"`
}

// GetLoginApp gets a LoginApp by ID
func (c *Client) GetLoginApp(ctx context.Context, appID uuid.UUID) (*LoginAppResponse, error) {
	var resp LoginAppResponse
	if err := c.client.Get(ctx, paths.GetLoginAppPath(appID), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// ListLoginApps lists all LoginApps, optionally filtered by organization (if not empty)
func (c *Client) ListLoginApps(ctx context.Context, organizationID uuid.UUID) ([]LoginAppResponse, error) {
	var resp []LoginAppResponse

	if err := c.client.Get(ctx, paths.ListLoginAppPath(organizationID), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return resp, nil
}

// CreateLoginApp creates a new LoginApp
func (c *Client) CreateLoginApp(ctx context.Context, req *LoginAppRequest) (*LoginAppResponse, error) {
	var resp LoginAppResponse
	if err := c.client.Post(ctx, paths.CreateLoginAppPath, req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// UpdateLoginApp updates an existing LoginApp
func (c *Client) UpdateLoginApp(ctx context.Context, req *LoginAppRequest, appID uuid.UUID) (*LoginAppResponse, error) {
	var resp LoginAppResponse
	if err := c.client.Put(ctx, paths.UpdateLoginAppPath(appID), req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// DeleteLoginApp deletes an existing LoginApp
func (c *Client) DeleteLoginApp(ctx context.Context, appID uuid.UUID) error {
	if err := c.client.Delete(ctx, paths.DeleteLoginAppPath(appID), nil); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}
