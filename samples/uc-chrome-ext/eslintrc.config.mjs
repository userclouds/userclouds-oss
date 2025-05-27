export default {
  env: {
    browser: true,
    es2021: true,
    node: true,
    webextensions: true,
  },
  extends: ['eslint:recommended', 'plugin:webextensions/recommended'],
  plugins: ['webextensions'],
  globals: {
    document: false,
    escape: false,
    navigator: false,
    unescape: false,
    window: false,
    describe: true,
    before: true,
    it: true,
    expect: true,
    sinon: true,
  },
  languageOptions: {
    globals: {
      chrome: 'readonly',
      webextensions: 'readonly',
    },
  },
  parserOptions: {
    ecmaVersion: 2020,
    sourceType: 'module',
  },
};
