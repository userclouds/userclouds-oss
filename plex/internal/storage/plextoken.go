package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uclog"
)

// PlexToken stores all tokens, codes, and client IDs associated with a token grant. In particular it stores
// the Plex-generated tokens and the underlying IDP's tokens as well as the Plex- and IDP-specific client IDs
// (since each Plex client ID is associated with 1 client ID per underlying IDP).
// We keep underlying tokens around because sometimes our customers want to do things like login with Google
// and then use the Google token to access some other Google API
// TODO: should this be a setting to reduce security surface area someday?
type PlexToken struct {
	ucdb.BaseModel

	ClientID string `db:"client_id" validate:"notempty"`

	AuthCode     string `db:"auth_code" validate:"notempty"`
	AccessToken  string `db:"access_token" validate:"notempty"`
	IDToken      string `db:"id_token"`      // can be empty, e.g. for client credentials flow
	RefreshToken string `db:"refresh_token"` // optional

	IDPSubject string `db:"idp_subject"` // can be empty, e.g. for client credentials flow
	Scopes     string `db:"scopes" validate:"notempty"`

	// If the token was generated as part of a login session (e.g. any interactive user login),
	// this will be the ID of a valid OIDCLoginSession.
	// It will be NonInteractiveSessionID for a non-interactive login (e.g. client credentials flow).
	SessionID uuid.UUID `db:"session_id"`

	// TODO: we current store this as a string to account for different IDP types (e.g. Google, Okta, etc.)
	// but it's quite possible it should actually be an *oidc.TokenInfo struct?
	UnderlyingToken string `db:"underlying_token"` // optional
}

func (pt PlexToken) getCursor(key pagination.Key, cursor *pagination.Cursor) {
	if key == "updated,id" {
		*cursor = pagination.Cursor(
			fmt.Sprintf(
				"updated:%v,id:%v",
				pt.Updated.UnixMicro(),
				pt.ID,
			),
		)
	}
}

func (PlexToken) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"created": pagination.TimestampKeyType,
		"updated": pagination.TimestampKeyType,
	}
}

func (pt PlexToken) isExpired() bool {
	for _, token := range []string{pt.AccessToken, pt.IDToken, pt.RefreshToken} {
		if isExpired, err := ucjwt.IsExpired(token); err == nil && !isExpired {
			return false
		}
	}

	return true
}

func (pt PlexToken) isInteractive() bool {
	return pt.SessionID != NonInteractiveSessionID
}

//go:generate genpageable PlexToken
//go:generate genvalidate PlexToken

func (s *Storage) cleanPlexToken(ctx context.Context, candidate PlexToken, dryRun bool) error {
	if !candidate.isExpired() {
		return nil
	}

	uclog.Debugf(ctx, "detected expired plex token '%v'", candidate.ID)

	if !dryRun {
		// detach the plex token from the session if applicable

		if candidate.isInteractive() {
			if session, err := s.GetOIDCLoginSession(ctx, candidate.SessionID); err == nil {
				if session.PlexTokenID == candidate.ID {
					session.PlexTokenID = uuid.Nil
					if err := s.SaveOIDCLoginSession(ctx, session); err != nil {
						return ucerr.Wrap(err)
					}

					uclog.Infof(
						ctx,
						"detached expired plex token '%v' from session '%v'",
						candidate.ID,
						candidate.SessionID,
					)
				}
			}
		}

		// delete the plex token

		if err := s.deletePlexToken(ctx, candidate.ID); err != nil {
			return ucerr.Wrap(err)
		}

		uclog.Infof(ctx, "deleted expired plex token '%v'", candidate.ID)
	}

	return nil
}

// CleanPlexTokens will look for expired and unreferenced plex tokens, evaluating
// up to maxCandidates tokens and only actually deleting the plex tokens if dryRun is false
func (s *Storage) CleanPlexTokens(ctx context.Context, maxCandidates int, dryRun bool) error {
	if maxCandidates < 1 {
		return ucerr.Errorf("maxCandidates must be greater than or equal to one: %d", maxCandidates)
	}

	candidatesPerPage := min(maxCandidates, pagination.MaxLimit)

	pager, err := NewPlexTokenPaginatorFromOptions(
		pagination.Limit(candidatesPerPage),
		pagination.SortKey("updated,id"),
	)
	if err != nil {
		return ucerr.Wrap(err)
	}

	numCandidates := 0

	uclog.Infof(ctx, "evaluating up to %d plex tokens", maxCandidates)

	for {
		pts, respFields, err := s.ListPlexTokensPaginated(ctx, *pager)
		if err != nil {
			return ucerr.Wrap(err)
		}

		for _, pt := range pts {
			if err := s.cleanPlexToken(ctx, pt, dryRun); err != nil {
				return ucerr.Wrap(err)
			}

			numCandidates++
			if numCandidates == maxCandidates {
				return nil
			}
		}

		if !pager.AdvanceCursor(*respFields) {
			return nil
		}
	}
}

func (s *Storage) deletePlexToken(ctx context.Context, objID uuid.UUID) error {
	const q = "DELETE FROM plex_tokens WHERE id = $1"

	if _, err := s.db.ExecContext(ctx, "deletePlexToken", q, objID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ucerr.Wrap(err)
	}

	return nil
}
