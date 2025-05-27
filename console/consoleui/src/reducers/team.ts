import { AnyAction } from 'redux';
import { RootState } from '../store';
import actions from '../actions';
import { Roles, UserRoles } from '../models/UserRoles';
import { getNewToggleEditValue } from './reducerHelper';

const teamReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case actions.GET_USERROLES_REQUEST:
      state.fetchingUserRoles = true;
      state.fetchUserRolesError = '';
      break;
    case actions.GET_USERROLES_SUCCESS:
      state.fetchingUserRoles = false;
      state.teamUserRoles = action.data.map((u: UserRoles) => {
        if (u.organization_role === Roles.AdminRole) {
          u.policy_role = Roles.UserGroupPolicyFullRole;
        }
        return u;
      });
      state.fetchUserRolesError = '';
      break;
    case actions.GET_USERROLES_ERROR:
      state.fetchingUserRoles = false;
      state.fetchUserRolesError = action.data;
      break;
    case actions.BULK_UPDATE_USERROLES_START:
      state.savingUserRoles = true;
      break;
    case actions.BULK_UPDATE_USERROLES_END:
      if (action.data === true) {
        // if all requests succeeded
        // exit edit mode
        state.userRolesEditMode = false;
      }
      state.savingUserRoles = false;
      break;
    case actions.MODIFY_USER_ROLE:
      {
        const matchingRole =
          state.modifiedUserRoles.find((m) => m.id === action.data.id) ||
          state.teamUserRoles!.find((m) => m.id === action.data.id);
        if (matchingRole) {
          const updatedRoles = {
            ...matchingRole,
          };
          if (action.data.organization_role) {
            updatedRoles.organization_role = action.data.organization_role;
            if (updatedRoles.organization_role === Roles.NoRole) {
              updatedRoles.policy_role = Roles.UserGroupPolicyNoRole;
            } else if (updatedRoles.organization_role === Roles.AdminRole) {
              updatedRoles.policy_role = Roles.UserGroupPolicyFullRole;
            }
          } else if (action.data.policy_role) {
            updatedRoles.policy_role = action.data.policy_role;
          }

          const otherUserRoles = state.modifiedUserRoles.filter(
            (u: UserRoles) => u.id !== action.data.id
          );
          state.modifiedUserRoles = [updatedRoles, ...otherUserRoles];
        }
      }
      break;
    case actions.UPDATE_USERROLES_REQUEST:
      // on team home page, we care about the set of bulk requests
      // not the start of each individual req, so this is intentionally
      // a no-op
      break;
    case actions.UPDATE_USERROLES_SUCCESS:
      {
        const userRoles: UserRoles = action.data;
        state.modifiedUserRoles = state.modifiedUserRoles.filter(
          (u: UserRoles) => u.id !== userRoles.id
        );
      }
      break;
    case actions.UPDATE_USERROLES_ERROR:
      state.userRolesBulkSaveErrors = [
        ...state.userRolesBulkSaveErrors,
        action.data,
      ];
      break;
    case actions.TOGGLE_USERROLES_EDIT_MODE:
      state.userRolesEditMode = getNewToggleEditValue(
        action.data,
        state.userRolesEditMode
      );
      state.userRolesBulkSaveErrors = [];
      state.modifiedUserRoles = [];
      break;
    case actions.GET_COMPANY_USERROLES_REQUEST:
      state.fetchingCompanyUserRoles = true;
      state.fetchCompanyUserRolesError = '';
      break;
    case actions.GET_COMPANY_USERROLES_SUCCESS:
      state.fetchingCompanyUserRoles = false;
      state.companyUserRoles = action.data;
      state.fetchCompanyUserRolesError = '';
      break;
    case actions.GET_COMPANY_USERROLES_ERROR:
      state.fetchingCompanyUserRoles = false;
      state.fetchCompanyUserRolesError = action.data;
      break;
    case actions.BULK_UPDATE_COMPANY_USERROLES_START:
      state.savingCompanyUserRoles = true;
      break;
    case actions.BULK_UPDATE_COMPANY_USERROLES_END:
      if (action.data === true) {
        // if all requests succeeded
        // exit edit mode
        state.companyUserRolesEditMode = false;
      }
      state.savingCompanyUserRoles = false;
      break;
    case actions.DELETE_COMPANY_USERROLES_REQUEST:
      // on team home page, we care about the set of bulk requests
      // not the start of each individual req, so this is intentionally
      // a no-op
      break;
    case actions.DELETE_COMPANY_USERROLES_SUCCESS:
      state.companyUserRolesDeleteQueue = [
        ...state.companyUserRolesDeleteQueue.filter((id) => id !== action.data),
      ];
      break;
    case actions.DELETE_COMPANY_USERROLES_ERROR:
      state.companyUserRolesBulkSaveErrors = [
        ...state.companyUserRolesBulkSaveErrors,
        action.data,
      ];
      break;
    case actions.MODIFY_COMPANY_USER_ROLE:
      {
        const matchingRole =
          state.modifiedCompanyUserRoles.find((m) => m.id === action.data.id) ||
          state.companyUserRoles!.find((m) => m.id === action.data.id);
        if (matchingRole) {
          const updatedRoles = {
            ...matchingRole,
          };
          updatedRoles.organization_role = action.data.organization_role;
          const otherUserRoles = state.modifiedCompanyUserRoles.filter(
            (u: UserRoles) => u.id !== action.data.id
          );
          state.modifiedCompanyUserRoles = [updatedRoles, ...otherUserRoles];
        }
      }
      break;
    case actions.UPDATE_COMPANY_USERROLES_REQUEST:
      // on team home page, we care about the set of bulk requests
      // not the start of each individual req, so this is intentionally
      // a no-op
      break;
    case actions.UPDATE_COMPANY_USERROLES_SUCCESS:
      {
        const userRoles: UserRoles = action.data;
        state.modifiedCompanyUserRoles = state.modifiedCompanyUserRoles.filter(
          (u: UserRoles) => u.id !== userRoles.id
        );
      }
      break;
    case actions.UPDATE_COMPANY_USERROLES_ERROR:
      state.companyUserRolesBulkSaveErrors = [
        ...state.companyUserRolesBulkSaveErrors,
        action.data,
      ];
      break;
    case actions.TOGGLE_COMPANY_USERROLES_FOR_DELETE:
      if (state.companyUserRolesDeleteQueue.includes(action.data)) {
        state.companyUserRolesDeleteQueue =
          state.companyUserRolesDeleteQueue.filter(
            (id: string) => id !== action.data
          );
      } else {
        state.companyUserRolesDeleteQueue = [
          ...state.companyUserRolesDeleteQueue,
          action.data,
        ];
      }
      break;
    case actions.TOGGLE_COMPANY_USERROLES_EDIT_MODE:
      state.companyUserRolesEditMode = getNewToggleEditValue(
        action.data,
        state.companyUserRolesEditMode
      );
      state.companyUserRolesBulkSaveErrors = [];
      state.companyUserRolesDeleteQueue = [];
      state.modifiedCompanyUserRoles = [];
      break;

    case actions.GET_COMPANY_INVITES_REQUEST:
      state.fetchingInvites = true;
      state.fetchInvitesError = '';
      break;
    case actions.GET_COMPANY_INVITES_SUCCESS:
      state.fetchingInvites = false;
      state.companyInvites = action.data;
      break;
    case actions.GET_COMPANY_INVITES_ERROR:
      state.fetchingInvites = false;
      state.fetchInvitesError = action.data;
      break;
    default:
      break;
  }
  return state;
};

export default teamReducer;
