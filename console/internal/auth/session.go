package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/middleware"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
)

// TODO: (#77): Fold the session management code & middleware into a shared
// infra/auth lib to be used by console, sample apps, plex/IDP, etc.

// SessionManager handles sessions for Console
type SessionManager struct {
	storage *companyconfig.Storage
}

// NewSessionManager creates a new cookie session manager for managing
// client web sessions.
func NewSessionManager(s *companyconfig.Storage) *SessionManager {
	return &SessionManager{
		storage: s,
	}
}

// SessionCookieName is the name of the cookie set on the client to associate with the server session
// -id now because we moved from a JWT in the cookie to just an ID that references the DB
const SessionCookieName = "auth-session-id"

const impersonationRequestPlaceholder = "requestingToken"

// GetAuthSession gets a user session from the cookie in the http request
func (sm *SessionManager) GetAuthSession(r *http.Request) (*companyconfig.Session, error) {
	ctx := r.Context()

	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		if !errors.Is(err, http.ErrNoCookie) {
			return nil, ucerr.Wrap(err)
		}

		// no cookie found, let's create a new session
		return &companyconfig.Session{
			BaseModel: ucdb.NewBase(),
		}, nil
	}

	id, err := uuid.FromString(cookie.Value)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	s, err := sm.storage.GetSession(ctx, id)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return s, nil
}

// ClearAuthSessionCookie modifies the http response to tell the client to clear the cookie
func (sm *SessionManager) ClearAuthSessionCookie(w http.ResponseWriter) {
	// TODO: we should also clean up the DB, and we should sweep DB even if this isn't called
	http.SetCookie(w, &http.Cookie{
		Name:    SessionCookieName,
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
	})
}

// SaveSession saves a session to the DB, and then ensures the cookie references it
func (sm *SessionManager) SaveSession(ctx context.Context, w http.ResponseWriter, s *companyconfig.Session) error {
	if err := sm.storage.SaveSession(ctx, s); err != nil {
		return ucerr.Wrap(err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:    SessionCookieName,
		Value:   s.ID.String(),
		Path:    "/",
		Expires: time.Now().UTC().AddDate(0, 0, 30), // TODO: customizable
	})

	return nil
}

// UserInfo holds core user info fields, derived from the AuthN token and the
// OIDC /userinfo endpoint. Because this is signed and stored server-side, the
// contents can be trusted.
// TODO: make this a real struct with a map for extended fields.
// TODO: maybe don't bother storing the /userinfo data and just the token+claims.
// TODO: this used to just embed oidc.TokenInfo, but we no longer store the access token etc
type UserInfo struct {
	RawIDToken string
	Claims     oidc.UCTokenClaims
}

// GetUserID returns the user ID of the auth'd user from the data determined
// via OIDC tokens/endpoints.
// NOTE: in general the OIDC subject is a string, but our IDP always uses UUIDs.
func (u *UserInfo) GetUserID() (uuid.UUID, error) {
	return uuid.FromString(u.Claims.Subject)
}

func getUserInfoFromMap(idToken string) (*UserInfo, error) {
	// TODO: (#72) Verify token (which also re-checks expiration).
	claims, err := ucjwt.ParseUCClaimsUnverified(idToken)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &UserInfo{
		RawIDToken: idToken,
		Claims:     *claims,
	}, nil
}

func (sm *SessionManager) unimpersonateSession(ctx context.Context, session *companyconfig.Session) error {

	// Move impersonator tokens to logged-in user tokens
	session.IDToken = session.ImpersonatorIDToken
	session.AccessToken = session.ImpersonatorAccessToken
	session.RefreshToken = session.ImpersonatorRefreshToken
	session.ImpersonatorIDToken = ""
	session.ImpersonatorAccessToken = ""
	session.ImpersonatorRefreshToken = ""

	if err := sm.storage.SaveSession(ctx, session); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// getSessionInfoFromCookie returns a UserInfo and associated Session associated with the auth session cookie if the UserInfo
// is valid and if the Session has an AccessToken and RefreshToken.
func (sm *SessionManager) getSessionInfoFromCookie(r *http.Request) (userInfo *UserInfo, impersonatorInfo *UserInfo, session *companyconfig.Session, err error) {
	// Get the session from the cookie -> DB record
	s, err := sm.GetAuthSession(r)
	if err != nil {
		// An error here likely means a bad/expired cookie; treat as
		// unauthenticated and move on.
		return nil, nil, nil, ucerr.Errorf("failed to load '%s' cookie: %w", SessionCookieName, err)
	}

	// TODO: taking advantage of Created.IsZero() is a bit of a BaseModel impl detail
	// should we have a .IsNew() method on BaseModel or ...?
	if s.Created.IsZero() {
		// No session found, not logged in.
		// use Warning so this doesn't log loudly as an error
		return nil, nil, nil, ucerr.NewWarning("no session found")
	}

	expired, err := ucjwt.IsExpired(s.RefreshToken)
	if err != nil {
		return nil, nil, nil, ucerr.Errorf("error parsing token in cookie: %w", SessionCookieName, err)
	}

	if expired {
		if s.ImpersonatorRefreshToken != "" {
			expiredImpersonator, err := ucjwt.IsExpired(s.ImpersonatorRefreshToken)
			if err != nil {
				return nil, nil, nil, ucerr.Errorf("error parsing impersonator token in cookie: %w", SessionCookieName, err)
			}
			if !expiredImpersonator {
				if err := sm.unimpersonateSession(r.Context(), s); err != nil {
					return nil, nil, nil, ucerr.Wrap(err)
				}
				expired = false
			}
		}
		if expired {
			return nil, nil, nil, ucerr.New("Token expired")
		}
	}

	userInfo, err = getUserInfoFromMap(s.IDToken)
	if err != nil {
		return nil, nil, nil, ucerr.Errorf("error parsing '%s' cookie: %w", SessionCookieName, err)
	}

	if s.AccessToken == "" {
		return nil, nil, nil, ucerr.Errorf("no access token session")
	}

	if s.RefreshToken == "" {
		return nil, nil, nil, ucerr.Errorf("no refresh token in session")
	}

	if s.ImpersonatorIDToken != "" && s.ImpersonatorIDToken != impersonationRequestPlaceholder {
		impersonatorInfo, err = getUserInfoFromMap(s.ImpersonatorIDToken)
		if err != nil {
			return nil, nil, nil, ucerr.Errorf("error parsing impersonator '%s' cookie: %w", SessionCookieName, err)
		}
	}

	return userInfo, impersonatorInfo, s, nil
}

type userInfoKeyType int

const (
	userInfoKey         userInfoKeyType = 0
	accessTokenKey      userInfoKeyType = 1
	refreshTokenKey     userInfoKeyType = 2
	impersonatorInfoKey userInfoKeyType = 3
)

// setupRequestContext puts the user info and access token from the session cookie into context.
func (sm *SessionManager) setupRequestContext(r *http.Request, userInfo *UserInfo, impersonatorInfo *UserInfo, s *companyconfig.Session) *http.Request {
	ctx := context.WithValue(r.Context(), userInfoKey, userInfo)
	ctx = context.WithValue(ctx, accessTokenKey, s.AccessToken)
	ctx = context.WithValue(ctx, refreshTokenKey, s.RefreshToken)

	if impersonatorInfo != nil {
		ctx = context.WithValue(ctx, impersonatorInfoKey, impersonatorInfo)
	}

	return r.WithContext(ctx)
}

// GetUserInfo returns a user profile from the context or an error, and is meant
// to be used in tandem with one of the session middlewares.
func GetUserInfo(r *http.Request) (*UserInfo, error) {
	userInfo, ok := r.Context().Value(userInfoKey).(*UserInfo)
	if !ok || userInfo == nil {
		return nil, ucerr.New("no UserInfo found in request context")
	}
	return userInfo, nil
}

// MustGetUserInfo is like GetUserInfo but panics on failure.
func MustGetUserInfo(r *http.Request) *UserInfo {
	userInfo, err := GetUserInfo(r)
	if err != nil {
		log.Fatalf("error getting user info in request to '%s' (did you forget to use middleware?): %v", r.RequestURI, err)
	}
	return userInfo
}

// GetImpersonatorInfo returns the impersonator's user profile from the context if it exists
func GetImpersonatorInfo(r *http.Request) *UserInfo {
	if userInfo, ok := r.Context().Value(impersonatorInfoKey).(*UserInfo); ok {
		return userInfo
	}
	return nil
}

// GetAccessTokens returns the access and refresh tokens for the logged in user from context or returns an error
func GetAccessTokens(ctx context.Context) (string, string, error) {
	accessToken, ok := ctx.Value(accessTokenKey).(string)
	if !ok || accessToken == "" {
		return "", "", ucerr.New("no access token found in request context")
	}
	refreshToken, ok := ctx.Value(refreshTokenKey).(string)
	if !ok || refreshToken == "" {
		return "", "", ucerr.New("no refresh token found in request context")
	}
	return accessToken, refreshToken, nil
}

// MustGetAccessTokens returns the access and refresh tokens for the logged in user or panics on failure
func MustGetAccessTokens(ctx context.Context) (string, string) {
	accessToken, refreshToken, err := GetAccessTokens(ctx)
	if err != nil {
		log.Fatalf("error getting access token in request: %v", err)
	}
	return accessToken, refreshToken
}

// RedirectIfNotLoggedIn is a simple middleware meant for interactive UI pages to
// redirect users to a login page if they're not logged in.
// For APIs, use `FailIfNotLoggedIn` instead.
func (sm *SessionManager) RedirectIfNotLoggedIn() middleware.Middleware {
	return middleware.Func(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userInfo, impersonatorInfo, s, err := sm.getSessionInfoFromCookie(r)
			if err != nil {
				uclog.Debugf(ctx, "error getting session info from cookie, redirecting to login")
				// Preserve original URL (path?query) after auth succeeds.
				// Note that r.RequestURI is NOT stripped by middleware.
				query := url.Values{}
				query.Add("redirect_to", r.RequestURI)
				redirectURL := url.URL{
					Path:     RedirectPath,
					RawQuery: query.Encode(),
				}
				uchttp.Redirect(w, r, redirectURL.String(), http.StatusTemporaryRedirect)
				return
			}
			next.ServeHTTP(w, sm.setupRequestContext(r, userInfo, impersonatorInfo, s))
		})
	})
}

// FailIfNotLoggedIn returns 401 if the user is not logged in or has bad
// authentication cookies/headers.
// Authorization checks can happen downstream of this, as this will guarantee
// that the user is authenticated on success.
// NOTE: a different error should be returned if the user *is* authenticated
// but does not have permissions on the object (403 Forbidden).
func (sm *SessionManager) FailIfNotLoggedIn() middleware.Middleware {
	return middleware.Func(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userInfo, impersonatorInfo, s, err := sm.getSessionInfoFromCookie(r)
			if err != nil {
				uchttp.Error(ctx, w, ucerr.Wrap(err), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, sm.setupRequestContext(r, userInfo, impersonatorInfo, s))
		})
	})
}
