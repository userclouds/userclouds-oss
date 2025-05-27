import {
  HTTPError,
  extractErrorMessage,
  makeAPIError,
} from '@userclouds/sharedui';

import { makeCompanyConfigURL } from '../API';
import Tenant, { SelectedTenant, TenantState } from '../models/Tenant';
import TenantURL from '../models/TenantURL';
import {
  createTenantSuccess,
  createTenantError,
  updateTenantCreationState,
} from '../actions/tenants';
import { redirect } from '../routing';
import { AppDispatch } from '../store';

export const fetchTenantsInCompany = (companyID: string): Promise<Tenant[]> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/companies/${encodeURIComponent(companyID)}/tenants`
    );
    return fetch(url)
      .then((response) => {
        if (!response.ok) {
          reject(new Error(response.statusText));
        }

        return response.json();
      }, reject)
      .then((json) => {
        resolve(json);
      }, reject);
  });
};

export const fetchTenantURLs = (tenantID: string): Promise<TenantURL[]> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/urls`
    );
    return fetch(url)
      .then((response) => {
        if (!response.ok) {
          reject(new Error(response.statusText));
        }

        return response.json();
      }, reject)
      .then((json) => {
        resolve(json);
      }, reject);
  });
};

export const createTenantURL = (tenantURL: TenantURL): Promise<TenantURL> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantURL.tenant_id)}/urls`
    );
    const req = {
      tenant_url: tenantURL,
    };
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify(req),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          reject(new Error(message));
        }

        return response.json();
      }, reject)
      .then((json) => {
        resolve(json.tenant_url);
      }, reject);
  });
};

export const updateTenantURL = (tenantURL: TenantURL): Promise<TenantURL> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantURL.tenant_id
      )}/urls/${encodeURIComponent(tenantURL.id)}`
    );
    const req = {
      tenant_url: tenantURL,
    };
    return fetch(url, {
      method: 'PUT',
      body: JSON.stringify(req),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          reject(new Error(message));
        }

        return response.json();
      }, reject)
      .then((json) => {
        resolve(json.tenant_url);
      }, reject);
  });
};

export const deleteTenantURL = (
  tenantId: string,
  tenantURLId: string
): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantId)}/urls/${encodeURIComponent(
        tenantURLId
      )}`
    );
    return fetch(url, {
      method: 'DELETE',
    })
      .then(
        (response) => {
          if (!response.ok) {
            reject(makeAPIError(null, 'Something went wrong.'));
          }
        },
        (e) => {
          reject(makeAPIError(e, 'Error deleting tenant URL'));
        }
      )
      .then(resolve, reject); // 204 no content on successful delete
  });
};

export const validateTenantURL = (tenantURL: TenantURL): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantURL.tenant_id
      )}/urls/actions/validate`
    );
    const req = {
      tenant_url_id: tenantURL.id,
    };
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify(req),
    })
      .then((response) => {
        if (!response.ok) {
          reject(new Error('Something went wrong.'));
        }
      }, reject)
      .then(() => {
        resolve();
      }, reject);
  });
};

export const createTenant = ({
  companyID,
  tenant,
  dispatch,
}: {
  companyID: string;
  tenant: Tenant;
  dispatch: AppDispatch;
}): Promise<SelectedTenant> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/companies/${encodeURIComponent(companyID)}/tenants`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({ tenant }),
    })
      .then(async (response) => {
        if (!response.ok) {
          reject(makeAPIError(null, 'Something went wrong.'));
        }
        return response.json();
      })
      .then(async (newTenant: SelectedTenant) => {
        let interval = 1000; // 1 second
        let totalTime = 300000; // 300 seconds
        dispatch(updateTenantCreationState(newTenant.state));

        while (newTenant.state === TenantState.CREATING) {
          // eslint-disable-next-line no-loop-func
          await new Promise((r) => setTimeout(r, interval));
          totalTime -= interval; // will retry for 180s total
          interval *= 1.2; // 1.2s 1.44s 1.728s 2.08s 2.5s... 23 retries in 5
          try {
            newTenant = await fetchTenant(newTenant.company_id, newTenant.id);
          } catch (e) {
            // ignore 404s since they probably mean we haven't created yet
            // TODO (sgarrity 6/23): there's an edge case here where we are stuck in this case forever
            if (e instanceof HTTPError) {
              const he = e as HTTPError;
              // NB: we include 403 here because it's possible the tenant exists but the authz
              // provisioning hasn't finished yet
              if (he.statusCode === 404 || he.statusCode === 403) {
                continue;
              } else {
                reject(makeAPIError(e, 'error creating tenant'));
              }
            }
          }

          if (newTenant.state === TenantState.FAILED_TO_PROVISION) {
            reject(makeAPIError(null, 'error creating tenant'));
          }

          if (totalTime < 0) {
            reject(
              makeAPIError(null, 'tenant create timed out after 180 seconds')
            );
          }
        }

        dispatch(updateTenantCreationState(newTenant.state));
        dispatch(createTenantSuccess(newTenant));
        redirect(
          `/?company_id=${newTenant.company_id}&tenant_id=${newTenant.id}`
        );
      })
      .catch((e) => {
        createTenantError(e.message);
        reject(makeAPIError(e, 'error creating tenant'));
      });
  });
};

export const fetchTenant = (
  companyID: string,
  tenantID: string
): Promise<SelectedTenant> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/companies/${encodeURIComponent(
        companyID
      )}/tenants/${encodeURIComponent(tenantID)}`
    );
    return fetch(url).then((response) => {
      if (!response.ok) {
        reject(makeAPIError(null, 'Something went wrong.'));
      }
      resolve(response.json());
    }, reject);
  });
};

export const updateTenant = (
  companyID: string,
  tenant: SelectedTenant
): Promise<SelectedTenant> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/companies/${encodeURIComponent(
        companyID
      )}/tenants/${encodeURIComponent(tenant.id)}`
    );
    const req = {
      tenant,
    };
    return fetch(url, {
      method: 'PUT',
      body: JSON.stringify(req),
    }).then((response) => {
      if (!response.ok) {
        reject(new Error('Something went wrong.'));
      }

      resolve(response.json());
    }, reject);
  });
};

export const deleteTenant = (
  companyID: string,
  tenantID: string
): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/companies/${encodeURIComponent(
        companyID
      )}/tenants/${encodeURIComponent(tenantID)}`
    );
    return fetch(url, {
      method: 'DELETE',
    }).then((response) => {
      if (!response.ok) {
        reject(new Error(`Unknown error deleting tenant (uuid: ${tenantID})`));
      }
      resolve();
    }, reject);
  });
};
