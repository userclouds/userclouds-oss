import { When, Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test';

import {
  enterText,
  enterTextInput,
  findCardByRowTitle,
  findCardByTitle,
} from './helpers';
import { CukeWorld } from '../support/world';

When(
  'I type {string} into the {string} field',
  async function (this: CukeWorld, text: string, fieldName: string) {
    await this.page.locator(`body [name="${fieldName}"]`).focus();
    // Move cursor to end
    if (process.platform === 'darwin') {
      // Mac
      await this.page.keyboard.press('Meta+ArrowRight');
    } else {
      // Linux (CI)
      await this.page.keyboard.press('End');
    }
    await enterText(this, text, fieldName);
  }
);

When(
  'I replace the text in the {string} field with {string}',
  async function (this: CukeWorld, fieldName: string, text: string) {
    // including body so we don't clash with meta tags, which often include name
    const field = this.page.locator(`body [name="${fieldName}"]`);
    await field.selectText();
    await field.fill('');
    await field.fill(text);
  }
);

When(
  'I type {string} into the input with ID {string}',
  async function (this: CukeWorld, text: string, inputID: string) {
    await enterTextInput(this, text, inputID);
  }
);

When(
  'I select {string} in the {string} field',
  async function (this: CukeWorld, textToSelect: string, fieldName: string) {
    const field = this.page.locator(`body [name="${fieldName}"]`);
    await field.focus();
    // Move cursor to beginning
    if (process.platform === 'darwin') {
      // Mac
      await this.page.keyboard.press('Meta+ArrowLeft');
    } else {
      // Linux (CI)
      await this.page.keyboard.press('Home');
    }
    const fieldValue = await field.inputValue();
    const match = fieldValue.indexOf(textToSelect);
    expect(match).not.toBe(-1);
    // move cursor to beginning of matching text
    for (let i = 0; i < match; i++) {
      await field.press('ArrowRight');
    }
    // select text
    await this.page.keyboard.down('Shift');
    for (let i = 0; i < textToSelect.length; i++) {
      await this.page.keyboard.press('ArrowRight');
    }
    await this.page.keyboard.up('Shift');
  }
);

When(
  'I submit the {string} form',
  async function (this: CukeWorld, name: string) {
    await this.page.$eval(`form[name="${name}"]`, (form: HTMLFormElement) =>
      form.submit()
    );
  }
);

When(
  'I select the option labeled {string} in the dropdown matching selector {string}',
  async function (this: CukeWorld, optionLabel, selector) {
    const select = await this.page.waitForSelector(selector);
    await select.selectOption({ label: optionLabel });
  }
);

When(
  'I select the option labeled {string} in the dropdown with ID {string}',
  async function (this: CukeWorld, optionLabel, id) {
    const select = await this.page.waitForSelector(`select#${id}`);
    await select.selectOption({ label: optionLabel });
  }
);

When(
  /I toggle the (checkbox|radio) labeled "([a-zA-Z0-9 -&?]+)"/,
  async function (this: CukeWorld, checkboxOrRadio: string, labelText: string) {
    const label = this.page.locator(
      `label:text-is("${labelText}"), label:has(:text-is("${labelText}"))`
    );
    const checkbox = label.locator(`input[type="${checkboxOrRadio}"]`);
    const checked = await checkbox.isChecked();
    await checkbox.click({ force: true });
    expect(await checkbox.isChecked()).toBe(!checked);
  }
);

When(
  'I click the delete icon next to row {int} of the editable list with ID {string}',
  async function (this: CukeWorld, rowNum: number, id: string) {
    const stringList = this.page.locator(`.editableStringList#${id}`);
    const rows = await stringList.locator('li').all();
    const row = rows[rowNum - 1];
    const deleteButton = await row.locator('button[title="Delete Element"]');
    await deleteButton.click();
  }
);

When(
  'I change the text in row {int} of the editable list with ID {string} to {string}',
  async function (
    this: CukeWorld,
    rowNum: number,
    id: string,
    newText: string
  ) {
    const stringList = this.page.locator(`.editableStringList#${id}`);
    const rows = await stringList.locator('li').all();
    const row = rows[rowNum - 1];
    const input = await row.locator('input[type]');
    await input.selectText();
    await input.type('');
    await input.type(newText);
  }
);

When(
  'I type {string} into the field matching selector {string}',
  async function (string, string2) {
    const el = await this.page.locator(string2);
    await el.fill(string);
  }
);

Then(
  'the values in the editable list with ID {string} should be',
  async function (this: CukeWorld, id: string, testData) {
    const stringList = this.page.locator(`.editableStringList#${id}`);
    const rows = await stringList.locator('li').all();
    const testRows = testData.rows();
    expect(rows.length).toEqual(testRows.length);
    for (let i = 0; i < testRows.length; i++) {
      const [value] = testRows[i];
      const input = rows[i].locator('input[type]');
      await expect(input).toHaveValue(value);
    }
  }
);

Then(
  'I should see a dropdown matching selector {string} with the following options',
  async function (this: CukeWorld, selector, testData) {
    const select = this.page.locator(selector);
    await expect(select).toBeVisible();
    const options = select.locator('option');
    const rows = testData.rows();
    await expect(options).toHaveCount(rows.length);
    for (let i = 0; i < rows.length; i++) {
      const [text, value, selected] = rows[i];
      if (value === '') {
        expect(await options.nth(i).getAttribute('value')).toBeFalsy();
      }
      // when we create new items programmatically, the test writer may not know the ID
      else if (value !== '*') {
        await expect(options.nth(i)).toHaveAttribute('value', value);
      }
      await expect(options.nth(i)).toHaveText(text);
      if (selected) {
        await expect(select).toHaveValue(value);
      }
    }
  }
);

Then(
  'I should see a dropdown matching selector {string} without the following options',
  async function (this: CukeWorld, selector, testData) {
    const select = this.page.locator(selector);
    await expect(select).toBeVisible();
    const options = select.locator('option');
    const rows = testData.rows();
    for (let i = 0; i < rows.length; i++) {
      const [text] = rows[i];
      for (let j = 0; j < (await options.count()); j++) {
        await expect(options.nth(j)).not.toHaveText(text);
      }
    }
  }
);

Then(
  'I should not see a dropdown matching selector {string}',
  async function (this: CukeWorld, selector) {
    const select = this.page.locator(selector);
    await expect(select).not.toBeVisible();
  }
);

Then(
  /the input with the name "([a-zA-Z0-9\-_]+)" should be (disabled|enabled)/,
  async function (
    this: CukeWorld,
    inputName: string,
    enabledOrDisabled: string
  ) {
    const input = this.page.locator(`body [name="${inputName}"]`);
    if (enabledOrDisabled === 'enabled') {
      await expect(input).toBeEnabled();
    } else {
      await expect(input).not.toBeEnabled();
    }
  }
);

Then(
  /the input with the ID "([a-zA-Z0-9\-_]+)" should be (disabled|enabled)/,
  async function (this: CukeWorld, inputID: string, enabledOrDisabled: string) {
    const input = this.page.locator(`input#${inputID}`);
    if (enabledOrDisabled === 'enabled') {
      await expect(input).toBeEnabled();
    } else {
      await expect(input).not.toBeEnabled();
    }
  }
);

Then(
  'the input with the name {string} should have the value {string}',
  async function (this: CukeWorld, inputName: string, inputValue: string) {
    const input = await this.page.locator(`body [name="${inputName}"]`);
    await expect(input).toHaveValue(inputValue);
  }
);

Then(
  'the input with the ID {string} should have the value {string}',
  async function (this: CukeWorld, inputID: string, inputValue: string) {
    const input = await this.page.locator(`input#${inputID}`);
    await expect(input).toHaveValue(inputValue);
  }
);

Then(
  /the input with the name "([a-zA-Z0-9\-_]+)" should be (valid|invalid)/,
  async function (this: CukeWorld, inputName: string, validOrInvalid: string) {
    const input = this.page.locator(`body [name="${inputName}"]`);
    const isValid: boolean = await input.evaluate((node: HTMLInputElement) => {
      const state: ValidityState = node.validity;
      return state.valid;
    });
    expect(isValid).toBe(validOrInvalid === 'valid');
  }
);

Then(
  /the (checkbox|radio) labeled "([a-zA-Z0-9& +\-?!]+)" should be (checked|unchecked)/,
  async function (
    this: CukeWorld,
    checkboxOrRadio: string,
    labelText: string,
    checked: string
  ) {
    const label = this.page.locator(
      `label:text-is("${labelText}"), label:has(:text-is("${labelText}"))`
    );
    const checkbox = label.locator(`input[type="${checkboxOrRadio}"]`);
    const shouldBeChecked = checked === 'checked';
    if (shouldBeChecked) {
      await expect(checkbox).toBeChecked();
    } else {
      await expect(checkbox).not.toBeChecked();
    }
  }
);

Then(
  /^I should( not)? see a checkbox labeled "([a-zA-Z ?+]+)"(?: within the "([^"]+)" card)?$/,
  async function (
    this: CukeWorld,
    not: string,
    label: string,
    cardTitle: string
  ) {
    let context;
    if (cardTitle) {
      context = await findCardByTitle(this, cardTitle);
    } else {
      context = this.page;
    }
    const selector = `label:has-text("${label}") input[type="checkbox"]`;
    const el = context?.locator(selector);
    if (not) {
      await expect(el).not.toBeVisible();
    } else {
      await expect(el).toBeVisible();
    }
  }
);

Then(
  /I should see the following form elements(?: within the form with ID "([a-zA-Z\-_]+)")?/,
  async function (this: CukeWorld, formID: string, testData) {
    const context = formID ? this.page.locator(`form#${formID}`) : this.page;
    for (const [tagName, type, name, value, disabled] of testData.rows()) {
      const selector = `${tagName}${
        type ? `[type="${type}"]` : ''
      }[name="${name}"]`;
      const el = context.locator(selector);
      if (tagName === 'textarea') {
        await expect(el).toHaveText(value);
      } else if (value.trim() === '') {
        await expect(el).toHaveValue('');
      } else {
        await expect(el).toHaveValue(value);
      }
      if (disabled === 'true') {
        await expect(el).not.toBeEnabled();
      }
    }
  }
);

Then(
  'I should see the following inputs',
  async function (this: CukeWorld, testData) {
    for (const [type, name, value, disabled] of testData.rows()) {
      const selector = `input${type ? `[type="${type}"]` : ''}[name="${name}"]`;
      const el = this.page.locator(selector);
      await expect(el).toHaveValue(value);
      if (disabled === 'true') {
        await expect(el).not.toBeEnabled();
      } else {
        await expect(el).toBeEnabled();
      }
    }
  }
);

Then(
  'I should see the following inputs within the {string} card',
  async function (this: CukeWorld, cardTitle, testData) {
    const card = await findCardByTitle(this, cardTitle);
    await expect(card).toBeVisible();
    for (const [type, name, value, disabled] of testData.rows()) {
      const selector = `input${type ? `[type="${type}"]` : ''}[name="${name}"]`;
      const el = await card?.first().locator(selector);
      await expect(el).toHaveValue(value);
      if (disabled === 'true') {
        await expect(el).not.toBeEnabled();
      } else {
        await expect(el).toBeEnabled();
      }
    }
  }
);

Then(
  'I should see the following inputs within the {string} cardrow',
  async function (this: CukeWorld, cardTitle, testData) {
    const card = await findCardByRowTitle(this, cardTitle);
    await expect(card).toBeVisible();
    for (const [type, name, value, disabled] of testData.rows()) {
      const selector = `input${type ? `[type="${type}"]` : ''}[name="${name}"]`;
      const el = await card?.first().locator(selector);
      await expect(el).toHaveValue(value);
      if (disabled === 'true') {
        await expect(el).not.toBeEnabled();
      } else {
        await expect(el).toBeEnabled();
      }
    }
  }
);

Then(
  /I should( not)? see a code editor with the ID "([a-zA-Z_-]+)"(?: and the value "(.+)")?/,
  async function (
    this: CukeWorld,
    not: string,
    editorID: string,
    expectedValue: string
  ) {
    const editor = this.page.locator(`#${editorID}`);

    if (not) {
      await expect(editor).not.toBeVisible();
    } else {
      // Access the editor's value via a DOM query
      const value = await this.page.evaluate((id) => {
        // Find the CodeMirror editor instance and get its value
        const editorElement = document.querySelector(`#${id} .cm-content`);
        return editorElement?.textContent || '';
      }, editorID);

      // Normalize whitespace
      const normalizedValue = value.replace(/\s+/g, ' ').trim();
      const normalizedExpectedValue = expectedValue.replace(/\s+/g, ' ').trim();

      expect(normalizedValue).toBe(normalizedExpectedValue);
    }
  }
);
When(
  'I replace the text in the code editor with the ID {string} with the value {string}',
  async function (this: CukeWorld, editorID: string, newValue: string) {
    const editor = this.page.locator(
      `#${editorID} .cm-content[contenteditable]`
    );

    // Ensure the editor is visible and ready for interaction
    await expect(editor).toBeVisible();
    await expect(editor).toBeEnabled();

    // Add a small delay to ensure the element is ready
    await this.page.waitForTimeout(100);

    // Use evaluate to set the content
    await editor.evaluate((node: HTMLDivElement, value: string) => {
      node.textContent = value;
    }, newValue);
  }
);
