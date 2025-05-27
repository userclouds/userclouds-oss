# uc-chrome-ext

Plug-in that looks up tokens on UserClouds with viewer's credentials.

To set it up, you need to add `https://<your-installed-extension-id>.chromiumapp.org/callback` to the list of Allowed Redirect URLs for
a tenant's login application, and then configure the extension ("Configure login") to use the Tenant URL and and Client ID for the application.

Click "Log In" to authenticate as an end user of that tenant. You may need to create a new end user.

Once you are logged in, the extension popup changes to "Lookup Tokens". You should configure the token selector to match however the website
you are looking at presents the tokenized information. For example, in ZenDesk customer view, the profile info is displayed within "div" and "span"
elements that contain a "data-identity-id" attribute, so the selector for tokens can be `div[data-identity-id], span[data-identity-id]`, and the token
is actually stored as the "title" attribute, so `title` should be used for the Tag property for tokens field.

## Development

This extension was created with [Extension CLI](https://oss.mobilefirst.me/extension-cli/)!

If you find this software helpful [star](https://github.com/MobileFirstLLC/extension-cli/) or [sponsor](https://github.com/sponsors/MobileFirstLLC) this project.

### Available Commands

| Commands         | Description                         |
| ---------------- | ----------------------------------- |
| `yarn run start` | build extension, watch file changes |
| `yarn run build` | generate release version            |
| `yarn run docs`  | generate source code docs           |
| `yarn run clean` | remove temporary files              |
| `yarn run test`  | run unit tests                      |
| `yarn run sync`  | update config files                 |

For CLI instructions see [User Guide &rarr;](https://oss.mobilefirst.me/extension-cli/)

### Learn More

**Extension Developer guides**

- [Getting started with extension development](https://developer.chrome.com/extensions/getstarted)
- Manifest configuration: [version 2](https://developer.chrome.com/extensions/manifest) - [version 3](https://developer.chrome.com/docs/extensions/mv3/intro/)
- [Permissions reference](https://developer.chrome.com/extensions/declare_permissions)
- [Chrome API reference](https://developer.chrome.com/docs/extensions/reference/)

**Extension Publishing Guides**

- [Publishing for Chrome](https://developer.chrome.com/webstore/publish)
- [Publishing for Edge](https://docs.microsoft.com/en-us/microsoft-edge/extensions-chromium/publish/publish-extension)
- [Publishing for Opera addons](https://dev.opera.com/extensions/publishing-guidelines/)
- [Publishing for Firefox](https://extensionworkshop.com/documentation/publish/submitting-an-add-on/)
