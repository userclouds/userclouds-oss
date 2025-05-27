/* global chrome */

import { TokenizerClient } from '@userclouds/sdk-typescript';

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'resolveTokens') {
    chrome.storage.sync
      .get(['accessToken', 'tenantURL'])
      .then(({ accessToken, tenantURL }) => {
        if (accessToken && tenantURL) {
          const client = new TokenizerClient(tenantURL, accessToken);

          client
            .resolveTokens(message.tokens, message.context, [])
            .then((resolvedTokens) => {
              sendResponse({
                type: 'resolveTokensResponse',
                status: 'success',
                resolvedTokens,
              });
            })
            .catch((err) => {
              sendResponse({
                type: 'resolveTokensResponse',
                status: 'error',
                error: err.body,
              });
            });
        } else {
          sendResponse({
            type: 'resolveTokensResponse',
            status: 'error',
            error: 'No access token',
          });
        }
      });

    return true;
  }
  return false;
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'inspectToken') {
    chrome.storage.sync
      .get(['accessToken', 'tenantURL'])
      .then(({ accessToken, tenantURL }) => {
        if (accessToken && tenantURL) {
          const client = new TokenizerClient(tenantURL, accessToken);

          client
            .inspectToken(message.token)
            .then((response) => {
              sendResponse({
                type: 'inspectTokenResponse',
                status: 'success',
                required_context: response.access_policy.required_context,
              });
            })
            .catch((err) => {
              sendResponse({
                type: 'inspectTokenResponse',
                status: 'error',
                error: err.body,
              });
            });
        } else {
          sendResponse({
            type: 'inspectTokenResponse',
            status: 'error',
            error: 'No access token',
          });
        }
      });

    return true;
  }
  return false;
});
