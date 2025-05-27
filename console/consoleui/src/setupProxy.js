// setupProxy.js is responsible for configuring the React development proxy.
// We can't use the package.json directive `"proxy": "http://console.dev.userclouds.tools:5300"`
// because by default it does NOT proxy requests with an Accept header of 'text/html'.
// This means that our react app can't log in or log out by redirecting to HTML pages
// that are hosted on the backend server.
// Instructions were taken from:
// https://create-react-app.dev/docs/proxying-api-requests-in-development/#configuring-the-proxy-manually,

// Ignore lint warnings becuase this is a development only file and must be
// pure JS without modules.
/* eslint-disable @typescript-eslint/no-require-imports */
/* eslint-disable import/no-extraneous-dependencies */

const { createProxyMiddleware } = require('http-proxy-middleware');

module.exports = (app) => {
  app.use(
    '/api',
    createProxyMiddleware({
      target: 'http://console.dev.userclouds.tools:5300',
      changeOrigin: true,
    })
  );
  app.use(
    '/auth',
    createProxyMiddleware({
      target: 'http://console.dev.userclouds.tools:5300',
      changeOrigin: true,
    })
  );
};
