package idp

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
)

// UsernamePasswordLoginRequest specifies the IDP request to login with username & password.
type UsernamePasswordLoginRequest struct {
	Username string `json:"username" validate:"notempty"`
	Password string `json:"password" validate:"notempty"`
	ClientID string `json:"client_id"`
}

//go:generate genvalidate UsernamePasswordLoginRequest

// LoginStatus indicates whether a login attempt succeeded or requires additional validation (e.g. MFA)
type LoginStatus string

// LoginStatus constants
const (
	LoginStatusSuccess        LoginStatus = "success"
	LoginStatusMFARequired    LoginStatus = "mfa_required"
	LoginStatusMFACodeInvalid LoginStatus = "mfa_code_invalid"
	LoginStatusMFACodeExpired LoginStatus = "mfa_code_expired"
)

// LoginResponse is the full response returned from an IDP login API
type LoginResponse struct {
	Status LoginStatus `json:"status"`

	UserID uuid.UUID `json:"user_id"`

	// MFAToken is included if the user requires MFA, either because it is required
	// for the tenant or because the user has a verified supported MFA channel
	MFAToken string `json:"mfa_token,omitempty"`

	// SupportedMFAChannels represents the supported MFA channel types and verified
	// channels for the user
	SupportedMFAChannels oidc.MFAChannels `json:"supported_mfa_channels"`

	// EvaluateSupportedMFAChannels indicates whether the user should be prompted to
	// reevaluate MFA settings
	EvaluateSupportedMFAChannels bool `json:"evaluate_supported_mfa_channels,omitempty"`

	// NewRecoveryCode is included for a successful login if the user did requires one
	NewRecoveryCode string `json:"new_recovery_code,omitempty"`
}

// TODO: Add validation tags to MFA Requests structs

// MFAChannelRequest allows the client to submit an MFA request for an MFA token and the specified channel
type MFAChannelRequest struct {
	MFAToken   uuid.UUID
	MFAChannel oidc.MFAChannel
}

//go:generate genvalidate MFAChannelRequest

// MFAClearPrimaryChannelRequest allows the client to submit a request to clear the primary channel for a client ID and MFA token
type MFAClearPrimaryChannelRequest struct {
	ClientID string
	MFAToken uuid.UUID
}

//go:generate genvalidate MFAClearPrimaryChannelRequest

// MFACreateChannelRequest allows the client to submit an MFA request for an MFA token to create a new channel
type MFACreateChannelRequest struct {
	MFAToken      uuid.UUID
	ChannelType   oidc.MFAChannelType
	ChannelTypeID string
}

//go:generate genvalidate MFACreateChannelRequest

// MFACodeResponse represents the response for GetMFACode and MFACreateChannel
type MFACodeResponse struct {
	MFAToken   uuid.UUID
	MFAChannel oidc.MFAChannel
	MFACode    string
}

// MFAGetChannelsRequest allows the client to submit a request to retrieve all supported MFA channels for a user and client ID
type MFAGetChannelsRequest struct {
	UserID   uuid.UUID
	ClientID string
}

//go:generate genvalidate MFAGetChannelsRequest

// MFAGetChannelsResponse represents the response to a request for the supported channels for a user and client ID
type MFAGetChannelsResponse struct {
	MFAToken    uuid.UUID
	MFAChannels oidc.MFAChannels
}

// MFALoginRequest allows the client to submit an MFA code
type MFALoginRequest struct {
	MFAToken uuid.UUID
	MFACode  string
}

//go:generate genvalidate MFALoginRequest

// MFAReissueRecoveryCodeResponse represents the response for a recovery code reissue request
type MFAReissueRecoveryCodeResponse struct {
	MFAToken   uuid.UUID
	MFAChannel oidc.MFAChannel
}

// AuthClient represents an auth-oriented client to talk to the Userclouds IDP
// This client is not intended to be exported outside of Userclouds
type AuthClient struct {
	client   *jsonclient.Client
	clientID string
}

// NewAuthClient constructs a new auth IDP client
func NewAuthClient(url string, clientID string, opts ...jsonclient.Option) (*AuthClient, error) {
	c := &AuthClient{
		client:   jsonclient.New(strings.TrimSuffix(url, "/"), opts...),
		clientID: clientID,
	}
	if err := c.client.ValidateBearerTokenHeader(); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return c, nil
}

// Login supports username/password login to the UC IDP
func (c *AuthClient) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	lr := UsernamePasswordLoginRequest{
		Username: username,
		Password: password,
		ClientID: c.clientID,
	}
	var response LoginResponse

	if err := c.client.Post(ctx, "/authn/uplogin", lr, &response, jsonclient.ParseOAuthError()); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &response, nil
}

// GetMFACode gets a new MFA code for the specified mfa token and channel from the UC IDP
func (c *AuthClient) GetMFACode(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*MFACodeResponse, error) {
	mfaTokenID, err := uuid.FromString(mfaToken)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	request := MFAChannelRequest{mfaTokenID, channel}
	var response MFACodeResponse
	if err := c.client.Post(ctx, "/authn/mfacode", request, &response); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &response, nil
}

// GetMFAChannels gets the supported MFA channels for a user and client ID
func (c *AuthClient) GetMFAChannels(ctx context.Context, userID uuid.UUID) (*MFAGetChannelsResponse, error) {
	request := MFAGetChannelsRequest{userID, c.clientID}
	var response MFAGetChannelsResponse
	if err := c.client.Post(ctx, "/authn/mfagetchannels", request, &response); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &response, nil
}

// LoginWithMFA sends the MFA code response
func (c *AuthClient) LoginWithMFA(ctx context.Context, mfaToken string, code string) (*LoginResponse, error) {
	mfaTokenID, err := uuid.FromString(mfaToken)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	request := MFALoginRequest{mfaTokenID, code}
	var response LoginResponse
	if err := c.client.Post(ctx, "/authn/mfaresponse", request, &response, jsonclient.ParseOAuthError()); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &response, nil
}

// MFAClearPrimaryChannel attempts to clear the primary MFA channel for the user, disabling MFA
func (c *AuthClient) MFAClearPrimaryChannel(ctx context.Context, mfaToken string) (*MFAGetChannelsResponse, error) {
	mfaTokenID, err := uuid.FromString(mfaToken)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	request := MFAClearPrimaryChannelRequest{c.clientID, mfaTokenID}
	var response MFAGetChannelsResponse
	if err := c.client.Post(ctx, "/authn/mfaclearprimarychannel", request, &response); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &response, nil
}

// MFACreateChannel attempts to create an MFA channel with the specified channel type and channel type id
func (c *AuthClient) MFACreateChannel(ctx context.Context, mfaToken string, channelType oidc.MFAChannelType, channelTypeID string) (*MFACodeResponse, error) {
	mfaTokenID, err := uuid.FromString(mfaToken)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	request := MFACreateChannelRequest{mfaTokenID, channelType, channelTypeID}
	var response MFACodeResponse
	if err := c.client.Post(ctx, "/authn/mfacreatechannel", request, &response); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &response, nil
}

// MFADeleteChannel attempts to delete the specified MFA channel
func (c *AuthClient) MFADeleteChannel(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*MFAGetChannelsResponse, error) {
	mfaTokenID, err := uuid.FromString(mfaToken)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	request := MFAChannelRequest{mfaTokenID, channel}
	var response MFAGetChannelsResponse
	if err := c.client.Post(ctx, "/authn/mfadeletechannel", request, &response); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &response, nil
}

// MFAMakePrimaryChannel attempts to make the specified MFA channel the primary channel for the user
func (c *AuthClient) MFAMakePrimaryChannel(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*MFAGetChannelsResponse, error) {
	mfaTokenID, err := uuid.FromString(mfaToken)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	request := MFAChannelRequest{mfaTokenID, channel}
	var response MFAGetChannelsResponse
	if err := c.client.Post(ctx, "/authn/mfamakeprimarychannel", request, &response); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &response, nil
}

// MFAReissueRecoveryCode attempts to issue a new recovery code for the user
func (c *AuthClient) MFAReissueRecoveryCode(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*MFAReissueRecoveryCodeResponse, error) {
	mfaTokenID, err := uuid.FromString(mfaToken)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	request := MFAChannelRequest{mfaTokenID, channel}
	var response MFAReissueRecoveryCodeResponse
	if err := c.client.Post(ctx, "/authn/mfareissuerecoverycode", request, &response); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &response, nil
}
