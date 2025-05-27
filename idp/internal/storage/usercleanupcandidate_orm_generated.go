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

// IsUserCleanupCandidateSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *UserStorage) IsUserCleanupCandidateSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM user_cleanup_candidates WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsUserCleanupCandidateSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetUserCleanupCandidate loads a UserCleanupCandidate by ID
func (s *UserStorage) GetUserCleanupCandidate(ctx context.Context, id uuid.UUID) (*UserCleanupCandidate, error) {
	const q = "SELECT id, updated, deleted, user_id, cleanup_reason, created FROM user_cleanup_candidates WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj UserCleanupCandidate
	if err := s.db.GetContext(ctx, "GetUserCleanupCandidate", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "UserCleanupCandidate %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetUserCleanupCandidateSoftDeleted loads a UserCleanupCandidate by ID iff it's soft-deleted
func (s *UserStorage) GetUserCleanupCandidateSoftDeleted(ctx context.Context, id uuid.UUID) (*UserCleanupCandidate, error) {
	const q = "SELECT id, updated, deleted, user_id, cleanup_reason, created FROM user_cleanup_candidates WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj UserCleanupCandidate
	if err := s.db.GetContext(ctx, "GetUserCleanupCandidateSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted UserCleanupCandidate %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetUserCleanupCandidatesForIDs loads multiple UserCleanupCandidate for a given list of IDs
func (s *UserStorage) GetUserCleanupCandidatesForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]UserCleanupCandidate, error) {
	items := make([]UserCleanupCandidate, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getUserCleanupCandidatesHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getUserCleanupCandidatesHelperForIDs loads multiple UserCleanupCandidate for a given list of IDs from the DB
func (s *UserStorage) getUserCleanupCandidatesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]UserCleanupCandidate, error) {
	const q = "SELECT id, updated, deleted, user_id, cleanup_reason, created FROM user_cleanup_candidates WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []UserCleanupCandidate
	if err := s.db.SelectContextWithDirty(ctx, "GetUserCleanupCandidatesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested UserCleanupCandidates  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListUserCleanupCandidatesPaginated loads a paginated list of UserCleanupCandidates for the specified paginator settings
func (s *UserStorage) ListUserCleanupCandidatesPaginated(ctx context.Context, p pagination.Paginator) ([]UserCleanupCandidate, *pagination.ResponseFields, error) {
	return s.listInnerUserCleanupCandidatesPaginated(ctx, p, false)
}

// listInnerUserCleanupCandidatesPaginated loads a paginated list of UserCleanupCandidates for the specified paginator settings
func (s *UserStorage) listInnerUserCleanupCandidatesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]UserCleanupCandidate, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, user_id, cleanup_reason, created FROM (SELECT id, updated, deleted, user_id, cleanup_reason, created FROM user_cleanup_candidates WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []UserCleanupCandidate
	if err := s.db.SelectContext(ctx, "ListUserCleanupCandidatesPaginated", &objsDB, q, queryFields...); err != nil {
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

// ListUserCleanupCandidatesForUserID loads the list of UserCleanupCandidates with a matching UserID field
func (s *UserStorage) ListUserCleanupCandidatesForUserID(ctx context.Context, userID uuid.UUID) ([]UserCleanupCandidate, error) {
	const q = "SELECT id, updated, deleted, user_id, cleanup_reason, created FROM user_cleanup_candidates WHERE user_id=$1 AND deleted='0001-01-01 00:00:00';"
	var objs []UserCleanupCandidate
	if err := s.db.SelectContext(ctx, "ListUserCleanupCandidatesForUserID", &objs, q, userID); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return objs, nil
}

// SaveUserCleanupCandidate saves a UserCleanupCandidate
func (s *UserStorage) SaveUserCleanupCandidate(ctx context.Context, obj *UserCleanupCandidate) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerUserCleanupCandidate(ctx, obj))
}

// SaveUserCleanupCandidate saves a UserCleanupCandidate
func (s *UserStorage) saveInnerUserCleanupCandidate(ctx context.Context, obj *UserCleanupCandidate) error {
	const q = "INSERT INTO user_cleanup_candidates (id, updated, deleted, user_id, cleanup_reason) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, user_id = $3, cleanup_reason = $4 WHERE (user_cleanup_candidates.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveUserCleanupCandidate", obj, q, obj.ID, obj.Deleted, obj.UserID, obj.CleanupReason); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "UserCleanupCandidate %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteUserCleanupCandidate soft-deletes a UserCleanupCandidate which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *UserStorage) DeleteUserCleanupCandidate(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerUserCleanupCandidate(ctx, objID, false))
}

// deleteInnerUserCleanupCandidate soft-deletes a UserCleanupCandidate which is currently alive
func (s *UserStorage) deleteInnerUserCleanupCandidate(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE user_cleanup_candidates SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteUserCleanupCandidate", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting UserCleanupCandidate %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "UserCleanupCandidate %v not found", objID)
	}
	return nil
}
