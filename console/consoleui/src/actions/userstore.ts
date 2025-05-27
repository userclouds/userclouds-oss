import { APIError, JSONValue } from '@userclouds/sharedui';
import { AppDispatch } from '../store';
import { Column, Schema } from '../models/TenantUserStoreConfig';
import PaginatedResult from '../models/PaginatedResult';
import {
  ColumnRetentionDurationsResponse,
  PurposeRetentionDuration,
} from '../models/ColumnRetentionDurations';
import Accessor from '../models/Accessor';
import { DataType } from '../models/DataType';
import { SqlshimDatabase } from '../models/SqlshimDatabase';
import { ObjectStore } from '../models/ObjectStore';

export const CHANGE_COLUMN_SEARCH_FILTER = 'CHANGE_COLUMN_SEARCH_FILTER';
export const CHANGE_SELECTED_COLUMN = 'CHANGE_SELECTED_COLUMN';
export const TOGGLE_COLUMN_EDIT_MODE = 'TOGGLE_COLUMN_EDIT_MODE';
export const MODIFY_USER_STORE_COLUMN = 'MODIFY_USER_STORE_COLUMN';
export const MODIFY_USER_STORE_COLUMN_DEFAULT_TRANSFORMER =
  'MODIFY_USER_STORE_COLUMN_DEFAULT_TRANSFORMER';
export const TOGGLE_COLUMN_PURPOSES_EDIT_MODE =
  'TOGGLE_COLUMN_PURPOSES_EDIT_MODE';

export const GET_COLUMN_DURATIONS_REQUEST = 'GET_COLUMN_DURATIONS_REQUEST';
export const GET_COLUMN_DURATIONS_SUCCESS = 'GET_COLUMN_DURATIONS_SUCCESS';
export const GET_COLUMN_DURATIONS_ERROR = 'GET_COLUMN_DURATIONS_ERROR';
export const UPDATE_COLUMN_DURATIONS_REQUEST =
  'UPDATE_COLUMN_DURATIONS_REQUEST';
export const UPDATE_COLUMN_DURATIONS_SUCCESS =
  'UPDATE_COLUMN_DURATIONS_SUCCESS';
export const UPDATE_COLUMN_DURATIONS_ERROR = 'UPDATE_COLUMN_DURATIONS_ERROR';
export const MODIFY_RETENTION_DURATION = 'MODIFY_RETENTION_DURATION';

export const FETCH_USER_STORE_COLUMN_REQUEST =
  'FETCH_USER_STORE_COLUMN_REQUEST';
export const FETCH_USER_STORE_COLUMN_SUCCESS =
  'FETCH_USER_STORE_COLUMN_SUCCESS';
export const FETCH_USER_STORE_COLUMN_ERROR = 'FETCH_USER_STORE_COLUMN_ERROR';

export const GET_ACCESSORS_FOR_COLUMN_REQUEST =
  'GET_ACCESSORS_FOR_COLUMN_REQUEST';
export const GET_ACCESSORS_FOR_COLUMN_SUCCESS =
  'GET_ACCESSORS_FOR_COLUMN_SUCCESS';
export const GET_ACCESSORS_FOR_COLUMN_ERROR = 'GET_ACCESSORS_FOR_COLUMN_ERROR';

export const GET_ACCESSOR_METRICS_REQUEST = 'GET_ACCESSOR_METRICS_REQUEST';
export const GET_ACCESSOR_METRICS_SUCCESS = 'GET_ACCESSOR_METRICS_SUCCESS';
export const GET_ACCESSOR_METRICS_ERROR = 'GET_ACCESSOR_METRICS_ERROR';

export const FETCH_USER_STORE_DISPLAY_COLUMNS_REQUEST =
  'FETCH_USER_STORE_DISPLAY_COLUMNS_REQUEST';
export const FETCH_USER_STORE_DISPLAY_COLUMNS_SUCCESS =
  'FETCH_USER_STORE_DISPLAY_COLUMNS_SUCCESS';
export const FETCH_USER_STORE_DISPLAY_COLUMNS_ERROR =
  'FETCH_USER_STORE_DISPLAY_COLUMNS_ERROR';

export const CREATE_USER_STORE_COLUMN_REQUEST =
  'CREATE_USER_STORE_COLUMN_REQUEST';
export const CREATE_USER_STORE_COLUMN_SUCCESS =
  'CREATE_USER_STORE_COLUMN_SUCCESS';
export const CREATE_USER_STORE_COLUMN_ERROR = 'CREATE_USER_STORE_COLUMN_ERROR';

export const UPDATE_USER_STORE_COLUMN_REQUEST =
  'UPDATE_USER_STORE_COLUMN_REQUEST';
export const UPDATE_USER_STORE_COLUMN_SUCCESS =
  'UPDATE_USER_STORE_COLUMN_SUCCESS';
export const UPDATE_USER_STORE_COLUMN_ERROR = 'UPDATE_USER_STORE_COLUMN_ERROR';

export const DELETE_USER_STORE_COLUMN_REQUEST =
  'DELETE_USER_STORE_COLUMN_REQUEST';
export const DELETE_USER_STORE_COLUMN_SUCCESS =
  'DELETE_USER_STORE_COLUMN_SUCCESS';
export const DELETE_USER_STORE_COLUMN_ERROR = 'DELETE_USER_STORE_COLUMN_ERROR';

export const GET_USER_STORE_CONFIG_REQUEST = 'GET_USER_STORE_CONFIG_REQUEST';
export const GET_USER_STORE_CONFIG_SUCCESS = 'GET_USER_STORE_CONFIG_SUCCESS';
export const GET_USER_STORE_CONFIG_ERROR = 'GET_USER_STORE_CONFIG_ERROR';

export const ADD_USER_STORE_COLUMN = 'ADD_USER_STORE_COLUMN';

export const MODIFY_BULK_USER_STORE_COLUMN = 'MODIFY_BULK_USER_STORE_COLUMN';

export const TOGGLE_USER_STORE_COLUMN_FOR_DELETE =
  'TOGGLE_USER_STORE_COLUMN_FOR_DELETE';

export const CREATE_BULK_USER_STORE_COLUMN_SUCCESS =
  'CREATE_BULK_USER_STORE_COLUMN_SUCCESS';
export const CREATE_BULK_USER_STORE_COLUMN_ERROR =
  'CREATE_BULK_USER_STORE_COLUMN_ERROR';

export const UPDATE_BULK_USER_STORE_COLUMN_SUCCESS =
  'UPDATE_BULK_USER_STORE_COLUMN_SUCCESS';
export const UPDATE_BULK_USER_STORE_COLUMN_ERROR =
  'UPDATE_BULK_USER_STORE_COLUMN_ERROR';

export const DELETE_BULK_USER_STORE_COLUMN_SUCCESS =
  'DELETE_BULK_USER_STORE_COLUMN_SUCCESS';
export const DELETE_BULK_USER_STORE_COLUMN_ERROR =
  'DELETE_BULK_USER_STORE_COLUMN_ERROR';

export const UPDATE_USER_STORE_CONFIG_REQUEST =
  'UPDATE_USER_STORE_CONFIG_REQUEST';
export const UPDATE_USER_STORE_CONFIG_SUCCESS =
  'UPDATE_USER_STORE_CONFIG_SUCCESS';
export const UPDATE_USER_STORE_CONFIG_ERROR = 'UPDATE_USER_STORE_CONFIG_ERROR';

export const TOGGLE_USER_STORE_EDIT_MODE = 'TOGGLE_USER_STORE_EDIT_MODE';

export const CHANGE_DATA_TYPE_SEARCH_FILTER = 'CHANGE_DATA_TYPE_SEARCH_FILTER';
export const CHANGE_SELECTED_DATA_TYPE = 'CHANGE_SELECTED_DATA_TYPE';
export const TOGGLE_DATA_TYPE_EDIT_MODE = 'TOGGLE_DATA_TYPE_EDIT_MODE';
export const MODIFY_DATA_TYPE = 'MODIFY_DATA_TYPE';

export const FETCH_DATA_TYPE_REQUEST = 'FETCH_DATA_TYPE_REQUEST';
export const FETCH_DATA_TYPE_SUCCESS = 'FETCH_DATA_TYPE_SUCCESS';
export const FETCH_DATA_TYPE_ERROR = 'FETCH_DATA_TYPE_ERROR';

export const FETCH_DATA_TYPES_REQUEST = 'FETCH_DATA_TYPES_REQUEST';
export const FETCH_DATA_TYPES_SUCCESS = 'FETCH_DATA_TYPES_SUCCESS';
export const FETCH_DATA_TYPES_ERROR = 'FETCH_DATA_TYPES_ERROR';

export const CREATE_DATA_TYPE_REQUEST = 'CREATE_DATA_TYPE_REQUEST';
export const CREATE_DATA_TYPE_SUCCESS = 'CREATE_DATA_TYPE_SUCCESS';
export const CREATE_DATA_TYPE_ERROR = 'CREATE_DATA_TYPE_ERROR';

export const UPDATE_DATA_TYPE_REQUEST = 'UPDATE_DATA_TYPE_REQUEST';
export const UPDATE_DATA_TYPE_SUCCESS = 'UPDATE_DATA_TYPE_SUCCESS';
export const UPDATE_DATA_TYPE_ERROR = 'UPDATE_DATA_TYPE_ERROR';

export const DELETE_DATA_TYPE_REQUEST = 'DELETE_DATA_TYPE_REQUEST';
export const DELETE_DATA_TYPE_SUCCESS = 'DELETE_DATA_TYPE_SUCCESS';
export const DELETE_DATA_TYPE_ERROR = 'DELETE_DATA_TYPE_ERROR';

export const TOGGLE_DATA_TYPE_FOR_DELETE = 'TOGGLE_DATA_TYPE_FOR_DELETE';

export const DELETE_BULK_DATA_TYPE_SUCCESS = 'DELETE_BULK_DATA_TYPE_SUCCESS';
export const DELETE_BULK_DATA_TYPE_ERROR = 'DELETE_BULK_DATA_TYPE_ERROR';

export const LOAD_CREATE_DATA_TYPE_PAGE = 'LOAD_CREATE_DATA_TYPE_PAGE';
export const MODIFY_DATA_TYPE_TO_CREATE = 'MODIFY_DATA_TYPE_TO_CREATE';
export const ADD_FIELD_TO_DATA_TYPE_TO_CREATE =
  'ADD_FIELD_TO_DATA_TYPE_TO_CREATE';
export const ADD_FIELD_TO_DATA_TYPE = 'ADD_FIELD_TO_DATA_TYPE';
export const BULK_DELETE_DATA_TYPES_REQUEST = 'BULK_DELETE_DATA_TYPES_REQUEST';
export const BULK_DELETE_DATA_TYPES_SUCCESS = 'BULK_DELETE_DATA_TYPES_SUCCESS';
export const BULK_DELETE_DATA_TYPES_FAILURE = 'BULK_DELETE_DATA_TYPES_FAILURE';

export const FETCH_USER_STORE_DATABASE_REQUEST =
  'FETCH_USER_STORE_DATABASE_REQUEST';
export const FETCH_USER_STORE_DATABASES_SUCCESS =
  'FETCH_USER_STORE_DATABASES_SUCCESS';
export const FETCH_USER_STORE_DATABASE_ERROR =
  'FETCH_USER_STORE_DATABASE_ERROR';

export const MODIFY_USER_STORE_DATABASE = 'MODIFY_USER_STORE_DATABASE';
export const SAVE_USER_STORE_DATABASE_REQUEST =
  'SAVE_USER_STORE_DATABASE_REQUEST';
export const SAVE_USER_STORE_DATABASE_SUCCESS =
  'SAVE_USER_STORE_DATABASE_SUCCESS';
export const SAVE_USER_STORE_DATABASE_ERROR = 'SAVE_USER_STORE_DATABASE_ERROR';
export const DELETE_DATABASE = 'DELETE_DATABASE';
export const SET_CURRENT_DATABASE = 'SET_CURRENT_DATABASE';
export const ADD_DATABASE = 'ADD_DATABASE';
export const CANCEL_DATABASE_DIALOG = 'CANCEL_DATABASE_DIALOG';
export const RESET_DATABASE_DIALOG_STATE = 'RESET_DATABASE_DIALOG_STATE';
export const TEST_DATABASE_CONNECTION_REQUEST =
  'TEST_DATABASE_CONNECTION_REQUEST';
export const TEST_DATABASE_CONNECTION_SUCCESS =
  'TEST_DATABASE_CONNECTION_SUCCESS';
export const TEST_DATABASE_CONNECTION_ERROR = 'TEST_DATABASE_CONNECTION_ERROR';

export const FETCH_USER_STORE_OBJECT_STORE_REQUEST =
  'FETCH_USER_STORE_OBJECT_STORE_REQUEST';
export const FETCH_USER_STORE_OBJECT_STORE_SUCCESS =
  'FETCH_USER_STORE_OBJECT_STORE_SUCCESS';
export const FETCH_USER_STORE_OBJECT_STORE_ERROR =
  'FETCH_USER_STORE_OBJECT_STORE_ERROR';
export const FETCH_USER_STORE_OBJECT_STORES_REQUEST =
  'FETCH_USER_STORE_OBJECT_STORES_REQUEST';
export const FETCH_USER_STORE_OBJECT_STORES_SUCCESS =
  'FETCH_USER_STORE_OBJECT_STORES_SUCCESS';
export const FETCH_USER_STORE_OBJECT_STORES_ERROR =
  'FETCH_USER_STORE_OBJECT_STORES_ERROR';
export const TOGGLE_EDIT_USER_STORE_OBJECT_STORE_MODE =
  'TOGGLE_EDIT_USER_STORE_OBJECT_STORE_MODE';
export const MODIFY_USER_STORE_OBJECT_STORE = 'MODIFY_USER_STORE_OBJECT_STORE';
export const SAVE_USER_STORE_OBJECT_STORE_REQUEST =
  'SAVE_USER_STORE_OBJECT_STORE_REQUEST';
export const SAVE_USER_STORE_OBJECT_STORE_SUCCESS =
  'SAVE_USER_STORE_OBJECT_STORE_SUCCESS';
export const SAVE_USER_STORE_OBJECT_STORE_ERROR =
  'SAVE_USER_STORE_OBJECT_STORE_ERROR';
export const TOGGLE_OBJECT_STORE_FOR_DELETE = 'TOGGLE_OBJECT_STORE_FOR_DELETE';

export const changeColumnSearchFilter = (changes: Record<string, string>) => ({
  type: CHANGE_COLUMN_SEARCH_FILTER,
  data: changes,
});

export const changeSelectedColumn = (column: Column) => ({
  type: CHANGE_SELECTED_COLUMN,
  data: column,
});

export const toggleColumnEditMode = (editMode?: boolean) => ({
  type: TOGGLE_COLUMN_EDIT_MODE,
  data: editMode,
});

export const modifyUserStoreColumn =
  (data: Record<string, JSONValue>) => (dispatch: AppDispatch) => {
    dispatch({
      type: MODIFY_USER_STORE_COLUMN,
      data: data,
    });
  };

export const modifyUserStoreColumnDefaultTransformer =
  (data: string) => (dispatch: AppDispatch) => {
    dispatch({
      type: MODIFY_USER_STORE_COLUMN_DEFAULT_TRANSFORMER,
      data: data,
    });
  };

export const toggleColumnPurposesEditMode = (editMode?: boolean) => ({
  type: TOGGLE_COLUMN_PURPOSES_EDIT_MODE,
  data: editMode,
});

export const getColumnDurationsRequest = () => ({
  type: GET_COLUMN_DURATIONS_REQUEST,
});

export const getColumnDurationsSuccess = (
  durations: ColumnRetentionDurationsResponse
) => ({
  type: GET_COLUMN_DURATIONS_SUCCESS,
  data: durations,
});

export const getColumnDurationsError = (error: APIError) => ({
  type: GET_COLUMN_DURATIONS_ERROR,
  data: error.message,
});

export const updateColumnDurationsRequest = () => ({
  type: UPDATE_COLUMN_DURATIONS_REQUEST,
});

export const updateColumnDurationsSuccess = (
  durations: ColumnRetentionDurationsResponse
) => ({
  type: UPDATE_COLUMN_DURATIONS_SUCCESS,
  data: durations,
});

export const updateColumnDurationsError = (error: APIError) => ({
  type: UPDATE_COLUMN_DURATIONS_ERROR,
  data: error.message,
});

export const modifyRetentionDuration = (
  duration: PurposeRetentionDuration
) => ({
  type: MODIFY_RETENTION_DURATION,
  data: duration,
});

export const fetchUserStoreColumnRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: FETCH_USER_STORE_COLUMN_REQUEST,
  });
};

export const fetchUserStoreColumnSuccess =
  (column: Column) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_USER_STORE_COLUMN_SUCCESS,
      data: column,
    });
  };

export const fetchUserStoreColumnError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_USER_STORE_COLUMN_ERROR,
      data: error,
    });
  };

export const getAccessorsForColumnRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: GET_ACCESSORS_FOR_COLUMN_REQUEST,
  });
};

export const getAccessorsForColumnSuccess =
  (accessors: PaginatedResult<Accessor>) => (dispatch: AppDispatch) => {
    dispatch({
      type: GET_ACCESSORS_FOR_COLUMN_SUCCESS,
      data: accessors,
    });
  };

export const getAccessorsForColumnError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: GET_ACCESSORS_FOR_COLUMN_ERROR,
      data: error.message,
    });
  };

export const fetchUserStoreDisplayColumnsRequest =
  () => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_USER_STORE_DISPLAY_COLUMNS_REQUEST,
    });
  };

export const fetchUserStoreDisplayColumnsSuccess =
  (columns: PaginatedResult<Column>) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_USER_STORE_DISPLAY_COLUMNS_SUCCESS,
      data: columns,
    });
  };

export const fetchUserStoreDisplayColumnsError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_USER_STORE_DISPLAY_COLUMNS_ERROR,
      data: error.message,
    });
  };

export const createUserStoreColumnRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: CREATE_USER_STORE_COLUMN_REQUEST,
  });
};

export const createUserStoreColumnSuccess =
  (column: Column) => (dispatch: AppDispatch) => {
    dispatch({
      type: CREATE_USER_STORE_COLUMN_SUCCESS,
      data: column,
    });
  };

export const createUserStoreColumnError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: CREATE_USER_STORE_COLUMN_ERROR,
      data: error.message,
    });
  };

export const updateUserStoreColumnRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: UPDATE_USER_STORE_COLUMN_REQUEST,
  });
};

export const updateUserStoreColumnSuccess =
  (column: Column) => (dispatch: AppDispatch) => {
    dispatch({
      type: UPDATE_USER_STORE_COLUMN_SUCCESS,
      data: column,
    });
  };

export const updateUserStoreColumnError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: UPDATE_USER_STORE_COLUMN_ERROR,
      data: error.message,
    });
  };

export const deleteUserStoreColumnRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: DELETE_USER_STORE_COLUMN_REQUEST,
  });
};

export const deleteUserStoreColumnSuccess = () => (dispatch: AppDispatch) => {
  dispatch({
    type: DELETE_USER_STORE_COLUMN_SUCCESS,
  });
};

export const deleteUserStoreColumnError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: DELETE_USER_STORE_COLUMN_ERROR,
      data: error.message,
    });
  };

export const getUserStoreConfigRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: GET_USER_STORE_CONFIG_REQUEST,
  });
};

export const getUserStoreConfigSuccess =
  (schema: Schema) => (dispatch: AppDispatch) => {
    dispatch({
      type: GET_USER_STORE_CONFIG_SUCCESS,
      data: schema,
    });
  };

export const getUserStoreConfigError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: GET_USER_STORE_CONFIG_ERROR,
      data: error.message,
    });
  };

export const updateUserStoreConfigRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: UPDATE_USER_STORE_CONFIG_REQUEST,
  });
};

export const updateUserStoreConfigSuccess = () => (dispatch: AppDispatch) => {
  dispatch({
    type: UPDATE_USER_STORE_CONFIG_SUCCESS,
  });
};

export const updateUserStoreConfigError = () => (dispatch: AppDispatch) => {
  dispatch({
    type: UPDATE_USER_STORE_CONFIG_ERROR,
  });
};

export const addUserStoreColumn = () => (dispatch: AppDispatch) => {
  dispatch({
    type: ADD_USER_STORE_COLUMN,
  });
};

export const createBulkUserStoreColumnSuccess =
  (column: Column) => (dispatch: AppDispatch) => {
    dispatch({
      type: CREATE_BULK_USER_STORE_COLUMN_SUCCESS,
      data: column,
    });
  };

export const createBulkUserStoreColumnError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: CREATE_BULK_USER_STORE_COLUMN_ERROR,
      data: error,
    });
  };

export const modifyBulkUserStoreColumn =
  (col: Column) => (dispatch: AppDispatch) => {
    dispatch({
      type: MODIFY_BULK_USER_STORE_COLUMN,
      data: col,
    });
  };

export const updateBulkUserStoreColumnSuccess =
  (column: Column) => (dispatch: AppDispatch) => {
    dispatch({
      type: UPDATE_BULK_USER_STORE_COLUMN_SUCCESS,
      data: column,
    });
  };

export const updateBulkUserStoreColumnError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: UPDATE_BULK_USER_STORE_COLUMN_ERROR,
      data: error.message,
    });
  };

export const deleteBulkUserStoreColumn =
  (col: Column) => (dispatch: AppDispatch) => {
    dispatch({
      type: TOGGLE_USER_STORE_COLUMN_FOR_DELETE,
      data: col,
    });
  };

export const deleteBulkUserStoreColumnSuccess =
  () => (dispatch: AppDispatch) => {
    dispatch({
      type: DELETE_BULK_USER_STORE_COLUMN_SUCCESS,
    });
  };

export const deleteBulkUserStoreColumnError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: DELETE_BULK_USER_STORE_COLUMN_ERROR,
      data: error.message,
    });
  };

export const toggleUserStoreEditMode = (editMode?: boolean) => ({
  type: TOGGLE_USER_STORE_EDIT_MODE,
  data: editMode,
});

export const changeDataTypeSearchFilter = (
  changes: Record<string, string>
) => ({
  type: CHANGE_DATA_TYPE_SEARCH_FILTER,
  data: changes,
});

export const changeSelectedDataType = (dataType: DataType) => ({
  type: CHANGE_SELECTED_DATA_TYPE,
  data: dataType,
});

export const toggleDataTypeEditMode = (editMode?: boolean) => ({
  type: TOGGLE_DATA_TYPE_EDIT_MODE,
  data: editMode,
});

export const modifyDataType =
  (data: Record<string, JSONValue>) => (dispatch: AppDispatch) => {
    dispatch({
      type: MODIFY_DATA_TYPE,
      data: data,
    });
  };

export const fetchDataTypeRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: FETCH_DATA_TYPE_REQUEST,
  });
};

export const fetchDataTypeSuccess =
  (dataType: DataType) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_DATA_TYPE_SUCCESS,
      data: dataType,
    });
  };

export const fetchDataTypeError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_DATA_TYPE_ERROR,
      data: error,
    });
  };

export const createDataTypeRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: CREATE_DATA_TYPE_REQUEST,
  });
};

export const createDataTypeSuccess =
  (dataType: DataType) => (dispatch: AppDispatch) => {
    dispatch({
      type: CREATE_DATA_TYPE_SUCCESS,
      data: dataType,
    });
  };

export const createDataTypeError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: CREATE_DATA_TYPE_ERROR,
      data: error.message,
    });
  };

export const updateDataTypeRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: UPDATE_DATA_TYPE_REQUEST,
  });
};

export const updateDataTypeSuccess =
  (dataType: DataType) => (dispatch: AppDispatch) => {
    dispatch({
      type: UPDATE_DATA_TYPE_SUCCESS,
      data: dataType,
    });
  };

export const updateDataTypeError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: UPDATE_DATA_TYPE_ERROR,
      data: error.message,
    });
  };

export const deleteDataTypeRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: DELETE_DATA_TYPE_REQUEST,
  });
};

export const deleteDataTypeSuccess = () => (dispatch: AppDispatch) => {
  dispatch({
    type: DELETE_DATA_TYPE_SUCCESS,
  });
};

export const deleteDataTypeError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: DELETE_DATA_TYPE_ERROR,
      data: error.message,
    });
  };

export const toggleDataTypeForDelete =
  (datatype: DataType) => (dispatch: AppDispatch) => {
    dispatch({
      type: TOGGLE_DATA_TYPE_FOR_DELETE,
      data: datatype.id,
    });
  };

export const deleteBulkDataTypeSuccess = () => (dispatch: AppDispatch) => {
  dispatch({
    type: DELETE_BULK_DATA_TYPE_SUCCESS,
  });
};

export const deleteBulkDataTypeError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: DELETE_BULK_DATA_TYPE_ERROR,
      data: error.message,
    });
  };
export const fetchDataTypesRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: FETCH_DATA_TYPES_REQUEST,
  });
};

export const fetchDataTypesSuccess =
  (datatypes: PaginatedResult<DataType>) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_DATA_TYPES_SUCCESS,
      data: datatypes,
    });
  };

export const fetchDataTypesError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_DATA_TYPES_ERROR,
      data: error.message,
    });
  };

export const loadCreateDataTypePage = () => ({
  type: LOAD_CREATE_DATA_TYPE_PAGE,
});

export const modifyDataTypeToCreate = (data: Record<string, JSONValue>) => ({
  type: MODIFY_DATA_TYPE_TO_CREATE,
  data,
});

export const addFieldToDataTypeToCreate = () => ({
  type: ADD_FIELD_TO_DATA_TYPE_TO_CREATE,
});

export const addFieldToDataType = () => ({
  type: ADD_FIELD_TO_DATA_TYPE,
});

export const bulkDeleteDataTypesRequest = () => ({
  type: BULK_DELETE_DATA_TYPES_REQUEST,
});

export const bulkDeleteDataTypesSuccess = () => ({
  type: BULK_DELETE_DATA_TYPES_SUCCESS,
});

export const bulkDeleteDataTypesFailure = () => ({
  type: BULK_DELETE_DATA_TYPES_FAILURE,
});

export const fetchUserStoreDatabaseRequest = () => (dispatch: AppDispatch) => {
  dispatch({
    type: FETCH_USER_STORE_DATABASE_REQUEST,
  });
};

export const fetchUserStoreDatabasesSuccess =
  (databases: PaginatedResult<SqlshimDatabase>) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_USER_STORE_DATABASES_SUCCESS,
      data: databases,
    });
  };

export const fetchUserStoreDatabaseError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_USER_STORE_DATABASE_ERROR,
      data: error,
    });
  };

export const addDatabase = () => ({
  type: ADD_DATABASE,
});

export const deleteDatabase = (id: string) => ({
  type: DELETE_DATABASE,
  data: id,
});

export const setCurrentDatabase = (databaseID: string) => ({
  type: SET_CURRENT_DATABASE,
  data: databaseID,
});

export const modifyUserstoreDatabase = (val: object) => ({
  type: MODIFY_USER_STORE_DATABASE,
  data: val,
});

export const saveUserStoreDatabaseRequest = () => ({
  type: SAVE_USER_STORE_DATABASE_REQUEST,
});

export const saveUserStoreDatabaseSuccess = () => ({
  type: SAVE_USER_STORE_DATABASE_SUCCESS,
});

export const saveUserStoreDatabaseError = (error: APIError) => ({
  type: SAVE_USER_STORE_DATABASE_ERROR,
  data: error.message,
});

export const editUserStoreDatabaseError = (error: string) => ({
  type: SAVE_USER_STORE_DATABASE_ERROR,
  data: error,
});

export const cancelDatabaseDialog = (id: string) => ({
  type: CANCEL_DATABASE_DIALOG,
  data: id,
});

export const resetDatabaseDialogState = () => ({
  type: RESET_DATABASE_DIALOG_STATE,
});

export const testDatabaseConnectionRequest = () => ({
  type: TEST_DATABASE_CONNECTION_REQUEST,
});

export const testDatabaseConnectionSuccess = () => ({
  type: TEST_DATABASE_CONNECTION_SUCCESS,
});

export const testDatabaseConnectionError = (error: APIError) => ({
  type: TEST_DATABASE_CONNECTION_ERROR,
  data: error.message,
});

export const fetchUserStoreObjectStoreRequest =
  () => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_USER_STORE_OBJECT_STORE_REQUEST,
    });
  };

export const fetchUserStoreObjectStoreSuccess =
  (database: ObjectStore) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_USER_STORE_OBJECT_STORE_SUCCESS,
      data: database,
    });
  };

export const fetchUserStoreObjectStoreError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_USER_STORE_OBJECT_STORE_ERROR,
      data: error,
    });
  };

export const fetchUserStoreObjectStoresRequest =
  () => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_USER_STORE_OBJECT_STORES_REQUEST,
    });
  };

export const fetchUserStoreObjectStoresSuccess =
  (objectStores: PaginatedResult<ObjectStore>) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_USER_STORE_OBJECT_STORES_SUCCESS,
      data: objectStores,
    });
  };

export const fetchUserStoreObjectStoresError =
  (error: APIError) => (dispatch: AppDispatch) => {
    dispatch({
      type: FETCH_USER_STORE_OBJECT_STORES_ERROR,
      data: error.message,
    });
  };

export const toggleEditUserstoreObjectStoreMode = (editMode?: boolean) => ({
  type: TOGGLE_EDIT_USER_STORE_OBJECT_STORE_MODE,
  data: editMode,
});

export const modifyUserstoreObjectStore = (val: object) => ({
  type: MODIFY_USER_STORE_OBJECT_STORE,
  data: val,
});

export const saveUserStoreObjectStoreRequest = () => ({
  type: SAVE_USER_STORE_OBJECT_STORE_REQUEST,
});

export const saveUserStoreObjectStoreSuccess = (val: object) => ({
  type: SAVE_USER_STORE_OBJECT_STORE_SUCCESS,
  data: val,
});

export const saveUserStoreObjectStoreError = (error: APIError) => ({
  type: SAVE_USER_STORE_OBJECT_STORE_ERROR,
  data: error.message,
});

export const toggleObjectStoreForDelete = (objectStore: ObjectStore) => ({
  type: TOGGLE_OBJECT_STORE_FOR_DELETE,
  data: objectStore.id,
});
