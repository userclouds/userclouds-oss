package oidc

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/storage"
)

func (h *Handler) authorizationCodeTokenExchange(w http.ResponseWriter, r *http.Request, s *storage.Storage, postForm *url.Values, plexApp *tenantplex.App) {
	ctx := r.Context()

	code := postForm.Get("code")
	if len(code) == 0 {
		jsonapi.MarshalErrorL(ctx, w, ucerr.NewRequestError(ucerr.Friendlyf(nil, "required query parameter 'code' missing or malformed")), "TokenCodeError")
		return
	}

	redirectURI := postForm.Get("redirect_uri")
	if len(redirectURI) == 0 {
		jsonapi.MarshalErrorL(ctx, w, ucerr.NewRequestError(ucerr.Friendlyf(nil, "required query parameter 'redirect_uri' missing or malformed")), "InvalidRedirect")
		return
	}

	if _, err := plexApp.ValidateRedirectURI(ctx, redirectURI); err != nil {
		jsonapi.MarshalErrorL(ctx, w, ucerr.NewRequestError(err), "InvalidRedirectURI")
		return
	}

	token, err := s.GetPlexTokenForAuthCode(ctx, code)
	if errors.Is(err, storage.ErrCodeNotFound) {
		jsonapi.MarshalErrorL(ctx, w, ucerr.Wrap(ucerr.ErrInvalidAuthorizationCode), "AuthCodeNotFound")
		return
	} else if err != nil {
		jsonapi.MarshalErrorL(ctx, w, ucerr.NewServerError(err), "AuthCodeError")
		return
	}

	// Get the session so we can check for PKCE parameters
	session, err := s.GetOIDCLoginSession(ctx, token.SessionID)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, ucerr.NewServerError(err), "FailedSessionGet")
		return
	}

	if redirectURI != session.RedirectURI {
		jsonapi.MarshalErrorL(ctx, w, ucerr.NewRequestError(ucerr.Friendlyf(nil, "provided 'redirect_uri' does not match the one provided at authorize time")), "MismatchedRedirectURI")
		return
	}

	if session.PKCEStateID != uuid.Nil {
		pkceState, err := s.GetPKCEState(ctx, session.PKCEStateID)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, ucerr.NewServerError(err), "FailedPKCEGet")
			return
		}

		if pkceState.Method.Validate() != nil {
			jsonapi.MarshalErrorL(ctx, w, ucerr.NewServerError(err), "BadCodeChallengeMethod")
			return
		}

		// Don't allow multiple attempts; fail with a non-revealing error
		if pkceState.Used {
			jsonapi.MarshalErrorL(ctx, w, ucerr.Wrap(ucerr.ErrInvalidCodeVerifier), "CodeVerifierUsed")
			return
		}

		pkceState.Used = true
		if err := s.SavePKCEState(ctx, pkceState); err != nil {
			jsonapi.MarshalErrorL(ctx, w, ucerr.NewServerError(err), "FailedSessionSave")
			return
		}

		codeVerifierParam := postForm.Get("code_verifier")
		if len(codeVerifierParam) < crypto.MinCodeVerifierBase64Length ||
			len(codeVerifierParam) > crypto.MaxCodeVerifierBase64Length {
			jsonapi.MarshalErrorL(ctx, w, ucerr.Wrap(ucerr.ErrInvalidCodeVerifier), "InvalidCodeVerifier")
			return
		}

		codeChallenge, err := crypto.CodeVerifier(codeVerifierParam).GetCodeChallenge(pkceState.Method)
		if err != nil || codeChallenge != pkceState.CodeChallenge {
			jsonapi.MarshalErrorL(ctx, w, ucerr.Wrap(ucerr.ErrInvalidCodeVerifier), "InvalidCodeVerifier2")
			return
		}
	}

	response := oidc.TokenResponse{
		TokenType:    "Bearer",
		IDToken:      token.IDToken,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
	}

	jsonapi.Marshal(w, response)
}
