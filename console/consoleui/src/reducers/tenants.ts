import { AnyAction } from 'redux';
import { v4 as uuidv4 } from 'uuid';
import {
  RootState,
  initialTenantPersistedState,
  initialTenantPageState,
} from '../store';
import {
  CREATE_TENANT_REQUEST,
  CREATE_TENANT_SUCCESS,
  CREATE_TENANT_ERROR,
  TOGGLE_EDIT_TENANT_MODE,
  MODIFY_TENANT_NAME,
  UPDATE_TENANT_REQUEST,
  UPDATE_TENANT_SUCCESS,
  UPDATE_TENANT_ERROR,
  DELETE_TENANT_REQUEST,
  DELETE_TENANT_SUCCESS,
  DELETE_TENANT_ERROR,
  GET_SELECTED_TENANT_REQUEST,
  GET_SELECTED_TENANT_SUCCESS,
  GET_SELECTED_TENANT_ERROR,
  GET_TENANTS_FOR_COMPANY_REQUEST,
  GET_TENANTS_FOR_COMPANY_SUCCESS,
  GET_TENANTS_FOR_COMPANY_ERROR,
  CREATE_TENANT_URL_REQUEST,
  CREATE_TENANT_URL_SUCCESS,
  CREATE_TENANT_URL_ERROR,
  UPDATE_TENANT_URL_REQUEST,
  UPDATE_TENANT_URL_SUCCESS,
  UPDATE_TENANT_URL_ERROR,
  GET_TENANT_URLS_REQUEST,
  GET_TENANT_URLS_SUCCESS,
  GET_TENANT_URLS_ERROR,
  ADD_TENANT_URL,
  DELETE_TENANT_URL_REQUEST,
  DELETE_TENANT_URL_SUCCESS,
  DELETE_TENANT_URL_ERROR,
  TOGGLE_TENANT_URL_EDIT_MODE,
  TOGGLE_CREATE_TENANT_DIALOG,
  TOGGLE_TENANT_URL_DIALOG_IS_OPEN,
  TOGGLE_TENANT_DATABASE_DIALOG_IS_OPEN,
  MODIFY_TENANT_URL,
  TOGGLE_TENANT_ISSUER_DIALOG_IS_OPEN,
  SET_CURRENT_URL,
  SET_CREATING_ISSUER,
  SET_TENANT_PROVIDER_NAME,
  SET_EDITING_ISSUER_INDEX,
  DELETE_TENANT_URL,
  UPDATE_TENANT_URL,
  CREATE_TENANT_URL,
  SET_CREATING_URL,
  UPDATE_TENANT_CREATION_STATE,
} from '../actions/tenants';
import Tenant, { SelectedTenant } from '../models/Tenant';
import TenantURL from '../models/TenantURL';
import { getNewToggleEditValue } from './reducerHelper';

const tenantsReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case CREATE_TENANT_REQUEST:
      state.creatingTenant = true;
      state.createTenantError = '';
      break;
    case CREATE_TENANT_SUCCESS: {
      state.creatingTenant = false;
      const newTenant: SelectedTenant = action.data;
      // Set the initial tenant creation state
      state.tenantCreationState = newTenant.state;
      // TODO: is it better to re-fetch than add? probably
      state.tenants = [newTenant, ...(state.tenants as Tenant[])];
      state.selectedTenantID = newTenant.id;
      Object.assign(state, initialTenantPersistedState, initialTenantPageState);
      state.selectedTenant = newTenant;
      state.createTenantDialogIsOpen = false;
      break;
    }
    case CREATE_TENANT_ERROR:
      state.creatingTenant = false;
      state.createTenantError = action.data;
      break;
    case UPDATE_TENANT_CREATION_STATE:
      state.tenantCreationState = action.data;
      break;
    case TOGGLE_EDIT_TENANT_MODE:
      state.editingTenant = getNewToggleEditValue(
        action.data,
        state.editingTenant
      );
      if (state.selectedTenant) {
        state.modifiedTenant = { ...state.selectedTenant };
        state.saveTenantError = '';
        state.saveTenantSuccess = '';
      }
      state.modifiedTenantUrls = state.tenantURLs ? [...state.tenantURLs] : [];
      state.tenantURLsToCreate = [];
      state.tenantURLsToUpdate = [];
      state.tenantURLsToDelete = [];

      state.modifiedSqlShimDatabases = state.sqlShimDatabases?.data?.length
        ? [...state.sqlShimDatabases.data]
        : [];
      break;
    case MODIFY_TENANT_NAME:
      if (state.modifiedTenant) {
        state.modifiedTenant = { ...state.modifiedTenant, name: action.data };
      }
      break;
    case UPDATE_TENANT_REQUEST:
      state.savingTenant = true;
      state.saveTenantSuccess = '';
      state.saveTenantError = '';
      break;
    case UPDATE_TENANT_SUCCESS: {
      state.savingTenant = false;
      const updatedTenant: SelectedTenant = action.data;
      // TODO: this feels hacky. We should operate on copies
      // or maybe just re-fetch tenants
      state.tenants = (state.tenants as Tenant[]).map((t: Tenant) =>
        updatedTenant.id === t.id ? updatedTenant : t
      );
      if (
        state.selectedTenant &&
        state.selectedTenant.id === updatedTenant.id
      ) {
        state.selectedTenant = updatedTenant;
      }
      state.saveTenantSuccess = `Successfully saved tenant: ${updatedTenant.name}`;
      state.editingTenant = false;
      break;
    }
    case UPDATE_TENANT_ERROR:
      state.savingTenant = false;
      state.saveTenantError = action.data;
      break;
    case DELETE_TENANT_REQUEST:
      // TODO: we may want to store the ID of the tenant being deleted
      state.deletingTenant = true;
      state.deleteTenantError = '';
      break;
    case DELETE_TENANT_SUCCESS:
      state.deletingTenant = false;
      // clear tenants to force a refetch
      // TODO: revisit this when you can delete a tenant
      // that isn't currently selected
      state.tenants = undefined;
      state.selectedTenantID = undefined;
      Object.assign(state, initialTenantPersistedState, initialTenantPageState);
      state.selectedTenant = undefined;
      break;
    case DELETE_TENANT_ERROR:
      state.deletingTenant = false;
      state.deleteTenantError = action.data;
      break;
    case GET_SELECTED_TENANT_REQUEST:
      state.fetchingSelectedTenant = true;
      state.selectedTenantID = undefined;
      state.selectedTenant = undefined;
      state.modifiedTenant = undefined;
      break;
    case GET_SELECTED_TENANT_SUCCESS: {
      const selectedTenant: SelectedTenant = action.data;
      Object.assign(state, initialTenantPersistedState, initialTenantPageState);
      state.tenantFetchError = '';
      if (state.tenants) {
        const matchingTenant = state.tenants.find(
          (t: Tenant) => t.id === selectedTenant.id
        );
        if (matchingTenant) {
          state.selectedTenantID = selectedTenant.id;
          state.selectedTenant = selectedTenant;
          state.modifiedTenant = { ...selectedTenant };
        }
      }
      state.fetchingSelectedTenant = false;
      break;
    }
    case GET_SELECTED_TENANT_ERROR:
      state.fetchingSelectedTenant = false;
      state.tenantFetchError = action.data;
      break;
    case GET_TENANTS_FOR_COMPANY_REQUEST:
      state.fetchingTenants = true;
      state.tenants = undefined;
      break;
    case GET_TENANTS_FOR_COMPANY_SUCCESS: {
      const [companyID, tenants] = action.data;
      if (companyID !== state.selectedCompanyID) {
        break;
      }
      state.fetchingTenants = false;
      state.tenants = tenants || [];
      Object.assign(state, initialTenantPersistedState, initialTenantPageState);
      state.selectedTenantID = undefined;
      state.selectedTenant = undefined;
      state.modifiedTenant = undefined;
      break;
    }
    case GET_TENANTS_FOR_COMPANY_ERROR:
      state.fetchingTenants = false;
      state.tenantFetchError = action.data;
      break;

    case CREATE_TENANT_URL_REQUEST:
      state.savingTenantURLs = true;
      break;
    case CREATE_TENANT_URL_SUCCESS:
      state.editingTenantURL = false;
      state.editingTenantURLError = '';
      state.creatingNewTenantURL = false;
      state.tenantURLs =
        state.tenantURLs?.map((url) => {
          return url === action.data.tenant_url ? action.data : url;
        }) || [];
      state.savingTenantURLs = false;
      break;
    case CREATE_TENANT_URL_ERROR:
      state.editingTenantURLError = action.data;
      state.savingTenantURLs = false;
      break;
    case UPDATE_TENANT_URL_REQUEST:
      state.savingTenantURLs = true;
      break;
    case UPDATE_TENANT_URL_SUCCESS:
      state.editingTenantURL = false;
      state.editingTenantURLError = '';
      state.tenantURLs =
        state.tenantURLs?.map((url) => {
          return url === action.data.tenant_url ? action.data : url;
        }) || [];
      state.savingTenantURLs = false;
      break;
    case UPDATE_TENANT_URL_ERROR:
      state.editingTenantURLError = action.data;
      state.savingTenantURLs = false;
      break;
    case GET_TENANT_URLS_REQUEST:
      state.fetchingTenantURLs = true;
      break;
    case GET_TENANT_URLS_SUCCESS:
      state.fetchingTenantURLs = false;
      state.tenantURLs = action.data;
      state.modifiedTenantUrls = state.tenantURLs ? [...state.tenantURLs] : [];
      break;
    case GET_TENANT_URLS_ERROR:
      state.fetchingTenantURLs = false;
      state.fetchingTenantURLsError = action.data;
      break;
    case DELETE_TENANT_URL_REQUEST:
      state.savingTenantURLs = true;
      break;
    case DELETE_TENANT_URL_SUCCESS:
      state.tenantURLs =
        state.tenantURLs?.filter((url) => url.id !== action.data) || [];
      state.savingTenantURLs = false;
      break;
    case DELETE_TENANT_URL_ERROR:
      state.savingTenantURLs = false;
      state.editingTenantURLError = action.data;
      break;
    case TOGGLE_TENANT_URL_EDIT_MODE: {
      const editMode = getNewToggleEditValue(
        action.data,
        state.editingTenantURL
      );
      if (editMode) {
        state.editingTenantURL = true;
        if (state.tenantURLs) {
          state.modifiedTenantUrls = [...state.tenantURLs];
        }
      } else {
        state.editingTenantURL = false;
        state.tenantURLsIsDirty = false;
        if (state.creatingNewTenantURL) {
          if (state.tenantURLs) {
            state.tenantURLs = state.tenantURLs.slice(0, -1);
          }
          state.creatingNewTenantURL = false;
        }
      }
      break;
    }
    case SET_CURRENT_URL:
      state.tenantURLDialogIsOpen = true;
      state.creatingNewTenantURL = false;
      state.currentTenantURL = action.data;
      state.modifiedTenantURL = { ...state.currentTenantURL! };
      state.editingTenantURL = true;
      break;
    case SET_CREATING_ISSUER:
      state.creatingIssuer = action.data;
      break;
    case SET_EDITING_ISSUER_INDEX:
      state.creatingIssuer = false;
      state.editingIssuerIndex = action.data;
      state.tenantIssuerDialogIsOpen = true;
      break;
    case SET_TENANT_PROVIDER_NAME:
      state.tenantProviderName = action.data;
      break;
    case TOGGLE_CREATE_TENANT_DIALOG:
      state.createTenantDialogIsOpen = getNewToggleEditValue(
        action.data,
        state.createTenantDialogIsOpen
      );
      break;
    case SET_CREATING_URL:
      state.creatingNewTenantURL = action.data;
      break;
    case MODIFY_TENANT_URL:
      {
        state.currentTenantURL = {
          ...state.currentTenantURL,
          ...action.data,
        };
        const existingUrlIndex = state.modifiedTenantUrls.findIndex(
          (url) => url.id === action.data.id
        );
        if (existingUrlIndex !== -1) {
          state.modifiedTenantUrls[existingUrlIndex] = {
            ...state.modifiedTenantUrls[existingUrlIndex],
            ...state.currentTenantURL,
          };
        } else {
          state.modifiedTenantUrls.push(action.data);
        }
        state.modifiedTenantUrls = [...state.modifiedTenantUrls];
      }
      break;
    case UPDATE_TENANT_URL:
      {
        const id = action.data.id;
        const existingUrlIndex = state.tenantURLsToUpdate.findIndex(
          (url) => url.id === id
        );
        if (existingUrlIndex !== -1) {
          state.tenantURLsToUpdate[existingUrlIndex] = {
            ...state.tenantURLsToUpdate[existingUrlIndex],
            ...state.currentTenantURL,
          };
        } else {
          state.tenantURLsToUpdate.push(action.data);
        }
        state.tenantURLsToUpdate = [...state.tenantURLsToUpdate];
        state.creatingNewTenantURL = false;
      }
      break;
    case CREATE_TENANT_URL:
      {
        const id = action.data.id;
        const existingUrlIndex = state.tenantURLsToCreate.findIndex(
          (url) => url.id === id
        );
        if (existingUrlIndex !== -1) {
          state.tenantURLsToCreate[existingUrlIndex] = {
            ...state.tenantURLsToCreate[existingUrlIndex],
            ...action.data,
          };
        } else {
          state.tenantURLsToCreate.push(action.data);
        }
        state.tenantURLsToCreate = [...state.tenantURLsToCreate];
        state.creatingNewTenantURL = false;
      }
      break;
    case ADD_TENANT_URL: {
      // Generate a temporary ID for the new URL
      const tempId = uuidv4();
      const newUrl = {
        id: tempId,
        tenant_id: state.selectedTenant?.id || '',
        tenant_url: '',
        system: false,
      } as TenantURL;

      state.modifiedTenantUrls = [...(state.modifiedTenantUrls || []), newUrl];
      state.currentTenantURL = newUrl; // Set the current URL for the dialog
      state.creatingNewTenantURL = true; // Set flag indicating creation mode
      state.tenantURLDialogIsOpen = true; // Open the dialog
      break;
    }
    case DELETE_TENANT_URL: {
      const urlIdToDelete = action.data;
      const originalUrlExists = state.tenantURLs?.some(
        (url) => url.id === urlIdToDelete
      );

      // Remove from the modified list regardless
      state.modifiedTenantUrls =
        state.modifiedTenantUrls?.filter((url) => url.id !== urlIdToDelete) ||
        [];

      // Only add to the delete list if it was an original, existing URL
      if (originalUrlExists) {
        state.tenantURLsToDelete = [
          ...(state.tenantURLsToDelete || []),
          urlIdToDelete,
        ];
      }

      // If the deleted URL was the one currently being edited in the dialog, reset it
      if (state.currentTenantURL?.id === urlIdToDelete) {
        state.currentTenantURL = undefined;
        state.creatingNewTenantURL = false; // Ensure this is reset if we cancel adding
        state.tenantURLDialogIsOpen = false; // Close dialog if the current item was deleted
      }
      break;
    }
    case TOGGLE_TENANT_URL_DIALOG_IS_OPEN:
      state.tenantURLDialogIsOpen = getNewToggleEditValue(
        action.data,
        state.tenantURLDialogIsOpen
      );
      break;
    case TOGGLE_TENANT_DATABASE_DIALOG_IS_OPEN:
      state.tenantDatabaseDialogIsOpen = getNewToggleEditValue(
        action.data,
        state.tenantDatabaseDialogIsOpen
      );
      break;
    case TOGGLE_TENANT_ISSUER_DIALOG_IS_OPEN:
      state.tenantIssuerDialogIsOpen = getNewToggleEditValue(
        action.data,
        state.tenantIssuerDialogIsOpen
      );
      break;

    default:
      break;
  }
  return state;
};

export default tenantsReducer;
