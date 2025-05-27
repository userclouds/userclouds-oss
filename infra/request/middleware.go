package request

import (
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/kubernetes"
	"userclouds.com/infra/middleware"
	"userclouds.com/infra/namespace/universe"
)

const ucMachineHeader = "X-Userclouds-Machine"

// Middleware guarantees that all incoming HTTP requests have a unique ID assigned
// to enable logging & tracing. This could go in LoggingMW but more explicit here
// and compiler will likely optimize it out anyway
func Middleware() middleware.Middleware {
	machineName := ""
	if universe.Current().IsKubernetes() {
		machineName = kubernetes.GetPodName()
	}
	return middleware.Func((func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := uuid.Must(uuid.NewV4())
			ctx := SetRequestData(r.Context(), r, id)
			if machineName != "" {
				w.Header().Add(ucMachineHeader, machineName)
			}
			w.Header().Add(RequestIDHeader, id.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}))
}
