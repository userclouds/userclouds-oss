import { AnyAction } from 'redux';
import { RootState } from '../store';
import {
  GET_ORGANIZATIONS_REQUEST,
  GET_ORGANIZATIONS_SUCCESS,
  GET_ORGANIZATIONS_ERROR,
  GET_ORGANIZATION_REQUEST,
  GET_ORGANIZATION_SUCCESS,
  GET_ORGANIZATION_ERROR,
  CREATE_ORGANIZATION_REQUEST,
  CREATE_ORGANIZATION_SUCCESS,
  CREATE_ORGANIZATION_ERROR,
  TOGGLE_ORGANIZATION_EDIT_MODE,
  UPDATE_ORGANIZATION_REQUEST,
  UPDATE_ORGANIZATION_SUCCESS,
  UPDATE_ORGANIZATION_ERROR,
  GET_LOGIN_APPS_FOR_ORG_REQUEST,
  GET_LOGIN_APPS_FOR_ORG_SUCCESS,
  GET_LOGIN_APPS_FOR_ORG_ERROR,
  MODIFY_ORGANIZATION,
} from '../actions/organizations';

const orgsReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case GET_ORGANIZATIONS_REQUEST:
      state.fetchingOrganizations = true;
      state.organizationsFetchError = '';
      break;
    case GET_ORGANIZATIONS_SUCCESS:
      state.fetchingOrganizations = false;
      state.organizations = action.data;
      break;
    case GET_ORGANIZATIONS_ERROR:
      state.fetchingOrganizations = false;
      state.organizationsFetchError = action.data;
      break;
    case GET_ORGANIZATION_REQUEST:
      state.organizationsFetchError = '';
      state.fetchingOrganizations = true;
      break;
    case GET_ORGANIZATION_SUCCESS:
      state.fetchingOrganizations = false;
      state.selectedOrganization = action.data;
      break;
    case GET_ORGANIZATION_ERROR:
      state.fetchingOrganizations = false;
      state.organizationsFetchError = action.data;
      break;
    case CREATE_ORGANIZATION_REQUEST:
      state.savingOrganization = true;
      state.createOrganizationError = '';
      break;
    case CREATE_ORGANIZATION_SUCCESS:
      state.savingOrganization = false;
      break;
    case CREATE_ORGANIZATION_ERROR:
      state.savingOrganization = false;
      state.createOrganizationError = action.data;
      break;
    case TOGGLE_ORGANIZATION_EDIT_MODE:
      if (state.selectedOrganization) {
        state.modifiedOrganization = {
          ...state.selectedOrganization,
        };
        state.editingOrganization = !state.editingOrganization;
      }
      break;
    case MODIFY_ORGANIZATION:
      state.modifiedOrganization = {
        ...state.modifiedOrganization,
        ...action.data,
      };
      break;
    case UPDATE_ORGANIZATION_REQUEST:
      state.savingOrganization = true;
      state.updateOrganizationError = '';
      break;
    case UPDATE_ORGANIZATION_SUCCESS:
      state.savingOrganization = false;
      state.selectedOrganization = action.data;
      state.editingOrganization = false;
      break;
    case UPDATE_ORGANIZATION_ERROR:
      state.savingOrganization = false;
      state.updateOrganizationError = action.data;
      break;
    case GET_LOGIN_APPS_FOR_ORG_REQUEST:
      state.fetchingLoginApps = true;
      state.loginAppsFetchError = '';
      break;
    case GET_LOGIN_APPS_FOR_ORG_SUCCESS:
      state.fetchingLoginApps = false;
      state.loginAppsForOrg = action.data;
      break;
    case GET_LOGIN_APPS_FOR_ORG_ERROR:
      state.fetchingLoginApps = false;
      state.loginAppsFetchError = action.data;
      break;
    default:
      break;
  }
  return state;
};

export default orgsReducer;
