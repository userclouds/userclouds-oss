package jsonclient_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/uctest"
)

type reqT struct {
	Foo string
}
type resT struct {
	Bar string
}
type testFunc func(t *testing.T, err error, res resT)

func TestPost(t *testing.T) {
	testClient(t,
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Method, http.MethodPost)
			var req reqT
			assert.IsNil(t, jsonapi.Unmarshal(r, &req))
			assert.Equal(t, req.Foo, "test")

			res := resT{"me"}
			jsonapi.Marshal(w, res)
		},
		func(t *testing.T, err error, res resT) {
			assert.NoErr(t, err)
			assert.Equal(t, res.Bar, "me")
		})
}

func TestPostError(t *testing.T) {
	ctx := context.Background()
	testClient(t,
		func(w http.ResponseWriter, _ *http.Request) {
			jsonapi.MarshalError(ctx, w, ucerr.New("blah"))
		},
		func(t *testing.T, err error, res resT) {
			assert.NotNil(t, err)
			assert.Equal(t, res, resT{})
		})
}

func testOAuthErrorHandler(w http.ResponseWriter, r *http.Request) {
	jsonapi.MarshalError(r.Context(), w, ucerr.OAuthError{
		ErrorType: "some_error",
		ErrorDesc: "an error occurred",
		Code:      http.StatusExpectationFailed,
	})
}
func TestOAuthError(t *testing.T) {
	oautheResult := func(t *testing.T, err error, res resT) {
		assert.NotNil(t, err)
		var oauthe jsonclient.JCOAuthError
		assert.True(t, errors.As(err, &oauthe))
		assert.Equal(t, oauthe.ErrorType, "some_error")
		assert.Equal(t, oauthe.ErrorDesc, "an error occurred")
		assert.Equal(t, oauthe.Code, http.StatusExpectationFailed)
		assert.Equal(t, res, resT{})
	}
	result := func(t *testing.T, err error, res resT) {
		assert.NotNil(t, err)
		var oauthe jsonclient.JCOAuthError
		assert.False(t, errors.As(err, &oauthe))
		var jerr jsonclient.Error
		assert.True(t, errors.As(err, &jerr))
		assert.Equal(t, jerr.StatusCode, http.StatusExpectationFailed)
		assert.Equal(t, res, resT{})
	}
	// Test with and without explicit flag to capture OAuth Errors
	testClient(t, testOAuthErrorHandler, oautheResult, jsonclient.ParseOAuthError())
	testClient(t, testOAuthErrorHandler, result)
}

func testClient(t *testing.T, handlerFunc http.HandlerFunc, resultFunc testFunc, opts ...jsonclient.Option) {
	ctx := context.Background()

	srv := httptest.NewServer(handlerFunc)
	defer srv.Close()

	body := reqT{"test"}
	var res resT
	opts = append(opts, jsonclient.Header("foo", "bar"))
	client := jsonclient.New(srv.URL, opts...)
	err := client.Post(ctx, "/", body, &res)
	resultFunc(t, err, res)
}

func TestAutomaticTokenRefresh(t *testing.T) {
	ctx := context.Background()

	j := uctest.CreateJWT(t, oidc.UCTokenClaims{}, "http://contoso.com")

	var numTokenRequests, numRequests int
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		numRequests++
		if r.URL.Path == "/token" {
			numTokenRequests++
			// this is janky but it conforms "enough" to the oauth CC spec that it works
			jsonapi.Marshal(w, map[string]any{"access_token": j,
				"token_type": "Bearer",
				"expires_in": 3600})
			return
		}
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	// should automatically request a token on the first call
	client := jsonclient.New(srv.URL,
		jsonclient.ClientCredentialsTokenSource(srv.URL+"/token", "client_id", "client_secret", nil))

	assert.NoErr(t, client.Get(ctx, "/", nil))
	assert.Equal(t, numTokenRequests, 1)
	assert.Equal(t, numRequests, 2)

	// second request shouldn't request a token again
	assert.NoErr(t, client.Get(ctx, "/", nil))
	assert.Equal(t, numTokenRequests, 1)
	assert.Equal(t, numRequests, 3)

	// make sure we didn't break something with tokensource is unused
	client = jsonclient.New(srv.URL)
	assert.NoErr(t, client.Get(ctx, "/", nil))
	assert.Equal(t, numTokenRequests, 1)
	assert.Equal(t, numRequests, 4)
}

func TestRetries(t *testing.T) {
	ctx := context.Background()

	client := jsonclient.New("http://localhost:1234/")
	err := client.Get(ctx, "/", nil)
	assert.NotNil(t, err)

	// TODO: I don't love this test, but it's the best I can think of for now
	// since we actually don't want it to hit eg. a mock handler
	assert.DoesNotContain(t, err.Error(), "retry")

	client = jsonclient.New("http://localhost:1234/", jsonclient.RetryNetworkErrors(true))
	err = client.Get(ctx, "/", nil)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "max retries exceeded")
}
