package request_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/kubernetes"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/request"
)

func TestMiddleware(t *testing.T) {
	requestID := uuid.Nil
	h := func(code int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
			fmt.Fprintf(w, "%d", code)
			requestID = request.GetRequestID(r.Context())
			assert.NotEqual(t, requestID, uuid.Nil)
		})
	}

	handler := request.Middleware().Apply(h(http.StatusOK))
	w := doRequest(handler)
	assertRequestID(t, requestID, w)
	assert.Equal(t, w.Header().Get("X-Userclouds-Machine"), "")
}

func TestMiddlewareInKubernetes(t *testing.T) {
	requestID := uuid.Nil
	dummyHandler := func(code int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
			fmt.Fprintf(w, "%d", code)
			requestID = request.GetRequestID(r.Context())
			assert.NotEqual(t, requestID, uuid.Nil)
		})
	}

	for _, uv := range universe.AllUniverses() {
		if !uv.IsKubernetes() {
			continue
		}

		t.Setenv(universe.EnvKeyUniverse, string(uv))
		t.Setenv(kubernetes.EnvPodName, "newman")
		handler := request.Middleware().Apply(dummyHandler(http.StatusOK))
		w := doRequest(handler)
		assertRequestID(t, requestID, w)
		assert.Equal(t, w.Header().Get("X-Userclouds-Machine"), "newman")
	}
}

func assertRequestID(t *testing.T, requestID uuid.UUID, w *httptest.ResponseRecorder) {
	requestIDHeader := w.Header().Get("X-Request-Id")
	assert.NotEqual(t, requestIDHeader, "")
	id, err := uuid.FromString(requestIDHeader)
	assert.NoErr(t, err)
	assert.Equal(t, id, requestID)
}
func doRequest(handler http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}
