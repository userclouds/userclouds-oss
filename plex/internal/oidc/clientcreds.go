package oidc

import (
	"net/http"
	"net/url"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/oidc"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
)

func (h *Handler) clientCredentialsTokenExchange(w http.ResponseWriter, r *http.Request, s *storage.Storage, postForm *url.Values, plexApp *tenantplex.App) {
	ctx := r.Context()

	tc := tenantconfig.MustGet(ctx)
	ts := multitenant.MustGetTenantState(ctx)
	tu := tenantconfig.MustGetTenantURLString(ctx)

	// 0 or more audiences may be specified as extra parameters
	audiences := (*postForm)["audience"]
	audiences = append(audiences, tu)

	// if the actually-used host is different from the primary, include both
	if tenantURL := ts.GetTenantURL(); tu != tenantURL {
		audiences = append(audiences, tenantURL)
	}

	// TODO: support 'scope' parameter, which is optional but not commonly supported?
	// https://www.oauth.com/oauth2-servers/access-tokens/client-credentials/
	scope := "unused"

	plexToken, err := storage.GenerateM2MPlexToken(ctx, &tc, s, plexApp.ClientID, scope, audiences)
	if err != nil {
		jsonapi.MarshalErrorL(ctx, w, err, "FailedToGenerateM2MToken")
		return
	}

	jsonapi.Marshal(w, oidc.TokenResponse{
		TokenType:    "Bearer",
		AccessToken:  plexToken.AccessToken,
		RefreshToken: plexToken.RefreshToken,
	})
}
