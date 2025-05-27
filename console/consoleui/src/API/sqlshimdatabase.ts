import {
  HTTPError,
  extractErrorMessage,
  makeAPIError,
} from '@userclouds/sharedui';
import { SqlshimDatabase } from '../models/SqlshimDatabase';
import { makeCompanyConfigURL } from '../API';
import PaginatedResult from '../models/PaginatedResult';

export const createTenantDatabase = async (
  tenantID: string,
  database: SqlshimDatabase
): Promise<SqlshimDatabase> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/databases`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify(database),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(makeAPIError(error, `error creating database`));
      });
  });
};

export const updateTenantDatabase = async (
  tenantID: string,
  database: SqlshimDatabase
): Promise<SqlshimDatabase> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/databases/${encodeURIComponent(database.id)}`
    );
    return fetch(url, {
      method: 'PUT',
      body: JSON.stringify(database),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(makeAPIError(error, `error updating database`));
      });
  });
};

export const deleteTenantDatabase = async (
  tenantID: string,
  databaseID: string
): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/databases/${encodeURIComponent(databaseID)}`
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
        reject(makeAPIError(error, `error updating database`));
      });
  });
};

export const getTenantDatabases = async (
  tenantID: string
): Promise<PaginatedResult<SqlshimDatabase>> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/databases`
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
        reject(makeAPIError(error, `error getting database`));
      });
  });
};

export const updateTenantDatabaseProxyPorts = async (
  tenantID: string
): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/update_database_proxy_ports`
    );
    return fetch(url, {
      method: 'POST',
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        resolve();
      })
      .catch((error) => {
        reject(makeAPIError(error, `error updating database proxy ports`));
      });
  });
};

export const testTenantDatabase = async (
  tenantID: string,
  database: SqlshimDatabase
): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/test_database`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({ database }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }
        response.json().then((data) => {
          if (!data.success) {
            reject(
              makeAPIError(
                new Error(data.error),
                `Unable to connect to database`
              )
            );
          }
          resolve();
        });
      })
      .catch((error) => {
        reject(makeAPIError(error, `Error`));
      });
  });
};
