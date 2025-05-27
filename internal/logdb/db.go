package logdb

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/migrate"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/companyconfig"
)

// TODO - should find a shared place for these
const (
	maxConnectionsPerDB     int = 5
	maxIdleConnectionsPerDB int = 5
)

// Connect connects to a per-tenant log DB.
// Will return sql.ErrNoRows if the tenant is not found.
func Connect(ctx context.Context, storage *companyconfig.Storage, tenantID uuid.UUID) (*ucdb.DB, error) {
	tenantInternal, err := storage.GetTenantInternal(ctx, tenantID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if err := tenantInternal.LogConfig.LogDB.Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	db, err := ConnectWithConfig(ctx, &tenantInternal.LogConfig.LogDB)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return db, nil
}

// ConnectWithConfig connects to a per-tenant log DB via config.
func ConnectWithConfig(ctx context.Context, cfg *ucdb.Config) (*ucdb.DB, error) {
	db, err := ucdb.NewWithLimits(ctx, cfg, migrate.SchemaValidator(Schema), maxConnectionsPerDB, maxIdleConnectionsPerDB)
	return db, ucerr.Wrap(err)
}
