package idp

import (
	"time"

	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/messaging/email/emailaddress"
)

// AuthnType defines the kinds of authentication factors
type AuthnType string

// AuthnType constants
const (
	AuthnTypePassword AuthnType = "password"
	AuthnTypeOIDC     AuthnType = "social"

	// Used for filter queries; not a valid type
	AuthnTypeAll AuthnType = "all"
)

// Validate implements Validateable
func (a AuthnType) Validate() error {
	if a == AuthnTypePassword || a == AuthnTypeOIDC || a == AuthnTypeAll || a == "" {
		return nil
	}
	return ucerr.Errorf("invalid AuthnType: %s", string(a))
}

// UserAuthn represents an authentication factor for a user.
// NOTE: some fields are not used in some circumstances, e.g. Password is only
// used when creating an account but never used when getting an account.
// TODO: use this for UpdateUser too.
type UserAuthn struct {
	AuthnType AuthnType `json:"authn_type"`

	// Fields specified if AuthnType == 'password'
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	// Fields specified if AuthnType == 'social'
	OIDCProvider  oidc.ProviderType `json:"oidc_provider,omitempty"`
	OIDCIssuerURL string            `json:"oidc_issuer_url,omitempty"`
	OIDCSubject   string            `json:"oidc_subject,omitempty"`
}

// UserMFAChannel represents a configured MFA channel for a user. A
// verified channel may be used for an MFA challenge, and the primary
// channel, which must be verified, is used by default for an MFA challenge.
type UserMFAChannel struct {
	ChannelType        oidc.MFAChannelType `json:"mfa_channel_type"`
	ChannelDescription string              `json:"mfa_channel_description"`
	Primary            bool                `json:"primary"`
	Verified           bool                `json:"verified"`
	LastVerified       time.Time           `json:"last_verified"`
}

// UserBaseProfile is a set of default user profile fields that are common in OIDC claims.
// Follow conventions of https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims for
// all standard fields.
type UserBaseProfile struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name,omitempty"`     // Full name in displayable form (incl titles, suffixes, etc) localized to end-user.
	Nickname      string `json:"nickname,omitempty"` // Casual name of the user, may or may not be same as Given Name.
	Picture       string `json:"picture,omitempty"`  // URL of the user's profile picture.

	// TODO: email is tricky; it's used for authn, 2fa, and (arguably) user profile.
	// If a user merges authns (e.g. I had 2 accounts, oops), then there can be > 1.
	// It may make sense to keep the primary user email (used for 2FA) in `User`, separately
	// from the profile, but allow 0+ profile emails (e.g. alternate contacts, merged accounts, etc).
}

func (up UserBaseProfile) extraValidate() error {
	if up.Email == "" {
		return nil
	}
	a := emailaddress.Address(up.Email)
	if err := a.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

//go:generate gendbjson UserBaseProfile

//go:generate genvalidate UserBaseProfile
