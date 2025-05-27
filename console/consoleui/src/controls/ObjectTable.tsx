import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  CardRow,
  Checkbox,
  EmptyState,
  IconFilter,
  IconLock2,
  InlineNotification,
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableRowHead,
  TableCell,
  Text,
  ToolTip,
  TextShortener,
} from '@userclouds/ui-component-lib';

import { AppDispatch, RootState } from '../store';
import PaginatedResult from '../models/PaginatedResult';
import UCObject, { objectColumns } from '../models/authz/Object';
import { ObjectType } from '../models/authz/ObjectType';
import { SelectedTenant } from '../models/Tenant';
import { getParamsAsObject } from './PaginationHelper';
import { Filter } from '../models/authz/SearchFilters';
import { makeCleanPageLink } from '../AppNavigation';
import { deleteTenantAuthZObject } from '../API/authzAPI';
import { postAlertToast, postSuccessToast } from '../thunks/notifications';
import { fetchAuthZObjects } from '../thunks/authz';
import {
  BULK_UPDATE_OBJECTS_END,
  BULK_UPDATE_OBJECTS_START,
  changeCurrentObjectSearchFilter,
  DELETE_OBJECT_ERROR,
  DELETE_OBJECT_SUCCESS,
  toggleSelectAllObjects,
  TOGGLE_OBJECT_FOR_DELETE,
} from '../actions/authz';

import DeleteWithConfirmationButton from './DeleteWithConfirmationButton';
import Pagination from './Pagination';
import Search from './Search';
import Link from './Link';
import PageCommon from '../pages/PageCommon.module.css';
import styles from './ObjectTable.module.css';

const PAGINATION_LIMIT = '50';
const prefix = 'objects_';

const fetchObjects =
  (
    selectedTenantID: string | undefined,
    params: URLSearchParams,
    objectTypeID?: string
  ) =>
  (dispatch: AppDispatch) => {
    const paramsAsObject = getParamsAsObject(prefix, params);
    // if objects_limit is not specified in querystring,
    // use the default
    if (!paramsAsObject.limit) {
      paramsAsObject.limit = PAGINATION_LIMIT;
    }
    if (selectedTenantID) {
      dispatch(
        fetchAuthZObjects(selectedTenantID, paramsAsObject, objectTypeID)
      );
    }
  };

const deleteObject =
  (tenantId: string, id: string, objectTypeID: string | undefined) =>
  (dispatch: AppDispatch) => {
    return deleteTenantAuthZObject(tenantId, id).then(
      () => {
        dispatch(fetchObjects(tenantId, new URLSearchParams(), objectTypeID));
        dispatch({
          type: DELETE_OBJECT_SUCCESS,
          data: id,
        });
        dispatch(postSuccessToast('Successfully deleted object'));
      },
      (error: Error) => {
        dispatch({
          type: DELETE_OBJECT_ERROR,
          data: id,
        });
        dispatch(postAlertToast('Error deleting object: ' + error));
      }
    );
  };

const onSaveObjects =
  (objectTypeID?: string) =>
  async (dispatch: AppDispatch, getState: () => RootState) => {
    const { selectedTenantID, objectDeleteQueue } = getState();
    if (!objectDeleteQueue || !selectedTenantID) {
      return;
    }
    dispatch({
      type: BULK_UPDATE_OBJECTS_START,
    });
    let promises: Array<Promise<void>> = [];
    if (objectDeleteQueue.length) {
      promises = objectDeleteQueue.map((id) =>
        dispatch(deleteObject(selectedTenantID, id, objectTypeID))
      );
      Promise.all(promises as Array<Promise<void>>).then(
        () => {
          dispatch({
            type: BULK_UPDATE_OBJECTS_END,
            data: true, // success
          });
          dispatch(
            fetchObjects(selectedTenantID, new URLSearchParams(), objectTypeID)
          );
        },
        () => {
          dispatch({
            type: BULK_UPDATE_OBJECTS_END,
            data: false, // complete or partial failure
          });
          dispatch(
            fetchObjects(selectedTenantID, new URLSearchParams(), objectTypeID)
          );
        }
      );
    }
  };

const changeObjectSearchFilter =
  (changes: Record<string, string>) => async (dispatch: AppDispatch) => {
    // TODO v2 Add Operator Column and set the operator with that
    dispatch(changeCurrentObjectSearchFilter(changes));
  };

type ObjectsProps = {
  objects: PaginatedResult<UCObject> | undefined;
  selectedTenant: SelectedTenant | undefined;
  objectTypes: PaginatedResult<ObjectType> | undefined;
  objectTypeError: string;
  objectError: string;
  isLoading: boolean;
  objectDeleteQueue: string[];
  editSuccess: string;
  editError: string;
  lastFetchedObjectsTypeID: string | undefined;
  objectSearchFilter: Filter;
  objectTypeID?: string | undefined;
  createButton?: boolean;
  query: URLSearchParams;
  detailLayout?: boolean;
  dispatch: AppDispatch;
};

const Objects = ({
  objects,
  selectedTenant,
  objectTypes,
  objectTypeError,
  objectError,
  isLoading,
  editSuccess,
  editError,
  lastFetchedObjectsTypeID,
  objectDeleteQueue,
  objectSearchFilter,
  objectTypeID,
  createButton,
  query,
  detailLayout = false,
  dispatch,
}: ObjectsProps) => {
  useEffect(() => {
    if (selectedTenant && objectTypeID !== lastFetchedObjectsTypeID) {
      dispatch(fetchObjects(selectedTenant.id, query, objectTypeID));
    }
  }, [selectedTenant, lastFetchedObjectsTypeID, objectTypeID, query, dispatch]);

  const deletePrompt = `Are you sure you want to delete ${objectDeleteQueue.length} object${
    objectDeleteQueue.length > 1 ? 's' : ''
  }? This action is irreversible.`;

  return (
    <>
      {!detailLayout && (
        <div className={PageCommon.listviewtablecontrols}>
          <div>
            <IconFilter />
          </div>
          <Search
            columns={objectColumns}
            changeSearchFilter={(filter: Filter) => {
              dispatch(changeObjectSearchFilter(filter));
            }}
            prefix={prefix}
            searchFilter={objectSearchFilter}
            id="objects"
          />

          <ToolTip>
            <>
              {
                'Objects are the nodes of your graph. They represent real world concepts like users, groups and assets. '
              }
              <a
                href="https://docs.userclouds.com/docs/key-concepts-1#objects"
                title="UserClouds documentation for key concepts in authorization"
                target="new"
                className={PageCommon.link}
              >
                Learn more here.
              </a>
            </>
          </ToolTip>
          {selectedTenant?.is_admin && (
            <ButtonGroup
              className={PageCommon.listviewtablecontrolsButtonGroup}
            >
              <Button theme="primary" size="small">
                <Link
                  href={'/objects/create' + makeCleanPageLink(query)}
                  applyStyles={false}
                >
                  Create Object
                </Link>
              </Button>
            </ButtonGroup>
          )}
        </div>
      )}

      {editSuccess && (
        <InlineNotification theme="success">{editSuccess}</InlineNotification>
      )}
      {editError && (
        <InlineNotification theme="alert">{editError}</InlineNotification>
      )}

      <>
        <div className={PageCommon.listviewpaginationcontrols}>
          <div className={PageCommon.listviewpaginationcontrolsdelete}>
            <DeleteWithConfirmationButton
              id="deleteObjectsButton"
              message={deletePrompt}
              onConfirmDelete={() => {
                dispatch(onSaveObjects());
              }}
              title="Delete Objects"
              disabled={objectDeleteQueue.length < 1}
            />
          </div>

          <Pagination
            prev={objects?.prev}
            next={objects?.next}
            prefix={prefix}
            isLoading={false}
          />
        </div>
      </>

      {objects ? (
        <>
          {objects.data && objects.data.length ? (
            <>
              <Table
                id="objectsTable"
                spacing="packed"
                className={styles.objectstable}
              >
                <TableHead floating>
                  <TableRow>
                    <TableRowHead>
                      <Checkbox
                        checked={
                          objectDeleteQueue.length === objects.data.length
                        }
                        onChange={() => {
                          dispatch(toggleSelectAllObjects());
                        }}
                      />
                    </TableRowHead>

                    <TableRowHead>ID</TableRowHead>
                    <TableRowHead>Name</TableRowHead>
                    <TableRowHead>Type</TableRowHead>
                    <TableRowHead key="delete_header" />
                  </TableRow>
                </TableHead>
                <TableBody>
                  {objectTypes?.data?.length &&
                    objects?.data?.map((o: UCObject) => {
                      const matchingType = objectTypes.data.find(
                        (ot) => ot.id === o.type_id
                      );
                      return (
                        <TableRow
                          key={o.id}
                          className={
                            (objectDeleteQueue.includes(o.id)
                              ? PageCommon.queuedfordelete
                              : '') +
                            ' ' +
                            PageCommon.listviewtablerow
                          }
                        >
                          <TableCell>
                            <Checkbox
                              id={'delete' + o.id}
                              name="delete object"
                              checked={objectDeleteQueue.includes(o.id)}
                              onChange={() => {
                                dispatch({
                                  type: TOGGLE_OBJECT_FOR_DELETE,
                                  data: o.id,
                                });
                              }}
                            />
                          </TableCell>
                          <TableCell>
                            {selectedTenant?.is_admin ? (
                              <Link
                                key={o.id}
                                href={
                                  `/objects/${o.id}` + makeCleanPageLink(query)
                                }
                              >
                                <TextShortener
                                  text={o.id}
                                  length={6}
                                  isCopyable={false}
                                />
                              </Link>
                            ) : (
                              <Text> {o.id}</Text>
                            )}
                          </TableCell>
                          <TableCell>{o.alias}</TableCell>
                          <TableCell>
                            {matchingType ? matchingType.type_name : o.type_id}
                          </TableCell>
                          <TableCell
                            className={PageCommon.listviewtabledeletecell}
                          >
                            <DeleteWithConfirmationButton
                              id="deleteObjectButton"
                              message="Are you sure you want to delete this object? This action is irreversible."
                              onConfirmDelete={() => {
                                if (selectedTenant) {
                                  dispatch(
                                    deleteObject(
                                      selectedTenant?.id,
                                      o.id,
                                      o.type_id
                                    )
                                  );
                                }
                              }}
                              title="Delete Object"
                            />
                          </TableCell>
                        </TableRow>
                      );
                    })}
                </TableBody>
              </Table>

              {selectedTenant?.is_admin && createButton && (
                <Link
                  href={'/objects/create' + makeCleanPageLink(query)}
                  applyStyles={false}
                >
                  <Button theme="outline">Create Object</Button>
                </Link>
              )}
            </>
          ) : (
            <CardRow>
              <EmptyState title="No objects" image={<IconLock2 size="large" />}>
                {selectedTenant?.is_admin && createButton && (
                  <Button theme="secondary">
                    <Link href={'/objects/create' + makeCleanPageLink(query)}>
                      Create Object
                    </Link>
                  </Button>
                )}
              </EmptyState>
            </CardRow>
          )}
        </>
      ) : isLoading ? (
        <Text>Loading...</Text>
      ) : (
        <InlineNotification theme="alert">
          {objectTypeError || objectError || 'Something went wrong'}
        </InlineNotification>
      )}
    </>
  );
};

const ObjectTable = connect((state: RootState) => {
  return {
    objects: state.authzObjects,
    objectError: state.fetchObjectsError,
    isLoading: state.fetchingAuthzObjects,
    objectDeleteQueue: state.objectDeleteQueue,
    editSuccess: state.editObjectsSuccess,
    editError: state.editObjectsError,
    lastFetchedObjectsTypeID: state.lastFetchedObjectTypeID,
    objectSearchFilter: state.currentObjectSearchFilter,
    location: state.location,
    query: state.query,
    featureFlags: state.featureFlags,
  };
})(Objects);

export default ObjectTable;
