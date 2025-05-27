package companyconfig

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
)

// MigratedTenant is used for tracking which tenants have had their user columns cleaned up
type MigratedTenant struct {
	ucdb.BaseModel
	RunID string `db:"run_id"`
}

// AreTenantUserColumnsCleaned returns true if there is an entry in the tenant_user_column_cleanup table
func (s *Storage) AreTenantUserColumnsCleaned(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	const q = "/* bypass-known-table-check */ SELECT id, created, updated, deleted, run_id FROM tenant_user_column_cleanup WHERE id = $1;"

	var obj MigratedTenant
	if err := s.db.GetContext(ctx, "AreTenantUserColumnsCleaned", &obj, q, tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return true, nil
}

// SetTenantUserColumnsCleaned records the run ID for the associated tenant ID
func (s *Storage) SetTenantUserColumnsCleaned(ctx context.Context, tenantID uuid.UUID, runID string) error {
	const q = "/* lint-sql-ok */ INSERT INTO tenant_user_column_cleanup (id, updated, run_id) VALUES ($1, NOW(), $2);"

	if _, err := s.db.ExecContext(
		ctx,
		"SetTenantUserColumnsCleaned",
		q,
		tenantID,
		runID,
	); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}
