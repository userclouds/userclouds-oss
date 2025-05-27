package uchttp

import (
	"net/http"

	"userclouds.com/infra/uclog"
)

// Redirect wraps http.Redirect to log the redirect.
func Redirect(w http.ResponseWriter, r *http.Request, path string, code int) {
	ctx := r.Context()
	uclog.Debugf(ctx, "redirecting to %s", path)
	http.Redirect(w, r, path, code)
}
