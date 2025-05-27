# Console & Plex UI

The UserClouds Developer Console & Plex front-end are both React-based UI applications which depend on a shared UI component library.

## Directory Structure

The packages are located as follows:

- Console - [/console/consoleui/](/console/consoleui/README.md)
- Plex - [/plex/plexui/](/plex/plexui/README.md)
- Shared UI - [/sharedui/](/sharedui/)
- Component Library - [/ui-component-lib/](/ui-component-lib/README.md)

Most of the explanations that follow apply to all 3 projects, except that SharedUI is a library which uses slightly different bundling/build settings and cannot be "run" like the applications. There is also a root-level package which acts as a base for the other packages. Most tools (yarn, eslint, etc.) walk up the directory tree to find config/deps, and Yarn hoists dependencies to the topmost level where they are shared. This lets us share configs/deps.

Also: ConsoleUI was bootstrapped with [Create React App](https://github.com/facebook/create-react-app). Specifically, I ran `npx create-react-app consoleui --template typescript`. Effectively this just generates the `package.json` file at the root which most other things are derived from. PlexUI & SharedUI were copied from that and modified.

## How to React in our repo

### Non-UI development

Just run `make dev` and use the Console or Plex as normal. This will serve the UI from the static asset bundle (webpack) that our 'prod' service would use. You can completely ignore the existence of this React app and all of the files underneath this directory, as they are simply the inputs to the build process which generate the static asset bundles under `console/consoleui/build` and `plex/plexui/build`.

### UI development

When iterating on React UI, it's best to point your browser at the React development server _instead of_ the Console/Plex service. Both technically serve the same UI routes & pages, but the Console/Plex service only serves the pre-built content. The development server provides hot-reloading of your code with better error reporting in the console.

1. Run `make dev` in one shell to ensure the DB and all backend services are running. NOTE: for Plex UI development, you'll run `UC_PLEX_UI_DEV_PORT=3011 make dev` - see note below.
2. In a separate shell, run `make consoleui-dev` and/or `make plexui-dev`. This runs specific `yarn` commands under the hood.
3. Open [https://console.dev.userclouds.tools:3010](https://console.dev.userclouds.tools:3010) for Console or [https://dev.userclouds.tools:3011](https://dev.userclouds.tools:3011) for Plex to view it in the browser.

The page will reload if you make edits. You will also see any compilation/lint errors in the console.

These use the statically built versions of `sharedui` so any changes to shared code will NOT be reflected above. If you also want to do development on shared components, run `make sharedui-dev` in yet-another-tab.

There is a proxy hosted in the React development server that maps routes NOT handled by the React app to the underlying server. Look at `setupProxy.js` - there are 2 of them, 1 for Plex and 1 for Console.

NOTE: Why do we have the `UC_PLEX_UI_DEV_PORT` env var? Unlike Plex, Console is an SPA that initiates requests to the backend, so the backend works the same regardless of where the UI is hosted (CORS issues aside...). But Plex's UI rendering is triggered/redirected to by the server, so the server needs to know where to redirect the user agent to.

### Dependencies

If you add/change dependencies or versions (either by directly editing `package.json` files or running `yarn add ...`), you should run `make ui-yarn-install` (which just runs `yarn install` under the hood) to update the dependencies in your `node_modules` subdirectory. This will also update `yarn.lock`. Running `make devsetup` will trigger `yarn install` as a side-effect via `make ui-yarn-install`, so those are 2 ways to ensure deps are up to date.

You can add new dependencies with `yarn add ...`. See https://yarnpkg.com/getting-started/usage. This will update `package.json` as well. NOTE: since we use yarn workspaces, extra care must be taken to install the dependency to the right `package.json` file; see docs for more detail.

### Building

Running `make console/consoleui/build` will build the Console UI asset bundle (and `make plex/plexui/build` does the same for Plex). This uses `yarn` under the hood, which builds the app for production to the `build` folder. This is configured to depend on `make sharedui/build` and should re-run that command if necessary, but the makefile rule is pretty dumb and only relies on the directory's timestamp. So if something isn't working (e.g., due to settings changes) you may want to nuke the build directories and try again.

Running `make dev` automatically builds the static asset bundle, so you shouldn't need to do this often.

Running our dev commands (e.g., `make sharedui-dev`) will _also_ update the same built files any time source changes are detected. Basically the "dev" versions just run a watch-build loop and share the same inputs/outputs.

The built bundle is part of `.gitignore` and is NOT checked in; our CodeBuild pipelines explicitly build the bundle from source each time.

It correctly bundles React in production mode and optimizes the build for the best performance. The build is minified and the filenames include the hashes. See the section about [deployment](https://facebook.github.io/create-react-app/docs/deployment) for more information.

### Linting

Running `make lint` and `make lintfix` automatically run ESLint & Prettier in the appropriate mode on this React app.

The ESLint configuration is in the file `.eslintrc.js`; you can edit it to change rules, plugins, and other features. Some customization was needed in order to make ESLint work well with React + TypeScript (see comments in that file for details). We use Prettier to autoformat our js/ts/tsx/etc files too, which is configured in `.prettierrc.js`. The config is shared globally across all projects in our mono repo.

### Testing

Running `make test` (from the `userclouds` home directory) will run Console UI's tests for you. You can also run `make consoleui-test` to run only these tests.

From consoleui, you can run `yarn run test` to run both functional and unit tests, `yarn run test:unit` to run only unit tests, and `yarn run test:func` to run only functional tests.

For more details see the testing [readme](./features/TEST-README.md).
