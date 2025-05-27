import {
  setWorldConstructor,
  World as CucumberWorld,
} from '@cucumber/cucumber';
import { Page, Browser, BrowserContext } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

export interface CukeWorld extends CucumberWorld {
  page: Page;
  browser: Browser;
  browserContext: BrowserContext;
  activeMocks: any[];
  makeAxeBuilder: () => AxeBuilder;
}

class World {}

setWorldConstructor(World);
