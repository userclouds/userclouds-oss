package tenantdb

import (
	"userclouds.com/infra/migrate"
)

// GetMigrations allows the migration tooling to access the service's internal migrations
func GetMigrations() migrate.Migrations {
	return Migrations[BaselineSchemaVersion+1:]
}
