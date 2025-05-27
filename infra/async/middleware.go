package async

import (
	"net/http"

	"userclouds.com/infra/middleware"
)

// PanicRecoverMiddleware makes sure that if we panic during request execution we capture that panic in the logs
// before exiting.
// TODO at some point we should not exit the process and instead return 500 and continue.
func PanicRecoverMiddleware() middleware.Middleware {
	defer recoverPanic()

	return middleware.Func((func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer recoverPanic()
			next.ServeHTTP(w, r)
		})
	}))
}
