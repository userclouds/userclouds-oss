package saml

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/crewjam/saml"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/tenantplex"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
)

var sessionMaxAge = time.Hour

const sessionCookieName = "saml-session"

// GetSession returns the *Session for this request.
//
// If the remote user has specified a username and password in the request
// then it is validated against the user database. If valid it sets a
// cookie and returns the newly created session object.
//
// If the remote user has specified invalid credentials then a login form
// is returned with an English-language toast telling the user their
// password was invalid.
//
// If a session cookie already exists and represents a valid session,
// then the session is returned
//
// If neither credentials nor a valid session cookie exist, this function
// sends a login form and returns nil.
// TODO: this masks some errors that maybe should be logged (like a DB connection failure
// in getSession), but mostly they are client-side eg. bad-cookie that we *probably* don't want to log?
func (h *handler) GetSessionOrRedirect(w http.ResponseWriter, r *http.Request, req *IdpAuthnRequest) *storage.SAMLSession {
	session, err := h.getSession(r)
	if err != nil {
		h.loginRedirect(w, r, req)
		return nil
	}

	return session
}

func (h *handler) getSession(r *http.Request) (*storage.SAMLSession, error) {
	ctx := r.Context()

	sessionCookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	uclog.Debugf(ctx, "got session cookie: %s", sessionCookie.Value)

	sid, err := uuid.FromString(sessionCookie.Value)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	s := tenantconfig.MustGetStorage(ctx)
	session, err := s.GetSAMLSession(ctx, sid)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if TimeNow().After(session.ExpireTime) {
		return nil, ucerr.New("session expired")
	}
	return session, nil
}

// NewAuthenticator creates a new authenticator object
func NewAuthenticator(ctx context.Context, redirectURL string, app *tenantplex.App) (*oidc.Authenticator, error) {
	tu := tenantconfig.MustGetTenantURLString(ctx)

	return oidc.NewAuthenticator(
		ctx,
		tu,
		app.ClientID,
		app.ClientSecret,
		fmt.Sprintf("%s/%s/%v", tu, redirectURL, app.ClientID),
	)
}

// AuthCallbackPath is the path to the callback handler
const AuthCallbackPath = "saml/callback"

func (h *handler) loginRedirect(w http.ResponseWriter, r *http.Request, req *IdpAuthnRequest) {
	ctx := r.Context()

	tu := tenantconfig.MustGetTenantURLString(ctx)

	_, app, err := h.getIDP(r)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	authr, err := NewAuthenticator(ctx, AuthCallbackPath, app)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	// if we got here, we didn't find a SAML session, so we need to create one and redirect
	redirectTo := fmt.Sprintf("%s/saml/sso/%v", tu, app.ClientID)

	session := &storage.SAMLSession{
		BaseModel:        ucdb.NewBase(),
		ExpireTime:       TimeNow().Add(sessionMaxAge),
		State:            fmt.Sprintf("%s#%s", crypto.MustRandomBase64(32), url.QueryEscape(redirectTo)),
		RequestBuffer:    req.RequestBuffer,
		RelayState:       req.RelayState,
		Groups:           []string{},         // TODO better factoring to avoid validate fails
		CustomAttributes: []saml.Attribute{}, // TODO better factoring to avoid validate fails
	}

	s := tenantconfig.MustGetStorage(ctx)
	err = s.SaveSAMLSession(ctx, session)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    session.ID.String(),
		MaxAge:   int(sessionMaxAge.Seconds()),
		HttpOnly: true,
		Secure:   r.URL.Scheme == "https",
		Path:     "/",
	})
	uclog.Debugf(ctx, "setting session cookie %v", session.ID.String())

	authCodeURL := authr.Config.AuthCodeURL(session.State)
	uclog.Debugf(ctx, "redirecting to login, state: '%s'", session.State)
	uchttp.Redirect(w, r, authCodeURL, http.StatusTemporaryRedirect)
}

// LoginCallback handles the OIDC callback from Plex and finishes the SAML SSO flow
func (h *handler) LoginCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	_, app, err := h.getIDP(r)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	session, err := h.getSession(r)
	if err != nil {
		uchttp.Error(ctx, w, ucerr.Errorf("Failed to get session: %w", err), http.StatusInternalServerError)
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
		uchttp.Error(ctx, w, ucerr.New("empty redirect URL"), http.StatusBadRequest)
		return
	}

	authr, err := NewAuthenticator(ctx, AuthCallbackPath, app)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	tokenInfo, statusCode, err := authr.ProcessAuthCodeCallback(r, session.State)
	if err != nil {
		uchttp.Error(ctx, w, err, statusCode)
		return
	} else if statusCode < 200 || statusCode > 299 {
		uclog.Errorf(ctx, "OIDC callback failed: %d", statusCode)
		// Don't return content in the response, just a code.
		w.WriteHeader(statusCode)
		return
	}

	session.UserEmail = tokenInfo.Claims.Email
	session.NameID = tokenInfo.Claims.Email
	session.ExpireTime = TimeNow().Add(sessionMaxAge)
	session.Index = hex.EncodeToString(randomBytes(32))
	session.UserName = tokenInfo.Claims.Name
	session.UserEmail = tokenInfo.Claims.Email

	s := tenantconfig.MustGetStorage(ctx)
	if err := s.SaveSAMLSession(ctx, session); err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	idpReq := &IdpAuthnRequest{
		RequestBuffer: session.RequestBuffer,
		RelayState:    session.RelayState,
		Now:           session.Created,
		HTTPRequest:   r, // used to determine IDP
	}

	if err := h.ValidateRequest(idpReq); err != nil {
		uchttp.Error(ctx, w, err, http.StatusBadRequest)
		return
	}

	h.ContinueSSO(w, r, idpReq, session)
}
