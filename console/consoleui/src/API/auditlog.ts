import {
  HTTPError,
  extractErrorMessage,
  makeAPIError,
} from '@userclouds/sharedui';
import PaginatedResult from '../models/PaginatedResult';
import { makeCompanyConfigURL, PAGINATION_API_VERSION } from '../API';
import { AuditLogEntry, DataAccessLogEntry } from '../models/AuditLogEntry';

export const fetchAuditLog = (
  tenantID: string,
  queryParams: Record<string, string>
): Promise<PaginatedResult<AuditLogEntry>> => {
  return new Promise(async (resolve, reject) => {
    if (!queryParams.version) {
      queryParams.version = PAGINATION_API_VERSION;
    }
    queryParams.sort_key = 'created,id';
    queryParams.sort_order = 'descending';
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/auditlog/entries`,
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
            `error fetching audit logs (tenant uuid: ${tenantID})`
          )
        );
      });
  });
};

export const fetchDataAccessLog = (
  tenantID: string,
  queryParams: Record<string, string>
): Promise<PaginatedResult<DataAccessLogEntry>> => {
  return new Promise(async (resolve, reject) => {
    if (!queryParams.version) {
      queryParams.version = PAGINATION_API_VERSION;
    }
    queryParams.sort_key = 'created,id';
    queryParams.sort_order = 'descending';
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/dataaccesslog/entries`,
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
            `error fetching data access logs (tenant uuid: ${tenantID})`
          )
        );
      });
  });
};

export const fetchDataAccessLogEntry = (
  tenantID: string,
  entryID: string
): Promise<DataAccessLogEntry> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/dataaccesslog/entries/${encodeURIComponent(entryID)}`
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
            `error fetching data access log entry (tenant uuid: ${tenantID}, entry uuid: ${entryID})`
          )
        );
      });
  });
};
