# UI Tests

## Unit Tests

Our unit tests are written in and run using [Jasmine](http://jasmine.github.io/). We are not wedded to this, but it was a simple place to start. These tests live alongside the modules they're testing, in order to make relative imports easier. Ex: `userstore.spec.ts` lives in `src/reducers` alongside `src/reducers/userstore.ts`.

## Functional Tests

Our functional tests are specified with [CucumberJS](https://github.com/cucumber/cucumber-js) and use [Gherkin syntax](https://cucumber.io/docs/gherkin/reference/). The browser automation is done using [Playwright](https://playwright.dev/docs/intro). Functional tests live in the `features/` directory. Individual steps live in `features/step_definitions/steps.ts`. Some code used by multiple steps is specified in `features/support/world.ts`. In these tests, we mock responses from the console API, using JSON fixtures located in `features/fixtures`. This means that if API payloads change, mocks have to change accordingly.

## Commands

From the `userclouds` home directory:

- `make test` runs all tests, including Console UI's tests.
- `make consoleui-test` to run only console-ui tests.

From `console/consoleui`:

- `yarn run test` runs both functional and unit tests.
- `yarn run test:unit` runs only unit tests.
- `yarn run test:func` runs only functional tests (see below for more details on running functional tests).
- `yarn run test:func:debug` runs functional tests with `node inspect`, letting you use the built-in NodeJS command-line debugger.
- `yarn run test:func:debug-ide` runs functional tests with `node --inspect`, letting you use an external debugger like Chrome DevTools or VS Code.

## Environment Variables

You can use the following environment variables when running functional tests:

- `DEBUG=true` - Enables debug mode, logging requests, responses, and browser console output.
- `DEBUG=pw:api` - Enables Playwright's verbose logging in addition to our debug features.
- `BROWSER=firefox` - Specifies which browser to use (default is Chromium).
- `HEADLESS=false` - Runs tests with a visible browser window instead of headless mode.
- `SLOWMO=50` - Adds a delay (in milliseconds) between test steps for easier visual tracking.
- `DEVTOOLS=true` - Opens browser developer tools automatically (use with `HEADLESS=false`).

## Tips

- There's a VSCode extension for Playwright: [https://playwright.dev/docs/getting-started-vscode](https://playwright.dev/docs/getting-started-vscode).
- Playwright can be made to run Chrome, Chromium, or Firefox. The default for our tests is Chromium, but you can specify another browser with the `BROWSER` environment variable, e.g. `BROWSER=firefox yarn run test:func`.
- Debug mode: by setting `DEBUG=true` when running `test:func`, you enter a debug mode where useful information is logged to the console. This includes requests made by the browser and the mock responses used to fulfill them, as well as output from the browser's console. In debug mode, the browser will stay open for 60 seconds after a failed test, giving you time to inspect the state of the page.
- SlowMo mode: This slows down test execution by adding a delay between actions, making it easier to follow what's happening. Enable it with `SLOWMO=50` (or any millisecond value) when running tests.
- Browser visibility: By default, the tests run with a headless browser, but if you'd like to watch your tests in action, set `HEADLESS=false` when running `test:func`.
- Browser dev tools: If you pass `HEADLESS=false DEVTOOLS=true`, the browser will run with the developer tools window open by default.
- To run a specific scenario, you can append `--name` to your test command. By default, name will accept a string, and run all test scenarios with that string in the name. E.g.: `yarn run test:func --name "details"` will run both "editing tenant details" and "viewing accessor details".
- You can also use the `--tags` command line argument. See documentation [here](https://cucumber.io/docs/cucumber/api/?lang=javascript#running-a-subset-of-scenarios). Tags can be specified above a feature or scenario name, like this:

```gherkin
@userstore
@accessors
@accessor_details
Feature: Accessor details page
```

So to run all the tests tagged with `@accessors`, you can execute `yarn run test:func --tags "@accessors"`. You can also skip certain scenarios by creating a tag like `@skip` or `@omit` and running `yarn run test:func --tags "not @skip"`.

## Tips for Debugging Functional Tests

1. Isolate test failures:

   - Run only the failing test. If it succeeds, add scenarios one by one to the test run until you get a failure. If it fails, remove steps from the scenario until you've created a minimal failing repro.
   - Try multiple browsers. If it fails in only one of them, ask what's different.
   - Does it fail in SLOWMO?
   - Does it fail in CI but not locally? Consider the differences between these environments. Pay attention to the yarn dependencies.

2. Pay special attention to timing:

   - By default, Playwright waits for every step to be true, until it hits a timeout. Waits are just polls under the hood. Playwright will execute the same query over and over on a recursive setTimeout until it evaluates to true.
   - Check for transitional states. Click a button, check for the loading state, THEN check for the final state.
   - Changes can happen really quickly. If you want to make sure a change (especially a fleeting one) happens after you click a button, consider constructing a step that sets up the poll first, then clicks the button, then awaits the fulfillment of the poll. E.g.:

   ```typescript
   // Set up poll
   const loading = this.page.waitForSelector('main[aria-busy=true]`);
   // Commit action
   await this.page.click('button');
   // evaluate poll
   await loading;
   ```

3. Set a breakpoint. You can put `debugger` statements in the TypeScript (step definition) code (though not the Cucumber code), and run with `test:func:debug` or `test:func:debug-ide` to step through code in a debugger.

4. Add logs to your app code. When `DEBUG=true` is set, page logs will show up in your terminal. Log extensively if you need to.

5. Add logs to your test steps. Make sure you know where along the way things go wrong.

6. **Make sure all your network requests are mocked**. If you run with `DEBUG=true HEADLESS=false DEVTOOLS=true`, network requests should be captured in the browser dev tools, and your browser will stay open for 60 seconds following failed tests. This gives you time to look for unmocked requests. You may find real bugs where your page is making unexpected or redundant requests. You may also have failed to match the request URL (e.g. the querystring is different). In `DEBUG` mode, you can also see incoming requests (preceded in the logs by `>>`) and outgoing responses (preceded in the logs by `<<`). Make sure each incoming request has a corresponding response.

7. You can pause test execution with `await page.pause();`.

## How Functional Tests are Organized

All functional tests are in the `features/` directory. This directory contains test files (with the file extension `.feature`) and the following subdirectories:

- `step_definitions`: This is a cucumber convention (but it's configurable in `cucumber.js`). The convention is to have a `steps.js` (`.ts` in our case) file, but the cucumber test runner will look at ANY files in this directory for step definitions. We also have a `helpers.ts` file in the `step_definitions` directory which contains a set of routines used by multiple steps.
- `fixtures`: Also a cucumber convention, but we could name this anything. This is where we keep our JSON mocks.
- `support`: By convention, this is where test setup code lives. `support/hooks.ts` contains hooks that run before/after each test scenario and before/after the test run as a whole. These do things like start the webpack dev server and launch the browser. `support/globalSetup.ts` has some code used by the hooks file.

Within the `features/` directory, it's possible (and probably advisable) to organize feature files (i.e., tests and test suites) within subdirectories. With the `features/step_definitions/` directory, it is likewise possible to organize step definitions using subdirectories or separate files.
