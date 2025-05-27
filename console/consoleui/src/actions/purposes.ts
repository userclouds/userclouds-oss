import { APIError } from '@userclouds/sharedui';
import PaginatedResult from '../models/PaginatedResult';
import Purpose from '../models/Purpose';

export const CHANGE_PURPOSE_SEARCH_FILTER = 'CHANGE_PURPOSE_SEARCH_FILTER';

export const GET_PURPOSES_REQUEST = 'GET_PURPOSES_REQUEST';
export const GET_PURPOSES_SUCCESS = 'GET_PURPOSES_SUCCESS';
export const GET_PURPOSES_ERROR = 'GET_PURPOSES_ERROR';

export const CHANGE_SELECTED_PURPOSE = 'CHANGE_SELECTED_PURPOSE';
export const GET_PURPOSE_REQUEST = 'GET_PURPOSE_REQUEST';
export const GET_PURPOSE_SUCCESS = 'GET_PURPOSE_SUCCESS';
export const GET_PURPOSE_ERROR = 'GET_PURPOSE_ERROR';
export const TOGGLE_PURPOSE_DETAILS_EDIT_MODE =
  'TOGGLE_PURPOSE_DETAILS_EDIT_MODE';
export const MODIFY_PURPOSE_DETAILS = 'MODIFY_PURPOSE_DETAILS';

export const CREATE_PURPOSE_REQUEST = 'CREATE_PURPOSE_REQUEST';
export const CREATE_PURPOSE_SUCCESS = 'CREATE_PURPOSE_SUCCESS';
export const CREATE_PURPOSE_ERROR = 'CREATE_PURPOSE_ERROR';

export const UPDATE_PURPOSE_REQUEST = 'UPDATE_PURPOSE_REQUEST';
export const UPDATE_PURPOSE_SUCCESS = 'UPDATE_PURPOSE_SUCCESS';
export const UPDATE_PURPOSE_ERROR = 'UPDATE_PURPOSE_ERROR';

export const DELETE_SINGLE_PURPOSE_REQUEST = 'DELETE_SINGLE_PURPOSE_REQUEST';
export const DELETE_SINGLE_PURPOSE_SUCCESS = 'DELETE_SINGLE_PURPOSE_SUCCESS';
export const DELETE_SINGLE_PURPOSE_ERROR = 'DELETE_SINGLE_PURPOSE_ERROR';

export const TOGGLE_PURPOSE_BULK_EDIT_MODE = 'TOGGLE_PURPOSE_BULK_EDIT_MODE';
export const TOGGLE_PURPOSE_FOR_DELETE = 'TOGGLE_PURPOSE_FOR_DELETE';

export const DELETE_PURPOSES_SINGLE_SUCCESS = 'DELETE_PURPOSES_SINGLE_SUCCESS';
export const DELETE_PURPOSES_SINGLE_ERROR = 'DELETE_PURPOSES_SINGLE_ERROR';

export const BULK_DELETE_PURPOSES_REQUEST = 'BULK_DELETE_PURPOSES_REQUEST';
export const BULK_DELETE_PURPOSES_SUCCESS = 'BULK_DELETE_PURPOSES_SUCCESS';
export const BULK_DELETE_PURPOSES_FAILURE = 'BULK_DELETE_PURPOSES_FAILURE';

export const DELETE_PURPOSES_SUCCESS = 'DELETE_PURPOSES_SUCCESS';
export const DELETE_PURPOSES_ERROR = 'DELETE_PURPOSES_ERROR';

export const changePurposeSearchFilter = (changes: Record<string, string>) => ({
  type: CHANGE_PURPOSE_SEARCH_FILTER,
  data: changes,
});
export const getPurposesRequest = () => ({
  type: GET_PURPOSES_REQUEST,
});
export const getPurposesSuccess = (purposes: PaginatedResult<Purpose>) => ({
  type: GET_PURPOSES_SUCCESS,
  data: purposes,
});
export const getPurposesError = (error: APIError) => ({
  type: GET_PURPOSES_ERROR,
  data: error.message,
});

export const changeSelectedPurpose = (purpose: Purpose) => ({
  type: CHANGE_SELECTED_PURPOSE,
  data: purpose,
});
export const getPurposeRequest = () => ({
  type: GET_PURPOSE_REQUEST,
});
export const getPurposeSuccess = (purpose: Purpose) => ({
  type: GET_PURPOSE_SUCCESS,
  data: purpose,
});
export const getPurposeError = (error: APIError) => ({
  type: GET_PURPOSE_ERROR,
  data: error.message,
});
export const togglePurposeDetailsEditMode = (editMode?: boolean) => ({
  type: TOGGLE_PURPOSE_DETAILS_EDIT_MODE,
  data: editMode,
});
export const modifyPurposeDetails = (
  modifications: Record<string, string | boolean>
) => ({
  type: MODIFY_PURPOSE_DETAILS,
  data: modifications,
});

export const createPurposeRequest = () => ({
  type: CREATE_PURPOSE_REQUEST,
});
export const createPurposeSuccess = (purpose: Purpose) => ({
  type: CREATE_PURPOSE_SUCCESS,
  data: purpose,
});
export const createPurposeError = (error: APIError) => ({
  type: CREATE_PURPOSE_ERROR,
  data: error.message,
});

export const updatePurposeRequest = () => ({
  type: UPDATE_PURPOSE_REQUEST,
});
export const updatePurposeSuccess = (purpose: Purpose) => ({
  type: UPDATE_PURPOSE_SUCCESS,
  data: purpose,
});
export const updatePurposeError = (error: APIError) => ({
  type: UPDATE_PURPOSE_ERROR,
  data: error.message,
});

export const deleteSinglePurposeRequest = () => ({
  type: DELETE_SINGLE_PURPOSE_REQUEST,
});
export const deleteSinglePurposeSuccess = () => ({
  type: DELETE_SINGLE_PURPOSE_SUCCESS,
});
export const deleteSinglePurposeError = (error: APIError) => ({
  type: DELETE_SINGLE_PURPOSE_ERROR,
  data: error.message,
});

export const togglePurposeBulkEditMode = (editMode?: boolean) => ({
  type: TOGGLE_PURPOSE_BULK_EDIT_MODE,
  data: editMode,
});

export const togglePurposeForDelete = (id: string) => ({
  type: TOGGLE_PURPOSE_FOR_DELETE,
  data: id,
});

export const bulkDeletePurposesRequest = () => ({
  type: BULK_DELETE_PURPOSES_REQUEST,
});

export const bulkDeletePurposesSuccess = () => ({
  type: BULK_DELETE_PURPOSES_SUCCESS,
});

export const bulkDeletePurposesFailure = () => ({
  type: BULK_DELETE_PURPOSES_FAILURE,
});

export const deletePurposesSingleSuccess = (purposeID: string) => ({
  type: DELETE_PURPOSES_SINGLE_SUCCESS,
  data: purposeID,
});

export const deletePurposesSingleError = (
  purposeID: string,
  error: APIError
) => ({
  type: DELETE_PURPOSES_SINGLE_ERROR,
  data: error.message,
});
