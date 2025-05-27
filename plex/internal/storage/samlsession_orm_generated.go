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

// IsSAMLSessionSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsSAMLSessionSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM saml_sessions WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsSAMLSessionSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetSAMLSession loads a SAMLSession by ID
func (s *Storage) GetSAMLSession(ctx context.Context, id uuid.UUID) (*SAMLSession, error) {
	const q = "SELECT id, updated, deleted, expire_time, _index, name_id, name_id_format, subject_id, groups, user_name, user_email, user_common_name, user_surname, user_given_name, user_scoped_affiliation, custom_attributes, state, relay_state, request_buffer, created FROM saml_sessions WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj SAMLSession
	if err := s.db.GetContext(ctx, "GetSAMLSession", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "SAMLSession %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetSAMLSessionSoftDeleted loads a SAMLSession by ID iff it's soft-deleted
func (s *Storage) GetSAMLSessionSoftDeleted(ctx context.Context, id uuid.UUID) (*SAMLSession, error) {
	const q = "SELECT id, updated, deleted, expire_time, _index, name_id, name_id_format, subject_id, groups, user_name, user_email, user_common_name, user_surname, user_given_name, user_scoped_affiliation, custom_attributes, state, relay_state, request_buffer, created FROM saml_sessions WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj SAMLSession
	if err := s.db.GetContext(ctx, "GetSAMLSessionSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted SAMLSession %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetSAMLSessionsForIDs loads multiple SAMLSession for a given list of IDs
func (s *Storage) GetSAMLSessionsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]SAMLSession, error) {
	items := make([]SAMLSession, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getSAMLSessionsHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getSAMLSessionsHelperForIDs loads multiple SAMLSession for a given list of IDs from the DB
func (s *Storage) getSAMLSessionsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]SAMLSession, error) {
	const q = "SELECT id, updated, deleted, expire_time, _index, name_id, name_id_format, subject_id, groups, user_name, user_email, user_common_name, user_surname, user_given_name, user_scoped_affiliation, custom_attributes, state, relay_state, request_buffer, created FROM saml_sessions WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []SAMLSession
	if err := s.db.SelectContextWithDirty(ctx, "GetSAMLSessionsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested SAMLSessions  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListSAMLSessionsPaginated loads a paginated list of SAMLSessions for the specified paginator settings
func (s *Storage) ListSAMLSessionsPaginated(ctx context.Context, p pagination.Paginator) ([]SAMLSession, *pagination.ResponseFields, error) {
	return s.listInnerSAMLSessionsPaginated(ctx, p, false)
}

// listInnerSAMLSessionsPaginated loads a paginated list of SAMLSessions for the specified paginator settings
func (s *Storage) listInnerSAMLSessionsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]SAMLSession, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, expire_time, _index, name_id, name_id_format, subject_id, groups, user_name, user_email, user_common_name, user_surname, user_given_name, user_scoped_affiliation, custom_attributes, state, relay_state, request_buffer, created FROM (SELECT id, updated, deleted, expire_time, _index, name_id, name_id_format, subject_id, groups, user_name, user_email, user_common_name, user_surname, user_given_name, user_scoped_affiliation, custom_attributes, state, relay_state, request_buffer, created FROM saml_sessions WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []SAMLSession
	if err := s.db.SelectContext(ctx, "ListSAMLSessionsPaginated", &objsDB, q, queryFields...); err != nil {
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

// SaveSAMLSession saves a SAMLSession
func (s *Storage) SaveSAMLSession(ctx context.Context, obj *SAMLSession) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerSAMLSession(ctx, obj))
}

// SaveSAMLSession saves a SAMLSession
func (s *Storage) saveInnerSAMLSession(ctx context.Context, obj *SAMLSession) error {
	const q = "INSERT INTO saml_sessions (id, updated, deleted, expire_time, _index, name_id, name_id_format, subject_id, groups, user_name, user_email, user_common_name, user_surname, user_given_name, user_scoped_affiliation, custom_attributes, state, relay_state, request_buffer) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, expire_time = $3, _index = $4, name_id = $5, name_id_format = $6, subject_id = $7, groups = $8, user_name = $9, user_email = $10, user_common_name = $11, user_surname = $12, user_given_name = $13, user_scoped_affiliation = $14, custom_attributes = $15, state = $16, relay_state = $17, request_buffer = $18 WHERE (saml_sessions.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveSAMLSession", obj, q, obj.ID, obj.Deleted, obj.ExpireTime, obj.Index, obj.NameID, obj.NameIDFormat, obj.SubjectID, obj.Groups, obj.UserName, obj.UserEmail, obj.UserCommonName, obj.UserSurname, obj.UserGivenName, obj.UserScopedAffiliation, obj.CustomAttributes, obj.State, obj.RelayState, obj.RequestBuffer); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "SAMLSession %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteSAMLSession soft-deletes a SAMLSession which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteSAMLSession(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerSAMLSession(ctx, objID, false))
}

// deleteInnerSAMLSession soft-deletes a SAMLSession which is currently alive
func (s *Storage) deleteInnerSAMLSession(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE saml_sessions SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteSAMLSession", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting SAMLSession %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "SAMLSession %v not found", objID)
	}
	return nil
}
