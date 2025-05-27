// NOTE: automatically generated file -- DO NOT EDIT

package crypto

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t CodeChallengeMethod) MarshalText() ([]byte, error) {
	switch t {
	case CodeChallengeMethodInvalid:
		return []byte("invalid"), nil
	case CodeChallengeMethodS256:
		return []byte("s256"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown CodeChallengeMethod value '%d'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *CodeChallengeMethod) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "invalid":
		*t = CodeChallengeMethodInvalid
	case "s256":
		*t = CodeChallengeMethodS256
	default:
		return ucerr.Friendlyf(nil, "unknown CodeChallengeMethod value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *CodeChallengeMethod) Validate() error {
	switch *t {
	case CodeChallengeMethodS256:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown CodeChallengeMethod value '%d'", *t)
	}
}

// Enum implements Enum
func (t CodeChallengeMethod) Enum() []any {
	return []any{
		"s256",
	}
}

// AllCodeChallengeMethods is a slice of all CodeChallengeMethod values
var AllCodeChallengeMethods = []CodeChallengeMethod{
	CodeChallengeMethodS256,
}

// just here for easier debugging
func (t CodeChallengeMethod) String() string {
	bs, err := t.MarshalText()
	if err != nil {
		return err.Error()
	}
	return string(bs)
}
