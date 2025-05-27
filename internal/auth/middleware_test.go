package auth_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/oidc"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/uctest"
)

func TestBearerToken(t *testing.T) {
	jwt := uctest.CreateJWT(t, oidc.UCTokenClaims{}, "https://bob.test.userclouds.tools")

	h := auth.Middleware(uctest.JWTVerifier{}, uuid.Nil).Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world!"))
	}))

	rr := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))
	h.ServeHTTP(rr, r)
	assert.Equal(t, http.StatusOK, rr.Code)

	// reset and try without
	rr = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rr, r)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAccessTokenHeader(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.Must(uuid.NewV4())

	h := auth.Middleware(uctest.JWTVerifier{}, uuid.Nil).Apply(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world!"))
	}))

	rr := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(multitenant.SetTenantState(ctx, &tenantmap.TenantState{ID: tenantID}))
	token, err := m2m.GetM2MSecret(ctx, tenantID)
	assert.NoErr(t, err)
	r.Header.Set("Authorization", fmt.Sprintf("AccessToken %s", token))
	h.ServeHTTP(rr, r)
	assert.Equal(t, http.StatusOK, rr.Code)

	// reset and try without
	rr = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rr, r)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
