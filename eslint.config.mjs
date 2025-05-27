import react from 'eslint-plugin-react';
import typescriptEslint from '@typescript-eslint/eslint-plugin';
import reactHooks from 'eslint-plugin-react-hooks';
import prettier from 'eslint-plugin-prettier';
import jsxa11y from 'eslint-plugin-jsx-a11y';
import { fixupPluginRules } from '@eslint/compat';
import globals from 'globals';
import tsParser from '@typescript-eslint/parser';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import js from '@eslint/js';
import { FlatCompat } from '@eslint/eslintrc';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const compat = new FlatCompat({
  baseDirectory: __dirname,
  recommendedConfig: js.configs.recommended,
  allConfig: js.configs.all,
});

export default [
  {
    ignores: [
      'ui-component-lib/pages',
      'console/consoleui/cucumber-report.json',
      'sharedui/build',
      'samples/uc-chrome-ext/dist',
      'samples/nodejs/.yarn',
      'samples/auth0nodejs/.yarn',
      '*/build/*',
      '*/node_modules/*',
      '*/storybook-static/*',
      '*/.rollup.cache/*',
      '*/dist/*',
    ],
  },
  ...compat.extends(
    'plugin:react/recommended',
    'airbnb',
    'plugin:@typescript-eslint/recommended',
    'prettier',
    'plugin:react/jsx-runtime'
  ),
  {
    plugins: {
      react,
      '@typescript-eslint': typescriptEslint,
      'react-hooks': fixupPluginRules(reactHooks),
      prettier,
      jsxa11y,
    },

    languageOptions: {
      globals: {
        ...globals.browser,
      },

      parser: tsParser,
      ecmaVersion: 12,
      sourceType: 'module',

      parserOptions: {
        ecmaFeatures: {
          jsx: true,
        },
      },
    },

    settings: {
      'import/resolver': {
        typescript: {
          project: ['tsconfig.json'],
        },
      },

      'jsx-a11y': {
        components: {
          Button: 'button',
          IconButton: 'button',
          TextInput: 'input',
          HiddenTextInput: 'input',
          Checkbox: 'input',
          Radio: 'input',
          Label: 'label',
          Select: 'select',
          TextArea: 'textarea',
        },
      },
    },

    rules: {
      ...jsxa11y.configs.recommended.rules,
      '@typescript-eslint/no-unused-vars': 'error',
      'react/jsx-filename-extension': [
        'warn',
        {
          extensions: ['.tsx'],
        },
      ],
      'react/jsx-curly-brace-presence': [
        'error',
        {
          props: 'never',
          children: 'never',
          propElementValues: 'always',
        },
      ],
      'react/no-unused-prop-types': 'warn',
      'import/extensions': [
        'error',
        'ignorePackages',
        {
          ts: 'never',
          tsx: 'never',
        },
      ],
      '@typescript-eslint/no-shadow': ['error'],
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'warn',
      'react/function-component-definition': 'off', // Enabled in consoleui

      // we can evaulate whether these are useful for us
      'no-use-before-define': 'off',
      'prettier/prettier': 'error',
      'no-shadow': 'off',
      'no-bitwise': 'off',
      'no-alert': 'off',
      'no-param-reassign': 'off',
      'prefer-template': 'off',
      'consistent-return': 'off',
      'prefer-object-spead': 'off',
      'no-nested-ternary': 'off',
      'arrow-body-style': 'off',
      'object-shorthand': 'off',
      'no-promise-executor-return': 'off',
      'import/prefer-default-export': 'off',
      'no-plusplus': 'off',
      '@typescript-eslint/no-unsafe-function-type': 'off',
      'no-restricted-globals': 'off',
      'prefer-destructuring': 'off',
      'react/button-has-type': 'off',
      'react/jsx-props-no-spreading': 'off',
      '@typescript-eslint/no-unused-expressions': 'off',
      'no-await-in-loop': 'off',
      'default-param-last': 'off',
      'max-classes-per-file': 'off',
      'no-underscore-dangle': 'off',
      'func-names': 'off',
      'import/no-named-as-default': 'off',
      // need to look into further
      'no-restricted-syntax': 'off',
      '@typescript-eslint/triple-slash-reference': 'off',
      // we probably want to re-enable these:
      'no-async-promise-executor': 'off',
      'no-prototype-builtins': 'off',
      'prefer-promise-reject-errors': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      'class-methods-use-this': 'off',
      'react/require-default-props': 'off',
      '@typescript-eslint/no-non-null-asserted-optional-chain': 'off',
      'react/no-unescaped-entities': 'off',
      'guard-for-in': 'off',
      'no-continue': 'off',
      'import/no-cycle': 'off',
      // we definitely want to re-enable these:
      'jsx-a11y/label-has-associated-control': [
        'error',
        {
          labelComponents: ['Label'],
          labelAttributes: ['label'],
          controlComponents: [
            'TextInput',
            'HiddenTextInput',
            'Checkbox',
            'Radio',
            'Select',
            'TextArea',
          ],
          assert: 'either',
          depth: 3,
        },
      ],
      'jsx-a11y/control-has-associated-label': [
        'error',
        {
          labelAttributes: ['label'],
          controlComponents: [
            'Button',
            'Checkbox',
            'Radio',
            'Select',
            'TextArea',
            'TextInput',
            'HiddenTextInput',
          ],
          ignoreElements: [
            'audio',
            'canvas',
            'embed',
            'input',
            'textarea',
            'tr',
            'video',
          ],
          ignoreRoles: [
            'grid',
            'listbox',
            'menu',
            'menubar',
            'radiogroup',
            'row',
            'tablist',
            'toolbar',
            'tree',
            'treegrid',
          ],
          depth: 3,
        },
      ],
      'jsx-a11y/no-static-element-interactions': [
        'error',
        {
          handlers: [
            'onClick',
            'onMouseDown',
            'onMouseUp',
            'onKeyPress',
            'onKeyDown',
            'onKeyUp',
          ],
        },
      ],
      'jsx-a11y/no-noninteractive-element-interactions': [
        'error',
        {
          handlers: [
            'onClick',
            'onMouseDown',
            'onMouseUp',
            'onKeyPress',
            'onKeyDown',
            'onKeyUp',
          ],
        },
      ],
      'jsx-a11y/click-events-have-key-events': 'error',
      'jsx-a11y/anchor-is-valid': [
        'error',
        {
          components: ['Link'],
          specialLink: ['to'],
          aspects: ['noHref', 'invalidHref', 'preferButton'],
        },
      ],
      'react/jsx-no-useless-fragment': 'off',
      'eslint-disable': 'off',
    },
  },
  {
    files: ['console/consoleui/**/features/**/*'],
    rules: {
      'no-console': 'off',
    },
  },
  {
    files: ['console/consoleui/**/*'],
    rules: {
      'react/function-component-definition': [
        'error',
        {
          namedComponents: 'arrow-function',
          unnamedComponents: 'arrow-function',
        },
      ],
      camelcase: 'off',
    },
  },
  {
    files: ['ui-component-lib/**/*'],

    rules: {
      'react/button-has-type': 'off',
      'react/jsx-props-no-spreading': 'off',
      'react/require-default-props': 'off',
      'import/no-extraneous-dependencies': 'off',
      'react/prop-types': 'off',
      '@typescript-eslint/no-shadow': 'off',
      'block-scoped-var': 'off',
      'react/no-unused-prop-types': 'warn',
    },
  },
  {
    files: ['ui-component-lib/src/**/*.stories.tsx'],

    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
    },
  },
  {
    files: ['public-repos/sdk-typescript/**/*'],

    rules: {
      'no-console': 'off',
      camelcase: 'off',
    },
  },
];
