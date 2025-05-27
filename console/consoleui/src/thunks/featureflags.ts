import { APIError } from '@userclouds/sharedui';

import { AppDispatch } from '../store';
import { fetchFeatureFlagsForUser } from '../API/featureflags';
import {
  getFeatureFlagsRequest,
  getFeatureFlagsSuccess,
  getFeatureFlagsError,
} from '../actions/featureflags';

export const fetchFeatureFlags =
  (userID: string, companyID: string, tenantID: string) =>
  (dispatch: AppDispatch) => {
    dispatch(getFeatureFlagsRequest());
    fetchFeatureFlagsForUser(userID, companyID, tenantID).then(
      (response) => {
        dispatch(getFeatureFlagsSuccess(response));
      },
      (error: APIError) => {
        dispatch(getFeatureFlagsError(error));
      }
    );
  };
