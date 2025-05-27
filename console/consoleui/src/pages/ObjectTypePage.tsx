import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  EmptyState,
  IconButton,
  Heading,
  IconDeleteBin,
  IconLock2,
  InputReadOnly,
  Label,
  InlineNotification,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  Text,
  TextInput,
  TextShortener,
  ToolTip,
} from '@userclouds/ui-component-lib';

import { PageTitle } from '../mainlayout/PageWrap';
import { AppDispatch, RootState } from '../store';
import { redirect } from '../routing';
import { makeCleanPageLink } from '../AppNavigation';
import {
  fetchAuthZObjectTypes,
  fetchAuthZObjectType,
  fetchAuthZEdgeTypes,
  createObjectType,
} from '../thunks/authz';
import {
  retrieveBlankObjectType,
  changeObjectType,
  toggleEdgeTypeForDelete,
} from '../actions/authz';
import EdgeType, { getEdgeTypeFilteredByType } from '../models/authz/EdgeType';
import { ObjectType } from '../models/authz/ObjectType';
import { SelectedTenant } from '../models/Tenant';
import PaginatedResult from '../models/PaginatedResult';
import Link from '../controls/Link';
import ObjectTable from '../controls/ObjectTable';
import PageCommon from './PageCommon.module.css';
import styles from './ObjectTypePage.module.css';

const CreateObjectTypeEditPage = ({
  objectType,
  selectedTenantID,
  saveSuccess,
  saveError,
  routeParams,
  dispatch,
}: {
  objectType: ObjectType | undefined;
  selectedTenantID: string | undefined;
  saveSuccess: string;
  saveError: string;
  routeParams: Record<string, string>;
  dispatch: AppDispatch;
}) => {
  const { objectTypeID } = routeParams;
  const isCreatePage = window.location.pathname.indexOf('create') > -1;
  useEffect(() => {
    if (selectedTenantID) {
      if (objectTypeID) {
        dispatch(fetchAuthZObjectType(selectedTenantID, objectTypeID));
      } else {
        dispatch(retrieveBlankObjectType());
      }
    }
  }, [dispatch, selectedTenantID, objectTypeID]);

  return objectType ? (
    <CardRow
      title="Basic Details"
      tooltip={
        <>Configure the basic details and attributes of this object type.</>
      }
      collapsible
    >
      <>
        <div className={PageCommon.carddetailsrow}>
          <Label htmlFor="name">
            Name
            <br />
            {isCreatePage ? (
              <TextInput
                id="name"
                value={objectType.type_name}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                  dispatch(changeObjectType({ type_name: e.target.value }));
                }}
              />
            ) : (
              <InputReadOnly>{objectType.type_name}</InputReadOnly>
            )}
          </Label>
          <Label htmlFor="object_type_id">
            ID
            <br />
            <TextShortener
              text={objectType.id}
              length={6}
              id="object_type_id"
            />
          </Label>

          {saveSuccess && (
            <InlineNotification theme="success">
              {saveSuccess}
            </InlineNotification>
          )}
          {!saveSuccess && saveError && (
            <InlineNotification theme="alert">{saveError}</InlineNotification>
          )}
        </div>
      </>
    </CardRow>
  ) : (
    <Text>Fetching Object Type...</Text>
  );
};

const ConnectedCreateObjectTypePage = connect((state: RootState) => {
  return {
    objectType: state.currentObjectType,
    selectedTenantID: state.selectedTenantID,
    saveSuccess: state.saveObjectTypeSuccess,
    saveError: state.saveObjectTypeError,
    routeParams: state.routeParams,
  };
})(CreateObjectTypeEditPage);

const EdgeTypesRow = ({
  edgeType,
  objectTypes,
  edgeTypeEditMode,
  edgeTypeDeleteQueue,
  query,
  dispatch,
}: {
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
        <Link
          key={edgeType.id}
          href={`/edgetypes/${edgeType.id}` + makeCleanPageLink(query)}
        >
          {edgeType.type_name}
        </Link>
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
  etError,
  edgeTypeEditMode,
  edgeTypeDeleteQueue,
  targetObjectTypeID,
  sourceObjectTypeID,
  query,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  objectTypes: PaginatedResult<ObjectType> | undefined;
  objectTypeError: string;
  edgeTypes: PaginatedResult<EdgeType> | undefined;
  etError: string;
  edgeTypeEditMode: boolean;
  edgeTypeDeleteQueue: string[];
  targetObjectTypeID: string | undefined;
  sourceObjectTypeID: string | undefined;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  return (
    <CardRow
      title={
        'Edge Types with this ' + (sourceObjectTypeID ? 'Source' : 'Target')
      }
      tooltip={
        <>
          {'View all edge types with this object type as their ' +
            (sourceObjectTypeID ? 'Source' : 'Target') +
            '. '}
          <a
            href="https://docs.userclouds.com/docs/key-concepts-1#edge-types"
            title="UserClouds documentation for key concepts in authorization"
          >
            Learn more about edge types here.
          </a>
        </>
      }
      collapsible
    >
      {objectTypeError || etError ? (
        <Text element="h4">{objectTypeError || etError}</Text>
      ) : !objectTypes || !edgeTypes ? (
        <Text element="h4">Loading...</Text>
      ) : !edgeTypes ||
        !edgeTypes.data ||
        getEdgeTypeFilteredByType(
          edgeTypes,
          targetObjectTypeID,
          sourceObjectTypeID
        ).length === 0 ? (
        <CardRow>
          <EmptyState
            title={`No edges with this ${sourceObjectTypeID ? 'source' : 'target'}`}
            image={<IconLock2 size="large" />}
          >
            {selectedTenant?.is_admin && (
              <Link
                href={'/edgetypes/create' + makeCleanPageLink(query)}
                applyStyles={false}
              >
                <Button theme="secondary">Create Edge Type</Button>
              </Link>
            )}
          </EmptyState>
        </CardRow>
      ) : (
        <Table
          className={styles.objecttypeedgetypestable}
          id={`edgeTypesWithThis${sourceObjectTypeID ? 'Source' : 'Target'}`}
        >
          <TableHead>
            <TableRow>
              <TableRowHead>Type Name</TableRowHead>
              <TableRowHead>Source Object Type</TableRowHead>
              <TableRowHead>Target Object Type</TableRowHead>
              <TableRowHead>ID</TableRowHead>
              {edgeTypeEditMode && <TableRowHead key="delete_head" />}
            </TableRow>
          </TableHead>
          <TableBody>
            {edgeTypes?.data &&
              getEdgeTypeFilteredByType(
                edgeTypes,
                targetObjectTypeID,
                sourceObjectTypeID
              ).map((et) => (
                <EdgeTypesRow
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
      )}
    </CardRow>
  );
};
const ConnectedEdgeTypes = connect((state: RootState) => {
  return {
    edgeType: state.edgeTypes,
    eError: state.fetchEdgeTypesError,
    edgeTypeEditMode: state.edgeTypeEditMode,
    editSuccess: state.editObjectsSuccess,
    editError: state.editObjectsError,
    edgeTypeDeleteQueue: state.edgeTypeDeleteQueue,
    savingEdgeTypes: state.savingEdgeTypes,
    query: state.query,
  };
})(EdgeTypes);

const ObjectTypePage = ({
  objectTypes,
  objectTypeError,
  edgeTypes,
  edgeTypeError,
  selectedTenant,
  location,
  query,
  routeParams,
  objectType,
  dispatch,
}: {
  objectTypes: PaginatedResult<ObjectType> | undefined;
  objectTypeError: string;
  edgeTypes: PaginatedResult<EdgeType> | undefined;
  edgeTypeError: string;
  selectedTenant: SelectedTenant | undefined;
  location: URL;
  query: URLSearchParams;
  routeParams: Record<string, string>;
  objectType: ObjectType | undefined;
  dispatch: AppDispatch;
}) => {
  const { pathname } = location;
  const { objectTypeID } = routeParams;
  useEffect(() => {
    if (selectedTenant) {
      dispatch(fetchAuthZObjectTypes(selectedTenant.id, query));
      dispatch(fetchAuthZEdgeTypes(selectedTenant.id, new URLSearchParams()));
    }
  }, [dispatch, selectedTenant, query]);
  const isCreatePage = pathname.indexOf('create') > -1;
  const cleanQuery = makeCleanPageLink(query);

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <PageTitle
          title={isCreatePage ? 'Create Object Type' : 'Object Type'}
          itemName={
            isCreatePage
              ? 'New Object Type'
              : objectType && objectType.type_name
          }
        />
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              Object types reflect possible categories of object in your
              authorization graph. Every object has exactly one object type.
            </>
          </ToolTip>
        </div>
        {isCreatePage && (
          <ButtonGroup className={PageCommon.listviewtablecontrolsButtonGroup}>
            <Button
              theme="secondary"
              size="small"
              onClick={() => {
                redirect(`/objecttypes?${cleanQuery}`);
              }}
            >
              Cancel
            </Button>
            {isCreatePage && (
              <Button
                theme="primary"
                size="small"
                disabled={!(objectType && objectType.type_name)}
                onClick={() => {
                  selectedTenant &&
                    objectType &&
                    dispatch(
                      createObjectType(selectedTenant.id as string, objectType)
                    );
                }}
              >
                Create Object Type
              </Button>
            )}
          </ButtonGroup>
        )}
      </div>
      <Card detailview>
        <ConnectedCreateObjectTypePage />
        {!isCreatePage && (
          <ConnectedEdgeTypes
            sourceObjectTypeID={objectTypeID}
            targetObjectTypeID={undefined}
            selectedTenant={selectedTenant}
            objectTypes={objectTypes}
            objectTypeError={objectTypeError}
            edgeTypes={edgeTypes}
            etError={edgeTypeError}
          />
        )}
        {!isCreatePage && (
          <ConnectedEdgeTypes
            sourceObjectTypeID={undefined}
            targetObjectTypeID={objectTypeID}
            selectedTenant={selectedTenant}
            objectTypes={objectTypes}
            objectTypeError={objectTypeError}
            edgeTypes={edgeTypes}
            etError={edgeTypeError}
          />
        )}
        {!isCreatePage && (
          <CardRow
            title="Objects"
            tooltip={<>View all objects of this particular object type.</>}
            collapsible
          >
            <ObjectTable
              selectedTenant={selectedTenant}
              objectTypes={objectTypes}
              objectTypeError={objectTypeError}
              objectTypeID={objectTypeID}
              detailLayout
            />
          </CardRow>
        )}
      </Card>
    </>
  );
};

export default connect((state: RootState) => {
  return {
    objectTypes: state.objectTypes,
    objectTypeError: state.fetchObjectTypesError,
    edgeTypes: state.edgeTypes,
    edgeTypeError: state.fetchEdgeTypesError,
    selectedTenant: state.selectedTenant,
    location: state.location,
    query: state.query,
    routeParams: state.routeParams,
    objectType: state.currentObjectType,
  };
})(ObjectTypePage);
