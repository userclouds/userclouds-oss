import { makeAPIError, HTTPError } from '@userclouds/sharedui';
import { getEnvData } from '../models/EnvData';

const { Universe: env, StatsSigAPIKey: key } = getEnvData();

export const fetchFeatureFlagsForUser = (
  userID: string,
  companyID: string,
  tenantID: string
): Promise<any> => {
  return new Promise((resolve, reject) => {
    const payload = {
      user: {
        userID,
        statsigEnvironment: {
          tier: env,
        },
        custom: {
          tenantID,
          companyID,
        },
      },
      hash: 'djb2',
    };
    return fetch(`https://featuregates.org/v1/initialize`, {
      body: JSON.stringify(payload), // encoding: window.btoa(postBody).split('').reverse().join('')
      headers: {
        'STATSIG-API-KEY': key,
        'STATSIG-ENCODED': '0',
      },
      method: 'POST',
    })
      .then((response) => {
        if (!response.ok) {
          // we probably don't want to expose that the error concerns
          // feature flags
          throw new HTTPError('something went wrong', response.status);
        }

        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e));
      });
  });
};
