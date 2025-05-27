package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/console/internal/tenantcache"
	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/kubernetes"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/service"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/security"
	"userclouds.com/plex"
	"userclouds.com/plex/manager"
	"userclouds.com/userevent"
)

// RootPath is the root path of the Console auth handler
const RootPath = "/auth"

// RedirectPath is the path to the endpoint on Console used to start the login
const RedirectPath = RootPath + "/redirect"

// AuthCallbackPath is the path to the endpoint on Console called back after Plex login
const AuthCallbackPath = RootPath + "/callback"

// InviteCallbackPath is the path to the endpoint on Console that gets invoked when an invited user successfully logs in
// for the first time via the invite (either by creating a new account or signing in with an existing one).
const InviteCallbackPath = RootPath + "/invitecallback"

// ImpersonateUserPath is the path to the delegated auth endpoint on Console to impersonate a user
const ImpersonateUserPath = RootPath + "/impersonateuser"

// UnimpersonateUserPath is the path to the delegated auth endpoint on Console to stop impersonating a user
const UnimpersonateUserPath = RootPath + "/unimpersonateuser"

// InviteStatePrefix is the prefix for the OIDC state value associated with a user invite + login.
const InviteStatePrefix = "invitekey"

// EmployeeLoginPath is the path to the endpoint on Console used to start an employee login
const EmployeeLoginPath = RootPath + "/employee/login"

type handler struct {
	cfg                   Config
	getConsoleURLCallback GetConsoleURLCallback
	storage               *companyconfig.Storage
	sessionMgr            *SessionManager
	authzClient           *authz.Client
	tenantCache           *tenantcache.Cache
	cacheConfig           *cache.Config
}

// GetConsoleURLCallback is a callback function to get the console URL.
// This is because we create the test server after the handler, and only then do we know the URL of the Console test server.
type GetConsoleURLCallback func() *url.URL

// NewHandler returns a new auth handler
func NewHandler(consoleTenantID uuid.UUID,
	clientCacheConfig *cache.Config,
	getConsoleURLCallback GetConsoleURLCallback,
	storage *companyconfig.Storage,
	sessions *SessionManager,
	tc *tenantcache.Cache) http.Handler {

	ctx := context.Background()

	// load our console auth config
	ten, err := storage.GetTenant(ctx, consoleTenantID)
	if err != nil {
		uclog.Fatalf(ctx, "error setting up console auth handler: %v", err)
	}

	mgr, err := manager.NewFromCompanyConfig(ctx, storage, consoleTenantID, clientCacheConfig)
	if err != nil {
		uclog.Fatalf(ctx, "error setting up console auth handler: %v", err)
	}
	las, err := mgr.GetLoginApps(ctx, consoleTenantID, ten.CompanyID)
	if err != nil || len(las) < 1 {
		uclog.Fatalf(ctx, "error setting up console auth handler: %v", err)
	}

	cfg := Config{
		TenantID:     consoleTenantID,
		TenantURL:    ten.TenantURL,
		CompanyID:    ten.CompanyID,
		ClientID:     las[0].ClientID,
		ClientSecret: las[0].ClientSecret,
	}

	tokenSource, err := m2m.GetM2MTokenSource(ctx, consoleTenantID)
	if err != nil {
		uclog.Fatalf(ctx, "error setting up console auth handler: %v", err)
	}

	authzClient, err := apiclient.NewAuthzClientWithTokenSource(ctx, storage, cfg.TenantID, cfg.TenantURL, tokenSource, apiclient.ClientCacheConfig(clientCacheConfig))
	if err != nil {
		uclog.Fatalf(ctx, "error setting up console auth handler: %v", err)
	}

	h := &handler{cfg, getConsoleURLCallback, storage, sessions, authzClient, tc, clientCacheConfig}

	hb := builder.NewHandlerBuilder()

	// Redirects to Plex for login.
	hb.HandleFunc(strings.TrimPrefix(RedirectPath, RootPath), h.loginRedirect)
	// Plex redirects back here after login.
	hb.HandleFunc(strings.TrimPrefix(AuthCallbackPath, RootPath), h.loginCallback)
	// Plex redirects here after an invited user logs in.
	hb.HandleFunc(strings.TrimPrefix(InviteCallbackPath, RootPath), h.inviteCallback)

	hb.Handle(strings.TrimPrefix(EmployeeLoginPath, RootPath), sessions.RedirectIfNotLoggedIn().Apply(http.HandlerFunc(h.employeeLogin)))

	// Logs out of the underlying IDP (Plex)
	hb.HandleFunc("/logout", h.logout)
	// Get info about the currently-logged in user.
	hb.Handle("/userinfo", sessions.FailIfNotLoggedIn().Apply(http.HandlerFunc(h.userInfo)))

	hb.Handle("/impersonateuser", sessions.FailIfNotLoggedIn().Apply(http.HandlerFunc(h.impersonateUser)))

	// Restore the session for the impersonating user
	hb.Handle("/unimpersonateuser", sessions.FailIfNotLoggedIn().Apply(http.HandlerFunc(h.unimpersonateUser)))

	return hb.Build()
}

func validateRedirectTo(redirectTo string) error {
	// Redirection seems like a possible future security hole, so be extra strict.
	// Allow anything in dev (to support react development), but only rooted paths otherwise.
	// TODO: if the first char is a forward slash, does that guarantee it's rooted?
	uv := universe.Current()
	if uv.IsDev() || uv.IsContainer() ||
		(redirectTo == "" || redirectTo[0] == '/') {
		return nil
	}
	return ucerr.Errorf("Invalid console redirect url: %s", redirectTo)
}

func (h *handler) newAuthenticator(ctx context.Context, requestHost string, redirectPath string) (*oidc.Authenticator, error) {
	return h.newAuthenticatorForConfig(ctx, requestHost, redirectPath, h.cfg)
}

func (h *handler) newAuthenticatorForConfig(ctx context.Context, requestHost string, redirectPath string, cfg Config) (*oidc.Authenticator, error) {
	redirectURL := h.getRedirectURL(requestHost, redirectPath)
	tenantURL := cfg.TenantURL
	cr := region.Current()
	if strings.Contains(requestHost, fmt.Sprintf(".%s.", cr)) {
		// request was made to a regional host, we should use a regional URL for the tenant
		tenantURL = companyconfig.GetTenantRegionalURL(tenantURL, cr, false)
		uclog.Debugf(ctx, "Using regional tenant URL: %s (instead of %s) since request host %s is regionalized", tenantURL, cfg.TenantURL, requestHost)
	} else if strings.Contains(requestHost, fmt.Sprintf(".%s-eks.", cr)) {
		// request was made to a regional eks host, we should use a regional URL for the tenant
		tenantURL = companyconfig.GetTenantRegionalURL(tenantURL, cr, true)
		uclog.Debugf(ctx, "Using EKS regional tenant URL: %s (instead of %s) since request host %s is regionalized", tenantURL, cfg.TenantURL, requestHost)
	}
	return oidc.NewAuthenticator(
		ctx,
		tenantURL,
		cfg.ClientID,
		cfg.ClientSecret,
		redirectURL)
}

func (h *handler) employeeLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	session, err := h.sessionMgr.GetAuthSession(r)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("Failed to get `auth-session` cookie: %w", err), http.StatusInternalServerError)
		return
	}
	requestQuery := r.URL.Query()

	// make sure request_session_id, request_state, and request_tenant_id are in request
	requestSessionID, err := uuid.FromString(requestQuery.Get("request_session_id"))
	if err != nil {
		uchttp.Error(ctx, w, ucerr.New("request_session_id missing or malformed in request"), http.StatusBadRequest)
		return
	}

	requestState := requestQuery.Get("request_state")
	if requestState == "" {
		uchttp.Error(ctx, w, ucerr.New("request_state missing or malformed in request"), http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.FromString(requestQuery.Get("request_tenant_id"))
	if err != nil {
		uchttp.Error(ctx, w, ucerr.New("request_tenant_id missing or malformed in request"), http.StatusBadRequest)
		return
	}

	// construct redirect URL for request tenant
	ten, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("tenant ID unrecognized: %v", tenantID), http.StatusBadRequest)
		return
	}

	tenantDB, err := h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	mgr := manager.NewFromDB(tenantDB, h.cacheConfig)
	tp, err := mgr.GetTenantPlex(ctx, tenantID)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("tenant ID unrecognized: %v", tenantID), http.StatusBadRequest)
		return
	}

	if tp.PlexConfig.PlexMap.EmployeeApp == nil {
		uchttp.Error(ctx, w, ucerr.Errorf("tenant '%v' does not have employee app", tenantID), http.StatusInternalServerError)
		return
	}

	redirectURI, err := url.Parse(fmt.Sprintf("%s/employee/authcallback", ten.TenantURL))
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	// redirect back, passing in request session id, request state, and the id token

	query := url.Values{}
	query.Add("request_session_id", requestSessionID.String())
	query.Add("request_state", requestState)
	query.Add("id_token", session.IDToken)
	redirectURI.RawQuery = query.Encode()

	uchttp.Redirect(w, r, redirectURI.String(), http.StatusFound)
}

func (h *handler) loginRedirect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authr, err := h.newAuthenticator(ctx, r.Host, AuthCallbackPath)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	session, err := h.sessionMgr.GetAuthSession(r)
	if err != nil {
		h.sessionMgr.ClearAuthSessionCookie(w)
		uchttp.Redirect(w, r, RedirectPath, http.StatusFound)
		return
	}

	redirectTo := r.URL.Query().Get("redirect_to")
	if err := validateRedirectTo(redirectTo); err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	var state string
	if session.State == "" {
		state = fmt.Sprintf("%s#%s", crypto.MustRandomBase64(32), url.QueryEscape(redirectTo))
		session.State = state
		err = h.sessionMgr.SaveSession(ctx, w, session)
	} else {
		// There is already an existing state, so we should use that. We will be redirecting to
		// the first URL passed to us (seems as good as any other), and ignoring any subsequent ones.
		state = session.State
	}

	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	authCodeURL := authr.Config.AuthCodeURL(state)
	uclog.Infof(ctx, "redirecting to login %v, state: '%s'", authCodeURL, state)
	uchttp.Redirect(w, r, authCodeURL, http.StatusTemporaryRedirect)
}

func (h *handler) loginCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	session, err := h.sessionMgr.GetAuthSession(r)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("Failed to get `auth-session` cookie: %w", err), http.StatusInternalServerError)
		return
	}

	stateParts := strings.Split(session.State, "#")
	if len(stateParts) != 2 {
		uchttp.Error(ctx, w, ucerr.Errorf("malformed state: %s", session.State), http.StatusBadRequest)
		return
	}

	redirectTo, err := url.QueryUnescape(stateParts[1])
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}
	if redirectTo == "" {
		redirectTo = "/" // default
	}

	authr, err := h.newAuthenticator(ctx, r.Host, AuthCallbackPath)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	tokenInfo, statusCode, err := authr.ProcessAuthCodeCallback(r, session.State)
	if err != nil {
		uchttp.Error(ctx, w, err, statusCode)
		return
	} else if statusCode < 200 || statusCode > 299 {
		// Don't return content in the response, just a code.
		w.WriteHeader(statusCode)
		return
	}

	if session.IDToken != "" {
		// This is an impersonation, so we should store the original user's session
		if session.ImpersonatorIDToken == impersonationRequestPlaceholder {
			session.ImpersonatorIDToken = session.IDToken
			session.ImpersonatorAccessToken = session.AccessToken
			session.ImpersonatorRefreshToken = session.RefreshToken
		} else {
			uclog.Errorf(ctx, "Impersonation session not requested but ID token already set")
		}
	}

	session.IDToken = tokenInfo.RawIDToken
	session.AccessToken = tokenInfo.AccessToken
	session.RefreshToken = tokenInfo.RefreshToken
	if err := h.sessionMgr.SaveSession(ctx, w, session); err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	uchttp.Redirect(w, r, redirectTo, http.StatusSeeOther)
}

func shouldConsumeInvite(ikt companyconfig.InviteKeyType) bool {
	switch ikt {
	case companyconfig.InviteKeyTypeExistingCompany:
		return true
	default:
		// For other invite types, we will "consume" the key when a certain action takes place
		return false
	}
}

func (h *handler) inviteCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	uclog.Debugf(ctx, "inviteCallback URL: %s", r.URL.String())

	session, err := h.sessionMgr.GetAuthSession(r)
	if err != nil {
		// User landed on this page without a session; redirect to login to create one
		uchttp.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		// User was not sent here by OIDC provider; redirect to login to create a session
		uchttp.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	stateParts := strings.Split(state, "#")
	if len(stateParts) != 2 || stateParts[0] != InviteStatePrefix {
		uchttp.Error(ctx, w, ucerr.Errorf("malformed state: %s", state), http.StatusBadRequest)
		return
	}
	key := stateParts[1]
	inviteKey, err := h.storage.GetValidInviteKey(ctx, key)
	if err != nil {
		// TODO: differentiate between storage failure and invalid key
		uchttp.Error(ctx, w, ucerr.Errorf("failed to retrieve invite: %w", err), http.StatusBadRequest)
		return
	}

	tenantDB, err := h.tenantCache.GetTenantDB(ctx, h.cfg.TenantID)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	plexManager := manager.NewFromDB(tenantDB, h.cacheConfig)
	loginApps, err := plexManager.GetLoginApps(ctx, h.cfg.TenantID, inviteKey.CompanyID)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("failed to retrieve login apps: %w", err), http.StatusInternalServerError)
		return
	}
	if len(loginApps) != 1 {
		uchttp.Error(ctx, w, ucerr.Errorf("expected 1 login app, got %d", len(loginApps)), http.StatusInternalServerError)
		return
	}

	authr, err := h.newAuthenticatorForConfig(ctx, r.Host, InviteCallbackPath, Config{
		TenantID:     h.cfg.TenantID,
		TenantURL:    h.cfg.TenantURL,
		ClientID:     loginApps[0].ClientID,
		ClientSecret: loginApps[0].ClientSecret,
		CompanyID:    inviteKey.CompanyID,
	})
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	tokenInfo, statusCode, err := authr.ProcessAuthCodeCallback(r, state)
	if err != nil {
		uchttp.Error(ctx, w, err, statusCode)
		return
	} else if statusCode < 200 || statusCode > 299 {
		// Don't return content in the response, just a code.
		w.WriteHeader(statusCode)
		return
	}

	session.IDToken = tokenInfo.RawIDToken
	session.AccessToken = tokenInfo.AccessToken
	session.RefreshToken = tokenInfo.RefreshToken
	if err := h.sessionMgr.SaveSession(ctx, w, session); err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	userID, err := uuid.FromString(tokenInfo.Claims.Subject)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	if inviteKey.InviteeUserID != uuid.Nil && inviteKey.InviteeUserID != userID {
		// Invite already bound to another user
		uchttp.Error(ctx, w, ucerr.New("invite already used by another user"), http.StatusBadRequest)
		return
	}

	// Bind invite to this user, and consume the invite if appropriate. Err on doing this even if the subsequent operation
	// fails, rather than allowing an invite to get reused multiple times in the event that re-saving the key fails.

	inviteKey.InviteeUserID = userID
	if shouldConsumeInvite(inviteKey.Type) {
		inviteKey.Used = true
	}

	if err := h.storage.SaveInviteKey(ctx, inviteKey); err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	if inviteKey.Type == companyconfig.InviteKeyTypeExistingCompany {

		tenants, err := h.storage.ListTenantsForCompany(ctx, inviteKey.CompanyID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			uchttp.Error(ctx, w, err, http.StatusInternalServerError)
			return
		}

		rbacClient := authz.NewRBACClient(h.authzClient)
		// Add employee to existing company in authz and also add to any associated tenant user stores
		if err := AddEmployeeRoleToCompany(ctx, h.cfg.TenantID, h.storage, h.tenantCache, rbacClient, userID, inviteKey.CompanyID, inviteKey.Role, inviteKey.TenantRoles, tenants, h.cacheConfig); err != nil {
			// Since the user ID and company ID should have been validated already, this is likely an ISE.
			uchttp.Error(ctx, w, err, http.StatusInternalServerError)
			return
		}
	}

	// Redirect to root page after accepting invite.
	uchttp.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *handler) logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// clear auth cookies locally
	h.sessionMgr.ClearAuthSessionCookie(w)

	redirectTo := r.URL.Query().Get("redirect_to")
	if err := validateRedirectTo(redirectTo); err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}
	if redirectTo == "" || redirectTo[0] == '/' {
		// Relative to base path
		redirectTo = h.getRedirectURL(r.Host, redirectTo)
	}

	query := url.Values{}
	query.Add("client_id", h.cfg.ClientID)
	query.Add("redirect_url", redirectTo)
	logoutURL := h.cfg.TenantURL + "/logout?" + query.Encode()
	uchttp.Redirect(w, r, logoutURL, http.StatusTemporaryRedirect)
}

type userInfoResponse struct {
	UserProfile         *idp.UserResponse `json:"user_profile"`
	ImpersonatorProfile *idp.UserResponse `json:"impersonator_profile,omitempty"`
}

func (h *handler) userInfo(w http.ResponseWriter, r *http.Request) {
	// We apply middleware to ensure this succeeds
	userInfo := MustGetUserInfo(r)

	resp := userInfoResponse{
		UserProfile: &idp.UserResponse{
			// Since this always comes from the UC IDP, it will be a valid UUID
			ID: uuid.FromStringOrNil(userInfo.Claims.Subject),
			Profile: userstore.Record{
				"name":           userInfo.Claims.Name,
				"email":          userInfo.Claims.Email,
				"email_verified": userInfo.Claims.EmailVerified,
				"picture":        userInfo.Claims.Picture,
			},
		},
	}

	if impersonatorInfo := GetImpersonatorInfo(r); impersonatorInfo != nil {
		resp.ImpersonatorProfile = &idp.UserResponse{
			ID: uuid.FromStringOrNil(impersonatorInfo.Claims.Subject),
			Profile: userstore.Record{
				"name":           impersonatorInfo.Claims.Name,
				"email":          impersonatorInfo.Claims.Email,
				"email_verified": impersonatorInfo.Claims.EmailVerified,
				"picture":        impersonatorInfo.Claims.Picture,
			},
		}
	}

	jsonapi.Marshal(w, resp)
}

type impersonateUserRequest struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	TargetUserID uuid.UUID `json:"target_user_id"`
}

func (h *handler) impersonateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req impersonateUserRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	ten, err := h.storage.GetTenant(ctx, req.TenantID)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("tenant ID unrecognized: %v", req.TenantID), http.StatusBadRequest)
		return
	}

	tokenSource, err := m2m.GetM2MTokenSource(ctx, ten.ID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	userInfo := MustGetUserInfo(r)
	userEventClient, err := userevent.NewClient(ten.TenantURL, tokenSource, security.PassXForwardedFor())
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := userEventClient.ReportEvents(ctx, []userevent.UserEvent{
		{
			BaseModel: ucdb.NewBase(),
			Type:      "impersonate_user",
			UserAlias: userInfo.Claims.Subject,
			Payload: userevent.Payload{
				"TargetID": req.TargetUserID.String(),
				"TenantID": req.TenantID.String(),
			},
		},
	}); err != nil {
		uclog.Errorf(ctx, "error reporting impersonate_user event: %v", err)
	}

	session, err := h.sessionMgr.GetAuthSession(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("Failed to get `auth-session` cookie: %w", err), jsonapi.Code(http.StatusInternalServerError))
		return
	}

	claims, err := ucjwt.ParseUCClaimsUnverified(session.RefreshToken)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("Failed to parse refresh token: %w", err), jsonapi.Code(http.StatusInternalServerError))
		return
	}

	// TODO: instead of rejecting the request, we can redirect the user to the employee login app of the other tenant

	if ten.TenantURL != claims.Issuer {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "Unable to impersonate user in different tenant than one used to log in"), jsonapi.Code(http.StatusForbidden))
		return
	}

	if session.ImpersonatorIDToken != "" && session.ImpersonatorIDToken != impersonationRequestPlaceholder {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("Already impersonating a user"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	session.ImpersonatorIDToken = impersonationRequestPlaceholder
	if err := h.sessionMgr.SaveSession(ctx, w, session); err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("Failed to save session: %w", err), jsonapi.Code(http.StatusInternalServerError))
		return
	}

	plexClient := plex.NewClient(ten.TenantURL, tokenSource, security.PassXForwardedFor())

	resp, err := plexClient.ImpersonateUser(ctx, session.RefreshToken, req.TargetUserID.String())
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h *handler) unimpersonateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	session, err := h.sessionMgr.GetAuthSession(r)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("Failed to get `auth-session` cookie: %w", err), http.StatusInternalServerError)
		return
	}

	if session.ImpersonatorIDToken == "" || session.ImpersonatorIDToken == impersonationRequestPlaceholder {
		uchttp.Error(ctx, w, ucerr.Errorf("No impersonation session to unimpersonate"), http.StatusBadRequest)
		return
	}

	if err := h.sessionMgr.unimpersonateSession(ctx, session); err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getRedirectPath just wraps some janky code that handles accessing console
// by a specific regional URL, eg. console.aws-us-east-1.staging.userclouds.com
// and ensures that once you've specified that path, you stay in that region
func (h *handler) getRedirectURL(requestHost string, path string) string {
	u := GetConsoleURL(requestHost, h.getConsoleURLCallback())

	// because service.Endpoint.BaseURL() is slightly different from url.URL.URL(),
	// we cons up a new endpoint and use BaseURL to prevent splitting logic
	e := service.NewEndpointFromURL(u)
	return e.BaseURL() + path
}

// GetConsoleURL gets an updated console URL to match the host-requested URL
// if-and-only-if the http request was directed to a specific regional instance
// of console.
func GetConsoleURL(requestHost string, consoleURL *url.URL) *url.URL {
	if consoleURL.Host == requestHost {
		return consoleURL
	}
	// Construct a regionalHost URL based on the current region and the base console URL.
	var regionalHost string
	if kubernetes.IsKubernetes() {
		regionalHost = fmt.Sprintf("console.%s-eks.%s", region.Current(), strings.TrimPrefix(consoleURL.Host, "console."))
	} else {
		regionalHost = fmt.Sprintf("console.%s.%s", region.Current(), strings.TrimPrefix(consoleURL.Host, "console."))
	}

	if requestHost != regionalHost {
		return consoleURL
	}
	return &url.URL{
		Scheme: consoleURL.Scheme,
		Host:   regionalHost,
		Path:   consoleURL.Path,
	}
}
