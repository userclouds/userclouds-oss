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

// IsOIDCAuthnSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsOIDCAuthnSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM authns_social WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsOIDCAuthnSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetOIDCAuthn loads a OIDCAuthn by ID
func (s *Storage) GetOIDCAuthn(ctx context.Context, id uuid.UUID) (*OIDCAuthn, error) {
	const q = "SELECT id, updated, deleted, user_id, type, oidc_issuer_url, oidc_sub, created FROM authns_social WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj OIDCAuthn
	if err := s.db.GetContext(ctx, "GetOIDCAuthn", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "OIDCAuthn %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetOIDCAuthnSoftDeleted loads a OIDCAuthn by ID iff it's soft-deleted
func (s *Storage) GetOIDCAuthnSoftDeleted(ctx context.Context, id uuid.UUID) (*OIDCAuthn, error) {
	const q = "SELECT id, updated, deleted, user_id, type, oidc_issuer_url, oidc_sub, created FROM authns_social WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj OIDCAuthn
	if err := s.db.GetContext(ctx, "GetOIDCAuthnSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted OIDCAuthn %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetOIDCAuthnsForIDs loads multiple OIDCAuthn for a given list of IDs
func (s *Storage) GetOIDCAuthnsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]OIDCAuthn, error) {
	items := make([]OIDCAuthn, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getOIDCAuthnsHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getOIDCAuthnsHelperForIDs loads multiple OIDCAuthn for a given list of IDs from the DB
func (s *Storage) getOIDCAuthnsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]OIDCAuthn, error) {
	const q = "SELECT id, updated, deleted, user_id, type, oidc_issuer_url, oidc_sub, created FROM authns_social WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []OIDCAuthn
	if err := s.db.SelectContextWithDirty(ctx, "GetOIDCAuthnsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested OIDCAuthns  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListOIDCAuthnsPaginated loads a paginated list of OIDCAuthns for the specified paginator settings
func (s *Storage) ListOIDCAuthnsPaginated(ctx context.Context, p pagination.Paginator) ([]OIDCAuthn, *pagination.ResponseFields, error) {
	return s.listInnerOIDCAuthnsPaginated(ctx, p, false)
}

// listInnerOIDCAuthnsPaginated loads a paginated list of OIDCAuthns for the specified paginator settings
func (s *Storage) listInnerOIDCAuthnsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]OIDCAuthn, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, user_id, type, oidc_issuer_url, oidc_sub, created FROM (SELECT id, updated, deleted, user_id, type, oidc_issuer_url, oidc_sub, created FROM authns_social WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []OIDCAuthn
	if err := s.db.SelectContext(ctx, "ListOIDCAuthnsPaginated", &objsDB, q, queryFields...); err != nil {
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

// ListOIDCAuthnsForUserID loads the list of OIDCAuthns with a matching UserID field
func (s *Storage) ListOIDCAuthnsForUserID(ctx context.Context, userID uuid.UUID) ([]OIDCAuthn, error) {
	const q = "SELECT id, updated, deleted, user_id, type, oidc_issuer_url, oidc_sub, created FROM authns_social WHERE user_id=$1 AND deleted='0001-01-01 00:00:00';"
	var objs []OIDCAuthn
	if err := s.db.SelectContext(ctx, "ListOIDCAuthnsForUserID", &objs, q, userID); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return objs, nil
}

// SaveOIDCAuthn saves a OIDCAuthn
func (s *Storage) SaveOIDCAuthn(ctx context.Context, obj *OIDCAuthn) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerOIDCAuthn(ctx, obj))
}

// SaveOIDCAuthn saves a OIDCAuthn
func (s *Storage) saveInnerOIDCAuthn(ctx context.Context, obj *OIDCAuthn) error {
	const q = "INSERT INTO authns_social (id, updated, deleted, user_id, type, oidc_issuer_url, oidc_sub) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, user_id = $3, type = $4, oidc_issuer_url = $5, oidc_sub = $6 WHERE (authns_social.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveOIDCAuthn", obj, q, obj.ID, obj.Deleted, obj.UserID, obj.Type, obj.OIDCIssuerURL, obj.OIDCSubject); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "OIDCAuthn %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteOIDCAuthn soft-deletes a OIDCAuthn which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteOIDCAuthn(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerOIDCAuthn(ctx, objID, false))
}

// deleteInnerOIDCAuthn soft-deletes a OIDCAuthn which is currently alive
func (s *Storage) deleteInnerOIDCAuthn(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE authns_social SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteOIDCAuthn", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting OIDCAuthn %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "OIDCAuthn %v not found", objID)
	}
	return nil
}
