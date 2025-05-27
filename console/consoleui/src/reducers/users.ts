import { AnyAction } from 'redux';
import { RootState } from '../store';
import actions from '../actions';
import {
  GET_CURRENT_TENANT_USER_REQUEST,
  GET_CURRENT_TENANT_USER_SUCCESS,
  GET_CURRENT_TENANT_USER_ERROR,
  GET_CURRENT_TENANT_USER_CONSENTED_PURPOSES_REQUEST,
  GET_CURRENT_TENANT_USER_CONSENTED_PURPOSES_SUCCESS,
  GET_CURRENT_TENANT_USER_CONSENTED_PURPOSES_ERROR,
  GET_CURRENT_TENANT_USER_EVENTS_REQUEST,
  GET_CURRENT_TENANT_USER_EVENTS_SUCCESS,
  GET_CURRENT_TENANT_USER_EVENTS_ERROR,
  TOGGLE_USER_EDIT_MODE,
  MODIFY_TENANT_USER_PROFILE,
  SAVE_TENANT_USER_REQUEST,
  SAVE_TENANT_USER_SUCCESS,
  SAVE_TENANT_USER_ERROR,
} from '../actions/users';

const userReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case actions.GET_TENANT_ORGS_REQUEST:
      state.tenantUsersSelectedOrganizationID = undefined;
      state.tenantUsers = undefined;
      state.fetchingOrganizations = true;
      break;
    case actions.GET_TENANT_ORGS_SUCCESS: {
      state.tenantUsersOrganizations = action.data;
      if (!state.tenantUsersSelectedOrganizationID) {
        state.tenantUsersSelectedOrganizationID = '';
      }
      state.fetchingOrganizations = false;
      break;
    }
    case actions.GET_TENANT_ORGS_ERROR:
      state.fetchingOrganizations = false;
      break;
    case actions.SELECT_USERS_TENANT_ORGANIZATION:
      state.fetchUsersError = '';
      state.tenantUsers = undefined;
      state.tenantUsersSelectedOrganizationID = action.data;
      break;
    case actions.GET_TENANT_USERS_REQUEST:
      state.tenantUsers = undefined;
      state.fetchingUsers = true;
      break;
    case actions.GET_TENANT_USERS_SUCCESS:
      state.fetchingUsers = false;
      state.tenantUsers = action.data;
      state.fetchUsersError = '';
      break;
    case actions.GET_TENANT_USERS_ERROR:
      state.fetchingUsers = false;
      state.fetchUsersError = action.data;
      break;
    case actions.TOGGLE_USER_FOR_DELETE:
      if (state.userDeleteQueue.includes(action.data)) {
        state.userDeleteQueue = state.userDeleteQueue.filter(
          (id: string) => id !== action.data
        );
      } else {
        state.userDeleteQueue = [...state.userDeleteQueue, action.data];
      }
      break;
    case actions.DELETE_USER_REQUEST:
      // on users index, we care about the set of bulk requests
      // not the start of each individual req, so this is intentionally
      // a no-op
      break;
    case actions.DELETE_USER_SUCCESS: {
      if (state.tenantUsers) {
        state.tenantUsers.data = [
          ...state.tenantUsers.data.filter((user) => user.id !== action.data),
        ];
        state.userDeleteQueue = [
          ...state.userDeleteQueue.filter((id) => id !== action.data),
        ];
      }
      break;
    }
    case actions.DELETE_USER_ERROR:
      state.userBulkSaveErrors = [...state.userBulkSaveErrors, action.data];
      break;
    case actions.BULK_UPDATE_USERS_START:
      state.savingUsers = true;
      break;
    case actions.BULK_UPDATE_USERS_END:
      state.savingUsers = false;
      break;
    case GET_CURRENT_TENANT_USER_REQUEST:
      state.currentTenantUserError = '';
      break;
    case GET_CURRENT_TENANT_USER_SUCCESS:
      state.currentTenantUser = { ...action.data };
      state.currentTenantUserError = '';
      break;
    case GET_CURRENT_TENANT_USER_ERROR:
      state.currentTenantUser = undefined;
      state.currentTenantUserError = action.data;
      break;
    case GET_CURRENT_TENANT_USER_CONSENTED_PURPOSES_REQUEST:
      state.currentTenantUserConsentedPurposes = undefined;
      state.currentTenantUserConsentedPurposesError = '';
      break;
    case GET_CURRENT_TENANT_USER_CONSENTED_PURPOSES_SUCCESS:
      state.currentTenantUserConsentedPurposes = [...action.data];
      state.currentTenantUserConsentedPurposesError = '';
      break;
    case GET_CURRENT_TENANT_USER_CONSENTED_PURPOSES_ERROR:
      state.currentTenantUserConsentedPurposes = undefined;
      state.currentTenantUserConsentedPurposesError = action.data;
      break;
    case GET_CURRENT_TENANT_USER_EVENTS_REQUEST:
      state.currentTenantUserEvents = undefined;
      state.currentTenantUserEventsError = '';
      break;
    case GET_CURRENT_TENANT_USER_EVENTS_SUCCESS:
      state.currentTenantUserEvents = [...action.data];
      state.currentTenantUserEventsError = '';
      break;
    case GET_CURRENT_TENANT_USER_EVENTS_ERROR:
      state.currentTenantUserEvents = undefined;
      state.currentTenantUserEventsError = action.data;
      break;
    case TOGGLE_USER_EDIT_MODE: {
      const editMode =
        action.data !== undefined ? action.data : !state.userEditMode;
      state.userEditMode = editMode;
      if (state.userEditMode) {
        state.currentTenantUserProfileEdited = {
          ...state.currentTenantUser?.profile,
        };
      } else {
        state.saveTenantUserError = '';
      }
      break;
    }
    case MODIFY_TENANT_USER_PROFILE:
      state.currentTenantUserProfileEdited = {
        ...state.currentTenantUserProfileEdited,
        ...action.data,
      };
      break;
    case SAVE_TENANT_USER_REQUEST:
      state.saveTenantUserError = '';
      break;
    case SAVE_TENANT_USER_SUCCESS: {
      if (state.currentTenantUser) {
        state.currentTenantUser.profile = {
          ...state.currentTenantUserProfileEdited,
        };
        state.currentTenantUser = { ...state.currentTenantUser };
      }
      state.userEditMode = false;
      break;
    }
    case SAVE_TENANT_USER_ERROR:
      state.saveTenantUserError = action.data;
      break;
    default:
      break;
  }
  return state;
};

export default userReducer;
