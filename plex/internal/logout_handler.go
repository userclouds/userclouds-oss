package internal

import (
	"net/http"
	"net/url"

	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/tenantconfig"
)

func (h *handler) logoutHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// NOTE: this is the redirect URL the app/caller requested, but the user agent
	// may be redirected somewhere else first before finally being sent here.
	redirectURL := r.URL.Query().Get("redirect_url")
	clientID := r.URL.Query().Get("client_id")

	p := tenantconfig.MustGetPlexMap(ctx)
	plexApp, _, err := p.FindAppForClientID(clientID)
	if err != nil {
		// TODO: differentiate error types (issue #103).
		uchttp.ErrorL(ctx, w, err, http.StatusBadRequest, "FindAppError")
		return
	}

	if _, err := plexApp.ValidateLogoutURI(ctx, redirectURL); err != nil {
		uchttp.ErrorL(ctx, w, err, http.StatusBadRequest, "LogoutURIError")
		return

	}
	// TODO: right now we just try to log out of the "active" provider, but in the
	// future we probably want to figure out which provider the user logged in to
	// and direct them there?
	client, err := provider.NewActiveClient(ctx, h.factory, clientID)
	if err != nil {
		uchttp.ErrorL(ctx, w, err, http.StatusBadRequest, "GetProviderError")
		return
	}

	// TODO: we should probably look at the current client session
	// and log out of whichever IDP issued the token, NOT whichever is primary.
	// e.g. if a user auth'd with a follower provider when the primary was down,
	// or if selectively routing some users to one provider or another.
	uclog.Infof(ctx, "logging out of auth provider [%s], client ID '%s', redirect to '%s'", client, clientID, redirectURL)
	logoutURL, err := client.Logout(ctx, redirectURL)
	if err != nil {
		// TODO: More granular error checking.
		uchttp.ErrorL(ctx, w, err, http.StatusInternalServerError, "ProviderError")
		return
	}

	uchttp.Redirect(w, r, logoutURL, http.StatusFound)
}

func (h *handler) auth0LogoutHandler(w http.ResponseWriter, r *http.Request) {
	returnTo := r.URL.Query().Get("returnTo")
	clientID := r.URL.Query().Get("client_id")
	redirectURL := &url.URL{
		Path: paths.LogoutPath,
		RawQuery: url.Values{
			"redirect_url": []string{returnTo},
			"client_id":    []string{clientID},
		}.Encode(),
	}
	uchttp.Redirect(w, r, redirectURL.String(), http.StatusTemporaryRedirect)
}
