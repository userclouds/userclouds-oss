// NOTE: automatically generated file -- DO NOT EDIT

package oidc

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t ProviderType) MarshalText() ([]byte, error) {
	switch t {
	case ProviderTypeCustom:
		return []byte("custom"), nil
	case ProviderTypeFacebook:
		return []byte("facebook"), nil
	case ProviderTypeGoogle:
		return []byte("google"), nil
	case ProviderTypeLinkedIn:
		return []byte("linkedin"), nil
	case ProviderTypeMicrosoft:
		return []byte("microsoft"), nil
	case ProviderTypeNone:
		return []byte("none"), nil
	case ProviderTypeUnsupported:
		return []byte("unsupported"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown ProviderType value '%d'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *ProviderType) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "custom":
		*t = ProviderTypeCustom
	case "facebook":
		*t = ProviderTypeFacebook
	case "google":
		*t = ProviderTypeGoogle
	case "linkedin":
		*t = ProviderTypeLinkedIn
	case "microsoft":
		*t = ProviderTypeMicrosoft
	case "none":
		*t = ProviderTypeNone
	case "unsupported":
		*t = ProviderTypeUnsupported
	default:
		return ucerr.Friendlyf(nil, "unknown ProviderType value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *ProviderType) Validate() error {
	switch *t {
	case ProviderTypeCustom:
		return nil
	case ProviderTypeFacebook:
		return nil
	case ProviderTypeGoogle:
		return nil
	case ProviderTypeLinkedIn:
		return nil
	case ProviderTypeMicrosoft:
		return nil
	case ProviderTypeNone:
		return nil
	case ProviderTypeUnsupported:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown ProviderType value '%d'", *t)
	}
}

// Enum implements Enum
func (t ProviderType) Enum() []any {
	return []any{
		"custom",
		"facebook",
		"google",
		"linkedin",
		"microsoft",
		"none",
		"unsupported",
	}
}

// AllProviderTypes is a slice of all ProviderType values
var AllProviderTypes = []ProviderType{
	ProviderTypeCustom,
	ProviderTypeFacebook,
	ProviderTypeGoogle,
	ProviderTypeLinkedIn,
	ProviderTypeMicrosoft,
	ProviderTypeNone,
	ProviderTypeUnsupported,
}

// just here for easier debugging
func (t ProviderType) String() string {
	bs, err := t.MarshalText()
	if err != nil {
		return err.Error()
	}
	return string(bs)
}
