/* global chrome */

const findTokens = async (selector, tag, separateLink) => {
  const [tab] = await chrome.tabs.query({
    active: true,
    lastFocusedWindow: true,
  });
  if (tab) {
    chrome.tabs.sendMessage(tab.id, {
      type: 'findTokens',
      selector,
      tag,
      separateLink,
    });
  }
};

const displayLogin = (show) => {
  document.getElementById('loginSection').style.display = show
    ? 'inline'
    : 'none';
  document.getElementById('logout').style.display = show ? 'none' : 'block';
  document.getElementById('findTokenSection').style.display = show
    ? 'none'
    : 'inline';
};

document.addEventListener('DOMContentLoaded', async () => {
  const {
    tenantURL,
    clientID,
    selector,
    tag,
    separateLink,
    accessToken,
    idToken,
  } = await chrome.storage.sync.get([
    'tenantURL',
    'clientID',
    'selector',
    'tag',
    'separateLink',
    'accessToken',
    'idToken',
  ]);

  document.querySelector('#tenantURL').value = tenantURL || '';
  document.querySelector('#clientID').value = clientID || '';
  document.querySelector('#selector').value = selector || '';
  document.querySelector('#separateLink').checked = separateLink;
  document.querySelector('#tag').value = tag || '';
  if (idToken) {
    try {
      const obj = JSON.parse(atob(idToken.split('.')[1].replaceAll('_', '/'))); // TODO: This replacement is probably only due to using the implicit flow
      document.querySelector('#loginInfo').innerHTML =
        `You are logged in as ${obj.name} (${obj.email})`;
    } catch (e) {
      // eslint-disable-next-line no-console
      console.error(e);
      document.querySelector('#loginInfo').innerHTML = 'You are logged in';
    }
  }

  if (accessToken) {
    displayLogin(false);
  }

  document.getElementById('login').addEventListener('click', async () => {
    const tenantURLValue = document.querySelector('#tenantURL').value;
    const clientIDValue = document.querySelector('#clientID').value;

    await chrome.storage.sync.set({
      tenantURL: tenantURLValue,
      clientID: clientIDValue,
      accessToken: '',
      idToken: '',
    });

    const redirectURL = chrome.identity.getRedirectURL('callback');
    const authParams = {
      client_id: clientIDValue,
      redirect_uri: redirectURL,
      response_type: 'token id_token', // TODO: add support for PKCE flow
      scope: 'openid profile',
      state: '123456789',
    };

    const query = new URLSearchParams(Object.entries(authParams));
    chrome.identity.launchWebAuthFlow(
      {
        url: `${tenantURLValue}/oidc/authorize?${query.toString()}`,
        interactive: true,
      },
      async (response) => {
        const url = new URL(response);
        const newAccessToken = url.searchParams.get('token');
        const newIDToken = url.searchParams.get('id_token');
        await chrome.storage.sync.set({
          accessToken: newAccessToken,
          idToken: newIDToken,
        });
        document.getElementById('logout').style.display = 'block';
      }
    );
  });

  const toggleTokenConfig = (showConfig = true) => {
    document.getElementById('showTokenConfig').style.display = showConfig
      ? 'none'
      : 'inline';
    document.getElementById('findTokens').style.display = showConfig
      ? 'none'
      : 'inline';
    document.getElementById('tokenConfig').style.display = showConfig
      ? 'inline'
      : 'none';
    document.getElementById('saveTokenConfig').style.display = showConfig
      ? 'inline'
      : 'none';
    document.getElementById('cancelEditTokenConfig').style.display = showConfig
      ? 'inline'
      : 'none';
  };

  document.getElementById('showTokenConfig').addEventListener('click', () => {
    toggleTokenConfig(true);
  });
  document
    .getElementById('cancelEditTokenConfig')
    .addEventListener('click', () => {
      toggleTokenConfig(false);
    });

  document
    .getElementById('saveTokenConfig')
    .addEventListener('click', async () => {
      const selectorValue = document.querySelector('#selector').value;
      const tagValue = document.querySelector('#tag').value;
      const separateLinkValue = document.querySelector('#separateLink').checked;

      await chrome.storage.sync.set({
        selector: selectorValue,
        tag: tagValue,
        separateLink: separateLinkValue,
      });
      toggleTokenConfig(false);
      findTokens(selectorValue, tagValue, separateLinkValue);
    });

  document.getElementById('logout').addEventListener('click', async () => {
    await chrome.storage.sync.set({
      accessToken: '',
      idToken: '',
    });
    displayLogin(true);
  });
});
