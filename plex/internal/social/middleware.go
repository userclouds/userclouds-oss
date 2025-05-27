package social

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/middleware"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
)

const stateSeparator = "|"

func decodeState(r *http.Request) (state string, sessionID uuid.UUID, tenantURL string, err error) {
	state = r.URL.Query().Get("state")
	if state == "" {
		return "", uuid.Nil, "", ucerr.New("query does not contain parameter 'state'")
	}

	stateParts := strings.Split(state, stateSeparator)
	if len(stateParts) != 2 {
		return "", uuid.Nil, "", ucerr.Errorf("query parameter 'state' does is not of the form <sessionID>|<tenantURL>: \"%s\"", state)
	}

	sessionID, err = uuid.FromString(stateParts[0])
	if err != nil {
		return "", uuid.Nil, "", ucerr.Wrap(err)
	}

	return state, sessionID, stateParts[1], nil
}

// EncodeState encodes the state passed as part of a social login request to a social provider
func EncodeState(sessionID uuid.UUID, tenantURL string) string {
	return fmt.Sprintf("%s%s%s", sessionID.String(), stateSeparator, tenantURL)
}

// Middleware is used to replace the request Host with the tenant URL if this is a localhost request
// NOTE: this is only supported in the dev environment
func Middleware() middleware.Middleware {
	localHostRegexp := regexp.MustCompile("^localhost.*")
	u := universe.Current()
	return middleware.Func((func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if u.IsDev() && localHostRegexp.MatchString(strings.ToLower(r.Host)) {
				if _, _, tenantURL, err := decodeState(r); err == nil {
					if parsedTenantURL, err := url.Parse(tenantURL); err == nil {
						r.Host = parsedTenantURL.Host
					}
				}
			}

			next.ServeHTTP(w, r.WithContext(r.Context()))
		})
	}))
}
