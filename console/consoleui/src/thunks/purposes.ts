import { APIError } from '@userclouds/sharedui';

import { AppDispatch, RootState } from '../store';
import PaginatedResult from '../models/PaginatedResult';
import Purpose from '../models/Purpose';
import {
  getPurposesRequest,
  getPurposesSuccess,
  getPurposesError,
  createPurposeRequest,
  createPurposeSuccess,
  createPurposeError,
  updatePurposeRequest,
  updatePurposeSuccess,
  updatePurposeError,
  deleteSinglePurposeRequest,
  deleteSinglePurposeSuccess,
  deleteSinglePurposeError,
  getPurposeRequest,
  getPurposeSuccess,
  getPurposeError,
  bulkDeletePurposesRequest,
  bulkDeletePurposesSuccess,
  bulkDeletePurposesFailure,
  deletePurposesSingleSuccess,
  deletePurposesSingleError,
} from '../actions/purposes';

import {
  fetchTenantPurposes,
  createTenantPurpose,
  updateTenantPurpose,
  deleteTenantPurpose,
  fetchTenantPurpose,
} from '../API/purposes';

import { postAlertToast, postSuccessToast } from './notifications';
import { redirect } from '../routing';
import { PAGINATION_API_VERSION } from '../API';

const PURPOSES_PAGE_SIZE = '20';
export const fetchPurposes =
  (tenantID: string, params: URLSearchParams) => (dispatch: AppDispatch) => {
    const paramsAsObject = [
      'purposes_starting_after',
      'purposes_ending_before',
      'purposes_limit',
      'purposes_filter',
    ].reduce((acc: Record<string, string>, paramName: string) => {
      if (params.has(paramName)) {
        acc[paramName.substring(9)] = params.get(paramName) as string;
      }
      return acc;
    }, {});
    if (!paramsAsObject.limit) {
      paramsAsObject.limit = PURPOSES_PAGE_SIZE;
    }
    if (!paramsAsObject.sort_order) {
      paramsAsObject.sort_order = 'ascending';
    }
    if (!paramsAsObject.sort_key) {
      paramsAsObject.sort_key = 'name,id';
    }
    if (!paramsAsObject.version) {
      paramsAsObject.version = PAGINATION_API_VERSION;
    }

    dispatch(getPurposesRequest());
    fetchTenantPurposes(tenantID, paramsAsObject).then(
      (result: PaginatedResult<Purpose>) => {
        dispatch(getPurposesSuccess(result));
      },
      (error: APIError) => {
        dispatch(getPurposesError(error));
      }
    );
  };

export const fetchPurpose =
  (tenantID: string, purposeID: string) => (dispatch: AppDispatch) => {
    dispatch(getPurposeRequest());
    fetchTenantPurpose(tenantID, purposeID).then(
      (purpose: Purpose) => {
        dispatch(getPurposeSuccess(purpose));
      },
      (error: APIError) => {
        dispatch(getPurposeError(error));
      }
    );
  };

export const savePurpose =
  (companyID: string, tenantID: string, purpose: Purpose, isNew: boolean) =>
  (dispatch: AppDispatch) => {
    const request = isNew ? createPurposeRequest : updatePurposeRequest;
    const success = isNew ? createPurposeSuccess : updatePurposeSuccess;
    const error = isNew ? createPurposeError : updatePurposeError;
    dispatch(request());
    (isNew
      ? createTenantPurpose(tenantID, purpose)
      : updateTenantPurpose(tenantID, purpose.id, purpose)
    ).then(
      (savedPurpose: Purpose) => {
        dispatch(success(savedPurpose));
        if (isNew) {
          dispatch(postSuccessToast('Successfully created purpose'));
          redirect(
            `/purposes/${savedPurpose.id}?company_id=${companyID}&tenant_id=${tenantID}`
          );
        }
      },
      (err: APIError) => {
        dispatch(error(err));
      }
    );
  };

export const deleteSinglePurpose =
  (companyID: string, tenantID: string, purposeID: string) =>
  (dispatch: AppDispatch) => {
    dispatch(deleteSinglePurposeRequest());
    deleteTenantPurpose(tenantID, purposeID).then(
      () => {
        dispatch(deleteSinglePurposeSuccess());
        dispatch(postSuccessToast('Successfully deleted purpose'));
        redirect(`/purposes?company_id=${companyID}&tenant_id=${tenantID}`);
      },
      (error: APIError) => {
        dispatch(deleteSinglePurposeError(error));
        dispatch(postAlertToast('Error deleting purpose: ' + error));
      }
    );
  };

export const deletePurposeBulk =
  (tenantID: string, purposeID: string) =>
  (dispatch: AppDispatch): Promise<boolean> => {
    return new Promise((resolve) => {
      return deleteTenantPurpose(tenantID, purposeID).then(
        () => {
          dispatch(deletePurposesSingleSuccess(purposeID));
          resolve(true);
        },
        (error: APIError) => {
          dispatch(deletePurposesSingleError(purposeID, error));
          resolve(false);
        }
      );
    });
  };

export const deletePurposes =
  (selectedTenantID: string, purposeIDs: string[]) =>
  (dispatch: AppDispatch, getState: () => RootState) => {
    const { query } = getState();

    dispatch(bulkDeletePurposesRequest());
    const reqs: Array<Promise<boolean>> = [];
    purposeIDs.forEach((id) => {
      reqs.push(dispatch(deletePurposeBulk(selectedTenantID, id)));
    });
    Promise.all(reqs).then((values: boolean[]) => {
      if (values.every((val) => val === true)) {
        dispatch(fetchPurposes(selectedTenantID, query));
        dispatch(bulkDeletePurposesSuccess());
      } else {
        if (!values.every((val) => val === false)) {
          // if all the reqs failed, there's no need to re-fetch
          dispatch(fetchPurposes(selectedTenantID, query));
        }
        dispatch(bulkDeletePurposesFailure());
      }
    });
  };
