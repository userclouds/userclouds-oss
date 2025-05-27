import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  CardFooter,
  EmptyState,
  Heading,
  IconButton,
  IconDeleteBin,
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
} from '@userclouds/ui-component-lib';

import { PageTitle } from '../mainlayout/PageWrap';
import { makeCleanPageLink } from '../AppNavigation';
import { AppDispatch, RootState } from '../store';
import {
  deleteTenantAuthZObjectType,
  deleteTenantAuthZEdgeType,
} from '../API/authzAPI';
import { ObjectType } from '../models/authz/ObjectType';
import { SelectedTenant } from '../models/Tenant';
import EdgeType from '../models/authz/EdgeType';
import { fetchAuthZObjectTypes, fetchAuthZEdgeTypes } from '../thunks/authz';
import Link from '../controls/Link';
import ObjectTable from '../controls/ObjectTable';
import AuthorizationCheck from '../controls/AuthorizationCheck';
import PageCommon from './PageCommon.module.css';
import {
  DELETE_OBJECT_TYPES_REQUEST,
  DELETE_OBJECT_TYPES_SUCCESS,
  DELETE_OBJECT_TYPES_ERROR,
  BULK_UPDATE_OBJECT_TYPES_START,
  BULK_UPDATE_OBJECT_TYPES_END,
  TOGGLE_OBJECT_TYPE_FOR_DELETE,
  TOGGLE_OBJECT_TYPE_EDIT_MODE,
  DELETE_EDGE_TYPES_REQUEST,
  DELETE_EDGE_TYPES_SUCCESS,
  DELETE_EDGE_TYPES_ERROR,
  BULK_UPDATE_EDGE_TYPES_START,
  BULK_UPDATE_EDGE_TYPES_END,
  TOGGLE_EDGE_TYPE_EDIT_MODE,
  toggleEdgeTypeForDelete,
} from '../actions/authz';
import Breadcrumbs from '../controls/Breadcrumbs';
import PaginatedResult from '../models/PaginatedResult';

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
      },
      () => {
        dispatch({
          type: DELETE_OBJECT_TYPES_ERROR,
          data: id,
        });
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
  objectTypeEditMode,
  savingObjectTypes,
  objectTypeDeleteQueue,
  fetchingObjectTypes,
  dispatch,
  readOnly,
  location,
  query,
}: {
  selectedTenant: SelectedTenant | undefined;
  objectTypes: PaginatedResult<ObjectType> | undefined;
  error: string;
  objectTypeEditMode: boolean;
  savingObjectTypes: boolean;
  objectTypeDeleteQueue: string[];
  fetchingObjectTypes: boolean;
  readOnly: boolean;
  location: URL;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  return (
    <Card
      title="Object types"
      description={
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
      }
      lockedMessage={
        !selectedTenant?.is_admin ? 'You do not have edit access' : ''
      }
    >
      {objectTypes ? (
        objectTypes.data && objectTypes.data.length ? (
          <>
            <Table>
              <TableHead>
                <TableRow>
                  <TableRowHead key="object_type_name">Type Name</TableRowHead>
                  {objectTypeEditMode && <TableRowHead key="delete_head" />}
                  <TableRowHead key="object_type_id">ID</TableRowHead>
                </TableRow>
              </TableHead>
              <TableBody>
                {objectTypes.data.map((ot) => (
                  <TableRow
                    key={ot.id}
                    className={
                      objectTypeDeleteQueue.includes(ot.id)
                        ? PageCommon.queuedfordelete
                        : ''
                    }
                  >
                    <TableCell title={ot.id}>
                      {selectedTenant?.is_admin ? (
                        <Link
                          key={ot.type_name}
                          href={
                            '/authz/objecttypes/' +
                            ot.id +
                            makeCleanPageLink(query)
                          }
                        >
                          {ot.type_name}
                        </Link>
                      ) : (
                        <Text>{ot.type_name}</Text>
                      )}
                    </TableCell>
                    <TableCell>
                      <TextShortener text={ot.id} length={36} />
                    </TableCell>
                    {objectTypeEditMode && (
                      <TableCell>
                        <IconButton
                          icon={<IconDeleteBin />}
                          onClick={() => {
                            dispatch({
                              type: TOGGLE_OBJECT_TYPE_FOR_DELETE,
                              data: ot.id,
                            });
                          }}
                          title="Delete object type"
                          aria-label="Delete object type"
                        />
                      </TableCell>
                    )}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            {selectedTenant?.is_admin && (
              <Link
                href={
                  location.pathname +
                  '/objecttypes/create' +
                  makeCleanPageLink(query)
                }
                applyStyles={false}
              >
                <Button theme="outline">Create Object Type</Button>
              </Link>
            )}
            <CardFooter>
              {!readOnly && (
                <ButtonGroup>
                  {!objectTypeEditMode ? (
                    <Button
                      theme="secondary"
                      onClick={() => {
                        dispatch({
                          type: TOGGLE_OBJECT_TYPE_EDIT_MODE,
                        });
                      }}
                    >
                      Edit
                    </Button>
                  ) : (
                    <>
                      <Button
                        theme="primary"
                        isLoading={savingObjectTypes}
                        disabled={!objectTypeEditMode}
                        onClick={() => {
                          dispatch(onSaveObjectTypes());
                        }}
                      >
                        Save
                      </Button>
                      <Button
                        theme="secondary"
                        onClick={() => {
                          dispatch({
                            type: TOGGLE_OBJECT_TYPE_EDIT_MODE,
                          });
                        }}
                      >
                        Cancel
                      </Button>
                    </>
                  )}
                </ButtonGroup>
              )}
            </CardFooter>
          </>
        ) : (
          <CardRow>
            <EmptyState
              title="No object types"
              image={<IconLock2 size="large" />}
            >
              <Button theme="secondary">
                <Link
                  href={
                    location.pathname +
                    '/objecttypes/create' +
                    makeCleanPageLink(query)
                  }
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
  );
};
const ConnectedObjectTypes = connect((state: RootState) => {
  return {
    selectedTenant: state.selectedTenant,
    objectTypes: state.objectTypes,
    oError: state.fetchObjectTypesError,
    objectTypeEditMode: state.objectTypeEditMode,
    objectTypeDeleteQueue: state.objectTypeDeleteQueue,
    savingObjectTypes: state.savingObjectTypes,
    location: state.location,
    query: state.query,
  };
})(ObjectTypes);

const deleteEdgeType =
  (tenantId: string, id: string) => (dispatch: AppDispatch) => {
    dispatch({
      type: DELETE_EDGE_TYPES_REQUEST,
      data: id,
    });
    return deleteTenantAuthZEdgeType(tenantId, id).then(
      () => {
        dispatch(fetchAuthZEdgeTypes(tenantId, new URLSearchParams()));
        dispatch({
          type: DELETE_EDGE_TYPES_SUCCESS,
          data: id,
        });
      },
      () => {
        dispatch({
          type: DELETE_EDGE_TYPES_ERROR,
          data: id,
        });
      }
    );
  };

const onSaveEdgeTypes =
  () => async (dispatch: AppDispatch, getState: () => RootState) => {
    const { selectedTenantID, edgeTypes, edgeTypeDeleteQueue } = getState();
    if (!edgeTypes || !selectedTenantID || !edgeTypeDeleteQueue) {
      return;
    }
    dispatch({
      type: BULK_UPDATE_EDGE_TYPES_START,
    });
    let promises: Array<Promise<void>> = [];
    if (edgeTypeDeleteQueue.length) {
      if (
        window.confirm(
          `Are you sure you want to remove ${
            edgeTypeDeleteQueue.length
          } edge type${edgeTypeDeleteQueue.length > 1 ? 's' : ''}?`
        )
      ) {
        promises = edgeTypeDeleteQueue.map((id) =>
          dispatch(deleteEdgeType(selectedTenantID, id))
        );
      }
      Promise.all(promises as Array<Promise<void>>).then(
        () => {
          dispatch({
            type: BULK_UPDATE_EDGE_TYPES_END,
            data: true, // success
          });
          dispatch(
            fetchAuthZEdgeTypes(selectedTenantID, new URLSearchParams())
          );
        },
        () => {
          dispatch({
            type: BULK_UPDATE_EDGE_TYPES_END,
            data: false, // complete or partial failure
          });
          dispatch(
            fetchAuthZEdgeTypes(selectedTenantID, new URLSearchParams())
          );
        }
      );
    }
  };

const EdgeTypesRow = ({
  selectedTenant,
  edgeType,
  objectTypes,
  edgeTypeEditMode,
  edgeTypeDeleteQueue,
  query,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  edgeType: EdgeType;
  objectTypes: PaginatedResult<ObjectType>;
  edgeTypeEditMode: boolean;
  edgeTypeDeleteQueue: string[];
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  if (!objectTypes || !objectTypes.data) {
    return <Heading>Loading...</Heading>;
  }
  const sourceObjectType = objectTypes.data.find(
    (ot) => ot.id === edgeType.source_object_type_id
  );
  const targetObjectType = objectTypes.data.find(
    (ot) => ot.id === edgeType.target_object_type_id
  );

  return (
    <TableRow
      key={edgeType.id}
      title={
        edgeTypeDeleteQueue.includes(edgeType.id) ? 'Queued for delete' : ''
      }
      className={
        edgeTypeDeleteQueue.includes(edgeType.id)
          ? PageCommon.queuedfordelete
          : ''
      }
    >
      <TableCell>
        {selectedTenant?.is_admin ? (
          <Link
            key={edgeType.id}
            href={`/authz/edgetypes/${edgeType.id}` + makeCleanPageLink(query)}
          >
            {edgeType.type_name}
          </Link>
        ) : (
          <Text>{edgeType.type_name}</Text>
        )}
      </TableCell>
      <TableCell>
        <TextShortener text={edgeType.id} length={36} />
      </TableCell>
      <TableCell>
        {sourceObjectType
          ? sourceObjectType.type_name
          : edgeType.source_object_type_id}
      </TableCell>
      <TableCell>
        {targetObjectType
          ? targetObjectType.type_name
          : edgeType.target_object_type_id}
      </TableCell>
      {edgeTypeEditMode && (
        <TableCell>
          <IconButton
            icon={<IconDeleteBin />}
            onClick={() => {
              dispatch(toggleEdgeTypeForDelete(edgeType.id));
            }}
            title="Delete Edge Type"
            aria-label="Delete Edge Type"
          />
        </TableCell>
      )}
    </TableRow>
  );
};

const EdgeTypes = ({
  selectedTenant,
  objectTypes,
  objectTypeError,
  edgeTypes,
  edgeTypeError,
  edgeTypeEditMode,
  edgeTypeDeleteQueue,
  savingEdgeTypes,
  fetchingEdgeTypes,
  fetchingObjectTypes,
  location,
  query,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  objectTypes: PaginatedResult<ObjectType> | undefined;
  objectTypeError: string;
  edgeTypes: PaginatedResult<EdgeType> | undefined;
  edgeTypeError: string;
  edgeTypeEditMode: boolean;
  savingEdgeTypes: boolean;
  fetchingEdgeTypes: boolean;
  fetchingObjectTypes: boolean;
  edgeTypeDeleteQueue: string[];
  location: URL;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  return (
    <Card
      title="Edge types"
      description={
        <>
          {
            'An edge type represents a possible relationship between two objects. Each edge has exactly one type. '
          }
          <a
            href="https://docs.userclouds.com/docs/key-concepts-1#edge-types"
            title="UserClouds documentation for key concepts in authorization"
            target="new"
            className={PageCommon.link}
          >
            Learn more here.
          </a>
        </>
      }
      lockedMessage={
        !selectedTenant?.is_admin ? 'You do not have edit access' : ''
      }
    >
      {edgeTypes && objectTypes ? (
        edgeTypes.data && edgeTypes.data.length ? (
          <>
            <Table>
              <TableHead>
                <TableRow>
                  <TableRowHead key="edge_type_name">Type Name</TableRowHead>
                  <TableRowHead key="edge_type_id">ID</TableRowHead>
                  <TableRowHead key="edge_type_source_object_type">
                    Source Object Type
                  </TableRowHead>
                  <TableRowHead key="edge_type_target_object_type">
                    Target Object Type
                  </TableRowHead>
                  {edgeTypeEditMode && (
                    <TableRowHead key="edge_type_delete_head" />
                  )}
                </TableRow>
              </TableHead>
              <TableBody>
                {edgeTypes.data.map((et) => (
                  <EdgeTypesRow
                    selectedTenant={selectedTenant}
                    edgeType={et}
                    objectTypes={objectTypes}
                    key={et.id}
                    edgeTypeEditMode={edgeTypeEditMode}
                    edgeTypeDeleteQueue={edgeTypeDeleteQueue}
                    query={query}
                    dispatch={dispatch}
                  />
                ))}
              </TableBody>
            </Table>
            {selectedTenant?.is_admin && (
              <>
                <ButtonGroup>
                  <Link
                    href={
                      location.pathname +
                      '/edgetypes/create' +
                      makeCleanPageLink(query)
                    }
                    applyStyles={false}
                  >
                    <Button theme="outline">Create Edge Type</Button>
                  </Link>
                  <Link
                    href={
                      location.pathname +
                      '/edges/create' +
                      makeCleanPageLink(query)
                    }
                    applyStyles={false}
                  >
                    <Button theme="outline">Create Edge</Button>
                  </Link>
                  <Link
                    href={
                      location.pathname + '/edges' + makeCleanPageLink(query)
                    }
                    applyStyles={false}
                  >
                    <Button theme="outline">Manage edges</Button>
                  </Link>
                </ButtonGroup>
                <CardFooter>
                  <ButtonGroup>
                    {!edgeTypeEditMode ? (
                      <Button
                        theme="secondary"
                        onClick={() => {
                          dispatch({
                            type: TOGGLE_EDGE_TYPE_EDIT_MODE,
                          });
                        }}
                      >
                        Edit
                      </Button>
                    ) : (
                      <>
                        <Button
                          theme="primary"
                          isLoading={savingEdgeTypes}
                          disabled={!edgeTypeEditMode}
                          onClick={() => {
                            dispatch(onSaveEdgeTypes());
                          }}
                        >
                          Save
                        </Button>
                        <Button
                          theme="secondary"
                          onClick={() => {
                            dispatch({
                              type: TOGGLE_EDGE_TYPE_EDIT_MODE,
                            });
                          }}
                        >
                          Cancel
                        </Button>
                      </>
                    )}
                  </ButtonGroup>
                </CardFooter>
              </>
            )}
          </>
        ) : (
          <CardRow>
            <EmptyState
              title="No edge types"
              image={<IconLock2 size="large" />}
            >
              <Button theme="secondary">
                <Link
                  href={
                    location.pathname +
                    '/edgetypes/create' +
                    makeCleanPageLink(query)
                  }
                  applyStyles={false}
                >
                  Create Edge Type
                </Link>
              </Button>
            </EmptyState>
          </CardRow>
        )
      ) : fetchingEdgeTypes || fetchingObjectTypes ? (
        <Text element="h4">Loading...</Text>
      ) : (
        <InlineNotification theme="alert">
          {objectTypeError || edgeTypeError || 'Something went wrong'}
        </InlineNotification>
      )}
    </Card>
  );
};
const ConnectedEdgeTypes = connect((state: RootState) => {
  return {
    edgeType: state.edgeTypes,
    edgeError: state.fetchEdgeTypesError,
    edgeTypeEditMode: state.edgeTypeEditMode,
    editSuccess: state.editObjectsSuccess,
    editError: state.editObjectsError,
    edgeTypeDeleteQueue: state.edgeTypeDeleteQueue,
    savingEdgeTypes: state.savingEdgeTypes,
    location: state.location,
    query: state.query,
  };
})(EdgeTypes);

const AuthZPage = ({
  objectTypes,
  objectTypeError,
  fetchingObjectTypes,
  edgeTypes,
  edgeTypeError,
  fetchingEdgeTypes,
  selectedTenant,
  query,
  dispatch,
}: {
  objectTypes: PaginatedResult<ObjectType> | undefined;
  objectTypeError: string;
  fetchingObjectTypes: boolean;
  edgeTypes: PaginatedResult<EdgeType> | undefined;
  edgeTypeError: string;
  fetchingEdgeTypes: boolean;
  selectedTenant: SelectedTenant | undefined;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenant) {
      if (!objectTypes && !fetchingObjectTypes) {
        dispatch(fetchAuthZObjectTypes(selectedTenant.id, query));
      }
      if (!edgeTypes && !fetchingEdgeTypes) {
        dispatch(fetchAuthZEdgeTypes(selectedTenant.id, query));
      }
    }
  }, [
    dispatch,
    objectTypes,
    fetchingObjectTypes,
    edgeTypes,
    fetchingEdgeTypes,
    selectedTenant,
    query,
  ]);

  return (
    <>
      <Breadcrumbs />

      <PageTitle
        title="Authorization"
        description={
          <>
            {
              'Manage authorization (roles, permissions & relationships) in your tenant. '
            }
            <a
              href="https://docs.userclouds.com/docs/key-concepts-1"
              title="UserClouds documentation for key concepts in authorization"
              target="new"
              className={PageCommon.link}
            >
              Learn more here.
            </a>
          </>
        }
      />
      <ConnectedObjectTypes
        fetchingObjectTypes={fetchingObjectTypes}
        error={objectTypeError}
        readOnly={!selectedTenant?.is_admin}
      />
      <ConnectedEdgeTypes
        selectedTenant={selectedTenant}
        objectTypes={objectTypes}
        objectTypeError={objectTypeError}
        fetchingEdgeTypes={fetchingEdgeTypes}
        fetchingObjectTypes={fetchingObjectTypes}
        edgeTypes={edgeTypes}
        edgeTypeError={edgeTypeError}
      />
      <ObjectTable
        selectedTenant={selectedTenant}
        objectTypes={objectTypes}
        objectTypeError={objectTypeError}
        createButton
      />
      <AuthorizationCheck selectedTenantID={selectedTenant?.id} />
    </>
  );
};

const ConnectedAuthZPage = connect((state: RootState) => {
  return {
    objectTypes: state.objectTypes,
    objectTypeError: state.fetchObjectTypesError,
    fetchingObjectTypes: state.fetchingObjectTypes,
    edgeTypes: state.edgeTypes,
    edgeTypeError: state.fetchEdgeTypesError,
    fetchingEdgeTypes: state.fetchingEdgeTypes,
    selectedTenant: state.selectedTenant,
    query: state.query,
  };
})(AuthZPage);

export default ConnectedAuthZPage;
