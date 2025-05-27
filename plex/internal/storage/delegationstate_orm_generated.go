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

// IsDelegationStateSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsDelegationStateSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM delegation_states WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsDelegationStateSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetDelegationState loads a DelegationState by ID
func (s *Storage) GetDelegationState(ctx context.Context, id uuid.UUID) (*DelegationState, error) {
	const q = "SELECT id, updated, deleted, authenticated_user_id, created FROM delegation_states WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj DelegationState
	if err := s.db.GetContext(ctx, "GetDelegationState", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "DelegationState %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetDelegationStateSoftDeleted loads a DelegationState by ID iff it's soft-deleted
func (s *Storage) GetDelegationStateSoftDeleted(ctx context.Context, id uuid.UUID) (*DelegationState, error) {
	const q = "SELECT id, updated, deleted, authenticated_user_id, created FROM delegation_states WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj DelegationState
	if err := s.db.GetContext(ctx, "GetDelegationStateSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted DelegationState %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetDelegationStatesForIDs loads multiple DelegationState for a given list of IDs
func (s *Storage) GetDelegationStatesForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]DelegationState, error) {
	items := make([]DelegationState, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getDelegationStatesHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getDelegationStatesHelperForIDs loads multiple DelegationState for a given list of IDs from the DB
func (s *Storage) getDelegationStatesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]DelegationState, error) {
	const q = "SELECT id, updated, deleted, authenticated_user_id, created FROM delegation_states WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []DelegationState
	if err := s.db.SelectContextWithDirty(ctx, "GetDelegationStatesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested DelegationStates  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListDelegationStatesPaginated loads a paginated list of DelegationStates for the specified paginator settings
func (s *Storage) ListDelegationStatesPaginated(ctx context.Context, p pagination.Paginator) ([]DelegationState, *pagination.ResponseFields, error) {
	return s.listInnerDelegationStatesPaginated(ctx, p, false)
}

// listInnerDelegationStatesPaginated loads a paginated list of DelegationStates for the specified paginator settings
func (s *Storage) listInnerDelegationStatesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]DelegationState, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, authenticated_user_id, created FROM (SELECT id, updated, deleted, authenticated_user_id, created FROM delegation_states WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []DelegationState
	if err := s.db.SelectContext(ctx, "ListDelegationStatesPaginated", &objsDB, q, queryFields...); err != nil {
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

// SaveDelegationState saves a DelegationState
func (s *Storage) SaveDelegationState(ctx context.Context, obj *DelegationState) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerDelegationState(ctx, obj))
}

// SaveDelegationState saves a DelegationState
func (s *Storage) saveInnerDelegationState(ctx context.Context, obj *DelegationState) error {
	const q = "INSERT INTO delegation_states (id, updated, deleted, authenticated_user_id) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, authenticated_user_id = $3 WHERE (delegation_states.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveDelegationState", obj, q, obj.ID, obj.Deleted, obj.AuthenticatedUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "DelegationState %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteDelegationState soft-deletes a DelegationState which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteDelegationState(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerDelegationState(ctx, objID, false))
}

// deleteInnerDelegationState soft-deletes a DelegationState which is currently alive
func (s *Storage) deleteInnerDelegationState(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE delegation_states SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteDelegationState", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting DelegationState %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "DelegationState %v not found", objID)
	}
	return nil
}
