import { APIError } from '@userclouds/sharedui';

export const GET_TENANT_KEYS_REQUEST = 'GET_TENANT_KEYS_REQUEST';
export const GET_TENANT_KEYS_SUCCESS = 'GET_TENANT_KEYS_SUCCESS';
export const GET_TENANT_KEYS_ERROR = 'GET_TENANT_KEYS_ERROR';
export const ROTATE_TENANT_KEYS_REQUEST = 'ROTATE_TENANT_KEYS_REQUEST';
export const ROTATE_TENANT_KEYS_SUCCESS = 'ROTATE_TENANT_KEYS_SUCCESS';
export const ROTATE_TENANT_KEYS_ERROR = 'ROTATE_TENANT_KEYS_ERROR';

export const getTenantKeysRequest = () => ({
  type: GET_TENANT_KEYS_REQUEST,
});

export const getTenantKeysSuccess = (keys: string[]) => ({
  type: GET_TENANT_KEYS_SUCCESS,
  data: keys,
});

export const getTenantKeysError = (error: APIError) => ({
  type: GET_TENANT_KEYS_ERROR,
  data: error.message,
});

export const rotateTenantKeysRequest = () => ({
  type: ROTATE_TENANT_KEYS_REQUEST,
});

export const rotateTenantKeysSuccess = () => ({
  type: ROTATE_TENANT_KEYS_SUCCESS,
});

export const rotateTenantKeysError = (error: APIError) => ({
  type: ROTATE_TENANT_KEYS_ERROR,
  data: error.message,
});
