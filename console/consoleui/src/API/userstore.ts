import {
  HTTPError,
  extractErrorMessage,
  makeAPIError,
} from '@userclouds/sharedui';

import { Column } from '../models/TenantUserStoreConfig';
import AccessPolicy from '../models/AccessPolicy';
import { makeCompanyConfigURL, PAGINATION_API_VERSION } from '../API';
import PaginatedResult from '../models/PaginatedResult';
import { MAX_LIMIT } from '../controls/PaginationHelper';
import {
  PurposeRetentionDuration,
  ColumnRetentionDurationsResponse,
  DurationType,
} from '../models/ColumnRetentionDurations';
import { DataType } from '../models/DataType';

export const createUserStoreColumn = async (
  tenantID: string,
  column: Column,
  composed_access_policy?: AccessPolicy,
  composed_token_access_policy?: AccessPolicy
): Promise<Column> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/columns`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({
        column,
        composed_access_policy,
        composed_token_access_policy,
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
        reject(makeAPIError(error, `error adding column to user store`));
      });
  });
};

export const fetchTenantUserStoreColumns = async (
  tenantID: string,
  queryParams: Record<string, string>
): Promise<PaginatedResult<Column>> => {
  return new Promise((resolve, reject) => {
    if (!queryParams.version) {
      queryParams.version = PAGINATION_API_VERSION;
    }
    if (!queryParams.limit) {
      queryParams.limit = String(MAX_LIMIT);
    }
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/columns`,
      queryParams
    );
    return fetch(url, {
      method: 'GET',
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(makeAPIError(error, `error fetching columns from user store`));
      });
  });
};

export const fetchUserStoreColumn = async (
  tenantID: string,
  columnID: string
): Promise<Column> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/columns/${columnID}`
    );
    return fetch(url, {
      method: 'GET',
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(makeAPIError(error, `error fetching column from user store`));
      });
  });
};

export const updateUserStoreColumn = async (
  tenantID: string,
  column: Column,
  composed_access_policy?: AccessPolicy,
  composed_token_access_policy?: AccessPolicy
): Promise<Column> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/columns/${encodeURIComponent(column.id)}`
    );
    return fetch(url, {
      method: 'PUT',
      body: JSON.stringify({
        column,
        composed_access_policy,
        composed_token_access_policy,
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
        reject(makeAPIError(error, `error editing user store column`));
      });
  });
};

export const deleteUserStoreColumn = async (
  tenantID: string,
  id: string
): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/columns/${encodeURIComponent(id)}`
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
        reject(makeAPIError(error));
      });
  });
};

export const fetchUserStoreColumnRetentionDurations = async (
  tenantID: string,
  columnID: string,
  durationType: DurationType
): Promise<ColumnRetentionDurationsResponse> => {
  return new Promise((resolve, reject) => {
    const req = {
      column_id: columnID,
      duration_type: durationType,
    };
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/columns/retentiondurations/actions/get`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify(req),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(
          makeAPIError(
            error,
            `error retrieving column retention durations from user store`
          )
        );
      });
  });
};

export const updateUserStoreColumnRetentionDurations = async (
  tenantID: string,
  columnID: string,
  durationType: DurationType,
  purposeRetentionDurations: PurposeRetentionDuration[]
): Promise<ColumnRetentionDurationsResponse> => {
  return new Promise((resolve, reject) => {
    const req = {
      column_id: columnID,
      duration_type: durationType,
      purpose_retention_durations: purposeRetentionDurations,
    };
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/columns/retentiondurations/actions/update`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify(req),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(
          makeAPIError(
            error,
            `error update column retention durations in user store`
          )
        );
      });
  });
};

export const createUserStoreDataType = async (
  tenantID: string,
  dataType: DataType
): Promise<DataType> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/datatypes`
    );
    return fetch(url, {
      method: 'POST',
      body: JSON.stringify({ data_type: dataType }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(makeAPIError(error, `error adding dataType to user store`));
      });
  });
};

export const fetchTenantUserStoreDataTypes = async (
  tenantID: string,
  queryParams: Record<string, string>
): Promise<PaginatedResult<DataType>> => {
  return new Promise((resolve, reject) => {
    if (!queryParams.version) {
      queryParams.version = PAGINATION_API_VERSION;
    }
    if (!queryParams.limit) {
      queryParams.limit = String(MAX_LIMIT);
    }
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(tenantID)}/userstore/datatypes`,
      queryParams
    );
    return fetch(url, {
      method: 'GET',
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(
          makeAPIError(error, `error fetching data types from user store`)
        );
      });
  });
};

export const fetchUserStoreDataType = async (
  tenantID: string,
  dataTypeID: string
): Promise<DataType> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/datatypes/${dataTypeID}`
    );
    return fetch(url, {
      method: 'GET',
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(makeAPIError(error, `error fetching data type from user store`));
      });
  });
};

export const updateUserStoreDataType = async (
  tenantID: string,
  dataType: DataType
): Promise<DataType> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/datatypes/${encodeURIComponent(dataType.id)}`
    );
    return fetch(url, {
      method: 'PUT',
      body: JSON.stringify({ data_type: dataType }),
    })
      .then(async (response) => {
        if (!response.ok) {
          const message = await extractErrorMessage(response);
          throw new HTTPError(message, response.status);
        }

        resolve(response.json());
      })
      .catch((error) => {
        reject(makeAPIError(error, `error editing user store data type`));
      });
  });
};

export const deleteUserStoreDataType = async (
  tenantID: string,
  id: string
): Promise<void> => {
  return new Promise((resolve, reject) => {
    const url = makeCompanyConfigURL(
      `/api/tenants/${encodeURIComponent(
        tenantID
      )}/userstore/datatypes/${encodeURIComponent(id)}`
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
        reject(makeAPIError(error, `error deleting data type from user store`));
      });
  });
};
