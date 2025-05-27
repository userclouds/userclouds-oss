import { AnyAction } from 'redux';

import { RootState } from '../store';
import { getNewToggleEditValue } from './reducerHelper';
import { setOperatorsForFilter } from '../controls/SearchHelper';
import {
  TOGGLE_DATASOURCE_LIST_EDIT_MODE,
  TOGGLE_DATASOURCE_FOR_DELETE,
  BULK_DELETE_DATASOURCES_REQUEST,
  BULK_DELETE_DATASOURCES_SUCCESS,
  BULK_DELETE_DATASOURCES_ERROR,
  DELETE_DATASOURCE_REQUEST,
  DELETE_DATASOURCE_SUCCESS,
  DELETE_DATASOURCE_ERROR,
  GET_TENANT_DATASOURCES_REQUEST,
  GET_TENANT_DATASOURCES_SUCCESS,
  GET_TENANT_DATASOURCES_ERROR,
  GET_DATASOURCE_REQUEST,
  GET_DATASOURCE_SUCCESS,
  GET_DATASOURCE_ERROR,
  MODIFY_DATASOURCE_DETAILS,
  TOGGLE_DATASOURCE_EDIT_MODE,
  UPDATE_DATASOURCE_REQUEST,
  UPDATE_DATASOURCE_SUCCESS,
  UPDATE_DATASOURCE_ERROR,
  LOAD_CREATE_DATASOURCE_PAGE,
  CREATE_DATASOURCE_REQUEST,
  CREATE_DATASOURCE_SUCCESS,
  CREATE_DATASOURCE_ERROR,
  GET_TENANT_DATASOURCEELEMENTS_REQUEST,
  GET_TENANT_DATASOURCEELEMENTS_SUCCESS,
  GET_TENANT_DATASOURCEELEMENTS_ERROR,
  GET_DATASOURCEELEMENT_REQUEST,
  GET_DATASOURCEELEMENT_SUCCESS,
  GET_DATASOURCEELEMENT_ERROR,
  TOGGLE_DATASOURCEELEMENT_EDIT_MODE,
  MODIFY_DATASOURCEELEMENT_DETAILS,
  CHANGE_DATASOURCES_SEARCH_FILTER,
  CHANGE_DATASOURCEELEMENTS_SEARCH_FILTER,
  UPDATE_DATASOURCEELEMENT_REQUEST,
  UPDATE_DATASOURCEELEMENT_SUCCESS,
  UPDATE_DATASOURCEELEMENT_ERROR,
} from '../actions/datamapping';
import DataSource from '../models/DataSource';

const datamappingReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case GET_TENANT_DATASOURCES_REQUEST:
      state.fetchingDataSources = true;
      state.fetchDataSourceError = '';
      state.dataSources = undefined;
      state.deleteDataSourceSuccess = '';
      state.deleteDataSourceError = '';
      state.bulkDeleteDataSourcesSuccess = '';
      state.bulkDeleteDataSourcesErrors = [];
      state.dataSourceSaveSuccess = '';
      state.dataSourceSaveError = '';
      break;
    case GET_TENANT_DATASOURCES_SUCCESS:
      state.fetchingDataSources = false;
      state.dataSources = action.data;
      break;
    case GET_TENANT_DATASOURCES_ERROR:
      state.fetchingDataSources = false;
      state.fetchDataSourceError = action.data;
      break;
    case TOGGLE_DATASOURCE_LIST_EDIT_MODE:
      state.deleteDataSourceSuccess = '';
      state.deleteDataSourceError = '';
      state.dataSourceSaveSuccess = '';
      state.dataSourceSaveError = '';
      break;
    case TOGGLE_DATASOURCE_FOR_DELETE:
      if (
        state.dataSourcesDeleteQueue &&
        state.dataSourcesDeleteQueue.indexOf(action.data) > -1
      ) {
        state.dataSourcesDeleteQueue = state.dataSourcesDeleteQueue.filter(
          (id) => action.data !== id
        );
      } else {
        state.dataSourcesDeleteQueue = [
          ...state.dataSourcesDeleteQueue,
          action.data,
        ];
      }
      break;
    case BULK_DELETE_DATASOURCES_REQUEST:
      state.bulkDeleteDataSourcesSuccess = '';
      state.bulkDeleteDataSourcesErrors = [];
      state.deleteDataSourceSuccess = '';
      state.deleteDataSourceError = '';
      state.deletingDataSources = true;
      break;
    case BULK_DELETE_DATASOURCES_SUCCESS:
      state.dataSourcesDeleteQueue = [];
      state.bulkDeleteDataSourcesSuccess =
        'Successfully deleted selected data sources';
      state.deletingDataSources = false;
      break;
    case BULK_DELETE_DATASOURCES_ERROR:
      state.bulkDeleteDataSourcesErrors = [
        ...state.bulkDeleteDataSourcesErrors,
        action.data,
      ];
      state.deletingDataSources = false;
      break;
    case DELETE_DATASOURCE_REQUEST:
      state.deletingDataSources = true;
      state.deleteDataSourceSuccess = '';
      state.deleteDataSourceError = '';
      state.bulkDeleteDataSourcesErrors = [];
      break;
    case DELETE_DATASOURCE_SUCCESS: {
      if (state.dataSources) {
        state.dataSources.data = state.dataSources?.data.filter(
          (ds: DataSource) => ds.id !== action.data
        );
        state.dataSourcesDeleteQueue = state.dataSourcesDeleteQueue.filter(
          (dsID: string) => dsID !== action.data
        );
      }
      if (!state.dataSourcesDeleteQueue.length) {
        state.deletingDataSources = false;
      }
      state.deleteDataSourceSuccess = `Successfully deleted data source with ID ${action.data}`;
      break;
    }
    case DELETE_DATASOURCE_ERROR:
      state.bulkDeleteDataSourcesErrors = [
        ...state.bulkDeleteDataSourcesErrors,
        action.data,
      ];
      state.deleteDataSourceError = action.data;
      if (!state.dataSourcesDeleteQueue.length) {
        state.deletingDataSources = false;
      }
      break;

    case GET_DATASOURCE_REQUEST:
      state.fetchingDataSources = true;
      state.fetchDataSourceError = '';
      state.selectedDataSource = undefined;
      state.dataSourceSaveSuccess = '';
      state.dataSourceSaveError = '';
      state.deleteDataSourceSuccess = '';
      state.deleteDataSourceError = '';
      break;
    case GET_DATASOURCE_SUCCESS:
      state.fetchingDataSources = false;
      state.selectedDataSource = { ...action.data };
      state.modifiedDataSource = { ...action.data };
      break;
    case GET_DATASOURCE_ERROR:
      state.fetchingDataSources = false;
      state.fetchDataSourceError = action.data;
      break;
    case TOGGLE_DATASOURCE_EDIT_MODE:
      state.dataSourceSaveSuccess = '';
      state.dataSourceSaveError = '';
      state.deleteDataSourceError = '';
      state.dataSourceDetailsEditMode = getNewToggleEditValue(
        action.data,
        state.dataSourceDetailsEditMode
      );
      break;
    case MODIFY_DATASOURCE_DETAILS: {
      const newModifiedDataSource = {
        ...state.modifiedDataSource,
        ...action.data,
      };
      newModifiedDataSource.config = {
        ...state.modifiedDataSource.config,
        ...action.data.config,
      };
      newModifiedDataSource.metadata = {
        ...state.modifiedDataSource.metadata,
        ...action.data.metadata,
      };
      state.modifiedDataSource = newModifiedDataSource;
      break;
    }
    case CHANGE_DATASOURCES_SEARCH_FILTER:
      state.dataSourcesSearchFilter = {
        ...state.dataSourcesSearchFilter,
        ...setOperatorsForFilter(action.data),
      };
      break;

    case UPDATE_DATASOURCE_REQUEST:
      state.savingDataSource = true;
      state.dataSourceSaveSuccess = '';
      state.dataSourceSaveError = '';
      break;
    case UPDATE_DATASOURCE_SUCCESS:
      state.savingDataSource = false;
      state.selectedDataSource = { ...state.modifiedDataSource };
      state.dataSourceDetailsEditMode = false;
      state.dataSourceSaveSuccess = 'Successfully updated data source';
      break;
    case UPDATE_DATASOURCE_ERROR:
      state.savingDataSource = false;
      state.dataSourceSaveError = 'Error updating data source';
      break;
    case LOAD_CREATE_DATASOURCE_PAGE:
      state.dataSourceSaveSuccess = '';
      state.dataSourceSaveError = '';
      break;
    case CREATE_DATASOURCE_REQUEST:
      state.savingDataSource = true;
      state.dataSourceSaveSuccess = '';
      state.dataSourceSaveError = '';
      break;
    case CREATE_DATASOURCE_SUCCESS:
      state.savingDataSource = false;
      state.dataSourceSaveSuccess = 'Successfully created data sourcce';
      break;
    case CREATE_DATASOURCE_ERROR:
      state.savingDataSource = false;
      state.dataSourceSaveError = `Error creating data source: ${action.data}`;
      break;

    case GET_TENANT_DATASOURCEELEMENTS_REQUEST:
      state.fetchingDataSourceElements = true;
      state.fetchDataSourceElementError = '';
      state.dataSourceElementSaveSuccess = '';
      state.dataSourceElementSaveError = '';
      state.dataSourceElements = undefined;
      break;
    case GET_TENANT_DATASOURCEELEMENTS_SUCCESS:
      state.fetchingDataSourceElements = false;
      state.dataSourceElements = action.data;
      break;
    case GET_TENANT_DATASOURCEELEMENTS_ERROR:
      state.fetchingDataSourceElements = false;
      state.fetchDataSourceElementError = action.data;
      break;
    case GET_DATASOURCEELEMENT_REQUEST:
      state.fetchingDataSourceElements = true;
      state.fetchDataSourceElementError = '';
      state.dataSourceElementSaveSuccess = '';
      state.dataSourceElementSaveError = '';
      state.selectedDataSourceElement = undefined;
      break;
    case GET_DATASOURCEELEMENT_SUCCESS:
      state.fetchingDataSourceElements = false;
      state.selectedDataSourceElement = action.data;
      state.modifiedDataSourceElement = { ...action.data };
      break;
    case GET_DATASOURCEELEMENT_ERROR:
      state.fetchingDataSourceElements = false;
      state.fetchDataSourceElementError = action.data;
      break;
    case TOGGLE_DATASOURCEELEMENT_EDIT_MODE:
      state.dataSourceElementSaveSuccess = '';
      state.dataSourceElementSaveError = '';
      state.dataSourceElementDetailsEditMode = getNewToggleEditValue(
        action.data,
        state.dataSourceElementDetailsEditMode
      );
      break;
    case MODIFY_DATASOURCEELEMENT_DETAILS: {
      const newModifiedDataSourceElement = {
        ...state.modifiedDataSourceElement,
        ...action.data,
      };
      newModifiedDataSourceElement.metadata = {
        ...state.modifiedDataSourceElement.metadata,
        ...action.data.metadata,
      };
      state.modifiedDataSourceElement = newModifiedDataSourceElement;
      break;
    }
    case CHANGE_DATASOURCEELEMENTS_SEARCH_FILTER:
      state.dataSourceElementsSearchFilter = {
        ...state.dataSourceElementsSearchFilter,
        ...setOperatorsForFilter(action.data),
      };
      break;
    case UPDATE_DATASOURCEELEMENT_REQUEST:
      state.savingDataSourceElement = true;
      state.dataSourceElementSaveSuccess = '';
      state.dataSourceElementSaveError = '';
      break;
    case UPDATE_DATASOURCEELEMENT_SUCCESS:
      state.savingDataSourceElement = false;
      state.selectedDataSourceElement = { ...state.modifiedDataSourceElement };
      state.dataSourceElementDetailsEditMode = false;
      state.dataSourceElementSaveSuccess =
        'Successfully updated data source element';
      break;
    case UPDATE_DATASOURCEELEMENT_ERROR:
      state.savingDataSourceElement = false;
      state.dataSourceElementSaveError = `Error saving data source element: ${action.data}`;
      break;

    default:
      break;
  }

  return state;
};

export default datamappingReducer;
