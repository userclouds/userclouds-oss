import { Given, When, Then } from '@cucumber/cucumber';
import { expect, Locator, Page } from '@playwright/test';

import {
  goToPage,
  pressButton,
  findCardByTitle,
  escapeSpecialCharacters,
} from './helpers';
import { CukeWorld } from '../support/world';

Given(
  /^I intend to (accept|dismiss) the confirm dialog$/,
  async function (this: CukeWorld, acceptOrDeny: string) {
    this.page.on('dialog', (dialog) =>
      acceptOrDeny === 'accept' ? dialog.accept() : dialog.dismiss()
    );
  }
);

Given(
  'I intend to type {string} in the prompt dialog',
  async function (this: CukeWorld, promptText: string) {
    this.page.on('dialog', (dialog) => dialog.accept(promptText));
  }
);

When('I reload the page', async function (this: CukeWorld) {
  await this.page.reload();
  await this.page.waitForLoadState('networkidle');
});

When(
  'I navigate to the page with path {string}',
  async function (this: CukeWorld, path: string) {
    await goToPage(this, path);
  }
);

When(
  'I click the button labeled {string}',
  async function (this: CukeWorld, buttonText: string) {
    await pressButton(this, buttonText);
  }
);

When(
  'I click the button labeled {string} in the {string} card',
  async function (this: CukeWorld, buttonLabel: string, cardTitle: string) {
    const card = await findCardByTitle(this, cardTitle);
    const selector = `button:is(:has-text("${buttonLabel}"), [aria-label="${buttonLabel}"])`;
    const button = await card?.locator(selector);
    await button?.click();
  }
);

When(
  /^I click the ([0-9]+)(?:th|nd|st|rd) button labeled "([^"]+)" in the "([^"]+)" card$/,
  async function (
    this: CukeWorld,
    ordinal: string,
    buttonLabel: string,
    cardTitle: string
  ) {
    const n = parseInt(ordinal, 10) - 1;
    const card = await findCardByTitle(this, cardTitle);
    const selector = `button:is(:has-text("${buttonLabel}"), [aria-label="${buttonLabel}"])`;
    const button = await card?.locator(selector).nth(n);
    await button?.click();
  }
);

When(
  'I click the button with ID {string}',
  async function (this: CukeWorld, buttonID: string) {
    const selector = `button#${buttonID}`;
    const button = await this.page.locator(selector);
    await button?.click();
  }
);

When(
  /^I click the ([0-9]+)(?:th|nd|st|rd) button with ID "([^"]+)"$/,
  async function (this: CukeWorld, ordinal: string, buttonID: string) {
    const n = parseInt(ordinal, 10) - 1;
    const selector = `button#${buttonID}`;
    const button = await this.page.locator(selector).nth(n);
    await button?.click();
  }
);

When(
  'I click the button labeled {string} in the footer of the {string} card',
  async function (this: CukeWorld, buttonText: string, cardTitle: string) {
    const card = await findCardByTitle(this, cardTitle);
    const footer = await card?.locator('footer');
    const button = await footer?.locator('button', { hasText: buttonText });
    await button?.click();
  }
);

When(
  'I click the element matching selector {string}',
  async function (this: CukeWorld, selector: string) {
    const el = await this.page.locator(selector);
    await el?.click();
  }
);

When(
  'I click the link with the href {string}',
  async function (this: CukeWorld, href: string) {
    const link = await this.page.waitForSelector(`a[href="${href}"]`);
    link.click();
  }
);

When(
  'I click the link with the text {string}',
  async function (this: CukeWorld, linkText: string) {
    const link = await this.page.waitForSelector(
      `a:visible:text-is("${linkText}")`
    );
    link.click();
  }
);

When('I wait for the network to be idle', async function (this: CukeWorld) {
  const ongoingRequests: Set<string> = new Set();

  // Listen for requests being made
  this.page.on('request', (request) => {
    ongoingRequests.add(request.url());
  });

  // Listen for requests completing
  this.page.on('requestfinished', (request) => {
    ongoingRequests.delete(request.url());
  });

  this.page.on('requestfailed', (request) => {
    ongoingRequests.delete(request.url());
  });

  // Wait for the network to be idle
  await this.page.waitForLoadState('networkidle');

  /* uncomment for logging
  // If there are still ongoing requests, log them
  if (ongoingRequests.size > 0) {
    console.log(
      'The following requests were still in progress:',
      Array.from(ongoingRequests)
    );
  } else {
    console.log('No ongoing requests, network is idle.');
  } */
});

When(
  'I click {string} in the sidebar',
  async function (this: CukeWorld, linkText: string) {
    await this.page
      .locator(`#mainNav ol > li > a:has-text("${linkText}")`, {
        hasText: linkText,
      })
      .click();
  }
);

When(
  'I click {string} header in the sidebar',
  async function (this: CukeWorld, headerText: string) {
    await this.page
      .locator(`#mainNav ol > li > button:has-text("${headerText}")`, {
        hasText: headerText,
      })
      .click();
  }
);

When(
  'I click to expand the {string} accordion',
  async function (this: CukeWorld, label: string) {
    await this.page.click(`details:not(.open) > summary:has-text("${label}")`);
    await this.page.locator(`details.open > summary:has-text("${label}")`);
  }
);

When(
  'I click the button to toggle the expansion of the {string} card',
  async function (this: CukeWorld, cardTitle: string) {
    const card = await findCardByTitle(this, cardTitle);
    const expandButton = await card?.locator('button[title="Set Closed"]');
    await expandButton?.click();
  }
);

When(
  'I select the option labeled {string} in the custom dropdown matching selector {string}',
  async function (this: CukeWorld, optionLabel, selector) {
    const select = await this.page.locator(selector);
    await select.focus();
    await this.page.keyboard.press(' ');
    await select.locator(`[role="option"]:has-text("${optionLabel}")`).click();
  }
);

When(
  /^I click the (button|buttons) to dismiss the notification$/,
  async function (this: CukeWorld, button: string) {
    if (button === 'button') {
      const expandButton = await this.page.locator(
        'button[title="dismiss this notification"]'
      );
      await expandButton.first().click();
    } else if (button === 'buttons') {
      const allButtons = await this.page.locator(
        'button[title="dismiss this notification"]'
      );
      const count = await allButtons.count(); // Get the number of buttons

      for (let i = 0; i < count; i++) {
        await allButtons.nth(i).click(); // Click each button one by one
      }
    }
  }
);

Then(
  'the page title should be {string}',
  async function (this: CukeWorld, title: string) {
    await expect(this.page).toHaveTitle(title);
  }
);

Then(
  'I should be on the page with the path {string}',
  async function (this: CukeWorld, path: string) {
    const url = await this.page.url();
    const { pathname } = new URL(url);
    expect(pathname).toBe(path);
  }
);

Then(
  'I should be on the page with the relative URL {string}',
  async function (this: CukeWorld, url: string) {
    const href = await this.page.url();
    const { pathname, search, hash } = new URL(href);
    expect(`${pathname}${search}${hash}`).toBe(url);
  }
);

Then(
  'I should be navigated to the page with the path {string}',
  async function (this: CukeWorld, path: string) {
    await this.page.waitForFunction(
      (expectedPath) => window.location.pathname === expectedPath,
      path
    );
    const url = await this.page.url();
    const { pathname } = new URL(url);
    expect(pathname).toBe(path);
  }
);

Then(
  /^I should( not)? see a "([^"]+)" with the text "([^"]+)"(?: within the "([^"]+)" card)?$/,
  async function (
    this: CukeWorld,
    not: string,
    selector: string,
    text: string,
    cardTitle: string
  ) {
    let context: Page | Locator;
    if (cardTitle) {
      context = await findCardByTitle(this, cardTitle);
      await expect(context).toBeVisible();
    } else {
      context = this.page;
    }

    const locator = context.locator(`${selector}:has-text("${text}"):visible`);

    if (not) {
      // Check that the element does not exist or is not visible
      await expect
        .poll(
          async () => {
            const count = await locator.count();
            return count;
          },
          {
            intervals: [5, 10, 10, 10, 10, 10, 10, 10, 10, 10, 20],
            timeout: 5000,
          }
        )
        .toBe(0);
    } else {
      const el = await locator.first();
      await expect
        .poll(
          async () => {
            const content = await el.textContent();
            return content;
          },
          {
            intervals: [5, 10, 10, 10, 10, 10, 10, 10, 10, 10, 20],
            timeout: 5000,
          }
        )
        .toBe(text);
    }
  }
);

Then(
  /^I should( not)? see an element matching selector "([^"]+)"$/,
  async function (this: CukeWorld, not: string, selector: string) {
    const el = this.page.locator(selector);
    if (not) {
      await expect(el).not.toBeVisible();
    } else {
      await expect(el).toBeVisible();
    }
  }
);

Then(
  /^I should see ([0-9]+) elements matching selector "([^"]+)"$/,
  async function (this: CukeWorld, count: number, selector: string) {
    const el = this.page.locator(selector);
    await expect(el).toHaveCount(Number(count));
  }
);

Then(
  'I should see the following text on the page',
  async function (this: CukeWorld, testData) {
    for (const [selector, textContent] of testData.rows()) {
      await expect
        .poll(
          async () => {
            const el = await this.page
              .locator(
                `${selector}:has-text('${escapeSpecialCharacters(
                  textContent
                )}')`
              )
              .first();
            return el.textContent();
          },
          {
            intervals: [0, 10, 10, 10, 10, 10, 10, 10, 10, 10, 20],
            timeout: 10000,
          }
        )
        .toContain(textContent);
    }
  }
);

Then(
  /^I should( not)? see the following text within the "([^"]+)" card$/,
  async function (this: CukeWorld, not: boolean, cardTitle: string, testData) {
    const card = await findCardByTitle(this, cardTitle);
    expect(card).not.toBe(null);
    if (card) {
      for (const [selector, textContent] of testData.rows()) {
        const el = card.locator(`${selector}:has-text('${textContent}')`);
        if (not) {
          await expect(el).not.toBeVisible();
        } else {
          await expect(el).toContainText(textContent);
        }
      }
    }
  }
);

Then(
  'I should see the following text within the dialog titled {string}',
  async function (this: CukeWorld, dialogTitle: string, testData) {
    const dialog = this.page.locator(
      `dialog:has(h1:has-text("${dialogTitle}"))`
    );
    expect(dialog).not.toBe(null);
    if (dialog) {
      for (const [selector, textContent] of testData.rows()) {
        const el = dialog.locator(`${selector}:has-text('${textContent}')`);
        await expect(el).toBeVisible();
      }
    }
  }
);

Then(
  /^I should( not)? see a dialog with the title "([^"]+)"$/,
  async function (this: CukeWorld, not: boolean, dialogTitle: string) {
    const dialog = this.page.locator(
      `dialog:has(h1:has-text("${dialogTitle}"))`
    );
    if (not) {
      await expect(dialog).not.toBeVisible();
    } else {
      await expect(dialog).toBeVisible();
    }
  }
);

Then(
  /^I should see a(n external)? link to "([^"]+)"( that opens in a new tab)?(?: within the "([^"]+)" card)?$/,
  async function (
    this: CukeWorld,
    external: boolean,
    url: string,
    newTab: boolean,
    cardTitle: string
  ) {
    let context;
    if (cardTitle) {
      context = await findCardByTitle(this, cardTitle);
    } else {
      context = this.page;
    }
    const link = await context?.locator(`a[href="${url}"]`);
    await expect(link).toHaveAttribute('href', url);
    if (external) {
      await expect(link).toHaveAttribute('rel', 'external');
    }
    if (newTab) {
      await expect(link).toHaveAttribute('target', 'new');
    }
  }
);

Then(
  /^I should see ([0-9]+) links to "([^"]+)"$/,
  async function (this: CukeWorld, count: number, url: string) {
    const context = this.page;
    const el = await context?.locator(`a[href="${url}"]`);
    await expect(el).toHaveCount(Number(count));
    await expect(el.first()).toHaveAttribute('href', url);
  }
);

// 'and the description "foo"' is optional
Then(
  /^I should( not)? see a card with the title "([^"]+)"(?: and the description "([^$]+)")?$/,
  async function (
    this: CukeWorld,
    not: boolean,
    title: string,
    description: string
  ) {
    if (not) {
      const el = this.page.locator(`section header h1:has-text("${title}")`);
      await expect(el).not.toBeVisible();
      return;
    }
    const card = await findCardByTitle(this, title);
    if (description) {
      const cardDescription = await card?.locator(
        `header p:has-text("${description}")`
      );
      await expect(cardDescription).toHaveText(description);
    }
  }
);

Then(
  /^I should( not)? see a cardrow with the title "([^"]+)"$/,
  async function (this: CukeWorld, not: boolean, title: string) {
    const el = this.page.locator(`div > h2`, {
      hasText: new RegExp(`^${title}`),
    });
    if (not) {
      await expect(el).not.toBeVisible();
      return;
    }
    await expect(el).toBeVisible();
  }
);

Then(
  /^I should( not)? see a button labeled "([a-zA-Z +]+)"(?: within the "([^"]+)" card)?$/,
  async function (
    this: CukeWorld,
    not: string,
    buttonLabel: string,
    cardTitle: string
  ) {
    let context;
    if (cardTitle) {
      context = await findCardByTitle(this, cardTitle);
    } else {
      context = this.page;
    }
    const selector = `button:is(:has(:text-is("${buttonLabel}")), [aria-label="${buttonLabel}"]):visible`;
    const el = context?.locator(selector);
    if (not) {
      await el?.waitFor({ state: 'detached' });
      await expect(el).not.toBeVisible();
    } else {
      await el?.waitFor({ state: 'visible' });
      await expect(el).toBeVisible();
    }
  }
);

Then(
  /^the button labeled "([a-zA-Z +]+)"(?: within the "([^"]+)" card)? should be (enabled|disabled)$/,
  async function (
    this: CukeWorld,
    buttonLabel: string,
    cardTitle: string,
    state: string
  ) {
    let context;
    if (cardTitle) {
      context = await findCardByTitle(this, cardTitle);
    } else {
      context = this.page;
    }
    const el = context?.locator(
      `button:visible:has(:text-is("${buttonLabel}"))`
    );
    if (state === 'enabled') {
      await expect(el).toBeEnabled();
    } else {
      await expect(el).not.toBeEnabled();
    }
  }
);

Then(
  /^the button with ID "([a-zA-Z +]+)" should be (enabled|disabled)$/,
  async function (this: CukeWorld, id: string, state: string) {
    const context = this.page;
    const el = context?.locator(`button#${id}`);
    if (state === 'enabled') {
      await expect(el).toBeEnabled();
    } else {
      await expect(el).not.toBeEnabled();
    }
  }
);

Then(
  /^I should( not)? see a button with title "([a-zA-Z +]+)""$/,
  async function (this: CukeWorld, not: string, title: string) {
    const context = this.page;
    const el = context?.locator(`button[title="${title}"]`);
    if (not) {
      await expect(el).not.toBeVisible();
    } else {
      await expect(el).toBeVisible();
    }
  }
);

Then(
  /^I should( not)? see an icon button with the title "([a-zA-Z +]+)"(?: within the "([^"]+)" card)?$/,
  async function (
    this: CukeWorld,
    not: string,
    title: string,
    cardTitle: string
  ) {
    let context;
    if (cardTitle) {
      context = await findCardByTitle(this, cardTitle);
    } else {
      context = this.page;
    }
    const el = context?.locator(`button[title="${title}"]`);
    if (not) {
      await expect(el).not.toBeVisible();
    } else {
      await expect(el).toBeVisible();
    }
  }
);

Then(
  /^the button with title "([a-zA-Z +]+)" should be (enabled|disabled)$/,
  async function (this: CukeWorld, title: string, state: string) {
    const context = this.page;
    const el = context?.locator(`button[title="${title}"]`);
    if (state === 'enabled') {
      await expect(el).toBeEnabled();
    } else {
      await expect(el).not.toBeEnabled();
    }
  }
);

Then(
  /^I should see ([0-9]+) buttons labeled "([a-zA-Z +]+)"(?: within the "([^"]+)" card)? and they should be (enabled|disabled)$/,
  async function (
    this: CukeWorld,
    count: number,
    buttonLabel: string,
    cardTitle: string,
    state: string
  ) {
    let context;
    if (cardTitle) {
      context = await findCardByTitle(this, cardTitle);
    } else {
      context = this.page;
    }
    const el = context?.locator(`button:has(:text-is("${buttonLabel}"))`);
    await expect(el).toHaveCount(Number(count));
    for (let i = 0; i < count; i++) {
      if (state === 'enabled') {
        await expect(el?.nth(i)).toBeEnabled();
      } else {
        await expect(el?.nth(i)).not.toBeEnabled();
      }
    }
  }
);

Then(
  /^I should see ([0-9]+) buttons with ID "([a-zA-Z +]+)" and they should be (enabled|disabled)$/,
  async function (
    this: CukeWorld,
    count: number,
    buttonLabel: string,
    state: string
  ) {
    const el = this.page?.locator(`button#${buttonLabel}`);
    await expect(el).toHaveCount(Number(count));
    for (let i = 0; i < count; i++) {
      if (state === 'enabled') {
        await expect(el?.nth(i)).toBeEnabled();
      } else {
        await expect(el?.nth(i)).not.toBeEnabled();
      }
    }
  }
);

Then(
  'I should see a toast notification with the text {string}',
  async function (this: CukeWorld, text: string) {
    const el = await this.page.locator(
      `#notificationCenter > li > div:has-text("${text}")`
    );
    await expect(el).toHaveText(text);
  }
);

Then(
  'I should see a custom dropdown matching selector {string} with the following options',
  async function (this: CukeWorld, selector: string, testData) {
    const select = this.page.locator(selector);
    await expect(select).toBeVisible();
    // open the dropdown
    await select.click();
    const options = select.locator('[role="option"]');
    const rows = testData.rows();
    await expect(options).toHaveCount(rows.length);
    for (let i = 0; i < rows.length; i++) {
      const [text, value, selected] = rows[i];
      const option = await options.nth(i);
      if (value === '') {
        expect(await option.getAttribute('data-value')).toBeFalsy();
      }
      // when we create new items programmatically, the test writer may not know the ID
      else if (value !== '*') {
        await expect(option).toHaveAttribute('data-value', value);
      }
      await expect(option).toHaveText(text);
      if (selected) {
        await expect(option).toHaveAttribute('aria-selected', 'true');
      }
    }
    // close the dropdown
    await select.blur();
  }
);
