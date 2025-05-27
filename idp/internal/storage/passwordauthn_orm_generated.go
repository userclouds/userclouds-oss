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

// IsPasswordAuthnSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsPasswordAuthnSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM authns_password WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsPasswordAuthnSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetPasswordAuthn loads a PasswordAuthn by ID
func (s *Storage) GetPasswordAuthn(ctx context.Context, id uuid.UUID) (*PasswordAuthn, error) {
	const q = "SELECT id, updated, deleted, user_id, username, password, created FROM authns_password WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj PasswordAuthn
	if err := s.db.GetContext(ctx, "GetPasswordAuthn", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "PasswordAuthn %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetPasswordAuthnSoftDeleted loads a PasswordAuthn by ID iff it's soft-deleted
func (s *Storage) GetPasswordAuthnSoftDeleted(ctx context.Context, id uuid.UUID) (*PasswordAuthn, error) {
	const q = "SELECT id, updated, deleted, user_id, username, password, created FROM authns_password WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj PasswordAuthn
	if err := s.db.GetContext(ctx, "GetPasswordAuthnSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted PasswordAuthn %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetPasswordAuthnsForIDs loads multiple PasswordAuthn for a given list of IDs
func (s *Storage) GetPasswordAuthnsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]PasswordAuthn, error) {
	items := make([]PasswordAuthn, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getPasswordAuthnsHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getPasswordAuthnsHelperForIDs loads multiple PasswordAuthn for a given list of IDs from the DB
func (s *Storage) getPasswordAuthnsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]PasswordAuthn, error) {
	const q = "SELECT id, updated, deleted, user_id, username, password, created FROM authns_password WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []PasswordAuthn
	if err := s.db.SelectContextWithDirty(ctx, "GetPasswordAuthnsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested PasswordAuthns  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListPasswordAuthnsPaginated loads a paginated list of PasswordAuthns for the specified paginator settings
func (s *Storage) ListPasswordAuthnsPaginated(ctx context.Context, p pagination.Paginator) ([]PasswordAuthn, *pagination.ResponseFields, error) {
	return s.listInnerPasswordAuthnsPaginated(ctx, p, false)
}

// listInnerPasswordAuthnsPaginated loads a paginated list of PasswordAuthns for the specified paginator settings
func (s *Storage) listInnerPasswordAuthnsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]PasswordAuthn, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, user_id, username, password, created FROM (SELECT id, updated, deleted, user_id, username, password, created FROM authns_password WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []PasswordAuthn
	if err := s.db.SelectContext(ctx, "ListPasswordAuthnsPaginated", &objsDB, q, queryFields...); err != nil {
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

// ListPasswordAuthnsForUserID loads the list of PasswordAuthns with a matching UserID field
func (s *Storage) ListPasswordAuthnsForUserID(ctx context.Context, userID uuid.UUID) ([]PasswordAuthn, error) {
	const q = "SELECT id, updated, deleted, user_id, username, password, created FROM authns_password WHERE user_id=$1 AND deleted='0001-01-01 00:00:00';"
	var objs []PasswordAuthn
	if err := s.db.SelectContext(ctx, "ListPasswordAuthnsForUserID", &objs, q, userID); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return objs, nil
}

// SavePasswordAuthn saves a PasswordAuthn
func (s *Storage) SavePasswordAuthn(ctx context.Context, obj *PasswordAuthn) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerPasswordAuthn(ctx, obj))
}

// SavePasswordAuthn saves a PasswordAuthn
func (s *Storage) saveInnerPasswordAuthn(ctx context.Context, obj *PasswordAuthn) error {
	const q = "INSERT INTO authns_password (id, updated, deleted, user_id, username, password) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, user_id = $3, username = $4, password = $5 WHERE (authns_password.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SavePasswordAuthn", obj, q, obj.ID, obj.Deleted, obj.UserID, obj.Username, obj.Password); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "PasswordAuthn %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeletePasswordAuthn soft-deletes a PasswordAuthn which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeletePasswordAuthn(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerPasswordAuthn(ctx, objID, false))
}

// deleteInnerPasswordAuthn soft-deletes a PasswordAuthn which is currently alive
func (s *Storage) deleteInnerPasswordAuthn(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE authns_password SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeletePasswordAuthn", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting PasswordAuthn %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "PasswordAuthn %v not found", objID)
	}
	return nil
}
