// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/lib/pq"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

// IsIDPSyncRecordSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsIDPSyncRecordSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM idp_sync_records WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsIDPSyncRecordSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetIDPSyncRecord loads a IDPSyncRecord by ID
func (s *Storage) GetIDPSyncRecord(ctx context.Context, id uuid.UUID) (*IDPSyncRecord, error) {
	const q = "SELECT id, updated, deleted, sync_run_id, object_id, error, warning, user_id, created FROM idp_sync_records WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj IDPSyncRecord
	if err := s.db.GetContext(ctx, "GetIDPSyncRecord", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "IDPSyncRecord %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetIDPSyncRecordSoftDeleted loads a IDPSyncRecord by ID iff it's soft-deleted
func (s *Storage) GetIDPSyncRecordSoftDeleted(ctx context.Context, id uuid.UUID) (*IDPSyncRecord, error) {
	const q = "SELECT id, updated, deleted, sync_run_id, object_id, error, warning, user_id, created FROM idp_sync_records WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj IDPSyncRecord
	if err := s.db.GetContext(ctx, "GetIDPSyncRecordSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted IDPSyncRecord %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetIDPSyncRecordsForIDs loads multiple IDPSyncRecord for a given list of IDs
func (s *Storage) GetIDPSyncRecordsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]IDPSyncRecord, error) {
	items := make([]IDPSyncRecord, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getIDPSyncRecordsHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getIDPSyncRecordsHelperForIDs loads multiple IDPSyncRecord for a given list of IDs from the DB
func (s *Storage) getIDPSyncRecordsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]IDPSyncRecord, error) {
	const q = "SELECT id, updated, deleted, sync_run_id, object_id, error, warning, user_id, created FROM idp_sync_records WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []IDPSyncRecord
	if err := s.db.SelectContextWithDirty(ctx, "GetIDPSyncRecordsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested IDPSyncRecords  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListIDPSyncRecordsPaginated loads a paginated list of IDPSyncRecords for the specified paginator settings
func (s *Storage) ListIDPSyncRecordsPaginated(ctx context.Context, p pagination.Paginator) ([]IDPSyncRecord, *pagination.ResponseFields, error) {
	return s.listInnerIDPSyncRecordsPaginated(ctx, p, false)
}

// listInnerIDPSyncRecordsPaginated loads a paginated list of IDPSyncRecords for the specified paginator settings
func (s *Storage) listInnerIDPSyncRecordsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]IDPSyncRecord, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, sync_run_id, object_id, error, warning, user_id, created FROM (SELECT id, updated, deleted, sync_run_id, object_id, error, warning, user_id, created FROM idp_sync_records WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []IDPSyncRecord
	if err := s.db.SelectContext(ctx, "ListIDPSyncRecordsPaginated", &objsDB, q, queryFields...); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	objs, respFields := pagination.ProcessResults(objsDB, p.GetCursor(), p.GetLimit(), p.IsForward(), p.GetSortKey())
	if respFields.HasNext {
		if err := p.ValidateCursor(respFields.Next); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	if respFields.HasPrev {
		if err := p.ValidateCursor(respFields.Prev); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
	}

	return objs, &respFields, nil
}

// SaveIDPSyncRecord saves a IDPSyncRecord
func (s *Storage) SaveIDPSyncRecord(ctx context.Context, obj *IDPSyncRecord) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerIDPSyncRecord(ctx, obj))
}

// SaveIDPSyncRecord saves a IDPSyncRecord
func (s *Storage) saveInnerIDPSyncRecord(ctx context.Context, obj *IDPSyncRecord) error {
	const q = "INSERT INTO idp_sync_records (id, updated, deleted, sync_run_id, object_id, error, warning, user_id) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, sync_run_id = $3, object_id = $4, error = $5, warning = $6, user_id = $7 WHERE (idp_sync_records.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveIDPSyncRecord", obj, q, obj.ID, obj.Deleted, obj.SyncRunID, obj.ObjectID, obj.Error, obj.Warning, obj.UserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "IDPSyncRecord %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteIDPSyncRecord soft-deletes a IDPSyncRecord which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteIDPSyncRecord(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerIDPSyncRecord(ctx, objID, false))
}

// deleteInnerIDPSyncRecord soft-deletes a IDPSyncRecord which is currently alive
func (s *Storage) deleteInnerIDPSyncRecord(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE idp_sync_records SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteIDPSyncRecord", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting IDPSyncRecord %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "IDPSyncRecord %v not found", objID)
	}
	return nil
}
