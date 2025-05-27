package oidc

import (
	"context"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/plex"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
)

// GetLoginRedirectURL constructs an OIDC-compliant redirect URL on successful login.
func GetLoginRedirectURL(ctx context.Context, session *storage.OIDCLoginSession) (string, error) {
	s := tenantconfig.MustGetStorage(ctx)

	// look up plex token for session

	plexToken, err := s.GetPlexToken(ctx, session.PlexTokenID)
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	// generate redirect

	query := url.Values{
		"state": []string{session.State},
	}
	rts, err := storage.NewResponseTypes(session.ResponseTypes)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	if rts.Contains(storage.AuthorizationCodeResponseType) {
		query.Add("code", plexToken.AuthCode)
	}
	if rts.Contains(storage.TokenResponseType) {
		query.Add("token", plexToken.AccessToken)
	}
	if rts.Contains(storage.IDTokenResponseType) {
		query.Add("id_token", plexToken.IDToken)
	}
	// TODO: support different 'response_mode' values: query, fragment, form_post, web_message, etc
	redirectURI, err := url.Parse(session.RedirectURI)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	redirectURI.RawQuery = query.Encode()
	if len(redirectURI.Host) == 0 && len(redirectURI.Path) == 0 {
		uclog.Debugf(ctx, "no host or path in redirect URI for session '%s'", session.ID)
	}

	// clear the plex token from session

	session.PlexTokenID = uuid.Nil
	if err := s.SaveOIDCLoginSession(ctx, session); err != nil {
		return "", ucerr.Wrap(err)
	}

	return redirectURI.String(), nil
}

// NewLoginResponse creates an OIDC-friendly login response object that can be returned
// to the caller (for client side redirects).
func NewLoginResponse(ctx context.Context, session *storage.OIDCLoginSession) (*plex.LoginResponse, error) {
	redirectTo, err := GetLoginRedirectURL(ctx, session)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	resp := plex.LoginResponse{
		RedirectTo: redirectTo,
	}
	uclog.Debugf(ctx, "NewLoginResponse redirecting to: %s", resp.RedirectTo)
	return &resp, nil
}
