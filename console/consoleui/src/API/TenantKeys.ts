import {
  HTTPError,
  makeAPIError,
  extractErrorMessage,
} from '@userclouds/sharedui';
import { makeCompanyConfigURL } from '../API';

export const fetchTenantPublicKeys = (tenantID: string): Promise<string[]> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/keys`
    );
    return fetch(url)
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        return response.json();
      }, reject)
      .then(
        (data: { public_keys: string[] }) => {
          resolve(data.public_keys);
        },
        (e) => {
          reject(
            makeAPIError(
              e,
              `error fetching public keys (tenant uuid: ${tenantID})`
            )
          );
        }
      );
  });
};

export const rotateTenantKeys = (tenantID: string): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/keys/actions/rotate`
    );
    return fetch(url, { method: 'PUT' })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve();
      }, reject)
      .catch((e) => {
        reject(
          makeAPIError(
            e,
            `error fetching public keys (tenant uuid: ${tenantID})`
          )
        );
      });
  });
};
