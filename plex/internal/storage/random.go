package storage

// Storage interface for the Login Multiplexer

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// TODO (sgarrity 7/23): refactor this file to be more readable

// ResponseType represents an OAuth/OIDC response type
type ResponseType string

// ResponseTypes is a collection of ResponseType objects
type ResponseTypes []ResponseType

// Supported OIDC Response Types
const (
	AuthorizationCodeResponseType ResponseType = "code"
	TokenResponseType             ResponseType = "token"
	IDTokenResponseType           ResponseType = "id_token"
	UnusedResponseType            ResponseType = "unused"
)

// AllowedResponseTypes is the set of all valid `response_type` parameters to the authorization endpoint
var AllowedResponseTypes = ResponseTypes{AuthorizationCodeResponseType, TokenResponseType, IDTokenResponseType}

// NewResponseTypes creates a ResponseTypes object from a space-delimited string containing response types.
func NewResponseTypes(spaceDelimStr string) (ResponseTypes, error) {
	// Multiple response types can be specified (see https://openid.net/specs/oauth-v2-multiple-response-types-1_0.html),
	// which is known as is known as the "hybrid flow" (as opposed to the "authorization code" or "implict" flow which
	// have a response_type of "code" and "token" respectively).
	// https://darutk.medium.com/diagrams-of-all-the-openid-connect-flows-6968e3990660 has a good explanation as well.
	responseTypeStrs := oidc.SplitTokens(spaceDelimStr)
	responseTypes := ResponseTypes(make([]ResponseType, 0, len(responseTypeStrs)))
	for _, v := range responseTypeStrs {
		rt := ResponseType(v)
		if !AllowedResponseTypes.Contains(rt) {
			return nil, ucerr.NewUnsupportedResponseError(spaceDelimStr)
		}
		if responseTypes.Contains(rt) {
			// silently ignore duplicate
			continue
		}
		responseTypes = append(responseTypes, rt)
	}

	return responseTypes, nil
}

// Contains returns true if a given response type is in the set of response types, false otherwise
func (rts ResponseTypes) Contains(rt ResponseType) bool {
	return slices.Contains(rts, rt)
}

// String implements Stringer by returning a space delimited string of response types
func (rts ResponseTypes) String() string {
	rtStr := ""
	// strings.Join would work better except Go doesn't let us typecast to []string :(
	for i, rt := range rts {
		rtStr = rtStr + string(rt)
		if i < len(rts)-1 {
			rtStr = rtStr + " "
		}
	}
	return rtStr
}

// OTPPurpose indicates why an OTP is generated/validated in a login session.
type OTPPurpose int

//go:generate genconstant OTPPurpose

const (
	// OTPPurposeInvalid implies the value was never initialized
	OTPPurposeInvalid OTPPurpose = 0

	// OTPPurposeLogin means the OTP was issued for passwordless login.
	OTPPurposeLogin OTPPurpose = 1

	// OTPPurposeAccountVerify means the OTP was issued to verify the email address on a
	// newly created account.
	OTPPurposeAccountVerify OTPPurpose = 2

	// OTPPurposeInvite is used for inviting new users to a tenant.
	OTPPurposeInvite OTPPurpose = 3

	// OTPPurposeResetPassword is used for resetting a user's password.
	OTPPurposeResetPassword OTPPurpose = 4
)

// AddAuthnProviderData is the temporary data used to store information about an authentication provider
// that might be added to the user's account upon successful login.
type AddAuthnProviderData struct {
	NewProviderAuthnType     idp.AuthnType     `json:"new_authn_type,omitempty"`
	NewOIDCProvider          oidc.ProviderType `json:"new_social_provider,omitempty"`
	NewProviderEmail         string            `json:"new_provider_email,omitempty"`
	NewProviderPassword      string            `json:"new_provider_password,omitempty"`
	NewProviderOIDCIssuerURL string            `json:"new_provider_oidc_issuer_url,omitempty"`
	NewProviderOIDCSubject   string            `json:"new_provider_oidc_subject,omitempty"`
	PermissionToAdd          bool              `json:"permission_to_add,omitempty"`
	DoNotAdd                 bool              `json:"do_not_add,omitempty"`
}

func (o *AddAuthnProviderData) extraValidate() error {
	if o.NewProviderAuthnType == idp.AuthnTypeOIDC {
		if !o.NewOIDCProvider.IsSupported() {
			return ucerr.Errorf("NewOIDCProvider '%v' is not a supported provider for OIDC authn", o.NewOIDCProvider)
		}
		if err := o.NewOIDCProvider.ValidateIssuerURL(o.NewProviderOIDCIssuerURL); err != nil {
			return ucerr.Wrap(err)
		}
	} else if o.NewProviderAuthnType == idp.AuthnTypePassword {
		if o.NewProviderPassword == "" {
			return ucerr.New("NewProviderPassword is required for password authn")
		}
	}
	return nil
}

//go:generate gendbjson AddAuthnProviderData

//go:generate genvalidate AddAuthnProviderData

// MFAChannelState captures state about an MFA channel within a given login session
type MFAChannelState struct {
	Failures []time.Time `json:"failures"`
}

// MFAChannelStates captures state about all MFA channels within a given login session
type MFAChannelStates struct {
	ChannelStates map[string]MFAChannelState `json:"channel_states"`
}

func (MFAChannelStates) getChannelKey(channel oidc.MFAChannel) string {
	return fmt.Sprintf("%v%s", channel.ChannelType, channel.ChannelTypeID)
}

// IsRestricted checks whether the MFA channel is restricted, returning a flag
// and an expiration time for the restriction.
func (mfacs MFAChannelStates) IsRestricted(channel oidc.MFAChannel) (isRestricted bool, expiration time.Time) {
	channelState, found := mfacs.ChannelStates[mfacs.getChannelKey(channel)]
	if !found || len(channelState.Failures) < mfaMaxFailures {
		return false, expiration
	}

	expiration = channelState.Failures[len(channelState.Failures)-1].Add(mfaRetryTimeout)
	return expiration.After(time.Now().UTC()), expiration
}

// RecordFailure records a failed challenge for an MFA channel
func (mfacs *MFAChannelStates) RecordFailure(channel oidc.MFAChannel) {
	key := mfacs.getChannelKey(channel)
	channelState, found := mfacs.ChannelStates[key]
	if !found {
		channelState = MFAChannelState{Failures: []time.Time{}}
	}

	currentTime := time.Now().UTC()

	failures := []time.Time{}
	for _, failure := range channelState.Failures {
		failureWindow := failure.Add(mfaFailureWindow)
		if currentTime.Before(failureWindow) {
			failures = append(failures, failure)
		}
	}
	failures = append(failures, currentTime)

	channelState.Failures = failures
	mfacs.ChannelStates[key] = channelState
}

// Reset resets the state for an MFA channel
func (mfacs *MFAChannelStates) Reset(channel oidc.MFAChannel) {
	delete(mfacs.ChannelStates, mfacs.getChannelKey(channel))
}

//go:generate gendbjson MFAChannelStates

// MFAPurpose indicates why we have established an MFA session
type MFAPurpose int

const (
	// MFAPurposeInvalid implies the value was never initialized
	MFAPurposeInvalid MFAPurpose = 0

	// MFAPurposeLogin means the MFA session is for a normal login flow.
	MFAPurposeLogin MFAPurpose = 1

	// MFAPurposeLoginSetup means the MFA session is part of initial MFA
	// setup during a login, when MFA is required but there is no verified
	// primary supported MFA channel.
	MFAPurposeLoginSetup MFAPurpose = 2

	// MFAPurposeConfigure means the MFA session is part of MFA setup
	// after login has already occurred, initiated by a user navigating
	// to their MFA settings page.
	MFAPurposeConfigure MFAPurpose = 3
)

// CanChallenge returns true if a challenge can be issued for the current MFA session.
func (p MFAPurpose) CanChallenge() bool {
	return p == MFAPurposeLogin
}

// CanModify returns true if the current MFA session allows the creation,
// deletion, or modification of primary status of MFA channels.
func (p MFAPurpose) CanModify() bool {
	return p == MFAPurposeLoginSetup || p == MFAPurposeConfigure
}

// IsConfiguration returns true if the current MFA session is a configuration session.
func (p MFAPurpose) IsConfiguration() bool {
	return p == MFAPurposeConfigure
}

// ShouldMask returns true if MFA channel names must be masked during
// the current MFA session.
func (p MFAPurpose) ShouldMask() bool {
	return p != MFAPurposeLoginSetup && p != MFAPurposeConfigure
}

//go:generate genconstant MFAPurpose

// MFAChallengeState indicates the state of the current challenge in an MFA session
type MFAChallengeState int

const (
	// MFAChallengeStateNoChallenge means there is no current challenge
	MFAChallengeStateNoChallenge MFAChallengeState = 0

	// MFAChallengeStateIssued means a challenge has been issued but not
	// yet responded to
	MFAChallengeStateIssued MFAChallengeState = 1

	// MFAChallengeStateBadChallenge means the last challenge attempt was incorrect
	MFAChallengeStateBadChallenge MFAChallengeState = 2

	// MFAChallengeStateExpired means the challenge code has expired
	MFAChallengeStateExpired MFAChallengeState = 3
)

//go:generate genconstant MFAChallengeState

// HasBeenIssued returns true if a challenge has been issued
func (mfacs MFAChallengeState) HasBeenIssued() bool {
	switch mfacs {
	case MFAChallengeStateIssued, MFAChallengeStateBadChallenge, MFAChallengeStateExpired:
		return true
	default:
		return false
	}
}

// IsFirstChallenge returns true if this was the first challenge
func (mfacs MFAChallengeState) IsFirstChallenge() bool {
	return mfacs == MFAChallengeStateIssued
}

// SessionOption represents an optional parameter for creating an OIDC Login Session.
type SessionOption interface {
	apply(*OIDCLoginSession) error
}

type oidcOptFunc func(*OIDCLoginSession) error

func (o oidcOptFunc) apply(s *OIDCLoginSession) error {
	return ucerr.Wrap(o(s))
}

// ApplyOption applies an option to a session.
func (s *OIDCLoginSession) ApplyOption(opt SessionOption) error {
	return ucerr.Wrap(opt.apply(s))
}

// CodeChallenge is specified when creating an OIDC Login Session if the Auth Code with PKCE flow is used.
func CodeChallenge(ctx context.Context, s *Storage, method crypto.CodeChallengeMethod, codeChallenge string) SessionOption {
	return oidcOptFunc(func(session *OIDCLoginSession) error {
		pkceState := &PKCEState{
			BaseModel:     ucdb.NewBase(),
			SessionID:     session.ID,
			CodeChallenge: codeChallenge,
			Method:        method,
			Used:          false,
		}

		if err := s.SavePKCEState(ctx, pkceState); err != nil {
			return ucerr.Wrap(err)
		}

		session.PKCEStateID = pkceState.ID
		return nil
	})
}

// Nonce is specified when an authorize request specifies a nonce.
func Nonce(nonce string) SessionOption {
	return oidcOptFunc(func(session *OIDCLoginSession) error {
		session.Nonce = nonce
		return nil
	})
}

// CreateOIDCLoginSession is a helper to create a login session with optional features.
// TODO: this should eventually be part of a higher-level manager class
func CreateOIDCLoginSession(ctx context.Context,
	s *Storage,
	clientID string,
	responseTypes ResponseTypes,
	redirectURI *url.URL,
	state string,
	scopes string,
	opts ...SessionOption) (uuid.UUID, error) {

	session := &OIDCLoginSession{
		BaseModel:        ucdb.NewBase(),
		ClientID:         clientID,
		ResponseTypes:    responseTypes.String(),
		RedirectURI:      redirectURI.String(),
		State:            state,
		Scopes:           scopes,
		MFAChannelStates: MFAChannelStates{ChannelStates: map[string]MFAChannelState{}},
	}

	for _, o := range opts {
		if err := session.ApplyOption(o); err != nil {
			return uuid.Nil, ucerr.Wrap(err)
		}
	}

	if err := s.SaveOIDCLoginSession(ctx, session); err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}

	return session.ID, nil
}

// SetOIDCLoginSessionOIDCProvider is a helper to associate an oidc login provider with a session.
func SetOIDCLoginSessionOIDCProvider(ctx context.Context, s *Storage, sessionID uuid.UUID, provider oidc.ProviderType, issuerURL string) error {
	session, err := s.GetOIDCLoginSession(ctx, sessionID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	session.OIDCProvider = provider
	session.OIDCIssuerURL = issuerURL
	err = s.SaveOIDCLoginSession(ctx, session)
	if err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
