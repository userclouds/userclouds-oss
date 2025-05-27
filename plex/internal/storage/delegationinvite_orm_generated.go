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

// IsDelegationInviteSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsDelegationInviteSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM delegation_invites WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsDelegationInviteSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetDelegationInvite loads a DelegationInvite by ID
func (s *Storage) GetDelegationInvite(ctx context.Context, id uuid.UUID) (*DelegationInvite, error) {
	const q = "SELECT id, updated, deleted, client_id, invited_to_account_id, created FROM delegation_invites WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj DelegationInvite
	if err := s.db.GetContext(ctx, "GetDelegationInvite", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "DelegationInvite %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetDelegationInviteSoftDeleted loads a DelegationInvite by ID iff it's soft-deleted
func (s *Storage) GetDelegationInviteSoftDeleted(ctx context.Context, id uuid.UUID) (*DelegationInvite, error) {
	const q = "SELECT id, updated, deleted, client_id, invited_to_account_id, created FROM delegation_invites WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj DelegationInvite
	if err := s.db.GetContext(ctx, "GetDelegationInviteSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted DelegationInvite %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetDelegationInvitesForIDs loads multiple DelegationInvite for a given list of IDs
func (s *Storage) GetDelegationInvitesForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]DelegationInvite, error) {
	items := make([]DelegationInvite, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getDelegationInvitesHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getDelegationInvitesHelperForIDs loads multiple DelegationInvite for a given list of IDs from the DB
func (s *Storage) getDelegationInvitesHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]DelegationInvite, error) {
	const q = "SELECT id, updated, deleted, client_id, invited_to_account_id, created FROM delegation_invites WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []DelegationInvite
	if err := s.db.SelectContextWithDirty(ctx, "GetDelegationInvitesForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested DelegationInvites  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListDelegationInvitesPaginated loads a paginated list of DelegationInvites for the specified paginator settings
func (s *Storage) ListDelegationInvitesPaginated(ctx context.Context, p pagination.Paginator) ([]DelegationInvite, *pagination.ResponseFields, error) {
	return s.listInnerDelegationInvitesPaginated(ctx, p, false)
}

// listInnerDelegationInvitesPaginated loads a paginated list of DelegationInvites for the specified paginator settings
func (s *Storage) listInnerDelegationInvitesPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]DelegationInvite, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, client_id, invited_to_account_id, created FROM (SELECT id, updated, deleted, client_id, invited_to_account_id, created FROM delegation_invites WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []DelegationInvite
	if err := s.db.SelectContext(ctx, "ListDelegationInvitesPaginated", &objsDB, q, queryFields...); err != nil {
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

// SaveDelegationInvite saves a DelegationInvite
func (s *Storage) SaveDelegationInvite(ctx context.Context, obj *DelegationInvite) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerDelegationInvite(ctx, obj))
}

// SaveDelegationInvite saves a DelegationInvite
func (s *Storage) saveInnerDelegationInvite(ctx context.Context, obj *DelegationInvite) error {
	const q = "INSERT INTO delegation_invites (id, updated, deleted, client_id, invited_to_account_id) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, client_id = $3, invited_to_account_id = $4 WHERE (delegation_invites.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveDelegationInvite", obj, q, obj.ID, obj.Deleted, obj.ClientID, obj.InvitedToAccountID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "DelegationInvite %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteDelegationInvite soft-deletes a DelegationInvite which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteDelegationInvite(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerDelegationInvite(ctx, objID, false))
}

// deleteInnerDelegationInvite soft-deletes a DelegationInvite which is currently alive
func (s *Storage) deleteInnerDelegationInvite(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE delegation_invites SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteDelegationInvite", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting DelegationInvite %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "DelegationInvite %v not found", objID)
	}
	return nil
}
