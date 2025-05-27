import { AnyAction } from 'redux';

import { APIError } from '@userclouds/sharedui';

import { DEFAULT_PAGE_LIMIT } from '../controls/PaginationHelper';
import { AppDispatch, RootState } from '../store';
import { redirect } from '../routing';
import PaginatedResult from '../models/PaginatedResult';
import { Column } from '../models/TenantUserStoreConfig';
import Mutator from '../models/Mutator';
import Accessor from '../models/Accessor';
import { SqlshimDatabase } from '../models/SqlshimDatabase';
import { ObjectStore } from '../models/ObjectStore';
import {
  ColumnRetentionDurationsResponse,
  PurposeRetentionDuration,
  DurationType,
} from '../models/ColumnRetentionDurations';
import {
  createUserStoreColumn,
  fetchTenantUserStoreColumns,
  fetchUserStoreColumn,
  updateUserStoreColumn,
  deleteUserStoreColumn,
  fetchUserStoreColumnRetentionDurations,
  updateUserStoreColumnRetentionDurations,
  fetchTenantUserStoreDataTypes,
  fetchUserStoreDataType,
  createUserStoreDataType,
  updateUserStoreDataType,
  deleteUserStoreDataType,
} from '../API/userstore';
import { deleteTenantAccessor } from '../API/accessors';
import { deleteTenantMutator, fetchTenantMutators } from '../API/mutators';
import {
  createTenantDatabase,
  deleteTenantDatabase,
  getTenantDatabases,
  updateTenantDatabase,
  testTenantDatabase,
} from '../API/sqlshimdatabase';
import {
  createTenantObjectStore,
  updateTenantObjectStore,
  getTenantObjectStore,
  listTenantObjectStores,
  deleteTenantObjectStore,
} from '../API/objectstore';
import {
  getUserStoreConfigRequest,
  getUserStoreConfigSuccess,
  getUserStoreConfigError,
  updateUserStoreConfigRequest,
  updateUserStoreConfigSuccess,
  updateUserStoreConfigError,
  deleteUserStoreColumnRequest,
  deleteBulkUserStoreColumnSuccess,
  deleteBulkUserStoreColumnError,
  updateBulkUserStoreColumnSuccess,
  updateBulkUserStoreColumnError,
  updateUserStoreColumnRequest,
  createUserStoreColumnRequest,
  createBulkUserStoreColumnSuccess,
  createBulkUserStoreColumnError,
  deleteUserStoreColumnError,
  deleteUserStoreColumnSuccess,
  updateUserStoreColumnSuccess,
  updateUserStoreColumnError,
  createUserStoreColumnSuccess,
  createUserStoreColumnError,
  fetchUserStoreColumnRequest,
  fetchUserStoreColumnSuccess,
  fetchUserStoreColumnError,
  fetchUserStoreDisplayColumnsSuccess,
  fetchUserStoreDisplayColumnsRequest,
  fetchUserStoreDisplayColumnsError,
  getColumnDurationsRequest,
  getColumnDurationsSuccess,
  getColumnDurationsError,
  updateColumnDurationsRequest,
  updateColumnDurationsSuccess,
  updateColumnDurationsError,
  fetchDataTypeRequest,
  fetchDataTypesRequest,
  fetchDataTypesSuccess,
  createDataTypeError,
  createDataTypeRequest,
  createDataTypeSuccess,
  deleteDataTypeError,
  deleteDataTypeRequest,
  deleteDataTypeSuccess,
  fetchDataTypeError,
  fetchDataTypesError,
  fetchDataTypeSuccess,
  updateDataTypeError,
  updateDataTypeRequest,
  updateDataTypeSuccess,
  bulkDeleteDataTypesRequest,
  bulkDeleteDataTypesSuccess,
  bulkDeleteDataTypesFailure,
  saveUserStoreDatabaseRequest,
  saveUserStoreDatabaseSuccess,
  saveUserStoreDatabaseError,
  fetchUserStoreDatabaseRequest,
  fetchUserStoreDatabasesSuccess,
  fetchUserStoreDatabaseError,
  saveUserStoreObjectStoreRequest,
  saveUserStoreObjectStoreSuccess,
  saveUserStoreObjectStoreError,
  fetchUserStoreObjectStoreRequest,
  fetchUserStoreObjectStoreSuccess,
  fetchUserStoreObjectStoreError,
  fetchUserStoreObjectStoresRequest,
  fetchUserStoreObjectStoresSuccess,
  fetchUserStoreObjectStoresError,
  testDatabaseConnectionRequest,
  testDatabaseConnectionSuccess,
  testDatabaseConnectionError,
} from '../actions/userstore';
import {
  bulkEditMutatorsError,
  bulkEditMutatorsRequest,
  bulkEditMutatorsSuccess,
  deleteMutatorError,
  deleteMutatorSuccess,
  getTenantMutatorsError,
  getTenantMutatorsRequest,
  getTenantMutatorsSuccess,
} from '../actions/mutators';
import {
  bulkEditAccessorsError,
  bulkEditAccessorsRequest,
  bulkEditAccessorsSuccess,
  deleteAccessorError,
  deleteAccessorSuccess,
} from '../actions/accessors';
import { PAGINATION_API_VERSION } from '../API';
import { postAlertToast, postSuccessToast } from './notifications';
import { fetchAccessors } from './accessors';
import { DataType } from '../models/DataType';
import AccessPolicy from '../models/AccessPolicy';

const COLUMNS_PAGE_SIZE = '50';
const OBJECT_STORE_PAGE_SIZE = '50';

export const fetchColumn =
  (tenantID: string, columnID: string) =>
  (dispatch: AppDispatch): Promise<void> => {
    return new Promise((resolve, reject) => {
      dispatch(fetchUserStoreColumnRequest());
      return fetchUserStoreColumn(tenantID, columnID).then(
        (response: Column) => {
          dispatch(fetchUserStoreColumnSuccess(response));
          resolve();
        },
        (error: APIError) => {
          dispatch(fetchUserStoreColumnError(error));
          reject(error);
        }
      );
    });
  };

export const createColumn =
  (tenantID: string, companyID: string | undefined, column: Column) =>
  (dispatch: AppDispatch, getState: () => RootState): Promise<void> => {
    const { modifiedAccessPolicy, modifiedTokenAccessPolicy } = getState();

    return new Promise((resolve, reject) => {
      dispatch(createUserStoreColumnRequest());
      return createUserStoreColumn(
        tenantID,
        column,
        modifiedAccessPolicy,
        modifiedTokenAccessPolicy
      ).then(
        (response: Column) => {
          dispatch(createUserStoreColumnSuccess(response));
          if (companyID) {
            redirect(
              `/columns/${response.id}?company_id=${
                companyID as string
              }&tenant_id=${tenantID}`
            );
          }
          resolve();
        },
        (error: APIError) => {
          dispatch(createUserStoreColumnError(error));
          reject(error);
        }
      );
    });
  };

export const updateColumn =
  (tenantID: string, column: Column) =>
  (dispatch: AppDispatch, getState: () => RootState): Promise<void> => {
    const { modifiedAccessPolicy, modifiedTokenAccessPolicy } = getState();

    return new Promise((resolve, reject) => {
      dispatch(updateUserStoreColumnRequest());
      return updateUserStoreColumn(
        tenantID,
        column,
        modifiedAccessPolicy,
        modifiedTokenAccessPolicy
      ).then(
        (response: Column) => {
          dispatch(updateUserStoreColumnSuccess(response));
          resolve();
        },
        (error: APIError) => {
          dispatch(updateUserStoreColumnError(error));
          reject(error);
        }
      );
    });
  };

export const deleteColumn =
  (tenantID: string, columnID: string) =>
  (dispatch: AppDispatch): Promise<void> => {
    return new Promise((resolve, reject) => {
      dispatch(deleteUserStoreColumnRequest());
      return deleteUserStoreColumn(tenantID, columnID).then(
        () => {
          dispatch(postSuccessToast('Successfully deleted column'));
          dispatch(deleteUserStoreColumnSuccess());
          dispatch(
            fetchUserStoreDisplayColumns(tenantID, new URLSearchParams())
          );
          resolve();
        },
        (error: APIError) => {
          dispatch(postAlertToast('Unable to delete column: ' + error));
          dispatch(deleteUserStoreColumnError(error));
          reject(error);
        }
      );
    });
  };

export const fetchUserStoreDisplayColumns =
  (tenantID: string, params: URLSearchParams) => (dispatch: AppDispatch) => {
    const paramsAsObject = [
      'columns_starting_after',
      'columns_ending_before',
      'columns_limit',
      'columns_filter',
      'columns_sort_key',
      'columns_sort_order',
    ].reduce((acc: Record<string, string>, paramName: string) => {
      if (params.has(paramName)) {
        acc[paramName.substring(8)] = params.get(paramName) as string;
      }
      return acc;
    }, {});
    if (!paramsAsObject.limit) {
      paramsAsObject.limit = COLUMNS_PAGE_SIZE;
    }
    if (!paramsAsObject.sort_order) {
      paramsAsObject.sort_order = 'ascending';
    }
    if (!paramsAsObject.sort_key) {
      paramsAsObject.sort_key = 'name,id';
    }
    if (!paramsAsObject.version) {
      paramsAsObject.version = PAGINATION_API_VERSION;
    }

    dispatch(fetchUserStoreDisplayColumnsRequest());
    return fetchTenantUserStoreColumns(tenantID, paramsAsObject).then(
      (result: PaginatedResult<Column>) => {
        dispatch(fetchUserStoreDisplayColumnsSuccess(result));
      },
      (error: APIError) => {
        dispatch(fetchUserStoreDisplayColumnsError(error));
      }
    );
  };

export const fetchUserStoreConfig =
  (tenantID: string) => (dispatch: AppDispatch) => {
    dispatch(getUserStoreConfigRequest());
    return fetchTenantUserStoreColumns(tenantID, {}).then(
      (result: PaginatedResult<Column>) => {
        dispatch(getUserStoreConfigSuccess({ columns: result.data }));
      },
      (error: APIError) => {
        dispatch(getUserStoreConfigError(error));
      }
    );
  };

export const createColumnBulk =
  (tenantID: string, column: Column) =>
  (dispatch: AppDispatch): Promise<void> => {
    return new Promise((resolve, reject) => {
      return createUserStoreColumn(tenantID, column).then(
        (response: Column) => {
          dispatch(createBulkUserStoreColumnSuccess(response));
          resolve();
        },
        (error: APIError) => {
          dispatch(createBulkUserStoreColumnError(error));
          reject(error);
        }
      );
    });
  };

export const updateColumnBulk =
  (tenantID: string, column: Column) =>
  (dispatch: AppDispatch): Promise<void> => {
    return new Promise((resolve, reject) => {
      return updateUserStoreColumn(tenantID, column).then(
        (response: Column) => {
          dispatch(updateBulkUserStoreColumnSuccess(response));
          resolve();
        },
        (error: APIError) => {
          dispatch(updateBulkUserStoreColumnError(error));
          reject(error);
        }
      );
    });
  };

export const deleteColumnBulk =
  (tenantID: string, columnID: string) =>
  (dispatch: AppDispatch): Promise<void> => {
    return new Promise((resolve, reject) => {
      return deleteUserStoreColumn(tenantID, columnID).then(
        () => {
          dispatch(deleteBulkUserStoreColumnSuccess());
          resolve();
        },
        (error: APIError) => {
          dispatch(deleteBulkUserStoreColumnError(error));
          reject(error);
        }
      );
    });
  };

export const fetchMutators =
  (tenantID: string, params: URLSearchParams) => (dispatch: AppDispatch) => {
    const paramsAsObject = [
      'mutators_starting_after',
      'mutators_ending_before',
      'mutators_limit',
      'mutators_filter',
      'mutators_sort_key',
      'mutators_sort_order',
    ].reduce((acc: Record<string, string>, paramName: string) => {
      if (params.has(paramName)) {
        acc[paramName.substring(9)] = params.get(paramName) as string;
      }
      return acc;
    }, {});
    if (!paramsAsObject.limit) {
      paramsAsObject.limit = String(DEFAULT_PAGE_LIMIT);
    }
    if (!paramsAsObject.sort_order) {
      paramsAsObject.sort_order = 'ascending';
    }
    if (!paramsAsObject.sort_key) {
      paramsAsObject.sort_key = 'name,id';
    }
    if (!paramsAsObject.version) {
      paramsAsObject.version = PAGINATION_API_VERSION;
    }
    dispatch(getTenantMutatorsRequest());
    return fetchTenantMutators(tenantID, paramsAsObject).then(
      (mutators: PaginatedResult<Mutator>) => {
        dispatch(getTenantMutatorsSuccess(mutators));
      },
      (error: APIError) => {
        dispatch(getTenantMutatorsError(error));
      }
    );
  };

export const fetchRetentionDurationsForColumn =
  (tenantID: string, columnID: string, durationType: DurationType) =>
  (dispatch: AppDispatch): Promise<ColumnRetentionDurationsResponse> => {
    dispatch(getColumnDurationsRequest());
    return new Promise(() => {
      return fetchUserStoreColumnRetentionDurations(
        tenantID,
        columnID,
        durationType
      ).then(
        (durations: ColumnRetentionDurationsResponse) => {
          dispatch(getColumnDurationsSuccess(durations));
        },
        (error: APIError) => {
          dispatch(getColumnDurationsError(error));
        }
      );
    });
  };

export const updateRetentionDurationsForColumn =
  (
    tenantID: string,
    columnID: string,
    durationType: DurationType,
    durations: PurposeRetentionDuration[]
  ) =>
  (dispatch: AppDispatch): Promise<ColumnRetentionDurationsResponse> => {
    dispatch(updateColumnDurationsRequest());
    return new Promise(() => {
      return updateUserStoreColumnRetentionDurations(
        tenantID,
        columnID,
        durationType,
        durations
      ).then(
        (updatedDurations: ColumnRetentionDurationsResponse) => {
          dispatch(updateColumnDurationsSuccess(updatedDurations));
        },
        (error: APIError) => {
          dispatch(updateColumnDurationsError(error));
        }
      );
    });
  };

export const saveUserStore =
  () => (dispatch: AppDispatch, getState: () => RootState) => {
    const {
      selectedTenantID,
      userStoreColumnsToAdd,
      userStoreColumnsToModify,
      userStoreColumnsToDelete,
    } = getState();

    if (!selectedTenantID) {
      return;
    }

    dispatch(updateUserStoreConfigRequest());
    const reqs: Promise<any>[] = [];
    userStoreColumnsToAdd.forEach((col) => {
      reqs.push(dispatch(createColumnBulk(selectedTenantID, col)));
    });
    for (const id in userStoreColumnsToModify) {
      reqs.push(
        dispatch(
          updateColumnBulk(selectedTenantID, userStoreColumnsToModify[id])
        )
      );
    }
    for (const id in userStoreColumnsToDelete) {
      reqs.push(dispatch(deleteColumnBulk(selectedTenantID, id)));
    }
    Promise.all(reqs).then(
      () => {
        dispatch(fetchUserStoreConfig(selectedTenantID));
        dispatch(
          fetchUserStoreDisplayColumns(selectedTenantID, new URLSearchParams())
        );
        dispatch(updateUserStoreConfigSuccess());
      },
      () => {
        dispatch(updateUserStoreConfigError());
      }
    );
  };

export const bulkDeleteMutatorsOrAccessors =
  (
    tenantID: string,
    deleteQueue: Record<string, Mutator> | Record<string, Accessor>,
    type: 'accessor' | 'mutator'
  ) =>
  (dispatch: AppDispatch, getState: () => RootState) => {
    const { query, accessorListIncludeAutogenerated } = getState();
    let bulkStart: () => AnyAction;
    let bulkSuccess: () => AnyAction;
    let bulkError: () => AnyAction;
    let singleSuccess: (id: string) => AnyAction;
    let singleError: (error: APIError) => AnyAction;
    let apiMethod: any;
    const keys = Object.keys(deleteQueue);
    if (!keys.length) {
      return;
    }
    if (type === 'mutator') {
      bulkStart = bulkEditMutatorsRequest;
      bulkSuccess = bulkEditMutatorsSuccess;
      bulkError = bulkEditMutatorsError;
      singleSuccess = deleteMutatorSuccess;
      singleError = deleteMutatorError;
      apiMethod = deleteTenantMutator;
    } else {
      bulkStart = bulkEditAccessorsRequest;
      bulkSuccess = bulkEditAccessorsSuccess;
      bulkError = bulkEditAccessorsError;
      singleSuccess = deleteAccessorSuccess;
      singleError = deleteAccessorError;
      apiMethod = deleteTenantAccessor;
    }
    dispatch(bulkStart());
    return Promise.all(
      Object.keys(deleteQueue).map((id: string) =>
        apiMethod(tenantID, id).then(
          () => {
            return dispatch(singleSuccess(id));
          },
          (error: APIError) => {
            dispatch(singleError(error));
            throw error;
          }
        )
      )
    ).then(
      () => {
        dispatch(bulkSuccess());
        // TODO: ideally the include_autogenerated bit should be in the querystring
        if (type === 'mutator') {
          dispatch(fetchMutators(tenantID, query));
        } else {
          dispatch(
            fetchAccessors(tenantID, query, accessorListIncludeAutogenerated)
          );
        }
      },
      () => {
        dispatch(bulkError());
      }
    );
  };

export const deleteSingleMutator =
  (tenantID: string, id: string) => (dispatch: AppDispatch) => {
    return new Promise(() => {
      return deleteTenantMutator(tenantID, id).then(
        () => {
          dispatch(fetchMutators(tenantID, new URLSearchParams()));
          dispatch(deleteMutatorSuccess(id));
          dispatch(postSuccessToast('Successfully deleted mutator'));
        },
        (error: APIError) => {
          dispatch(deleteMutatorError(error));
          dispatch(postAlertToast('Error deleting mutator: ' + error));
        }
      );
    });
  };

export const deleteSingleAccessor =
  (tenantID: string, id: string) =>
  (dispatch: AppDispatch, getState: () => RootState) => {
    const { accessorListIncludeAutogenerated } = getState();
    return new Promise(() => {
      return deleteTenantAccessor(tenantID, id).then(
        () => {
          dispatch(
            fetchAccessors(
              tenantID,
              new URLSearchParams(),
              accessorListIncludeAutogenerated
            )
          );
          dispatch(deleteAccessorSuccess(id));
          dispatch(postSuccessToast('Successfully deleted accessor'));
        },
        (error: APIError) => {
          dispatch(deleteAccessorError(error));
          dispatch(postAlertToast('Error deleting accessor: ' + error));
        }
      );
    });
  };

export const fetchDataType =
  (tenantID: string, dataTypeID: string) =>
  (dispatch: AppDispatch): Promise<void> => {
    return new Promise((resolve, reject) => {
      dispatch(fetchDataTypeRequest());
      return fetchUserStoreDataType(tenantID, dataTypeID).then(
        (response: DataType) => {
          dispatch(fetchDataTypeSuccess(response));
          resolve();
        },
        (error: APIError) => {
          dispatch(fetchDataTypeError(error));
          reject(error);
        }
      );
    });
  };

export const createDataType =
  (tenantID: string, companyID: string | undefined, dataType: DataType) =>
  (dispatch: AppDispatch): Promise<void> => {
    return new Promise((resolve, reject) => {
      dispatch(createDataTypeRequest());
      return createUserStoreDataType(tenantID, dataType).then(
        (response: DataType) => {
          dispatch(createDataTypeSuccess(response));
          dispatch(postSuccessToast('Successfully created data type'));
          redirect(
            `/datatypes/${response.id}?company_id=${
              companyID as string
            }&tenant_id=${tenantID}`
          );
          resolve();
        },
        (error: APIError) => {
          dispatch(createDataTypeError(error));
          reject(error);
        }
      );
    });
  };

export const updateDataType =
  (tenantID: string, dataType: DataType) =>
  (dispatch: AppDispatch): Promise<void> => {
    return new Promise((resolve, reject) => {
      dispatch(updateDataTypeRequest());
      return updateUserStoreDataType(tenantID, dataType).then(
        (response: DataType) => {
          dispatch(updateDataTypeSuccess(response));
          resolve();
        },
        (error: APIError) => {
          dispatch(updateDataTypeError(error));
          reject(error);
        }
      );
    });
  };

export const deleteDataType =
  (tenantID: string, dataTypeID: string) =>
  (dispatch: AppDispatch): Promise<void> => {
    return new Promise((resolve, reject) => {
      dispatch(deleteDataTypeRequest());
      return deleteUserStoreDataType(tenantID, dataTypeID).then(
        () => {
          dispatch(postSuccessToast('Successfully deleted dataType'));
          dispatch(deleteDataTypeSuccess());
          dispatch(fetchDataTypes(tenantID, new URLSearchParams()));
          resolve();
        },
        (error: APIError) => {
          dispatch(postAlertToast('Unable to delete dataType: ' + error));
          dispatch(deleteDataTypeError(error));
          reject(error);
        }
      );
    });
  };

export const deleteDataTypeBulk =
  (tenantID: string, dataTypeID: string) =>
  (dispatch: AppDispatch, getState: () => RootState): Promise<boolean> => {
    const { dataTypes } = getState();
    const matchingDataType = dataTypes?.data.find(
      (dataType: DataType) => dataType.id === dataTypeID
    );
    return new Promise((resolve, reject) => {
      if (!matchingDataType) {
        return reject();
      }
      return deleteUserStoreDataType(tenantID, matchingDataType.id).then(
        () => {
          dispatch(deleteDataTypeSuccess());
          resolve(true);
        },
        (error: APIError) => {
          dispatch(deleteDataTypeError(error));
          resolve(false);
        }
      );
    });
  };

export const bulkDeleteDataTypes =
  (selectedTenantID: string, dataTypeIDs: string[]) =>
  (dispatch: AppDispatch, getState: () => RootState) => {
    const { query } = getState();

    dispatch(bulkDeleteDataTypesRequest());
    const reqs: Array<Promise<boolean>> = [];
    dataTypeIDs.forEach((id) => {
      reqs.push(dispatch(deleteDataTypeBulk(selectedTenantID, id)));
    });
    Promise.all(reqs).then((values: boolean[]) => {
      if (values.every((val) => val === true)) {
        dispatch(fetchDataTypes(selectedTenantID, query));
        dispatch(bulkDeleteDataTypesSuccess());
      } else {
        if (!values.every((val) => val === false)) {
          fetchDataTypes(selectedTenantID, query);
        }
        dispatch(bulkDeleteDataTypesFailure());
      }
    });
  };

export const fetchDataTypes =
  (tenantID: string, params: URLSearchParams) => (dispatch: AppDispatch) => {
    const paramsAsObject = [
      'data_types_starting_after',
      'data_types_ending_before',
      'data_types_limit',
      'data_types_filter',
      'data_types_sort_order',
    ].reduce((acc: Record<string, string>, paramName: string) => {
      if (params.has(paramName)) {
        acc[paramName.substring(11)] = params.get(paramName) as string;
      }
      return acc;
    }, {});
    if (!paramsAsObject.limit) {
      paramsAsObject.limit = COLUMNS_PAGE_SIZE;
    }
    if (!paramsAsObject.version) {
      paramsAsObject.version = PAGINATION_API_VERSION;
    }
    if (!paramsAsObject.sort_order) {
      paramsAsObject.sort_order = 'ascending';
    }
    if (!paramsAsObject.sort_key) {
      paramsAsObject.sort_key = 'name,id';
    }

    dispatch(fetchDataTypesRequest());
    return fetchTenantUserStoreDataTypes(tenantID, paramsAsObject).then(
      (result: PaginatedResult<DataType>) => {
        dispatch(fetchDataTypesSuccess(result));
      },
      (error: APIError) => {
        dispatch(fetchDataTypesError(error));
      }
    );
  };

export const handleCreateDataType =
  () => (dispatch: AppDispatch, getState: () => RootState) => {
    const { selectedCompanyID, selectedTenantID, dataTypeToCreate } =
      getState();

    if (selectedTenantID && selectedCompanyID) {
      dispatch(
        createDataType(selectedTenantID, selectedCompanyID, dataTypeToCreate)
      );
    }
  };

export const updateTenantSqlShim =
  (tenantID: string, database: SqlshimDatabase, onSuccess?: Function) =>
  async (dispatch: AppDispatch) => {
    dispatch(saveUserStoreDatabaseRequest());
    updateTenantDatabase(tenantID, database).then(
      () => {
        dispatch(saveUserStoreDatabaseSuccess());
        if (onSuccess) {
          onSuccess();
        }
      },
      (error: APIError) => {
        dispatch(saveUserStoreDatabaseError(error));
      }
    );
  };

export const getTenantSqlShims =
  (tenantID: string) => async (dispatch: AppDispatch) => {
    dispatch(fetchUserStoreDatabaseRequest());

    getTenantDatabases(tenantID).then(
      (databases: PaginatedResult<SqlshimDatabase>) => {
        dispatch(fetchUserStoreDatabasesSuccess(databases));
      },
      (error: APIError) => {
        dispatch(fetchUserStoreDatabaseError(error));
      }
    );
  };

export const getBatchUpdateDatabasePromises = (
  tenantID: string,
  currentDatabases: SqlshimDatabase[],
  modifiedDatabases: SqlshimDatabase[]
): Promise<any>[] => {
  const promises: Promise<any>[] = [];

  const currentDatabaseMap = new Map(currentDatabases.map((db) => [db.id, db]));

  modifiedDatabases?.length &&
    modifiedDatabases.forEach((modifiedDb) => {
      const currentDb = currentDatabaseMap.get(modifiedDb.id);
      if (!currentDb) {
        promises.push(createTenantDatabase(tenantID, modifiedDb));
      } else if (JSON.stringify(modifiedDb) !== JSON.stringify(currentDb)) {
        promises.push(updateTenantDatabase(tenantID, modifiedDb));
      }
      currentDatabaseMap.delete(modifiedDb.id);
    });

  currentDatabaseMap?.forEach((db) => {
    promises.push(deleteTenantDatabase(tenantID, db.id));
  });

  return promises;
};

export const testDatabaseConnection =
  (tenantID: string, database: SqlshimDatabase) =>
  async (dispatch: AppDispatch) => {
    dispatch(testDatabaseConnectionRequest());
    testTenantDatabase(tenantID, database).then(
      () => {
        dispatch(testDatabaseConnectionSuccess());
      },
      (error: APIError) => {
        dispatch(testDatabaseConnectionError(error));
      }
    );
  };

export const createObjectStore =
  (
    companyID: string,
    tenantID: string,
    objectStore: ObjectStore,
    modifiedAccessPolicy: AccessPolicy | undefined
  ) =>
  async (dispatch: AppDispatch) => {
    dispatch(saveUserStoreObjectStoreRequest());
    createTenantObjectStore(tenantID, objectStore, modifiedAccessPolicy).then(
      (newObjectStore: ObjectStore) => {
        redirect(
          `/object_stores/${newObjectStore.id}?company_id=${
            companyID as string
          }&tenant_id=${tenantID}`
        );
      },
      (error: APIError) => {
        dispatch(saveUserStoreObjectStoreError(error));
      }
    );
  };

export const updateObjectStore =
  (
    tenantID: string,
    objectStore: ObjectStore,
    modifiedAccessPolicy: AccessPolicy | undefined
  ) =>
  async (dispatch: AppDispatch) => {
    dispatch(saveUserStoreObjectStoreRequest());
    updateTenantObjectStore(tenantID, objectStore, modifiedAccessPolicy).then(
      (updatedStore: ObjectStore) => {
        dispatch(saveUserStoreObjectStoreSuccess(updatedStore));
        dispatch(
          postSuccessToast('Successfully updated object store connection')
        );
      },
      (error: APIError) => {
        dispatch(saveUserStoreObjectStoreError(error));
      }
    );
  };

export const getObjectStore =
  (tenantID: string, objectStoreID: string) =>
  async (dispatch: AppDispatch) => {
    dispatch(fetchUserStoreObjectStoreRequest());
    getTenantObjectStore(tenantID, objectStoreID).then(
      (objectStore: ObjectStore) => {
        dispatch(fetchUserStoreObjectStoreSuccess(objectStore));
      },
      (error: APIError) => {
        dispatch(fetchUserStoreObjectStoreError(error));
      }
    );
  };

export const fetchObjectStores =
  (tenantID: string, params: URLSearchParams) => (dispatch: AppDispatch) => {
    const paramsAsObject = [
      'objectstore_starting_after',
      'objectstore_ending_before',
      'objectstore_limit',
      'objectstore_filter',
    ].reduce((acc: Record<string, string>, paramName: string) => {
      if (params.has(paramName)) {
        acc[paramName.substring(12)] = params.get(paramName) as string;
      }
      return acc;
    }, {});
    if (!paramsAsObject.limit) {
      paramsAsObject.limit = OBJECT_STORE_PAGE_SIZE;
    }
    if (!paramsAsObject.sort_order) {
      paramsAsObject.sort_order = 'ascending';
    }
    if (!paramsAsObject.sort_key) {
      paramsAsObject.sort_key = 'name,id';
    }
    if (!paramsAsObject.version) {
      paramsAsObject.version = PAGINATION_API_VERSION;
    }
    dispatch(fetchUserStoreObjectStoresRequest());
    return listTenantObjectStores(tenantID, paramsAsObject).then(
      (objectStores: PaginatedResult<ObjectStore>) => {
        dispatch(fetchUserStoreObjectStoresSuccess(objectStores));
      },
      (error: APIError) => {
        dispatch(fetchUserStoreObjectStoresError(error));
      }
    );
  };

export const deleteObjectStore =
  (tenantID: string, objectStoreID: string) =>
  async (dispatch: AppDispatch) => {
    deleteTenantObjectStore(tenantID, objectStoreID).then(
      () => {
        dispatch(postSuccessToast('Successfully deleted object store'));
        dispatch(fetchObjectStores(tenantID, new URLSearchParams()));
      },
      (error: APIError) => {
        dispatch(postAlertToast('Unable to delete object store: ' + error));
      }
    );
  };

export const bulkDeleteObjectStores =
  (tenantID: string, objectStoreIDs: string[]) =>
  async (dispatch: AppDispatch, getState: () => RootState) => {
    const { query } = getState();
    return Promise.all(
      objectStoreIDs.map((id: string) => deleteTenantObjectStore(tenantID, id))
    ).then(
      () => {
        dispatch(postSuccessToast('Successfully deleted object stores'));
        dispatch(fetchObjectStores(tenantID, query));
      },
      (error: APIError) => {
        dispatch(fetchObjectStores(tenantID, query));
        dispatch(postAlertToast('Unable to delete object stores: ' + error));
      }
    );
  };
