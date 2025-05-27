import { APIError, JSONValue } from '@userclouds/sharedui';
import { AppDispatch } from '../store';
import PaginatedResult from '../models/PaginatedResult';
import Organization from '../models/Organization';
import LoginApp from '../models/LoginApp';

export const GET_ORGANIZATIONS_REQUEST = 'GET_ORGANIZATIONS_REQUEST';
export const GET_ORGANIZATIONS_SUCCESS = 'GET_ORGANIZATIONS_SUCCESS';
export const GET_ORGANIZATIONS_ERROR = 'GET_ORGANIZATIONS_ERROR';
export const CREATE_ORGANIZATION_REQUEST = 'CREATE_ORGANIZATION_REQUEST';
export const CREATE_ORGANIZATION_SUCCESS = 'CREATE_ORGANIZATION_SUCCESS';
export const CREATE_ORGANIZATION_ERROR = 'CREATE_ORGANIZATION_ERROR';
export const GET_ORGANIZATION_REQUEST = 'GET_ORGANIZATION_REQUEST';
export const GET_ORGANIZATION_SUCCESS = 'GET_ORGANIZATION_SUCCESS';
export const GET_ORGANIZATION_ERROR = 'GET_ORGANIZATION_ERROR';
export const TOGGLE_ORGANIZATION_EDIT_MODE = 'TOGGLE_ORGANIZATION_EDIT_MODE';
export const MODIFY_ORGANIZATION = 'MODIFY_ORGANIZATION';
export const UPDATE_ORGANIZATION_REQUEST = 'UPDATE_ORGANIZATION_REQUEST';
export const UPDATE_ORGANIZATION_SUCCESS = 'UPDATE_ORGANIZATION_SUCCESS';
export const UPDATE_ORGANIZATION_ERROR = 'UPDATE_ORGANIZATION_ERROR';
export const GET_LOGIN_APPS_FOR_ORG_REQUEST = 'GET_LOGIN_APPS_FOR_ORG_REQUEST';
export const GET_LOGIN_APPS_FOR_ORG_SUCCESS = 'GET_LOGIN_APPS_FOR_ORG_SUCCESS';
export const GET_LOGIN_APPS_FOR_ORG_ERROR = 'GET_LOGIN_APPS_FOR_ORG_ERROR';

export const getOrganizationsRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: GET_ORGANIZATIONS_REQUEST,
  });
};

export const getOrganizationsSuccess =
  (result: PaginatedResult<Organization>) => (dispatch: AppDispatch) => {
    dispatch({
      type: GET_ORGANIZATIONS_SUCCESS,
      data: result,
    });
  };

export const getOrganizationsError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: GET_ORGANIZATIONS_ERROR,
      data: error.message,
    });
  };

export const createOrganizationRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: CREATE_ORGANIZATION_REQUEST,
  });
};

export const createOrganizationSuccess =
  (org: Organization) => (dispatch: AppDispatch) => {
    dispatch({
      type: CREATE_ORGANIZATION_SUCCESS,
      data: org,
    });
  };

export const createOrganizationError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: CREATE_ORGANIZATION_ERROR,
      data: error.message,
    });
  };

export const getOrganizationRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: GET_ORGANIZATION_REQUEST,
  });
};

export const getOrganizationSuccess =
  (org: Organization) => (dispatch: AppDispatch) => {
    dispatch({
      type: GET_ORGANIZATION_SUCCESS,
      data: org,
    });
  };

export const getOrganizationError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: GET_ORGANIZATION_ERROR,
      data: error.message,
    });
  };

export const toggleOrganizationEditMode = () => (dispatch: AppDispatch) => {
  dispatch({
    type: TOGGLE_ORGANIZATION_EDIT_MODE,
  });
};

export const modifyOrganization = (data: Record<string, JSONValue>) => ({
  type: MODIFY_ORGANIZATION,
  data,
});

export const updateOrganizationRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: UPDATE_ORGANIZATION_REQUEST,
  });
};

export const updateOrganizationSuccess =
  (org: Organization) => (dispatch: AppDispatch) => {
    dispatch({
      type: UPDATE_ORGANIZATION_SUCCESS,
      data: org,
    });
  };

export const updateOrganizationError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: UPDATE_ORGANIZATION_ERROR,
      data: error.message,
    });
  };

export const getLoginAppsForOrgRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: GET_LOGIN_APPS_FOR_ORG_REQUEST,
  });
};

export const getLoginAppsForOrgSuccess =
  (result: LoginApp[]) => (dispatch: AppDispatch) => {
    dispatch({
      type: GET_LOGIN_APPS_FOR_ORG_SUCCESS,
      data: result,
    });
  };

export const getLoginAppsForOrgError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: GET_LOGIN_APPS_FOR_ORG_ERROR,
      data: error.message,
    });
  };
