import { APIError } from '@userclouds/sharedui';

import TenantPlexConfig, {
  UpdatePlexConfigReason,
} from '../models/TenantPlexConfig';
import LoginApp from '../models/LoginApp';
import { TelephonyProviderProperties } from '../models/TelephonyProvider';
import { PageParametersResponse } from '../models/PageParameters';
import { AppMessageElement, MessageType } from '../models/MessageElements';
import Provider from '../models/Provider';
import {
  getBlankCustomOIDCProvider,
  OIDCProvider,
} from '../models/OIDCProvider';

export const GET_PLEX_CONFIG_REQUEST = 'GET_PLEX_CONFIG_REQUEST';
export const GET_PLEX_CONFIG_SUCCESS = 'GET_PLEX_CONFIG_SUCCESS';
export const GET_PLEX_CONFIG_ERROR = 'GET_PLEX_CONFIG_ERROR';
export const MODIFY_PLEX_CONFIG = 'MODIFY_PLEX_CONFIG';
export const UPDATE_PLEX_CONFIG_REQUEST = 'UPDATE_PLEX_CONFIG_REQUEST';
export const UPDATE_PLEX_CONFIG_SUCCESS = 'UPDATE_PLEX_CONFIG_SUCCESS';
export const UPDATE_PLEX_CONFIG_ERROR = 'UPDATE_PLEX_CONFIG_ERROR';

export const SELECT_PLEX_APP = 'SELECT_PLEX_APP';
export const MODIFY_PLEX_APP = 'MODIFY_PLEX_APP';
export const MODIFY_PLEX_EMPLOYEE_APP = 'MODIFY_PLEX_EMPLOYEE_APP';
export const CLONE_PLEX_APP_SETTINGS = 'CLONE_PLEX_APP_SETTINGS';

export const SELECT_PLEX_PROVIDER = 'SELECT_PLEX_PROVIDER';
export const MODIFY_PLEX_PROVIDER = 'MODIFY_PLEX_PROVIDER';
export const TOGGLE_AUTH0_APPS_EDIT_MODE = 'TOGGLE_AUTH0_APPS_EDIT_MODE';
export const TOGGLE_COGNITO_APPS_EDIT_MODE = 'TOGGLE_COGNITO_APPS_EDIT_MODE';
export const TOGGLE_UC_APPS_EDIT_MODE = 'TOGGLE_UC_APPS_EDIT_MODE';

export const MODIFY_TELEPHONY_PROVIDER = 'MODIFY_TELEPHONY_PROVIDER';

export const GET_PAGE_PARAMETERS_REQUEST = 'GET_PAGE_PARAMETERS_REQUEST';
export const GET_PAGE_PARAMETERS_SUCCESS = 'GET_PAGE_PARAMETERS_SUCCESS';
export const GET_PAGE_PARAMETERS_ERROR = 'GET_PAGE_PARAMETERS_ERROR';
export const MODIFY_PAGE_PARAMETERS = 'MODIFY_PAGE_PARAMETERS';
export const RESET_PAGE_PARAMETERS = 'RESET_PAGE_PARAMETERS';
export const UPDATE_PAGE_PARAMETERS_REQUEST = 'UPDATE_PAGE_PARAMETERS_REQUEST';
export const UPDATE_PAGE_PARAMETERS_SUCCESS = 'UPDATE_PAGE_PARAMETERS_SUCCESS';
export const UPDATE_PAGE_PARAMETERS_ERROR = 'UPDATE_PAGE_PARAMETERS_ERROR';

export const GET_EMAIL_MSG_ELEMENTS_REQUEST = 'GET_EMAIL_MSG_ELEMENTS_REQUEST';
export const GET_EMAIL_MSG_ELEMENTS_SUCCESS = 'GET_EMAIL_MSG_ELEMENTS_SUCCESS';
export const GET_EMAIL_MSG_ELEMENTS_ERROR = 'GET_EMAIL_MSG_ELEMENTS_ERROR';
export const CHANGE_SELECTED_EMAIL_MSG_TYPE = 'CHANGE_SELECTED_EMAIL_MSG_TYPE';
export const MODIFY_EMAIL_MSG_ELEMENTS = 'MODIFY_EMAIL_MSG_ELEMENTS';
export const RESET_EMAIL_MSG_ELEMENTS = 'RESET_EMAIL_MSG_ELEMENTS';
export const UPDATE_EMAIL_MSG_ELEMENTS_REQUEST =
  'UPDATE_EMAIL_MSG_ELEMENTS_REQUEST';
export const UPDATE_EMAIL_MSG_ELEMENTS_SUCCESS =
  'UPDATE_EMAIL_MSG_ELEMENTS_SUCCESS';
export const UPDATE_EMAIL_MSG_ELEMENTS_ERROR =
  'UPDATE_EMAIL_MSG_ELEMENTS_ERROR';

export const GET_SMS_MSG_ELEMENTS_REQUEST = 'GET_SMS_MSG_ELEMENTS_REQUEST';
export const GET_SMS_MSG_ELEMENTS_SUCCESS = 'GET_SMS_MSG_ELEMENTS_SUCCESS';
export const GET_SMS_MSG_ELEMENTS_ERROR = 'GET_SMS_MSG_ELEMENTS_ERROR';
export const CHANGE_SELECTED_SMS_MSG_TYPE = 'CHANGE_SELECTED_SMS_MSG_TYPE';
export const MODIFY_SMS_MSG_ELEMENTS = 'MODIFY_SMS_MSG_ELEMENTS';
export const RESET_SMS_MSG_ELEMENTS = 'RESET_SMS_MSG_ELEMENTS';
export const UPDATE_SMS_MSG_ELEMENTS_REQUEST =
  'UPDATE_SMS_MSG_ELEMENTS_REQUEST';
export const UPDATE_SMS_MSG_ELEMENTS_SUCCESS =
  'UPDATE_SMS_MSG_ELEMENTS_SUCCESS';
export const UPDATE_SMS_MSG_ELEMENTS_ERROR = 'UPDATE_SMS_MSG_ELEMENTS_ERROR';
export const GET_BLANK_OIDC_PROVIDER = 'GET_BLANK_OIDC_PROVIDER';
export const CHANGE_CURRENT_OIDC_PROVIDER = 'CHANGE_CURRENT_OIDC_PROVIDER';
export const TOGGLE_PLEX_CONFIG_EDIT_MODE = 'TOGGLE_PLEX_CONFIG_EDIT_MODE';
export const UPDATE_OIDC_PROVIDER_REQUEST = 'UPDATE_OIDC_PROVIDER_REQUEST';
export const UPDATE_OIDC_PROVIDER_SUCCESS = 'UPDATE_OIDC_PROVIDER_SUCCESS';
export const UPDATE_OIDC_PROVIDER_ERROR = 'UPDATE_OIDC_PROVIDER_ERROR';

export const ADD_EXTERNAL_OIDC_ISSUER = 'ADD_EXTERNAL_OIDC_ISSUER';
export const EDIT_EXTERNAL_OIDC_ISSUER = 'EDIT_EXTERNAL_OIDC_ISSUER';
export const DELETE_EXTERNAL_OIDC_ISSUER = 'DELETE_EXTERNAL_OIDC_ISSUER';

export const getPlexConfigRequest = () => ({
  type: GET_PLEX_CONFIG_REQUEST,
});
export const getPlexConfigSuccess = (config: TenantPlexConfig) => ({
  type: GET_PLEX_CONFIG_SUCCESS,
  data: config,
});
export const getPlexConfigError = (error: APIError) => ({
  type: GET_PLEX_CONFIG_ERROR,
  data: error.message,
});
export const updatePlexConfigRequest = () => ({
  type: UPDATE_PLEX_CONFIG_REQUEST,
});
export const updatePlexConfigSuccess = (
  config: TenantPlexConfig,
  reason?: UpdatePlexConfigReason
) => ({
  type: UPDATE_PLEX_CONFIG_SUCCESS,
  data: {
    config,
    reason,
  },
});
export const updatePlexConfigError = (error: APIError) => ({
  type: UPDATE_PLEX_CONFIG_ERROR,
  data: error.message,
});
export const modifyPlexConfig = (data: TenantPlexConfig) => ({
  type: MODIFY_PLEX_CONFIG,
  data,
});

export const selectPlexApp = (appID: string) => ({
  type: SELECT_PLEX_APP,
  data: appID,
});
export const modifyPlexApp = (data: LoginApp) => ({
  type: MODIFY_PLEX_APP,
  data,
});
export const modifyPlexEmployeeApp = (data: LoginApp) => ({
  type: MODIFY_PLEX_EMPLOYEE_APP,
  data,
});
export const clonePlexAppSettings = (appID: string) => ({
  type: CLONE_PLEX_APP_SETTINGS,
  data: appID,
});

export const selectPlexProvider = (providerID: string) => ({
  type: SELECT_PLEX_PROVIDER,
  data: providerID,
});
export const modifyPlexProvider = (data: Provider) => ({
  type: MODIFY_PLEX_PROVIDER,
  data,
});
export const toggleAuth0AppsEditMode = (editMode?: boolean) => ({
  type: TOGGLE_AUTH0_APPS_EDIT_MODE,
  data: editMode,
});
export const toggleCognitoAppsEditMode = (editMode?: boolean) => ({
  type: TOGGLE_COGNITO_APPS_EDIT_MODE,
  data: editMode,
});

export const toggleUCAppsEditMode = (editMode?: boolean) => ({
  type: TOGGLE_UC_APPS_EDIT_MODE,
  data: editMode,
});

export const modifyTelephonyProvider = (
  changes: TelephonyProviderProperties
) => ({
  type: MODIFY_TELEPHONY_PROVIDER,
  data: changes,
});

export const getPageParametersRequest = () => ({
  type: GET_PAGE_PARAMETERS_REQUEST,
});
export const getPageParametersSuccess = (params: PageParametersResponse) => ({
  type: GET_PAGE_PARAMETERS_SUCCESS,
  data: params,
});
export const getPageParametersError = (error: APIError) => ({
  type: GET_PAGE_PARAMETERS_ERROR,
  data: error.message,
});
export const modifyPageParameters = (data: PageParametersResponse) => ({
  type: MODIFY_PAGE_PARAMETERS,
  data,
});
export const resetPageParameters = (data: PageParametersResponse) => ({
  type: RESET_PAGE_PARAMETERS,
  data,
});
export const updatePageParametersRequest = () => ({
  type: UPDATE_PAGE_PARAMETERS_REQUEST,
});
export const updatePageParametersSuccess = (
  params: PageParametersResponse
) => ({
  type: UPDATE_PAGE_PARAMETERS_SUCCESS,
  data: params,
});
export const updatePageParametersError = (error: APIError) => ({
  type: UPDATE_PAGE_PARAMETERS_ERROR,
  data: error.message,
});

export const getEmailMessageElementsRequest = () => ({
  type: GET_EMAIL_MSG_ELEMENTS_REQUEST,
});
export const getEmailMessageElementsSuccess = (
  elements: AppMessageElement[]
) => ({
  type: GET_EMAIL_MSG_ELEMENTS_SUCCESS,
  data: elements,
});
export const getEmailMessageElementsError = (error: APIError) => ({
  type: GET_EMAIL_MSG_ELEMENTS_ERROR,
  data: error.message,
});
export const modifyEmailMessageElements = (
  appID: string,
  messageType: MessageType,
  element_name: string,
  value: string
) => ({
  type: MODIFY_EMAIL_MSG_ELEMENTS,
  data: {
    app_id: appID,
    message_type: messageType,
    element_name,
    value,
  },
});
export const resetEmailMessageElements = () => ({
  type: RESET_EMAIL_MSG_ELEMENTS,
});
export const changeSelectedEmailMessageType = (type: MessageType) => ({
  type: CHANGE_SELECTED_EMAIL_MSG_TYPE,
  data: type,
});
export const updateEmailMessageElementsRequest = () => ({
  type: UPDATE_EMAIL_MSG_ELEMENTS_REQUEST,
});
export const updateEmailMessageElementsSuccess = (
  elements: AppMessageElement[]
) => ({
  type: UPDATE_EMAIL_MSG_ELEMENTS_SUCCESS,
  data: elements,
});
export const updateEmailMessageElementsError = (error: APIError) => ({
  type: UPDATE_EMAIL_MSG_ELEMENTS_ERROR,
  data: error.message,
});

export const getSMSMessageElementsRequest = () => ({
  type: GET_SMS_MSG_ELEMENTS_REQUEST,
});
export const getSMSMessageElementsSuccess = (
  elements: AppMessageElement[]
) => ({
  type: GET_SMS_MSG_ELEMENTS_SUCCESS,
  data: elements,
});
export const getSMSMessageElementsError = (error: APIError) => ({
  type: GET_SMS_MSG_ELEMENTS_ERROR,
  data: error.message,
});
export const resetSMSMessageElements = () => ({
  type: RESET_SMS_MSG_ELEMENTS,
});
export const changeSelectedSMSMessageType = (type: MessageType) => ({
  type: CHANGE_SELECTED_SMS_MSG_TYPE,
  data: type,
});
export const modifySMSMessageElements = (
  appID: string,
  messageType: MessageType,
  element_name: string,
  value: string
) => ({
  type: MODIFY_SMS_MSG_ELEMENTS,
  data: {
    app_id: appID,
    message_type: messageType,
    element_name,
    value,
  },
});
export const updateSMSMessageElementsRequest = () => ({
  type: UPDATE_SMS_MSG_ELEMENTS_REQUEST,
});
export const updateSMSMessageElementsSuccess = (
  elements: AppMessageElement[]
) => ({
  type: UPDATE_SMS_MSG_ELEMENTS_SUCCESS,
  data: elements,
});
export const updateSMSMessageElementsError = (error: APIError) => ({
  type: UPDATE_SMS_MSG_ELEMENTS_ERROR,
  data: error.message,
});

export const changeCurrentOIDCProvider = (oidcProvider: OIDCProvider) => ({
  type: CHANGE_CURRENT_OIDC_PROVIDER,
  data: oidcProvider,
});

export const getBlankOIDCProvider = () => ({
  type: CHANGE_CURRENT_OIDC_PROVIDER,
  data: getBlankCustomOIDCProvider(),
});

export const togglePlexConfigEditMode = (editMode?: boolean) => ({
  type: TOGGLE_PLEX_CONFIG_EDIT_MODE,
  data: editMode,
});

export const updateOIDCProviderRequest = () => ({
  type: UPDATE_OIDC_PROVIDER_REQUEST,
});

export const updateOIDCProviderSuccess = () => ({
  type: UPDATE_OIDC_PROVIDER_SUCCESS,
});

export const updateOIDCProviderError = (error: APIError) => ({
  type: UPDATE_OIDC_PROVIDER_ERROR,
  data: error.message,
});

export const addExternalOIDCIssuer = () => ({
  type: ADD_EXTERNAL_OIDC_ISSUER,
});

export const editExternalOIDCIssuer = (issuer: string) => ({
  type: EDIT_EXTERNAL_OIDC_ISSUER,
  data: issuer,
});

export const deleteExternalOIDCIssuer = (index: number) => ({
  type: DELETE_EXTERNAL_OIDC_ISSUER,
  data: index,
});
