import React, { useEffect } from 'react';
import { connect, useDispatch } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  IconButton,
  IconDeleteBin,
  InputReadOnly,
  Label,
  Select,
  InlineNotification,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  Text,
  TextInput,
  ToolTip,
  TextShortener,
  GlobalStyles,
} from '@userclouds/ui-component-lib';

import { AppDispatch, RootState } from '../store';

import { SelectedTenant } from '../models/Tenant';
import { ObjectType } from '../models/authz/ObjectType';
import EdgeType, {
  Attribute,
  AttributeFlavors,
} from '../models/authz/EdgeType';
import PaginatedResult from '../models/PaginatedResult';
import {
  fetchAuthZObjectTypes,
  fetchAuthZEdgeTypes,
  fetchAuthZEdgeType,
  createEdgeType,
  updateEdgeType,
} from '../thunks/authz';
import {
  changeAttributeFlavorForEdgeType,
  addRowToEdgeType,
  deleteAttributeForEdgeType,
  changeEdgeTypeAttribute,
  changeEdgeType,
  retrieveBlankEdgeType,
  toggleEdgeTypeEditMode,
} from '../actions/authz';
import { redirect } from '../routing';
import { makeCleanPageLink } from '../AppNavigation';
import { PageTitle } from '../mainlayout/PageWrap';
import PageCommon from './PageCommon.module.css';
import styles from './EdgeTypePage.module.css';

const AttributeRow = ({
  attribute,
  edgeType,
  readOnly,
  index,
}: {
  attribute: Attribute;
  edgeType: EdgeType;
  readOnly: boolean;
  index: number;
}) => {
  let flavor = '';
  if (attribute.direct) {
    flavor = AttributeFlavors.direct;
  } else if (attribute.propagate) {
    flavor = AttributeFlavors.propagate;
  } else {
    flavor = AttributeFlavors.inherit;
  }

  const dispatch: AppDispatch = useDispatch();
  return (
    <TableRow key={attribute.id}>
      <TableCell>
        {readOnly ? (
          <InputReadOnly>{attribute.name}</InputReadOnly>
        ) : (
          <Label htmlFor={`attributeName` + index}>
            <TextInput
              id={`attributeName` + index}
              value={attribute.name}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                attribute.name = e.target.value;
                dispatch(changeEdgeTypeAttribute(attribute));
              }}
            />
            {attribute.name}
          </Label>
        )}
      </TableCell>
      <TableCell>
        {readOnly ? (
          <InputReadOnly>{flavor}</InputReadOnly>
        ) : (
          <Label htmlFor={`attributeFlavor` + index}>
            <Select
              id={`attributeFlavor` + index}
              name={'attribute_flavor_' + index}
              items={[
                AttributeFlavors.direct,
                AttributeFlavors.inherit,
                AttributeFlavors.propagate,
              ]}
              value={flavor}
              onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
                dispatch(
                  changeAttributeFlavorForEdgeType(attribute, e.target.value)
                );
              }}
            >
              {[
                AttributeFlavors.direct,
                AttributeFlavors.inherit,
                AttributeFlavors.propagate,
              ].map((t) => (
                <option key={t + attribute.name} value={t}>
                  {t}
                </option>
              ))}
            </Select>
          </Label>
        )}
      </TableCell>
      <TableCell>
        {!readOnly && (
          <>
            <IconButton
              id={`deleteAttribute` + index}
              icon={<IconDeleteBin />}
              onClick={() => {
                dispatch(deleteAttributeForEdgeType(attribute, edgeType));
              }}
              title="Delete Field"
              aria-label="Delete Field"
            />
            <Label
              htmlFor={`deleteAttribute` + index}
              className={GlobalStyles.visuallyHidden}
            >
              Delete Field
            </Label>
          </>
        )}
      </TableCell>
    </TableRow>
  );
};

const AttributesTable = ({
  edgeType,
  readOnly,
}: {
  edgeType: EdgeType;
  readOnly: boolean;
}) => {
  return (
    <Table id="attributes" className={styles.edgetypeattributestable}>
      <TableHead>
        <TableRow>
          <TableRowHead key="attribute_name_header">
            Attribute Name
          </TableRowHead>
          <TableRowHead key="attribute_flavor_header">Flavor</TableRowHead>
          <TableRowHead key="attribute_delete_header" />
        </TableRow>
      </TableHead>
      <TableBody>
        {edgeType.attributes &&
          edgeType.attributes.map((attribute, i) => (
            <AttributeRow
              attribute={attribute}
              edgeType={edgeType}
              readOnly={readOnly}
              key={attribute.id}
              index={i}
            />
          ))}
      </TableBody>
    </Table>
  );
};

const EdgeTypeEditPage = ({
  objectTypes,
  edgeType,
  edgeTypes,
  selectedTenant,
  saveSuccess,
  saveError,
  editMode,
  isNewPage,
  edgeTypeID,
  dispatch,
}: {
  objectTypes: PaginatedResult<ObjectType> | undefined;
  edgeType: EdgeType | undefined;
  edgeTypes: PaginatedResult<EdgeType> | undefined;
  selectedTenant: SelectedTenant | undefined;
  saveSuccess: string;
  saveError: string;
  editMode: boolean;
  isNewPage: boolean;
  edgeTypeID: string | undefined;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenant) {
      if (!objectTypes) {
        dispatch(
          fetchAuthZObjectTypes(selectedTenant.id, new URLSearchParams())
        );
      }
      if (!edgeTypes) {
        dispatch(fetchAuthZEdgeTypes(selectedTenant.id, new URLSearchParams()));
      }
    }
  }, [dispatch, objectTypes, edgeTypes, selectedTenant]);

  useEffect(() => {
    if (selectedTenant) {
      if (edgeTypeID) {
        dispatch(fetchAuthZEdgeType(selectedTenant.id, edgeTypeID));
      } else if (objectTypes) {
        dispatch(retrieveBlankEdgeType());
      }
    }
  }, [dispatch, selectedTenant, edgeTypeID, objectTypes]);

  const readOnly = !selectedTenant?.is_admin || !editMode;

  return edgeType ? (
    <Card detailview>
      <CardRow
        title="Basic Details"
        tooltip={<>Configure the basic details of this edge type.</>}
        collapsible
      >
        <>
          <div className={PageCommon.carddetailsrow}>
            <Label htmlFor="name">
              Name
              <br />
              {readOnly ? (
                <InputReadOnly>{edgeType.type_name}</InputReadOnly>
              ) : (
                <TextInput
                  id="name"
                  value={edgeType.type_name}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    dispatch(changeEdgeType({ type_name: e.target.value }));
                  }}
                />
              )}
            </Label>

            <Label htmlFor="id">
              ID
              <br />
              <TextShortener text={edgeType.id} length={6} />
            </Label>

            <Label htmlFor="sourceObjectType">
              Source Object Type
              <br />
              {!objectTypes ? (
                <Text element="h4">Loading...</Text>
              ) : (
                <>
                  {readOnly || !isNewPage ? (
                    <InputReadOnly>
                      {
                        objectTypes?.data?.find(
                          (t) => t.id === edgeType.source_object_type_id
                        )?.type_name
                      }
                    </InputReadOnly>
                  ) : (
                    objectTypes.data.length && (
                      <Select
                        id="sourceObjectType"
                        name="source_object_type"
                        items={objectTypes}
                        value={
                          objectTypes?.data?.find(
                            (t) => t.id === edgeType.source_object_type_id
                          )?.type_name
                        }
                        onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
                          objectTypes?.data &&
                            dispatch(
                              changeEdgeType({
                                source_object_type_id: objectTypes.data.find(
                                  (t) => t.type_name === e.target.value
                                )?.id!,
                              })
                            );
                        }}
                      >
                        {objectTypes.data.map((t) => (
                          <option key={t.id} value={t.type_name}>
                            {t.type_name}
                          </option>
                        ))}
                      </Select>
                    )
                  )}
                </>
              )}
            </Label>
            <Label htmlFor="targetObjectType">
              Target Object Type
              <br />
              {!objectTypes ? (
                <Text element="h4">Loading...</Text>
              ) : (
                <>
                  {readOnly || !isNewPage ? (
                    <InputReadOnly>
                      {
                        objectTypes?.data?.find(
                          (t) => t.id === edgeType.target_object_type_id
                        )?.type_name
                      }
                    </InputReadOnly>
                  ) : (
                    objectTypes?.data?.length && (
                      <Select
                        id="targetObjectType"
                        name="target_object_type"
                        items={objectTypes}
                        value={
                          objectTypes?.data?.find(
                            (t) => t.id === edgeType.target_object_type_id
                          )?.type_name
                        }
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                          objectTypes?.data &&
                            dispatch(
                              changeEdgeType({
                                target_object_type_id: objectTypes.data.find(
                                  (t) => t.type_name === e.target.value
                                )?.id!,
                              })
                            );
                        }}
                      >
                        {objectTypes.data.map((t) => (
                          <option key={t.id} value={t.type_name}>
                            {t.type_name}
                          </option>
                        ))}
                      </Select>
                    )
                  )}
                </>
              )}
            </Label>
          </div>
        </>
      </CardRow>

      <CardRow
        title="Attributes"
        tooltip={<>Configure the attributes of this edge type.</>}
        collapsible
      >
        {saveSuccess && (
          <InlineNotification theme="success">{saveSuccess}</InlineNotification>
        )}
        {saveError && (
          <InlineNotification theme="alert">{saveError}</InlineNotification>
        )}

        <AttributesTable edgeType={edgeType} readOnly={readOnly} />
        {selectedTenant?.is_admin && editMode && (
          <ButtonGroup>
            <Button
              theme="secondary"
              onClick={() => {
                dispatch(addRowToEdgeType());
              }}
            >
              Add Attribute
            </Button>
          </ButtonGroup>
        )}
      </CardRow>
    </Card>
  ) : (
    <Text>Fetching Edge Type...</Text>
  );
};

const EdgeTypePage = ({
  objectTypes,
  edgeType,
  edgeTypes,
  selectedTenant,
  saveSuccess,
  saveError,
  editMode,
  location,
  query,
  routeParams,
  dispatch,
}: {
  objectTypes: PaginatedResult<ObjectType> | undefined;
  edgeType: EdgeType | undefined;
  edgeTypes: PaginatedResult<EdgeType> | undefined;
  selectedTenant: SelectedTenant | undefined;
  saveSuccess: string;
  saveError: string;
  editMode: boolean;
  location: URL;
  query: URLSearchParams;
  routeParams: Record<string, string>;
  dispatch: AppDispatch;
}) => {
  const { pathname } = location;
  const { edgeTypeID } = routeParams;
  const isNewPage = pathname.indexOf('create') > -1;
  const name = (isNewPage ? 'Create' : 'Edit') + ' Edge Type';
  const cleanQuery = makeCleanPageLink(query);

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <PageTitle
          title={name}
          itemName={isNewPage ? 'New Edge' : edgeType?.type_name}
        />
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              Edge Types reflect possible relationships between different types
              of objects in your authorization graph.
            </>
          </ToolTip>
        </div>

        {isNewPage ? (
          <ButtonGroup>
            <Button
              theme="secondary"
              size="small"
              onClick={() => {
                redirect(`/edgetypes?${cleanQuery}`);
              }}
            >
              Cancel
            </Button>
            <Button
              theme="primary"
              size="small"
              disabled={edgeType && edgeType.type_name === ''}
              onClick={() => {
                edgeType &&
                  dispatch(createEdgeType(selectedTenant?.id || '', edgeType));
              }}
            >
              Create Edge Type
            </Button>
          </ButtonGroup>
        ) : editMode ? (
          <ButtonGroup>
            <Button
              theme="secondary"
              size="small"
              onClick={() => {
                dispatch(toggleEdgeTypeEditMode(false));
                if (edgeTypeID) {
                  selectedTenant &&
                    dispatch(fetchAuthZEdgeType(selectedTenant.id, edgeTypeID));
                }
              }}
            >
              Cancel
            </Button>

            <Button
              theme="primary"
              size="small"
              disabled={edgeType && edgeType.type_name === ''}
              onClick={() => {
                edgeType &&
                  dispatch(updateEdgeType(selectedTenant?.id || '', edgeType));
              }}
            >
              Save Edge Type
            </Button>
          </ButtonGroup>
        ) : (
          <Button
            theme="primary"
            onClick={() => {
              dispatch(toggleEdgeTypeEditMode(true));
            }}
          >
            Edit Edge Type
          </Button>
        )}
      </div>

      <EdgeTypeEditPage
        edgeTypeID={edgeTypeID}
        isNewPage={isNewPage}
        objectTypes={objectTypes}
        edgeType={edgeType}
        edgeTypes={edgeTypes}
        selectedTenant={selectedTenant}
        saveSuccess={saveSuccess}
        saveError={saveError}
        editMode={editMode}
        dispatch={dispatch}
      />
    </>
  );
};

export default connect((state: RootState) => {
  return {
    objectTypes: state.objectTypes,
    objectTypeError: state.fetchObjectTypesError,
    edgeType: state.currentEdgeType,
    edgeTypes: state.edgeTypes,
    selectedTenant: state.selectedTenant,
    saveSuccess: state.saveEdgeTypeSuccess,
    saveError: state.saveEdgeTypeError,
    editMode: state.edgeTypeEditMode,
    location: state.location,
    query: state.query,
    routeParams: state.routeParams,
  };
})(EdgeTypePage);
