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

// IsSecretSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsSecretSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM policy_secrets WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsSecretSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetSecret loads a Secret by ID
func (s *Storage) GetSecret(ctx context.Context, id uuid.UUID) (*Secret, error) {
	const q = "SELECT id, updated, deleted, name, value, created FROM policy_secrets WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj Secret
	if err := s.db.GetContext(ctx, "GetSecret", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "Secret %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetSecretSoftDeleted loads a Secret by ID iff it's soft-deleted
func (s *Storage) GetSecretSoftDeleted(ctx context.Context, id uuid.UUID) (*Secret, error) {
	const q = "SELECT id, updated, deleted, name, value, created FROM policy_secrets WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj Secret
	if err := s.db.GetContext(ctx, "GetSecretSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted Secret %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetSecretsForIDs loads multiple Secret for a given list of IDs
func (s *Storage) GetSecretsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]Secret, error) {
	items := make([]Secret, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getSecretsHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getSecretsHelperForIDs loads multiple Secret for a given list of IDs from the DB
func (s *Storage) getSecretsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]Secret, error) {
	const q = "SELECT id, updated, deleted, name, value, created FROM policy_secrets WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []Secret
	if err := s.db.SelectContextWithDirty(ctx, "GetSecretsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested Secrets  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListSecretsPaginated loads a paginated list of Secrets for the specified paginator settings
func (s *Storage) ListSecretsPaginated(ctx context.Context, p pagination.Paginator) ([]Secret, *pagination.ResponseFields, error) {
	return s.listInnerSecretsPaginated(ctx, p, false)
}

// listInnerSecretsPaginated loads a paginated list of Secrets for the specified paginator settings
func (s *Storage) listInnerSecretsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]Secret, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, name, value, created FROM (SELECT id, updated, deleted, name, value, created FROM policy_secrets WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []Secret
	if err := s.db.SelectContext(ctx, "ListSecretsPaginated", &objsDB, q, queryFields...); err != nil {
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

// SaveSecret saves a Secret
func (s *Storage) SaveSecret(ctx context.Context, obj *Secret) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerSecret(ctx, obj))
}

// SaveSecret saves a Secret
func (s *Storage) saveInnerSecret(ctx context.Context, obj *Secret) error {
	const q = "INSERT INTO policy_secrets (id, updated, deleted, name, value) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, name = $3, value = $4 WHERE (policy_secrets.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveSecret", obj, q, obj.ID, obj.Deleted, obj.Name, obj.Value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "Secret %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteSecret soft-deletes a Secret which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteSecret(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerSecret(ctx, objID, false))
}

// deleteInnerSecret soft-deletes a Secret which is currently alive
func (s *Storage) deleteInnerSecret(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE policy_secrets SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteSecret", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting Secret %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "Secret %v not found", objID)
	}
	return nil
}
