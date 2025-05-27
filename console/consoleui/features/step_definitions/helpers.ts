import { Locator } from '@playwright/test';
import { JSONValue } from '@userclouds/sharedui';
import { CukeWorld } from '../support/world';
import { DEBUG_MODE, SLOW_MO } from '../support/globalSetup';

export type RequestMethod = 'GET' | 'POST' | 'PUT' | 'DELETE';
export type RequestParams = {
  url: string;
  method?: RequestMethod | undefined;
  status?: number;
  body?: JSONValue;
  headers?: JSONValue;
};

export async function mockRequest(
  world: CukeWorld,
  params: RequestParams,
  options: JSONValue & { times: number | typeof Infinity }
) {
  const { url, method, status, body, headers } = params;
  const response: JSONValue = {};
  response.status = status || 200;
  if (params.hasOwnProperty('body')) {
    response.contentType = 'application/json';
    response.body = JSON.stringify(body);
  }
  if (headers) {
    response.headers = headers;
  }
  let routeOptions: JSONValue = {};
  if (options?.hasOwnProperty('times')) {
    if (options?.times !== Infinity) {
      routeOptions.times = options.times;
    }
  } else {
    routeOptions = { times: 1 };
  }
  // make a queue of the mocks as they're registered
  if (routeOptions.times) {
    const times = routeOptions.times as number;
    for (let i = 0; i < times; i++) {
      world.activeMocks.push(params);
    }
  }
  // introduce a slight delay to simulate real conditions
  let interval = SLOW_MO ? Math.min(SLOW_MO * 2, 200) : 200;
  // stagger fulfillment of mocks that were queued in direct succession
  interval += world.activeMocks.length * 50;
  await world.page.route(
    url,
    (route) => {
      if (!method || method === route.request().method()) {
        setTimeout(() => {
          if (DEBUG_MODE) {
            console.log(
              Date.now(),
              `Found a matching mock for request to ${url}. Responding to ${route
                .request()
                .method()} request with status ${
                response.status
              } and the following body:`,
              response.body
            );
          }
          route.fulfill(response);
          // remove mocks from queue as they're fulfilled
          const matchingMock = world.activeMocks.findIndex(
            (mock: any) =>
              mock.method === method &&
              mock.url === url &&
              mock.status === status
          );
          if (matchingMock > -1) {
            world.activeMocks.splice(matchingMock, 1);
          }
        }, interval);
      } else {
        route.fallback();
      }
    },
    routeOptions
  );
}

export async function goToPage(world: CukeWorld, path: string) {
  const URL_PREFIX = `http://console.dev.userclouds.tools:${
    process.env.PORT || 3333
  }`;
  await world.page.goto(URL_PREFIX + path);
}

export async function enterText(
  world: CukeWorld,
  text: string,
  fieldName: string
) {
  // including body so we don't clash with meta tags, which often include name
  await world.page.locator(`body [name="${fieldName}"]`).type(text);
}

export async function enterTextInput(
  world: CukeWorld,
  text: string,
  inputID: string
) {
  await world.page.locator(`input#${inputID}`).type(text);
}

export async function pressButton(world: CukeWorld, buttonText: string) {
  const button = world.page.locator(
    `button:visible:is(:has-text("${buttonText}"), [aria-label="${buttonText}"])`
  );
  await button.first().click();
}

export async function findCardByTitle(world: CukeWorld, cardTitle: string) {
  return world.page.locator(`section:has(h1:has-text("${cardTitle}"))`);
}

export async function findCardByRowTitle(world: CukeWorld, rowTitle: string) {
  return world.page.locator(`section:has(h2:has-text("${rowTitle}"))`);
}

export async function findTableByCardTitle(
  world: CukeWorld,
  cardTitle: string
) {
  const card = await findCardByTitle(world, cardTitle);
  return card?.locator('table').first();
}

export async function findTableByCardRowTitle(
  world: CukeWorld,
  rowTitle: string
) {
  const card = await findCardByRowTitle(world, rowTitle);
  return card?.locator('table').first();
}

export const selectedDropdownLabel = async (select: Locator) => {
  return select.evaluate((dropdown) => {
    const dropdownElem = dropdown as HTMLSelectElement;
    return dropdownElem.options[dropdownElem.selectedIndex].innerText;
  });
};

export const hexToRgb = (hex: string) => {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
  return result
    ? `rgb(${parseInt(result[1], 16)}, ${parseInt(result[2], 16)}, ${parseInt(
        result[3],
        16
      )})`
    : '';
};

export const escapeSpecialCharacters = (text: string) => {
  return text.replace(/[-/\\^$*+?.()|[\]{}]/g, '\\$&').replace(/'/g, "\\'");
};
