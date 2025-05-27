// NOTE: automatically generated file -- DO NOT EDIT

package search

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t IndexType) MarshalText() ([]byte, error) {
	switch t {
	case IndexTypeDeprecated:
		return []byte("deprecated"), nil
	case IndexTypeNgram:
		return []byte("ngram"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown IndexType value '%s'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *IndexType) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "deprecated":
		*t = IndexTypeDeprecated
	case "ngram":
		*t = IndexTypeNgram
	default:
		return ucerr.Friendlyf(nil, "unknown IndexType value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *IndexType) Validate() error {
	switch *t {
	case IndexTypeDeprecated:
		return nil
	case IndexTypeNgram:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown IndexType value '%s'", *t)
	}
}

// Enum implements Enum
func (t IndexType) Enum() []any {
	return []any{
		"deprecated",
		"ngram",
	}
}

// AllIndexTypes is a slice of all IndexType values
var AllIndexTypes = []IndexType{
	IndexTypeDeprecated,
	IndexTypeNgram,
}
