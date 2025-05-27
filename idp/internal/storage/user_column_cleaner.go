package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/infra/uctypes/uuidarray"
)

type userColumnCleaner struct {
	s *Storage
}

func newUserColumnCleaner(s *Storage) userColumnCleaner {
	return userColumnCleaner{s: s}
}

func (c userColumnCleaner) cleanUpTable(ctx context.Context, columnID uuid.UUID) error {
	columnNames := c.getTableNamesForID(columnID)
	for _, columnName := range columnNames {
		if err := c.dropTable(ctx, columnName); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

func (c userColumnCleaner) cleanUpTenant(ctx context.Context) error {
	if err := c.cleanUpUsersTable(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	columnIDs, err := c.getAllColumnIDs(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	for _, columnID := range columnIDs {
		if err := c.cleanUpTable(ctx, columnID); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

func (c userColumnCleaner) cleanUpUsersTable(ctx context.Context) error {
	expectedColumns := set.NewStringSet("id", "created", "updated", "deleted", "organization_id", "_version")
	userColumnNames, err := c.getAllUserColumnNames(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	for _, userColumnName := range userColumnNames {
		if !expectedColumns.Contains(userColumnName) {
			if err := c.dropUserColumn(ctx, userColumnName); err != nil {
				return ucerr.Wrap(err)
			}
		}
	}

	return nil
}

func (c userColumnCleaner) dropTable(ctx context.Context, tableName string) error {
	q := fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName)

	_, err := c.s.db.ExecContext(ctx, "userColumnCleaner.dropTable", q)
	return ucerr.Wrap(err)
}

func (c userColumnCleaner) dropUserColumn(ctx context.Context, columnName string) error {
	q := fmt.Sprintf("ALTER TABLE users DROP COLUMN IF EXISTS %s;", columnName)

	_, err := c.s.db.ExecContext(ctx, "userColumnCleaner.dropUserColumn", q)
	return ucerr.Wrap(err)
}

func (c userColumnCleaner) getAllColumnIDs(ctx context.Context) (uuidarray.UUIDArray, error) {
	const q = "/* lint-sql-ok */ SELECT DISTINCT id from columns;"

	var ids uuidarray.UUIDArray
	if err := c.s.db.SelectContext(ctx, "userColumnCleaner.getAllColumnIDs", &ids, q); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return ids, nil
}

func (c userColumnCleaner) getAllUserColumnNames(ctx context.Context) ([]string, error) {
	const q = "/* lint-sql-ok */ SELECT column_name FROM information_schema.columns WHERE table_name='users';"

	var columnNames []string
	if err := c.s.db.SelectContext(ctx, "userColumnCleaner.getAllUserColumnNames", &columnNames, q); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return columnNames, nil
}

func (c userColumnCleaner) getTableNamesForID(columnID uuid.UUID) []string {
	idString := strings.ReplaceAll(columnID.String(), "-", "")
	return []string{
		fmt.Sprintf("col_%s", idString),
		fmt.Sprintf("col_%s_deleted", idString),
	}
}

// CleanUpUserColumns will clean up the user columns for the tenant
func (s *Storage) CleanUpUserColumns(ctx context.Context) error {

	ucc := newUserColumnCleaner(s)
	return ucerr.Wrap(ucc.cleanUpTenant(ctx))
}
