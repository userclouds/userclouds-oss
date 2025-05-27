// setupProxy.js is responsible for configuring the React development proxy.
// This file allows the react development server to "look like" the real site as much as possible
// by passing any paths not handled by the react app on to the origin server.
// Instructions were taken from:
// https://create-react-app.dev/docs/proxying-api-requests-in-development/#configuring-the-proxy-manually,

// Ignore lint warnings because this is a development only file and must be
// pure JS without modules.
import { createProxyMiddleware } from 'http-proxy-middleware';

export default (app) => {
  app.use('/', (req, res, next) => {
    // If the URL does NOT start with "/plexui", proxy it to the Plex server,
    // otherwise try to serve it from the React/Express web server.
    // NOTE: because of multitenant plex, we need to use the right target URL based on the request's Host header.
    // Additionally, we need to forward the request to devlb which runs on port 3333 and proxies the request
    // to the actual Plex port (hence the 'split' operation to strip off the React/Express dev web server port,
    // which is 3011 for plex UI dev, and replacement with port 3333).
    // TODO: this is *yet another* place we reference the homepage/basename (/plexui).
    // I think there is a way to load this from `package.json` but I gave up trying after 15 min.
    // TODO: how to keep port in sync with Plex config?
    if (!/^\/plexui/.test(req.url)) {
      return createProxyMiddleware({
        target: `https://${req.headers.host.split(':')[0]}:3333`,
        changeOrigin: true,
      })(req, res, next);
    }
    return next();
  });
};
