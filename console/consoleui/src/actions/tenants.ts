import { APIError, JSONValue } from '@userclouds/sharedui';
import Tenant, { SelectedTenant } from '../models/Tenant';
import TenantURL from '../models/TenantURL';

export const CREATE_TENANT_REQUEST = 'CREATE_TENANT_REQUEST';
export const createTenantRequest = () => ({
  type: CREATE_TENANT_REQUEST,
});

export const CREATE_TENANT_SUCCESS = 'CREATE_TENANT_SUCCESS';
export const createTenantSuccess = (tenant: Tenant) => ({
  type: CREATE_TENANT_SUCCESS,
  data: tenant,
});

export const CREATE_TENANT_ERROR = 'CREATE_TENANT_ERROR';
export const createTenantError = (error: string) => ({
  type: CREATE_TENANT_ERROR,
  data: error,
});

export const TOGGLE_EDIT_TENANT_MODE = 'TOGGLE_EDIT_TENANT_MODE';
export const toggleEditTenantMode = (editMode?: boolean) => ({
  type: TOGGLE_EDIT_TENANT_MODE,
  data: editMode,
});

export const MODIFY_TENANT_NAME = 'MODIFY_TENANT_NAME';
export const modifyTenantName = (val: string) => ({
  type: MODIFY_TENANT_NAME,
  data: val,
});

export const UPDATE_TENANT_REQUEST = 'UPDATE_TENANT_REQUEST';
export const updateTenantRequest = () => ({
  type: UPDATE_TENANT_REQUEST,
});
export const UPDATE_TENANT_SUCCESS = 'UPDATE_TENANT_SUCCESS';
export const updateTenantSuccess = (tenant: Tenant) => ({
  type: UPDATE_TENANT_SUCCESS,
  data: tenant,
});
export const UPDATE_TENANT_ERROR = 'UPDATE_TENANT_ERROR';
export const updateTenantError = (error: APIError) => ({
  type: UPDATE_TENANT_ERROR,
  data: error.message,
});

export const DELETE_TENANT_REQUEST = 'DELETE_TENANT_REQUEST';
export const deleteTenantRequest = (tenantID: string) => ({
  type: DELETE_TENANT_REQUEST,
  data: tenantID,
});
export const DELETE_TENANT_SUCCESS = 'DELETE_TENANT_SUCCESS';
export const deleteTenantSuccess = (tenantID: string) => ({
  type: DELETE_TENANT_SUCCESS,
  data: tenantID,
});
export const DELETE_TENANT_ERROR = 'DELETE_TENANT_ERROR';
export const deleteTenantError = (error: APIError) => ({
  type: DELETE_TENANT_ERROR,
  data: error.message,
});

export const SET_FETCHING_TENANT = 'SET_FETCHING_TENANT';
export const setFetchingTenant = (editMode?: boolean) => ({
  type: SET_FETCHING_TENANT,
  data: editMode,
});

export const GET_SELECTED_TENANT_REQUEST = 'GET_SELECTED_TENANT_REQUEST';
export const getSelectedTenantRequest = () => ({
  type: GET_SELECTED_TENANT_REQUEST,
});
export const GET_SELECTED_TENANT_SUCCESS = 'GET_SELECTED_TENANT_SUCCESS';
export const getSelectedTenantSuccess = (tenant: SelectedTenant) => ({
  type: GET_SELECTED_TENANT_SUCCESS,
  data: tenant,
});
export const GET_SELECTED_TENANT_ERROR = 'GET_SELECTED_TENANT_ERROR';
export const getSelectedTenantError = (error: APIError) => ({
  type: GET_SELECTED_TENANT_ERROR,
  data: error.message,
});

export const GET_TENANTS_FOR_COMPANY_REQUEST =
  'GET_TENANTS_FOR_COMPANY_REQUEST';
export const getTenantsForCompanyRequest = () => ({
  type: GET_TENANTS_FOR_COMPANY_REQUEST,
});
export const GET_TENANTS_FOR_COMPANY_SUCCESS =
  'GET_TENANTS_FOR_COMPANY_SUCCESS';
export const getTenantsForCompanySuccess = (
  companyID: string,
  tenants: Tenant[]
) => ({
  type: GET_TENANTS_FOR_COMPANY_SUCCESS,
  data: [companyID, tenants],
});
export const GET_TENANTS_FOR_COMPANY_ERROR = 'GET_TENANTS_FOR_COMPANY_ERROR';
export const getTenantsForCompanyError = (error: APIError) => ({
  type: GET_TENANTS_FOR_COMPANY_ERROR,
  data: error.message,
});

export const TOGGLE_CREATE_TENANT_DIALOG = 'TOGGLE_CREATE_TENANT_DIALOG';
export const toggleCreateTenantDialog = (isOpen?: boolean) => ({
  type: TOGGLE_CREATE_TENANT_DIALOG,
  data: isOpen,
});

export const CREATE_TENANT_URL_REQUEST = 'CREATE_TENANT_URL_REQUEST';
export const createTenantURLRequest = () => ({
  type: CREATE_TENANT_URL_REQUEST,
});
export const CREATE_TENANT_URL_SUCCESS = 'CREATE_TENANT_URL_SUCCESS';
export const createTenantURLSuccess = (url: TenantURL) => ({
  type: CREATE_TENANT_URL_SUCCESS,
  data: url,
});
export const CREATE_TENANT_URL_ERROR = 'CREATE_TENANT_URL_ERROR';
export const createTenantURLError = (error: APIError) => ({
  type: CREATE_TENANT_URL_ERROR,
  data: error.message,
});

export const SET_CURRENT_URL = 'SET_CURRENT_URL';
export const setCurrentURL = (url: TenantURL) => ({
  type: SET_CURRENT_URL,
  data: url,
});

export const UPDATE_TENANT_URL_REQUEST = 'UPDATE_TENANT_URL_REQUEST';
export const updateTenantURLRequest = () => ({
  type: UPDATE_TENANT_URL_REQUEST,
});
export const UPDATE_TENANT_URL_SUCCESS = 'UPDATE_TENANT_URL_SUCCESS';
export const updateTenantURLSuccess = (url: TenantURL) => ({
  type: UPDATE_TENANT_URL_SUCCESS,
  data: url,
});
export const UPDATE_TENANT_URL_ERROR = 'UPDATE_TENANT_URL_ERROR';
export const updateTenantURLError = (error: APIError) => ({
  type: UPDATE_TENANT_URL_ERROR,
  data: error.message,
});

export const GET_TENANT_URLS_REQUEST = 'GET_TENANT_URLS_REQUEST';
export const getTenantURLsRequest = () => ({
  type: GET_TENANT_URLS_REQUEST,
});
export const GET_TENANT_URLS_SUCCESS = 'GET_TENANT_URLS_SUCCESS';
export const getTenantURLsSuccess = (urls: TenantURL[]) => ({
  type: GET_TENANT_URLS_SUCCESS,
  data: urls,
});
export const GET_TENANT_URLS_ERROR = 'GET_TENANT_URLS_ERROR';
export const getTenantURLsError = (error: APIError) => ({
  type: GET_TENANT_URLS_ERROR,
  data: error.message,
});

export const MODIFY_TENANT_URL = 'MODIFY_TENANT_URL';
export const modifyTenantURL = (data: Record<string, JSONValue>) => ({
  type: MODIFY_TENANT_URL,
  data,
});

export const ADD_TENANT_URL = 'ADD_TENANT_URL';
export const addTenantURL = () => ({
  type: ADD_TENANT_URL,
});

export const SET_CREATING_URL = 'SET_CREATING_URL';
export const setCreatingUrl = (creating: boolean) => ({
  type: SET_CREATING_URL,
  data: creating,
});

export const UPDATE_TENANT_URL = 'UPDATE_TENANT_URL';
export const updateTenantURL = (url: TenantURL) => ({
  type: UPDATE_TENANT_URL,
  data: url,
});

export const CREATE_TENANT_URL = 'CREATE_TENANT_URL';
export const createTenantURL = (url: TenantURL) => ({
  type: CREATE_TENANT_URL,
  data: url,
});

export const DELETE_TENANT_URL = 'DELETE_TENANT_URL';
export const deleteTenantURL = (id: string) => ({
  type: DELETE_TENANT_URL,
  data: id,
});

export const DELETE_TENANT_URL_REQUEST = 'DELETE_TENANT_URL_REQUEST';
export const deleteTenantURLRequest = () => ({
  type: DELETE_TENANT_URL_REQUEST,
});
export const DELETE_TENANT_URL_SUCCESS = 'DELETE_TENANT_URL_SUCCESS';
export const deleteTenantURLSuccess = (val: string) => ({
  type: DELETE_TENANT_URL_SUCCESS,
  data: val,
});
export const DELETE_TENANT_URL_ERROR = 'DELETE_TENANT_URL_ERROR';
export const deleteTenantURLError = (error: APIError) => ({
  type: DELETE_TENANT_URL_ERROR,
  data: error.message,
});

export const TOGGLE_TENANT_URL_EDIT_MODE = 'TOGGLE_TENANT_URL_EDIT_MODE';
export const toggleTenantURLEditMode = (editMode?: boolean) => ({
  type: TOGGLE_TENANT_URL_EDIT_MODE,
  data: editMode,
});

export const SET_CREATING_ISSUER = 'SET_CREATING_ISSUER';
export const setCreateIssuer = (value: boolean) => ({
  type: SET_CREATING_ISSUER,
  data: value,
});

export const SET_EDITING_ISSUER_INDEX = 'SET_EDITING_ISSUER_INDEX';
export const setEditingIssuer = (index: number) => ({
  type: SET_EDITING_ISSUER_INDEX,
  data: index,
});

export const SET_TENANT_PROVIDER_NAME = 'SET_TENANT_PROVIDER_NAME';
export const setTenantProviderName = (value: string) => ({
  type: SET_TENANT_PROVIDER_NAME,
  data: value,
});

export const TOGGLE_TENANT_URL_DIALOG_IS_OPEN =
  'TOGGLE_TENANT_URL_DIALOG_IS_OPEN';
export const toggleTenantURLDialogIsOpen = (isOpen?: boolean) => ({
  type: TOGGLE_TENANT_URL_DIALOG_IS_OPEN,
  data: isOpen,
});

export const TOGGLE_TENANT_DATABASE_DIALOG_IS_OPEN =
  'TOGGLE_TENANT_DATABASE_DIALOG_IS_OPEN';
export const toggleTenantDatabseDialogIsOpen = (isOpen?: boolean) => ({
  type: TOGGLE_TENANT_DATABASE_DIALOG_IS_OPEN,
  data: isOpen,
});
export const TOGGLE_TENANT_ISSUER_DIALOG_IS_OPEN =
  'TOGGLE_TENANT_ISSUER_DIALOG_IS_OPEN';
export const toggleTenantIssuerDialogIsOpen = (isOpen?: boolean) => ({
  type: TOGGLE_TENANT_ISSUER_DIALOG_IS_OPEN,
  data: isOpen,
});

export const UPDATE_TENANT_CREATION_STATE = 'UPDATE_TENANT_CREATION_STATE';
export const updateTenantCreationState = (state: string) => ({
  type: UPDATE_TENANT_CREATION_STATE,
  data: state,
});
