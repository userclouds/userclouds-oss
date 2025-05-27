package reactdev

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/request"
	"userclouds.com/infra/uclog"
)

// PlexUIRoot is the root path on Plex for the React-based UI.
// NOTE: there are 2 other things that need to be kept in sync:
// 1. `plexui/package.json` has a "homepage" property that must match this exactly
// 2. `plexui/src/index.tsx` sets `BrowserRouter.basename` to this value.
const PlexUIRoot = "/plexui"

// EnvPlexUIDevPort is an environment variable which, if set, tells Plex's server to render all UI from
// the React development server instead of the static built react bundle.
// Unlike Console, which is a true SPA, Plex is a set of mostly independent pages which are rendered based on
// server-driven interactions (e.g. redirecting the user agent to an OIDC authorize endpoint, clicking a magic link, etc).
// TODO: once this stabilizes, we can probably automate this with a `make` rule but it would need
// to (a) set the port env var and also (b) kick off both the sharedui and plexui in development mode,
// which (AFAIK) requires 3 separate shells.
const EnvPlexUIDevPort = "UC_PLEX_UI_DEV_PORT"

// UIBaseURL will return the correct base URL for Plex UI based on the context (host etc)
// of the current request + PlexUIRoot, potentially adjusting the port of the URL to point to a React
// development server if a set of conditions are met: EnvPlexUIDevPort is set to a valid port,
// the service is running in the dev universe, and the host is *.dev.userclouds.tools.
func UIBaseURL(ctx context.Context) *url.URL {
	u := &url.URL{}
	u.Host = request.GetHostname(ctx)
	u.Path = PlexUIRoot

	// Dev-only feature: send request to React development server login page instead of compiled login page.
	if !universe.Current().IsDev() {
		return u
	}

	plexUIDevPort := os.Getenv(EnvPlexUIDevPort)
	if len(plexUIDevPort) == 0 {
		// Env var not set; just ignore
		return u
	}

	port, err := strconv.ParseUint(plexUIDevPort, 10, 32)
	if err != nil {
		uclog.Warningf(ctx, "Error parsing env var '%s' (value: '%s'): %v", EnvPlexUIDevPort, plexUIDevPort, err)
		return u
	}

	hostAndPort := strings.Split(u.Host, ":")
	if len(hostAndPort) == 0 || len(hostAndPort) > 2 {
		uclog.Warningf(ctx, "Expected host to be of the form <host>:<port> or just <host>, but got %s instead", u.Host)
		return u
	}

	if !strings.HasSuffix(hostAndPort[0], "dev.userclouds.tools") {
		uclog.Warningf(ctx, "Expected host to end with 'dev.userclouds.tools' in dev mode, but got %s instead", u.Host)
		return u
	}

	u.Host = fmt.Sprintf("%s:%d", hostAndPort[0], port)
	return u
}
