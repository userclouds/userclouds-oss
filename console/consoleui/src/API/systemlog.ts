import {
  HTTPError,
  extractErrorMessage,
  makeAPIError,
} from '@userclouds/sharedui';
import { SystemLogEntry, SystemLogEntryRecord } from '../models/SystemLogEntry';
import PaginatedResult from '../models/PaginatedResult';
import { makeCompanyConfigURL, PAGINATION_API_VERSION } from '../API';

export const fetchSystemLogs = (
  tenantID: string,
  queryParams: Record<string, string>
): Promise<PaginatedResult<SystemLogEntry>> => {
  return new Promise(async (resolve, reject) => {
    if (!queryParams.version) {
      queryParams.version = PAGINATION_API_VERSION;
    }

    // queryParams.type = 'app';

    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/runs`,
      queryParams
    );

    return fetch(url)
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(
          makeAPIError(
            error,
            `error fetching system logs (tenant uuid: ${tenantID})`
          )
        );
      });
  });
};
export const fetchSingleLogEntry = (
  tenantID: string,
  id: string
): Promise<PaginatedResult<SystemLogEntry>> => {
  return new Promise(async (resolve, reject) => {
    const queryParams = {} as Record<string, string>;
    if (!queryParams.version) {
      queryParams.version = PAGINATION_API_VERSION;
    }
    queryParams.filter = `(('id',EQ,'${id}'))`;
    queryParams.type = 'app';

    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/runs`,
      queryParams
    );

    return fetch(url)
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(makeAPIError(error, `error fetching system log with id: ${id}`));
      });
  });
};
export const fetchSystemLogDetails = (
  tenantID: string,
  queryParams: Record<string, string>,
  entryID: string
): Promise<PaginatedResult<SystemLogEntryRecord>> => {
  return new Promise(async (resolve, reject) => {
    if (!queryParams.version) {
      queryParams.version = PAGINATION_API_VERSION;
    }

    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/runs/${encodeURIComponent(
        entryID
      )}`,
      queryParams
    );

    return fetch(url)
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(
          makeAPIError(
            error,
            `error fetching system log records (tenant uuid: ${tenantID})`
          )
        );
      });
  });
};
