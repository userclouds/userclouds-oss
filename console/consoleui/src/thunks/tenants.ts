import { APIError } from '@userclouds/sharedui';

import { AppDispatch } from '../store';
import { redirect } from '../routing';
import Tenant, { SelectedTenant, TenantState } from '../models/Tenant';
import TenantURL from '../models/TenantURL';
import {
  getLastViewedTenant,
  setLastViewedTenant,
  removeLastViewedTenant,
} from '../util/localStorageTenant';
import {
  getSelectedTenantRequest,
  getSelectedTenantSuccess,
  getSelectedTenantError,
  getTenantsForCompanyRequest,
  getTenantsForCompanySuccess,
  getTenantsForCompanyError,
  updateTenantRequest,
  updateTenantSuccess,
  updateTenantError,
  deleteTenantRequest,
  deleteTenantSuccess,
  deleteTenantError,
  getTenantURLsRequest,
  getTenantURLsSuccess,
  getTenantURLsError,
  createTenantURLRequest,
  createTenantURLSuccess,
  createTenantURLError,
  updateTenantURLRequest,
  updateTenantURLSuccess,
  updateTenantURLError,
  deleteTenantURLRequest,
  deleteTenantURLSuccess,
  deleteTenantURLError,
} from '../actions/tenants';
import {
  fetchTenantsInCompany,
  fetchTenant,
  updateTenant,
  deleteTenant,
  fetchTenantURLs,
  createTenantURL,
  updateTenantURL,
  deleteTenantURL,
} from '../API/tenants';
import { postSuccessToast } from './notifications';

export const fetchTenants =
  (
    companyID: string,
    selectedTenantID: string | undefined,
    pathname: string,
    queryParams: URLSearchParams
  ) =>
  (dispatch: AppDispatch) => {
    dispatch(getTenantsForCompanyRequest());
    fetchTenantsInCompany(companyID).then(
      (tenants: Tenant[]) => {
        if (pathname !== '/tenants/create') {
          // Make sure the selected tenant ID and company ID are always in the
          // search query string, and that the query tenant ID is valid

          const tenantQueryID = queryParams.get('tenant_id') || undefined;
          const companyQueryID = queryParams.get('company_id') || undefined;

          if (
            selectedTenantID !== tenantQueryID ||
            (!selectedTenantID && !tenantQueryID)
          ) {
            selectedTenantID = undefined;
            if (tenants.length) {
              // First try to use the tenant from the URL
              let foundTenant = tenants.find(
                (t: Tenant) => t.id === tenantQueryID
              );

              // If no tenant in URL, try to use the last viewed tenant
              if (!foundTenant && !tenantQueryID) {
                const lastViewedTenantID = getLastViewedTenant(companyID);
                foundTenant = tenants.find(
                  (t: Tenant) => t.id === lastViewedTenantID
                );
              }

              // Fall back to first tenant if neither exists
              selectedTenantID = foundTenant?.id || tenants[0].id;
            }
          }

          if (
            (tenantQueryID && selectedTenantID !== tenantQueryID) ||
            (!tenantQueryID && selectedTenantID) ||
            !companyQueryID
          ) {
            const redirectQuery = new URLSearchParams(queryParams);
            if (companyID) {
              redirectQuery.set('company_id', companyID);
            } else {
              redirectQuery.delete('company_id');
            }
            if (selectedTenantID) {
              redirectQuery.set('tenant_id', selectedTenantID);
              // Update last viewed tenant when redirecting to a different tenant
              setLastViewedTenant(companyID, selectedTenantID);
            } else {
              redirectQuery.delete('tenant_id');
            }
            redirect(`${pathname}?${redirectQuery.toString()}`, true);
          } else if (selectedTenantID) {
            // Store the selected tenant ID in localStorage
            setLastViewedTenant(companyID, selectedTenantID);

            dispatch(getSelectedTenantRequest());
            fetchTenant(companyID, selectedTenantID)
              .then((selectedTenant: SelectedTenant) => {
                dispatch(getSelectedTenantSuccess(selectedTenant));
              })
              .catch((error: APIError) => {
                dispatch(getSelectedTenantError(error));
              });
          }
        }

        dispatch(getTenantsForCompanySuccess(companyID, tenants));
      },
      (error: APIError) => {
        dispatch(getTenantsForCompanyError(error));
      }
    );
  };

export const saveTenant =
  (modifiedTenant: Tenant, tenant: SelectedTenant) =>
  async (dispatch: AppDispatch): Promise<any> => {
    dispatch(updateTenantRequest());
    return updateTenant(modifiedTenant.company_id, {
      id: modifiedTenant.id,
      name: modifiedTenant.name,
      company_id: modifiedTenant.company_id,
      use_organizations: modifiedTenant.use_organizations !== null,
      state: TenantState.ACTIVE, // always active if we're updating it here
      is_admin: tenant.is_admin,
      is_member: tenant.is_member,
    }).then(
      (savedTenant: Tenant) => {
        dispatch(updateTenantSuccess(savedTenant));
      },
      (error: APIError) => {
        dispatch(updateTenantError(error));
      }
    );
  };

export const handleDeleteTenant =
  (companyID: string, tenantId: string) => async (dispatch: AppDispatch) => {
    dispatch(deleteTenantRequest(tenantId));

    // Remove from localStorage if it's the tenant being deleted
    const lastViewedTenant = getLastViewedTenant(companyID);
    if (lastViewedTenant === tenantId) {
      removeLastViewedTenant(companyID);
    }

    deleteTenant(companyID, tenantId).then(
      () => {
        dispatch(deleteTenantSuccess(tenantId));
        dispatch(fetchTenants(companyID, undefined, '', new URLSearchParams()));
      },
      (error: APIError) => {
        dispatch(deleteTenantError(error));
      }
    );
  };

export const getTenantURLs =
  (tenantID: string) => async (dispatch: AppDispatch) => {
    dispatch(getTenantURLsRequest());
    fetchTenantURLs(tenantID).then(
      (urls: TenantURL[]) => {
        dispatch(getTenantURLsSuccess(urls));
      },
      (error: APIError) => {
        dispatch(getTenantURLsError(error));
      }
    );
  };

export const createNewTenantURL =
  (tenantURL: TenantURL, onSave?: Function) =>
  async (dispatch: AppDispatch) => {
    dispatch(createTenantURLRequest());
    createTenantURL(tenantURL).then(
      (url: TenantURL) => {
        dispatch(createTenantURLSuccess(url));

        const tid = url.tenant_id;
        setTimeout(() => {
          dispatch(getTenantURLs(tid));
        }, 5000);
        onSave && onSave();
      },
      (error: APIError) => {
        dispatch(createTenantURLError(error));
      }
    );
  };

export const saveTenantURL =
  (tenantURL: TenantURL, onSuccess?: Function) =>
  async (dispatch: AppDispatch) => {
    dispatch(updateTenantURLRequest());
    updateTenantURL(tenantURL).then(
      (url: TenantURL) => {
        dispatch(updateTenantURLSuccess(url));

        const tid = url.tenant_id;
        setTimeout(() => {
          dispatch(getTenantURLs(tid));
        }, 5000);
        onSuccess && onSuccess();
      },
      (error) => {
        dispatch(updateTenantURLError(error));
      }
    );
  };

export const handleDeleteTenantURL =
  (
    tenantID: string,
    val: string,
    confirmation: boolean = true,
    toast: boolean = true,
    onSuccess?: Function
  ) =>
  (dispatch: AppDispatch) => {
    if (
      !confirmation ||
      window.confirm(
        'Are you sure you want to delete this URL? This cannot be undone.'
      )
    ) {
      dispatch(deleteTenantURLRequest());
      deleteTenantURL(tenantID, val).then(
        () => {
          dispatch(deleteTenantURLSuccess(val));
          onSuccess && onSuccess();
          toast && dispatch(postSuccessToast('Successfully deleted URL.'));
        },
        (err: APIError) => {
          dispatch(deleteTenantURLError(err));
        }
      );
    }
  };

export const getBatchModifyTenantURLPromises = (
  tenantID: string,
  createURLs: TenantURL[],
  updateURLs: TenantURL[],
  deleteURLs: string[]
) => {
  const promises: Promise<any>[] = [];
  if (createURLs.length > 0) {
    for (const url of createURLs) {
      promises.push(createTenantURL(url));
    }
  }
  if (updateURLs.length > 0) {
    for (const url of updateURLs) {
      promises.push(updateTenantURL(url));
    }
  }
  if (deleteURLs.length > 0) {
    for (const urlID of deleteURLs) {
      promises.push(deleteTenantURL(tenantID, urlID));
    }
  }
  return promises;
};
