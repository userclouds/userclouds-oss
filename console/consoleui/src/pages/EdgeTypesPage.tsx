import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  Card,
  CardRow,
  Checkbox,
  EmptyState,
  Heading,
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
import { deleteTenantAuthZEdgeType } from '../API/authzAPI';
import { ObjectType } from '../models/authz/ObjectType';
import { SelectedTenant } from '../models/Tenant';
import EdgeType, {
  EDGE_TYPE_COLUMNS,
  EDGE_TYPE_PREFIX,
} from '../models/authz/EdgeType';
import PaginatedResult from '../models/PaginatedResult';
import { Filter } from '../models/authz/SearchFilters';
import { fetchAuthZObjectTypes, fetchAuthZEdgeTypes } from '../thunks/authz';
import { postAlertToast, postSuccessToast } from '../thunks/notifications';
import {
  DELETE_EDGE_TYPES_REQUEST,
  DELETE_EDGE_TYPES_SUCCESS,
  DELETE_EDGE_TYPES_ERROR,
  BULK_UPDATE_EDGE_TYPES_START,
  BULK_UPDATE_EDGE_TYPES_END,
  toggleEdgeTypeForDelete,
  changeCurrentEdgeTypeSearchFilter,
  toggleSelectAllEdgeTypes,
} from '../actions/authz';

import Link from '../controls/Link';
import Search from '../controls/Search';
import Pagination from '../controls/Pagination';
import DeleteWithConfirmationButton from '../controls/DeleteWithConfirmationButton';

import PageCommon from './PageCommon.module.css';
import styles from './EdgeTypesPage.module.css';

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
        dispatch(postSuccessToast('Successfully deleted edge type'));
      },
      (error: Error) => {
        dispatch({
          type: DELETE_EDGE_TYPES_ERROR,
          data: id,
        });
        dispatch(postAlertToast('Error deleting edge type: ' + error));
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
      promises = edgeTypeDeleteQueue.map((id) =>
        dispatch(deleteEdgeType(selectedTenantID, id))
      );
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
  deleteQueue,
  query,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  edgeType: EdgeType;
  objectTypes: PaginatedResult<ObjectType>;
  deleteQueue: string[];
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
      title={deleteQueue.includes(edgeType.id) ? 'Queued for delete' : ''}
      className={
        (deleteQueue.includes(edgeType.id) ? PageCommon.queuedfordelete : '') +
        ' ' +
        PageCommon.listviewtablerow
      }
    >
      <TableCell>
        <Checkbox
          id={'delete' + edgeType.id}
          name="delete policy"
          checked={deleteQueue.includes(edgeType.id)}
          onChange={() => {
            dispatch(toggleEdgeTypeForDelete(edgeType.id));
          }}
        />
      </TableCell>
      <TableCell>
        {selectedTenant?.is_admin ? (
          <Link
            key={edgeType.id}
            href={`/edgetypes/${edgeType.id}` + makeCleanPageLink(query)}
          >
            {edgeType.type_name}
          </Link>
        ) : (
          <Text>{edgeType.type_name}</Text>
        )}
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
      <TableCell>
        <TextShortener text={edgeType.id} length={6} />
      </TableCell>

      <TableCell align="right" className={PageCommon.listviewtabledeletecell}>
        <div className={PageCommon.listviewpaginationcontrolsdelete}>
          <DeleteWithConfirmationButton
            id="deleteEdgeTypeButton"
            message="Are you sure you want to delete this edge type? This action is irreversible."
            onConfirmDelete={() => {
              if (selectedTenant) {
                dispatch(deleteEdgeType(selectedTenant.id, edgeType.id));
              }
            }}
            title="Delete Edge Type"
          />
        </div>
      </TableCell>
    </TableRow>
  );
};

const EdgeTypes = ({
  selectedTenant,
  objectTypes,
  objectTypeError,
  edgeTypes,
  edgeTypeError,
  deleteQueue,
  fetchingEdgeTypes,
  fetchingObjectTypes,
  query,
  edgeTypeSearchFilter,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  objectTypes: PaginatedResult<ObjectType> | undefined;
  objectTypeError: string;
  edgeTypes: PaginatedResult<EdgeType> | undefined;
  edgeTypeError: string;
  fetchingEdgeTypes: boolean;
  fetchingObjectTypes: boolean;
  deleteQueue: string[];
  query: URLSearchParams;
  edgeTypeSearchFilter: Filter;
  dispatch: AppDispatch;
}) => {
  const deletePrompt = `Are you sure you want to delete ${
    deleteQueue.length
  } edge type${deleteQueue.length === 1 ? '' : 's'}? This action is irreversible.`;

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div>
          <IconFilter />
        </div>
        <Search
          id="edgeTypes"
          columns={EDGE_TYPE_COLUMNS}
          changeSearchFilter={(filter: Filter) => {
            dispatch(changeCurrentEdgeTypeSearchFilter(filter));
          }}
          prefix={EDGE_TYPE_PREFIX}
          searchFilter={edgeTypeSearchFilter}
        />
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
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
          </ToolTip>
        </div>

        {selectedTenant?.is_admin && (
          <Button
            theme="primary"
            size="small"
            className={PageCommon.listviewtablecontrolsButton}
          >
            <Link
              href={'/edgetypes/create' + makeCleanPageLink(query)}
              applyStyles={false}
            >
              {' '}
              Create Edge Type
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
        {edgeTypes && objectTypes ? (
          edgeTypes.data && edgeTypes.data.length ? (
            <>
              <div className={PageCommon.listviewpaginationcontrols}>
                <div className={PageCommon.listviewpaginationcontrolsdelete}>
                  <DeleteWithConfirmationButton
                    id="deleteMutatorsButton"
                    message={deletePrompt}
                    onConfirmDelete={() => {
                      dispatch(onSaveEdgeTypes());
                    }}
                    title="Delete Edge Types"
                    disabled={deleteQueue.length < 1}
                  />
                </div>
                <Pagination
                  prev={edgeTypes?.prev}
                  next={edgeTypes?.next}
                  isLoading={fetchingEdgeTypes}
                  prefix={EDGE_TYPE_PREFIX}
                />
              </div>
              <Table
                spacing="nowrap"
                id="edgeTypes"
                className={styles.edgetypestable}
              >
                <TableHead floating>
                  <TableRow>
                    <TableRowHead>
                      <Checkbox
                        checked={
                          Object.keys(deleteQueue).length ===
                          edgeTypes.data.length
                        }
                        onChange={() => {
                          dispatch(toggleSelectAllEdgeTypes());
                        }}
                      />
                    </TableRowHead>
                    <TableRowHead key="edge_type_name">Type Name</TableRowHead>
                    <TableRowHead key="edge_type_source_object_type">
                      Source Object Type
                    </TableRowHead>
                    <TableRowHead key="edge_type_target_object_type">
                      Target Object Type
                    </TableRowHead>
                    <TableRowHead key="edge_type_id">ID</TableRowHead>
                    <TableRowHead key="edge_type_delete_head" />
                  </TableRow>
                </TableHead>
                <TableBody>
                  {edgeTypes.data.map((et) => (
                    <EdgeTypesRow
                      selectedTenant={selectedTenant}
                      edgeType={et}
                      objectTypes={objectTypes}
                      key={et.id}
                      deleteQueue={deleteQueue}
                      query={query}
                      dispatch={dispatch}
                    />
                  ))}
                </TableBody>
              </Table>
            </>
          ) : (
            <CardRow>
              <EmptyState
                title="No edge types"
                image={<IconLock2 size="large" />}
              >
                <Button theme="secondary">
                  <Link
                    href={'/edgetypes/create' + makeCleanPageLink(query)}
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
    </>
  );
};
const ConnectedEdgeTypes = connect((state: RootState) => {
  return {
    edgeError: state.fetchEdgeTypesError,
    editSuccess: state.editObjectsSuccess,
    editError: state.editObjectsError,
    deleteQueue: state.edgeTypeDeleteQueue,
    query: state.query,
    edgeTypeSearchFilter: state.edgeTypeSearchFilter,
  };
})(EdgeTypes);

const EdgeTypesPage = ({
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
      if (!objectTypes) {
        dispatch(
          fetchAuthZObjectTypes(selectedTenant.id, new URLSearchParams())
        );
      }
      dispatch(fetchAuthZEdgeTypes(selectedTenant.id, query));
    }
  }, [dispatch, objectTypes, selectedTenant, query]);

  return (
    <ConnectedEdgeTypes
      selectedTenant={selectedTenant}
      objectTypes={objectTypes}
      objectTypeError={objectTypeError}
      fetchingEdgeTypes={fetchingEdgeTypes}
      fetchingObjectTypes={fetchingObjectTypes}
      edgeTypes={edgeTypes}
      edgeTypeError={edgeTypeError}
    />
  );
};

const ConnectedEdgeTypesPage = connect((state: RootState) => {
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
})(EdgeTypesPage);

export default ConnectedEdgeTypesPage;
