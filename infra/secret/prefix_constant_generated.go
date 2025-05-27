// NOTE: automatically generated file -- DO NOT EDIT

package secret

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t Prefix) MarshalText() ([]byte, error) {
	switch t {
	case PrefixAWS:
		return []byte("aws://secrets/"), nil
	case PrefixDev:
		return []byte("dev://"), nil
	case PrefixDevLiteral:
		return []byte("dev-literal://"), nil
	case PrefixEnv:
		return []byte("env://"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown Prefix value '%s'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *Prefix) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "aws://secrets/":
		*t = PrefixAWS
	case "dev://":
		*t = PrefixDev
	case "dev-literal://":
		*t = PrefixDevLiteral
	case "env://":
		*t = PrefixEnv
	default:
		return ucerr.Friendlyf(nil, "unknown Prefix value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *Prefix) Validate() error {
	switch *t {
	case PrefixAWS:
		return nil
	case PrefixDev:
		return nil
	case PrefixDevLiteral:
		return nil
	case PrefixEnv:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown Prefix value '%s'", *t)
	}
}

// Enum implements Enum
func (t Prefix) Enum() []any {
	return []any{
		"aws://secrets/",
		"dev://",
		"dev-literal://",
		"env://",
	}
}

// AllPrefixes is a slice of all Prefix values
var AllPrefixes = []Prefix{
	PrefixAWS,
	PrefixDev,
	PrefixDevLiteral,
	PrefixEnv,
}
