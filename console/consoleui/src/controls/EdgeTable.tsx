import { useEffect } from 'react';
import { connect } from 'react-redux';
import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  Checkbox,
  EmptyState,
  IconLock2,
  IconFilter,
  InlineNotification,
  LoaderDots,
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
import { makeCleanPageLink } from '../AppNavigation';
import {
  BULK_UPDATE_EDGES_END,
  BULK_UPDATE_EDGES_START,
  changeCurrentEdgeSearchFilter,
  DELETE_EDGE_ERROR,
  DELETE_EDGE_SUCCESS,
  TOGGLE_EDGE_FOR_DELETE,
  TOGGLE_SELECT_ALL_EDGES,
} from '../actions/authz';
import { SelectedTenant } from '../models/Tenant';
import { Filter } from '../models/authz/SearchFilters';
import PaginatedResult from '../models/PaginatedResult';
import Edge, { edgeColumns, edgesPrefix } from '../models/authz/Edge';
import { deleteTenantAuthZEdge } from '../API/authzAPI';
import { fetchAuthZEdges } from '../thunks/authz';
import { postAlertToast, postSuccessToast } from '../thunks/notifications';
import Link from './Link';
import Search from './Search';
import Pagination from './Pagination';
import { getParamsAsObject } from './PaginationHelper';
import PageCommon from '../pages/PageCommon.module.css';
import DeleteWithConfirmationButton from './DeleteWithConfirmationButton';
import styles from './EdgeTable.module.css';

const PAGINATION_LIMIT = '50';

const fetchEdges =
  (selectedTenantID: string | undefined, params: URLSearchParams) =>
  (dispatch: AppDispatch) => {
    const paramsAsObject = getParamsAsObject(edgesPrefix, params);
    // if objects_limit is not specified in querystring,
    // use the default
    if (!paramsAsObject.limit) {
      paramsAsObject.limit = PAGINATION_LIMIT;
    }
    if (selectedTenantID) {
      dispatch(fetchAuthZEdges(selectedTenantID, paramsAsObject));
    }
  };

const deleteEdge =
  (tenantId: string, id: string) => (dispatch: AppDispatch) => {
    return deleteTenantAuthZEdge(tenantId, id).then(
      () => {
        dispatch(fetchEdges(tenantId, new URLSearchParams()));
        dispatch({
          type: DELETE_EDGE_SUCCESS,
          data: id,
        });
        dispatch(postSuccessToast('Successfully deleted edge'));
      },
      (error: Error) => {
        dispatch({
          type: DELETE_EDGE_ERROR,
          data: error,
        });
        dispatch(postAlertToast('Error deleting edge: ' + error));
      }
    );
  };

const onSaveEdges =
  () => async (dispatch: AppDispatch, getState: () => RootState) => {
    const { selectedTenantID, edgeDeleteQueue } = getState();
    if (!edgeDeleteQueue || !selectedTenantID) {
      return;
    }
    if (edgeDeleteQueue.length) {
      dispatch({
        type: BULK_UPDATE_EDGES_START,
      });
      let promises: Array<Promise<void>> = [];
      promises = edgeDeleteQueue.map((id) =>
        dispatch(deleteEdge(selectedTenantID, id))
      );

      Promise.all(promises).then(
        () => {
          dispatch({
            type: BULK_UPDATE_EDGES_END,
            data: true, // success
          });
          dispatch(fetchEdges(selectedTenantID, new URLSearchParams()));
        },
        () => {
          dispatch({
            type: BULK_UPDATE_EDGES_END,
            data: false, // complete or partial failure
          });
          dispatch(fetchEdges(selectedTenantID, new URLSearchParams()));
        }
      );
    }
  };

const changeEdgeSearchFilter =
  (changes: Record<string, string>) => async (dispatch: AppDispatch) => {
    // TODO v2 Add Operator Column and set the operator with that
    dispatch(changeCurrentEdgeSearchFilter(changes));
  };

type EdgesProps = {
  edges: PaginatedResult<Edge> | undefined;
  selectedTenant: SelectedTenant | undefined;
  edgeError: string;
  isLoading: boolean;
  editMode: boolean;
  edgeDeleteQueue: string[];
  editSuccess: string;
  editError: string;
  edgesSearchFilter: Filter;

  createButton?: boolean;
  query: URLSearchParams;
  dispatch: AppDispatch;
};

const Edges = ({
  edges,
  selectedTenant,
  edgeError,
  isLoading,
  editMode,
  editSuccess,
  editError,
  edgeDeleteQueue,
  edgesSearchFilter,
  createButton,
  query,
  dispatch,
}: EdgesProps) => {
  useEffect(() => {
    if (selectedTenant) {
      dispatch(fetchEdges(selectedTenant.id, query));
    }
  }, [selectedTenant, query, dispatch]);

  const deletePrompt = `Are you sure you want to delete ${
    edgeDeleteQueue.length
  } edge${edgeDeleteQueue.length === 1 ? '' : 's'}? This action is irreversible.`;

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div>
          <IconFilter />
        </div>
        <Search
          id="edges"
          columns={edgeColumns}
          changeSearchFilter={(filter: Filter) => {
            dispatch(changeEdgeSearchFilter(filter));
          }}
          prefix={edgesPrefix}
          searchFilter={edgesSearchFilter}
        />

        <ToolTip>
          <>
            {
              'Edges are one-way connections between the objects on your graph. They reflect the real-world relationships between objects in your systems. '
            }
            <a
              href="https://docs.userclouds.com/docs/key-concepts-1#edges"
              title="UserClouds documentation for key concepts in authorization"
              target="new"
              className={PageCommon.link}
            >
              Learn more here.
            </a>
          </>
        </ToolTip>
        {selectedTenant?.is_admin && (
          <ButtonGroup className={PageCommon.listviewtablecontrolsButtonGroup}>
            <Button theme="primary" size="small">
              <Link
                href={`/edges/create` + makeCleanPageLink(query)}
                applyStyles={false}
              >
                Create Edge
              </Link>
            </Button>
          </ButtonGroup>
        )}
      </div>

      <Card isDirty={editMode} listview>
        {isLoading && (
          <LoaderDots assistiveText="Loading ..." size="small" theme="brand" />
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
                id="deleteEdgesButton"
                message={deletePrompt}
                onConfirmDelete={() => {
                  dispatch(onSaveEdges());
                }}
                title="Delete Edges"
                disabled={edgeDeleteQueue.length < 1}
              />
            </div>

            <Pagination
              prev={edges?.prev}
              next={edges?.next}
              prefix={edgesPrefix}
              isLoading={false}
            />
          </div>

          {edges ? (
            edges.data && edges.data.length ? (
              <Table
                spacing="packed"
                isResponsive={false}
                id="edges"
                className={styles.edgestable}
              >
                <TableHead floating>
                  <TableRow>
                    <TableRowHead>
                      <Checkbox
                        checked={edgeDeleteQueue.length > 0}
                        onChange={() => {
                          dispatch({
                            type: TOGGLE_SELECT_ALL_EDGES,
                          });
                        }}
                      />
                    </TableRowHead>

                    <TableRowHead>Edge ID</TableRowHead>
                    <TableRowHead>Edge Type</TableRowHead>
                    <TableRowHead>Source Object</TableRowHead>
                    <TableRowHead>Target Object</TableRowHead>
                    <TableRowHead key="delete_header" />
                  </TableRow>
                </TableHead>
                <TableBody>
                  {edges &&
                    edges.data &&
                    edges.data.map((edge) => {
                      return (
                        <TableRow
                          key={edge.id}
                          className={
                            (edgeDeleteQueue.includes(edge.id)
                              ? PageCommon.queuedfordelete
                              : '') +
                            (' ' + PageCommon.listviewtablerow)
                          }
                        >
                          <TableCell>
                            <Checkbox
                              id={'delete' + edge.id}
                              name="delete object"
                              checked={edgeDeleteQueue.includes(edge.id)}
                              onChange={() => {
                                dispatch({
                                  type: TOGGLE_EDGE_FOR_DELETE,
                                  data: edge.id,
                                });
                              }}
                            />
                          </TableCell>
                          <TableCell>
                            <Link
                              key={edge.id}
                              href={
                                `/edges/${edge.id}` + makeCleanPageLink(query)
                              }
                            >
                              <TextShortener
                                text={edge.id}
                                length={6}
                                isCopyable={false}
                              />
                            </Link>
                          </TableCell>
                          <TableCell>
                            <Link
                              key={edge.edge_type_id}
                              href={
                                `/edgetypes/${edge.edge_type_id}` +
                                makeCleanPageLink(query)
                              }
                            >
                              <TextShortener
                                text={edge.edge_type_id}
                                length={6}
                                isCopyable={false}
                              />
                            </Link>
                          </TableCell>
                          <TableCell>
                            <Link
                              key={edge.source_object_id}
                              href={
                                `/objects/${edge.source_object_id}` +
                                makeCleanPageLink(query)
                              }
                            >
                              <TextShortener
                                text={edge.source_object_id}
                                length={6}
                                isCopyable={false}
                              />
                            </Link>
                          </TableCell>
                          <TableCell>
                            <Link
                              key={edge.target_object_id}
                              href={
                                `/objects/${edge.target_object_id}` +
                                makeCleanPageLink(query)
                              }
                            >
                              <TextShortener
                                text={edge.target_object_id}
                                length={6}
                                isCopyable={false}
                              />
                            </Link>
                          </TableCell>
                          <TableCell
                            className={PageCommon.listviewtabledeletecell}
                          >
                            <DeleteWithConfirmationButton
                              id="deleteEdgeButton"
                              message="Are you sure you want to delete this edge? This action is irreversible."
                              onConfirmDelete={() => {
                                if (selectedTenant) {
                                  dispatch(
                                    deleteEdge(selectedTenant?.id, edge.id)
                                  );
                                }
                              }}
                              title="Delete Edge"
                            />
                          </TableCell>
                        </TableRow>
                      );
                    })}
                </TableBody>
              </Table>
            ) : (
              <CardRow>
                <EmptyState title="No Edges" image={<IconLock2 size="large" />}>
                  {selectedTenant?.is_admin && createButton && (
                    <Button theme="secondary">
                      <Link href={`/edges/create` + makeCleanPageLink(query)}>
                        Create Edge
                      </Link>
                    </Button>
                  )}
                </EmptyState>
              </CardRow>
            )
          ) : isLoading ? (
            <LoaderDots
              assistiveText="Loading ..."
              size="small"
              theme="brand"
            />
          ) : (
            <Text>{edgeError || 'Loading...'}</Text>
          )}
        </>
      </Card>
    </>
  );
};

const EdgeTable = connect((state: RootState) => {
  return {
    edges: state.displayEdges,
    edgeError: state.fetchEdgesError,
    isLoading: state.fetchingAuthzEdges,
    editMode: state.edgeEditMode,
    edgeDeleteQueue: state.edgeDeleteQueue,
    editSuccess: state.editEdgesSuccess,
    editError: state.editEdgesError,
    savingEdges: state.savingEdges,
    edgesSearchFilter: state.currentEdgeSearchFilter,
    query: state.query,
    featureFlags: state.featureFlags,
  };
})(Edges);

export default EdgeTable;
export { EdgeTable };
