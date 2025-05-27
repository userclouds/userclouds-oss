package jsonclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/internal/security"
)

func TestHeaders(t *testing.T) {
	testXFFClient(t,
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Header.Get("foo"), "bar")
			assert.Equal(t, r.Header.Get("X-Forwarded-For"), "1.1.1.1,2.2.2.2")
		},
		func(_ *testing.T, _ error, _ resT) {

		})
}

func testXFFClient(t *testing.T, handlerFunc http.HandlerFunc, resultFunc testFunc, opts ...jsonclient.Option) {
	ctx := context.Background()

	s := security.Status{IPs: []string{"1.1.1.1", "2.2.2.2"}}
	ctx = security.SetSecurityStatus(ctx, &s)

	srv := httptest.NewServer(handlerFunc)
	defer srv.Close()

	body := reqT{"test"}
	var res resT
	opts = append(opts,
		jsonclient.Header("foo", "bar"),
		jsonclient.PerRequestHeader(func(ctx context.Context) (string, string) {
			return "X-Forwarded-For", strings.Join(security.GetSecurityStatus(ctx).IPs, ",")
		}),
	)
	client := jsonclient.New(srv.URL, opts...)
	err := client.Post(ctx, "/", body, &res)
	resultFunc(t, err, res)
}
