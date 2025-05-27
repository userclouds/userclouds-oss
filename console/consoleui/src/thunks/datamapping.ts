import { APIError } from '@userclouds/sharedui';

import { AppDispatch } from '../store';
import {
  fetchTenantDataSources,
  fetchTenantDataSource,
  fetchTenantDataSourceElement,
  deleteTenantDataSource,
  fetchTenantDataSourceElements,
} from '../API/datamapping';
import {
  getTenantDataSourcesRequest,
  getTenantDataSourcesSuccess,
  getTenantDataSourcesError,
  getDataSourceRequest,
  getDataSourceSuccess,
  getDataSourceError,
  getDataSourceElementRequest,
  getDataSourceElementSuccess,
  getDataSourceElementError,
  deleteDataSourceRequest,
  deleteDataSourceSuccess,
  deleteDataSourceError,
  getTenantDataSourceElementsRequest,
  getTenantDataSourceElementsSuccess,
  getTenantDataSourceElementsError,
} from '../actions/datamapping';
import PaginatedResult from '../models/PaginatedResult';
import DataSource, {
  DataSourceElement,
  dataSourcesPrefix,
  dataSourceElementsPrefix,
} from '../models/DataSource';
import { getParamsAsObject } from '../controls/PaginationHelper';
import { postSuccessToast } from './notifications';
import { redirect } from '../routing';

export const fetchDataSources =
  (tenantID: string, query: URLSearchParams) => (dispatch: AppDispatch) => {
    dispatch(getTenantDataSourcesRequest());
    return fetchTenantDataSources(
      tenantID,
      getParamsAsObject(dataSourcesPrefix, query)
    ).then(
      (data: PaginatedResult<DataSource>) => {
        dispatch(getTenantDataSourcesSuccess(data));
      },
      (e: APIError) => {
        dispatch(getTenantDataSourcesError(e));
      }
    );
  };

export const fetchDataSource =
  (tenantID: string, dataSourceID: string) => (dispatch: AppDispatch) => {
    dispatch(getDataSourceRequest(dataSourceID));
    fetchTenantDataSource(tenantID, dataSourceID).then(
      (data: DataSource) => {
        dispatch(getDataSourceSuccess(data));
      },
      (e: APIError) => {
        dispatch(getDataSourceError(e));
      }
    );
  };

export const deleteDataSource =
  (companyID: string, tenantID: string, dataSourceID: string) =>
  (dispatch: AppDispatch) => {
    dispatch(deleteDataSourceRequest());
    deleteTenantDataSource(tenantID, dataSourceID).then(
      () => {
        dispatch(deleteDataSourceSuccess(dataSourceID));
        dispatch(postSuccessToast('Successfully deleted data source'));
        redirect(`/datasources?company_id=${companyID}&tenant_id=${tenantID}`);
      },
      (error: APIError) => {
        dispatch(deleteDataSourceError(error));
      }
    );
  };

export const fetchDataSourceElement =
  (tenantID: string, elementID: string) => (dispatch: AppDispatch) => {
    dispatch(getDataSourceElementRequest(elementID));
    return fetchTenantDataSourceElement(tenantID, elementID).then(
      (data: DataSourceElement) => {
        dispatch(getDataSourceElementSuccess(data));
      },
      (e: APIError) => {
        dispatch(getDataSourceElementError(e));
      }
    );
  };

export const fetchDataSourceElements =
  (tenantID: string, query: URLSearchParams) => (dispatch: AppDispatch) => {
    dispatch(getTenantDataSourceElementsRequest());
    fetchTenantDataSourceElements(
      tenantID,
      getParamsAsObject(dataSourceElementsPrefix, query)
    ).then(
      (data: PaginatedResult<DataSourceElement>) => {
        dispatch(getTenantDataSourceElementsSuccess(data));
      },
      (e: APIError) => {
        dispatch(getTenantDataSourceElementsError(e));
      }
    );
  };
