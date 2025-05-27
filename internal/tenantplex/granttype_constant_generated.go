// NOTE: automatically generated file -- DO NOT EDIT

package tenantplex

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t GrantType) MarshalText() ([]byte, error) {
	switch t {
	case GrantTypeAuthorizationCode:
		return []byte("authorization_code"), nil
	case GrantTypeClientCredentials:
		return []byte("client_credentials"), nil
	case GrantTypeDeviceCode:
		return []byte("urn:ietf:params:oauth:grant-type:device_code"), nil
	case GrantTypeImplicit:
		return []byte("implicit"), nil
	case GrantTypeMFA:
		return []byte("mfa"), nil
	case GrantTypePassword:
		return []byte("password"), nil
	case GrantTypeRefreshToken:
		return []byte("refresh_token"), nil
	case GrantTypeUnknown:
		return []byte(""), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown GrantType value '%s'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *GrantType) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "authorization_code":
		*t = GrantTypeAuthorizationCode
	case "client_credentials":
		*t = GrantTypeClientCredentials
	case "urn:ietf:params:oauth:grant-type:device_code":
		*t = GrantTypeDeviceCode
	case "implicit":
		*t = GrantTypeImplicit
	case "mfa":
		*t = GrantTypeMFA
	case "password":
		*t = GrantTypePassword
	case "refresh_token":
		*t = GrantTypeRefreshToken
	case "":
		*t = GrantTypeUnknown
	default:
		return ucerr.Friendlyf(nil, "unknown GrantType value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *GrantType) Validate() error {
	switch *t {
	case GrantTypeAuthorizationCode:
		return nil
	case GrantTypeClientCredentials:
		return nil
	case GrantTypeDeviceCode:
		return nil
	case GrantTypeImplicit:
		return nil
	case GrantTypeMFA:
		return nil
	case GrantTypePassword:
		return nil
	case GrantTypeRefreshToken:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown GrantType value '%s'", *t)
	}
}

// Enum implements Enum
func (t GrantType) Enum() []any {
	return []any{
		"authorization_code",
		"client_credentials",
		"urn:ietf:params:oauth:grant-type:device_code",
		"implicit",
		"mfa",
		"password",
		"refresh_token",
	}
}

// AllGrantTypes is a slice of all GrantType values
var AllGrantTypes = []GrantType{
	GrantTypeAuthorizationCode,
	GrantTypeClientCredentials,
	GrantTypeDeviceCode,
	GrantTypeImplicit,
	GrantTypeMFA,
	GrantTypePassword,
	GrantTypeRefreshToken,
}
