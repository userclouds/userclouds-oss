import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  EmptyState,
  Heading,
  IconLock2,
  InputReadOnly,
  Label,
  InlineNotification,
  Select,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  TextInput,
  ToolTip,
  TextShortener,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import { createTenantAuthZObject } from '../API/authzAPI';
import { AppDispatch, RootState } from '../store';
import UCObject from '../models/authz/Object';
import Edge from '../models/authz/Edge';
import EdgeType from '../models/authz/EdgeType';
import { ObjectType } from '../models/authz/ObjectType';
import {
  CREATE_OBJECT_REQUEST,
  CREATE_OBJECT_SUCCESS,
  CREATE_OBJECT_ERROR,
  RETRIEVE_BLANK_OBJECT,
  CHANGE_OBJECT,
} from '../actions/authz';
import {
  fetchAuthZObjectTypes,
  fetchAuthZEdgeTypes,
  fetchAuthZObject,
  fetchAuthZEdgesForObject,
} from '../thunks/authz';
import Link from '../controls/Link';
import { PageTitle } from '../mainlayout/PageWrap';
import PageCommon from './PageCommon.module.css';
import styles from './ObjectPage.module.css';
import { redirect } from '../routing';
import PaginatedResult from '../models/PaginatedResult';
import AuthZAuthorizationCheck from '../controls/AuthorizationCheck';
import { SelectedTenant } from '../models/Tenant';

const AuthZEdgesOnObject = ({
  objectTypes,
  edgeTypes,
  object,
  edges,
  otherObjects,
  source,
  query,
  routeParams,
  selectedTenant,
}: {
  objectTypes: PaginatedResult<ObjectType> | undefined;
  edgeTypes: PaginatedResult<EdgeType> | undefined;
  object: UCObject | undefined;
  edges: Edge[] | undefined;
  otherObjects: UCObject[] | undefined;
  source: boolean;
  query: URLSearchParams;
  routeParams: Record<string, string>;
  selectedTenant: SelectedTenant | undefined;
}) => {
  const cleanQuery = makeCleanPageLink(query);
  const { objectID } = routeParams;
  if (
    !objectTypes ||
    !edgeTypes ||
    !object ||
    !edges ||
    !otherObjects ||
    !objectID
  ) {
    return <></>;
  }

  const rows: {
    edge: Edge;
    edgeType: EdgeType | undefined;
    sourceObject: string;
    sourceObjectType: ObjectType | undefined;
    sourceType: string;
    sourceAlias: string;
    targetObject: string;
    targetObjectType: ObjectType | undefined;
    targetType: string;
    targetAlias: string;
  }[] = [];
  objectTypes?.data &&
    edgeTypes?.data &&
    edges?.forEach((edge, i) => {
      const edgeType = edgeTypes.data.find((et) => et.id === edge.edge_type_id);
      const sourceObject =
        edge.source_object_id === objectID ? object : otherObjects[i];
      const targetObject =
        edge.target_object_id === objectID ? object : otherObjects[i];
      const sourceObjectType =
        edgeType &&
        objectTypes.data.find((ot) => ot.id === edgeType.source_object_type_id);
      const targetObjectType =
        edgeType &&
        objectTypes.data.find((ot) => ot.id === edgeType.target_object_type_id);
      const sourceAlias = sourceObject
        ? sourceObject.alias
        : edge.source_object_id;
      const targetAlias = targetObject
        ? targetObject.alias
        : edge.target_object_id;
      if (!source && object && targetObject && object.id === targetObject.id) {
        rows.push({
          edge: edge,
          edgeType: edgeType,
          sourceObject: sourceObject ? sourceObject.id : '',
          sourceObjectType: sourceObjectType,
          sourceType: sourceObjectType ? sourceObjectType.type_name : '',
          sourceAlias: sourceAlias,
          targetObject: targetObject ? targetObject.id : '',
          targetObjectType: targetObjectType,
          targetType: targetObjectType ? targetObjectType.type_name : '',
          targetAlias: targetAlias,
        });
      } else if (
        source &&
        object &&
        sourceObject &&
        object.id === sourceObject.id
      ) {
        rows.push({
          edge: edge,
          edgeType: edgeType,
          sourceObject: sourceObject ? sourceObject.id : '',
          sourceObjectType: sourceObjectType,
          sourceType: sourceObjectType ? sourceObjectType.type_name : '',
          sourceAlias: sourceAlias,
          targetObject: targetObject ? targetObject.id : '',
          targetObjectType: targetObjectType,
          targetType: targetObjectType ? targetObjectType.type_name : '',
          targetAlias: targetAlias,
        });
      }
    });
  return (
    <CardRow
      title={'Edges with this ' + (source ? 'source' : 'target')}
      tooltip={
        <>
          {'View edges that connect ' +
            (source
              ? 'this object to other objects.'
              : 'other objects to this object.')}
        </>
      }
      collapsible
    >
      <Table className={styles.objectedgetable}>
        <TableHead>
          <TableRow>
            <TableRowHead>ID</TableRowHead>
            <TableRowHead>Edge Type</TableRowHead>
            {!source && <TableRowHead>Source Object</TableRowHead>}
            {!source && <TableRowHead>Source Type</TableRowHead>}
            {source && <TableRowHead>Target Object</TableRowHead>}
            {source && <TableRowHead>Target Type</TableRowHead>}
          </TableRow>
        </TableHead>
        <TableBody>
          {rows.length > 0 ? (
            rows.map((row) => (
              <TableRow key={row.edge.id}>
                <TableCell>
                  {' '}
                  <Link href={`/edges/${row.edge.id}${cleanQuery}`}>
                    {row.edge.id}
                  </Link>
                </TableCell>
                <TableCell>
                  {row.edgeType && (
                    <Link href={`/edgetypes/${row.edgeType.id}${cleanQuery}`}>
                      {row.edgeType.type_name}
                    </Link>
                  )}
                </TableCell>
                {!source && (
                  <TableCell>
                    <Link
                      href={`/objects/${row.edge.source_object_id}${cleanQuery}`}
                    >
                      {row.sourceAlias}
                    </Link>
                  </TableCell>
                )}
                {!source && (
                  <TableCell>
                    {row.sourceObjectType && (
                      <Link
                        href={`/objecttypes/${row.sourceObjectType.id}${cleanQuery}`}
                      >
                        {row.sourceType}
                      </Link>
                    )}
                  </TableCell>
                )}
                {source && (
                  <TableCell>
                    <Link
                      href={`/objects/${row.edge.target_object_id}${cleanQuery}`}
                    >
                      {row.targetAlias}
                    </Link>
                  </TableCell>
                )}
                {source && (
                  <TableCell>
                    {row.targetObjectType && (
                      <Link
                        href={`/objecttypes/${row.targetObjectType.id}${cleanQuery}`}
                      >
                        {row.targetType}
                      </Link>
                    )}
                  </TableCell>
                )}
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell colSpan={4}>
                <EmptyState title="No edges" image={<IconLock2 size="large" />}>
                  {selectedTenant?.is_admin && (
                    <Link href={`/edges/create` + makeCleanPageLink(query)}>
                      <Button theme="secondary">Create Edge</Button>
                    </Link>
                  )}
                </EmptyState>
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </CardRow>
  );
};

const ConnectedAuthZEdgesOnObject = connect((state: RootState) => {
  return {
    selectedTenant: state.selectedTenant,
    objectTypes: state.objectTypes,
    edgeTypes: state.edgeTypes,
    object: state.currentObject,
    edges: state.edgesForObject,
    otherObjects: state.objectsForEdgesForObject,
    query: state.query,
    routeParams: state.routeParams,
  };
})(AuthZEdgesOnObject);

const saveObject =
  (tenantID: string, object: UCObject) => (dispatch: AppDispatch) => {
    dispatch({
      type: CREATE_OBJECT_REQUEST,
    });
    createTenantAuthZObject(tenantID, object).then(
      (response: UCObject) => {
        dispatch({
          type: CREATE_OBJECT_SUCCESS,
          data: response,
        });
        redirect(`/objects/${response.id}?tenant_id=${tenantID}`);
      },
      (error) => {
        dispatch({
          type: CREATE_OBJECT_ERROR,
          data: error.message,
        });
      }
    );
  };

const AuthZObjectDetail = ({
  selectedTenantID,
  objectTypes,
  otError,
  edgeTypes,
  etError,
  object,
  oError,
  eError,
  saveSuccess,
  saveError,
  location,
  routeParams,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  objectTypes: PaginatedResult<ObjectType> | undefined;
  otError: string;
  edgeTypes: PaginatedResult<EdgeType> | undefined;
  etError: string;
  object: UCObject | undefined;
  oError: string;
  eError: string;
  saveSuccess: string;
  saveError: string;
  location: URL;
  routeParams: Record<string, string>;
  dispatch: AppDispatch;
}) => {
  const { pathname } = location;
  const { objectID } = routeParams;
  const isCreatePage = pathname.indexOf('create') > -1;

  useEffect(() => {
    if (selectedTenantID) {
      if (!objectTypes) {
        dispatch(
          fetchAuthZObjectTypes(selectedTenantID, new URLSearchParams())
        );
      }
      if (!edgeTypes) {
        dispatch(fetchAuthZEdgeTypes(selectedTenantID, new URLSearchParams()));
      }
      if (objectID) {
        dispatch(fetchAuthZObject(selectedTenantID, objectID));
        dispatch(fetchAuthZEdgesForObject(selectedTenantID, objectID));
      }
      if (isCreatePage) {
        dispatch({
          type: RETRIEVE_BLANK_OBJECT,
        });
      }
    }
  }, [
    selectedTenantID,
    objectTypes,
    edgeTypes,
    isCreatePage,
    objectID,
    dispatch,
  ]);

  if (otError || etError || oError || eError) {
    return (
      <InlineNotification theme="alert">
        {otError || etError || oError || eError}
      </InlineNotification>
    );
  }

  if (!objectTypes || !objectTypes.data || !edgeTypes || !object) {
    return <Heading>Loading...</Heading>;
  }

  const matchingType = objectTypes.data.find((ot) => ot.id === object.type_id);

  const alias = object.alias || object.id;
  return (
    <CardRow
      title="Object Details"
      tooltip={
        <>
          {(isCreatePage ? 'Set' : 'View') +
            ' the configuration of this object.'}
        </>
      }
      collapsible
    >
      <>
        <div className={PageCommon.carddetailsrow}>
          <Label>
            Name
            <br />
            {isCreatePage ? (
              <TextInput
                id="name"
                value={object.alias}
                onChange={(e: React.ChangeEvent) => {
                  object.alias = (e.target as HTMLInputElement).value;
                  dispatch({
                    type: CHANGE_OBJECT,
                    data: object,
                  });
                }}
              />
            ) : (
              <InputReadOnly>{alias}</InputReadOnly>
            )}
          </Label>

          <Label>
            Object Type
            <br />
            {isCreatePage ? (
              <Select
                value={object.type_id}
                onChange={(e: React.ChangeEvent) => {
                  object.type_id = (e.target as HTMLSelectElement).value;
                  dispatch({
                    type: CHANGE_OBJECT,
                    data: object,
                  });
                }}
              >
                {objectTypes.data.map((ot) => (
                  <option value={ot.id} key={ot.id}>
                    {ot.type_name}
                  </option>
                ))}
              </Select>
            ) : (
              <InputReadOnly>
                {matchingType ? matchingType.type_name : object.type_id}
              </InputReadOnly>
            )}
          </Label>

          <Label htmlFor="object_id">
            ID
            <TextShortener text={object.id} length={6} id="object_id" />
          </Label>
        </div>

        {saveSuccess && (
          <InlineNotification theme="success">{saveSuccess}</InlineNotification>
        )}
        {saveError && (
          <InlineNotification theme="alert">{saveError}</InlineNotification>
        )}
      </>
    </CardRow>
  );
};

const ConnectedAuthZObjectDetail = connect((state: RootState) => {
  return {
    selectedTenantID: state.selectedTenantID,
    objectTypes: state.objectTypes,
    otError: state.fetchObjectTypesError,
    edgeTypes: state.edgeTypes,
    etError: state.fetchEdgeTypesError,
    object: state.currentObject,
    oError: state.fetchObjectError,
    eError: state.fetchEdgesForObjectError,
    saveSuccess: state.saveObjectSuccess,
    saveError: state.saveObjectError,
    location: state.location,
    routeParams: state.routeParams,
  };
})(AuthZObjectDetail);

const ObjectPage = ({
  location,
  query,
  object,
  selectedTenantID,
  dispatch,
}: {
  location: URL;
  query: URLSearchParams;
  object: UCObject | undefined;
  selectedTenantID: string | undefined;
  dispatch: AppDispatch;
}) => {
  const { pathname } = location;
  const isCreatePage = pathname.indexOf('create') > -1;
  const cleanQuery = makeCleanPageLink(query);

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <PageTitle
          title={isCreatePage ? 'Create an Object' : 'Object Detail Page'}
          itemName={
            isCreatePage
              ? 'New Object'
              : object && (object.alias ? object.alias : object.id)
          }
        />
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              Objects reflect real-world users, groups and resources in your
              authorization system.
            </>
          </ToolTip>
        </div>
        {isCreatePage && (
          <ButtonGroup className={PageCommon.listviewtablecontrolsButtonGroup}>
            <Button
              theme="secondary"
              size="small"
              onClick={() => {
                redirect(`/objects?${cleanQuery}`);
              }}
            >
              Cancel
            </Button>
            <Button
              theme="primary"
              size="small"
              onClick={() => {
                object &&
                  dispatch(saveObject(selectedTenantID as string, object));
              }}
            >
              Create Object
            </Button>
          </ButtonGroup>
        )}
      </div>
      <Card detailview>
        <ConnectedAuthZObjectDetail />
        <br />
        {!isCreatePage && (
          <>
            <ConnectedAuthZEdgesOnObject source />
            <br />
            <ConnectedAuthZEdgesOnObject source={false} />
          </>
        )}
        <div />

        <AuthZAuthorizationCheck
          selectedTenantID={selectedTenantID}
          sourceObjectID={object?.id}
        />
      </Card>
    </>
  );
};

export default connect((state: RootState) => ({
  location: state.location,
  query: state.query,
  object: state.currentObject,
  selectedTenantID: state.selectedTenantID,
}))(ObjectPage);
