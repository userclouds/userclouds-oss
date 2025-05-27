package sqlshim

// DatabaseType is an enum for supported SQLShim database types
type DatabaseType string

const (
	// DatabaseTypePostgres represents a Postgres database
	DatabaseTypePostgres DatabaseType = "postgres"

	// DatabaseTypeMySQL represents a MySQL database
	DatabaseTypeMySQL DatabaseType = "mysql"
)

//go:generate genconstant DatabaseType
