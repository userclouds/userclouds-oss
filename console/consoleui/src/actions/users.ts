import { JSONValue } from '@userclouds/sharedui';
import { UserProfileSerialized } from '../models/UserProfile';

export const GET_CURRENT_TENANT_USER_REQUEST =
  'GET_CURRENT_TENANT_USER_REQUEST';
export const GET_CURRENT_TENANT_USER_SUCCESS =
  'GET_CURRENT_TENANT_USER_SUCCESS';
export const GET_CURRENT_TENANT_USER_ERROR = 'GET_CURRENT_TENANT_USER_ERROR';
export const GET_CURRENT_TENANT_USER_CONSENTED_PURPOSES_REQUEST =
  'GET_CURRENT_TENANT_USER_CONSENTED_PURPOSES_REQUEST';
export const GET_CURRENT_TENANT_USER_CONSENTED_PURPOSES_SUCCESS =
  'GET_CURRENT_TENANT_USER_CONSENTED_PURPOSES_SUCCESS';
export const GET_CURRENT_TENANT_USER_CONSENTED_PURPOSES_ERROR =
  'GET_CURRENT_TENANT_USER_CONSENTED_PURPOSES_ERROR';
export const GET_CURRENT_TENANT_USER_EVENTS_REQUEST =
  'GET_CURRENT_TENANT_USER_EVENTS_REQUEST';
export const GET_CURRENT_TENANT_USER_EVENTS_SUCCESS =
  'GET_CURRENT_TENANT_USER_EVENTS_SUCCESS';
export const GET_CURRENT_TENANT_USER_EVENTS_ERROR =
  'GET_CURRENT_TENANT_USER_EVENTS_ERROR';
export const SAVE_TENANT_USER_REQUEST = 'SAVE_TENANT_USER_REQUEST';
export const SAVE_TENANT_USER_SUCCESS = 'SAVE_TENANT_USER_SUCCESS';
export const SAVE_TENANT_USER_ERROR = 'SAVE_TENANT_USER_ERROR';
export const TOGGLE_USER_EDIT_MODE = 'TOGGLE_USER_EDIT_MODE';
export const MODIFY_TENANT_USER_PROFILE = 'MODIFY_TENANT_USER_PROFILE';

export const getCurrentTenantUserRequest = () => ({
  type: GET_CURRENT_TENANT_USER_REQUEST,
});
export const getCurrentTenantUserSuccess = (user: UserProfileSerialized) => ({
  type: GET_CURRENT_TENANT_USER_SUCCESS,
  data: user,
});
export const getCurrentTenantUserError = (error: string) => ({
  type: GET_CURRENT_TENANT_USER_ERROR,
  data: error,
});

export const getCurrentTenantUserConsentedPurposesRequest = () => ({
  type: GET_CURRENT_TENANT_USER_CONSENTED_PURPOSES_REQUEST,
});
export const getCurrentTenantUserConsentedPurposesSuccess = (
  purposes: Array<object>
) => ({
  type: GET_CURRENT_TENANT_USER_CONSENTED_PURPOSES_SUCCESS,
  data: purposes,
});
export const getCurrentTenantUserConsentedPurposesError = (error: string) => ({
  type: GET_CURRENT_TENANT_USER_CONSENTED_PURPOSES_ERROR,
  data: error,
});

export const getCurrentTenantUserEventsRequest = () => ({
  type: GET_CURRENT_TENANT_USER_EVENTS_REQUEST,
});
export const getCurrentTenantUserEventsSuccess = (events: Array<object>) => ({
  type: GET_CURRENT_TENANT_USER_EVENTS_SUCCESS,
  data: events,
});
export const getCurrentTenantUserEventsError = (error: string) => ({
  type: GET_CURRENT_TENANT_USER_EVENTS_ERROR,
  data: error,
});

export const saveTenantUserRequest = () => ({
  type: SAVE_TENANT_USER_REQUEST,
});
export const saveTenantUserSuccess = (user: UserProfileSerialized) => ({
  type: SAVE_TENANT_USER_SUCCESS,
  data: user,
});
export const saveTenantUserError = (error: string) => ({
  type: SAVE_TENANT_USER_ERROR,
  data: error,
});

export const toggleUserEditMode = (editMode?: boolean) => ({
  type: TOGGLE_USER_EDIT_MODE,
  data: editMode,
});

export const modifyTenantUserProfile = (data: Record<string, JSONValue>) => ({
  type: MODIFY_TENANT_USER_PROFILE,
  data: data,
});
