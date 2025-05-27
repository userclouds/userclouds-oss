package jsonclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonclient"
)

func TestHeaderFuncOption(t *testing.T) {
	ctx := context.Background()

	type k int
	var key k = 1
	headerName := "myheader"
	var val string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Header.Get(headerName), val)
	}))

	val = "foo"
	first := context.WithValue(ctx, key, val)
	client := jsonclient.New(srv.URL, jsonclient.PerRequestHeader(func(ctx context.Context) (string, string) {
		v := ctx.Value(key).(string)
		return headerName, v
	}))
	assert.NoErr(t, client.Get(first, "/", nil))

	val = "bar"
	second := context.WithValue(ctx, key, val)
	assert.NoErr(t, client.Get(second, "/", nil))

	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	}))
	client = jsonclient.New(srv.URL, jsonclient.PerRequestHeader(func(ctx context.Context) (string, string) {
		return "", ""
	}))
	assert.NoErr(t, client.Get(ctx, "/", nil))
}
