/* eslint-disable no-alert */
/* global chrome */

async function resolveDOMToken(elem, token, requiredContext, updateBorder) {
  const element = elem;
  const context = {};
  Object.keys(requiredContext).forEach((key) => {
    context[key] = window.prompt(`Please provide ${key}`);
  });

  const resolveResponse = await chrome.runtime.sendMessage({
    type: 'resolveTokens',
    tokens: [token],
    context,
  });
  if (resolveResponse.status === 'success') {
    if (resolveResponse.resolvedTokens[0].data) {
      element.innerHTML = resolveResponse.resolvedTokens[0].data;
      element.setAttribute('data-uc-resolved', token);
      if (updateBorder) {
        element.style.border = '2px solid green';
      }
    } else {
      window.alert(
        `Error resolving token "${token}: Failed access policy check`
      );
    }
  } else {
    window.alert(`Error resolving token "${token}: ${resolveResponse.error}`);
    if (updateBorder) {
      element.style.border = '2px solid red';
    }
  }
}

async function annotateTokensInDOM(selector, tag, separateLink) {
  if (!selector) {
    return;
  }

  document.querySelectorAll(selector).forEach(async (elem) => {
    const element = elem;
    let token;
    if (tag) {
      token = element.getAttribute(tag);
    } else {
      token = element.getAttribute('data-uc-token');
      if (!token) {
        token = element.innerText;
      }
    }
    if (token) {
      const resolved = element.getAttribute('data-uc-resolved');
      if (resolved === token) {
        return;
      }

      const inspected = element.getAttribute('data-uc-inspected');
      if (inspected === token) {
        return;
      }

      const inspectResponse = await chrome.runtime.sendMessage({
        type: 'inspectToken',
        token,
      });
      element.setAttribute('data-uc-inspected', token);

      if (inspectResponse.status === 'success') {
        if (separateLink) {
          const anchor = document.createElement('a');
          anchor.href = '#';
          anchor.addEventListener('click', (e) => {
            resolveDOMToken(
              element,
              token,
              inspectResponse.required_context || {},
              false
            );
            e.preventDefault();
          });
          anchor.innerText = 'resolve';
          element.replaceChildren(`${token} `);
          element.append(anchor);
        } else {
          const anchor = document.createElement('a');
          anchor.href = '#';
          anchor.addEventListener('click', (e) => {
            resolveDOMToken(
              element,
              token,
              inspectResponse.required_context || {},
              true
            );
            e.preventDefault();
          });
          anchor.innerText = token;
          anchor.title = 'click to resolve token';
          element.setAttribute('data-uc-token', token);
          element.replaceChildren(anchor);
          element.style.border = '1px solid blue';
        }
      }
    }
  });
}

chrome.runtime.onMessage.addListener((message) => {
  if (message.type === 'findTokens') {
    annotateTokensInDOM(message.selector, message.tag, message.separateLink);
  }
});

setInterval(async () => {
  if (!chrome.runtime?.id) {
    return;
  }

  const { accessToken, selector, tag, separateLink } =
    await chrome.storage.sync.get([
      'accessToken',
      'selector',
      'tag',
      'separateLink',
    ]);

  if (!accessToken) {
    return;
  }

  annotateTokensInDOM(selector, tag, separateLink);
}, 500);
