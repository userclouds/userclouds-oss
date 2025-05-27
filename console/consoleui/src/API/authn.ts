import {
  APIError,
  HTTPError,
  extractErrorMessage,
  tryGetJSON,
  makeAPIError,
} from '@userclouds/sharedui';

import { makeCompanyConfigURL } from '../API';
import TenantPlexConfig, { ensureDefaults } from '../models/TenantPlexConfig';
import {
  TenantAppMessageElements,
  EmailMessageElementsSavePayload,
  SMSMessageElementsSavePayload,
} from '../models/MessageElements';
import {
  PageParametersResponse,
  PageParametersSavePayload,
  ImageUploadResponse,
} from '../models/PageParameters';
import { SAMLIDP } from '../models/SAMLIDP';
import { OIDCProvider } from '../models/OIDCProvider';

export const createOIDCProvider = async (
  tenantID: string,
  oidcProvider: OIDCProvider
): Promise<OIDCProvider> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      '/api/tenants/' + tenantID + '/oidcproviders/create'
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({ oidc_provider: oidcProvider }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'Error creating OIDC provider'));
      });
  });
};

export const deleteOIDCProvider = async (
  tenantID: string,
  oidcProviderName: string
): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      '/api/tenants/' + tenantID + '/oidcproviders/delete'
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({ oidc_provider_name: oidcProviderName }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve();
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'Error deleting OIDC provider'));
      });
  });
};

export const forceFetchTenantPlexConfig = async (
  tenantID: string
): Promise<TenantPlexConfig> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/plexconfig`
    );
    return fetch(url)
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        return response.json();
      }, reject)
      .then((json) => resolve(ensureDefaults(json as TenantPlexConfig)), reject)
      .catch((e) => {
        reject(makeAPIError(e, 'Error fetching tenant plex config'));
      });
  });
};

export const saveTenantPlexConfig = async (
  tenantID: string,
  tenantConfig: TenantPlexConfig
): Promise<TenantPlexConfig> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/plexconfig`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify(tenantConfig),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        return response.json();
      }, reject)
      .then((json: TenantPlexConfig) => {
        resolve(ensureDefaults(json));
      }, reject)
      .catch((e: Error) => {
        if (e instanceof APIError) {
          reject(e);
        } else {
          reject(makeAPIError(e, 'Error fetching tenant plex config'));
        }
      });
  });
};

export const addLoginAppToTenant = async (
  tenantID: string,
  appID: string,
  name: string,
  clientID: string,
  clientSecret: string
): Promise<TenantPlexConfig> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/loginapps`
    );
    const req = {
      app_id: appID,
      name: name,
      client_id: clientID,
      client_secret: clientSecret,
    };
    try {
      const rawResponse = await fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      });
      const jsonResponse = await tryGetJSON(rawResponse);
      resolve(
        ensureDefaults({
          tenant_config: jsonResponse,
        } as unknown as TenantPlexConfig)
      );
    } catch (e) {
      reject(
        makeAPIError(
          e,
          `error adding login app to tenant (tenant uuid: ${tenantID}, app id: ${appID})`
        )
      );
    }
  });
};

export const deleteLoginAppFromTenant = async (
  tenantID: string,
  appID: string
): Promise<TenantPlexConfig> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/loginapps/${encodeURIComponent(appID)}`
    );
    try {
      const rawResponse = await fetch(url, {
        method: 'DELETE',
      });
      const jsonResponse = await tryGetJSON(rawResponse);
      resolve(
        ensureDefaults({
          tenant_config: jsonResponse,
        } as unknown as TenantPlexConfig)
      );
    } catch (e) {
      reject(
        makeAPIError(
          e,
          `error deleting login app from tenant (tenant uuid: ${tenantID}, app id: ${appID})`
        )
      );
    }
  });
};

export const enableSAMLIDPForLoginApp = async (
  tenantID: string,
  appID: string
): Promise<SAMLIDP> => {
  return new Promise(async (resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/loginapps/actions/samlidp?app_id=${encodeURIComponent(appID)}`
    );
    try {
      const rawResponse = await fetch(url, {
        method: 'POST',
      });
      const jsonResponse = await tryGetJSON(rawResponse);
      const samlIDP = jsonResponse as SAMLIDP;
      resolve(samlIDP);
    } catch (e) {
      reject(
        makeAPIError(
          e,
          `error adding SAML IDP to login app (tenant uuid: ${tenantID}, app id: ${appID})`
        )
      );
    }
  });
};

export const saveTenantEmailMessageElements = async (
  payload: EmailMessageElementsSavePayload
): Promise<TenantAppMessageElements> => {
  const tenantID = payload.modified_message_type_message_elements.tenant_id;
  return new Promise((resolve, reject) => {
    return fetch(`/api/tenants/${encodeURIComponent(tenantID)}/emailelements`, {
      body: JSON.stringify(payload),
      method: 'POST',
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'Error saving tenant email elements'));
      });
  });
};

export const fetchTenantEmailMessageElements = async (
  tenantID: string
): Promise<TenantAppMessageElements> => {
  return new Promise((resolve, reject) => {
    return fetch(`/api/tenants/${encodeURIComponent(tenantID)}/emailelements`, {
      method: 'GET',
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'Error fetching tenant email elements'));
      });
  });
};

export const saveTenantSMSMessageElements = async (
  payload: SMSMessageElementsSavePayload
): Promise<TenantAppMessageElements> => {
  const tenantID = payload.modified_message_type_message_elements.tenant_id;
  return new Promise((resolve, reject) => {
    return fetch(`/api/tenants/${encodeURIComponent(tenantID)}/smselements`, {
      body: JSON.stringify(payload),
      method: 'POST',
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        return response.json();
      }, reject)
      .then((json) => {
        if ('tenant_app_message_elements' in json) {
          resolve(json);
        }
        reject();
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'Error saving tenant sms elements'));
      });
  });
};

export const fetchTenantSMSMessageElements = async (
  tenantID: string
): Promise<TenantAppMessageElements> => {
  return new Promise((resolve, reject) => {
    return fetch(`/api/tenants/${encodeURIComponent(tenantID)}/smselements`, {
      method: 'GET',
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'Error fetching tenant sms elements'));
      });
  });
};

export const fetchAppPageParameters = async (
  tenantID: string,
  appID: string
): Promise<PageParametersResponse> => {
  return new Promise((resolve, reject) => {
    return fetch(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/apppageparameters/${encodeURIComponent(appID)}`,
      {
        method: 'GET',
      }
    )
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'Error fetching page parameters'));
      });
  });
};

export const saveAppPageParameters = async (
  tenantID: string,
  appID: string,
  payload: PageParametersSavePayload
): Promise<PageParametersResponse> => {
  return new Promise((resolve, reject) => {
    return fetch(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/apppageparameters/${encodeURIComponent(appID)}`,
      {
        body: JSON.stringify(payload),
        method: 'PUT',
      }
    )
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'Error saving page parameters'));
      });
  });
};

export const uploadImageForApp = async (
  tenantID: string,
  appID: string,
  image: File
): Promise<ImageUploadResponse> => {
  const data = new FormData();
  data.append('image', image);
  return new Promise((resolve, reject) => {
    return fetch(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/uploadlogo?app_id=${encodeURIComponent(appID)}`,
      {
        body: data,
        method: 'POST',
      }
    )
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'error uploading image'));
      });
  });
};
