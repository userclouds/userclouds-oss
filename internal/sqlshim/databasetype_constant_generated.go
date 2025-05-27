// NOTE: automatically generated file -- DO NOT EDIT

package sqlshim

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t DatabaseType) MarshalText() ([]byte, error) {
	switch t {
	case DatabaseTypeMySQL:
		return []byte("mysql"), nil
	case DatabaseTypePostgres:
		return []byte("postgres"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown DatabaseType value '%s'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *DatabaseType) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "mysql":
		*t = DatabaseTypeMySQL
	case "postgres":
		*t = DatabaseTypePostgres
	default:
		return ucerr.Friendlyf(nil, "unknown DatabaseType value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *DatabaseType) Validate() error {
	switch *t {
	case DatabaseTypeMySQL:
		return nil
	case DatabaseTypePostgres:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown DatabaseType value '%s'", *t)
	}
}

// Enum implements Enum
func (t DatabaseType) Enum() []any {
	return []any{
		"mysql",
		"postgres",
	}
}

// AllDatabaseTypes is a slice of all DatabaseType values
var AllDatabaseTypes = []DatabaseType{
	DatabaseTypeMySQL,
	DatabaseTypePostgres,
}
