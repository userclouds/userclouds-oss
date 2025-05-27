package storage

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// DeleteColumnValueRetentionDurationsByPurposeID deletes all ColumnValueRetentionDuration
// entries that reference the specified purpose ID.
func (s *Storage) DeleteColumnValueRetentionDurationsByPurposeID(ctx context.Context, purposeID uuid.UUID) error {
	const q = "UPDATE column_value_retention_durations SET deleted=CLOCK_TIMESTAMP() WHERE purpose_id=$1 AND deleted='0001-01-01 00:00:00' RETURNING id;"

	var deletedIDs []uuid.UUID
	if err := s.db.SelectContext(ctx, "DeleteColumnValueRetentionDurationsByPurposeID", &deletedIDs, q, purposeID); err != nil {
		return ucerr.Wrap(err)
	}

	if s.cm != nil && len(deletedIDs) > 0 {
		for _, id := range deletedIDs {
			objBase := ColumnValueRetentionDuration{VersionBaseModel: ucdb.NewVersionBaseWithID(id)}
			sentinel, err := cache.TakeItemLock(ctx, cache.Delete, *s.cm, objBase)
			if err != nil {
				uclog.Warningf(ctx, "Error taking lock for delete: %v", err)
				continue
			}
			defer cache.DeleteItemFromCache[ColumnValueRetentionDuration](ctx, *s.cm, objBase, sentinel)
		}
	}

	uclog.Infof(ctx, "Deleted %d ColumnValueRetentionDuration entries for purpose ID %v", len(deletedIDs), purposeID)
	return nil
}
