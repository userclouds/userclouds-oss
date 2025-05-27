import { AnyAction } from 'redux';
import { RootState } from '../store';
import {
  GET_PLEX_CONFIG_REQUEST,
  GET_PLEX_CONFIG_SUCCESS,
  GET_PLEX_CONFIG_ERROR,
  MODIFY_PLEX_CONFIG,
  UPDATE_PLEX_CONFIG_REQUEST,
  UPDATE_PLEX_CONFIG_ERROR,
  UPDATE_PLEX_CONFIG_SUCCESS,
  MODIFY_PLEX_APP,
  MODIFY_PLEX_EMPLOYEE_APP,
  CLONE_PLEX_APP_SETTINGS,
  SELECT_PLEX_APP,
  SELECT_PLEX_PROVIDER,
  MODIFY_PLEX_PROVIDER,
  MODIFY_TELEPHONY_PROVIDER,
  TOGGLE_AUTH0_APPS_EDIT_MODE,
  TOGGLE_COGNITO_APPS_EDIT_MODE,
  TOGGLE_UC_APPS_EDIT_MODE,
  GET_PAGE_PARAMETERS_REQUEST,
  GET_PAGE_PARAMETERS_SUCCESS,
  GET_PAGE_PARAMETERS_ERROR,
  MODIFY_PAGE_PARAMETERS,
  RESET_PAGE_PARAMETERS,
  UPDATE_PAGE_PARAMETERS_REQUEST,
  UPDATE_PAGE_PARAMETERS_SUCCESS,
  UPDATE_PAGE_PARAMETERS_ERROR,
  GET_EMAIL_MSG_ELEMENTS_REQUEST,
  GET_EMAIL_MSG_ELEMENTS_SUCCESS,
  GET_EMAIL_MSG_ELEMENTS_ERROR,
  MODIFY_EMAIL_MSG_ELEMENTS,
  RESET_EMAIL_MSG_ELEMENTS,
  CHANGE_SELECTED_EMAIL_MSG_TYPE,
  UPDATE_EMAIL_MSG_ELEMENTS_REQUEST,
  UPDATE_EMAIL_MSG_ELEMENTS_SUCCESS,
  UPDATE_EMAIL_MSG_ELEMENTS_ERROR,
  GET_SMS_MSG_ELEMENTS_REQUEST,
  GET_SMS_MSG_ELEMENTS_SUCCESS,
  GET_SMS_MSG_ELEMENTS_ERROR,
  MODIFY_SMS_MSG_ELEMENTS,
  RESET_SMS_MSG_ELEMENTS,
  CHANGE_SELECTED_SMS_MSG_TYPE,
  UPDATE_SMS_MSG_ELEMENTS_REQUEST,
  UPDATE_SMS_MSG_ELEMENTS_SUCCESS,
  UPDATE_SMS_MSG_ELEMENTS_ERROR,
  CHANGE_CURRENT_OIDC_PROVIDER,
  TOGGLE_PLEX_CONFIG_EDIT_MODE,
  UPDATE_OIDC_PROVIDER_REQUEST,
  UPDATE_OIDC_PROVIDER_ERROR,
  UPDATE_OIDC_PROVIDER_SUCCESS,
  ADD_EXTERNAL_OIDC_ISSUER,
  EDIT_EXTERNAL_OIDC_ISSUER,
  DELETE_EXTERNAL_OIDC_ISSUER,
} from '../actions/authn';
import TenantPlexConfig, {
  UpdatePlexConfigReason,
  modifyTelephonyProvider,
} from '../models/TenantPlexConfig';
import LoginApp from '../models/LoginApp';
import Provider, { ProviderType } from '../models/Provider';
import { blankAuth0Provider } from '../models/Auth0Provider';
import { blankUCProvider } from '../models/UCProvider';
import {
  modifyMessageElement,
  cloneMessageSettings,
} from '../models/MessageElements';
import { blankCognitoProvider } from '../models/CognitoProvider';
import { getNewToggleEditValue } from './reducerHelper';

const authnReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case GET_PLEX_CONFIG_REQUEST:
      state.fetchingPlexConfig = true;
      state.selectedPlexApp = undefined;
      state.modifiedPlexApp = undefined;
      state.plexEmployeeApp = undefined;
      state.modifiedPlexEmployeeApp = undefined;
      state.selectedPlexProvider = undefined;
      state.modifiedPlexProvider = undefined;
      break;
    case GET_PLEX_CONFIG_SUCCESS: {
      const newData = JSON.stringify(action.data);
      state.fetchingPlexConfig = false;
      state.tenantPlexConfig = JSON.parse(newData);
      state.modifiedPlexConfig = JSON.parse(newData);
      state.savePlexConfigSuccess = undefined;
      state.savePlexConfigError = '';
      state.plexConfigIsDirty = false;
      state.selectedPlexApp = undefined;
      state.modifiedPlexApp = undefined;
      state.selectedPlexProvider = undefined;
      state.modifiedPlexProvider = undefined;
      if (state.tenantPlexConfig) {
        if (!state.tenantPlexConfig.tenant_config.plex_map.employee_app.id) {
          state.plexEmployeeApp = undefined;
          state.modifiedPlexEmployeeApp = undefined;
        } else {
          const newApp = JSON.stringify(
            state.tenantPlexConfig.tenant_config.plex_map.employee_app
          );
          state.plexEmployeeApp = JSON.parse(newApp);
          state.modifiedPlexEmployeeApp = JSON.parse(newApp);
        }
      }
      break;
    }
    case GET_PLEX_CONFIG_ERROR:
      state.fetchingPlexConfig = false;
      state.fetchPlexConfigError = action.data;
      break;
    case MODIFY_PLEX_CONFIG: {
      const newData = JSON.stringify(action.data);
      state.modifiedPlexConfig = JSON.parse(newData);
      state.plexConfigIsDirty =
        newData !== JSON.stringify(state.tenantPlexConfig);
      state.savePlexConfigSuccess = undefined;
      break;
    }
    case MODIFY_PLEX_APP: {
      const newData = JSON.stringify(action.data);
      state.modifiedPlexApp = JSON.parse(newData);
      state.plexConfigIsDirty =
        newData !== JSON.stringify(state.selectedPlexApp);
      state.savePlexConfigSuccess = undefined;
      break;
    }
    case MODIFY_PLEX_EMPLOYEE_APP: {
      const newData = JSON.stringify(action.data);
      state.modifiedPlexEmployeeApp = JSON.parse(newData) as LoginApp;
      if (state.modifiedPlexConfig && state.modifiedPlexEmployeeApp) {
        state.modifiedPlexConfig.tenant_config.plex_map.employee_app =
          state.modifiedPlexEmployeeApp;
      }
      state.plexConfigIsDirty =
        newData !== JSON.stringify(state.plexEmployeeApp);
      state.savePlexConfigSuccess = undefined;
      break;
    }
    case MODIFY_PLEX_PROVIDER: {
      const newData = JSON.stringify(action.data);
      state.modifiedPlexProvider = JSON.parse(newData) as Provider;
      if (state.modifiedPlexProvider.type === ProviderType.uc) {
        delete state.modifiedPlexProvider.auth0;
        delete state.modifiedPlexProvider.cognito;
        state.modifiedPlexProvider.uc =
          state.modifiedPlexProvider.uc || blankUCProvider();
      } else if (state.modifiedPlexProvider.type === ProviderType.auth0) {
        delete state.modifiedPlexProvider.uc;
        delete state.modifiedPlexProvider.cognito;
        state.modifiedPlexProvider.auth0 =
          state.modifiedPlexProvider.auth0 || blankAuth0Provider();
      } else if (state.modifiedPlexProvider.type === ProviderType.cognito) {
        delete state.modifiedPlexProvider.uc;
        delete state.modifiedPlexProvider.auth0;
        state.modifiedPlexProvider.cognito =
          state.modifiedPlexProvider.cognito || blankCognitoProvider();
      }
      state.plexConfigIsDirty =
        newData !== JSON.stringify(state.selectedPlexProvider);
      state.savePlexConfigSuccess = undefined;
      break;
    }
    case CHANGE_CURRENT_OIDC_PROVIDER:
      state.savePlexConfigSuccess = undefined;
      state.savePlexConfigError = '';
      state.savingOIDCProvider = false;
      state.saveOIDCProviderSuccess = false;
      state.saveOIDCProviderError = '';
      state.oidcProvider = { ...action.data };
      break;
    case TOGGLE_PLEX_CONFIG_EDIT_MODE:
      state.savePlexConfigSuccess = undefined;
      state.savePlexConfigError = '';
      state.savingOIDCProvider = false;
      state.saveOIDCProviderSuccess = false;
      state.saveOIDCProviderError = '';
      state.editingPlexConfig = getNewToggleEditValue(
        action.data,
        state.editingPlexConfig
      );
      break;
    case MODIFY_TELEPHONY_PROVIDER:
      if (state.modifiedPlexConfig) {
        state.modifiedPlexConfig = modifyTelephonyProvider(
          state.modifiedPlexConfig,
          action.data
        );
        state.plexConfigIsDirty =
          JSON.stringify(state.modifiedPlexConfig) !==
          JSON.stringify(state.selectedPlexProvider);
      }
      break;
    case UPDATE_OIDC_PROVIDER_REQUEST:
      state.savingOIDCProvider = true;
      state.saveOIDCProviderSuccess = false;
      state.saveOIDCProviderError = '';
      break;
    case UPDATE_OIDC_PROVIDER_SUCCESS:
      state.savingOIDCProvider = false;
      state.saveOIDCProviderSuccess = true;
      break;
    case UPDATE_OIDC_PROVIDER_ERROR:
      state.savingOIDCProvider = false;
      state.saveOIDCProviderError = action.data;
      break;
    case UPDATE_PLEX_CONFIG_REQUEST:
      state.savingPlexConfig = true;
      state.savePlexConfigSuccess = undefined;
      state.savePlexConfigError = '';
      break;
    case UPDATE_PLEX_CONFIG_SUCCESS: {
      const { config, reason } = action.data;
      const newData = JSON.stringify(config);
      state.savingPlexConfig = false;
      state.savePlexConfigSuccess = reason || UpdatePlexConfigReason.Default;
      state.tenantPlexConfig = JSON.parse(newData) as TenantPlexConfig;
      state.modifiedPlexConfig = JSON.parse(newData) as TenantPlexConfig;

      if (state.selectedPlexApp) {
        const matchingApp =
          state.tenantPlexConfig.tenant_config.plex_map.apps.find(
            (app: LoginApp) => app.id === state.selectedPlexApp!.id
          );
        if (matchingApp) {
          const newAppData = JSON.stringify(matchingApp);
          state.selectedPlexApp = JSON.parse(newAppData);
          state.modifiedPlexApp = JSON.parse(newAppData);
        } else {
          state.selectedPlexApp = undefined;
          state.modifiedPlexApp = undefined;
        }
      }

      if (state.selectedPlexProvider) {
        const matchingProvider =
          state.tenantPlexConfig.tenant_config.plex_map.providers.find(
            (provider: Provider) =>
              provider.id === state.selectedPlexProvider!.id
          );
        if (matchingProvider) {
          const newProviderData = JSON.stringify(matchingProvider);
          state.selectedPlexProvider = JSON.parse(newProviderData);
          state.modifiedPlexProvider = JSON.parse(newProviderData);
        } else {
          state.selectedPlexProvider = undefined;
          state.modifiedPlexProvider = undefined;
        }
      }

      state.plexConfigIsDirty = false;
      // TODO: on provider page,
      // only exit edit mode for the apps
      // that have just been saved
      state.auth0AppsEditMode = false;
      state.cognitoAppsEditMode = false;
      state.ucAppsEditMode = false;
      state.editingPlexConfig = false;
      break;
    }
    case UPDATE_PLEX_CONFIG_ERROR:
      state.savingPlexConfig = false;
      state.savePlexConfigError = action.data;
      break;
    case SELECT_PLEX_APP:
      state.savePlexConfigSuccess = undefined;
      state.savePlexConfigError = '';
      state.plexConfigIsDirty = false;
      if (state.tenantPlexConfig) {
        // TODO: we should ditch any modifications if you navigate away
        // from the AuthN home page or Plex App home page,
        // but this resets the potentially dirty config just in case
        state.modifiedPlexConfig = JSON.parse(
          JSON.stringify(state.tenantPlexConfig)
        );
        const matchingApp =
          state.tenantPlexConfig.tenant_config.plex_map.apps.find(
            (app: LoginApp) => app.id === action.data
          );
        if (matchingApp) {
          const newAppData = JSON.stringify(matchingApp);
          state.selectedPlexApp = JSON.parse(newAppData);
          state.modifiedPlexApp = JSON.parse(newAppData);
          break;
        }
      }
      state.selectedPlexApp = undefined;
      state.modifiedPlexApp = undefined;
      break;
    case SELECT_PLEX_PROVIDER:
      state.savePlexConfigSuccess = undefined;
      state.savePlexConfigError = '';
      state.plexConfigIsDirty = false;
      state.auth0AppsEditMode = false;
      state.cognitoAppsEditMode = false;
      state.ucAppsEditMode = false;
      if (state.tenantPlexConfig) {
        // TODO: we should ditch any modifications if you navigate away
        // from the AuthN home page or Plex App home page,
        // but this resets the potentially dirty config just in case
        state.modifiedPlexConfig = JSON.parse(
          JSON.stringify(state.tenantPlexConfig)
        );

        const matchingProvider =
          state.tenantPlexConfig.tenant_config.plex_map.providers.find(
            (provider: Provider) => provider.id === action.data
          );
        if (matchingProvider) {
          const newProviderData = JSON.stringify(matchingProvider);
          state.selectedPlexProvider = JSON.parse(newProviderData);
          state.modifiedPlexProvider = JSON.parse(newProviderData);
          break;
        }
      }
      state.selectedPlexProvider = undefined;
      state.modifiedPlexProvider = undefined;
      break;
    case TOGGLE_AUTH0_APPS_EDIT_MODE:
      state.auth0AppsEditMode = !state.auth0AppsEditMode;
      break;
    case TOGGLE_COGNITO_APPS_EDIT_MODE:
      state.cognitoAppsEditMode = !state.cognitoAppsEditMode;
      break;
    case TOGGLE_UC_APPS_EDIT_MODE:
      state.ucAppsEditMode = !state.ucAppsEditMode;
      break;
    case CLONE_PLEX_APP_SETTINGS: {
      const appID = action.data;
      const pageParameters = state.appPageParameters[appID];
      if (
        state.tenantEmailMessageElements &&
        state.tenantSMSMessageElements &&
        state.selectedPlexApp &&
        pageParameters
      ) {
        state.modifiedPageParameters = JSON.parse(
          JSON.stringify(pageParameters)
        );
        state.appToClone = appID;
        state.modifiedEmailMessageElements = cloneMessageSettings(
          state.selectedPlexApp.id, // target app
          appID, // source app
          state.tenantEmailMessageElements
        );
        state.emailMessageElementsAreDirty =
          JSON.stringify(state.modifiedEmailMessageElements) !==
          JSON.stringify(state.tenantEmailMessageElements);
        state.modifiedSMSMessageElements = cloneMessageSettings(
          state.selectedPlexApp.id, // target app
          appID, // source app
          state.tenantSMSMessageElements
        );
        state.smsMessageElementsAreDirty =
          JSON.stringify(state.modifiedSMSMessageElements) !==
          JSON.stringify(state.tenantSMSMessageElements);
        state.savePlexConfigSuccess = undefined;
      }
      break;
    }

    case GET_PAGE_PARAMETERS_REQUEST:
      state.fetchingPageParameters = true;
      state.pageParametersFetchError = '';
      state.pageParametersSaveSuccess = false;
      state.pageParametersSaveError = '';
      break;
    case GET_PAGE_PARAMETERS_SUCCESS:
      state.fetchingPageParameters = false;
      state.appPageParameters[action.data.app_id] = action.data;
      state.modifiedPageParameters = action.data;
      break;
    case GET_PAGE_PARAMETERS_ERROR:
      state.fetchingPageParameters = false;
      state.pageParametersFetchError = action.data;
      break;
    case MODIFY_PAGE_PARAMETERS:
      state.modifiedPageParameters = action.data;
      break;
    case RESET_PAGE_PARAMETERS:
      state.modifiedPageParameters = action.data;
      break;
    case UPDATE_PAGE_PARAMETERS_REQUEST:
      state.savingPageParameters = true;
      state.pageParametersFetchError = '';
      state.pageParametersSaveError = '';
      break;
    case UPDATE_PAGE_PARAMETERS_SUCCESS:
      state.savingPageParameters = false;
      state.appPageParameters[action.data.app_id] = action.data;
      state.modifiedPageParameters = action.data;
      state.pageParametersSaveSuccess = true;
      break;
    case UPDATE_PAGE_PARAMETERS_ERROR:
      state.savingPageParameters = false;
      state.pageParametersSaveError = action.data;
      break;

    case GET_EMAIL_MSG_ELEMENTS_REQUEST:
      state.fetchingEmailMessageElements = true;
      state.emailMessageElementsFetchError = '';
      state.emailMessageElementsSaveSuccess = false;
      state.emailMessageElementsSaveError = '';
      break;
    case GET_EMAIL_MSG_ELEMENTS_SUCCESS: {
      const newEmailData = JSON.stringify(action.data);
      state.fetchingEmailMessageElements = false;
      state.tenantEmailMessageElements = JSON.parse(newEmailData);
      state.modifiedEmailMessageElements = JSON.parse(newEmailData);
      state.emailMessageElementsAreDirty = false;
      break;
    }
    case GET_EMAIL_MSG_ELEMENTS_ERROR:
      state.fetchingEmailMessageElements = false;
      state.emailMessageElementsFetchError = action.data;
      break;
    case MODIFY_EMAIL_MSG_ELEMENTS: {
      // TODO: unit test
      const { app_id, message_type, element_name, value } = action.data;
      if (state.modifiedEmailMessageElements) {
        state.modifiedEmailMessageElements = modifyMessageElement(
          state.modifiedEmailMessageElements,
          app_id,
          message_type,
          element_name,
          value
        );
      }
      state.emailMessageElementsAreDirty =
        JSON.stringify(state.modifiedEmailMessageElements) !==
        JSON.stringify(state.tenantEmailMessageElements);
      break;
    }
    case RESET_EMAIL_MSG_ELEMENTS:
      if (state.tenantEmailMessageElements) {
        state.modifiedEmailMessageElements = [
          ...state.tenantEmailMessageElements,
        ];
        state.emailMessageElementsAreDirty = false;
      }
      break;
    case CHANGE_SELECTED_EMAIL_MSG_TYPE:
      state.selectedEmailMessageType = action.data;
      if (state.tenantEmailMessageElements) {
        state.modifiedEmailMessageElements = [
          ...state.tenantEmailMessageElements,
        ];
      }
      break;
    case UPDATE_EMAIL_MSG_ELEMENTS_REQUEST:
      state.savingEmailMessageElements = true;
      state.emailMessageElementsFetchError = '';
      state.emailMessageElementsSaveSuccess = false;
      state.emailMessageElementsSaveError = '';
      break;
    case UPDATE_EMAIL_MSG_ELEMENTS_SUCCESS:
      state.savingEmailMessageElements = false;
      state.emailMessageElementsSaveSuccess = true;
      state.tenantEmailMessageElements = action.data;
      state.modifiedEmailMessageElements = [...action.data];
      state.emailMessageElementsAreDirty = false;
      break;
    case UPDATE_EMAIL_MSG_ELEMENTS_ERROR:
      state.savingEmailMessageElements = false;
      state.emailMessageElementsSaveError = action.data;
      break;

    case GET_SMS_MSG_ELEMENTS_REQUEST:
      state.fetchingSMSMessageElements = true;
      state.smsMessageElementsFetchError = '';
      state.smsMessageElementsSaveSuccess = false;
      state.smsMessageElementsSaveError = '';
      break;
    case GET_SMS_MSG_ELEMENTS_SUCCESS: {
      const newSMSData = JSON.stringify(action.data);
      state.fetchingSMSMessageElements = false;
      state.tenantSMSMessageElements = JSON.parse(newSMSData);
      state.modifiedSMSMessageElements = JSON.parse(newSMSData);
      state.smsMessageElementsAreDirty = false;
      break;
    }
    case GET_SMS_MSG_ELEMENTS_ERROR:
      state.fetchingSMSMessageElements = false;
      state.smsMessageElementsFetchError = action.data;
      break;
    case MODIFY_SMS_MSG_ELEMENTS: {
      // TODO: unit test
      const { app_id, message_type, element_name, value } = action.data;
      if (state.modifiedSMSMessageElements) {
        state.modifiedSMSMessageElements = modifyMessageElement(
          state.modifiedSMSMessageElements,
          app_id,
          message_type,
          element_name,
          value
        );
      }
      state.smsMessageElementsAreDirty =
        JSON.stringify(state.modifiedSMSMessageElements) !==
        JSON.stringify(state.tenantSMSMessageElements);
      break;
    }
    case RESET_SMS_MSG_ELEMENTS:
      if (state.tenantSMSMessageElements) {
        state.modifiedSMSMessageElements = [...state.tenantSMSMessageElements];
        state.smsMessageElementsAreDirty = false;
      }
      break;
    case CHANGE_SELECTED_SMS_MSG_TYPE:
      state.selectedSMSMessageType = action.data;
      if (state.tenantSMSMessageElements) {
        state.modifiedSMSMessageElements = [...state.tenantSMSMessageElements];
      }
      break;
    case UPDATE_SMS_MSG_ELEMENTS_REQUEST:
      state.savingSMSMessageElements = true;
      state.smsMessageElementsFetchError = '';
      state.smsMessageElementsSaveSuccess = false;
      state.smsMessageElementsSaveError = '';
      break;
    case UPDATE_SMS_MSG_ELEMENTS_SUCCESS:
      state.savingSMSMessageElements = false;
      state.smsMessageElementsSaveSuccess = true;
      state.tenantSMSMessageElements = action.data;
      state.modifiedSMSMessageElements = [...action.data];
      state.smsMessageElementsAreDirty = false;
      break;
    case UPDATE_SMS_MSG_ELEMENTS_ERROR:
      state.savingSMSMessageElements = false;
      state.smsMessageElementsSaveError = action.data;
      break;
    case ADD_EXTERNAL_OIDC_ISSUER:
      state.editingPlexConfig = true;
      state.creatingIssuer = true;
      state.tenantIssuerDialogIsOpen = true;
      if (state.modifiedPlexConfig) {
        state.modifiedPlexConfig.tenant_config.external_oidc_issuers?.length > 0
          ? state.modifiedPlexConfig.tenant_config.external_oidc_issuers.push(
              ''
            )
          : (state.modifiedPlexConfig.tenant_config.external_oidc_issuers = [
              '',
            ]);
        state.editingIssuerIndex =
          state.modifiedPlexConfig.tenant_config.external_oidc_issuers.length -
          1;
        state.modifiedPlexConfig = { ...state.modifiedPlexConfig };
        state.plexConfigIsDirty =
          JSON.stringify(state.modifiedPlexConfig) !==
          JSON.stringify(state.tenantPlexConfig);
      }
      break;
    case EDIT_EXTERNAL_OIDC_ISSUER:
      if (state.modifiedPlexConfig && state.editingIssuerIndex >= 0) {
        state.modifiedPlexConfig.tenant_config.external_oidc_issuers[
          state.editingIssuerIndex
        ] = action.data;
        state.modifiedPlexConfig.tenant_config.external_oidc_issuers = [
          ...state.modifiedPlexConfig.tenant_config.external_oidc_issuers,
        ];
        state.modifiedPlexConfig = { ...state.modifiedPlexConfig };
        state.plexConfigIsDirty =
          JSON.stringify(state.modifiedPlexConfig) !==
          JSON.stringify(state.tenantPlexConfig);
      }
      break;
    case DELETE_EXTERNAL_OIDC_ISSUER:
      if (state.modifiedPlexConfig) {
        const index = action.data;
        if (
          index > -1 &&
          index <
            state.modifiedPlexConfig.tenant_config.external_oidc_issuers?.length
        ) {
          state.modifiedPlexConfig.tenant_config.external_oidc_issuers.splice(
            index,
            1
          );
          state.modifiedPlexConfig.tenant_config.external_oidc_issuers = [
            ...state.modifiedPlexConfig.tenant_config.external_oidc_issuers,
          ];
          state.modifiedPlexConfig = {
            ...state.modifiedPlexConfig,
          };
          state.plexConfigIsDirty =
            JSON.stringify(state.modifiedPlexConfig) !==
            JSON.stringify(state.tenantPlexConfig);
        }
      }
      break;
    default:
      break;
  }

  return state;
};

export default authnReducer;
