package companyconfig

import (
	"context"

	"userclouds.com/infra/ucerr"
)

// HasAnyTenants returns true if there are any non-deleted tenants
func (s *Storage) HasAnyTenants(ctx context.Context) (bool, error) {
	const q = "SELECT COUNT(*) FROM tenants WHERE deleted='0001-01-01 00:00:00' LIMIT 1;"

	var c int
	if err := s.db.GetContext(ctx, "HasAnyTenants", &c, q); err != nil {
		return false, ucerr.Wrap(err)
	}

	return c == 1, nil
}
