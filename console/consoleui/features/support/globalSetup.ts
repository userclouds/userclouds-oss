import { chromium, firefox, Browser, BrowserType } from '@playwright/test';
import server from '../../scripts/start.js';

process.env.BABEL_ENV = 'production';

const USER_AGENTS: Record<string, BrowserType> = {
  chrome: chromium,
  chromium: chromium,
  firefox: firefox,
};
const BROWSER = (process.env.BROWSER as string) || 'chromium';
export const HEADLESS = process.env.HEADLESS !== 'false';
export const USER_AGENT = USER_AGENTS[BROWSER];
export const DEVTOOLS = process.env.DEVTOOLS === 'true';
export const SLOW_MO = process.env.SLOWMO
  ? parseInt(process.env.SLOWMO, 10)
  : 0;
export const DEBUG_MODE = process.env.DEBUG;
export const PORT = process.env.PORT || '3010';
export const DOMAIN = 'console.dev.userclouds.tools';
export const PROTOCOL = 'http://';
export const HOST = PROTOCOL + DOMAIN;

let initialized = false;
let _browser: Browser;
let _server: any;
let _teardown: () => void;

const go = async () => {
  if (!initialized) {
    await server.serverStarted;
    console.log('TESTS: server started');
    await server.bundleBuilt;
    console.log('TESTS: bundle built');

    _browser = await USER_AGENT.launch({
      headless: HEADLESS,
      devtools: DEVTOOLS,
      slowMo: SLOW_MO || 0,
    });
    console.log('TESTS: browser launched');
    _teardown = server.teardown;
    _server = server.server;

    initialized = true;
  }
  return {
    browser: _browser,
    server: _server,
    teardown: _teardown,
  };
};

export default go;
