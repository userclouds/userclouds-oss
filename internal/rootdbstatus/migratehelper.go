package rootdbstatus

import (
	"context"

	"userclouds.com/infra/migrate"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/yamlconfig"
)

// GetServiceData returns information about this "service" used for DB migrations.
func GetServiceData(ctx context.Context) (*migrate.ServiceData, error) {
	var cfg Config
	if err := yamlconfig.LoadDatabaseConfig(ctx, "rootdbstatus", &cfg, false); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &migrate.ServiceData{
		DBCfg:                    &cfg.DB,
		Migrations:               GetMigrations(),
		BaselineVersion:          -1,
		BaselineCreateStatements: nil,
	}, nil
}

// GetMigrations allows the migration tooling to access the service's internal migrations
func GetMigrations() migrate.Migrations {
	return Migrations
}
