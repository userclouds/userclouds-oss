import {
  HTTPError,
  extractErrorMessage,
  makeAPIError,
} from '@userclouds/sharedui';
import { makeCompanyConfigURL, PAGINATION_API_VERSION } from '../API';
import PaginatedResult from '../models/PaginatedResult';
import Organization from '../models/Organization';
import Region from '../models/Region';

export const fetchOrganizations = async (
  tenantID: string,
  queryParams: Record<string, string>
): Promise<PaginatedResult<Organization>> => {
  return new Promise((resolve, reject) => {
    if (!queryParams.version) {
      queryParams.version = PAGINATION_API_VERSION;
    }
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/organizations`,
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
        reject(makeAPIError(error, 'error fetching organizations'));
      });
  });
};

export const fetchOrganization = async (
  tenantID: string,
  orgID: string
): Promise<Organization> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/organizations/${encodeURIComponent(orgID)}`
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
        reject(makeAPIError(error, 'error fetching organization'));
      });
  });
};

export const createOrganization = async (
  tenantID: string,
  orgID: string,
  orgName: string,
  region: Region
): Promise<Organization> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/organizations`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({
        id: orgID,
        name: orgName,
        region: region,
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
        reject(makeAPIError(error, 'error creating organization'));
      });
  });
};

export const updateOrganization = async (
  tenantID: string,
  organization: Organization
): Promise<Organization> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/organizations/${encodeURIComponent(organization.id)}`
    );
    return fetch(url, {
      method: 'PUT',
      body: JSON.stringify(organization),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'error updating organization'));
      });
  });
};
