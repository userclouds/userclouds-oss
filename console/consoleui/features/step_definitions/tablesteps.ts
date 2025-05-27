import { When, Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test';

import {
  findTableByCardRowTitle,
  findTableByCardTitle,
  selectedDropdownLabel,
} from './helpers';
import { CukeWorld } from '../support/world';

When(
  'I click the delete button in row {int} of the table in the {string} card',
  async function (this: CukeWorld, rowNumber: number, cardTitle: string) {
    const table = await findTableByCardTitle(this, cardTitle);
    if (table) {
      const trs = await table.locator('> tbody > tr').all();
      const tds = await trs[rowNumber - 1].locator('td').all();
      const button = await tds[tds.length - 1].locator('button');
      await button.click();
    }
  }
);

When(
  'I click the delete button in row {int} of the table with ID {string}',
  async function (this: CukeWorld, rowNumber: number, tableID: string) {
    const table = await this.page.locator(`table#${tableID}`);
    if (table) {
      const trs = await table.locator('> tbody > tr').all();
      const tds = await trs[rowNumber - 1].locator('td').all();
      const button = await tds[tds.length - 1].locator('button');
      await button.click();
    }
  }
);

When(
  'I click the {string} button in row {int} of the table with ID {string}',
  async function (
    this: CukeWorld,
    buttonType: string,
    rowNumber: number,
    tableID: string
  ) {
    const table = await this.page.locator(`table#${tableID}`);
    const trs = await table.locator('tbody > tr').all();
    const row = trs[rowNumber - 1];

    // Define the title text for the SVG inside the button
    const titleText = buttonType === 'delete' ? 'Delete Bin' : 'Pencil';

    // Search all <td> elements in the row to locate the button
    const button = await row
      .locator(`td button:has(svg > title:has-text("${titleText}"))`)
      .first();

    await button.click();
  }
);

When(
  'I toggle the checkbox in column {int} of row {int} of the table with ID {string}',
  async function (
    this: CukeWorld,
    columnNumber: number,
    rowNumber: number,
    id: string
  ) {
    const table = await this.page.locator(`table#${id}`);
    if (table) {
      const trs = await table.locator('tbody > tr').all();
      const tds = await trs[rowNumber - 1].locator('td').all();
      const label = await tds[columnNumber - 1].locator('label');
      const checkbox = label.locator(`input[type="checkbox"]`);
      const checked = await checkbox.isChecked();
      await checkbox.dispatchEvent('click');
      expect(await checkbox.isChecked()).toBe(!checked);
    }
  }
);

When(
  'I click the radio input in row {int} of the table with ID {string}',
  async function (this: CukeWorld, rowNumber: number, tableID: string) {
    const table = await this.page.locator(`table#${tableID}`);
    const trs = await table.locator('tbody > tr').all();
    const tds = await trs[rowNumber - 1].locator('td').all();
    const input = await tds[0].locator('input[type="radio"]');
    await input.click({ force: true });
  }
);

When(
  'I toggle the checkbox in column {int} of row {int} of the table in the {string} card',
  async function (
    this: CukeWorld,
    columnNumber: number,
    rowNumber: number,
    cardTitle: string
  ) {
    const table = await findTableByCardTitle(this, cardTitle);
    if (table) {
      const trs = await table.locator('tbody > tr').all();
      const tds = await trs[rowNumber - 1].locator('td').all();
      const checkbox = await tds[columnNumber - 1].locator(
        'input[type="checkbox"]'
      );
      await checkbox.click({ force: true });
    }
  }
);

When(
  'I enter {string} into column {int} of row {int} of the table in the {string} card',
  async function (
    this: CukeWorld,
    inputText: string,
    columnNumber: number,
    rowNumber: number,
    cardTitle: string
  ) {
    const table = await findTableByCardTitle(this, cardTitle);
    if (table) {
      const trs = await table.locator('tbody > tr').all();
      const tds = await trs[rowNumber - 1].locator('td').all();
      const input = await tds[columnNumber - 1].locator('input[type="text"]');
      await input.selectText();
      await input.type(inputText);
    }
  }
);

When(
  'I select {string} from the dropdown in column {int} of row {int} of the table with ID {string}',
  async function (
    this: CukeWorld,
    optionLabel: string,
    columnNumber: number,
    rowNumber: number,
    tableID: string
  ) {
    const table = await this.page.locator(`table#${tableID}`);
    if (table) {
      const trs = await table.locator('tbody > tr').all();
      const tds = await trs[rowNumber - 1].locator('td').all();
      const select = await tds[columnNumber - 1].locator('select');
      await select.selectOption({ label: optionLabel });
    }
  }
);

When(
  'I select {string} from the dropdown in column {int} of row {int} of the table in the {string} card',
  async function (
    this: CukeWorld,
    optionLabel: string,
    columnNumber: number,
    rowNumber: number,
    cardTitle: string
  ) {
    const table = await findTableByCardTitle(this, cardTitle);
    if (table) {
      const trs = await table.locator('tbody > tr').all();
      const tds = await trs[rowNumber - 1].locator('td').all();
      const select = await tds[columnNumber - 1].locator('select');
      await select.selectOption({ label: optionLabel });
    }
  }
);

Then(
  'I should see a table within the {string} card with the following data',
  async function (this: CukeWorld, cardTitle, testData) {
    const table = await findTableByCardTitle(this, cardTitle);
    if (table) {
      const trs = table.locator('> tbody > tr');
      const rows = testData.rows();
      await expect(trs).toHaveCount(rows.length);
      for (let i = 0; i < rows.length; i++) {
        const row = rows[i];
        const tds = await trs.nth(i).locator('> td');
        for (let j = 0; j < row.length; j++) {
          const select = tds.nth(j).locator('select');
          const selectExists = (await select.count()) > 0;
          if (selectExists) {
            const selectedText = await selectedDropdownLabel(select);
            await expect(selectedText).toEqual(row[j]);
          } else {
            await expect(tds.nth(j)).toHaveText(row[j]);
          }
        }
      }
    }
  }
);

Then(
  'I should see a table within the {string} card row with the following data',
  async function (this: CukeWorld, rowTitle, testData) {
    const table = await findTableByCardRowTitle(this, rowTitle);
    if (table) {
      const trs = table.locator('> tbody > tr');
      const rows = testData.rows();
      await expect(trs).toHaveCount(rows.length);
      for (let i = 0; i < rows.length; i++) {
        const row = rows[i];
        const tds = await trs.nth(i).locator('> td');
        for (let j = 0; j < row.length; j++) {
          const select = tds.nth(j).locator('select');
          const selectExists = (await select.count()) > 0;
          if (selectExists) {
            const selectedText = await selectedDropdownLabel(select);
            await expect(selectedText).toEqual(row[j]);
          } else {
            await expect(tds.nth(j)).toHaveText(row[j]);
          }
        }
      }
    }
  }
);

Then(
  'I should see a table with ID {string} and the following data',
  async function (this: CukeWorld, tableID, testData) {
    const table = this.page.locator(`table#${tableID}`);
    await expect(table).toBeVisible({ timeout: 5000 });

    const trs = table.locator('tbody > tr');
    const rows = testData.rows();
    await expect(trs).toHaveCount(rows.length);

    for (let i = 0; i < rows.length; i++) {
      const row = rows[i];
      const tds = trs.nth(i).locator('> td');

      for (let j = 0; j < row.length; j++) {
        const td = tds.nth(j);
        const select = td.locator('select');
        const selectExists = (await select.count()) > 0;

        if (selectExists) {
          // Wait for the select to be visible and enabled
          await expect(select).toBeVisible({ timeout: 5000 });
          await expect(select).toBeEnabled({ timeout: 5000 });

          const expectedText = row[j].trim();

          // If the expected text matches the placeholder pattern, verify the default state
          if (expectedText.startsWith('Select a')) {
            const selectedText = await selectedDropdownLabel(select);
            expect(selectedText?.trim()).toEqual(expectedText);
          } else {
            // Handle normal selected options
            const selectedText = await selectedDropdownLabel(select);
            if (!selectedText) {
              throw new Error(
                `No selected option text found in column ${j + 1} of row ${i + 1}`
              );
            }

            // Compare the trimmed text, ignoring any "(default)" suffix
            const actualText = selectedText.trim().replace(' (default)', '');
            expect(actualText).toEqual(expectedText);
          }
        } else {
          // For non-select cells, check the text content
          await expect(td).toHaveText(row[j].trim(), { timeout: 5000 });
        }
      }
    }
  }
);

Then(
  /row ([0-9]+) of the table in the "([a-zA-Z\-_ ]+)" card should( not)? be marked for delete/,
  async function (
    this: CukeWorld,
    rowNumber: number,
    cardTitle: string,
    notMarked: string
  ) {
    const table = await findTableByCardTitle(this, cardTitle);
    if (table) {
      const trs = await table.locator('tbody > tr').all();
      if (notMarked) {
        expect(await trs[rowNumber - 1].getAttribute('class')).not.toContain(
          'queuedfordelete'
        );
      } else {
        expect(await trs[rowNumber - 1].getAttribute('class')).toContain(
          'queuedfordelete'
        );
      }
    }
  }
);

Then(
  /row ([0-9]+) of the table with ID "([a-zA-Z\-_ ]+)" should( not)? be marked for delete/,
  async function (
    this: CukeWorld,
    rowNumber: number,
    tableID: string,
    notMarked: string
  ) {
    const table = await this.page.locator(`table#${tableID}`);
    if (table) {
      const trs = await table.locator('tbody > tr').all();
      if (notMarked) {
        expect(await trs[rowNumber - 1].getAttribute('class')).not.toContain(
          'queuedfordelete'
        );
      } else {
        expect(await trs[rowNumber - 1].getAttribute('class')).toContain(
          'queuedfordelete'
        );
      }
    }
  }
);

Then(
  /^the (input|dropdown) in column (\d+) of row (\d+) of the table in the "([^"]+)" card should have the value "([^"]+)"$/,
  async function (
    this: CukeWorld,
    inputOrSelect: string,
    columnNumber: number,
    rowNumber: number,
    cardTitle: string,
    inputValue: string
  ) {
    const table = await findTableByCardTitle(this, cardTitle);
    if (table) {
      const trs = await table.locator('tbody > tr').all();
      const tds = await trs[rowNumber - 1].locator('td').all();
      const input = await tds[columnNumber - 1].locator(
        inputOrSelect === 'dropdown' ? 'select' : 'input'
      );
      await expect(input).toHaveValue(inputValue);
    }
  }
);

Then(
  /^I should see a dropdown in column (\d+) of row (\d+) of the table in the "([^"]+)" card with the following options$/,
  async function (
    this: CukeWorld,
    columnNumber: number,
    rowNumber: number,
    cardTitle: string,
    testData
  ) {
    const table = await findTableByCardTitle(this, cardTitle);
    if (table) {
      const trs = table.locator('tbody > tr');
      const tds = trs.nth(rowNumber - 1).locator('td');
      const select = tds.nth(columnNumber - 1).locator('select');
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
  }
);
Then(
  /^I should see a dropdown in column (\d+) of row (\d+) of the table with ID "([^"]+)" and the following options$/,
  async function (
    this: CukeWorld,
    columnNumber: number,
    rowNumber: number,
    tableID: string,
    testData
  ) {
    const table = await this.page.locator(`table#${tableID}`);
    if (table) {
      const trs = table.locator('tbody > tr');
      const tds = trs.nth(rowNumber - 1).locator('td');
      const select = tds.nth(columnNumber - 1).locator('select');
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
  }
);

Then(
  'the input in column {int} of row {int} of the table in the {string} card should be disabled',
  async function (
    this: CukeWorld,
    columnNumber: number,
    rowNumber: number,
    cardTitle: string
  ) {
    const table = await findTableByCardTitle(this, cardTitle);
    if (table) {
      const trs = await table.locator('tbody > tr').all();
      const tds = await trs[rowNumber - 1].locator('td').all();
      const input = await tds[columnNumber - 1].locator('input');
      await expect(input).not.toBeEnabled();
    }
  }
);

Then(
  /^the checkbox in column (\d+) of row (\d+) of the table in the "([^"]+)" card should be (un)?checked$/,
  async function (
    this: CukeWorld,
    columnNumber: number,
    rowNumber: number,
    cardTitle: string,
    unchecked: string
  ) {
    const table = await findTableByCardTitle(this, cardTitle);
    if (table) {
      const trs = await table.locator('tbody > tr').all();
      const tds = await trs[rowNumber - 1].locator('td').all();
      const input = await tds[columnNumber - 1].locator(
        'input[type="checkbox"]'
      );
      if (unchecked) {
        await expect(input).not.toBeChecked();
      } else {
        await expect(input).toBeChecked();
      }
    }
  }
);

Then(
  'I should see a filter with the text {string}',
  async function (this: CukeWorld, filterString) {
    const filter = await this.page.locator(`div:has-text("${filterString}")`);
    await expect(filter).toHaveText(filterString);
  }
);
