import { chromium, firefox, Browser, BrowserType } from "@playwright/test";

process.env.BABEL_ENV = "production";

const USER_AGENTS: Record<string, BrowserType> = {
  chrome: chromium,
  chromium: chromium,
  firefox: firefox,
};
const BROWSER = (process.env.BROWSER as string) || "chromium";
export const HEADLESS = process.env.HEADLESS !== "false";
export const USER_AGENT = USER_AGENTS[BROWSER] || chromium;
export const DEVTOOLS = process.env.DEVTOOLS === "true";
export const SLOW_MO = process.env.SLOWMO
  ? parseInt(process.env.SLOWMO, 10)
  : 0;
export const DEBUG_MODE = process.env.DEBUG;
export const PORT = process.env.PORT || "3010";

async function isServerRunning(url: string): Promise<boolean> {
  try {
    const response = await fetch(url, {
      method: "HEAD",
      signal: AbortSignal.timeout(1000)
    });
    return response.ok;
  } catch {
    return false;
  }
}

const FORCE_START_SERVER = process.env.FORCE_START_SERVER === "true";

export const PROTOCOL = "http://";
export const DOMAIN = "localhost";
export const HOST = PROTOCOL + DOMAIN + ":" + PORT;

let initialized = false;
let _browser: Browser;
let _server: any;
let _teardown: () => void;

const go = async () => {
  if (!initialized) {
    const serverRunning = !FORCE_START_SERVER && (await isServerRunning(HOST));

    if (serverRunning) {
      console.log("TESTS: detected running dev server at", HOST);
      console.log(
        "TESTS: reusing existing server (set FORCE_START_SERVER=true to start a new one)",
      );
      _teardown = async () => {
        console.log("TESTS: keeping external server running");
      };
      _server = null;
    } else {
      console.log("TESTS: no server detected, starting dev server at", HOST);
      // Only import server if we need to start it
      const serverModule = await import("../../scripts/start.js");
      const server = serverModule.default;
      await server.serverStarted;
      console.log("TESTS: server started");
      await server.bundleBuilt;
      console.log("TESTS: bundle built");
      _teardown = server.teardown;
      _server = server.server;
    }

    _browser = await USER_AGENT.launch({
      headless: HEADLESS,
      devtools: DEVTOOLS,
      slowMo: SLOW_MO || 0,
    });
    console.log("TESTS: browser launched");

    initialized = true;
  }
  return {
    browser: _browser,
    server: _server,
    teardown: _teardown,
  };
};

export default go;