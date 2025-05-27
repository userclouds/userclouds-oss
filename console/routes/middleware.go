package routes

import (
	"net/http"
	"strings"

	"userclouds.com/console/internal/auth"
	"userclouds.com/infra/middleware"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
)

// this middleware exists to prevent host injection against console (it's handled elsewhare by
// multitenant.Middleware because everything else is multi-tenant by hostname)
// NB: we use GetConsoleURLCallback instead of just a string because we want to be able to
// to update the console URL in tests to httptest.NewServer(), and this means
// we have to call it every request. In all cases it's effectively a constant from the
// first request, but it's cheap enough I'm not adding complexity by caching it
// (in both prod and test paths, it is a function that returns a pointer deref)
// TODO (sgarrity 7/24): logserver is also "vulnerable" here but it doesn't return HTML
// so it's not really vulnerable, and I don't want to mess with it right now.
func rejectNonConsoleHostMiddleware(cb auth.GetConsoleURLCallback) middleware.Middleware {
	return middleware.Func(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if host := auth.GetConsoleURL(r.Host, cb()).Host; r.Host != host {
				if strings.Split(r.Host, ":")[0] != strings.Split(host, ":")[0] {
					uclog.Warningf(ctx, "Invalid host %v (expected %v)", r.Host, host)
					uchttp.Error(ctx, w, ucerr.Friendlyf(nil, "Invalid host %v", r.Host), http.StatusBadRequest)
					return
				}
				uclog.Warningf(ctx, "Request host port is different %v (expected %v)", r.Host, host)
			}
			next.ServeHTTP(w, r)
		})
	})
}
