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

// IsOTPStateSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsOTPStateSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM otp_states WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsOTPStateSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetOTPState loads a OTPState by ID
func (s *Storage) GetOTPState(ctx context.Context, id uuid.UUID) (*OTPState, error) {
	const q = "SELECT id, updated, deleted, session_id, user_id, email, code, expires, used, purpose, created FROM otp_states WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj OTPState
	if err := s.db.GetContext(ctx, "GetOTPState", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "OTPState %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetOTPStateSoftDeleted loads a OTPState by ID iff it's soft-deleted
func (s *Storage) GetOTPStateSoftDeleted(ctx context.Context, id uuid.UUID) (*OTPState, error) {
	const q = "SELECT id, updated, deleted, session_id, user_id, email, code, expires, used, purpose, created FROM otp_states WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj OTPState
	if err := s.db.GetContext(ctx, "GetOTPStateSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted OTPState %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetOTPStatesForIDs loads multiple OTPState for a given list of IDs
func (s *Storage) GetOTPStatesForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]OTPState, error) {
	items := make([]OTPState, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getOTPStatesHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getOTPStatesHelperForIDs loads multiple OTPState for a given list of IDs from the DB
func (s *Storage) getOTPStatesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]OTPState, error) {
	const q = "SELECT id, updated, deleted, session_id, user_id, email, code, expires, used, purpose, created FROM otp_states WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []OTPState
	if err := s.db.SelectContextWithDirty(ctx, "GetOTPStatesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested OTPStates  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListOTPStatesPaginated loads a paginated list of OTPStates for the specified paginator settings
func (s *Storage) ListOTPStatesPaginated(ctx context.Context, p pagination.Paginator) ([]OTPState, *pagination.ResponseFields, error) {
	return s.listInnerOTPStatesPaginated(ctx, p, false)
}

// listInnerOTPStatesPaginated loads a paginated list of OTPStates for the specified paginator settings
func (s *Storage) listInnerOTPStatesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]OTPState, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, session_id, user_id, email, code, expires, used, purpose, created FROM (SELECT id, updated, deleted, session_id, user_id, email, code, expires, used, purpose, created FROM otp_states WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []OTPState
	if err := s.db.SelectContext(ctx, "ListOTPStatesPaginated", &objsDB, q, queryFields...); err != nil {
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

// SaveOTPState saves a OTPState
func (s *Storage) SaveOTPState(ctx context.Context, obj *OTPState) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerOTPState(ctx, obj))
}

// SaveOTPState saves a OTPState
func (s *Storage) saveInnerOTPState(ctx context.Context, obj *OTPState) error {
	const q = "INSERT INTO otp_states (id, updated, deleted, session_id, user_id, email, code, expires, used, purpose) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, session_id = $3, user_id = $4, email = $5, code = $6, expires = $7, used = $8, purpose = $9 WHERE (otp_states.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveOTPState", obj, q, obj.ID, obj.Deleted, obj.SessionID, obj.UserID, obj.Email, obj.Code, obj.Expires, obj.Used, obj.Purpose); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "OTPState %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteOTPState soft-deletes a OTPState which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteOTPState(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerOTPState(ctx, objID, false))
}

// deleteInnerOTPState soft-deletes a OTPState which is currently alive
func (s *Storage) deleteInnerOTPState(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE otp_states SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteOTPState", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting OTPState %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "OTPState %v not found", objID)
	}
	return nil
}
