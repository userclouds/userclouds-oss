import {
  Before,
  BeforeAll,
  After,
  AfterAll,
  setDefaultTimeout,
} from '@cucumber/cucumber';
import { TestStepResultStatus } from '@cucumber/messages';
import { Request, Response } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

import globalSetup, { DEBUG_MODE } from './globalSetup';
import { CukeWorld } from './world';
import { axeConfig, getAxeRules, getOnlyRules } from './axe-config';

setDefaultTimeout(30000);

// TIP: DEBUG=true will work, but DEBUG=pw:api will enable Playwright's
// verbose logging mode, in addition to our debugging features:
// https://playwright.dev/docs/debug

BeforeAll({ timeout: 120000 }, async () => {
  if (DEBUG_MODE) {
    console.log(`You're using debug mode. This means:
      1. Network activity will be logged to the terminal.
         '>>' indicates an incoming request from the webpage.
         '<<' indicates an response from the dev server (via a mock)
      2. We will log when an incoming request matches a specified mock.
         See 'mockRequest' in features/step_definitions/helpers.ts
      3. Console logs from the browser will be logged to the terminal.
         You may comment that out below, where you see "this.page.on('console'..."
      4. If HEADLESS=false, the browser will stay open for 60 seconds following a test failure.
         Configure this below in the 'After' hook.
    `);
  }
  await globalSetup();
});

Before(async function (this: CukeWorld) {
  const { browser } = await globalSetup();
  this.browser = browser;
  this.browserContext = await this.browser.newContext({
    locale: 'en-US',
    timezoneId: 'America/Los_Angeles',
  });
  this.page = await this.browserContext.newPage();

  this.makeAxeBuilder = () => {
    const builder = new AxeBuilder({ page: this.page }).withTags(
      axeConfig.tags
    );

    const onlyRules = getOnlyRules();
    if (onlyRules && onlyRules.length > 0) {
      builder.withRules(onlyRules);
    } else {
      const rules = getAxeRules();
      if (rules.length > 0) {
        builder.disableRules(
          rules.filter((rule) => !rule.enabled).map((rule) => rule.id)
        );
      }
    }

    return builder;
  };

  this.activeMocks = [];
  if (DEBUG_MODE) {
    // Capture requests
    this.page.on('request', async (request: Request) => {
      console.log('>>', Date.now(), request.method(), request.url());
    });
    this.page.on('response', async (response: Response) => {
      console.log('<<', Date.now(), response.status(), response.url());
    });
    // Forward logs from pages
    this.page.on('console', (message: any) => {
      // COMMENT ME OUT IF PAGE LOGS ARE NOISY!
      console.log(message);
    });
  }
  this.browserContext.route('**/api/**', () => {
    // do not handle unmocked requests
  });
});

After({ timeout: DEBUG_MODE ? 650000 : 10000 }, async function (hookContext) {
  if (
    DEBUG_MODE &&
    process.env.HEADLESS === 'false' &&
    hookContext.result?.status === TestStepResultStatus.FAILED
  ) {
    console.debug('In debug mode and test failed.');

    // UNCOMMENT ME FOR A SCREENSHOT OF PAGE ON FAILURE:
    // console.debug("Taking a screenshot and saving it to functional-test-failure.png");
    // await this.page.screenshot({ path: 'functional-test-failure.png', fullPage: true });

    // UNCOMMENT ME TO LOG PAGE HTML CONTENTS OF BODY TAG
    // console.debug(await this.page.locator('body').first().innerHTML());

    console.debug(
      'Sleeping for 60s before closing browser. You may change this setting in features/support/hooks.ts'
    );
    await new Promise((resolve) => setTimeout(resolve, 60000));
  }
  if (DEBUG_MODE && this.activeMocks.length) {
    console.debug(`Feature: ${hookContext.gherkinDocument.feature?.name}`);
    console.debug(`Scenario: ${hookContext.pickle.name}`);
    console.debug('The following mocked requests were not fulfilled:');
    this.activeMocks.forEach((mock: any) => {
      console.debug(`[${mock.method}] ${mock.status} ${mock.url}`);
    });
  }
  this.activeMocks = [];
  await this.page.unroute('*');
  await this.page.close();
  await this.browserContext.clearCookies();
  await this.browserContext.close();
});

AfterAll({ timeout: 30000 }, async () => {
  const { browser, teardown } = await globalSetup();
  await browser.close();
  await teardown();
});
