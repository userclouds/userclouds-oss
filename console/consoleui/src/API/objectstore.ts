import {
  HTTPError,
  extractErrorMessage,
  makeAPIError,
} from '@userclouds/sharedui';
import AccessPolicy from '../models/AccessPolicy';
import { ObjectStore } from '../models/ObjectStore';
import PaginatedResult from '../models/PaginatedResult';
import { makeCompanyConfigURL, PAGINATION_API_VERSION } from '../API';
import { MAX_LIMIT } from '../controls/PaginationHelper';

export const createTenantObjectStore = async (
  tenantID: string,
  object_store: ObjectStore,
  composed_access_policy?: AccessPolicy
): Promise<ObjectStore> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/objectstores`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({
        object_store,
        composed_access_policy,
      }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(makeAPIError(error, `error saving object store`));
      });
  });
};

export const updateTenantObjectStore = async (
  tenantID: string,
  object_store: ObjectStore,
  composed_access_policy?: AccessPolicy
): Promise<ObjectStore> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/objectstores/${
        object_store.id
      }`
    );
    return fetch(url, {
      method: 'PUT',
      body: JSON.stringify({
        object_store,
        composed_access_policy,
      }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(makeAPIError(error, `error saving object store`));
      });
  });
};

export const getTenantObjectStore = async (
  tenantID: string,
  objectStoreID: string
): Promise<ObjectStore> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/objectstores/${objectStoreID}`
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
        reject(makeAPIError(error, `error getting object store`));
      });
  });
};

export const listTenantObjectStores = async (
  tenantID: string,
  queryParams: Record<string, string>
): Promise<PaginatedResult<ObjectStore>> => {
  return new Promise((resolve, reject) => {
    if (!queryParams.version) {
      queryParams.version = PAGINATION_API_VERSION;
    }
    if (!queryParams.limit) {
      queryParams.limit = String(MAX_LIMIT);
    }
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/objectstores`,
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
        reject(makeAPIError(error, `error listing object stores`));
      });
  });
};

export const deleteTenantObjectStore = async (
  tenantID: string,
  objectStoreID: string
): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/objectstores/${objectStoreID}`
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
      })
      .catch((error) => {
        reject(makeAPIError(error, `error deleting object store`));
      });
  });
};
