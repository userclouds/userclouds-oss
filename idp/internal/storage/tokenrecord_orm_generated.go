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

// IsTokenRecordSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsTokenRecordSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM token_records WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsTokenRecordSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetTokenRecord loads a TokenRecord by ID
func (s *Storage) GetTokenRecord(ctx context.Context, id uuid.UUID) (*TokenRecord, error) {
	const q = "SELECT id, updated, deleted, data, token, user_id, column_id, transformer_id, transformer_version, access_policy_id, created FROM token_records WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj TokenRecord
	if err := s.db.GetContext(ctx, "GetTokenRecord", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "TokenRecord %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetTokenRecordSoftDeleted loads a TokenRecord by ID iff it's soft-deleted
func (s *Storage) GetTokenRecordSoftDeleted(ctx context.Context, id uuid.UUID) (*TokenRecord, error) {
	const q = "SELECT id, updated, deleted, data, token, user_id, column_id, transformer_id, transformer_version, access_policy_id, created FROM token_records WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj TokenRecord
	if err := s.db.GetContext(ctx, "GetTokenRecordSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted TokenRecord %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetTokenRecordsForIDs loads multiple TokenRecord for a given list of IDs
func (s *Storage) GetTokenRecordsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]TokenRecord, error) {
	items := make([]TokenRecord, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getTokenRecordsHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getTokenRecordsHelperForIDs loads multiple TokenRecord for a given list of IDs from the DB
func (s *Storage) getTokenRecordsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]TokenRecord, error) {
	const q = "SELECT id, updated, deleted, data, token, user_id, column_id, transformer_id, transformer_version, access_policy_id, created FROM token_records WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []TokenRecord
	if err := s.db.SelectContextWithDirty(ctx, "GetTokenRecordsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested TokenRecords  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListTokenRecordsPaginated loads a paginated list of TokenRecords for the specified paginator settings
func (s *Storage) ListTokenRecordsPaginated(ctx context.Context, p pagination.Paginator) ([]TokenRecord, *pagination.ResponseFields, error) {
	return s.listInnerTokenRecordsPaginated(ctx, p, false)
}

// listInnerTokenRecordsPaginated loads a paginated list of TokenRecords for the specified paginator settings
func (s *Storage) listInnerTokenRecordsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]TokenRecord, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, data, token, user_id, column_id, transformer_id, transformer_version, access_policy_id, created FROM (SELECT id, updated, deleted, data, token, user_id, column_id, transformer_id, transformer_version, access_policy_id, created FROM token_records WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []TokenRecord
	if err := s.db.SelectContext(ctx, "ListTokenRecordsPaginated", &objsDB, q, queryFields...); err != nil {
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

// ListTokenRecordsForUserID loads the list of TokenRecords with a matching UserID field
func (s *Storage) ListTokenRecordsForUserID(ctx context.Context, userID uuid.UUID) ([]TokenRecord, error) {
	const q = "SELECT id, updated, deleted, data, token, user_id, column_id, transformer_id, transformer_version, access_policy_id, created FROM token_records WHERE user_id=$1 AND deleted='0001-01-01 00:00:00';"
	var objs []TokenRecord
	if err := s.db.SelectContext(ctx, "ListTokenRecordsForUserID", &objs, q, userID); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return objs, nil
}

// SaveTokenRecord saves a TokenRecord
func (s *Storage) SaveTokenRecord(ctx context.Context, obj *TokenRecord) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerTokenRecord(ctx, obj))
}

// SaveTokenRecord saves a TokenRecord
func (s *Storage) saveInnerTokenRecord(ctx context.Context, obj *TokenRecord) error {
	const q = "INSERT INTO token_records (id, updated, deleted, data, token, user_id, column_id, transformer_id, transformer_version, access_policy_id) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, data = $3, token = $4, user_id = $5, column_id = $6, transformer_id = $7, transformer_version = $8, access_policy_id = $9 WHERE (token_records.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveTokenRecord", obj, q, obj.ID, obj.Deleted, obj.Data, obj.Token, obj.UserID, obj.ColumnID, obj.TransformerID, obj.TransformerVersion, obj.AccessPolicyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "TokenRecord %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteTokenRecord soft-deletes a TokenRecord which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteTokenRecord(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerTokenRecord(ctx, objID, false))
}

// deleteInnerTokenRecord soft-deletes a TokenRecord which is currently alive
func (s *Storage) deleteInnerTokenRecord(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE token_records SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteTokenRecord", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting TokenRecord %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "TokenRecord %v not found", objID)
	}
	return nil
}
