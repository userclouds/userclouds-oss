import { APIError } from '@userclouds/sharedui';

import DataSource, { DataSourceElement } from '../models/DataSource';
import PaginatedResult from '../models/PaginatedResult';

export const TOGGLE_DATASOURCE_LIST_EDIT_MODE =
  'TOGGLE_DATASOURCE_LIST_EDIT_MODE';
export const TOGGLE_DATASOURCE_FOR_DELETE = 'TOGGLE_DATASOURCE_FOR_DELETE';

export const BULK_DELETE_DATASOURCES_REQUEST =
  'BULK_DELETE_DATASOURCES_REQUEST';
export const BULK_DELETE_DATASOURCES_SUCCESS =
  'BULK_DELETE_DATASOURCES_SUCCESS';
export const BULK_DELETE_DATASOURCES_ERROR = 'BULK_DELETE_DATASOURCES_ERROR';

export const DELETE_DATASOURCE_REQUEST = 'DELETE_DATASOURCE_REQUEST';
export const DELETE_DATASOURCE_SUCCESS = 'DELETE_DATASOURCE_SUCCESS';
export const DELETE_DATASOURCE_ERROR = 'DELETE_DATASOURCE_ERROR';

export const GET_TENANT_DATASOURCES_REQUEST = 'GET_TENANT_DATASOURCES_REQUEST';
export const GET_TENANT_DATASOURCES_SUCCESS = 'GET_TENANT_DATASOURCES_SUCCESS';
export const GET_TENANT_DATASOURCES_ERROR = 'GET_TENANT_DATASOURCES_ERROR';

export const GET_DATASOURCE_REQUEST = 'GET_DATASOURCE_REQUEST';
export const GET_DATASOURCE_SUCCESS = 'GET_DATASOURCE_SUCCESS';
export const GET_DATASOURCE_ERROR = 'GET_DATASOURCE_ERROR';

export const MODIFY_DATASOURCE_DETAILS = 'MODIFY_DATASOURCE_DETAILS';
export const TOGGLE_DATASOURCE_EDIT_MODE = 'TOGGLE_DATASOURCE_EDIT_MODE';

export const UPDATE_DATASOURCE_REQUEST = 'UPDATE_DATASOURCE_REQUEST';
export const UPDATE_DATASOURCE_SUCCESS = 'UPDATE_DATASOURCE_SUCCESS';
export const UPDATE_DATASOURCE_ERROR = 'UPDATE_DATASOURCE_ERROR';

export const LOAD_CREATE_DATASOURCE_PAGE = 'LOAD_CREATE_DATASOURCE_PAGE';

export const CREATE_DATASOURCE_REQUEST = 'CREATE_DATASOURCE_REQUEST';
export const CREATE_DATASOURCE_SUCCESS = 'CREATE_DATASOURCE_SUCCESS';
export const CREATE_DATASOURCE_ERROR = 'CREATE_DATASOURCE_ERROR';

export const CHANGE_DATASOURCES_SEARCH_FILTER =
  'CHANGE_DATASOURCES_SEARCH_FILTER';

export const toggleDataSourceListEditMode = (editMode?: boolean) => ({
  type: TOGGLE_DATASOURCE_LIST_EDIT_MODE,
  data: editMode,
});

export const toggleDataSourceForDelete = (dataSource: DataSource) => ({
  type: TOGGLE_DATASOURCE_FOR_DELETE,
  data: dataSource.id,
});

export const bulkDeleteDataSourcesRequest = () => ({
  type: BULK_DELETE_DATASOURCES_REQUEST,
});

export const bulkDeleteDataSourcesSuccess = () => ({
  type: BULK_DELETE_DATASOURCES_SUCCESS,
});

export const bulkDeleteDataSourcesError = () => ({
  type: BULK_DELETE_DATASOURCES_ERROR,
});

export const deleteDataSourceRequest = () => ({
  type: DELETE_DATASOURCE_REQUEST,
});

export const deleteDataSourceSuccess = (dataSourceID: string) => ({
  type: DELETE_DATASOURCE_SUCCESS,
  data: dataSourceID,
});

export const deleteDataSourceError = (error: APIError) => ({
  type: DELETE_DATASOURCE_ERROR,
  data: error.message,
});

export const getTenantDataSourcesRequest = () => ({
  type: GET_TENANT_DATASOURCES_REQUEST,
});

export const getTenantDataSourcesSuccess = (
  dataSources: PaginatedResult<DataSource>
) => ({
  type: GET_TENANT_DATASOURCES_SUCCESS,
  data: dataSources,
});

export const getTenantDataSourcesError = (error: APIError) => ({
  type: GET_TENANT_DATASOURCES_ERROR,
  data: error.message,
});

export const getDataSourceRequest = (dataSourceID: string) => ({
  type: GET_DATASOURCE_REQUEST,
  data: dataSourceID,
});

export const getDataSourceSuccess = (dataSource: DataSource) => ({
  type: GET_DATASOURCE_SUCCESS,
  data: dataSource,
});

export const getDataSourceError = (error: APIError) => ({
  type: GET_DATASOURCE_ERROR,
  data: error.message,
});

export const toggleDataSourceEditMode = (editMode?: boolean) => ({
  type: TOGGLE_DATASOURCE_EDIT_MODE,
  data: editMode,
});

export const modifyDataSourceDetails = (data: Record<string, any>) => ({
  type: MODIFY_DATASOURCE_DETAILS,
  data,
});

export const updateDataSourceRequest = () => ({
  type: UPDATE_DATASOURCE_REQUEST,
});

export const updateDataSourceSuccess = (dataSource: DataSource) => ({
  type: UPDATE_DATASOURCE_SUCCESS,
  data: dataSource,
});

export const updateDataSourceError = (error: APIError) => ({
  type: UPDATE_DATASOURCE_ERROR,
  data: error.message,
});

export const loadCreateDataSourcePage = () => ({
  type: LOAD_CREATE_DATASOURCE_PAGE,
});

export const createDataSourceRequest = () => ({
  type: CREATE_DATASOURCE_REQUEST,
});

export const createDataSourceSuccess = (dataSource: DataSource) => ({
  type: CREATE_DATASOURCE_SUCCESS,
  data: dataSource,
});

export const createDataSourceError = (error: APIError) => ({
  type: CREATE_DATASOURCE_ERROR,
  data: error.message,
});

export const changeDataSourcesSearchFilter = (
  changes: Record<string, string>
) => ({
  type: CHANGE_DATASOURCES_SEARCH_FILTER,
  data: changes,
});

///

export const TOGGLE_DATASOURCEELEMENT_LIST_EDIT_MODE =
  'TOGGLE_DATASOURCEELEMENT_LIST_EDIT_MODE';
export const GET_TENANT_DATASOURCEELEMENTS_REQUEST =
  'GET_TENANT_DATASOURCEELEMENTS_REQUEST';
export const GET_TENANT_DATASOURCEELEMENTS_SUCCESS =
  'GET_TENANT_DATASOURCEELEMENTS_SUCCESS';
export const GET_TENANT_DATASOURCEELEMENTS_ERROR =
  'GET_TENANT_DATASOURCEELEMENTS_ERROR';
export const GET_DATASOURCEELEMENT_REQUEST = 'GET_DATASOURCEELEMENT_REQUEST';
export const GET_DATASOURCEELEMENT_SUCCESS = 'GET_DATASOURCEELEMENT_SUCCESS';
export const GET_DATASOURCEELEMENT_ERROR = 'GET_DATASOURCEELEMENT_ERROR';
export const MODIFY_DATASOURCEELEMENT_DETAILS =
  'MODIFY_DATASOURCEELEMENT_DETAILS';
export const TOGGLE_DATASOURCEELEMENT_EDIT_MODE =
  'TOGGLE_DATASOURCEELEMENT_EDIT_MODE';
export const UPDATE_DATASOURCEELEMENT_REQUEST =
  'UPDATE_DATASOURCEELEMENT_REQUEST';
export const UPDATE_DATASOURCEELEMENT_SUCCESS =
  'UPDATE_DATASOURCEELEMENT_SUCCESS';
export const UPDATE_DATASOURCEELEMENT_ERROR = 'UPDATE_DATASOURCEELEMENT_ERROR';
export const LOAD_CREATE_DATASOURCEELEMENT_PAGE =
  'LOAD_CREATE_DATASOURCEELEMENT_PAGE';
export const CREATE_DATASOURCEELEMENT_REQUEST =
  'CREATE_DATASOURCEELEMENT_REQUEST';
export const CREATE_DATASOURCEELEMENT_SUCCESS =
  'CREATE_DATASOURCEELEMENT_SUCCESS';
export const CREATE_DATASOURCEELEMENT_ERROR = 'CREATE_DATASOURCEELEMENT_ERROR';
export const CHANGE_DATASOURCEELEMENTS_SEARCH_FILTER =
  'CHANGE_DATASOURCEELEMENTS_SEARCH_FILTER';

export const toggleDataSourceElementListEditMode = (editMode?: boolean) => ({
  type: TOGGLE_DATASOURCEELEMENT_LIST_EDIT_MODE,
  data: editMode,
});

export const getTenantDataSourceElementsRequest = () => ({
  type: GET_TENANT_DATASOURCEELEMENTS_REQUEST,
});

export const getTenantDataSourceElementsSuccess = (
  elements: PaginatedResult<DataSourceElement>
) => ({
  type: GET_TENANT_DATASOURCEELEMENTS_SUCCESS,
  data: elements,
});

export const getTenantDataSourceElementsError = (error: APIError) => ({
  type: GET_TENANT_DATASOURCEELEMENTS_ERROR,
  data: error.message,
});

export const getDataSourceElementRequest = (elementID: string) => ({
  type: GET_DATASOURCEELEMENT_REQUEST,
  data: elementID,
});

export const getDataSourceElementSuccess = (element: DataSourceElement) => ({
  type: GET_DATASOURCEELEMENT_SUCCESS,
  data: element,
});

export const getDataSourceElementError = (error: APIError) => ({
  type: GET_DATASOURCEELEMENT_ERROR,
  data: error.message,
});

export const toggleDataSourceElementEditMode = (editMode?: boolean) => ({
  type: TOGGLE_DATASOURCEELEMENT_EDIT_MODE,
  data: editMode,
});

export const modifyDataSourceElementDetails = (data: Record<string, any>) => ({
  type: MODIFY_DATASOURCEELEMENT_DETAILS,
  data,
});

export const updateDataSourceElementRequest = () => ({
  type: UPDATE_DATASOURCEELEMENT_REQUEST,
});

export const updateDataSourceElementSuccess = (element: DataSourceElement) => ({
  type: UPDATE_DATASOURCEELEMENT_SUCCESS,
  data: element,
});

export const updateDataSourceElementError = (error: APIError) => ({
  type: UPDATE_DATASOURCEELEMENT_ERROR,
  data: error.message,
});

export const loadCreateDataSourceElementPage = () => ({
  type: LOAD_CREATE_DATASOURCEELEMENT_PAGE,
});

export const createDataSourceElementRequest = () => ({
  type: CREATE_DATASOURCEELEMENT_REQUEST,
});

export const createDataSourceElementSuccess = (element: DataSourceElement) => ({
  type: CREATE_DATASOURCEELEMENT_SUCCESS,
  data: element,
});

export const createDataSourceElementError = (error: APIError) => ({
  type: CREATE_DATASOURCEELEMENT_ERROR,
  data: error.message,
});

export const changeDataSourceElementsSearchFilter = (
  changes: Record<string, string>
) => ({
  type: CHANGE_DATASOURCEELEMENTS_SEARCH_FILTER,
  data: changes,
});
