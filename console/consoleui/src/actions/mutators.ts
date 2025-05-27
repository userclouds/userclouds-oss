import { APIError, JSONValue } from '@userclouds/sharedui';

import Mutator, { MutatorColumn } from '../models/Mutator';
import PaginatedResult from '../models/PaginatedResult';

export const CHANGE_MUTATOR_SEARCH_FILTER = 'CHANGE_MUTATOR_SEARCH_FILTER';
export const TOGGLE_MUTATOR_LIST_EDIT_MODE = 'TOGGLE_MUTATOR_LIST_EDIT_MODE';
export const TOGGLE_MUTATOR_FOR_DELETE = 'TOGGLE_MUTATOR_FOR_DELETE';
export const DELETE_MUTATOR_SUCCESS = 'DELETE_MUTATOR_SUCCESS';
export const DELETE_MUTATOR_ERROR = 'DELETE_MUTATOR_ERROR';
export const BULK_EDIT_MUTATORS_REQUEST = 'BULK_EDIT_MUTATORS_REQUEST';
export const BULK_EDIT_MUTATORS_SUCCESS = 'BULK_EDIT_MUTATORS_SUCCESS';
export const BULK_EDIT_MUTATORS_ERROR = 'BULK_EDIT_MUTATORS_ERROR';
export const GET_TENANT_MUTATORS_REQUEST = 'GET_TENANT_MUTATORS_REQUEST';
export const GET_TENANT_MUTATORS_SUCCESS = 'GET_TENANT_MUTATORS_SUCCESS';
export const GET_TENANT_MUTATORS_ERROR = 'GET_TENANT_MUTATORS_ERROR';
export const GET_MUTATOR_REQUEST = 'GET_MUTATOR_REQUEST';
export const GET_MUTATOR_SUCCESS = 'GET_MUTATOR_SUCCESS';
export const GET_MUTATOR_ERROR = 'GET_MUTATOR_ERROR';
export const MODIFY_MUTATOR_DETAILS = 'MODIFY_MUTATOR_DETAILS';
export const MODIFY_MUTATOR_TO_CREATE = 'MODIFY_MUTATOR_TO_CREATE';
export const TOGGLE_MUTATOR_EDIT_MODE = 'TOGGLE_MUTATOR_EDIT_MODE';
export const TOGGLE_MUTATOR_DETAILS_EDIT_MODE =
  'TOGGLE_MUTATOR_DETAILS_EDIT_MODE';
export const TOGGLE_MUTATOR_COLUMNS_EDIT_MODE =
  'TOGGLE_MUTATOR_COLUMNS_EDIT_MODE';
export const TOGGLE_MUTATOR_SELECTOR_EDIT_MODE =
  'TOGGLE_MUTATOR_SELECTOR_EDIT_MODE';
export const TOGGLE_MUTATOR_POLICIES_EDIT_MODE =
  'TOGGLE_MUTATOR_POLICIES_EDIT_MODE';
export const UPDATE_MUTATOR_REQUEST = 'UPDATE_MUTATOR_REQUEST';
export const UPDATE_MUTATOR_SUCCESS = 'UPDATE_MUTATOR_SUCCESS';
export const UPDATE_MUTATOR_ERROR = 'UPDATE_MUTATOR_ERROR';
export const ADD_MUTATOR_COLUMN = 'ADD_MUTATOR_COLUMN';
export const TOGGLE_MUTATOR_COLUMN_FOR_DELETE =
  'TOGGLE_MUTATOR_COLUMN_FOR_DELETE';
export const CHANGE_MUTATOR_ADD_COLUMN_DROPDOWN_VALUE =
  'CHANGE_MUTATOR_ADD_COLUMN_DROPDOWN_VALUE';
export const SAVE_MUTATOR_COLUMNS_REQUEST = 'SAVE_MUTATOR_COLUMNS_REQUEST';
export const SAVE_MUTATOR_COLUMNS_SUCCESS = 'SAVE_MUTATOR_COLUMNS_SUCCESS';
export const SAVE_MUTATOR_COLUMNS_ERROR = 'SAVE_MUTATOR_COLUMNS_ERROR';
export const CHANGE_SELECTED_ACCESS_POLICY_FOR_MUTATOR =
  'CHANGE_SELECTED_ACCESS_POLICY_FOR_MUTATOR';
export const CHANGE_SELECTED_NORMALIZER_FOR_COLUMN =
  'CHANGE_SELECTED_NORMALIZER_FOR_COLUMN';
export const UPDATE_MUTATOR_POLICIES_REQUEST =
  'UPDATE_MUTATOR_POLICIES_REQUEST';
export const UPDATE_MUTATOR_POLICIES_SUCCESS =
  'UPDATE_MUTATOR_POLICIES_SUCCESS';
export const UPDATE_MUTATOR_POLICIES_ERROR = 'UPDATE_MUTATOR_POLICIES_ERROR';
export const LOAD_CREATE_MUTATOR_PAGE = 'LOAD_CREATE_MUTATOR_PAGE';
export const CREATE_MUTATOR_REQUEST = 'CREATE_MUTATOR_REQUEST';
export const CREATE_MUTATOR_SUCCESS = 'CREATE_MUTATOR_SUCCESS';
export const CREATE_MUTATOR_ERROR = 'CREATE_MUTATOR_ERROR';

export const changeMutatorSearchFilter = (changes: Record<string, string>) => ({
  type: CHANGE_MUTATOR_SEARCH_FILTER,
  data: changes,
});

export const toggleMutatorListEditMode = (editMode?: boolean) => ({
  type: TOGGLE_MUTATOR_LIST_EDIT_MODE,
  data: editMode,
});

export const toggleMutatorForDelete = (mutator: Mutator) => ({
  type: TOGGLE_MUTATOR_FOR_DELETE,
  data: mutator,
});

export const bulkEditMutatorsRequest = () => ({
  type: BULK_EDIT_MUTATORS_REQUEST,
});

export const bulkEditMutatorsSuccess = () => ({
  type: BULK_EDIT_MUTATORS_SUCCESS,
});

export const bulkEditMutatorsError = () => ({
  type: BULK_EDIT_MUTATORS_ERROR,
});

export const deleteMutatorSuccess = (mutatorID: string) => ({
  type: DELETE_MUTATOR_SUCCESS,
  data: mutatorID,
});

export const deleteMutatorError = (error: APIError) => ({
  type: DELETE_MUTATOR_ERROR,
  data: error.message,
});

export const getTenantMutatorsRequest = () => ({
  type: GET_TENANT_MUTATORS_REQUEST,
});

export const getTenantMutatorsSuccess = (
  mutators: PaginatedResult<Mutator>
) => ({
  type: GET_TENANT_MUTATORS_SUCCESS,
  data: mutators,
});

export const getTenantMutatorsError = (error: APIError) => ({
  type: GET_TENANT_MUTATORS_ERROR,
  data: error.message,
});

export const getMutatorRequest = (mutatorID: string) => ({
  type: GET_MUTATOR_REQUEST,
  data: mutatorID,
});

export const getMutatorSuccess = (mutator: Mutator) => ({
  type: GET_MUTATOR_SUCCESS,
  data: mutator,
});

export const getMutatorError = (error: APIError) => ({
  type: GET_MUTATOR_ERROR,
  data: error.message,
});

export const toggleMutatorEditMode = (editMode?: boolean) => ({
  type: TOGGLE_MUTATOR_EDIT_MODE,
  data: editMode,
});

export const toggleMutatorDetailsEditMode = (editMode?: boolean) => ({
  type: TOGGLE_MUTATOR_DETAILS_EDIT_MODE,
  data: editMode,
});

export const modifyMutatorDetails = (data: Record<string, any>) => ({
  type: MODIFY_MUTATOR_DETAILS,
  data,
});

export const modifyMutatorToCreate = (data: Record<string, JSONValue>) => ({
  type: MODIFY_MUTATOR_TO_CREATE,
  data,
});

export const updateMutatorRequest = () => ({
  type: UPDATE_MUTATOR_REQUEST,
});

export const updateMutatorSuccess = (mutator: Mutator) => ({
  type: UPDATE_MUTATOR_SUCCESS,
  data: mutator,
});

export const updateMutatorError = (error: APIError) => ({
  type: UPDATE_MUTATOR_ERROR,
  data: error.message,
});

export const toggleMutatorColumnsEditMode = (editMode?: boolean) => ({
  type: TOGGLE_MUTATOR_COLUMNS_EDIT_MODE,
  data: editMode,
});

export const toggleMutatorColumnForDelete = (column: MutatorColumn) => ({
  type: TOGGLE_MUTATOR_COLUMN_FOR_DELETE,
  data: column,
});

export const changeMutatorAddColumnDropdownValue = (value: string) => ({
  type: CHANGE_MUTATOR_ADD_COLUMN_DROPDOWN_VALUE,
  data: value,
});

export const addMutatorColumn = (value: string) => ({
  type: ADD_MUTATOR_COLUMN,
  data: value,
});

export const changeSelectedNormalizerForColumn = (
  columnID: string,
  normalizerID: string,
  isNew: boolean
) => ({
  type: CHANGE_SELECTED_NORMALIZER_FOR_COLUMN,
  data: {
    columnID,
    normalizerID,
    isNew,
  },
});

export const toggleMutatorSelectorEditMode = (editMode?: boolean) => ({
  type: TOGGLE_MUTATOR_SELECTOR_EDIT_MODE,
  data: editMode,
});

export const toggleMutatorPoliciesEditMode = (editMode?: boolean) => ({
  type: TOGGLE_MUTATOR_POLICIES_EDIT_MODE,
  data: editMode,
});

export const changeSelectedAccessPolicyForMutator = (policyID: string) => ({
  type: CHANGE_SELECTED_ACCESS_POLICY_FOR_MUTATOR,
  data: policyID,
});

export const loadCreateMutatorPage = () => ({
  type: LOAD_CREATE_MUTATOR_PAGE,
});

export const createMutatorRequest = () => ({
  type: CREATE_MUTATOR_REQUEST,
});

export const createMutatorSuccess = (mutator: Mutator) => ({
  type: CREATE_MUTATOR_SUCCESS,
  data: mutator,
});

export const createMutatorError = (error: APIError) => ({
  type: CREATE_MUTATOR_ERROR,
  data: error.message,
});
