import {
  HTTPError,
  extractErrorMessage,
  makeAPIError,
} from '@userclouds/sharedui';

import { UserProfileSerialized, UserEvent } from '../models/UserProfile';
import { makeCompanyConfigURL } from '../API';

export const fetchTenantUser = (
  tenantID: string,
  userID: string
): Promise<UserProfileSerialized> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/users/${encodeURIComponent(
        userID
      )}`
    );
    return fetch(url)
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      })
      .catch((e) => {
        reject(
          makeAPIError(
            e,
            `error fetching user (tenant uuid: ${tenantID}, user uuid: ${userID})`
          )
        );
      });
  });
};

export const fetchTenantUserConsentedPurposes = (
  tenantID: string,
  userID: string
): Promise<Array<object>> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/consentedpurposesforuser/${encodeURIComponent(userID)}`
    );
    return fetch(url)
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      })
      .catch((e) => {
        reject(
          makeAPIError(
            e,
            `error fetching consented purposes (tenant uuid: ${tenantID}, user uuid: ${userID})`
          )
        );
      });
  });
};

export const fetchTenantUserEvents = (
  tenantID: string,
  userID: string
): Promise<UserEvent[]> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userevents`,
      {
        user_alias: userID,
      }
    );
    return fetch(url)
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      })
      .catch((e) => {
        reject(
          makeAPIError(
            e,
            `error fetching user events (tenant uuid: ${tenantID}, user uuid: ${userID})`
          )
        );
      });
  });
};
