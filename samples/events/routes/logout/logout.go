package logout

import (
	"net/http"
	"net/url"

	"github.com/userclouds/userclouds/samples/events/auth"

	"userclouds.com/infra/uchttp"
)

// Handler handles logout
func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var scheme string
	if r.TLS == nil {
		scheme = "http"
	} else {
		scheme = "https"
	}

	redirectURL, err := url.Parse(scheme + "://" + r.Host)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, auth.GetLogoutURL(auth.GetClientID(), redirectURL.String()), http.StatusTemporaryRedirect)
}
