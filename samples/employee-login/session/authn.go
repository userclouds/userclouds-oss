package session

import (
	"context"
	"log"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/gorilla/sessions"
	"github.com/userclouds/userclouds/samples/employee-login/app"

	"userclouds.com/infra/middleware"
)

// TODO: (#77): Fold the session management code & middleware into a shared
// infra/auth lib to be used by console, sample apps, plex/IDP, etc.

func getSessionFromCookie(r *http.Request) (*sessions.Session, bool) {
	// Look up the auth session cookie and find the session data
	// associated with it (currently using a FilesystemStore which
	// stores a session ID in the cookie but session data in the server's
	// temp directory).
	// NOTE: move to using the DB for this like Steve's tasks app.
	session, err := app.GetAuthSession(r)
	if err != nil {
		// An error here likely means a bad/expired cookie; treat as
		// unauthenticated and move on.
		return nil, false
	}

	if session.IsNew {
		// No cookie found, not logged in.
		return nil, false
	}

	// Check required values
	if _, ok := session.Values["profile"]; !ok {
		log.Printf("No `profile` in `session.Values`: %v", session.Values)
		return nil, false
	}

	if _, ok := session.Values["user_id"]; !ok {
		log.Printf("No `user_id` in `session.Values`: %v", session.Values)
		return nil, false
	}

	// TODO: (#72) Verify token (which also re-checks expiration), but even
	// better yet:

	return session, true
}

type key int

// UserSessionKey defines the key to access a session in a context
const UserSessionKey key = 0

func setupRequestContext(session *sessions.Session, r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), UserSessionKey, session.Values))
}

// GetAccessToken returns the access token from the context
func GetAccessToken(r *http.Request) string {
	values, ok := r.Context().Value(UserSessionKey).(map[any]any)
	if !ok || values == nil {
		panic("No session found in request context, this is an auth bug.")
	}
	accessToken, ok := values["access_token"]
	if !ok {
		panic("Malformed session found in request context (no 'access_token'), this is an auth bug.")
	}
	return accessToken.(string)
}

// GetProfile returns a user profile from the context
func GetProfile(r *http.Request) map[string]any {
	values, ok := r.Context().Value(UserSessionKey).(map[any]any)
	if !ok || values == nil {
		panic("No session found in request context, this is an auth bug.")
	}
	profile, ok := values["profile"]
	if !ok {
		panic("Malformed session found in request context (no 'profile'), this is an auth bug.")
	}
	castProfile, ok := profile.(map[string]any)
	if !ok {
		panic("Malformed session found in request context ('profile' is wrong type), this is an auth bug.")
	}
	return castProfile
}

// GetUserID returns a user ID from the context
func GetUserID(r *http.Request) uuid.UUID {
	values, ok := r.Context().Value(UserSessionKey).(map[any]any)
	if !ok || values == nil {
		panic("No session found in request context, this is an auth bug.")
	}
	if _, ok := values["user_id"]; !ok {
		panic("Malformed session found in request context (no 'user_id'), this is an auth bug.")
	}
	id, ok := values["user_id"].(string)
	if !ok {
		panic("Malformed session found in request context ('user_id' is not a string), this is an auth bug.")
	}
	return uuid.Must(uuid.FromString(id))
}

// RedirectIfNotAuthenticated is middleware to force login if needed
func RedirectIfNotAuthenticated(redirectURL string) middleware.Middleware {
	return middleware.Func(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, isAuthed := getSessionFromCookie(r)
			if !isAuthed {
				http.Redirect(w, r, redirectURL, http.StatusSeeOther)
			} else {
				next.ServeHTTP(w, setupRequestContext(session, r))
			}
		})
	})
}

// RedirectIfAuthenticated is middleware to automatically bypass login if logged in
func RedirectIfAuthenticated(redirectURL string) middleware.Middleware {
	return middleware.Func(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, isAuthed := getSessionFromCookie(r)
			if isAuthed {
				http.Redirect(w, setupRequestContext(session, r), redirectURL, http.StatusSeeOther)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	})
}
