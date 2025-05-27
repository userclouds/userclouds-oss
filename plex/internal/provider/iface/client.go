package iface

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"

	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/tenantplex"
)

// ErrUserNotFound is used when a GetUser* command fails to find a user, and is distinct
var ErrUserNotFound error = ucerr.New("user not found")

// ClassifyGetUserError distinguishes between the user not being found and a legitimate error
func ClassifyGetUserError(err error) error {
	var jsonClientErr jsonclient.Error
	if errors.As(err, &jsonClientErr) && jsonClientErr.StatusCode == http.StatusNotFound {
		return ucerr.Wrap(ErrUserNotFound)
	}
	return ucerr.Wrap(err)
}

// LoginResponseWithClaims returns claims about a user on successful login OR
// indicates that an MFA challenge/response is required.
type LoginResponseWithClaims struct {
	// Status is the underlying IDP login status
	Status idp.LoginStatus

	// Claims are included only if MFA required is false
	Claims jwt.MapClaims

	// MFAToken returned from IDP when additional factor is required;
	// treat as a cryptographic secret as it substitutes for username/password
	// on subsequent response.
	MFAToken string

	// MFAProvider is the UUID of the Provider which issued the token, so
	// we know how to route the request properly on subsequent response.
	MFAProvider uuid.UUID

	// SupportedMFAChannels is the set of MFA channel type and channel ID pairs supported
	// by the user. If there is more than one option, the user will be prompted to
	// select a channel type for the MFA challenge.
	SupportedMFAChannels oidc.MFAChannels

	// EvaluateSupportedMFAChannels is true if the user should be prompted to
	// update their MFA settings after a successful login.
	EvaluateSupportedMFAChannels bool

	// NewRecoveryCode is non-empty if the user has had a new active recovery code
	// generated that they should be presented with at the end of their login flow.
	NewRecoveryCode string
}

// MFAGetChannelsResponse is the response to an MFAGetChannels request
type MFAGetChannelsResponse struct {
	// MFAToken identifies the MFARequest in IDP
	MFAToken string

	// MFAProvider is the UUID of the Provider which issued the token, so
	// we know how to route the request properly on subsequent response.
	MFAProvider uuid.UUID

	// SupportedMFAChannels is the set of MFA channel type and channel ID pairs supported
	// by the user. If there is more than one option, the user will be prompted to
	// select a channel type for the MFA challenge.
	SupportedMFAChannels oidc.MFAChannels
}

// UserProfile is an alias for the IDP UserBaseProfile type, plus authn fields
type UserProfile idp.UserBaseProfileAndAuthnResponse

// GetFriendlyName returns an appropriate user-facing description of a user based on what fields are available
func (up UserProfile) GetFriendlyName() string {
	if up.Name != "" {
		return up.Name
	}

	if up.Email != "" {
		return up.Email
	}

	return up.ID
}

// ToUserstoreRecord converts a UserProfile to a userstore.Record
func (up UserProfile) ToUserstoreRecord() userstore.Record {
	return userstore.Record{
		"name":           up.Name,
		"nickname":       up.Nickname,
		"email":          up.Email,
		"email_verified": up.EmailVerified,
		"picture":        up.Picture,
	}
}

// NewUserProfileFromClaims creates a UserProfile from a set of token claims
func NewUserProfileFromClaims(claims oidc.UCTokenClaims) *UserProfile {
	return &UserProfile{
		ID:        claims.Subject,
		UpdatedAt: claims.UpdatedAt,
		UserBaseProfile: idp.UserBaseProfile{
			Email:         claims.Email,
			EmailVerified: claims.EmailVerified,
			Name:          claims.Name,
			Nickname:      claims.Nickname,
			Picture:       claims.Picture,
		},
		OrganizationID: claims.OrganizationID,
	}
}

// Client handles communication to a backend IDP for auth operations and
// abstracts differences between auth providers.
type Client interface {
	// UsernamePasswordLogin provides username & password credentials to the IDP and returns a token.
	UsernamePasswordLogin(ctx context.Context, username, password string) (*LoginResponseWithClaims, error) // authn

	// LoginURL returns a redirectable URL for logging in, whether via Plex or the provider itself
	LoginURL(context.Context, uuid.UUID, *tenantplex.App) (*url.URL, error) // authn

	// Logout performs side-effects in the IDP required to log out and returns
	// a URL that Plex should redirect the user agent to.
	// NOTE: This makes sense for Web apps, but for Mobile apps it is not needed
	// unless the app uses a browser-based login flow.
	Logout(ctx context.Context, redirectURL string) (string, error) // authn

	// SupportsMFAConfiguration returns true if the client supports creating or deleting an MFA channel, or selecting
	// a primary MFA channel, as well as whether reissuing recovery codes is supported.
	SupportsMFAConfiguration() bool

	// MFAChallenge issues an MFA challenge on the specified channel, returning a potentially augmented channel.
	MFAChallenge(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*oidc.MFAChannel, error) // authn

	// MFAClearPrimaryChannel will clear the primary channel, returning the updated set of supported channels if successful
	MFAClearPrimaryChannel(ctx context.Context, mfaToken string) (*oidc.MFAChannels, error) // authn

	// MFACreateChannel creates and returns the specified channel, issuing a challenge for that channel if creation is successful
	MFACreateChannel(ctx context.Context, mfaToken string, channelType oidc.MFAChannelType, channelTypeID string) (*oidc.MFAChannel, error) // authn

	// MFADeleteChannel deletes the specified channel, returning the updated set of supported channels if successful
	MFADeleteChannel(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*oidc.MFAChannels, error) // authn

	// MFAGetChannels retrieves all supported channels for a user, along with an MFA token for interacting with those channels
	MFAGetChannels(ctx context.Context, userID string) (*MFAGetChannelsResponse, error) // authn

	// MFALogin responds to a challenged login attempt with a challenge code for the specified channel to complete the login flow.
	MFALogin(ctx context.Context, mfaToken string, challengeCode string, channel oidc.MFAChannel) (*LoginResponseWithClaims, error) // authn

	// MFAMakePrimaryChannel will make the specified channel the primary channel, returning the updated set of supported channels if successful
	MFAMakePrimaryChannel(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*oidc.MFAChannels, error) // authn

	// MFAReissueRecoveryCode will issue a new recovery code for a user, returning the updated recovery code channel if successful
	MFAReissueRecoveryCode(ctx context.Context, mfaToken string, channel oidc.MFAChannel) (*oidc.MFAChannel, error)

	// String implements Stringer and ensures all clients are printable
	String() string
}

// ManagementClient defines a client used to operate across a provider, vs on a specific app
// Used for configuring that provider, and/or managing the underlying user objects etc
type ManagementClient interface {
	// CreateUserWithPassword creates a new user account with username + password auth.
	CreateUserWithPassword(ctx context.Context, username, password string, profile UserProfile) (string, error) // authn

	// CreateUserWithOIDC creates a new user account with an OIDC provider as the auth mechanism.
	CreateUserWithOIDC(ctx context.Context, provider oidc.ProviderType, issuerURL string, oidcSubject string, profile UserProfile) (string, error) // authn

	// GetUser looks up a user by IDP-specific user ID and returns user/profile info.
	GetUser(ctx context.Context, userID string) (*UserProfile, error) // userstore

	// GetUserForOIDC looks up a user by oidc provider + issuer URL + oidc subject
	// TODO (sgarrity 2/24): the addition of email here is a bit of a hack to support Cognito + O365 OIDC, usually ignored
	GetUserForOIDC(ctx context.Context, provider oidc.ProviderType, issuerURL string, oidcSubject string, email string) (*UserProfile, error) // userstore

	// ListUsers returns a full list of users (where available)
	ListUsers(ctx context.Context) ([]UserProfile, error)

	// ListUsersUpdatedDuring returns a partial list of users based on created/updated time
	ListUsersUpdatedDuring(ctx context.Context, since time.Time, until time.Time) ([]UserProfile, error) // userstore

	// ListUsersForEmail looks up a user by email and authentication type and returns user/profile info.
	// Email is unique for some authentication type (username+password) but there can be > 1 result for others (social).
	ListUsersForEmail(ctx context.Context, email string, authnType idp.AuthnType) ([]UserProfile, error) // userstore

	// SetEmailVerified marks the email of a user as verified (or not).
	// NOTE: it takes the user ID as a the key, not email, because underlying
	// IDPs may have multiple accounts for a given email using different auth methods and we need to be specific.
	SetEmailVerified(ctx context.Context, userID string, verified bool) error // userstore

	// UpdateUsernamePassword updates or creates credentials in the IDP.
	UpdateUsernamePassword(ctx context.Context, username, password string) error // authn

	// UpdateUser updates a user's profile
	UpdateUser(ctx context.Context, userID string, profile UserProfile) error // userstore

	// AddPasswordAuthnToUser adds username/password authentication to an existing user
	AddPasswordAuthnToUser(ctx context.Context, userID string, username string, password string) error // authn

	// AddOIDCAuthnToUser adds a social provider as an authenticator to an existing user
	AddOIDCAuthnToUser(ctx context.Context, userID string, provider oidc.ProviderType, issuerURL string, oidcSubject string) error // authn

	// String implements Stringer and ensures all clients are printable
	String() string
}
