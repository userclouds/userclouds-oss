import {
  HTTPError,
  extractErrorMessage,
  makeAPIError,
} from '@userclouds/sharedui';
import { PolicySecret } from '../models/PolicySecret';
import PaginatedResult from '../models/PaginatedResult';
import { makeCompanyConfigURL, PAGINATION_API_VERSION } from '../API';
import { MAX_LIMIT } from '../controls/PaginationHelper';

export const createTenantPolicySecret = async (
  tenantID: string,
  secret: PolicySecret
): Promise<PolicySecret> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/policies/secrets`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({
        secret,
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
        reject(makeAPIError(error, `error saving secret`));
      });
  });
};

export const listTenantPolicySecrets = async (
  tenantID: string,
  queryParams: Record<string, string>
): Promise<PaginatedResult<PolicySecret>> => {
  return new Promise((resolve, reject) => {
    if (!queryParams.version) {
      queryParams.version = PAGINATION_API_VERSION;
    }
    if (!queryParams.limit) {
      queryParams.limit = String(MAX_LIMIT);
    }
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/policies/secrets`,
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
        reject(makeAPIError(error, `error listing secrets`));
      });
  });
};

export const deleteTenantPolicySecret = async (
  tenantID: string,
  secretID: string
): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/policies/secrets/${secretID}`
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
        reject(makeAPIError(error, `error deleting secret`));
      });
  });
};
