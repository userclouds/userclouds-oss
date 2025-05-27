import {
  HTTPError,
  extractErrorMessage,
  makeAPIError,
} from '@userclouds/sharedui';

import { makeCompanyConfigURL, PAGINATION_API_VERSION } from '../API';

import Accessor, {
  AccessorSavePayload,
  ExecuteAccessorResponse,
} from '../models/Accessor';
import PaginatedResult from '../models/PaginatedResult';

export const executeTenantAccessor = async (
  tenantID: string,
  accessor_id: string,
  context: Record<string, any>,
  selector_values: string[],
  pageSize: number
): Promise<ExecuteAccessorResponse> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/executeaccessor?limit=${pageSize || '100'}`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({ accessor_id, context, selector_values }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'error executing accessor'));
      });
  });
};

export const fetchTenantAccessors = async (
  tenantID: string,
  searchParams: Record<string, string>
): Promise<PaginatedResult<Accessor>> => {
  return new Promise((resolve, reject) => {
    if (!searchParams.version) {
      searchParams.version = PAGINATION_API_VERSION;
    }

    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/accessors`,
      searchParams
    );
    return fetch(url)
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'error fetching user policy permissions'));
      });
  });
};

export const fetchTenantAccessor = async (
  tenantID: string,
  accessorID: string,
  version?: string
): Promise<Accessor> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/accessors/${encodeURIComponent(accessorID)}${
        version ? '?version=' + encodeURIComponent(version) : ''
      }`
    );
    return fetch(url)
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'error fetching accessor'));
      });
  });
};

export const createTenantAccessor = async (
  tenantID: string,
  accessor: AccessorSavePayload
): Promise<Accessor> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/accessors`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify(accessor),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'error creating accessor'));
      });
  });
};

export const updateTenantAccessor = async (
  tenantID: string,
  accessor: AccessorSavePayload
): Promise<Accessor> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/accessors/${encodeURIComponent(accessor.id)}`
    );
    return fetch(url, {
      method: 'PUT',
      body: JSON.stringify(accessor),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'error updating accessor'));
      });
  });
};

export const deleteTenantAccessor = async (
  tenantID: string,
  accessorID: string
): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/accessors/${encodeURIComponent(accessorID)}`
    );
    return fetch(url, {
      method: 'DELETE',
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve();
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'error deleting accessor'));
      });
  });
};
