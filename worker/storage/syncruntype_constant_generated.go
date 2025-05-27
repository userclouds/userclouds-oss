// NOTE: automatically generated file -- DO NOT EDIT

package storage

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t SyncRunType) MarshalText() ([]byte, error) {
	switch t {
	case SyncRunTypeApp:
		return []byte("app"), nil
	case SyncRunTypeUser:
		return []byte("user"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown SyncRunType value '%s'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *SyncRunType) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "app":
		*t = SyncRunTypeApp
	case "user":
		*t = SyncRunTypeUser
	default:
		return ucerr.Friendlyf(nil, "unknown SyncRunType value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *SyncRunType) Validate() error {
	switch *t {
	case SyncRunTypeApp:
		return nil
	case SyncRunTypeUser:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown SyncRunType value '%s'", *t)
	}
}

// Enum implements Enum
func (t SyncRunType) Enum() []any {
	return []any{
		"app",
		"user",
	}
}

// AllSyncRunTypes is a slice of all SyncRunType values
var AllSyncRunTypes = []SyncRunType{
	SyncRunTypeApp,
	SyncRunTypeUser,
}
