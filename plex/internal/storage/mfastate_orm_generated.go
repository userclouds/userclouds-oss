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

// IsMFAStateSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsMFAStateSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM mfa_states WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsMFAStateSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetMFAState loads a MFAState by ID
func (s *Storage) GetMFAState(ctx context.Context, id uuid.UUID) (*MFAState, error) {
	const q = "SELECT id, updated, deleted, session_id, token, provider, channel_id, supported_channels, purpose, challenge_state, evaluate_supported_channels, created FROM mfa_states WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj MFAState
	if err := s.db.GetContext(ctx, "GetMFAState", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "MFAState %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetMFAStateSoftDeleted loads a MFAState by ID iff it's soft-deleted
func (s *Storage) GetMFAStateSoftDeleted(ctx context.Context, id uuid.UUID) (*MFAState, error) {
	const q = "SELECT id, updated, deleted, session_id, token, provider, channel_id, supported_channels, purpose, challenge_state, evaluate_supported_channels, created FROM mfa_states WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj MFAState
	if err := s.db.GetContext(ctx, "GetMFAStateSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted MFAState %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetMFAStatesForIDs loads multiple MFAState for a given list of IDs
func (s *Storage) GetMFAStatesForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]MFAState, error) {
	items := make([]MFAState, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getMFAStatesHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getMFAStatesHelperForIDs loads multiple MFAState for a given list of IDs from the DB
func (s *Storage) getMFAStatesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]MFAState, error) {
	const q = "SELECT id, updated, deleted, session_id, token, provider, channel_id, supported_channels, purpose, challenge_state, evaluate_supported_channels, created FROM mfa_states WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []MFAState
	if err := s.db.SelectContextWithDirty(ctx, "GetMFAStatesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested MFAStates  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListMFAStatesPaginated loads a paginated list of MFAStates for the specified paginator settings
func (s *Storage) ListMFAStatesPaginated(ctx context.Context, p pagination.Paginator) ([]MFAState, *pagination.ResponseFields, error) {
	return s.listInnerMFAStatesPaginated(ctx, p, false)
}

// listInnerMFAStatesPaginated loads a paginated list of MFAStates for the specified paginator settings
func (s *Storage) listInnerMFAStatesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]MFAState, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, session_id, token, provider, channel_id, supported_channels, purpose, challenge_state, evaluate_supported_channels, created FROM (SELECT id, updated, deleted, session_id, token, provider, channel_id, supported_channels, purpose, challenge_state, evaluate_supported_channels, created FROM mfa_states WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []MFAState
	if err := s.db.SelectContext(ctx, "ListMFAStatesPaginated", &objsDB, q, queryFields...); err != nil {
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

// SaveMFAState saves a MFAState
func (s *Storage) SaveMFAState(ctx context.Context, obj *MFAState) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerMFAState(ctx, obj))
}

// SaveMFAState saves a MFAState
func (s *Storage) saveInnerMFAState(ctx context.Context, obj *MFAState) error {
	const q = "INSERT INTO mfa_states (id, updated, deleted, session_id, token, provider, channel_id, supported_channels, purpose, challenge_state, evaluate_supported_channels) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9, $10) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, session_id = $3, token = $4, provider = $5, channel_id = $6, supported_channels = $7, purpose = $8, challenge_state = $9, evaluate_supported_channels = $10 WHERE (mfa_states.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveMFAState", obj, q, obj.ID, obj.Deleted, obj.SessionID, obj.Token, obj.Provider, obj.ChannelID, obj.SupportedChannels, obj.Purpose, obj.ChallengeState, obj.EvaluateSupportedChannels); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "MFAState %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteMFAState soft-deletes a MFAState which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteMFAState(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerMFAState(ctx, objID, false))
}

// deleteInnerMFAState soft-deletes a MFAState which is currently alive
func (s *Storage) deleteInnerMFAState(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE mfa_states SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteMFAState", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting MFAState %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "MFAState %v not found", objID)
	}
	return nil
}
