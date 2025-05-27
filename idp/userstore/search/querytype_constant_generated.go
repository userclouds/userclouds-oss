// NOTE: automatically generated file -- DO NOT EDIT

package search

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t QueryType) MarshalText() ([]byte, error) {
	switch t {
	case QueryTypeTerm:
		return []byte("term"), nil
	case QueryTypeWildcard:
		return []byte("wildcard"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown QueryType value '%s'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *QueryType) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "term":
		*t = QueryTypeTerm
	case "wildcard":
		*t = QueryTypeWildcard
	default:
		return ucerr.Friendlyf(nil, "unknown QueryType value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *QueryType) Validate() error {
	switch *t {
	case QueryTypeTerm:
		return nil
	case QueryTypeWildcard:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown QueryType value '%s'", *t)
	}
}

// Enum implements Enum
func (t QueryType) Enum() []any {
	return []any{
		"term",
		"wildcard",
	}
}

// AllQueryTypes is a slice of all QueryType values
var AllQueryTypes = []QueryType{
	QueryTypeTerm,
	QueryTypeWildcard,
}
