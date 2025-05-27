package migration

import (
	"context"

	"userclouds.com/infra/migrate"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/internal/tenantdb"
)

// TODO: if/when we build a tenant service (to manage tenantdb), this should move
// there along with the /migrations handler that currently lives in idp

// GetServiceData returns information about this "service" used for DB migrations.
func GetServiceData(ctx context.Context, uv universe.Universe) (*migrate.ServiceData, error) {
	// TODO: There is no single DB for the tenants, but we could always return
	// the companyconfig DB here which would make some of the migrate code simpler;
	// there'd be no need to separately get both tenantdb service data AND console service data.

	return &migrate.ServiceData{
		DBCfg:                    nil,
		Migrations:               tenantdb.GetMigrations(),
		BaselineVersion:          tenantdb.BaselineSchemaVersion,
		BaselineCreateStatements: tenantdb.SchemaBaseline.CreateStatements,
		PostgresOnlyExtensions:   tenantdb.SchemaBaseline.PostgresOnlyExtensions,
	}, nil
}
