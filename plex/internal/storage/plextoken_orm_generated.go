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

// IsPlexTokenSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsPlexTokenSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM plex_tokens WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsPlexTokenSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetPlexToken loads a PlexToken by ID
func (s *Storage) GetPlexToken(ctx context.Context, id uuid.UUID) (*PlexToken, error) {
	const q = "SELECT id, updated, deleted, client_id, auth_code, access_token, id_token, refresh_token, idp_subject, scopes, session_id, underlying_token, created FROM plex_tokens WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj PlexToken
	if err := s.db.GetContext(ctx, "GetPlexToken", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "PlexToken %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetPlexTokenSoftDeleted loads a PlexToken by ID iff it's soft-deleted
func (s *Storage) GetPlexTokenSoftDeleted(ctx context.Context, id uuid.UUID) (*PlexToken, error) {
	const q = "SELECT id, updated, deleted, client_id, auth_code, access_token, id_token, refresh_token, idp_subject, scopes, session_id, underlying_token, created FROM plex_tokens WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj PlexToken
	if err := s.db.GetContext(ctx, "GetPlexTokenSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted PlexToken %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetPlexTokensForIDs loads multiple PlexToken for a given list of IDs
func (s *Storage) GetPlexTokensForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]PlexToken, error) {
	items := make([]PlexToken, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getPlexTokensHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getPlexTokensHelperForIDs loads multiple PlexToken for a given list of IDs from the DB
func (s *Storage) getPlexTokensHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]PlexToken, error) {
	const q = "SELECT id, updated, deleted, client_id, auth_code, access_token, id_token, refresh_token, idp_subject, scopes, session_id, underlying_token, created FROM plex_tokens WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []PlexToken
	if err := s.db.SelectContextWithDirty(ctx, "GetPlexTokensForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested PlexTokens  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListPlexTokensPaginated loads a paginated list of PlexTokens for the specified paginator settings
func (s *Storage) ListPlexTokensPaginated(ctx context.Context, p pagination.Paginator) ([]PlexToken, *pagination.ResponseFields, error) {
	return s.listInnerPlexTokensPaginated(ctx, p, false)
}

// listInnerPlexTokensPaginated loads a paginated list of PlexTokens for the specified paginator settings
func (s *Storage) listInnerPlexTokensPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]PlexToken, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, client_id, auth_code, access_token, id_token, refresh_token, idp_subject, scopes, session_id, underlying_token, created FROM (SELECT id, updated, deleted, client_id, auth_code, access_token, id_token, refresh_token, idp_subject, scopes, session_id, underlying_token, created FROM plex_tokens WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []PlexToken
	if err := s.db.SelectContext(ctx, "ListPlexTokensPaginated", &objsDB, q, queryFields...); err != nil {
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

// SavePlexToken saves a PlexToken
func (s *Storage) SavePlexToken(ctx context.Context, obj *PlexToken) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerPlexToken(ctx, obj))
}

// SavePlexToken saves a PlexToken
func (s *Storage) saveInnerPlexToken(ctx context.Context, obj *PlexToken) error {
	const q = "INSERT INTO plex_tokens (id, updated, deleted, client_id, auth_code, access_token, id_token, refresh_token, idp_subject, scopes, session_id, underlying_token) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, client_id = $3, auth_code = $4, access_token = $5, id_token = $6, refresh_token = $7, idp_subject = $8, scopes = $9, session_id = $10, underlying_token = $11 WHERE (plex_tokens.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SavePlexToken", obj, q, obj.ID, obj.Deleted, obj.ClientID, obj.AuthCode, obj.AccessToken, obj.IDToken, obj.RefreshToken, obj.IDPSubject, obj.Scopes, obj.SessionID, obj.UnderlyingToken); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "PlexToken %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}
