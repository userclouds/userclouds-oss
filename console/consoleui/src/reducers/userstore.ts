import { v4 as uuidv4 } from 'uuid';
import { AnyAction } from 'redux';

import {
  GET_USER_STORE_CONFIG_REQUEST,
  GET_USER_STORE_CONFIG_SUCCESS,
  GET_USER_STORE_CONFIG_ERROR,
  FETCH_USER_STORE_COLUMN_REQUEST,
  FETCH_USER_STORE_COLUMN_ERROR,
  FETCH_USER_STORE_COLUMN_SUCCESS,
  CREATE_USER_STORE_COLUMN_REQUEST,
  CREATE_USER_STORE_COLUMN_SUCCESS,
  CREATE_USER_STORE_COLUMN_ERROR,
  UPDATE_USER_STORE_COLUMN_REQUEST,
  UPDATE_USER_STORE_COLUMN_SUCCESS,
  UPDATE_USER_STORE_COLUMN_ERROR,
  DELETE_USER_STORE_COLUMN_REQUEST,
  DELETE_USER_STORE_COLUMN_SUCCESS,
  DELETE_USER_STORE_COLUMN_ERROR,
  CHANGE_SELECTED_COLUMN,
  TOGGLE_COLUMN_EDIT_MODE,
  MODIFY_USER_STORE_COLUMN,
  MODIFY_USER_STORE_COLUMN_DEFAULT_TRANSFORMER,
  FETCH_USER_STORE_DISPLAY_COLUMNS_REQUEST,
  FETCH_USER_STORE_DISPLAY_COLUMNS_SUCCESS,
  FETCH_USER_STORE_DISPLAY_COLUMNS_ERROR,
  TOGGLE_COLUMN_PURPOSES_EDIT_MODE,
  GET_ACCESSORS_FOR_COLUMN_REQUEST,
  GET_ACCESSORS_FOR_COLUMN_SUCCESS,
  GET_ACCESSORS_FOR_COLUMN_ERROR,
  GET_ACCESSOR_METRICS_REQUEST,
  GET_ACCESSOR_METRICS_SUCCESS,
  GET_ACCESSOR_METRICS_ERROR,
  GET_COLUMN_DURATIONS_REQUEST,
  GET_COLUMN_DURATIONS_SUCCESS,
  GET_COLUMN_DURATIONS_ERROR,
  MODIFY_RETENTION_DURATION,
  UPDATE_COLUMN_DURATIONS_REQUEST,
  UPDATE_COLUMN_DURATIONS_SUCCESS,
  UPDATE_COLUMN_DURATIONS_ERROR,
  ADD_USER_STORE_COLUMN,
  CREATE_BULK_USER_STORE_COLUMN_ERROR,
  CREATE_BULK_USER_STORE_COLUMN_SUCCESS,
  DELETE_BULK_USER_STORE_COLUMN_ERROR,
  DELETE_BULK_USER_STORE_COLUMN_SUCCESS,
  MODIFY_BULK_USER_STORE_COLUMN,
  TOGGLE_USER_STORE_COLUMN_FOR_DELETE,
  TOGGLE_USER_STORE_EDIT_MODE,
  UPDATE_BULK_USER_STORE_COLUMN_ERROR,
  UPDATE_BULK_USER_STORE_COLUMN_SUCCESS,
  UPDATE_USER_STORE_CONFIG_ERROR,
  UPDATE_USER_STORE_CONFIG_REQUEST,
  UPDATE_USER_STORE_CONFIG_SUCCESS,
  CHANGE_COLUMN_SEARCH_FILTER,
  DELETE_DATA_TYPE_ERROR,
  DELETE_DATA_TYPE_SUCCESS,
  DELETE_DATA_TYPE_REQUEST,
  UPDATE_DATA_TYPE_ERROR,
  CHANGE_DATA_TYPE_SEARCH_FILTER,
  CHANGE_SELECTED_DATA_TYPE,
  CREATE_DATA_TYPE_ERROR,
  CREATE_DATA_TYPE_REQUEST,
  CREATE_DATA_TYPE_SUCCESS,
  FETCH_DATA_TYPE_ERROR,
  FETCH_DATA_TYPE_REQUEST,
  FETCH_DATA_TYPE_SUCCESS,
  MODIFY_DATA_TYPE,
  TOGGLE_DATA_TYPE_EDIT_MODE,
  TOGGLE_DATA_TYPE_FOR_DELETE,
  UPDATE_DATA_TYPE_REQUEST,
  UPDATE_DATA_TYPE_SUCCESS,
  FETCH_DATA_TYPES_REQUEST,
  FETCH_DATA_TYPES_SUCCESS,
  FETCH_DATA_TYPES_ERROR,
  LOAD_CREATE_DATA_TYPE_PAGE,
  MODIFY_DATA_TYPE_TO_CREATE,
  ADD_FIELD_TO_DATA_TYPE_TO_CREATE,
  ADD_FIELD_TO_DATA_TYPE,
  BULK_DELETE_DATA_TYPES_REQUEST,
  BULK_DELETE_DATA_TYPES_SUCCESS,
  BULK_DELETE_DATA_TYPES_FAILURE,
  SAVE_USER_STORE_DATABASE_REQUEST,
  SAVE_USER_STORE_DATABASE_SUCCESS,
  SAVE_USER_STORE_DATABASE_ERROR,
  MODIFY_USER_STORE_DATABASE,
  FETCH_USER_STORE_DATABASE_REQUEST,
  FETCH_USER_STORE_DATABASES_SUCCESS,
  FETCH_USER_STORE_DATABASE_ERROR,
  SAVE_USER_STORE_OBJECT_STORE_REQUEST,
  SAVE_USER_STORE_OBJECT_STORE_SUCCESS,
  SAVE_USER_STORE_OBJECT_STORE_ERROR,
  TOGGLE_EDIT_USER_STORE_OBJECT_STORE_MODE,
  MODIFY_USER_STORE_OBJECT_STORE,
  FETCH_USER_STORE_OBJECT_STORE_REQUEST,
  FETCH_USER_STORE_OBJECT_STORE_SUCCESS,
  FETCH_USER_STORE_OBJECT_STORE_ERROR,
  FETCH_USER_STORE_OBJECT_STORES_REQUEST,
  FETCH_USER_STORE_OBJECT_STORES_SUCCESS,
  FETCH_USER_STORE_OBJECT_STORES_ERROR,
  TOGGLE_OBJECT_STORE_FOR_DELETE,
  DELETE_DATABASE,
  SET_CURRENT_DATABASE,
  ADD_DATABASE,
  TEST_DATABASE_CONNECTION_REQUEST,
  TEST_DATABASE_CONNECTION_SUCCESS,
  TEST_DATABASE_CONNECTION_ERROR,
  CANCEL_DATABASE_DIALOG,
  RESET_DATABASE_DIALOG_STATE,
} from '../actions/userstore';
import {
  TOGGLE_MUTATOR_LIST_EDIT_MODE,
  BULK_EDIT_MUTATORS_REQUEST,
  BULK_EDIT_MUTATORS_SUCCESS,
  BULK_EDIT_MUTATORS_ERROR,
  TOGGLE_MUTATOR_FOR_DELETE,
  DELETE_MUTATOR_SUCCESS,
  DELETE_MUTATOR_ERROR,
  GET_TENANT_MUTATORS_REQUEST,
  GET_TENANT_MUTATORS_SUCCESS,
  GET_TENANT_MUTATORS_ERROR,
  GET_MUTATOR_REQUEST,
  GET_MUTATOR_SUCCESS,
  GET_MUTATOR_ERROR,
  MODIFY_MUTATOR_DETAILS,
  MODIFY_MUTATOR_TO_CREATE,
  TOGGLE_MUTATOR_DETAILS_EDIT_MODE,
  TOGGLE_MUTATOR_COLUMNS_EDIT_MODE,
  TOGGLE_MUTATOR_SELECTOR_EDIT_MODE,
  TOGGLE_MUTATOR_POLICIES_EDIT_MODE,
  UPDATE_MUTATOR_REQUEST,
  UPDATE_MUTATOR_SUCCESS,
  UPDATE_MUTATOR_ERROR,
  ADD_MUTATOR_COLUMN,
  TOGGLE_MUTATOR_COLUMN_FOR_DELETE,
  CHANGE_SELECTED_ACCESS_POLICY_FOR_MUTATOR,
  CHANGE_SELECTED_NORMALIZER_FOR_COLUMN,
  LOAD_CREATE_MUTATOR_PAGE,
  CREATE_MUTATOR_REQUEST,
  CREATE_MUTATOR_SUCCESS,
  CREATE_MUTATOR_ERROR,
  TOGGLE_MUTATOR_EDIT_MODE,
  CHANGE_MUTATOR_SEARCH_FILTER,
} from '../actions/mutators';
import {
  TOGGLE_ACCESSOR_LIST_EDIT_MODE,
  BULK_EDIT_ACCESSORS_REQUEST,
  BULK_EDIT_ACCESSORS_SUCCESS,
  BULK_EDIT_ACCESSORS_ERROR,
  TOGGLE_ACCESSOR_FOR_DELETE,
  DELETE_ACCESSOR_SUCCESS,
  DELETE_ACCESSOR_ERROR,
  GET_TENANT_ACCESSORS_REQUEST,
  GET_TENANT_ACCESSORS_SUCCESS,
  GET_TENANT_ACCESSORS_ERROR,
  GET_ACCESSOR_REQUEST,
  GET_ACCESSOR_SUCCESS,
  GET_ACCESSOR_ERROR,
  MODIFY_ACCESSOR_DETAILS,
  ADD_PURPOSE_TO_ACCESSOR,
  REMOVE_PURPOSE_FROM_ACCESSOR,
  MODIFY_ACCESSOR_TO_CREATE,
  MODIFY_COLUMN_IN_ACCESSOR_TO_CREATE,
  ADD_PURPOSE_TO_ACCESSOR_TO_CREATE,
  REMOVE_PURPOSE_FROM_ACCESSOR_TO_CREATE,
  TOGGLE_ACCESSOR_DETAILS_EDIT_MODE,
  TOGGLE_ACCESSOR_COLUMNS_EDIT_MODE,
  UPDATE_ACCESSOR_DETAILS_REQUEST,
  UPDATE_ACCESSOR_DETAILS_SUCCESS,
  UPDATE_ACCESSOR_DETAILS_ERROR,
  ADD_ACCESSOR_COLUMN,
  TOGGLE_ACCESSOR_COLUMN_FOR_DELETE,
  SAVE_ACCESSOR_COLUMNS_CONFIGURATION_REQUEST,
  SAVE_ACCESSOR_COLUMNS_CONFIGURATION_SUCCESS,
  SAVE_ACCESSOR_COLUMNS_CONFIGURATION_ERROR,
  CHANGE_SELECTED_ACCESS_POLICY_FOR_ACCESSOR,
  CHANGE_SELECTED_TRANSFORMER_FOR_COLUMN,
  CHANGE_SELECTED_TOKEN_ACCESS_POLICY_FOR_COLUMN,
  UPDATE_ACCESSOR_POLICIES_REQUEST,
  UPDATE_ACCESSOR_POLICIES_SUCCESS,
  UPDATE_ACCESSOR_POLICIES_ERROR,
  LOAD_CREATE_ACCESSOR_PAGE,
  CREATE_ACCESSOR_REQUEST,
  CREATE_ACCESSOR_SUCCESS,
  CREATE_ACCESSOR_ERROR,
  CHANGE_ACCESSOR_SEARCH_FILTER,
  CHANGE_ACCESSOR_LIST_INCLUDE_AUTOGENERATED,
  EXECUTE_ACCESSOR_CHANGE_CONTEXT,
  EXECUTE_ACCESSOR_CHANGE_SELECTOR_VALUES,
  EXECUTE_ACCESSOR_SUCCESS,
  EXECUTE_ACCESSOR_ERROR,
  EXECUTE_ACCESSOR_RESET,
  TOGGLE_ACCESSOR_EDIT_MODE,
} from '../actions/accessors';
import {
  GET_PURPOSES_REQUEST,
  GET_PURPOSES_SUCCESS,
  GET_PURPOSES_ERROR,
  CHANGE_SELECTED_PURPOSE,
  GET_PURPOSE_REQUEST,
  GET_PURPOSE_SUCCESS,
  GET_PURPOSE_ERROR,
  TOGGLE_PURPOSE_DETAILS_EDIT_MODE,
  MODIFY_PURPOSE_DETAILS,
  CREATE_PURPOSE_REQUEST,
  CREATE_PURPOSE_SUCCESS,
  CREATE_PURPOSE_ERROR,
  UPDATE_PURPOSE_REQUEST,
  UPDATE_PURPOSE_SUCCESS,
  UPDATE_PURPOSE_ERROR,
  DELETE_SINGLE_PURPOSE_REQUEST,
  DELETE_SINGLE_PURPOSE_SUCCESS,
  DELETE_SINGLE_PURPOSE_ERROR,
  TOGGLE_PURPOSE_BULK_EDIT_MODE,
  TOGGLE_PURPOSE_FOR_DELETE,
  BULK_DELETE_PURPOSES_REQUEST,
  BULK_DELETE_PURPOSES_FAILURE,
  BULK_DELETE_PURPOSES_SUCCESS,
  DELETE_PURPOSES_SINGLE_ERROR,
  DELETE_PURPOSES_SINGLE_SUCCESS,
  CHANGE_PURPOSE_SEARCH_FILTER,
} from '../actions/purposes';
import { RootState } from '../store';
import { NilUuid } from '../models/Uuids';
import { blankResourceID } from '../models/ResourceID';
import {
  Column,
  ColumnIndexType,
  columnsAreEqual,
  blankColumnConstraints,
  uniqueIDsAvailable,
  immutableAvailable,
  partialUpdatesAvailable,
  uniqueValuesAvailable,
  NativeDataTypes,
} from '../models/TenantUserStoreConfig';
import Accessor, {
  blankAccessor,
  AccessorColumn,
  columnToAccessorColumn,
} from '../models/Accessor';
import Mutator, { blankMutator, MutatorColumn } from '../models/Mutator';
import Transformer from '../models/Transformer';
import { PurposeRetentionDuration } from '../models/ColumnRetentionDurations';
import {
  blankCompositeField,
  blankDataType,
  DataType,
} from '../models/DataType';
import Purpose from '../models/Purpose';
import AccessPolicy from '../models/AccessPolicy';
import { measureDataPrivacyStats } from '../util/DataAnalysis';
import { getNewToggleEditValue, deepEqual } from './reducerHelper';
import { setOperatorsForFilter } from '../controls/SearchHelper';
import { blankSqlShimDatabase } from '../models/SqlshimDatabase';

const modifiedMutatorIsDirty = (
  selectedMutator: Mutator | undefined,
  modifiedMutator: Mutator | undefined,
  mutatorColumnsToAdd: MutatorColumn[],
  mutatorColumnsToDelete: Record<string, MutatorColumn>
) => {
  if (!selectedMutator || !modifiedMutator) {
    return false;
  }
  return (
    !deepEqual(selectedMutator, modifiedMutator) ||
    mutatorColumnsToAdd.length > 0 ||
    Object.keys(mutatorColumnsToDelete).length > 0
  );
};

const userStoreReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case GET_USER_STORE_CONFIG_REQUEST:
      state.fetchingUserStoreConfig = true;
      state.userStoreColumns = undefined;
      state.userStoreColumnsToModify = {};
      state.userStoreColumnsToAdd = [];
      state.userStoreColumnsToDelete = {};
      break;
    case GET_USER_STORE_CONFIG_SUCCESS:
      state.fetchingUserStoreConfig = false;
      state.userStoreColumns =
        action.data && action.data.columns ? action.data.columns : [];
      // we fetch after successful bulk edit
      // so don't reset save
      state.saveUserStoreConfigErrors = [];
      state.userStoreConfigIsDirty = false;
      break;
    case GET_USER_STORE_CONFIG_ERROR:
      state.fetchingUserStoreConfig = false;
      state.fetchUserStoreConfigError = action.data;
      break;
    case ADD_USER_STORE_COLUMN:
      state.userStoreColumnsToAdd = [
        ...state.userStoreColumnsToAdd,
        {
          id: uuidv4(),
          table: 'users',
          name: '',
          data_type: NativeDataTypes.String,
          access_policy: blankResourceID(),
          default_transformer: blankResourceID(),
          default_token_access_policy: blankResourceID(),
          is_array: false,
          index_type: ColumnIndexType.None,
          is_system: false,
          search_indexed: false,
          constraints: blankColumnConstraints(),
        },
      ];
      state.userStoreConfigIsDirty = true;
      break;
    case MODIFY_BULK_USER_STORE_COLUMN: {
      const {
        id,
        table,
        name,
        data_type,
        access_policy,
        default_transformer,
        default_token_access_policy,
        type,
        is_array,
        index_type,
        is_system,
        search_indexed,
        constraints,
      } = action.data;
      const matchingColumn = state.userStoreColumns?.find(
        (col: Column) => col.id === id
      );
      if (matchingColumn) {
        // this is a modification to a saved column,
        // not a new, unsaved column
        if (columnsAreEqual(matchingColumn, action.data)) {
          // we're restoring this item to its previous state
          // eslint-disable-next-line @typescript-eslint/no-unused-vars
          const { [id]: editedColumn, ...rest } =
            state.userStoreColumnsToModify;
          state.userStoreColumnsToModify = rest;
          state.userStoreConfigIsDirty = !!Object.keys(
            state.userStoreColumnsToModify
          ).length;
        } else {
          state.userStoreColumnsToModify = {
            ...state.userStoreColumnsToModify,
            [id]: action.data,
          };
          state.userStoreConfigIsDirty = true;
        }
      } else {
        state.userStoreColumnsToAdd = state.userStoreColumnsToAdd.map((col) => {
          if (col.id === id) {
            return {
              id,
              table,
              name,
              data_type,
              access_policy,
              default_transformer,
              default_token_access_policy,
              type,
              is_array,
              index_type,
              is_system,
              search_indexed,
              constraints,
            };
          }
          return col;
        });
        state.userStoreConfigIsDirty = true;
      }
      break;
    }
    case TOGGLE_USER_STORE_COLUMN_FOR_DELETE: {
      let newColumn = false;

      if (state.userStoreColumnsToModify[action.data.id]) {
        // removing a column with unsaved modifications
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        const { [action.data.id]: deletable, ...rest } =
          state.userStoreColumnsToModify;
        state.userStoreColumnsToModify = rest;
      } else {
        // check to see if we are removing an added column
        const matchingColumns = state.userStoreColumnsToAdd.filter(
          (col: Column) => col.id === action.data.id
        );
        newColumn = matchingColumns.length > 0;

        state.userStoreColumnsToAdd = state.userStoreColumnsToAdd.filter(
          (col: Column) => col.id !== action.data.id
        );
      }

      if (!newColumn) {
        const { [action.data.id]: isAlreadyQueued, ...rest } =
          state.userStoreColumnsToDelete;
        if (isAlreadyQueued) {
          state.userStoreColumnsToDelete = rest;
        } else {
          state.userStoreColumnsToDelete = {
            ...state.userStoreColumnsToDelete,
            [action.data.id]: action.data,
          };
        }
        if (Object.keys(state.userStoreColumnsToDelete).length) {
          state.userStoreConfigIsDirty = true;
        }
      }
      break;
    }
    case UPDATE_USER_STORE_CONFIG_REQUEST:
      state.savingUserStoreConfig = true;
      state.saveUserStoreConfigSuccess = '';
      state.saveUserStoreConfigErrors = [];
      break;
    case UPDATE_USER_STORE_CONFIG_SUCCESS:
      state.savingUserStoreConfig = false;
      state.saveUserStoreConfigSuccess = 'Successfully saved.';
      state.saveUserStoreConfigErrors = [];
      // we have to fetch again in a thunk
      state.userStoreColumns = undefined;
      state.userStoreConfigIsDirty = false;
      state.userStoreEditMode = false;
      break;
    case UPDATE_USER_STORE_CONFIG_ERROR:
      state.savingUserStoreConfig = false;
      break;

    case CHANGE_COLUMN_SEARCH_FILTER:
      state.columnSearchFilter = {
        ...state.columnSearchFilter,
        ...setOperatorsForFilter(action.data),
      };
      break;
    case TOGGLE_COLUMN_EDIT_MODE: {
      state.columnEditMode = getNewToggleEditValue(
        action.data,
        state.columnEditMode
      );
      state.savingColumnSuccess = '';
      state.saveColumnError = '';
      if (state.selectedColumn) {
        state.modifiedColumn = { ...state.selectedColumn };
      }
      break;
    }
    case MODIFY_USER_STORE_COLUMN:
      state.modifiedColumn = {
        ...state.modifiedColumn,
        ...action.data,
      };
      if (state.modifiedColumn && state.dataTypes) {
        if (!uniqueIDsAvailable(state.modifiedColumn, state.dataTypes.data)) {
          state.modifiedColumn.constraints.unique_id_required = false;
        }
        if (!immutableAvailable(state.modifiedColumn, state.dataTypes.data)) {
          state.modifiedColumn.constraints.immutable_required = false;
        }
        if (!partialUpdatesAvailable(state.modifiedColumn)) {
          state.modifiedColumn.constraints.partial_updates = false;
        }
        if (
          !uniqueValuesAvailable(state.modifiedColumn, state.dataTypes.data)
        ) {
          state.modifiedColumn.constraints.unique_required = false;
        }
      }
      state.columnIsDirty =
        JSON.stringify(state.selectedColumn) !==
        JSON.stringify(state.modifiedColumn);
      {
        const newData = JSON.stringify(state.modifiedColumn);
        state.modifiedColumn = JSON.parse(newData) as Column;
      }
      break;
    case MODIFY_USER_STORE_COLUMN_DEFAULT_TRANSFORMER:
      if (state.transformers && state.modifiedColumn) {
        const transformer = state.transformers.data.find(
          (t) => t.id === action.data
        );
        if (transformer) {
          state.modifiedColumn.default_transformer = {
            id: transformer.id,
            name: transformer.name,
          };
          state.columnIsDirty =
            JSON.stringify(state.selectedColumn) !==
            JSON.stringify(state.modifiedColumn);
          state.modifiedColumn = { ...state.modifiedColumn };
        }
      }
      break;
    case FETCH_USER_STORE_DISPLAY_COLUMNS_REQUEST:
      state.fetchingColumn = true;
      state.fetchingColumnError = '';
      state.saveColumnError = '';
      state.savingColumnSuccess = '';
      state.saveUserStoreConfigSuccess = '';
      state.saveUserStoreConfigErrors = [];
      break;
    case FETCH_USER_STORE_DISPLAY_COLUMNS_SUCCESS:
      state.fetchingColumn = false;
      state.fetchingColumnError = '';
      state.userStoreDisplayColumns = action.data;
      break;
    case FETCH_USER_STORE_DISPLAY_COLUMNS_ERROR:
      state.fetchingColumn = false;
      state.fetchingColumnError = action.data;
      break;
    case CHANGE_SELECTED_COLUMN:
      state.selectedColumn = action.data;
      state.modifiedColumn = action.data;
      state.columnEditMode = false;
      state.fetchingColumnError = '';
      state.saveColumnError = '';
      state.savingColumnSuccess = '';
      break;
    case TOGGLE_COLUMN_PURPOSES_EDIT_MODE: {
      state.columnPurposesEditMode = getNewToggleEditValue(
        action.data,
        state.columnPurposesEditMode
      );
      state.retentionDurationsSaveError = '';
      state.purposeSettingsAreDirty = false;
      if (state.columnRetentionDurations) {
        state.modifiedRetentionDurations = [
          ...state.columnRetentionDurations.purpose_retention_durations,
        ];
      }
      break;
    }
    case GET_ACCESSOR_METRICS_REQUEST:
      state.fetchingAccessorMetrics = true;
      state.fetchAccessorMetricsError = '';
      state.accessorMetrics = undefined;
      break;
    case GET_ACCESSOR_METRICS_SUCCESS:
      state.fetchingAccessorMetrics = false;
      state.accessorMetrics = action.data;
      break;
    case GET_ACCESSOR_METRICS_ERROR:
      state.fetchingAccessorMetrics = false;
      state.fetchAccessorMetricsError = action.data;
      break;
    case GET_ACCESSORS_FOR_COLUMN_REQUEST:
      state.fetchingAccessors = true;
      state.fetchAccessorsError = '';
      state.saveAccessorColumnsError = '';
      state.saveAccessorColumnsSuccess = '';
      state.saveAccessorError = '';
      state.saveAccessorPoliciesError = '';
      state.saveAccessorPoliciesSuccess = '';
      state.saveAccessorSuccess = '';
      state.createAccessorError = '';
      state.bulkUpdateAccessorsErrors = [];
      state.accessorsForColumn = undefined;
      break;
    case GET_ACCESSORS_FOR_COLUMN_SUCCESS:
      state.fetchingAccessors = false;
      state.accessorsForColumn = action.data;
      break;
    case GET_ACCESSORS_FOR_COLUMN_ERROR:
      state.fetchingAccessors = false;
      state.fetchAccessorsError = action.data;
      break;
    case GET_COLUMN_DURATIONS_REQUEST:
      state.fetchingColumnRetentionDurations = true;
      state.columnDurationsFetchError = '';
      break;
    case GET_COLUMN_DURATIONS_SUCCESS: {
      const { purpose_retention_durations } = action.data;
      state.fetchingColumnRetentionDurations = false;
      state.columnRetentionDurations = action.data;
      state.modifiedRetentionDurations = [...purpose_retention_durations];
      state.purposeSettingsAreDirty = false;
      break;
    }
    case GET_COLUMN_DURATIONS_ERROR:
      state.fetchingColumnRetentionDurations = false;
      state.columnDurationsFetchError = action.data;
      break;
    case MODIFY_RETENTION_DURATION: {
      const { purpose_id } = action.data;
      state.modifiedRetentionDurations = state.modifiedRetentionDurations.map(
        (duration: PurposeRetentionDuration) =>
          duration.purpose_id === purpose_id ? action.data : duration
      );
      state.purposeSettingsAreDirty =
        JSON.stringify(
          state.columnRetentionDurations?.purpose_retention_durations
        ) !== JSON.stringify(state.modifiedRetentionDurations);
      break;
    }
    case UPDATE_COLUMN_DURATIONS_REQUEST:
      state.retentionDurationsSaveError = '';
      state.savingColumnRetentionDurations = true;
      break;
    case UPDATE_COLUMN_DURATIONS_SUCCESS: {
      const { purpose_retention_durations } = action.data;
      state.columnRetentionDurations = action.data;
      state.savingColumnRetentionDurations = false;
      state.modifiedRetentionDurations = [...purpose_retention_durations];
      state.columnPurposesEditMode = false;
      state.retentionDurationsSaveSuccess = true;
      break;
    }
    case UPDATE_COLUMN_DURATIONS_ERROR:
      state.retentionDurationsSaveError = action.data;
      state.savingColumnRetentionDurations = false;
      break;
    case FETCH_USER_STORE_COLUMN_REQUEST:
      state.fetchingColumn = true;
      state.fetchingColumnError = '';
      state.saveColumnError = '';
      state.savingColumnSuccess = '';
      state.saveUserStoreConfigSuccess = '';
      state.saveUserStoreConfigErrors = [];
      break;
    case FETCH_USER_STORE_COLUMN_SUCCESS:
      state.fetchingColumn = false;
      state.fetchingColumnError = '';
      state.selectedColumn = action.data;
      state.modifiedColumn = action.data;
      break;
    case FETCH_USER_STORE_COLUMN_ERROR:
      state.fetchingColumn = false;
      state.fetchingColumnError = action.data;
      break;
    case CREATE_USER_STORE_COLUMN_REQUEST:
      state.savingColumn = true;
      state.saveColumnError = '';
      break;
    case CREATE_USER_STORE_COLUMN_SUCCESS:
      state.savingColumn = false;
      state.selectedColumn = action.data;
      break;
    case CREATE_USER_STORE_COLUMN_ERROR:
      state.savingColumn = false;
      state.saveColumnError = action.data;
      break;
    case UPDATE_USER_STORE_COLUMN_REQUEST:
      state.savingColumn = true;
      state.saveColumnError = '';
      break;
    case UPDATE_USER_STORE_COLUMN_SUCCESS:
      state.savingColumn = false;
      state.selectedColumn = action.data;
      state.columnEditMode = false;
      state.savingColumnSuccess = 'Column successfully saved.';
      break;
    case UPDATE_USER_STORE_COLUMN_ERROR:
      state.savingColumn = false;
      state.saveColumnError = action.data;
      break;
    case DELETE_USER_STORE_COLUMN_REQUEST:
      state.savingColumn = true;
      state.saveColumnError = '';
      break;
    case DELETE_USER_STORE_COLUMN_SUCCESS:
      state.savingColumn = false;
      state.saveColumnError = '';
      state.selectedColumn = undefined;
      state.columnEditMode = false;
      break;
    case DELETE_USER_STORE_COLUMN_ERROR:
      state.savingColumn = false;
      state.saveColumnError = action.data;
      break;
    case TOGGLE_USER_STORE_EDIT_MODE:
      state.userStoreEditMode = !state.userStoreEditMode;
      state.userStoreColumnsToAdd = [];
      state.userStoreColumnsToModify = {};
      state.userStoreColumnsToDelete = {};
      state.saveUserStoreConfigErrors = [];
      state.saveColumnError = '';
      state.fetchingColumnError = '';
      state.savingColumnSuccess = '';
      state.saveUserStoreConfigSuccess = '';
      break;
    case CREATE_BULK_USER_STORE_COLUMN_SUCCESS:
      // NO-OP. We handle success in bulk
      break;
    case CREATE_BULK_USER_STORE_COLUMN_ERROR:
      state.saveUserStoreConfigErrors = [
        ...state.saveUserStoreConfigErrors,
        action.data,
      ];
      break;
    case UPDATE_BULK_USER_STORE_COLUMN_SUCCESS:
      // NO-OP. We handle success in bulk
      break;
    case UPDATE_BULK_USER_STORE_COLUMN_ERROR:
      state.saveUserStoreConfigErrors = [
        ...state.saveUserStoreConfigErrors,
        action.data,
      ];
      break;
    case DELETE_BULK_USER_STORE_COLUMN_SUCCESS:
      // NO-OP. We handle success in bulk
      break;
    case DELETE_BULK_USER_STORE_COLUMN_ERROR:
      state.saveUserStoreConfigErrors = [
        ...state.saveUserStoreConfigErrors,
        action.data,
      ];
      break;
    case FETCH_DATA_TYPES_REQUEST:
      state.fetchingDataTypes = true;
      state.fetchingDataTypeError = '';
      state.savingDataTypeSuccess = '';
      break;
    case FETCH_DATA_TYPES_SUCCESS:
      state.fetchingDataTypes = false;
      state.fetchingDataTypeError = '';
      state.dataTypes = action.data;
      break;
    case FETCH_DATA_TYPES_ERROR:
      state.fetchingDataTypes = false;
      state.fetchingDataTypeError = action.data;
      break;
    case TOGGLE_DATA_TYPE_FOR_DELETE:
      if (state.dataTypesDeleteQueue.includes(action.data)) {
        state.dataTypesDeleteQueue = state.dataTypesDeleteQueue.filter(
          (id: string) => id !== action.data
        );
      } else {
        state.dataTypesDeleteQueue = [
          ...state.dataTypesDeleteQueue,
          action.data,
        ];
      }
      break;
    case CHANGE_DATA_TYPE_SEARCH_FILTER:
      state.dataTypeSearchFilter = {
        ...state.dataTypeSearchFilter,
        ...setOperatorsForFilter(action.data),
      };
      break;
    case TOGGLE_DATA_TYPE_EDIT_MODE: {
      state.dataTypeEditMode = getNewToggleEditValue(
        action.data,
        state.dataTypeEditMode
      );
      state.savingDataTypeSuccess = '';
      state.saveDataTypeError = '';
      if (state.selectedDataType) {
        state.modifiedDataType = { ...state.selectedDataType };
      }
      break;
    }
    case MODIFY_DATA_TYPE:
      state.modifiedDataType = {
        ...state.modifiedDataType,
        ...action.data,
      };
      state.dataTypeIsDirty = !deepEqual(
        state.selectedDataType,
        state.modifiedDataType
      );
      break;
    case CHANGE_SELECTED_DATA_TYPE:
      state.selectedDataType = action.data;
      state.modifiedDataType = action.data;
      state.dataTypeEditMode = false;
      state.fetchingDataTypeError = '';
      state.saveDataTypeError = '';
      state.savingDataTypeSuccess = '';
      break;
    case FETCH_DATA_TYPE_REQUEST:
      state.fetchingDataType = true;
      state.fetchingDataTypeError = '';
      state.saveDataTypeError = '';
      state.savingDataTypeSuccess = '';
      state.saveUserStoreConfigSuccess = '';
      state.saveUserStoreConfigErrors = [];
      break;
    case FETCH_DATA_TYPE_SUCCESS:
      state.fetchingDataType = false;
      state.fetchingDataTypeError = '';
      state.selectedDataType = action.data;
      state.modifiedDataType = action.data;
      break;
    case FETCH_DATA_TYPE_ERROR:
      state.fetchingDataType = false;
      state.fetchingDataTypeError = action.data;
      break;
    case CREATE_DATA_TYPE_REQUEST:
      state.savingDataType = true;
      state.saveDataTypeError = '';
      break;
    case CREATE_DATA_TYPE_SUCCESS:
      state.savingDataType = false;
      state.selectedDataType = action.data;
      break;
    case CREATE_DATA_TYPE_ERROR:
      state.savingDataType = false;
      state.saveDataTypeError = action.data;
      break;
    case UPDATE_DATA_TYPE_REQUEST:
      state.savingDataType = true;
      state.saveDataTypeError = '';
      break;
    case UPDATE_DATA_TYPE_SUCCESS:
      state.savingDataType = false;
      state.selectedDataType = action.data;
      state.dataTypeEditMode = false;
      state.savingDataTypeSuccess = 'DataType successfully saved.';
      break;
    case UPDATE_DATA_TYPE_ERROR:
      state.savingDataType = false;
      state.saveDataTypeError = action.data;
      break;
    case DELETE_DATA_TYPE_REQUEST:
      state.savingDataType = true;
      state.saveDataTypeError = '';
      break;
    case DELETE_DATA_TYPE_SUCCESS:
      state.savingDataType = false;
      state.saveDataTypeError = '';
      state.selectedDataType = undefined;
      state.dataTypeEditMode = false;
      break;
    case DELETE_DATA_TYPE_ERROR:
      state.savingDataType = false;
      state.saveDataTypeError = action.data;
      break;
    case LOAD_CREATE_DATA_TYPE_PAGE:
      state.dataTypeToCreate = blankDataType();
      state.saveDataTypeError = '';
      break;
    case MODIFY_DATA_TYPE_TO_CREATE:
      {
        state.dataTypeToCreate = {
          ...state.dataTypeToCreate,
          ...action.data,
        };
        const newData = JSON.stringify(state.dataTypeToCreate);
        state.dataTypeToCreate = JSON.parse(newData) as DataType;
      }
      break;
    case ADD_FIELD_TO_DATA_TYPE_TO_CREATE:
      {
        state.dataTypeToCreate.composite_attributes.fields = [
          ...state.dataTypeToCreate.composite_attributes.fields,
          blankCompositeField(),
        ];
        const newData = JSON.stringify(state.dataTypeToCreate);
        state.dataTypeToCreate = JSON.parse(newData) as DataType;
      }
      break;
    case ADD_FIELD_TO_DATA_TYPE:
      if (state.modifiedDataType) {
        state.modifiedDataType.composite_attributes.fields = [
          ...state.modifiedDataType.composite_attributes.fields,
          blankCompositeField(),
        ];
        const newData = JSON.stringify(state.modifiedDataType);
        state.modifiedDataType = JSON.parse(newData) as DataType;
        state.dataTypeIsDirty = !deepEqual(
          state.selectedDataType,
          state.modifiedDataType
        );
      }
      break;
    case BULK_DELETE_DATA_TYPES_REQUEST:
      state.savingDataType = true;
      state.saveDataTypeError = '';
      state.savingDataTypeSuccess = '';
      break;
    case BULK_DELETE_DATA_TYPES_SUCCESS:
      state.savingDataType = false;
      state.savingDataTypeSuccess = 'Successfully deleted data types.';
      state.dataTypeEditMode = false;
      break;
    case BULK_DELETE_DATA_TYPES_FAILURE:
      state.savingDataType = false;
      break;
    case CHANGE_ACCESSOR_SEARCH_FILTER:
      state.accessorSearchFilter = {
        ...state.accessorSearchFilter,
        ...setOperatorsForFilter(action.data),
      };
      break;
    case CHANGE_ACCESSOR_LIST_INCLUDE_AUTOGENERATED:
      state.accessorListIncludeAutogenerated = action.data;
      break;
    case GET_TENANT_ACCESSORS_REQUEST:
      state.fetchingAccessors = true;
      state.fetchAccessorsError = '';
      state.selectedAccessorID = undefined;
      state.selectedAccessor = undefined;
      state.modifiedAccessor = undefined;
      state.saveAccessorColumnsSuccess = '';
      state.saveAccessorColumnsError = '';
      state.saveAccessorError = '';
      state.saveAccessorPoliciesError = '';
      state.saveAccessorPoliciesSuccess = '';
      state.saveAccessorSuccess = '';
      state.createAccessorError = '';
      state.bulkUpdateAccessorsErrors = [];
      break;
    case GET_TENANT_ACCESSORS_SUCCESS:
      state.fetchingAccessors = false;
      state.accessors = action.data;
      break;
    case GET_TENANT_ACCESSORS_ERROR:
      state.fetchingAccessors = false;
      state.fetchAccessorsError = action.data;
      break;
    case GET_ACCESSOR_REQUEST:
      state.fetchingAccessors = true;
      state.fetchAccessorsError = '';
      state.selectedAccessor = undefined;
      state.modifiedAccessor = undefined;
      state.modifiedAccessorIsDirty = !deepEqual(
        state.selectedAccessor,
        state.modifiedAccessor
      );
      state.selectedAccessorID = action.data;
      state.userStoreColumnsToAdd = [];
      state.userStoreColumnsToDelete = {};
      state.saveAccessorColumnsSuccess = '';
      state.saveAccessorColumnsError = '';
      break;
    case GET_ACCESSOR_SUCCESS: {
      state.fetchingAccessors = false;
      state.selectedAccessor = action.data as Accessor;
      state.selectedAccessorID = (action.data as Accessor).id;
      state.modifiedAccessor = action.data as Accessor;
      state.modifiedAccessorIsDirty = !deepEqual(
        state.selectedAccessor,
        state.modifiedAccessor
      );
      state.accessorDetailsEditMode = false;
      state.saveAccessorColumnsSuccess = '';
      state.saveAccessorColumnsError = '';
      break;
    }
    case GET_ACCESSOR_ERROR:
      state.fetchingAccessors = false;
      state.fetchAccessorsError = action.data;
      break;
    case MODIFY_ACCESSOR_DETAILS:
      state.modifiedAccessor = {
        ...state.modifiedAccessor,
        ...action.data,
      };
      state.modifiedAccessorIsDirty = !deepEqual(
        state.selectedAccessor,
        state.modifiedAccessor
      );

      break;
    case ADD_PURPOSE_TO_ACCESSOR:
      if (state.purposes && state.modifiedAccessor) {
        const selectedPurpose = state.purposes.data.find(
          (p: Purpose) => p.id === action.data
        );
        if (selectedPurpose) {
          state.modifiedAccessor = {
            ...state.modifiedAccessor,
            purposes: [
              ...(state.modifiedAccessor.purposes
                ? state.modifiedAccessor.purposes
                : []),
              selectedPurpose,
            ],
          };
        }

        state.modifiedAccessorIsDirty = !deepEqual(
          state.selectedAccessor,
          state.modifiedAccessor
        );
      }
      break;
    case REMOVE_PURPOSE_FROM_ACCESSOR: {
      if (state.modifiedAccessor) {
        let newPurposes = [...state.modifiedAccessor.purposes];
        newPurposes = newPurposes.filter((purpose) => {
          return purpose.id !== action.data.id;
        });
        state.modifiedAccessor = {
          ...state.modifiedAccessor,
          purposes: newPurposes,
        };
        state.modifiedAccessorIsDirty = !deepEqual(
          state.selectedAccessor,
          state.modifiedAccessor
        );
      }
      break;
    }
    case TOGGLE_ACCESSOR_DETAILS_EDIT_MODE: {
      state.accessorDetailsEditMode = getNewToggleEditValue(
        action.data,
        state.accessorDetailsEditMode
      );
      state.accessorSelectorEditMode = false;
      if (state.selectedAccessor && state.modifiedAccessor) {
        state.modifiedAccessor = { ...state.selectedAccessor };
      }
      break;
    }
    case MODIFY_ACCESSOR_TO_CREATE:
      state.accessorAddColumnDropdownValue = '';
      state.accessorToCreate = {
        ...state.accessorToCreate,
        ...action.data,
      };
      break;

    case MODIFY_COLUMN_IN_ACCESSOR_TO_CREATE:
      {
        const { columnID, columnData } = action.data;
        const newColumns = state.accessorToCreate.columns.map((col) => {
          if (col.id === columnID) {
            return {
              ...col,
              ...columnData,
            };
          }
          return col;
        });
        state.accessorToCreate = {
          ...state.accessorToCreate,
          columns: newColumns,
        };
      }
      break;

    case ADD_PURPOSE_TO_ACCESSOR_TO_CREATE:
      if (state.purposes && state.accessorToCreate) {
        const selectedPurpose = state.purposes.data.find(
          (p: Purpose) => p.id === action.data
        );
        if (selectedPurpose) {
          state.accessorToCreate = {
            ...state.accessorToCreate,
            purposes: [
              ...(state.accessorToCreate.purposes
                ? state.accessorToCreate.purposes
                : []),
              selectedPurpose,
            ],
          };
        }
      }
      break;
    case REMOVE_PURPOSE_FROM_ACCESSOR_TO_CREATE:
      {
        let newPurposes = state.accessorToCreate
          ? [...state.accessorToCreate.purposes]
          : [];
        newPurposes = newPurposes.filter((purpose) => {
          return purpose.id !== action.data.id;
        });
        state.accessorToCreate = {
          ...state.accessorToCreate,
          purposes: newPurposes,
        };
      }
      break;
    case TOGGLE_ACCESSOR_COLUMNS_EDIT_MODE: {
      state.accessorColumnsEditMode = action.data;
      state.accessorColumnsToAdd = [];
      state.accessorColumnsToDelete = {};
      state.accessorAddColumnDropdownValue = '';
      state.saveAccessorColumnsSuccess = '';
      state.saveAccessorColumnsError = '';
      state.accessorColumnsToAdd = [];
      state.accessorColumnsToDelete = {};
      const {
        id,
        name,
        description,
        data_life_cycle_state,
        columns,
        access_policy,
        selector_config,
        purposes,
        version,
        is_system,
        is_audit_logged,
        are_column_access_policies_overridden,
        use_search_index,
      } = state.selectedAccessor as Accessor;
      state.modifiedAccessor = {
        id,
        name,
        description,
        data_life_cycle_state,
        columns,
        access_policy: access_policy,
        selector_config,
        purposes,
        version,
        is_system,
        is_audit_logged,
        are_column_access_policies_overridden,
        use_search_index,
      };
      state.modifiedAccessorIsDirty = !deepEqual(
        state.selectedAccessor,
        state.modifiedAccessor
      );
      break;
    }
    case TOGGLE_ACCESSOR_EDIT_MODE: {
      /* this is the code from TOGGLE_ACCESSOR_DETAILS_EDIT_MODE */
      state.accessorDetailsEditMode = getNewToggleEditValue(
        action.data,
        state.accessorDetailsEditMode
      );
      state.accessorSelectorEditMode = false;
      if (state.selectedAccessor && state.modifiedAccessor) {
        state.modifiedAccessor = { ...state.selectedAccessor };
      }
      /* end TOGGLE_ACCESSOR_DETAILS_EDIT_MODE */

      state.accessorColumnsEditMode = state.accessorDetailsEditMode;
      state.accessorColumnsToAdd = [];
      state.accessorColumnsToDelete = {};
      state.accessorAddColumnDropdownValue = '';
      state.saveAccessorColumnsSuccess = '';
      state.saveAccessorColumnsError = '';
      state.accessorColumnsToAdd = [];
      state.accessorColumnsToDelete = {};
      const {
        id,
        name,
        description,
        data_life_cycle_state,
        columns,
        access_policy,
        selector_config,
        purposes,
        version,
        is_system,
        is_audit_logged,
        are_column_access_policies_overridden,
        use_search_index,
      } = state.selectedAccessor as Accessor;
      state.modifiedAccessor = {
        id,
        name,
        description,
        data_life_cycle_state,
        columns,
        access_policy: access_policy,
        selector_config,
        purposes,
        version,
        is_system,
        is_audit_logged,
        are_column_access_policies_overridden,
        use_search_index,
      };
      state.modifiedAccessorIsDirty = !deepEqual(
        state.selectedAccessor,
        state.modifiedAccessor
      );
      break;
    }
    case LOAD_CREATE_ACCESSOR_PAGE:
      state.accessorToCreate = blankAccessor();
      state.createAccessorError = '';
      break;
    case CREATE_ACCESSOR_REQUEST:
    case UPDATE_ACCESSOR_DETAILS_REQUEST:
      state.savingAccessor = true;
      state.saveAccessorSuccess = '';
      state.saveAccessorError = '';
      break;
    case UPDATE_ACCESSOR_DETAILS_SUCCESS: {
      state.savingAccessor = false;
      state.selectedAccessor = action.data as Accessor;
      state.modifiedAccessor = action.data as Accessor;
      state.modifiedAccessorIsDirty = !deepEqual(
        state.selectedAccessor,
        state.modifiedAccessor
      );
      state.accessorDetailsEditMode = false;
      state.accessorSelectorEditMode = false;
      state.saveAccessorSuccess = 'Successfully saved accessor';
      break;
    }
    case CREATE_ACCESSOR_SUCCESS:
      state.savingAccessor = false;
      state.accessorToCreate = blankAccessor();
      break;
    case CREATE_ACCESSOR_ERROR:
      state.savingAccessor = false;
      state.createAccessorError = action.data;
      break;
    case UPDATE_ACCESSOR_DETAILS_ERROR:
      state.savingAccessor = false;
      state.saveAccessorError = action.data;
      break;
    case TOGGLE_ACCESSOR_COLUMN_FOR_DELETE: {
      const newlyAdded = state.accessorColumnsToAdd.findIndex(
        (col: AccessorColumn) => col.id === action.data.id
      );
      if (state.modifiedAccessor) {
        const newlyAddedModifiedAccessor =
          state.modifiedAccessor.columns.findIndex(
            (col: AccessorColumn) => col.id === action.data.id
          );
        state.modifiedAccessor.columns.splice(newlyAddedModifiedAccessor, 1);
        state.modifiedAccessor = {
          ...state.modifiedAccessor,
          columns: [...state.modifiedAccessor.columns],
        };
        state.modifiedAccessorIsDirty = !deepEqual(
          state.selectedAccessor,
          state.modifiedAccessor
        );
      }
      if (newlyAdded > -1) {
        // removing an unsaved column
        state.accessorColumnsToAdd.splice(newlyAdded, 1);
        state.accessorColumnsToAdd = [...state.accessorColumnsToAdd];
      } else {
        const { [action.data.id]: isAlreadyQueued, ...restOfDelete } =
          state.accessorColumnsToDelete;
        if (isAlreadyQueued) {
          // unqueueing a queued column
          state.accessorColumnsToDelete = restOfDelete;
        } else {
          // removing a persisted column
          state.accessorColumnsToDelete = {
            ...state.accessorColumnsToDelete,
            [action.data.id]: action.data,
          };
        }
      }
      break;
    }
    case ADD_ACCESSOR_COLUMN: {
      state.accessorAddColumnDropdownValue = action.data;
      const column = state.userStoreColumns?.find(
        (col: Column) => col.id === action.data
      );
      if (column) {
        // we can naively push because the dropdown shouldn't display
        // columns that already exist on accessor or are queued
        state.accessorColumnsToAdd = [
          ...state.accessorColumnsToAdd,
          {
            id: column.id,
            name: column.name,
            table: column.table,
            data_type_id: column.data_type.id,
            data_type_name: column.data_type.name,
            is_array: column.is_array,
            transformer_id: '',
            transformer_name: '',
            default_access_policy_id: column.access_policy.id,
            default_access_policy_name: column.access_policy.name,
            token_access_policy_id: column.default_token_access_policy.id,
            token_access_policy_name: column.default_token_access_policy.name,
            default_transformer_name: column.default_transformer.name,
          },
        ];
        state.accessorAddColumnDropdownValue = '';
        if (state.modifiedAccessor) {
          const columns = [
            ...state.modifiedAccessor.columns,
            columnToAccessorColumn(column),
          ];
          state.modifiedAccessor = {
            ...state.modifiedAccessor,
            columns: columns,
          };
        }
      }
      state.modifiedAccessorIsDirty = !deepEqual(
        state.selectedAccessor,
        state.modifiedAccessor
      );

      break;
    }
    case CHANGE_SELECTED_TRANSFORMER_FOR_COLUMN: {
      const { columnID, transformerID, isNew } = action.data;
      const matchingTransformer =
        transformerID === NilUuid
          ? blankResourceID()
          : state.transformers?.data?.find(
              (t: Transformer) => t.id === transformerID
            );
      if (matchingTransformer) {
        if (state.modifiedAccessor) {
          state.modifiedAccessor = {
            ...state.modifiedAccessor,
            columns: state.modifiedAccessor.columns.map((c: AccessorColumn) => {
              if (c.id === columnID) {
                return {
                  ...c,
                  transformer_id: matchingTransformer.id,
                  transformer_name: matchingTransformer.name,
                };
              }
              return c;
            }),
          };
          state.modifiedAccessorIsDirty = !deepEqual(
            state.selectedAccessor,
            state.modifiedAccessor
          );
        }
        if (isNew) {
          state.accessorColumnsToAdd = state.accessorColumnsToAdd?.map(
            (c: AccessorColumn) => {
              if (c.id === columnID) {
                return {
                  ...c,
                  transformer_id: matchingTransformer.id,
                  transformer_name: matchingTransformer.name,
                };
              }
              return c;
            }
          );
        }
      }
      break;
    }
    case CHANGE_SELECTED_TOKEN_ACCESS_POLICY_FOR_COLUMN: {
      const { columnID, policyID, isNew } = action.data;
      const matchingPolicy = state.allAccessPolicies?.data?.find(
        (p: AccessPolicy) => p.id === policyID
      );
      if (matchingPolicy) {
        if (state.modifiedAccessor) {
          state.modifiedAccessor = {
            ...state.modifiedAccessor,
            columns: state.modifiedAccessor.columns.map((c: AccessorColumn) => {
              if (c.id === columnID) {
                return {
                  ...c,
                  token_access_policy_id: matchingPolicy.id,
                  token_access_policy_name: matchingPolicy.name,
                };
              }
              return c;
            }),
          };
          state.modifiedAccessorIsDirty = !deepEqual(
            state.selectedAccessor,
            state.modifiedAccessor
          );
        }
        if (isNew) {
          state.accessorColumnsToAdd = state.accessorColumnsToAdd?.map(
            (c: AccessorColumn) => {
              if (c.id === columnID) {
                return {
                  ...c,
                  token_access_policy_id: matchingPolicy.id,
                  token_access_policy_name: matchingPolicy.name,
                };
              }
              return c;
            }
          );
        }
      }
      break;
    }

    case SAVE_ACCESSOR_COLUMNS_CONFIGURATION_REQUEST:
      state.saveAccessorColumnsSuccess = '';
      state.saveAccessorColumnsError = '';
      state.savingAccessor = true;
      break;
    case SAVE_ACCESSOR_COLUMNS_CONFIGURATION_SUCCESS:
      state.saveAccessorColumnsSuccess =
        'Column configuration successfully updated';
      state.savingAccessor = false;
      state.accessorAddColumnDropdownValue = '';
      state.accessorColumnsEditMode = false;
      state.accessors = undefined;
      state.selectedAccessor = action.data as Accessor;
      state.modifiedAccessor = action.data as Accessor;
      state.accessorColumnsToAdd = [];
      state.accessorColumnsToDelete = {};
      break;
    case SAVE_ACCESSOR_COLUMNS_CONFIGURATION_ERROR:
      state.savingAccessor = false;
      state.saveAccessorColumnsError = action.data;
      break;
    case CHANGE_SELECTED_ACCESS_POLICY_FOR_ACCESSOR:
      state.selectedAccessPolicyForAccessor = action.data;
      state.modifiedAccessorIsDirty = true;

      break;
    case UPDATE_ACCESSOR_POLICIES_REQUEST:
      state.savingAccessor = true;
      state.saveAccessorPoliciesSuccess = '';
      state.saveAccessorPoliciesError = '';
      break;
    case UPDATE_ACCESSOR_POLICIES_SUCCESS:
      state.savingAccessor = false;
      state.saveAccessorPoliciesSuccess =
        'Successfully saved accessor policies';
      state.accessorPoliciesEditMode = false;
      state.selectedAccessor = action.data as Accessor;
      state.modifiedAccessor = action.data as Accessor;
      break;
    case UPDATE_ACCESSOR_POLICIES_ERROR:
      state.savingAccessor = false;
      state.saveAccessorPoliciesError = action.data;
      break;
    case GET_TENANT_MUTATORS_REQUEST:
      state.fetchingMutators = true;
      state.fetchMutatorsError = '';
      state.selectedMutatorID = undefined;
      state.selectedMutator = undefined;
      state.modifiedMutator = undefined;
      state.saveMutatorColumnsSuccess = '';
      state.saveMutatorColumnsError = '';
      break;
    case TOGGLE_ACCESSOR_FOR_DELETE: {
      const { [action.data.id]: alreadyQueued, ...rest } =
        state.accessorsToDelete;
      if (alreadyQueued) {
        state.accessorsToDelete = rest;
      } else {
        state.accessorsToDelete = {
          ...state.accessorsToDelete,
          [action.data.id]: action.data,
        };
      }
      break;
    }
    case DELETE_ACCESSOR_SUCCESS: {
      if (state.accessors) {
        state.accessors.data = state.accessors?.data.filter(
          (accessor: Accessor) => accessor.id !== action.data
        );
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        const { [action.data]: deletedAccessor, ...rest } =
          state.accessorsToDelete;
        state.accessorsToDelete = rest;
      }
      break;
    }
    case DELETE_ACCESSOR_ERROR:
      state.bulkUpdateAccessorsErrors = [
        ...state.bulkUpdateAccessorsErrors,
        action.data,
      ];
      break;
    case BULK_EDIT_ACCESSORS_REQUEST:
      state.updatingAccessors = true;
      state.bulkUpdateAccessorsSuccess = '';
      state.bulkUpdateAccessorsErrors = [];
      break;
    case BULK_EDIT_ACCESSORS_SUCCESS:
      state.updatingAccessors = false;
      // we can't get a count here, because we're removing accessors on each successful delete
      state.bulkUpdateAccessorsSuccess = 'Successfully deleted accessors';
      state.accessorsToDelete = {};
      state.accessorListEditMode = false;
      break;
    case BULK_EDIT_ACCESSORS_ERROR:
      state.updatingAccessors = false;
      break;
    case EXECUTE_ACCESSOR_CHANGE_CONTEXT:
      state.executeAccessorContext = action.data;
      break;
    case EXECUTE_ACCESSOR_CHANGE_SELECTOR_VALUES:
      state.executeAccessorSelectorValues = action.data;
      break;
    case EXECUTE_ACCESSOR_RESET:
      state.executeAccessorResponse = undefined;
      state.executeAccessorError = '';
      state.executeAccessorStats = undefined;
      break;
    case EXECUTE_ACCESSOR_SUCCESS: {
      const { resp, piiFields, sensitiveFields } = action.data;
      const parsedData = resp.data ? resp.data.map(JSON.parse) : undefined;
      state.executeAccessorResponse = resp;
      if (resp.data?.length && piiFields.length) {
        const { frequencies, uniqueness } = measureDataPrivacyStats(
          parsedData,
          piiFields as string[],
          sensitiveFields as string[]
        );
        state.executeAccessorStats = { frequencies, uniqueness };
      }
      break;
    }
    case EXECUTE_ACCESSOR_ERROR:
      state.executeAccessorError = action.data;
      break;

    case CHANGE_MUTATOR_SEARCH_FILTER:
      state.mutatorSearchFilter = {
        ...state.mutatorSearchFilter,
        ...setOperatorsForFilter(action.data),
      };
      break;
    case GET_TENANT_MUTATORS_SUCCESS:
      state.fetchingMutators = false;
      state.mutators = action.data;
      break;
    case GET_TENANT_MUTATORS_ERROR:
      state.fetchingMutators = false;
      state.fetchMutatorsError = action.data;
      break;
    case GET_MUTATOR_REQUEST:
      state.fetchingMutators = true;
      state.fetchMutatorsError = '';
      state.selectedMutator = undefined;
      state.modifiedMutator = undefined;
      state.selectedMutatorID = action.data;
      state.modifiedMutatorIsDirty = modifiedMutatorIsDirty(
        state.selectedMutator,
        state.modifiedMutator,
        state.mutatorColumnsToAdd,
        state.mutatorColumnsToDelete
      );
      state.userStoreColumnsToAdd = [];
      state.userStoreColumnsToDelete = {};
      state.saveMutatorColumnsSuccess = '';
      state.saveMutatorColumnsError = '';
      break;
    case GET_MUTATOR_SUCCESS: {
      state.fetchingMutators = false;
      state.selectedMutator = action.data;
      state.selectedMutatorID = (action.data as Mutator).id;
      state.modifiedMutator = { ...action.data };
      state.saveMutatorColumnsSuccess = '';
      state.saveMutatorColumnsError = '';
      state.modifiedMutatorIsDirty = modifiedMutatorIsDirty(
        state.selectedMutator,
        state.modifiedMutator,
        state.mutatorColumnsToAdd,
        state.mutatorColumnsToDelete
      );

      break;
    }
    case GET_MUTATOR_ERROR:
      state.fetchingMutators = false;
      state.fetchMutatorsError = action.data;
      break;
    case MODIFY_MUTATOR_DETAILS:
      state.modifiedMutator = {
        ...state.modifiedMutator,
        ...action.data,
      };
      state.modifiedMutatorIsDirty = modifiedMutatorIsDirty(
        state.selectedMutator,
        state.modifiedMutator,
        state.mutatorColumnsToAdd,
        state.mutatorColumnsToDelete
      );

      break;
    case TOGGLE_MUTATOR_EDIT_MODE: {
      state.mutatorDetailsEditMode = getNewToggleEditValue(
        action.data,
        state.mutatorDetailsEditMode
      );
      state.mutatorSelectorEditMode = state.mutatorDetailsEditMode;
      state.mutatorColumnsEditMode = state.mutatorDetailsEditMode;
      state.mutatorPoliciesEditMode = state.mutatorDetailsEditMode;
      if (state.selectedMutator && state.modifiedMutator) {
        state.modifiedMutator = { ...state.selectedMutator };
      }
      break;
    }
    case TOGGLE_MUTATOR_DETAILS_EDIT_MODE: {
      state.mutatorDetailsEditMode = getNewToggleEditValue(
        action.data,
        state.mutatorDetailsEditMode
      );
      state.savingMutatorSuccess = '';
      state.savingMutatorError = '';
      state.mutatorSelectorEditMode = false;
      if (state.selectedMutator && state.modifiedMutator) {
        state.modifiedMutator = { ...state.selectedMutator };
      }
      state.modifiedMutatorIsDirty = modifiedMutatorIsDirty(
        state.selectedMutator,
        state.modifiedMutator,
        state.mutatorColumnsToAdd,
        state.mutatorColumnsToDelete
      );

      break;
    }
    case MODIFY_MUTATOR_TO_CREATE:
      state.mutatorAddColumnDropdownValue = '';
      state.mutatorToCreate = {
        ...state.mutatorToCreate,
        ...action.data,
      };
      break;
    case TOGGLE_MUTATOR_COLUMNS_EDIT_MODE: {
      state.mutatorColumnsEditMode = action.data;
      state.mutatorColumnsToAdd = [];
      state.mutatorColumnsToDelete = {};
      state.mutatorAddColumnDropdownValue = '';
      state.saveMutatorColumnsSuccess = '';
      state.saveMutatorColumnsError = '';
      state.modifiedMutatorIsDirty = modifiedMutatorIsDirty(
        state.selectedMutator,
        state.modifiedMutator,
        state.mutatorColumnsToAdd,
        state.mutatorColumnsToDelete
      );

      break;
    }
    case TOGGLE_MUTATOR_SELECTOR_EDIT_MODE: {
      state.mutatorSelectorEditMode = getNewToggleEditValue(
        action.data,
        state.mutatorSelectorEditMode
      );
      state.mutatorDetailsEditMode = false;
      if (state.selectedMutator && state.modifiedMutator) {
        state.modifiedMutator = { ...state.selectedMutator };
      }
      state.modifiedMutatorIsDirty = modifiedMutatorIsDirty(
        state.selectedMutator,
        state.modifiedMutator,
        state.mutatorColumnsToAdd,
        state.mutatorColumnsToDelete
      );

      break;
    }
    case TOGGLE_MUTATOR_POLICIES_EDIT_MODE: {
      state.mutatorPoliciesEditMode = getNewToggleEditValue(
        action.data,
        state.mutatorPoliciesEditMode
      );
      state.selectedAccessPolicyForMutator = '';
      state.saveMutatorPoliciesSuccess = '';
      state.saveMutatorPoliciesError = '';
      if (state.selectedMutator && state.modifiedMutator) {
        state.modifiedMutator = { ...state.selectedMutator };
      }
      state.modifiedMutatorIsDirty = modifiedMutatorIsDirty(
        state.selectedMutator,
        state.modifiedMutator,
        state.mutatorColumnsToAdd,
        state.mutatorColumnsToDelete
      );

      break;
    }
    case LOAD_CREATE_MUTATOR_PAGE:
      state.mutatorToCreate = blankMutator();
      state.createMutatorError = '';
      break;
    case CREATE_MUTATOR_REQUEST:
    case UPDATE_MUTATOR_REQUEST:
      state.savingMutator = true;
      state.savingMutatorSuccess = '';
      state.savingMutatorError = '';
      state.modifiedMutatorIsDirty = modifiedMutatorIsDirty(
        state.selectedMutator,
        state.modifiedMutator,
        state.mutatorColumnsToAdd,
        state.mutatorColumnsToDelete
      );

      break;
    case UPDATE_MUTATOR_SUCCESS: {
      state.savingMutator = false;
      state.savingMutatorSuccess = 'Successfully saved mutator';

      state.selectedMutator = action.data;
      state.modifiedMutator = { ...action.data };
      state.mutatorDetailsEditMode = false;
      state.mutatorSelectorEditMode = false;
      state.modifiedMutatorIsDirty = modifiedMutatorIsDirty(
        state.selectedMutator,
        state.modifiedMutator,
        state.mutatorColumnsToAdd,
        state.mutatorColumnsToDelete
      );

      break;
    }
    case CREATE_MUTATOR_SUCCESS:
      state.savingMutator = false;
      state.mutatorToCreate = blankMutator();
      break;
    case CREATE_MUTATOR_ERROR:
      state.savingMutator = false;
      state.createMutatorError = action.data;
      break;
    case UPDATE_MUTATOR_ERROR:
      state.savingMutator = false;
      state.savingMutatorError = action.data;
      break;
    case TOGGLE_MUTATOR_COLUMN_FOR_DELETE: {
      const newlyAdded = state.mutatorColumnsToAdd.findIndex(
        (col: MutatorColumn) => col.id === action.data.id
      );
      if (newlyAdded > -1) {
        // removing an unsaved column
        state.mutatorColumnsToAdd.splice(newlyAdded, 1);
        state.mutatorColumnsToAdd = [...state.mutatorColumnsToAdd];
      } else {
        const { [action.data.id]: isAlreadyQueued, ...restOfDelete } =
          state.mutatorColumnsToDelete;
        if (isAlreadyQueued) {
          // unqueueing a queued column
          state.mutatorColumnsToDelete = restOfDelete;
        } else {
          // removing a persisted column
          state.mutatorColumnsToDelete = {
            ...state.mutatorColumnsToDelete,
            [action.data.id]: action.data,
          };
        }
      }
      state.modifiedMutatorIsDirty = modifiedMutatorIsDirty(
        state.selectedMutator,
        state.modifiedMutator,
        state.mutatorColumnsToAdd,
        state.mutatorColumnsToDelete
      );
      break;
    }
    case ADD_MUTATOR_COLUMN: {
      state.mutatorAddColumnDropdownValue = action.data;
      const column = state.userStoreColumns?.find(
        (col: Column) => col.id === action.data
      );
      if (column) {
        // we can naively push because the dropdown shouldn't display
        // columns that already exist on mutator or are queued
        state.mutatorColumnsToAdd = [
          ...state.mutatorColumnsToAdd,
          {
            id: column.id,
            name: column.name,
            table: column.table,
            data_type_id: column.data_type.id,
            data_type_name: column.data_type.name,
            is_array: column.is_array,
            normalizer_id: '',
            normalizer_name: '',
          },
        ];
        state.mutatorAddColumnDropdownValue = '';
        state.modifiedMutatorIsDirty = modifiedMutatorIsDirty(
          state.selectedMutator,
          state.modifiedMutator,
          state.mutatorColumnsToAdd,
          state.mutatorColumnsToDelete
        );
      }
      break;
    }
    case CHANGE_SELECTED_NORMALIZER_FOR_COLUMN: {
      const { columnID, normalizerID, isNew } = action.data;
      const matchingTransformer = state.transformers?.data?.find(
        (t: Transformer) => t.id === normalizerID
      );
      if (matchingTransformer) {
        if (isNew) {
          state.mutatorColumnsToAdd = state.mutatorColumnsToAdd?.map(
            (c: MutatorColumn) => {
              if (c.id === columnID) {
                return {
                  ...c,
                  normalizer_id: matchingTransformer.id,
                  normalizer_name: matchingTransformer.name,
                };
              }
              return c;
            }
          );
        } else if (state.modifiedMutator) {
          state.modifiedMutator = {
            ...state.modifiedMutator,
            columns: state.modifiedMutator.columns.map((c: MutatorColumn) => {
              if (c.id === columnID) {
                return {
                  ...c,
                  normalizer_id: matchingTransformer.id,
                  normalizer_name: matchingTransformer.name,
                };
              }
              return c;
            }),
          };
        }
        state.modifiedMutatorIsDirty = modifiedMutatorIsDirty(
          state.selectedMutator,
          state.modifiedMutator,
          state.mutatorColumnsToAdd,
          state.mutatorColumnsToDelete
        );
      }
      break;
    }
    case CHANGE_SELECTED_ACCESS_POLICY_FOR_MUTATOR:
      state.selectedAccessPolicyForMutator = action.data;
      state.modifiedMutatorIsDirty = modifiedMutatorIsDirty(
        state.selectedMutator,
        state.modifiedMutator,
        state.mutatorColumnsToAdd,
        state.mutatorColumnsToDelete
      );

      break;
    case TOGGLE_MUTATOR_LIST_EDIT_MODE: {
      state.mutatorListEditMode = getNewToggleEditValue(
        action.data,
        state.mutatorListEditMode
      );
      state.mutatorsToDelete = {};
      state.bulkUpdateMutatorsErrors = [];
      break;
    }
    case TOGGLE_MUTATOR_FOR_DELETE: {
      const { [action.data.id]: alreadyQueued, ...rest } =
        state.mutatorsToDelete;
      if (alreadyQueued) {
        state.mutatorsToDelete = rest;
      } else {
        state.mutatorsToDelete = {
          ...state.mutatorsToDelete,
          [action.data.id]: action.data,
        };
      }
      break;
    }
    case DELETE_MUTATOR_SUCCESS: {
      if (state.mutators) {
        state.mutators.data = state.mutators?.data.filter(
          (mutator: Mutator) => mutator.id !== action.data
        );
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        const { [action.data]: deletedMutator, ...rest } =
          state.mutatorsToDelete;
        state.mutatorsToDelete = rest;
      }
      break;
    }
    case DELETE_MUTATOR_ERROR:
      state.bulkUpdateMutatorsErrors = [
        ...state.bulkUpdateMutatorsErrors,
        action.data,
      ];
      break;
    case BULK_EDIT_MUTATORS_REQUEST:
      state.updatingMutators = true;
      state.bulkUpdateMutatorsSuccess = '';
      state.bulkUpdateMutatorsErrors = [];
      break;
    case BULK_EDIT_MUTATORS_SUCCESS:
      state.updatingMutators = false;
      // we can't get a count here, because we're removing mutators on each successful delete
      state.bulkUpdateMutatorsSuccess = 'Successfully deleted mutators';
      state.mutatorsToDelete = {};
      state.mutatorListEditMode = false;
      break;
    case BULK_EDIT_MUTATORS_ERROR:
      state.updatingMutators = false;
      break;

    case TOGGLE_ACCESSOR_LIST_EDIT_MODE: {
      state.accessorListEditMode = getNewToggleEditValue(
        action.data,
        state.accessorListEditMode
      );
      state.accessorsToDelete = {};
      state.bulkUpdateAccessorsErrors = [];
      break;
    }

    case CHANGE_PURPOSE_SEARCH_FILTER:
      state.purposeSearchFilter = {
        ...state.purposeSearchFilter,
        ...setOperatorsForFilter(action.data),
      };
      break;

    case GET_PURPOSES_REQUEST:
      state.fetchingPurposes = true;
      state.selectedPurpose = undefined;
      state.modifiedPurpose = undefined;
      state.purposesFetchError = '';
      state.createPurposeError = '';
      state.savePurposeError = '';
      state.deletePurposeError = '';
      break;
    case GET_PURPOSES_SUCCESS:
      state.fetchingPurposes = false;
      state.purposes = action.data;
      break;
    case GET_PURPOSES_ERROR:
      state.fetchingPurposes = false;
      state.purposesFetchError = action.data;
      break;
    case BULK_DELETE_PURPOSES_REQUEST:
      state.purposesBulkSaving = true;
      state.deletePurposesErrors = [];
      state.deletePurposesSuccess = '';
      break;
    case BULK_DELETE_PURPOSES_FAILURE:
      state.purposesBulkSaving = false;
      break;
    case BULK_DELETE_PURPOSES_SUCCESS:
      state.purposesBulkSaving = false;
      state.deletePurposesSuccess = 'Successfully saved.';
      state.purposesBulkEditMode = false;
      break;
    case DELETE_PURPOSES_SINGLE_SUCCESS:
      state.purposesDeleteQueue = state.purposesDeleteQueue.filter(
        (purposeID: string) => purposeID !== action.data
      );
      break;
    case DELETE_PURPOSES_SINGLE_ERROR:
      state.deletePurposesErrors = [...state.deletePurposesErrors, action.data];
      state.purposesBulkSaving = false;
      break;
    case CHANGE_SELECTED_PURPOSE:
      state.selectedPurpose = action.data;
      state.modifiedPurpose = action.data;
      state.createPurposeError = '';
      state.savePurposeError = '';
      state.deletePurposeError = '';
      break;
    case GET_PURPOSE_REQUEST:
      state.fetchingPurposes = true;
      state.selectedPurpose = undefined;
      state.modifiedPurpose = undefined;
      state.purposesFetchError = '';
      state.createPurposeError = '';
      state.savePurposeError = '';
      state.deletePurposeError = '';
      break;
    case GET_PURPOSE_SUCCESS:
      state.selectedPurpose = action.data;
      state.modifiedPurpose = action.data;
      state.fetchingPurposes = false;
      break;
    case GET_PURPOSE_ERROR:
      state.fetchingPurposes = false;
      state.purposesFetchError = action.data;
      break;
    case TOGGLE_PURPOSE_DETAILS_EDIT_MODE: {
      const editMode =
        action.data !== undefined ? action.data : !state.purposeDetailsEditMode;
      state.purposeDetailsEditMode = editMode;
      if (state.selectedPurpose) {
        state.modifiedPurpose = { ...state.selectedPurpose };
      }
      state.createPurposeError = '';
      state.savePurposeError = '';
      break;
    }
    case MODIFY_PURPOSE_DETAILS:
      state.modifiedPurpose = {
        ...state.modifiedPurpose,
        ...action.data,
      };
      break;
    case CREATE_PURPOSE_REQUEST:
      state.creatingPurpose = true;
      state.createPurposeError = '';
      state.savePurposeError = '';
      state.deletePurposeError = '';
      break;
    case CREATE_PURPOSE_SUCCESS:
      state.creatingPurpose = false;
      break;
    case CREATE_PURPOSE_ERROR:
      state.creatingPurpose = false;
      state.createPurposeError = action.data;
      break;
    case UPDATE_PURPOSE_REQUEST:
      state.savingPurpose = true;
      state.createPurposeError = '';
      state.savePurposeError = '';
      state.deletePurposeError = '';
      break;
    case UPDATE_PURPOSE_SUCCESS:
      state.savingPurpose = false;
      state.selectedPurpose = action.data;
      state.modifiedPurpose = action.data;
      state.purposeDetailsEditMode = false;
      break;
    case UPDATE_PURPOSE_ERROR:
      state.savingPurpose = false;
      state.savePurposeError = action.data;
      break;
    case DELETE_SINGLE_PURPOSE_REQUEST:
      state.deletingPurpose = true;
      state.createPurposeError = '';
      state.savePurposeError = '';
      state.deletePurposeError = '';
      break;
    case DELETE_SINGLE_PURPOSE_SUCCESS:
      state.deletingPurpose = false;
      break;
    case DELETE_SINGLE_PURPOSE_ERROR:
      state.deletingPurpose = false;
      state.deletePurposeError = action.data;
      break;
    case TOGGLE_PURPOSE_BULK_EDIT_MODE:
      state.purposesBulkEditMode = getNewToggleEditValue(
        action.data,
        state.purposesBulkEditMode
      );
      state.createPurposeError = '';
      state.savePurposeError = '';
      state.purposesDeleteQueue = [];
      state.deletePurposesSuccess = '';
      state.deletePurposesErrors = [];
      break;
    case TOGGLE_PURPOSE_FOR_DELETE: {
      const index = state.purposesDeleteQueue.indexOf(action.data);
      if (index > -1) {
        state.purposesDeleteQueue = state.purposesDeleteQueue.toSpliced(
          index,
          1
        );
      } else {
        state.purposesDeleteQueue = [...state.purposesDeleteQueue, action.data];
      }
      break;
    }

    case FETCH_USER_STORE_DATABASE_REQUEST:
      state.fetchingSqlshimDatabase = true;
      state.currentSqlshimDatabase = undefined;
      state.modifiedSqlshimDatabase = undefined;
      break;
    case FETCH_USER_STORE_DATABASES_SUCCESS:
      state.fetchingSqlshimDatabase = false;
      state.sqlShimDatabases = action.data;
      break;
    case FETCH_USER_STORE_DATABASE_ERROR:
      state.fetchingSqlshimDatabase = false;
      break;
    case MODIFY_USER_STORE_DATABASE: {
      const databaseUpdate = action.data;

      state.modifiedSqlshimDatabase = {
        ...state.modifiedSqlshimDatabase,
        ...databaseUpdate,
      };

      const dbID = state.modifiedSqlshimDatabase?.id;
      if (dbID) {
        const index = state.modifiedSqlShimDatabases.findIndex(
          (db) => db.id === dbID
        );

        if (index !== -1) {
          state.modifiedSqlShimDatabases[index] = {
            ...state.modifiedSqlShimDatabases[index],
            ...databaseUpdate,
          };
        }
      }

      state.databaseIsDirty =
        JSON.stringify(state.modifiedSqlshimDatabase) !==
        JSON.stringify(state.currentSqlshimDatabase);
      break;
    }
    case SAVE_USER_STORE_DATABASE_REQUEST:
      state.savingSqlshimDatabase = true;
      state.saveSqlshimDatabaseSuccess = '';
      state.saveSqlshimDatabaseError = '';
      break;
    case SAVE_USER_STORE_DATABASE_SUCCESS: {
      state.savingSqlshimDatabase = false;
      state.saveSqlshimDatabaseSuccess = `Successfully saved database`;
      break;
    }
    case SAVE_USER_STORE_DATABASE_ERROR:
      state.savingSqlshimDatabase = false;
      state.saveSqlshimDatabaseError = action.data;
      break;
    case DELETE_DATABASE:
      state.modifiedSqlShimDatabases = [
        ...state.modifiedSqlShimDatabases.filter((db) => db.id !== action.data),
      ];
      state.creatingDatabase = false;
      break;
    case CANCEL_DATABASE_DIALOG:
      if (state.creatingDatabase) {
        state.modifiedSqlShimDatabases = [
          ...state.modifiedSqlShimDatabases.filter(
            (db) => db.id !== action.data
          ),
        ];
        state.creatingDatabase = false;
      } else {
        const updatedDatabases = state.sqlShimDatabases?.data
          ? state.sqlShimDatabases?.data.map((db) => {
              if (db.id === action.data.id) {
                // Replace with the matching database from modifiedSqlShimDatabases
                return (
                  state.modifiedSqlShimDatabases.find(
                    (modifiedDb) => modifiedDb.id === db.id
                  ) || db
                );
              }
              return db;
            })
          : [];
        state.modifiedSqlShimDatabases = updatedDatabases;
      }
      state.testSqlshimDatabaseSuccess = false;
      state.testSqlshimDatabaseError = '';
      break;

    case RESET_DATABASE_DIALOG_STATE:
      state.testSqlshimDatabaseSuccess = false;
      state.testSqlshimDatabaseError = '';
      break;

    case SET_CURRENT_DATABASE:
      {
        state.tenantDatabaseDialogIsOpen = true;
        state.creatingDatabase = false;
        const currentDBIndex = state.modifiedSqlShimDatabases.findIndex(
          (db) => {
            return action.data === db.id;
          }
        );

        state.modifiedSqlshimDatabase =
          currentDBIndex < 0
            ? blankSqlShimDatabase()
            : state.modifiedSqlShimDatabases[currentDBIndex];

        if (currentDBIndex < 0) {
          state.modifiedSqlShimDatabases = [
            ...state.modifiedSqlShimDatabases,
            state.modifiedSqlshimDatabase,
          ];
        }
        state.currentSqlshimDatabase = state.modifiedSqlshimDatabase;
        state.saveSqlshimDatabaseError = '';
        state.saveSqlshimDatabaseSuccess = '';
        state.testSqlshimDatabaseSuccess = false;
        state.testSqlshimDatabaseError = '';
      }
      break;
    case ADD_DATABASE:
      state.tenantDatabaseDialogIsOpen = true;
      state.modifiedSqlshimDatabase = blankSqlShimDatabase();
      state.creatingDatabase = true;
      state.currentSqlshimDatabase = state.modifiedSqlshimDatabase;
      state.modifiedSqlShimDatabases = [
        ...state.modifiedSqlShimDatabases,
        state.modifiedSqlshimDatabase,
      ];
      state.databaseIsDirty = true;
      state.saveSqlshimDatabaseError = '';
      state.saveSqlshimDatabaseSuccess = '';
      break;
    case TEST_DATABASE_CONNECTION_REQUEST:
      state.testSqlshimDatabaseSuccess = false;
      state.testSqlshimDatabaseError = '';
      state.testingSqlshimDatabase = true;
      break;
    case TEST_DATABASE_CONNECTION_SUCCESS:
      state.testSqlshimDatabaseSuccess = true;
      state.testingSqlshimDatabase = false;
      break;
    case TEST_DATABASE_CONNECTION_ERROR:
      state.testSqlshimDatabaseError = action.data;
      state.testingSqlshimDatabase = false;
      break;
    case FETCH_USER_STORE_OBJECT_STORE_REQUEST:
      state.fetchingObjectStore = true;
      state.currentObjectStore = undefined;
      state.modifiedObjectStore = undefined;
      break;
    case FETCH_USER_STORE_OBJECT_STORE_SUCCESS:
      state.fetchingObjectStore = false;
      state.currentObjectStore = { ...action.data };
      state.modifiedObjectStore = { ...action.data };
      break;
    case FETCH_USER_STORE_OBJECT_STORE_ERROR:
      state.fetchingObjectStore = false;
      break;
    case FETCH_USER_STORE_OBJECT_STORES_REQUEST:
      state.fetchingObjectStore = true;
      state.objectStores = undefined;
      break;
    case FETCH_USER_STORE_OBJECT_STORES_SUCCESS:
      state.fetchingObjectStore = false;
      state.objectStores = { ...action.data };
      break;
    case FETCH_USER_STORE_OBJECT_STORES_ERROR:
      state.fetchingObjectStore = false;
      break;
    case TOGGLE_OBJECT_STORE_FOR_DELETE:
      if (state.objectStoreDeleteQueue.includes(action.data)) {
        state.objectStoreDeleteQueue = state.objectStoreDeleteQueue.filter(
          (id: string) => id !== action.data
        );
      } else {
        state.objectStoreDeleteQueue = [
          ...state.objectStoreDeleteQueue,
          action.data,
        ];
      }
      break;
    case TOGGLE_EDIT_USER_STORE_OBJECT_STORE_MODE:
      if (state.currentObjectStore) {
        state.editingObjectStore = getNewToggleEditValue(
          action.data,
          state.editingObjectStore
        );
      }
      break;
    case MODIFY_USER_STORE_OBJECT_STORE:
      state.modifiedObjectStore = {
        ...state.modifiedObjectStore,
        ...action.data,
      };
      break;
    case SAVE_USER_STORE_OBJECT_STORE_REQUEST:
      state.savingObjectStore = true;
      state.saveObjectStoreError = '';
      break;
    case SAVE_USER_STORE_OBJECT_STORE_SUCCESS: {
      state.savingObjectStore = false;
      state.editingObjectStore = false;
      state.currentObjectStore = { ...action.data };
      state.modifiedObjectStore = { ...action.data };
      break;
    }
    case SAVE_USER_STORE_OBJECT_STORE_ERROR:
      state.savingObjectStore = false;
      state.saveObjectStoreError = action.data;
      break;

    default:
      break;
  }

  return state;
};

export default userStoreReducer;
