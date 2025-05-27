import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  Card,
  InputReadOnly,
  Label,
  InlineNotification,
  Select,
  Text,
  TextInput,
  ToolTip,
  ButtonGroup,
  CardRow,
} from '@userclouds/ui-component-lib';

import { PageTitle } from '../mainlayout/PageWrap';
import { AppDispatch, RootState } from '../store';
import { makeCleanPageLink } from '../AppNavigation';
import { createTenantAuthZEdge } from '../API/authzAPI';
import {
  fetchAuthZObjectTypes,
  fetchAuthZEdgeTypes,
  fetchAuthZEdge,
} from '../thunks/authz';
import {
  CHANGE_EDGE,
  RETRIEVE_BLANK_EDGE,
  CREATE_EDGE_REQUEST,
  CREATE_EDGE_SUCCESS,
  CREATE_EDGE_ERROR,
} from '../actions/authz';
import EdgeType from '../models/authz/EdgeType';
import Edge from '../models/authz/Edge';
import { ObjectType } from '../models/authz/ObjectType';
import { SelectedTenant } from '../models/Tenant';
import Link from '../controls/Link';
import PageCommon from './PageCommon.module.css';
import { redirect } from '../routing';
import { postSuccessToast } from '../thunks/notifications';
import PaginatedResult from '../models/PaginatedResult';

const CreateEdgeEditPage = ({
  edge,
  objectTypes,
  edgeTypes,
  selectedTenantID,
  saveSuccess,
  saveError,
  edgeError,
  edgeTypeError,
  location,
  query,
  routeParams,
  dispatch,
}: {
  edge: Edge | undefined;
  objectTypes: PaginatedResult<ObjectType> | undefined;
  edgeTypes: PaginatedResult<EdgeType> | undefined;
  selectedTenantID: string | undefined;
  saveSuccess: string;
  saveError: string;
  edgeError: string;
  edgeTypeError: string;
  location: URL;
  query: URLSearchParams;
  routeParams: Record<string, string>;
  dispatch: AppDispatch;
}) => {
  const { edgeID } = routeParams;
  const cleanQuery = makeCleanPageLink(query);
  const isCreatePage = location.pathname.indexOf('create') > -1;

  useEffect(() => {
    if (selectedTenantID) {
      if (!objectTypes) {
        dispatch(
          fetchAuthZObjectTypes(selectedTenantID, new URLSearchParams())
        );
      }
    }
  }, [dispatch, objectTypes, selectedTenantID]);
  useEffect(() => {
    if (selectedTenantID) {
      if (!edgeTypes) {
        dispatch(fetchAuthZEdgeTypes(selectedTenantID, new URLSearchParams()));
      } else if (isCreatePage && (!edge || !edge.edge_type_id)) {
        dispatch({
          type: CHANGE_EDGE,
          data: { ...edge, edge_type_id: edgeTypes.data[0].id },
        });
      }
    }
  }, [dispatch, selectedTenantID, edge, edgeTypes, isCreatePage]);
  useEffect(() => {
    if (selectedTenantID) {
      if (edgeID) {
        dispatch(fetchAuthZEdge(selectedTenantID, edgeID));
      }
      if (isCreatePage) {
        dispatch({
          type: RETRIEVE_BLANK_EDGE,
        });
      }
    }
  }, [dispatch, selectedTenantID, edgeID, isCreatePage]);

  if (edgeError) {
    return <InlineNotification theme="alert">{edgeError}</InlineNotification>;
  }

  if (edgeTypeError) {
    return (
      <InlineNotification theme="alert">{edgeTypeError}</InlineNotification>
    );
  }

  return (
    <Card detailview>
      {edge && edgeTypes ? (
        <CardRow
          title="Basic Details"
          tooltip={
            <>
              {(isCreatePage ? 'Configure' : 'View') +
                ' the details of this edge.'}
            </>
          }
          collapsible
        >
          <>
            <div className={PageCommon.carddetailsrow}>
              <Label>
                ID
                <InputReadOnly monospace>{edge.id}</InputReadOnly>
              </Label>
              <Label htmlFor="edgeType">
                Edge Type
                <br />
                {isCreatePage ? (
                  <Select
                    value={edge.edge_type_id}
                    onChange={(e: React.ChangeEvent) => {
                      edge.edge_type_id = (e.target as HTMLSelectElement).value;
                      dispatch({
                        type: CHANGE_EDGE,
                        data: edge,
                      });
                    }}
                  >
                    {edgeTypes.data.map((et) => (
                      <option value={et.id} key={et.id}>
                        {et.type_name}
                      </option>
                    ))}
                  </Select>
                ) : (
                  <Link href={`/edgetypes/${edge.edge_type_id}${cleanQuery}`}>
                    {edge.edge_type_id}
                  </Link>
                )}
              </Label>
              <Label>
                Source Object ID
                <br />
                {isCreatePage ? (
                  <TextInput
                    id="source object id"
                    value={edge.source_object_id}
                    onChange={(e: React.ChangeEvent) => {
                      edge.source_object_id = (
                        e.target as HTMLInputElement
                      ).value;
                      dispatch({
                        type: CHANGE_EDGE,
                        data: edge,
                      });
                    }}
                  />
                ) : (
                  <Link href={`/objects/${edge.source_object_id}${cleanQuery}`}>
                    {edge.source_object_id}
                  </Link>
                )}
              </Label>
              <Label>
                Target Object ID
                <br />
                {isCreatePage ? (
                  <TextInput
                    id="target object id"
                    value={edge.target_object_id}
                    onChange={(e: React.ChangeEvent) => {
                      edge.target_object_id = (
                        e.target as HTMLInputElement
                      ).value;
                      dispatch({
                        type: CHANGE_EDGE,
                        data: edge,
                      });
                    }}
                  />
                ) : (
                  <Link href={`/objects/${edge.target_object_id}${cleanQuery}`}>
                    {edge.target_object_id}
                  </Link>
                )}
              </Label>

              {saveSuccess && (
                <InlineNotification theme="success">
                  {saveSuccess}
                </InlineNotification>
              )}
              {saveError && (
                <InlineNotification theme="alert">
                  {saveError}
                </InlineNotification>
              )}
            </div>
          </>
        </CardRow>
      ) : (
        <Text>Loading...</Text>
      )}
    </Card>
  );
};

const saveEdge = (tenantID: string, edge: Edge) => (dispatch: AppDispatch) => {
  dispatch({
    type: CREATE_EDGE_REQUEST,
  });
  createTenantAuthZEdge(tenantID, edge).then(
    (response: Edge) => {
      dispatch({
        type: CREATE_EDGE_SUCCESS,
        data: response,
      });
      dispatch(postSuccessToast('Successfully created edge'));
      redirect(`/${response.id}?tenant_id=${tenantID}`);
    },
    (error) => {
      dispatch({
        type: CREATE_EDGE_ERROR,
        data: error.message,
      });
    }
  );
};

const ConnectedCreateEdgePage = connect((state: RootState) => {
  return {
    edge: state.currentEdge,
    selectedTenantID: state.selectedTenantID,
    saveSuccess: state.saveEdgeSuccess,
    saveError: state.saveEdgeError,
    objectTypes: state.objectTypes,
    objectTypeError: state.fetchObjectTypesError,
    edgeTypes: state.edgeTypes,
    edgeTypeError: state.fetchEdgeTypesError,
    edgeError: state.fetchEdgeError,
    location: state.location,
    query: state.query,
    routeParams: state.routeParams,
  };
})(CreateEdgeEditPage);

const EdgePage = ({
  edge,
  objectTypes,
  edgeTypes,
  selectedTenant,
  location,
  query,
  dispatch,
}: {
  edge: Edge | undefined;
  objectTypes: PaginatedResult<ObjectType> | undefined;
  edgeTypes: PaginatedResult<EdgeType> | undefined;
  selectedTenant: SelectedTenant | undefined;
  location: URL;
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
    }
  }, [dispatch, objectTypes, selectedTenant]);
  useEffect(() => {
    if (selectedTenant) {
      if (!edgeTypes) {
        dispatch(fetchAuthZEdgeTypes(selectedTenant.id, new URLSearchParams()));
      }
    }
  }, [dispatch, edgeTypes, selectedTenant]);
  const { pathname } = location;
  const isCreatePage = pathname.indexOf('create') > -1;
  const name = (isCreatePage ? 'Create' : '') + ' Edge';
  const cleanQuery = makeCleanPageLink(query);

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <PageTitle
          title={name}
          itemName={isCreatePage ? 'New Edge' : edge && edge.id}
        />
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              Edges reflect real-world relationships between two objects in your
              authorization graph.
            </>
          </ToolTip>
        </div>
        {isCreatePage && (
          <ButtonGroup className={PageCommon.listviewtablecontrolsButtonGroup}>
            <Button
              theme="secondary"
              size="small"
              onClick={() => {
                redirect(`/edges?${cleanQuery}`);
              }}
            >
              Cancel
            </Button>

            <Button
              theme="primary"
              size="small"
              onClick={() => {
                edge &&
                  selectedTenant &&
                  dispatch(saveEdge(selectedTenant.id as string, edge));
              }}
            >
              Create Edge
            </Button>
          </ButtonGroup>
        )}
      </div>

      <ConnectedCreateEdgePage />
    </>
  );
};

export default connect((state: RootState) => {
  return {
    edge: state.currentEdge,
    objectTypes: state.objectTypes,
    edgeTypes: state.edgeTypes,
    selectedTenant: state.selectedTenant,
    location: state.location,
    query: state.query,
  };
})(EdgePage);
