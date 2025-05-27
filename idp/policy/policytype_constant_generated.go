// NOTE: automatically generated file -- DO NOT EDIT

package policy

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t PolicyType) MarshalText() ([]byte, error) {
	switch t {
	case PolicyTypeCompositeAnd:
		return []byte("composite_and"), nil
	case PolicyTypeCompositeOr:
		return []byte("composite_or"), nil
	case PolicyTypeInvalid:
		return []byte("invalid"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown PolicyType value '%s'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *PolicyType) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "composite_and":
		*t = PolicyTypeCompositeAnd
	case "composite_or":
		*t = PolicyTypeCompositeOr
	case "invalid":
		*t = PolicyTypeInvalid
	default:
		return ucerr.Friendlyf(nil, "unknown PolicyType value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *PolicyType) Validate() error {
	switch *t {
	case PolicyTypeCompositeAnd:
		return nil
	case PolicyTypeCompositeOr:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown PolicyType value '%s'", *t)
	}
}

// Enum implements Enum
func (t PolicyType) Enum() []any {
	return []any{
		"composite_and",
		"composite_or",
	}
}

// AllPolicyTypes is a slice of all PolicyType values
var AllPolicyTypes = []PolicyType{
	PolicyTypeCompositeAnd,
	PolicyTypeCompositeOr,
}
