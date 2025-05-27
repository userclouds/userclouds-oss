package idp

import (
	"context"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/paths"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/sdkclient"
	"userclouds.com/infra/ucerr"
)

// ManagementClient represents a management-oriented client to talk to the Userclouds IDP
// This client is not intended to be exported outside of Userclouds.
type ManagementClient struct {
	jsonClient *sdkclient.Client
	idpClient  *Client
}

// NewManagementClient constructs a new management IDP client
func NewManagementClient(url string, opts ...jsonclient.Option) (*ManagementClient, error) {
	jsonClient := sdkclient.New(url, "idp-management", opts...)
	if err := jsonClient.ValidateBearerTokenHeader(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	c := &ManagementClient{
		jsonClient: jsonClient,
		idpClient:  &Client{client: jsonClient},
	}
	return c, nil
}

// GetClient returns the underlying IDP client
func (c *ManagementClient) GetClient() *Client {
	return c.idpClient
}

// UpdateUsernamePasswordRequest is used to keep the follower IDP(s) in sync
type UpdateUsernamePasswordRequest struct {
	Username string `json:"username" validate:"notempty"`
	Password string `json:"password" validate:"notempty"`
}

//go:generate genvalidate UpdateUsernamePasswordRequest

// UpdateUsernamePassword updates the stored password for a user for follower-sync purposes
func (c *ManagementClient) UpdateUsernamePassword(ctx context.Context, username, password string) error {
	lr := UpdateUsernamePasswordRequest{
		Username: username,
		Password: password,
	}

	return ucerr.Wrap(c.jsonClient.Post(ctx, "/authn/upupdate", lr, nil))
}

// CreateUser creates a user without authn. Profile is optional (nil is ok)
func (c *ManagementClient) CreateUser(ctx context.Context, profile userstore.Record, opts ...Option) (uuid.UUID, error) {
	id, err := c.idpClient.CreateUser(ctx, profile, opts...)
	return id, ucerr.Wrap(err)
}

// ExecuteAccessor is a passthrough to idp.Client.ExecuteAccessor
func (c *ManagementClient) ExecuteAccessor(ctx context.Context, accessorID uuid.UUID, clientContext policy.ClientContext, selectorValues userstore.UserSelectorValues, opts ...Option) (*ExecuteAccessorResponse, error) {
	resp, err := c.idpClient.ExecuteAccessor(ctx, accessorID, clientContext, selectorValues, opts...)
	return resp, ucerr.Wrap(err)
}

// NewPasswordAuthn creates a new UserAuthn for username + password.
func NewPasswordAuthn(username, password string) UserAuthn {
	return UserAuthn{
		AuthnType: AuthnTypePassword,
		Username:  username,
		Password:  password,
	}
}

// CreateUserWithPassword creates a user on the IDP
func (c *ManagementClient) CreateUserWithPassword(ctx context.Context, username, password string, profile userstore.Record, opts ...Option) (uuid.UUID, error) {
	var options options
	for _, opt := range opts {
		opt.apply(&options)
	}

	req := CreateUserAndAuthnRequest{
		Profile:        profile,
		UserAuthn:      NewPasswordAuthn(username, password),
		ID:             options.userID,
		OrganizationID: options.organizationID,
	}

	var res UserResponse

	if err := c.jsonClient.Post(ctx, "/authn/users", req, &res); err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}

	return res.ID, nil
}

// NewOIDCAuthn creates a new UserAuthn for OIDC login.
func NewOIDCAuthn(provider oidc.ProviderType, issuerURL string, oidcSubject string) UserAuthn {
	return UserAuthn{
		AuthnType:     AuthnTypeOIDC,
		OIDCProvider:  provider,
		OIDCIssuerURL: issuerURL,
		OIDCSubject:   oidcSubject,
	}
}

// CreateUserWithOIDC creates a user on the IDP
func (c *ManagementClient) CreateUserWithOIDC(ctx context.Context, provider oidc.ProviderType, issuerURL string, subject string, profile userstore.Record, opts ...Option) (uuid.UUID, error) {

	var options options
	for _, opt := range opts {
		opt.apply(&options)
	}

	req := CreateUserAndAuthnRequest{
		Profile:        profile,
		UserAuthn:      NewOIDCAuthn(provider, issuerURL, subject),
		ID:             options.userID,
		OrganizationID: options.organizationID,
	}

	var res UserResponse

	if err := c.jsonClient.Post(ctx, "/authn/users", req, &res); err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}

	return res.ID, nil
}

// UserBaseProfileResponse is the response struct for GetUserBaseProfiles
type UserBaseProfileResponse struct {
	UserBaseProfile
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	UpdatedAt      int64  `json:"updated_at"` // seconds since the Unix Epoch (UTC)
}

// ListUserBaseProfilesResponse is the paginated response from listing user base profiles.
type ListUserBaseProfilesResponse struct {
	Data []UserBaseProfileResponse `json:"data"`
	pagination.ResponseFields
}

// GetUserBaseProfiles returns user base profiles for the given IDs
func (c *ManagementClient) GetUserBaseProfiles(ctx context.Context, ids []uuid.UUID, opts ...Option) ([]UserBaseProfileResponse, error) {
	stringIDs := make([]string, len(ids))
	for i, id := range ids {
		stringIDs[i] = id.String()
	}

	requestURL := url.URL{
		Path: "/authn/baseprofiles",
		RawQuery: url.Values{
			"user_ids": stringIDs,
		}.Encode(),
	}

	var res ListUserBaseProfilesResponse
	if err := c.jsonClient.Get(ctx, requestURL.String(), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return res.Data, nil
}

// ListUserBaseProfiles lists all user base profiles
func (c *ManagementClient) ListUserBaseProfiles(ctx context.Context, opts ...Option) (*ListUserBaseProfilesResponse, error) {

	var options options
	for _, opt := range opts {
		opt.apply(&options)
	}

	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var res ListUserBaseProfilesResponse
	query := pager.Query()
	if options.organizationID != uuid.Nil {
		query.Set("organization_id", options.organizationID.String())
	}
	requestURL := url.URL{
		Path:     "/authn/baseprofiles",
		RawQuery: query.Encode(),
	}

	if err := c.jsonClient.Get(ctx, requestURL.String(), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// UpdateUser is a passthrough to IDP UpdateUser
func (c *ManagementClient) UpdateUser(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*UserResponse, error) {
	resp, err := c.idpClient.UpdateUser(ctx, id, req)
	return resp, ucerr.Wrap(err)
}

// DeleteUser deletes a user by ID
func (c *ManagementClient) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return ucerr.Wrap(c.idpClient.DeleteUser(ctx, id))
}

// UserBaseProfileAndAuthnResponse is the response struct for GetUserBaseProfiles
type UserBaseProfileAndAuthnResponse struct {
	UserBaseProfile
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	UpdatedAt      int64  `json:"updated_at"` // seconds since the Unix Epoch (UTC)

	Authns      []UserAuthn      `json:"authns"`
	MFAChannels []UserMFAChannel `json:"mfa_channels"`
}

// GetUserBaseProfileAndAuthN gets a user base profile by ID
func (c *ManagementClient) GetUserBaseProfileAndAuthN(ctx context.Context, id uuid.UUID, opts ...Option) (*UserBaseProfileAndAuthnResponse, error) {
	requestURL := url.URL{
		Path: fmt.Sprintf("/authn/baseprofileswithauthn/%s", id),
	}

	var options options
	for _, opt := range opts {
		opt.apply(&options)
	}

	var res UserBaseProfileAndAuthnResponse
	if err := c.jsonClient.Get(ctx, requestURL.String(), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// ListUserBaseProfilesAndAuthNResponse is the paginated response from listing base user profiles and authn information.
type ListUserBaseProfilesAndAuthNResponse struct {
	Data []UserBaseProfileAndAuthnResponse `json:"data"`
}

// GetUserBaseProfileForOIDC gets a user by OIDC provider / issuer URL / ID
func (c *ManagementClient) GetUserBaseProfileForOIDC(ctx context.Context, provider oidc.ProviderType, issuerURL string, oidcSubject string) (*UserBaseProfileAndAuthnResponse, error) {

	prov, err := provider.MarshalText()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	reqURL := url.URL{
		Path: "/authn/baseprofileswithauthn",
		RawQuery: url.Values{
			"provider":   []string{string(prov)},
			"issuer_url": []string{issuerURL},
			"subject":    []string{oidcSubject},
		}.Encode(),
	}

	var res ListUserBaseProfilesAndAuthNResponse
	if err := c.jsonClient.Get(ctx, reqURL.String(), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if len(res.Data) != 1 {
		return nil, ucerr.Errorf("unexpected number of results (%d)", len(res.Data))
	}

	return &res.Data[0], nil
}

// ListUserBaseProfilesAndAuthNForEmail gets user base profiles associated with an email and authn type
func (c *ManagementClient) ListUserBaseProfilesAndAuthNForEmail(ctx context.Context, email string, authnType AuthnType) ([]UserBaseProfileAndAuthnResponse, error) {

	requestURL := url.URL{
		Path: "/authn/baseprofileswithauthn",
		RawQuery: url.Values{
			"email":      []string{email},
			"authn_type": []string{string(authnType)},
		}.Encode(),
	}

	var res ListUserBaseProfilesAndAuthNResponse
	if err := c.jsonClient.Get(ctx, requestURL.String(), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return res.Data, nil
}

// AddAuthnToUserRequest is the request struct to add a authn to a user
type AddAuthnToUserRequest struct {
	AuthnType     AuthnType         `json:"authn_type"`
	UserID        uuid.UUID         `json:"user_id" validate:"notnil"`
	Username      string            `json:"username"`
	Password      string            `json:"password"`
	OIDCProvider  oidc.ProviderType `json:"oidc_provider" validate:"skip"`
	OIDCIssuerURL string            `json:"oidc_issuer_url"`
	OIDCSubject   string            `json:"oidc_subject"`
}

func (r *AddAuthnToUserRequest) extraValidate() error {
	switch r.AuthnType {
	case AuthnTypePassword:
		if r.Username == "" || r.Password == "" {
			return ucerr.Errorf("Username and Password must be set for password auth: '%v'", r)
		}
	case AuthnTypeOIDC:
		if err := r.OIDCProvider.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
		if !r.OIDCProvider.IsSupported() {
			return ucerr.Errorf("OIDCProvider is unsupported: '%v'", r)
		}
		if err := r.OIDCProvider.ValidateIssuerURL(r.OIDCIssuerURL); err != nil {
			return ucerr.Wrap(err)
		}
		if r.OIDCSubject == "" {
			return ucerr.Errorf("OIDCSubject must be set for OIDC auth: '%v'", r)
		}
	default:
		return ucerr.Errorf("invalid AuthnType '%v'", r.AuthnType)
	}

	return nil
}

//go:generate genvalidate AddAuthnToUserRequest

// AddPasswordAuthnToUser adds username/password authentication to an existing user
func (c *ManagementClient) AddPasswordAuthnToUser(ctx context.Context, userID string, username string, password string) error {
	req := AddAuthnToUserRequest{
		AuthnType: AuthnTypePassword,
		UserID:    uuid.Must(uuid.FromString(userID)),
		Username:  username,
		Password:  password,
	}

	return ucerr.Wrap(c.jsonClient.Post(ctx, paths.AddAuthnToUser, req, nil))
}

// AddOIDCAuthnToUser adds an oidc provider as an authenticator to an existing user
func (c *ManagementClient) AddOIDCAuthnToUser(ctx context.Context, userID string, provider oidc.ProviderType, issuerURL string, oidcSubject string) error {
	req := AddAuthnToUserRequest{
		AuthnType:     AuthnTypeOIDC,
		UserID:        uuid.Must(uuid.FromString(userID)),
		OIDCProvider:  provider,
		OIDCIssuerURL: issuerURL,
		OIDCSubject:   oidcSubject,
	}

	return ucerr.Wrap(c.jsonClient.Post(ctx, paths.AddAuthnToUser, req, nil))
}
