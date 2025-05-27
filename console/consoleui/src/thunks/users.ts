import { APIError } from '@userclouds/sharedui';
import { AppDispatch } from '../store';
import {
  getCurrentTenantUserRequest,
  getCurrentTenantUserSuccess,
  getCurrentTenantUserError,
  getCurrentTenantUserConsentedPurposesRequest,
  getCurrentTenantUserConsentedPurposesSuccess,
  getCurrentTenantUserConsentedPurposesError,
  getCurrentTenantUserEventsRequest,
  getCurrentTenantUserEventsSuccess,
  getCurrentTenantUserEventsError,
  saveTenantUserRequest,
  saveTenantUserSuccess,
  saveTenantUserError,
} from '../actions/users';
import {
  fetchTenantUser,
  fetchTenantUserConsentedPurposes,
  fetchTenantUserEvents,
} from '../API/users';
import { UserProfile, UserProfileSerialized } from '../models/UserProfile';
import API from '../API';

export const fetchUser =
  (tenantID: string, userID: string) => (dispatch: AppDispatch) => {
    dispatch(getCurrentTenantUserRequest());
    fetchTenantUser(tenantID, userID).then(
      (user: UserProfileSerialized) => {
        dispatch(getCurrentTenantUserSuccess(user));
      },
      (error: Error) => {
        dispatch(getCurrentTenantUserError(error.message));
      }
    );
  };

export const fetchUserConsentedPurposes =
  (tenantID: string, userID: string) => (dispatch: AppDispatch) => {
    dispatch(getCurrentTenantUserConsentedPurposesRequest());
    fetchTenantUserConsentedPurposes(tenantID, userID).then(
      (purposes: Array<object>) => {
        dispatch(getCurrentTenantUserConsentedPurposesSuccess(purposes));
      },
      (error: Error) => {
        dispatch(getCurrentTenantUserConsentedPurposesError(error.message));
      }
    );
  };

export const fetchUserEvents =
  (tenantID: string, userID: string) => (dispatch: AppDispatch) => {
    dispatch(getCurrentTenantUserEventsRequest());
    fetchTenantUserEvents(tenantID, userID).then(
      (events: Array<object>) => {
        dispatch(getCurrentTenantUserEventsSuccess(events));
      },
      (error: Error) => {
        dispatch(getCurrentTenantUserEventsError(error.message));
      }
    );
  };

export const saveUser =
  (tenantID: string, user: UserProfile) => (dispatch: AppDispatch) => {
    dispatch(saveTenantUserRequest());

    API.updateUser(tenantID, user).then((result: UserProfile | APIError) => {
      if (result instanceof APIError) {
        dispatch(saveTenantUserError(result.message));
      } else {
        dispatch(
          saveTenantUserSuccess(result.toJSON() as UserProfileSerialized)
        );
      }
    });
  };
