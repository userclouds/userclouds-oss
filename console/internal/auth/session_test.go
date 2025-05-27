package auth_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"

	"userclouds.com/console/internal/auth"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/testdb"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/testkeys"
	"userclouds.com/test/testlogtransport"
)

func TestSession(t *testing.T) {
	ctx := context.Background()
	tdb := testdb.New(t, migrate.NewTestSchema(companyconfig.Schema))
	storage, err := companyconfig.NewStorage(ctx, tdb, testhelpers.NewCacheConfig())
	assert.NoErr(t, err)
	s := auth.NewSessionManager(storage)

	dummyRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	session, err := s.GetAuthSession(dummyRequest)
	assert.NoErr(t, err)

	// NB: the very very long claims exist to test that we don't fail
	// at 4096 char cookies like we used to in July 2022 ... ask me how I know
	claims := oidc.UCTokenClaims{
		StandardClaims: oidc.StandardClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "1234"}},
		Email:          "foo@contoso.com",
		Name:           "foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz foo bar baz ",
		Picture:        "https://lh3.googleusercontent.com/a-/AFdZucrWpyALXBNtPm76Q98j-R6sjvkeD75DpscJEJr7_HU9H_RBUt0CTMb1_I5MhwPZOq2hDgzK4M4mvSoBSXl9S-v6E2QzQu8IPa0KPlhrM5O2iVBlOth5FaJjHTnOm4htSZL-DaWe1FNspm_wrwhnlkP9xJhkwkfwzPLhoXCm32weS7wpCmdjM_wFssM0ZsFLRmwbxnwzeLhfTFwixjh6ZN0-pGM-zXNM3bJrv6MXV7MXU4mF7JyWhcLx2yPkdgtkVAwtjVxYydRmBGSG26PD2FBoZs4nn1Y65oabySqGysgzWKtNiJZBRqKF7pyMW0AT0TcM827JI93L9Rj3_QNpOckbHOU7LcoDN5pVQspkzCRYoZqOGmY4K-cBRs9XzrJ_6cRHyIEI7pEiSGmxk34tZxxl2IIMSNbRRkJ5VhjdGdPnQh64m10p-hA1vTNjn1-PTnWezAE5H_dVkk9dcp-Ml_apO43LIXmalmKYHUgd8O_uL2x54QgO1Ax_HoM_gkK4SgkAv7_428nNSJGRxwpKcGOoT5gpELDoLDwlastC4LPmXOOXVJVIeWO6bvYpP9wUwNsm1qIwOpjxCv2aPZn46wWTwIvPFCz0jktgcy3myStv6GfU0K-9Kl1vX2aosvZUJpH_FtL_F6etB8ClnEKv2mp0r4-sqkyo-yfF6fYOQBA8ZDioCX7-JdgvtdMtRBh2ek25pxtW2DPPLGpldSS6a1gLPAqVwwLwzHe2vvyGmxYHlcPnVRme40nWnSaqBaulwepD4k0c=s96-c",
	}
	idT, err := ucjwt.CreateToken(ctx,
		testkeys.GetPrivateKey(t),
		testkeys.Config.KeyID,
		uuid.Must(uuid.NewV4()),
		claims,
		"testissuer",
		60*60 /* an hour should be ok for testing */)
	assert.NoErr(t, err)

	cookieResponseRecorder := httptest.NewRecorder()

	session.IDToken = idT
	session.AccessToken = idT
	session.RefreshToken = idT
	assert.IsNil(t, s.SaveSession(ctx, cookieResponseRecorder, session), assert.Must())

	cookies := cookieResponseRecorder.Result().Cookies()
	assert.Equal(t, len(cookies), 1)
	assert.Equal(t, cookies[0].Name, auth.SessionCookieName)

	redirectHandler := s.RedirectIfNotLoggedIn().Apply(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusAccepted)
	}))

	failHandler := s.FailIfNotLoggedIn().Apply(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusAccepted)
	}))

	t.Run("NoRedirectIfAuthed", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		r.AddCookie(cookies[0])
		redirectHandler.ServeHTTP(w, r)
		assert.Equal(t, w.Code, http.StatusAccepted)
	})

	t.Run("NoFailIfAuthed", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		r.AddCookie(cookies[0])
		failHandler.ServeHTTP(w, r)
		assert.Equal(t, w.Code, http.StatusAccepted)
	})

	t.Run("RedirectIfNotAuthed", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/some/path", nil)

		redirectHandler.ServeHTTP(w, r)
		assert.Equal(t, w.Code, http.StatusTemporaryRedirect)
		location := w.Header().Get("Location")
		assert.Equal(t, location, fmt.Sprintf("%s?redirect_to=%s", auth.RedirectPath, url.QueryEscape("/some/path")))
	})

	t.Run("FailIfNotAuthed", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/some/path", nil)

		tt := testlogtransport.InitLoggerAndTransportsForTests(t)

		failHandler.ServeHTTP(w, r)
		assert.Equal(t, w.Code, http.StatusUnauthorized)

		// make sure we log this as a warning not an error
		tt.AssertMessagesByLogLevel(uclog.LogLevelWarning, 1)
		tt.AssertMessagesByLogLevel(uclog.LogLevelError, 0)
	})
}
