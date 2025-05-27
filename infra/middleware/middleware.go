package middleware

import (
	"io"
	"net/http"
)

// Middleware decorates a server-side handler.
type Middleware interface {
	Apply(http.Handler) http.Handler
}

// Func is a simple Middleware implementation.
type Func func(http.Handler) http.Handler

// Apply implements Middleware.
func (f Func) Apply(next http.Handler) http.Handler {
	return f(next)
}

// Chain takes a series of HTTP server middlewares and returns a single
// middleware which applies them in order. Any entry in the list may be nil.
func Chain(mws ...Middleware) Middleware {
	return Func(func(h http.Handler) http.Handler {
		// We need to wrap in reverse order to have the first middleware from
		// the list get the request first.
		for i := len(mws) - 1; i >= 0; i-- {
			if mw := mws[i]; mw != nil {
				h = mw.Apply(h)
			}
		}
		return h
	})
}

// ReadAll guarantees that all request bodies are read to completion so the
// server can re-use connections.
func ReadAll() Middleware {
	return Func(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			io.Copy(io.Discard, r.Body)
		})
	})
}
