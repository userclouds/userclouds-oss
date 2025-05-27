// NOTE: automatically generated file -- DO NOT EDIT

package oidc

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t AuthenticatorType) MarshalText() ([]byte, error) {
	switch t {
	case AuthenticatorTypeAuth0Guardian:
		return []byte("auth0guardian"), nil
	case AuthenticatorTypeDuoMobile:
		return []byte("duomobile"), nil
	case AuthenticatorTypeGoogleAuthenticator:
		return []byte("googleauthenticator"), nil
	case AuthenticatorTypeMicrosoftAuthenticator:
		return []byte("microsoftauthenticator"), nil
	case AuthenticatorTypeTwilioAuthy:
		return []byte("twilioauthy"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown AuthenticatorType value '%d'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *AuthenticatorType) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "auth0guardian":
		*t = AuthenticatorTypeAuth0Guardian
	case "duomobile":
		*t = AuthenticatorTypeDuoMobile
	case "googleauthenticator":
		*t = AuthenticatorTypeGoogleAuthenticator
	case "microsoftauthenticator":
		*t = AuthenticatorTypeMicrosoftAuthenticator
	case "twilioauthy":
		*t = AuthenticatorTypeTwilioAuthy
	default:
		return ucerr.Friendlyf(nil, "unknown AuthenticatorType value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *AuthenticatorType) Validate() error {
	switch *t {
	case AuthenticatorTypeAuth0Guardian:
		return nil
	case AuthenticatorTypeDuoMobile:
		return nil
	case AuthenticatorTypeGoogleAuthenticator:
		return nil
	case AuthenticatorTypeMicrosoftAuthenticator:
		return nil
	case AuthenticatorTypeTwilioAuthy:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown AuthenticatorType value '%d'", *t)
	}
}

// Enum implements Enum
func (t AuthenticatorType) Enum() []any {
	return []any{
		"auth0guardian",
		"duomobile",
		"googleauthenticator",
		"microsoftauthenticator",
		"twilioauthy",
	}
}

// AllAuthenticatorTypes is a slice of all AuthenticatorType values
var AllAuthenticatorTypes = []AuthenticatorType{
	AuthenticatorTypeAuth0Guardian,
	AuthenticatorTypeDuoMobile,
	AuthenticatorTypeGoogleAuthenticator,
	AuthenticatorTypeMicrosoftAuthenticator,
	AuthenticatorTypeTwilioAuthy,
}

// just here for easier debugging
func (t AuthenticatorType) String() string {
	bs, err := t.MarshalText()
	if err != nil {
		return err.Error()
	}
	return string(bs)
}
