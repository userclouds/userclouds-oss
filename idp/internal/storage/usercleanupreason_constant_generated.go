// NOTE: automatically generated file -- DO NOT EDIT

package storage

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t UserCleanupReason) MarshalText() ([]byte, error) {
	switch t {
	case UserCleanupReasonDuplicateValue:
		return []byte("duplicatevalue"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown UserCleanupReason value '%d'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *UserCleanupReason) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "duplicatevalue":
		*t = UserCleanupReasonDuplicateValue
	default:
		return ucerr.Friendlyf(nil, "unknown UserCleanupReason value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *UserCleanupReason) Validate() error {
	switch *t {
	case UserCleanupReasonDuplicateValue:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown UserCleanupReason value '%d'", *t)
	}
}

// Enum implements Enum
func (t UserCleanupReason) Enum() []any {
	return []any{
		"duplicatevalue",
	}
}

// AllUserCleanupReasons is a slice of all UserCleanupReason values
var AllUserCleanupReasons = []UserCleanupReason{
	UserCleanupReasonDuplicateValue,
}

// just here for easier debugging
func (t UserCleanupReason) String() string {
	bs, err := t.MarshalText()
	if err != nil {
		return err.Error()
	}
	return string(bs)
}
