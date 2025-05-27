package internal

import (
	"net/http"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/multitenant"
	plexoidc "userclouds.com/plex/internal/oidc"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
)

func (h *handler) handleEmployeeAuthCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tc := tenantconfig.MustGet(ctx)
	s := tenantconfig.MustGetStorage(ctx)
	tu := tenantconfig.MustGetTenantURLString(ctx)
	ts := multitenant.MustGetTenantState(ctx)

	tenant, err := h.companyConfigStorage.GetTenant(ctx, ts.ID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	// look up oidc login session and validate state

	query := r.URL.Query()
	sessionID, err := uuid.FromString(query.Get("request_session_id"))
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	session, err := s.GetOIDCLoginSession(ctx, sessionID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	if state := query.Get("request_state"); state != session.State {
		uchttp.Error(ctx, w, ucerr.New("request state does not match"), http.StatusBadRequest)
		return
	}

	// extract claims and verify that issuer is console and employee id is valid

	pk, err := ucjwt.LoadRSAPublicKey([]byte(tc.Keys.PublicKey))
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	idToken := query.Get("id_token")
	claims, err := ucjwt.ParseUCClaimsVerified(idToken, pk)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	// TODO: we need to ensure that the Issuer is the UC console tenant URL, and the
	// Auth config settings are currently the only way to get this programmatically
	// at the moment for a particular universe. Once we move to a better internal M2M
	// auth system we'll want to revisit how we do this.
	if claims.Issuer != h.consoleTenantInfo.TenantURL {
		uchttp.Error(ctx, w, ucerr.Errorf("unexpected id token issuer: %s", claims.Issuer), http.StatusBadRequest)
		return
	}

	employeeID, err := uuid.FromString(claims.Subject)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	// verify the employee exists in this tenant
	tokenEndpointURL, err := url.Parse(h.consoleTenantInfo.TenantURL)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}
	tokenEndpointURL.Path = "/oidc/token"
	authzClient, err := apiclient.NewAuthzClientWithTokenSource(ctx, h.companyConfigStorage, ts.ID, tokenEndpointURL.String(), h.m2mAuth)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}
	rbacClient := authz.NewRBACClient(authzClient)

	employeeProfile := iface.NewUserProfileFromClaims(*claims)

	if _, err := rbacClient.GetUser(ctx, employeeID); err != nil {
		uchttp.Error(ctx, w,
			ucerr.Friendlyf(ucerr.Errorf("user '%v' does not exist in tenant '%v'", employeeID, ts.ID),
				"user '%s' does not have access to '%v' tenant", employeeProfile.GetFriendlyName(), tenant.Name),
			http.StatusForbidden)
		return
	}

	// generate plex token for the employee profile

	if err := storage.GenerateUserPlexToken(ctx, tu, &tc, s, employeeProfile, session, nil); err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	// generate appropriate redirect for the oidc login session and plex token and redirect

	redirectURL, err := plexoidc.GetLoginRedirectURL(ctx, session)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	uchttp.Redirect(w, r, redirectURL, http.StatusFound)
}
