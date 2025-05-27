package oidc

import (
	"net/http"
	"net/url"

	"userclouds.com/idp"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
)

// https://datatracker.ietf.org/doc/html/rfc6749#section-4.3
func (h *Handler) passwordTokenExchange(w http.ResponseWriter, r *http.Request, s *storage.Storage, postForm *url.Values, plexApp *tenantplex.App) {
	ctx := r.Context()

	username := (*postForm).Get("username")
	password := (*postForm).Get("password")
	if username == "" || password == "" {
		jsonapi.MarshalErrorL(ctx, w, ucerr.Errorf("No username or password provided"), "NoUsernameOrPassword", jsonapi.Code(http.StatusBadRequest))
		return
	}

	// TODO (sgarrity 5/9/23): validateScopes only checks that openid is present, which isn't required for the password flow
	// but it seems as if we should validate that the scopes are valid for the client?
	scope := (*postForm).Get("scope")
	if scope == "" {
		// TODO (sgarrity 5/9/23): the PlexToken object currently requires scopes to be non-empty, but the password flow
		// doesn't require scopes. The CCF flow likewise doesn't require them, and this is the solution we chose there,
		// so I'm copying it here for now. It seems as if we should relax the validation rule but want to think about
		// that a bit more first.
		scope = "unused"
	}
	scopes := oidc.SplitTokens(scope)

	amc, err := provider.NewActiveClient(ctx, h.factory, plexApp.ClientID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	res, err := amc.UsernamePasswordLogin(ctx, username, password)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if res.Status == idp.LoginStatusMFARequired {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "MFA required, and not supported in this flow"), jsonapi.Code(http.StatusUnauthorized))
		return
	}

	if res.Status != idp.LoginStatusSuccess {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("Login failed"), jsonapi.Code(http.StatusUnauthorized))
		return
	}

	claims, err := oidc.ExtractClaims(res.Claims)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	profile := iface.NewUserProfileFromClaims(*claims)

	tc := tenantconfig.MustGet(ctx)
	token, err := storage.GenerateUserPlexTokenWithoutSession(ctx, &tc, s, profile, scopes, plexApp)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// NB: ID tokens are not usually returned as part of the ROPC flow although there is some debate
	// some places use "special" scopes to allow requesting it etc, we can see if people need it
	// https://stackoverflow.com/questions/41421160/how-to-get-id-token-along-with-access-token-from-identityserver4-via-password?rq=1
	response := oidc.TokenResponse{
		TokenType:    "Bearer",
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
	}

	jsonapi.Marshal(w, response)
}
