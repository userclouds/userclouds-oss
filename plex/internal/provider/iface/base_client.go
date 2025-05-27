package iface

import (
	"context"
	"net/url"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/tenantplex"
)

// BaseClient should be embedded within any concrete Client implementations, which can choose which interface methods they support
type BaseClient struct {
}

// UsernamePasswordLogin is part of the Client interface
func (BaseClient) UsernamePasswordLogin(ctx context.Context, username, password string) (*LoginResponseWithClaims, error) {
	return nil, ucerr.New("method 'UsernamePasswordLogin' not supported by client")
}

// LoginURL is part of the Client interface
func (BaseClient) LoginURL(context.Context, uuid.UUID, *tenantplex.App) (*url.URL, error) {
	return nil, ucerr.New("method 'LoginURL' not supported by client")
}

// Logout is part of the Client interface
func (BaseClient) Logout(ctx context.Context, redirectURL string) (string, error) {
	return "", ucerr.New("method 'Logout' not supported by client")
}

// SupportsMFAConfiguration is part of the Client interface
func (BaseClient) SupportsMFAConfiguration() bool {
	return false
}

// MFAChallenge is part of the Client interface
func (BaseClient) MFAChallenge(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*oidc.MFAChannel, error) {
	return nil, ucerr.New("method 'MFAChallenge' not supported by client")
}

// MFAClearPrimaryChannel is part of the Client interface
func (BaseClient) MFAClearPrimaryChannel(ctx context.Context, mfaToken string) (*oidc.MFAChannels, error) {
	return nil, ucerr.New("method 'MFAClearPrimaryChannel' not supported by client")
}

// MFACreateChannel is part of the Client interface
func (BaseClient) MFACreateChannel(ctx context.Context, mfaToken string, channelType oidc.MFAChannelType, channelTypeID string) (*oidc.MFAChannel, error) {
	return nil, ucerr.New("method 'MFACreateChannel' not supported by client")
}

// MFADeleteChannel is part of the Client interface
func (BaseClient) MFADeleteChannel(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*oidc.MFAChannels, error) {
	return nil, ucerr.New("method 'MFADeleteChannel' not supported by client")
}

// MFAGetChannels is part of the Client interface
func (BaseClient) MFAGetChannels(ctx context.Context, userID string) (*MFAGetChannelsResponse, error) {
	return nil, ucerr.New("method 'MFAGetChannels' not supported by client")
}

// MFALogin is part of the Client interface
func (BaseClient) MFALogin(ctx context.Context, mfaToken string, challengeCode string, channel oidc.MFAChannel) (*LoginResponseWithClaims, error) {
	return nil, ucerr.New("method 'MFALogin' not supported by client")
}

// MFAMakePrimaryChannel is part of the Client interface
func (BaseClient) MFAMakePrimaryChannel(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*oidc.MFAChannels, error) {
	return nil, ucerr.New("method 'MFAMakePrimaryChannel' not supported by client")
}

// MFAReissueRecoveryCode is part of the Client interface
func (BaseClient) MFAReissueRecoveryCode(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*oidc.MFAChannel, error) {
	return nil, ucerr.New("method 'MFAReissueRecoveryCode' not supported by client")
}

// BaseManagementClient should be embedded within any concrete ManagementClient implementations, which can choose which interface methods they support
type BaseManagementClient struct {
}

// CreateUserWithPassword is part of the ManagementClient interface
func (BaseManagementClient) CreateUserWithPassword(ctx context.Context, username, password string, profile UserProfile) (string, error) {
	return "", ucerr.New("method 'CreateUserWithPassword' not supported by client")
}

// CreateUserWithOIDC is part of the ManagementClient interface
func (BaseManagementClient) CreateUserWithOIDC(ctx context.Context, provider oidc.ProviderType, issuerURL string, oidcSubject string, profile UserProfile) (string, error) {
	return "", ucerr.New("method 'CreateUserWithOIDC' not supported by client")
}

// GetUser is part of the ManagementClient interface
func (BaseManagementClient) GetUser(ctx context.Context, userID string) (*UserProfile, error) {
	return nil, ucerr.New("method 'GetUser' not supported by client")
}

// GetUserForOIDC is part of the ManagementClient interface
func (BaseManagementClient) GetUserForOIDC(ctx context.Context, provider oidc.ProviderType, issuerURL string, oidcSubject string, email string) (*UserProfile, error) {
	return nil, ucerr.New("method 'GetUserForOIDC' not supported by client")
}

// ListUsersUpdatedDuring is part of the ManagementClient interface
func (BaseManagementClient) ListUsersUpdatedDuring(ctx context.Context, since time.Time, until time.Time) ([]UserProfile, error) {
	return nil, ucerr.New("method 'ListUsersUpdatedDuring' not supported by client")
}

// ListUsersForEmail is part of the ManagementClient interface
func (BaseManagementClient) ListUsersForEmail(ctx context.Context, email string, authnType idp.AuthnType) ([]UserProfile, error) {
	return nil, ucerr.New("method 'ListUsersForEmail' not supported by client")
}

// SetEmailVerified is part of the ManagementClient interface
func (BaseManagementClient) SetEmailVerified(ctx context.Context, userID string, verified bool) error {
	return ucerr.New("method 'SetEmailVerified' not supported by client")
}

// UpdateUsernamePassword is part of the ManagementClient interface
func (BaseManagementClient) UpdateUsernamePassword(ctx context.Context, username, password string) error {
	return ucerr.New("method 'UpdateUsernamePassword' not supported by client")
}

// UpdateUser is part of the ManagementClient interface
func (BaseManagementClient) UpdateUser(ctx context.Context, userID string, profile UserProfile) error {
	return ucerr.New("method 'UpdateUser' not supported by client")
}

// AddPasswordAuthnToUser is part of the ManagementClient interface
func (BaseManagementClient) AddPasswordAuthnToUser(ctx context.Context, userID string, username string, password string) error {
	return ucerr.New("method 'AddPasswordAuthnToUser' not supported by client")
}

// AddOIDCAuthnToUser is part of the ManagementClient interface
func (BaseManagementClient) AddOIDCAuthnToUser(ctx context.Context, userID string, provider oidc.ProviderType, issuerURL string, oidcSubject string) error {
	return ucerr.New("method 'AddOIDCAuthnToUser' not supported by client")
}

// ListUsers is part of the ManagementClient interface
func (BaseManagementClient) ListUsers(ctx context.Context) ([]UserProfile, error) {
	return nil, ucerr.New("method 'ListUsers' not supported by client")
}
