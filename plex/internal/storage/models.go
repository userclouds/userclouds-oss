package storage

import (
	"database/sql/driver"
	"time"

	"github.com/crewjam/saml"
	"github.com/gofrs/uuid"
	"github.com/lib/pq"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// OIDCLoginSession represents a single OIDC request
// TODO: This should expire and get cleaned up at some point.
// TODO: This is being used for all login & verification session tracking and should
// be refactored. e.g. OTP Flow could be a struct referenced by login session AND the email verification logic.
type OIDCLoginSession struct {
	ucdb.BaseModel

	ClientID      string `db:"client_id" validate:"notempty"`
	ResponseTypes string `db:"response_types" validate:"notempty"` // space delimited list
	RedirectURI   string `db:"redirect_uri" validate:"notempty"`
	State         string `db:"state" validate:"notempty"`
	Scopes        string `db:"scopes" validate:"notempty"`
	Nonce         string `db:"nonce"`

	// OIDC Login Provider
	OIDCProvider oidc.ProviderType `db:"social_provider"`

	// OIDC Issuer URL
	OIDCIssuerURL string `db:"oidc_issuer_url"`

	// If not uuid.Nil, this refers to a MFAState which means this login request
	// has been MFA challenged.
	MFAStateID uuid.UUID `db:"mfa_state_id"`

	MFAChannelStates MFAChannelStates `db:"mfa_channel_states" validate:"skip"`

	// If not uuid.Nil, this refers to a OTPState.
	OTPStateID uuid.UUID `db:"otp_state_id"`

	// If not uuid.Nil, this refers to a PKCEState.
	PKCEStateID uuid.UUID `db:"pkce_state_id"`

	// If not uuid.Nil, this refers to a DelegationState object
	DelegationStateID uuid.UUID `db:"delegation_state_id"`

	// If not uuid.Nil, this refers to a PlexToken object
	PlexTokenID uuid.UUID `db:"plex_token_id"`

	AddAuthnProviderData AddAuthnProviderData `db:"add_authn_provider_data"`
}

func (s *OIDCLoginSession) extraValidate() error {
	return ucerr.Wrap(s.OIDCProvider.ValidateIssuerURL(s.OIDCIssuerURL))
}

//go:generate genpageable OIDCLoginSession

//go:generate genvalidate OIDCLoginSession

// MFAState contains state for MFA challenges which may be required on some logins.
type MFAState struct {
	ucdb.BaseModel

	SessionID                 uuid.UUID         `db:"session_id" validate:"notnil"`
	Token                     string            `db:"token" validate:"notempty"`
	Provider                  uuid.UUID         `db:"provider" validate:"notnil"`
	ChannelID                 uuid.UUID         `db:"channel_id"`
	SupportedChannels         oidc.MFAChannels  `db:"supported_channels"`
	Purpose                   MFAPurpose        `db:"purpose"`
	ChallengeState            MFAChallengeState `db:"challenge_state"`
	EvaluateSupportedChannels bool              `db:"evaluate_supported_channels"`
}

func (mfas *MFAState) extraValidate() error {
	if mfas.ChannelID != uuid.Nil {
		if _, err := mfas.SupportedChannels.FindChannel(mfas.ChannelID); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

//go:generate genpageable MFAState

//go:generate genvalidate MFAState

// OTPState contains state for One Time Passwords associated with passwordless login,
// magic links, email verification, and invites.
type OTPState struct {
	ucdb.BaseModel

	// ID of the OIDCLoginSession which generated this OTP
	SessionID uuid.UUID `db:"session_id" validate:"notnil"`

	UserID  string     `db:"user_id"`
	Email   string     `db:"email" validate:"notempty"`
	Code    string     `db:"code" validate:"notempty"`
	Expires time.Time  `db:"expires"`
	Used    bool       `db:"used"`
	Purpose OTPPurpose `db:"purpose"`
}

func (o OTPState) extraValidate() error {
	if o.Purpose != OTPPurposeInvite && o.UserID == "" {
		return ucerr.Errorf("OTPState.UserID (%v) can't be nil if OTPState.Purpose (%s) is not OTPPurposeInvite", o.ID, o.Purpose)
	}
	return nil
}

//go:generate genpageable OTPState

//go:generate genvalidate OTPState

// PKCEState contains state for logins with Authorizaton Code + Proof Key Code Exchange (PKCE)
type PKCEState struct {
	ucdb.BaseModel

	// ID of the OIDCLoginSession which spawned this PKCE flow
	SessionID uuid.UUID `db:"session_id" validate:"notnil"`

	CodeChallenge string                     `db:"code_challenge" validate:"notempty"`
	Method        crypto.CodeChallengeMethod `db:"method"`
	Used          bool                       `db:"used"`
}

//go:generate genpageable PKCEState

//go:generate genvalidate PKCEState

// DelegationState contains state for a login session that allows delegation
type DelegationState struct {
	ucdb.BaseModel

	AuthenticatedUserID string `db:"authenticated_user_id"`
}

//go:generate genpageable DelegationState

//go:generate genvalidate DelegationState

// DelegationInvite represents invitations sent to delegate account access
// TODO: once we figure out delegation past the demo phase, this could either
// be integrated into the core plex invite flow, or moved out into another app
type DelegationInvite struct {
	ucdb.BaseModel

	ClientID           string `db:"client_id"`
	InvitedToAccountID string `db:"invited_to_account_id"`
}

//go:generate genpageable DelegationInvite

//go:generate genvalidate DelegationInvite

// SAMLSession represents a user session. It is returned by the
// SessionProvider implementation's GetSession method. Fields here
// are used to set fields in the SAML assertion.
type SAMLSession struct {
	ucdb.BaseModel

	ExpireTime time.Time `db:"expire_time"`
	Index      string    `db:"_index"`

	NameID       string `db:"name_id"`
	NameIDFormat string `db:"name_id_format"`
	SubjectID    string `db:"subject_id"`

	Groups                pq.StringArray `db:"groups" validate:"skip"`
	UserName              string         `db:"user_name" validate:"skip"`
	UserEmail             string         `db:"user_email" validate:"skip"`
	UserCommonName        string         `db:"user_common_name" validate:"skip"`
	UserSurname           string         `db:"user_surname" validate:"skip"`
	UserGivenName         string         `db:"user_given_name" validate:"skip"`
	UserScopedAffiliation string         `db:"user_scoped_affiliation" validate:"skip"`

	CustomAttributes AttributeList `db:"custom_attributes" validate:"skip"`

	// used for the internal OIDC redirect to log you in during SAML IDP auth
	State string `db:"state" validate:"skip"`

	RelayState    string `db:"relay_state" validate:"skip"`
	RequestBuffer []byte `db:"request_buffer" validate:"skip"`
}

//go:generate genpageable SAMLSession

//go:generate genvalidate SAMLSession

// AttributeList exists just to manage DB interactions for this type
type AttributeList []saml.Attribute

// Value implements the driver.Valuer interface.
func (a AttributeList) Value() (driver.Value, error) {
	return pq.GenericArray{A: a}.Value()
}

// Scan implements the sql.Scanner interface.
func (a *AttributeList) Scan(src any) error {
	return pq.GenericArray{A: a}.Scan(src)
}
