package social

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/addauthn"
	"userclouds.com/plex/internal/invite"
	"userclouds.com/plex/internal/loginapp"
	plexoidc "userclouds.com/plex/internal/oidc"
	"userclouds.com/plex/internal/otp"
	"userclouds.com/plex/internal/paths"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/reactdev"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
)

type handler struct {
	factory provider.Factory
}

// NewHandler returns a new social login handler for plex
func NewHandler(factory provider.Factory) http.Handler {
	h := &handler{factory}

	hb := builder.NewHandlerBuilder()
	handlerBuilder(hb, h)

	// Support for 3rd party OIDC (e.g. social sign-on).
	hb.HandleFunc("/login", h.login)

	hb.HandleFunc(paths.SocialCallbackSubPath, h.callback)

	// TODO: not happy with this endpoint design yet but shipping this tonight
	// so it will work. /userinfo didn't seem right, and putting this in IDP
	// would suck because the token is stored in Plex storage.
	hb.HandleFunc("/underlying", h.underlyingTokenHandler)

	return hb.Build()
}

//go:generate genhandler /social

func (h *handler) newOIDCAuthenticator(ctx context.Context, tc *tenantplex.TenantConfig, provider oidc.ProviderType, issuerURL string) (*oidc.Authenticator, error) {
	ts := multitenant.MustGetTenantState(ctx)
	redirectURL, err := tc.GetOIDCRedirectURL(ts.TenantURL, provider, issuerURL)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	redirectURL.Path = fmt.Sprintf("%s%s", paths.SocialRootPath, paths.SocialCallbackSubPath)
	authr, err := h.factory.NewOIDCAuthenticator(ctx, provider, issuerURL, tc.OIDCProviders, redirectURL)
	return authr, ucerr.Wrap(err)
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tc := tenantconfig.MustGet(ctx)
	s := tenantconfig.MustGetStorage(ctx)

	sessionID, err := uuid.FromString(r.URL.Query().Get("session_id"))
	if err != nil {
		uchttp.ErrorL(ctx, w,
			ucerr.Errorf("required query parameter 'session_id' missing or malformed: %w", err),
			http.StatusBadRequest, "InvalidSessionID")
		return
	}

	providerName := r.URL.Query().Get("oidc_provider")
	prov, err := tc.OIDCProviders.GetProviderForName(providerName)
	if err != nil || !prov.IsConfigured() {
		uchttp.ErrorL(ctx, w, err, http.StatusBadRequest, "BadOIDCProvider")
		return
	}

	providerType := prov.GetType()
	if err := storage.SetOIDCLoginSessionOIDCProvider(ctx, s, sessionID, providerType, prov.GetIssuerURL()); err != nil {
		uchttp.ErrorL(ctx, w, err, http.StatusInternalServerError, "FailedToSaveSession")
		return
	}

	authr, err := h.newOIDCAuthenticator(ctx, &tc, providerType, prov.GetIssuerURL())
	if err != nil {
		uchttp.ErrorL(ctx, w, err, http.StatusBadRequest, "FailedAuthr")
		return
	}

	ts := multitenant.MustGetTenantState(ctx)
	authCodeURL := authr.Config.AuthCodeURL(EncodeState(sessionID, ts.GetTenantURL()), authr.AuthCodeOptionGetter(r)...)
	uchttp.Redirect(w, r, authCodeURL, http.StatusTemporaryRedirect)
}

func (h *handler) callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uclog.Debugf(ctx, "URL for 3rd party social callback: %s", r.URL)

	if r.URL.Query().Has("error") {
		uchttp.ErrorL(ctx, w,
			ucerr.Errorf("social login failed with error '%v' and error description '%v'",
				r.URL.Query().Get("error"),
				r.URL.Query().Get("error_description")),
			http.StatusBadRequest,
			"FailedSocialAuth")
		return
	}

	// Step 1: Validate/parse redirect/state/scope parameters.
	// TODO: (#76) use state for security check only, and store components in DB.

	state, sessionID, _, err := decodeState(r)
	if err != nil {
		uchttp.ErrorL(ctx, w,
			ucerr.Errorf("required query parameter 'state' missing or malformed: %w", err),
			http.StatusBadRequest, "InvalidSessionID")
		return
	}

	s := tenantconfig.MustGetStorage(ctx)
	session, err := s.GetOIDCLoginSession(ctx, sessionID)
	if err != nil {
		uchttp.ErrorL(ctx, w,
			ucerr.Errorf("invalid 'session_id': %w", err), http.StatusBadRequest, "FailedSessionGet")
		return
	}

	tc := tenantconfig.MustGet(ctx)
	authr, err := h.newOIDCAuthenticator(ctx, &tc, session.OIDCProvider, session.OIDCIssuerURL)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	// Step 2: Exchange code for token, validate & read claims.
	tokenInfo, statusCode, err := authr.ProcessAuthCodeCallback(r, state)
	if err != nil {
		uchttp.Error(ctx, w, err, statusCode)
		return
	} else if statusCode < 200 || statusCode > 299 {
		// Don't return content in the response, just a code.
		w.WriteHeader(statusCode)
		return
	}

	// Step 3: Create/update user account in IDP
	claims, err := oidc.ExtractClaims(tokenInfo.Profile)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	prov, err := provider.NewActiveManagementClient(ctx, h.factory, session.ClientID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	// First see if the user already exists
	user, err := prov.GetUserForOIDC(ctx, session.OIDCProvider, session.OIDCIssuerURL, claims.Subject, claims.Email)
	if err != nil {
		if errors.Is(err, iface.ErrUserNotFound) {
			// User not found, ensure user is cleared out
			user = nil
		} else {
			uclog.Errorf(ctx, "failed to get user for OIDC: %v: provider: %v URL: %s subject: %s email: %s", err, session.OIDCProvider, session.OIDCIssuerURL, claims.Subject, claims.Email)
			uchttp.Error(ctx, w, err, http.StatusInternalServerError)
			return
		}
	}

	app, _, err := tc.PlexMap.FindAppForClientID(session.ClientID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	if user != nil {
		hasAccess, err := loginapp.CheckLoginAccessForUser(ctx, tc, app, user.ID)
		if err != nil {
			jsonapi.MarshalErrorL(ctx, w, err, "RestrictedAccessError")
			return
		}
		if !hasAccess {
			jsonapi.MarshalErrorL(ctx, w, ucerr.Friendlyf(nil, "You are not permitted to login to this app"), "RestrictedAccessDenied", jsonapi.Code(http.StatusForbidden))
			return
		}
	}

	profile := iface.NewUserProfileFromClaims(*claims)
	var userID string

	if user != nil {
		// A user exists for this social provider already
		userID = user.ID

		// Check the session to see if we need to add a new authn provider
		addauthn.CheckAndAddAuthnToUser(ctx, session, userID, profile.Email, prov)
	} else {
		// If no existing user is found from this oidc provider, first check if the email is already in use by another account
		authns, err := addauthn.CheckForExistingAccounts(ctx, session, claims.Email, claims.Subject, s, prov)
		if err != nil {
			uchttp.Error(ctx, w, err, http.StatusInternalServerError)
			return
		}
		if authns != nil {
			// We found existing accounts, so redirect to Email Exists UI to allow the user
			// to add this oidc provider to an existing account
			redirectTo := reactdev.UIBaseURL(ctx)
			redirectTo.Path = redirectTo.Path + paths.EmailExistsUIPath
			authnsStr := strings.Join(authns, ",")
			redirectTo.RawQuery = url.Values{
				"session_id": []string{session.ID.String()},
				"authns":     []string{authnsStr},
				"email":      []string{claims.Email},
			}.Encode()

			uchttp.Redirect(w, r, redirectTo.String(), http.StatusSeeOther)
			return
		}

		// Check if there is a valid invite associated with the session
		validInvite, _, err := invite.CheckForValidInvite(ctx, session)
		if err != nil {
			uchttp.Error(ctx, w, err, http.StatusInternalServerError)
			return
		}

		var sessionInvite *storage.OIDCLoginSession
		// If there is no valid invite, check if there is an outstanding invite to this email
		if !validInvite {
			recentInvites, err := s.ListRecentOTPStates(ctx, storage.OTPPurposeInvite, claims.Email, 1)
			if err != nil {
				uchttp.Error(ctx, w, err, http.StatusInternalServerError)
				return
			}

			// if the most recent invite to this email is not bound to a user and not expired, prepare to swap out the session
			// after the user has been created
			if len(recentInvites) > 0 {
				recentInvite := recentInvites[0]
				if !recentInvite.Used && recentInvite.UserID == "" && recentInvite.Expires.After(time.Now().UTC()) {
					sessionInvite, err = s.GetOIDCLoginSession(ctx, recentInvite.SessionID)
					if err != nil {
						uclog.Errorf(ctx, "invalid session id found in OTP state: %v", err)
					} else {
						validInvite = true
					}
				}
			}
		}

		if tc.DisableSignUps && !validInvite && !slices.Contains(tc.BootstrapAccountEmails, claims.Email) {
			// If we're not allowing signups, and there's no valid invite in the session, and the email is not a bootstrap account
			// we should only get, not create, a user with social creds
			// TODO: this probably misses a merge case where the user already has a u/p
			// login with matching email that we should auto-merge? But pushing this logic
			// to IDP seems broken?
			uchttp.Error(ctx, w, ucerr.Friendlyf(nil, "Sorry, invites are currently required."), http.StatusForbidden)
			return

		}

		// Create a new user account
		// TODO: we don't currently mark the email as valid here (like we do in normal invite flows)
		userID, err = prov.CreateUserWithOIDC(ctx, session.OIDCProvider, session.OIDCIssuerURL, claims.Subject, *profile)
		if err != nil {
			uchttp.Error(ctx, w, err, http.StatusInternalServerError)
			return
		}

		if err := loginapp.AddLoginAccessForUserIfNecessary(ctx, tc, app, userID); err != nil {
			uchttp.Error(ctx, w, err, http.StatusInternalServerError)
			return
		}

		if sessionInvite != nil {
			// Now that the user has been created with the relevant OIDC details, we can swap out the session
			session = sessionInvite
		}
	}

	profile.ID = userID

	tu := tenantconfig.MustGetTenantURLString(ctx)
	if err := storage.GenerateUserPlexToken(ctx, tu, &tc, s, profile, session, tokenInfo); err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	if err := otp.BindInviteToUser(ctx, s, session, profile.ID, profile.Email, app); err != nil &&
		!errors.Is(err, otp.ErrNoInviteAssociatedWithSession) {
		if errors.Is(err, otp.ErrInviteBoundToAnotherUser) {
			uchttp.Error(ctx, w, err, http.StatusBadRequest)
			return
		}
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	redirectURL, err := plexoidc.GetLoginRedirectURL(ctx, session)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	auditlog.Post(ctx, auditlog.NewEntry(profile.ID, auditlog.LoginSuccess,
		auditlog.Payload{"ID": app.ID, "Name": app.Name, "Actor": profile.Email, "Type": "Social", "OIDCIssuerURL": session.OIDCIssuerURL, "OIDCSubject": claims.Subject}))
	uchttp.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (h *handler) underlyingTokenHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	bearerToken, err := ucjwt.ExtractBearerToken(&r.Header)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Wrap(err), http.StatusUnauthorized)
		return
	}

	s := tenantconfig.MustGetStorage(ctx)
	token, err := s.GetPlexTokenForAccessToken(ctx, bearerToken)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Wrap(err), http.StatusBadRequest)
		return
	}

	// error nicely if it's not here
	if token.UnderlyingToken == "" {
		uchttp.Error(ctx, w, ucerr.New("No underlying token found"), http.StatusBadRequest)
		return
	}

	// since we store this as a string and not an object, we need to unmarshal it first
	// or we get escaped quotes in the JSON output, etc
	var ut oidc.TokenInfo
	if err := json.Unmarshal([]byte(token.UnderlyingToken), &ut); err != nil {
		uchttp.Error(ctx, w, ucerr.Wrap(err), http.StatusInternalServerError)
		return
	}

	jsonapi.Marshal(w, ut)
}
