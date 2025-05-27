/* eslint max-classes-per-file: ["error", 5] */

import { version } from '../../package.json';

class APIError extends Error {
  code: number;

  e: Error | undefined;

  body: string;

  constructor(
    message: string,
    body: string,
    code: number,
    e: Error | undefined
  ) {
    super(message);
    Object.setPrototypeOf(this, APIError.prototype);
    this.body = body;
    this.code = code;
    this.e = e;
  }
}

class APIErrorResponse {
  error: string;

  id: string;

  secondary_id: string;

  identical: boolean;

  constructor(
    error: string,
    id: string,
    secondary_id: string,
    identical: boolean
  ) {
    this.error = error;
    this.id = id;
    this.secondary_id = secondary_id;
    this.identical = identical;
  }

  static fromJSON(json: string): APIErrorResponse {
    let obj = JSON.parse(json);
    let nestedError = obj.error;
    while (nestedError.error) {
      obj = nestedError;
      nestedError = obj.error;
    }
    return new APIErrorResponse(
      obj.error,
      obj.id,
      obj.secondary_id,
      obj.identical
    );
  }
}

class BaseClient {
  authHeader: string;

  baseUrl: string;

  constructor(baseUrl: string, authHeader: string) {
    this.baseUrl = baseUrl;
    this.authHeader = authHeader;
  }

  protected makeRequest<T>(
    url: string,
    method = 'GET',
    params: { [key: string]: string } = {},
    body: string | undefined = undefined
  ): Promise<T> {
    return new Promise((resolve, reject) => {
      fetch(this.makeURL(url, params), {
        method,
        body,
        headers: {
          Authorization: `Bearer ${this.authHeader}`,
          'X-Usercloudssdk-Version': version,
          'User-Agent': `UserClouds TypeScript SDK v${version}`,
        },
      })
        .then(async (response) => {
          if (!response.ok) {
            const responseText = await response.text();
            reject(
              new APIError(
                response.statusText,
                responseText,
                response.status,
                undefined
              )
            );
          }

          if (response.status === 204) {
            // go away typescript, you don't understand me :)
            // I wish there was a way to make this safer but I haven't figured
            // it out yet -- in some cases we use this function to with <T> void
            // (like DELETE, where there is no object to return), but since
            // typescript is compile-time-only, I haven't found notation that
            // lets us guarantee this codepath only runs when T===void
            resolve(null as T);
          }

          return response.json();
        }, reject)
        .then((json) => {
          resolve(json);
        }, reject);
    });
  }

  protected makePaginatedRequest<T>(
    url: string,
    parameters: { [key: string]: string },
    startingAfter: string,
    limit: number
  ): Promise<[T[], boolean]> {
    const params = { ...parameters };
    params.starting_after = startingAfter;
    params.limit = limit.toString();
    return new Promise((resolve, reject) => {
      fetch(this.makeURL(url, params), {
        headers: { Authorization: `Bearer ${this.authHeader}` },
      })
        .then(async (response) => {
          if (!response.ok) {
            const responseText = await response.text();
            reject(
              new APIError(
                response.statusText,
                responseText,
                response.status,
                undefined
              )
            );
          }

          return response.json();
        }, reject)
        .then((json) => {
          if ('data' in json) {
            resolve([json.data, json.has_next]);
          } else {
            reject(new Error('An unknown error occurred'));
          }
        }, reject);
    });
  }

  protected makeURL(path: string, params: { [key: string]: string } = {}) {
    const url = new URL(path, this.baseUrl);
    Object.keys(params).forEach((key) => {
      url.searchParams.append(key, params[key]);
    });
    return url.toString();
  }
}

export default BaseClient;
export { APIError, APIErrorResponse };
