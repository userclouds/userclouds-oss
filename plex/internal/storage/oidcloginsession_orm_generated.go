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

// IsOIDCLoginSessionSoftDeleted returns true if the id is associated with a soft-deleted row but no undeleted rows
func (s *Storage) IsOIDCLoginSessionSoftDeleted(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = "/* lint-sql-ok */ SELECT deleted FROM oidc_login_sessions WHERE id=$1 ORDER By deleted LIMIT 1;"

	var deleted time.Time
	if err := s.db.GetContext(ctx, "IsOIDCLoginSessionSoftDeleted", &deleted, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, ucerr.Wrap(err)
	}

	return !deleted.IsZero(), nil
}

// GetOIDCLoginSession loads a OIDCLoginSession by ID
func (s *Storage) GetOIDCLoginSession(ctx context.Context, id uuid.UUID) (*OIDCLoginSession, error) {
	const q = "SELECT id, updated, deleted, client_id, response_types, redirect_uri, state, scopes, nonce, social_provider, oidc_issuer_url, mfa_state_id, mfa_channel_states, otp_state_id, pkce_state_id, delegation_state_id, plex_token_id, add_authn_provider_data, created FROM oidc_login_sessions WHERE id=$1 AND deleted='0001-01-01 00:00:00';"

	var obj OIDCLoginSession
	if err := s.db.GetContext(ctx, "GetOIDCLoginSession", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "OIDCLoginSession %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}
	return &obj, nil
}

// GetOIDCLoginSessionSoftDeleted loads a OIDCLoginSession by ID iff it's soft-deleted
func (s *Storage) GetOIDCLoginSessionSoftDeleted(ctx context.Context, id uuid.UUID) (*OIDCLoginSession, error) {
	const q = "SELECT id, updated, deleted, client_id, response_types, redirect_uri, state, scopes, nonce, social_provider, oidc_issuer_url, mfa_state_id, mfa_channel_states, otp_state_id, pkce_state_id, delegation_state_id, plex_token_id, add_authn_provider_data, created FROM oidc_login_sessions WHERE id=$1 AND deleted<>'0001-01-01 00:00:00';"

	var obj OIDCLoginSession
	if err := s.db.GetContext(ctx, "GetOIDCLoginSessionSoftDeleted", &obj, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ucerr.Friendlyf(err, "soft-deleted OIDCLoginSession %v not found", id)
		}
		return nil, ucerr.Wrap(err)
	}

	return &obj, nil
}

// GetOIDCLoginSessionsForIDs loads multiple OIDCLoginSession for a given list of IDs
func (s *Storage) GetOIDCLoginSessionsForIDs(ctx context.Context, errorOnMissing bool, ids ...uuid.UUID) ([]OIDCLoginSession, error) {
	items := make([]OIDCLoginSession, 0, len(ids))

	missed := set.NewUUIDSet(ids...) // Assume we will miss all keys, and remove from this list if we get them from the cache
	dirty := true
	if missed.Size() > 0 {
		itemsFromDB, err := s.getOIDCLoginSessionsHelperForIDs(ctx, dirty, true, missed.Items()...)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		items = append(items, itemsFromDB...)
	}

	return items, nil
}

// getOIDCLoginSessionsHelperForIDs loads multiple OIDCLoginSession for a given list of IDs from the DB
func (s *Storage) getOIDCLoginSessionsHelperForIDs(ctx context.Context, dirty bool, errorOnMissing bool, ids ...uuid.UUID) ([]OIDCLoginSession, error) {
	const q = "SELECT id, updated, deleted, client_id, response_types, redirect_uri, state, scopes, nonce, social_provider, oidc_issuer_url, mfa_state_id, mfa_channel_states, otp_state_id, pkce_state_id, delegation_state_id, plex_token_id, add_authn_provider_data, created FROM oidc_login_sessions WHERE id=ANY($1) AND deleted='0001-01-01 00:00:00';"
	var objects []OIDCLoginSession
	if err := s.db.SelectContextWithDirty(ctx, "GetOIDCLoginSessionsForIDs", &objects, q, dirty, pq.Array(ids)); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if errorOnMissing && len(ids) != len(objects) {
		requestedIDs := set.NewUUIDSet(ids...)
		loadedIDs := set.NewUUIDSet()
		for _, obj := range objects {
			loadedIDs.Insert(obj.ID)
		}
		missingIDs := requestedIDs.Difference(loadedIDs)
		return nil, ucerr.Friendlyf(nil, "Not all requested OIDCLoginSessions  were loaded. requested: %v loaded: %v missing: [%v]", len(ids), len(objects), missingIDs)
	}
	return objects, nil
}

// ListOIDCLoginSessionsPaginated loads a paginated list of OIDCLoginSessions for the specified paginator settings
func (s *Storage) ListOIDCLoginSessionsPaginated(ctx context.Context, p pagination.Paginator) ([]OIDCLoginSession, *pagination.ResponseFields, error) {
	return s.listInnerOIDCLoginSessionsPaginated(ctx, p, false)
}

// listInnerOIDCLoginSessionsPaginated loads a paginated list of OIDCLoginSessions for the specified paginator settings
func (s *Storage) listInnerOIDCLoginSessionsPaginated(ctx context.Context, p pagination.Paginator, forceDBRead bool) ([]OIDCLoginSession, *pagination.ResponseFields, error) {
	queryFields, err := p.GetQueryFields()
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// the inner query requires an alias for postgres, so we always call it tmp
	// the outer query is just to reverse the order of the results in the case of paging backwards with forward sort
	q := fmt.Sprintf("SELECT id, updated, deleted, client_id, response_types, redirect_uri, state, scopes, nonce, social_provider, oidc_issuer_url, mfa_state_id, mfa_channel_states, otp_state_id, pkce_state_id, delegation_state_id, plex_token_id, add_authn_provider_data, created FROM (SELECT id, updated, deleted, client_id, response_types, redirect_uri, state, scopes, nonce, social_provider, oidc_issuer_url, mfa_state_id, mfa_channel_states, otp_state_id, pkce_state_id, delegation_state_id, plex_token_id, add_authn_provider_data, created FROM oidc_login_sessions WHERE deleted='0001-01-01 00:00:00' %s ORDER BY %s LIMIT %d) tmp ORDER BY %s;", p.GetWhereClause(), p.GetInnerOrderByClause(), p.GetLimit()+1, p.GetOuterOrderByClause())

	var objsDB []OIDCLoginSession
	if err := s.db.SelectContext(ctx, "ListOIDCLoginSessionsPaginated", &objsDB, q, queryFields...); err != nil {
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

// SaveOIDCLoginSession saves a OIDCLoginSession
func (s *Storage) SaveOIDCLoginSession(ctx context.Context, obj *OIDCLoginSession) error {
	if err := obj.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(s.saveInnerOIDCLoginSession(ctx, obj))
}

// SaveOIDCLoginSession saves a OIDCLoginSession
func (s *Storage) saveInnerOIDCLoginSession(ctx context.Context, obj *OIDCLoginSession) error {
	const q = "INSERT INTO oidc_login_sessions (id, updated, deleted, client_id, response_types, redirect_uri, state, scopes, nonce, social_provider, oidc_issuer_url, mfa_state_id, mfa_channel_states, otp_state_id, pkce_state_id, delegation_state_id, plex_token_id, add_authn_provider_data) VALUES ($1, CLOCK_TIMESTAMP(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17) ON CONFLICT (id, deleted) DO UPDATE SET updated = CLOCK_TIMESTAMP(), deleted = $2, client_id = $3, response_types = $4, redirect_uri = $5, state = $6, scopes = $7, nonce = $8, social_provider = $9, oidc_issuer_url = $10, mfa_state_id = $11, mfa_channel_states = $12, otp_state_id = $13, pkce_state_id = $14, delegation_state_id = $15, plex_token_id = $16, add_authn_provider_data = $17 WHERE (oidc_login_sessions.id = $1) RETURNING created, updated; /* allow-multiple-target-use no-match-cols-vals */"
	if err := s.db.GetContext(ctx, "SaveOIDCLoginSession", obj, q, obj.ID, obj.Deleted, obj.ClientID, obj.ResponseTypes, obj.RedirectURI, obj.State, obj.Scopes, obj.Nonce, obj.OIDCProvider, obj.OIDCIssuerURL, obj.MFAStateID, obj.MFAChannelStates, obj.OTPStateID, obj.PKCEStateID, obj.DelegationStateID, obj.PlexTokenID, obj.AddAuthnProviderData); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ucerr.Friendlyf(err, "OIDCLoginSession %v not found", obj.ID)
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteOIDCLoginSession soft-deletes a OIDCLoginSession which is currently alive
// Note that this will fail on an already-deleted object (since we don't want to re-delete
// tombstoned objects and corrupt the deletion timestamp)
func (s *Storage) DeleteOIDCLoginSession(ctx context.Context, objID uuid.UUID) error {
	return ucerr.Wrap(s.deleteInnerOIDCLoginSession(ctx, objID, false))
}

// deleteInnerOIDCLoginSession soft-deletes a OIDCLoginSession which is currently alive
func (s *Storage) deleteInnerOIDCLoginSession(ctx context.Context, objID uuid.UUID, wrappedDelete bool) error {
	const q = "UPDATE oidc_login_sessions SET deleted=CLOCK_TIMESTAMP() WHERE id=$1 AND deleted='0001-01-01 00:00:00' RETURNING deleted;"
	res, err := s.db.ExecContext(ctx, "DeleteOIDCLoginSession", q, objID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return ucerr.Errorf("Error deleting OIDCLoginSession %v: %w", objID, err)
	}
	if ra == 0 {
		// we wrap sql.ErrNoRows here to be consistent
		return ucerr.Friendlyf(sql.ErrNoRows, "OIDCLoginSession %v not found", objID)
	}
	return nil
}
