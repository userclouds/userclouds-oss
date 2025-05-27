import {
  HTTPError,
  extractErrorMessage,
  makeAPIError,
} from '@userclouds/sharedui';

import PaginatedResult from '../models/PaginatedResult';
import Purpose from '../models/Purpose';
import { makeCompanyConfigURL, PAGINATION_API_VERSION } from '../API';

export const fetchTenantPurposes = async (
  tenantID: string,
  queryParams: Record<string, string>
): Promise<PaginatedResult<Purpose>> => {
  return new Promise((resolve, reject) => {
    if (!queryParams.version) {
      queryParams.version = PAGINATION_API_VERSION;
    }
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/purposes`,
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
        // resolve({ data: [], has_next: false, next: '', has_prev: false, prev: '' });
        reject(makeAPIError(error, `error fetching purposes`));
      });
  });
};

export const fetchTenantPurpose = async (
  tenantID: string,
  purposeID: string
): Promise<Purpose> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/purposes/${encodeURIComponent(purposeID)}`
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
        reject(makeAPIError(error, `error fetching purpose`));
      });
  });
};

export const createTenantPurpose = async (
  tenantID: string,
  purpose: Purpose
): Promise<Purpose> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/purposes`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({ purpose }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(makeAPIError(error, `error creating purpose`));
      });
  });
};

export const updateTenantPurpose = async (
  tenantID: string,
  purposeID: string,
  purpose: Purpose
): Promise<Purpose> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/purposes/${encodeURIComponent(purposeID)}`
    );
    return fetch(url, {
      method: 'PUT',
      body: JSON.stringify({ purpose }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(makeAPIError(error, `error updating purpose`));
      });
  });
};

export const deleteTenantPurpose = async (
  tenantID: string,
  purposeID: string
): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/purposes/${encodeURIComponent(purposeID)}`
    );
    return fetch(url, { method: 'DELETE' })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve();
      })
      .catch((error) => {
        reject(makeAPIError(error, `error deleting purpose ${purposeID}`));
      });
  });
};
