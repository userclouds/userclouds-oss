import {
  HTTPError,
  extractErrorMessage,
  makeAPIError,
} from '@userclouds/sharedui';
import Mutator, { MutatorSavePayload, MutatorColumn } from '../models/Mutator';

import { makeCompanyConfigURL, PAGINATION_API_VERSION } from '../API';
import PaginatedResult from '../models/PaginatedResult';

export const fetchTenantMutators = (
  tenantID: string,
  queryParams: Record<string, string>
): Promise<PaginatedResult<Mutator>> => {
  return new Promise((resolve, reject) => {
    if (!queryParams.version) {
      queryParams.version = PAGINATION_API_VERSION;
    }
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/mutators`,
      queryParams
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

export const fetchTenantMutator = (
  tenantID: string,
  mutatorID: string,
  version?: string
): Promise<Mutator> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/mutators/${encodeURIComponent(mutatorID)}${
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
        reject(makeAPIError(e, 'error fetching mutator'));
      });
  });
};

export const createTenantMutator = (
  tenantID: string,
  mutator: MutatorSavePayload
): Promise<Mutator> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/mutators`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify(mutator),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'error creating mutator'));
      });
  });
};

export const updateTenantMutator = async (
  tenantID: string,
  mutator: MutatorSavePayload
): Promise<Mutator> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/mutators/${encodeURIComponent(mutator.id)}`
    );
    return fetch(url, {
      method: 'PUT',
      body: JSON.stringify(mutator),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'error updating mutator'));
      });
  });
};

export const updateTenantMutatorColumns = async (
  tenantID: string,
  mutatorID: string,
  columns: MutatorColumn[]
): Promise<Mutator> => {
  return new Promise((resolve, reject) => {
    const payload = columns.map((column: MutatorColumn) => ({
      column: {
        id: column.id,
      },
      normalizer: {
        id: column.normalizer_id,
      },
    }));
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/mutators/columns/${encodeURIComponent(mutatorID)}`
    );
    return fetch(url, {
      method: 'PUT',
      body: JSON.stringify(payload),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'error updating mutator'));
      });
  });
};

export const deleteTenantMutator = (
  tenantID: string,
  mutatorID: string
): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/mutators/${encodeURIComponent(mutatorID)}`
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
        reject(makeAPIError(e, 'error deleting mutator'));
      });
  });
};
