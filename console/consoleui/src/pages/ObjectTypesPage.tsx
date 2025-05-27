import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  Card,
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
  TextShortener,
  ToolTip,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import { AppDispatch, RootState } from '../store';
import { deleteTenantAuthZObjectType } from '../API/authzAPI';
import {
  ObjectType,
  OBJECT_TYPE_COLUMNS,
  OBJECT_TYPE_PREFIX,
} from '../models/authz/ObjectType';
import { SelectedTenant } from '../models/Tenant';

import PaginatedResult from '../models/PaginatedResult';
import { Filter } from '../models/authz/SearchFilters';
import { fetchAuthZObjectTypes } from '../thunks/authz';
import { postAlertToast, postSuccessToast } from '../thunks/notifications';
import {
  DELETE_OBJECT_TYPES_REQUEST,
  DELETE_OBJECT_TYPES_SUCCESS,
  DELETE_OBJECT_TYPES_ERROR,
  BULK_UPDATE_OBJECT_TYPES_START,
  BULK_UPDATE_OBJECT_TYPES_END,
  TOGGLE_OBJECT_TYPE_FOR_DELETE,
  changeCurrentObjectTypeSearchFilter,
  toggleSelectAllObjectTypes,
} from '../actions/authz';
import Search from '../controls/Search';
import Pagination from '../controls/Pagination';
import DeleteWithConfirmationButton from '../controls/DeleteWithConfirmationButton';
import Link from '../controls/Link';
import PageCommon from './PageCommon.module.css';
import styles from './ObjectTypesPage.module.css';

const deleteObjectType =
  (tenantId: string, id: string) => (dispatch: AppDispatch) => {
    dispatch({
      type: DELETE_OBJECT_TYPES_REQUEST,
      data: id,
    });
    return deleteTenantAuthZObjectType(tenantId, id).then(
      () => {
        dispatch(fetchAuthZObjectTypes(tenantId, new URLSearchParams()));
        dispatch({
          type: DELETE_OBJECT_TYPES_SUCCESS,
          data: id,
        });
        dispatch(postSuccessToast('Successfully deleted object type'));
      },
      (error: Error) => {
        dispatch({
          type: DELETE_OBJECT_TYPES_ERROR,
          data: id,
        });
        dispatch(postAlertToast('Error deleting object type:' + error));
      }
    );
  };

const onSaveObjectTypes =
  () => async (dispatch: AppDispatch, getState: () => RootState) => {
    const { selectedTenantID, objectTypes, objectTypeDeleteQueue } = getState();
    if (!objectTypes || !selectedTenantID || !objectTypeDeleteQueue) {
      return;
    }
    dispatch({
      type: BULK_UPDATE_OBJECT_TYPES_START,
    });
    let promises: Array<Promise<void>> = [];
    if (objectTypeDeleteQueue.length) {
      promises = objectTypeDeleteQueue.map((id) =>
        dispatch(deleteObjectType(selectedTenantID, id))
      );
    }
    Promise.all(promises as Array<Promise<void>>).then(
      () => {
        dispatch({
          type: BULK_UPDATE_OBJECT_TYPES_END,
          data: true, // success
        });
        dispatch(
          fetchAuthZObjectTypes(selectedTenantID, new URLSearchParams())
        );
      },
      () => {
        dispatch({
          type: BULK_UPDATE_OBJECT_TYPES_END,
          data: false, // complete or partial failure
        });
        dispatch(
          fetchAuthZObjectTypes(selectedTenantID, new URLSearchParams())
        );
      }
    );
  };

const ObjectTypes = ({
  selectedTenant,
  objectTypes,
  error,
  deleteQueue,
  fetchingObjectTypes,
  dispatch,
  query,
  objectTypeSearchFilter,
}: {
  selectedTenant: SelectedTenant | undefined;
  objectTypes: PaginatedResult<ObjectType> | undefined;
  error: string;
  deleteQueue: string[];
  fetchingObjectTypes: boolean;
  query: URLSearchParams;
  objectTypeSearchFilter: Filter;
  dispatch: AppDispatch;
}) => {
  const deletePrompt = `Are you sure you want to delete ${
    deleteQueue.length
  } object type${deleteQueue.length === 1 ? '' : 's'}? This action is irreversible.`;

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div>
          <IconFilter />
        </div>
        <Search
          id="objectTypes"
          columns={OBJECT_TYPE_COLUMNS}
          changeSearchFilter={(filter: Filter) => {
            dispatch(changeCurrentObjectTypeSearchFilter(filter));
          }}
          prefix={OBJECT_TYPE_PREFIX}
          searchFilter={objectTypeSearchFilter}
        />
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              {
                'Object types, like users and groups, define the structure of your authorization model. All objects have exactly one type. '
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
        </div>

        {selectedTenant?.is_admin && (
          <Button
            theme="primary"
            size="small"
            className={PageCommon.listviewtablecontrolsButton}
          >
            <Link
              href={'/objecttypes/create' + makeCleanPageLink(query)}
              applyStyles={false}
            >
              {' '}
              Create Object Type
            </Link>
          </Button>
        )}
      </div>

      <Card
        lockedMessage={
          !selectedTenant?.is_admin ? 'You do not have edit access' : ''
        }
        listview
      >
        {objectTypes?.data ? (
          objectTypes.data.length ? (
            <>
              <div className={PageCommon.listviewpaginationcontrols}>
                <div className={PageCommon.listviewpaginationcontrolsdelete}>
                  <DeleteWithConfirmationButton
                    id="deleteObjectTypesButton"
                    message={deletePrompt}
                    onConfirmDelete={() => {
                      dispatch(onSaveObjectTypes());
                    }}
                    title="Delete Object Types"
                    disabled={deleteQueue.length < 1}
                  />
                </div>
                <Pagination
                  prev={objectTypes?.prev}
                  next={objectTypes?.next}
                  isLoading={fetchingObjectTypes}
                  prefix={OBJECT_TYPE_PREFIX}
                />
              </div>
              <Table
                spacing="packed"
                id="objectTypes"
                className={styles.objecttypestable}
              >
                <TableHead floating>
                  <TableRow>
                    <TableRowHead>
                      <Checkbox
                        checked={
                          Object.keys(deleteQueue).length ===
                          objectTypes.data.length
                        }
                        onChange={() => {
                          dispatch(toggleSelectAllObjectTypes());
                        }}
                      />
                    </TableRowHead>
                    <TableRowHead key="object_type_name">
                      Type Name
                    </TableRowHead>

                    <TableRowHead key="object_type_id">ID</TableRowHead>
                    <TableRowHead key="delete_head" />
                  </TableRow>
                </TableHead>
                <TableBody>
                  {objectTypes.data.map((ot) => (
                    <TableRow
                      key={ot.id}
                      className={
                        (deleteQueue.includes(ot.id)
                          ? PageCommon.queuedfordelete
                          : '') +
                        (' ' + PageCommon.listviewtablerow)
                      }
                    >
                      <TableCell>
                        <Checkbox
                          id={'delete' + ot.id}
                          name="delete object type"
                          checked={deleteQueue.includes(ot.id)}
                          onChange={() => {
                            dispatch({
                              type: TOGGLE_OBJECT_TYPE_FOR_DELETE,
                              data: ot.id,
                            });
                          }}
                        />
                      </TableCell>
                      <TableCell title={ot.id}>
                        {selectedTenant?.is_admin ? (
                          <Link
                            key={ot.type_name}
                            href={
                              '/objecttypes/' + ot.id + makeCleanPageLink(query)
                            }
                          >
                            {ot.type_name}
                          </Link>
                        ) : (
                          <Text>{ot.type_name}</Text>
                        )}
                      </TableCell>
                      <TableCell>
                        <TextShortener text={ot.id} length={6} />
                      </TableCell>
                      <TableCell
                        align="right"
                        className={PageCommon.listviewtabledeletecell}
                      >
                        <DeleteWithConfirmationButton
                          id="deleteObjectTypeButton"
                          message="Are you sure you want to delete this object type? This action is irreversible."
                          onConfirmDelete={() => {
                            if (selectedTenant) {
                              dispatch(
                                deleteObjectType(selectedTenant?.id, ot.id)
                              );
                            }
                          }}
                          title="Delete Object Type"
                        />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </>
          ) : (
            <CardRow>
              <EmptyState
                title="No object types"
                image={<IconLock2 size="large" />}
              >
                <Button theme="secondary">
                  <Link
                    href={'/objecttypes/create' + makeCleanPageLink(query)}
                    applyStyles={false}
                  >
                    Create Object Type
                  </Link>
                </Button>
              </EmptyState>
            </CardRow>
          )
        ) : fetchingObjectTypes ? (
          <Text element="h4">Loading...</Text>
        ) : (
          <InlineNotification theme="alert">
            {error || 'Something went wrong'}
          </InlineNotification>
        )}
      </Card>
    </>
  );
};
const ConnectedObjectTypes = connect((state: RootState) => {
  return {
    selectedTenant: state.selectedTenant,
    objectTypes: state.objectTypes,
    oError: state.fetchObjectTypesError,
    objectTypeEditMode: state.objectTypeEditMode,
    deleteQueue: state.objectTypeDeleteQueue,
    savingObjectTypes: state.savingObjectTypes,
    location: state.location,
    query: state.query,
    objectTypeSearchFilter: state.objectTypeSearchFilter,
  };
})(ObjectTypes);

const ObjectTypesPage = ({
  objectTypeError,
  fetchingObjectTypes,
  selectedTenant,
  query,
  dispatch,
}: {
  objectTypeError: string;
  fetchingObjectTypes: boolean;
  selectedTenant: SelectedTenant | undefined;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenant) {
      dispatch(fetchAuthZObjectTypes(selectedTenant.id, query));
    }
  }, [dispatch, selectedTenant, query]);

  return (
    <ConnectedObjectTypes
      fetchingObjectTypes={fetchingObjectTypes}
      error={objectTypeError}
    />
  );
};

const ConnectedObjectTypesPage = connect((state: RootState) => {
  return {
    objectTypeError: state.fetchObjectTypesError,
    fetchingObjectTypes: state.fetchingObjectTypes,
    selectedTenant: state.selectedTenant,
    query: state.query,
  };
})(ObjectTypesPage);

export default ConnectedObjectTypesPage;
