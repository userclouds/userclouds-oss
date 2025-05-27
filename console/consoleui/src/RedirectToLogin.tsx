import { GetAuthURL } from './Auth';

// NOTE: In production, the Go server handles bad/missing auth cookies and redirects
// the user to login. But in dev mode (which we use a lot), the SPA is hosted by Express
// (not our Go server) so we need to detect 401s and redirect the user to login.
// It's not great that this behavior diverges :(
//
// TODO: I think there's a bug here due to a race which manifests as an "invalid state" error
// on successful login. If multiple calls to this method occur in a short window of time (which
// happens because we call this from multiple places if parallal AJAX requests all fail with 401),
// it seems to trigger multiple fetches of the "Auth URL", which both sets a cookie
// (via the set-cookie header) and redirects us to Plex (HTTP 307).
// There seems to be a race though where the browser follows the HTTP 307 redirect link returned by
// *one* call to this endpoint (with state = "foo") but has the cookie value set by a *different*
// call (with state = "bar").
// As a result, when the Plex login flow completes, the state doesn't match.
// AFAICT, each call to the Console Auth/redirect URL generates a unique state AND unique cookie
// value which references a different server-side session.
function redirectToLoginOn401(msg: string, code: number, redirectHref: string) {
  if (code === 401) {
    window.location.href = GetAuthURL(redirectHref);
  } else {
    // TODO: something better than logging to console
    // eslint-disable-next-line no-console
    console.log(msg);
  }
}

export default redirectToLoginOn401;
