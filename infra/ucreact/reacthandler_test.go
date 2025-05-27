package ucreact_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucreact"
)

type handlerResponse struct {
	Text string `json:"text"`
}

func ensureResponseText(t *testing.T, w *httptest.ResponseRecorder, text string) {
	t.Helper()
	res := w.Result()
	defer res.Body.Close()
	var resp handlerResponse
	err := json.NewDecoder(res.Body).Decode(&resp)
	assert.NoErr(t, err)
	assert.Equal(t, resp.Text, text)
}

func TestFallbackHandler(t *testing.T) {
	primaryHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("testheader", "testvalue")
		if r.RequestURI == "/validpath" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(handlerResponse{Text: "validpath"})
		} else if r.RequestURI == "/failpath" {
			w.WriteHeader(http.StatusBadRequest)
		}
		w.WriteHeader(http.StatusNotFound)
	})
	fallbackHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("fallbackheader", "fallbackvalue")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(handlerResponse{Text: "fallback"})
	})
	h := ucreact.NewFallbackIfNotFoundHandler(primaryHandler, fallbackHandler)

	t.Run("ValidPath", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/validpath", nil)
		h.ServeHTTP(w, r)
		assert.Equal(t, w.Result().StatusCode, http.StatusCreated)
		// Primary handler header should be set, not fallback
		assert.Equal(t, w.Result().Header.Get("testheader"), "testvalue")
		assert.Equal(t, w.Result().Header.Get("fallbackheader"), "")
		ensureResponseText(t, w, "validpath")
	})

	t.Run("FailPath", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/failpath", nil)
		h.ServeHTTP(w, r)
		// Primary handler header should be set, not fallback
		assert.Equal(t, w.Result().Header.Get("testheader"), "testvalue")
		assert.Equal(t, w.Result().Header.Get("fallbackheader"), "")
		assert.Equal(t, w.Result().StatusCode, http.StatusBadRequest)
	})

	t.Run("NotFound", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/anotherpath", nil)
		h.ServeHTTP(w, r)
		assert.Equal(t, w.Result().StatusCode, http.StatusOK)
		// Primary handler header should NOT get set; only fallback
		assert.Equal(t, w.Result().Header.Get("testheader"), "")
		assert.Equal(t, w.Result().Header.Get("fallbackheader"), "fallbackvalue")
		ensureResponseText(t, w, "fallback")
	})
}
