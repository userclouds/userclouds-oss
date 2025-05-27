// NOTE: automatically generated file -- DO NOT EDIT

package userstore

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t ColumnIndexType) MarshalText() ([]byte, error) {
	switch t {
	case ColumnIndexTypeIndexed:
		return []byte("indexed"), nil
	case ColumnIndexTypeNone:
		return []byte("none"), nil
	case ColumnIndexTypeUnique:
		return []byte("unique"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown ColumnIndexType value '%s'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *ColumnIndexType) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "indexed":
		*t = ColumnIndexTypeIndexed
	case "none":
		*t = ColumnIndexTypeNone
	case "unique":
		*t = ColumnIndexTypeUnique
	default:
		return ucerr.Friendlyf(nil, "unknown ColumnIndexType value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *ColumnIndexType) Validate() error {
	switch *t {
	case ColumnIndexTypeIndexed:
		return nil
	case ColumnIndexTypeNone:
		return nil
	case ColumnIndexTypeUnique:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown ColumnIndexType value '%s'", *t)
	}
}

// Enum implements Enum
func (t ColumnIndexType) Enum() []any {
	return []any{
		"indexed",
		"none",
		"unique",
	}
}

// AllColumnIndexTypes is a slice of all ColumnIndexType values
var AllColumnIndexTypes = []ColumnIndexType{
	ColumnIndexTypeIndexed,
	ColumnIndexTypeNone,
	ColumnIndexTypeUnique,
}
