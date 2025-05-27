import { Then } from '@cucumber/cucumber';
import { expect } from '@playwright/test'; // Import expect directly from playwright
import { CukeWorld } from '../support/world';

Then(
  'the page should have no accessibility violations',
  async function (this: CukeWorld) {
    // Ensure makeAxeBuilder is added to the CukeWorld type and initialized in your hooks
    if (!this.makeAxeBuilder) {
      throw new Error(
        'makeAxeBuilder function is not available in the world context. Ensure it is initialized in hooks.'
      );
    }

    const accessibilityScanResults = await this.makeAxeBuilder().analyze();

    expect(accessibilityScanResults.violations).toEqual([]);
  }
);
