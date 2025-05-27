package ucv8go

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/internal/auditlog"
)

func TestNetworkRequest(t *testing.T) {
	ctx := context.Background()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// if we test Authorization here we need a more complex setup for basic auth vs this
		assert.Equal(t, r.Header.Get("UC-Test"), "foobar")

		u, p, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, u, "user")
		assert.Equal(t, p, "pass")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	}))
	defer srv.Close()

	jsc, _, err := NewJSContext(ctx, nil, auditlog.AccessPolicyCustom, nil)
	assert.NoErr(t, err)

	v, err := jsc.RunScript(`networkRequest({url: "`+srv.URL+`", headers: {"UC-Test": ["foobar"]}, auth: {user: "user", password: "pass"}})`, "main.js")
	assert.NoErr(t, err)
	assert.Equal(t, v.String(), "hello")
}
