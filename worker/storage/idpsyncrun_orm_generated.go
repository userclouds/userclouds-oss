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

// IsIDPSyncRunSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsIDPSyncRunSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM idp_sync_runs WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsIDPSyncRunSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetIDPSyncRun loads a IDPSyncRun by ID
func (s *Storage) GetIDPSyncRun(ctx context.Context, id uuid.UUID) (*IDPSyncRun, error) {
	const q = "SELECT id, updated, deleted, type, active_provider_id, follower_provider_ids, since, until, error, total_records, failed_records, warning_records, created FROM idp_sync_runs WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj IDPSyncRun
	if err := s.db.GetContext(ctx, "GetIDPSyncRun", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "IDPSyncRun %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetIDPSyncRunSoftDeleted loads a IDPSyncRun by ID iff it's soft-deleted
func (s *Storage) GetIDPSyncRunSoftDeleted(ctx context.Context, id uuid.UUID) (*IDPSyncRun, error) {
	const q = "SELECT id, updated, deleted, type, active_provider_id, follower_provider_ids, since, until, error, total_records, failed_records, warning_records, created FROM idp_sync_runs WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj IDPSyncRun
	if err := s.db.GetContext(ctx, "GetIDPSyncRunSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted IDPSyncRun %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetIDPSyncRunsForIDs loads multiple IDPSyncRun for a given list of IDs
func (s *Storage) GetIDPSyncRunsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]IDPSyncRun, error) {
	items := make([]IDPSyncRun, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getIDPSyncRunsHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getIDPSyncRunsHelperForIDs loads multiple IDPSyncRun for a given list of IDs from the DB
func (s *Storage) getIDPSyncRunsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]IDPSyncRun, error) {
	const q = "SELECT id, updated, deleted, type, active_provider_id, follower_provider_ids, since, until, error, total_records, failed_records, warning_records, created FROM idp_sync_runs WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []IDPSyncRun
	if err := s.db.SelectContextWithDirty(ctx, "GetIDPSyncRunsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested IDPSyncRuns  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListIDPSyncRunsPaginated loads a paginated list of IDPSyncRuns for the specified paginator settings
func (s *Storage) ListIDPSyncRunsPaginated(ctx context.Context, p pagination.Paginator) ([]IDPSyncRun, *pagination.ResponseFields, error) {
	return s.listInnerIDPSyncRunsPaginated(ctx, p, false)
}

// listInnerIDPSyncRunsPaginated loads a paginated list of IDPSyncRuns for the specified paginator settings
func (s *Storage) listInnerIDPSyncRunsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]IDPSyncRun, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, type, active_provider_id, follower_provider_ids, since, until, error, total_records, failed_records, warning_records, created FROM (SELECT id, updated, deleted, type, active_provider_id, follower_provider_ids, since, until, error, total_records, failed_records, warning_records, created FROM idp_sync_runs WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []IDPSyncRun
	if err := s.db.SelectContext(ctx, "ListIDPSyncRunsPaginated", &objsDB, q, queryFields...); err != nil {
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

// SaveIDPSyncRun saves a IDPSyncRun
func (s *Storage) SaveIDPSyncRun(ctx context.Context, obj *IDPSyncRun) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerIDPSyncRun(ctx, obj))
}

// SaveIDPSyncRun saves a IDPSyncRun
func (s *Storage) saveInnerIDPSyncRun(ctx context.Context, obj *IDPSyncRun) error {
	const q = "INSERT INTO idp_sync_runs (id, updated, deleted, type, active_provider_id, follower_provider_ids, since, until, error, total_records, failed_records, warning_records) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, type = $3, active_provider_id = $4, follower_provider_ids = $5, since = $6, until = $7, error = $8, total_records = $9, failed_records = $10, warning_records = $11 WHERE (idp_sync_runs.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveIDPSyncRun", obj, q, obj.ID, obj.Deleted, obj.Type, obj.ActiveProviderID, obj.FollowerProviderIDs, obj.Since, obj.Until, obj.Error, obj.TotalRecords, obj.FailedRecords, obj.WarningRecords); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "IDPSyncRun %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteIDPSyncRun soft-deletes a IDPSyncRun which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteIDPSyncRun(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerIDPSyncRun(ctx, objID, false))
}

// deleteInnerIDPSyncRun soft-deletes a IDPSyncRun which is currently alive
func (s *Storage) deleteInnerIDPSyncRun(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE idp_sync_runs SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteIDPSyncRun", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting IDPSyncRun %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "IDPSyncRun %v not found", objID)
	}
	return nil
}
