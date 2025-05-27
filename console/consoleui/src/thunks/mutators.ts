import { APIError } from '@userclouds/sharedui';
import {
  createMutatorRequest,
  createMutatorSuccess,
  createMutatorError,
  getMutatorRequest,
  getMutatorSuccess,
  getMutatorError,
  updateMutatorError,
  updateMutatorRequest,
  updateMutatorSuccess,
} from '../actions/mutators';
import {
  createTenantMutator,
  fetchTenantMutator,
  updateTenantMutator,
} from '../API/mutators';
import Mutator, { MutatorColumn, MutatorSavePayload } from '../models/Mutator';
import { redirect } from '../routing';
import { AppDispatch, RootState } from '../store';
import { postSuccessToast } from './notifications';

export const handleCreateMutator =
  () => (dispatch: AppDispatch, getState: () => RootState) => {
    const {
      selectedCompanyID,
      selectedTenantID,
      mutatorToCreate,
      modifiedAccessPolicy,
    } = getState();
    if (selectedTenantID && selectedCompanyID) {
      mutatorToCreate.composed_access_policy = modifiedAccessPolicy;
      dispatch(
        createMutator(selectedTenantID, selectedCompanyID, mutatorToCreate)
      );
    }
  };

export const createMutator =
  (
    selectedTenantID: string,
    selectedCompanyID: string,
    mutatorToCreate: MutatorSavePayload
  ) =>
  (dispatch: AppDispatch) => {
    if (selectedTenantID) {
      dispatch(createMutatorRequest);
      createTenantMutator(selectedTenantID, mutatorToCreate).then(
        (response: Mutator) => {
          dispatch(createMutatorSuccess(response));
          dispatch(postSuccessToast('Successfully created mutator'));
          redirect(
            `/mutators/${response.id}/latest?company_id=${
              selectedCompanyID as string
            }&tenant_id=${selectedTenantID}`
          );
        },
        (error: APIError) => {
          dispatch(createMutatorError(error));
          document.getElementById('aboutMutator')?.scrollIntoView();
        }
      );
    }
  };

export const fetchMutator =
  (tenantID: string, mutatorID: string, version: string) =>
  (dispatch: AppDispatch) => {
    dispatch(getMutatorRequest(mutatorID));
    return fetchTenantMutator(
      tenantID,
      mutatorID,
      !isNaN(parseInt(version, 10)) ? version : undefined
    ).then(
      (mutator: Mutator) => {
        dispatch(getMutatorSuccess(mutator));
      },
      (error: APIError) => {
        dispatch(getMutatorError(error));
      }
    );
  };

export const saveMutatorDetails =
  (tenantID: string, mutatorToUpdate: Mutator) =>
  (dispatch: AppDispatch, getState: () => RootState): Promise<void> => {
    const {
      selectedCompanyID,
      modifiedAccessPolicy,
      mutatorColumnsToAdd,
      mutatorColumnsToDelete,
    } = getState();

    const mutator: MutatorSavePayload = {
      id: mutatorToUpdate.id,
      name: mutatorToUpdate.name?.trim(),
      description: mutatorToUpdate.description?.trim(),
      columns: mutatorToUpdate.columns,
      access_policy_id: mutatorToUpdate.access_policy.id,
      selector_config: mutatorToUpdate.selector_config,
    };
    mutator.composed_access_policy = modifiedAccessPolicy;
    const newColumns = mutator.columns
      .filter(
        (col: MutatorColumn) => !mutatorColumnsToDelete.hasOwnProperty(col.id)
      )
      .concat(mutatorColumnsToAdd);
    mutator.columns = newColumns;
    dispatch(updateMutatorRequest);
    return updateTenantMutator(tenantID, mutator).then(
      (resp: Mutator) => {
        dispatch(updateMutatorSuccess(resp));
        dispatch(postSuccessToast('Successfully updated mutator'));
        redirect(
          `/mutators/${resp.id}/${resp.version}?company_id=${
            selectedCompanyID as string
          }&tenant_id=${tenantID}`
        );
      },
      (error: APIError) => {
        dispatch(updateMutatorError(error));
      }
    );
  };
