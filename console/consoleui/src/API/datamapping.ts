import {
  HTTPError,
  extractErrorMessage,
  makeAPIError,
} from '@userclouds/sharedui';
import DataSource, { DataSourceElement } from '../models/DataSource';

import { makeCompanyConfigURL, PAGINATION_API_VERSION } from '../API';
import PaginatedResult from '../models/PaginatedResult';

export const fetchTenantDataSources = (
  tenantID: string,
  queryParams: Record<string, string>
): Promise<PaginatedResult<DataSource>> => {
  return new Promise((resolve, reject) => {
    if (!queryParams.version) {
      queryParams.version = PAGINATION_API_VERSION;
    }
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/datamapping/datasources`,
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
        reject(makeAPIError(e, 'error fetching data sources'));
      });
  });
};

export const fetchTenantDataSource = (
  tenantID: string,
  dataSourceID: string
): Promise<DataSource> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/datamapping/datasources/${encodeURIComponent(dataSourceID)}`
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
        reject(makeAPIError(e, 'error fetching data source'));
      });
  });
};

export const createTenantDataSource = (
  tenantID: string,
  dataSource: DataSource
): Promise<DataSource> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/datamapping/datasources`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({ datasource: dataSource }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'error creating data source'));
      });
  });
};

export const updateTenantDataSource = (
  tenantID: string,
  dataSource: DataSource
): Promise<DataSource> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/datamapping/datasources/${encodeURIComponent(dataSource.id)}`
    );
    return fetch(url, {
      method: 'PUT',
      body: JSON.stringify({ datasource: dataSource }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'error updating data source'));
      });
  });
};

export const deleteTenantDataSource = (
  tenantID: string,
  dataSourceID: string
): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/datamapping/datasources/${encodeURIComponent(dataSourceID)}`
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
        reject(makeAPIError(e, 'error deleting data source'));
      });
  });
};

export const fetchTenantDataSourceElements = (
  tenantID: string,
  queryParams: Record<string, string>
): Promise<PaginatedResult<DataSourceElement>> => {
  return new Promise((resolve, reject) => {
    if (!queryParams.version) {
      queryParams.version = PAGINATION_API_VERSION;
    }
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/datamapping/elements`,
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
        reject(makeAPIError(e, 'error fetching data sources'));
      });
  });
};

export const fetchTenantDataSourceElement = (
  tenantID: string,
  elementID: string
): Promise<DataSourceElement> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/datamapping/elements/${encodeURIComponent(elementID)}`
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
        reject(makeAPIError(e, 'error fetching data source'));
      });
  });
};

export const createTenantDataSourceElement = (
  tenantID: string,
  element: DataSourceElement
): Promise<DataSourceElement> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/datamapping/elements`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({ element: element }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'error creating data source'));
      });
  });
};

export const updateTenantDataSourceElement = async (
  tenantID: string,
  element: DataSourceElement
): Promise<DataSourceElement> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/datamapping/elements/${encodeURIComponent(element.id)}`
    );
    return fetch(url, {
      method: 'PUT',
      body: JSON.stringify({ element: element }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve(response.json());
      }, reject)
      .catch((e) => {
        reject(makeAPIError(e, 'error updating data source'));
      });
  });
};

export const deleteTenantDataSourceElement = (
  tenantID: string,
  elementID: string
): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/datamapping/elements/${encodeURIComponent(elementID)}`
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
        reject(makeAPIError(e, 'error deleting data source'));
      });
  });
};
