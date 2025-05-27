import {
  HTTPError,
  extractErrorMessage,
  makeAPIError,
} from '@userclouds/sharedui';
import LoginApp from '../models/LoginApp';
import { makeCompanyConfigURL } from '../API';

export const fetchLoginApps = async (
  tenantID: string,
  orgID?: string
): Promise<LoginApp[]> => {
  return new Promise((resolve, reject) => {
    const params: Record<string, string> = {};
    if (orgID) {
      params.organization_id = orgID;
    }
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/loginapps`,
      params
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
