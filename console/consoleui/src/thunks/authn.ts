import { APIError } from '@userclouds/sharedui';

import {
  getPlexConfigRequest,
  getPlexConfigSuccess,
  getPlexConfigError,
  updatePlexConfigRequest,
  updatePlexConfigSuccess,
  updatePlexConfigError,
  getPageParametersRequest,
  getPageParametersSuccess,
  getPageParametersError,
  getEmailMessageElementsRequest,
  getEmailMessageElementsSuccess,
  getEmailMessageElementsError,
  getSMSMessageElementsRequest,
  getSMSMessageElementsSuccess,
  getSMSMessageElementsError,
} from '../actions/authn';
import {
  forceFetchTenantPlexConfig,
  saveTenantPlexConfig,
  fetchAppPageParameters,
  fetchTenantEmailMessageElements,
  fetchTenantSMSMessageElements,
} from '../API/authn';
import { AppDispatch } from '../store';
import TenantPlexConfig, {
  UpdatePlexConfigReason,
} from '../models/TenantPlexConfig';
import { PageParametersResponse } from '../models/PageParameters';
import { TenantAppMessageElements } from '../models/MessageElements';
import { postSuccessToast } from './notifications';

export const fetchPlexConfig =
  (tenantID: string) => (dispatch: AppDispatch) => {
    dispatch(getPlexConfigRequest());
    forceFetchTenantPlexConfig(tenantID).then(
      (config: TenantPlexConfig) => {
        dispatch(getPlexConfigSuccess(config));
      },
      (error: APIError) => {
        dispatch(getPlexConfigError(error));
      }
    );
  };

export const savePlexConfig =
  (
    tenantID: string,
    modifiedConfig: TenantPlexConfig,
    reason: UpdatePlexConfigReason | undefined,
    toast: boolean = true
  ) =>
  async (dispatch: AppDispatch) => {
    if (!modifiedConfig) {
      return;
    }

    dispatch(updatePlexConfigRequest());
    saveTenantPlexConfig(tenantID, modifiedConfig).then(
      (plexConfig: TenantPlexConfig) => {
        dispatch(updatePlexConfigSuccess(plexConfig, reason));
        if (toast) {
          dispatch(postSuccessToast('Successfully updated.'));
        }
      },
      (error: APIError) => {
        dispatch(updatePlexConfigError(error));
      }
    );
  };

export const fetchPageParams =
  (tenantID: string, appID: string) => (dispatch: AppDispatch) => {
    dispatch(getPageParametersRequest());
    fetchAppPageParameters(tenantID, appID).then(
      (response: PageParametersResponse) => {
        dispatch(getPageParametersSuccess(response));
      },
      (error: APIError) => {
        dispatch(getPageParametersError(error));
      }
    );
  };

export const fetchEmailMessageElements =
  (tenantID: string) => async (dispatch: AppDispatch) => {
    if (!tenantID) {
      return;
    }

    dispatch(getEmailMessageElementsRequest());
    fetchTenantEmailMessageElements(tenantID).then(
      (resp: TenantAppMessageElements) => {
        dispatch(
          getEmailMessageElementsSuccess(
            resp.tenant_app_message_elements.app_message_elements
          )
        );
      },
      (error: APIError) => {
        dispatch(getEmailMessageElementsError(error));
      }
    );
  };

export const fetchSMSMessageElements =
  (tenantID: string) => async (dispatch: AppDispatch) => {
    if (!tenantID) {
      return;
    }

    dispatch(getSMSMessageElementsRequest());
    fetchTenantSMSMessageElements(tenantID).then(
      (resp: TenantAppMessageElements) => {
        dispatch(
          getSMSMessageElementsSuccess(
            resp.tenant_app_message_elements.app_message_elements
          )
        );
      },
      (error: APIError) => {
        dispatch(getSMSMessageElementsError(error));
      }
    );
  };
