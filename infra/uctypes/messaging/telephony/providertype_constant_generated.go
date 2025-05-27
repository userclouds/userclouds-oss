// NOTE: automatically generated file -- DO NOT EDIT

package telephony

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t ProviderType) MarshalText() ([]byte, error) {
	switch t {
	case ProviderTypeNone:
		return []byte("none"), nil
	case ProviderTypeTwilio:
		return []byte("twilio"), nil
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
	case "none":
		*t = ProviderTypeNone
	case "twilio":
		*t = ProviderTypeTwilio
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
	case ProviderTypeNone:
		return nil
	case ProviderTypeTwilio:
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
		"none",
		"twilio",
		"unsupported",
	}
}

// AllProviderTypes is a slice of all ProviderType values
var AllProviderTypes = []ProviderType{
	ProviderTypeNone,
	ProviderTypeTwilio,
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
