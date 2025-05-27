package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"userclouds.com/infra/assert"
	. "userclouds.com/infra/middleware"
)

func TestChain(t *testing.T) {
	var order string

	first := Func(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order += "first"
			next.ServeHTTP(w, r)
		})
	})

	second := Func(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order += "second"
			next.ServeHTTP(w, r)
		})
	})

	mw := Chain(first, second)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello world")
	})

	handler := mw.Apply(h)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, r)

	assert.True(t, strings.HasPrefix(order, "first"))
}
