package sqlshimingest

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/helpers"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/tenantmap"
)

// IngestSqlshimDatabaseSchemas ingests the sqlshim database schema for the given tenant
func IngestSqlshimDatabaseSchemas(ctx context.Context, ts *tenantmap.TenantState, databaseID uuid.UUID) error {
	uclog.Infof(ctx, "Worker ingesting sqlshim database schemas for tenant %v", ts.ID)
	return ucerr.Wrap(helpers.IngestSqlshimDatabaseSchemas(ctx, ts, databaseID))
}
