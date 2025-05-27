import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  Checkbox,
  IconCheckmark,
  IconDash,
  InputReadOnly,
  Label,
  Select,
  InlineNotification,
  Table,
  TableHead,
  TableBody,
  TableCell,
  TableRow,
  TableRowHead,
  Text,
  TextArea,
  TextInput,
} from '@userclouds/ui-component-lib';

import PaginatedResult from '../models/PaginatedResult';
import {
  CompositeField,
  DataType,
  isValidDataType,
  validFieldName,
} from '../models/DataType';
import { PageTitle } from '../mainlayout/PageWrap';
import { RootState, AppDispatch } from '../store';
import {
  fetchDataType,
  fetchDataTypes,
  updateDataType,
} from '../thunks/userstore';
import {
  addFieldToDataType,
  modifyDataType,
  toggleDataTypeEditMode,
} from '../actions/userstore';

import PageCommon from './PageCommon.module.css';
import styles from './DataTypeDetailPage.module.css';

const DataTypeDetailPage = ({
  selectedTenantID,
  modifiedDataType,
  isDirty,
  dataTypes,
  editMode,
  isSaving,
  saveError,
  routeParams,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  modifiedDataType: DataType | undefined;
  dataTypes: PaginatedResult<DataType> | undefined;
  isDirty: boolean;
  editMode: boolean;
  isSaving: boolean;
  saveError: string;
  routeParams: Record<string, string>;
  dispatch: AppDispatch;
}) => {
  const { datatypeID } = routeParams;

  useEffect(() => {
    if (selectedTenantID) {
      dispatch(fetchDataTypes(selectedTenantID, new URLSearchParams()));
      dispatch(fetchDataType(selectedTenantID, datatypeID));
    }
  }, [selectedTenantID, datatypeID, dispatch]);

  if (!modifiedDataType) {
    return (
      <InlineNotification theme="alert">
        Unable to fetch data type
      </InlineNotification>
    );
  }

  return (
    <form
      onSubmit={(e: React.FormEvent) => {
        e.preventDefault();
        if (selectedTenantID) {
          dispatch(updateDataType(selectedTenantID, modifiedDataType));
        }
      }}
    >
      <div className={PageCommon.listviewtablecontrols}>
        <PageTitle
          title={(editMode ? 'Edit' : 'View') + ' Data Type'}
          itemName={modifiedDataType.name}
        />

        {!modifiedDataType.is_native && (
          <ButtonGroup className={PageCommon.listviewtablecontrolsButtonGroup}>
            {editMode ? (
              <>
                <Button
                  theme="secondary"
                  size="small"
                  disabled={isSaving}
                  onClick={() => {
                    if (
                      !isDirty ||
                      window.confirm(
                        'You have unsaved changes. Are you sure you want to cancel editing?'
                      )
                    ) {
                      dispatch(toggleDataTypeEditMode());
                    }
                  }}
                >
                  Cancel
                </Button>
                <Button
                  theme="primary"
                  size="small"
                  type="submit"
                  isLoading={isSaving}
                  disabled={
                    isSaving || !isDirty || !isValidDataType(modifiedDataType)
                  }
                >
                  Save Data Type
                </Button>
              </>
            ) : (
              <Button
                theme="secondary"
                size="small"
                disabled={isSaving}
                onClick={() => {
                  dispatch(toggleDataTypeEditMode());
                }}
              >
                Edit
              </Button>
            )}
          </ButtonGroup>
        )}
      </div>

      <Card id="dataTypeFormCard" detailview>
        {saveError && (
          <InlineNotification theme="alert">{saveError}</InlineNotification>
        )}

        <CardRow
          title="Basic Details"
          tooltip={
            <>
              Define the basic details of this data type. &nbsp;
              <a
                href="https://docs.userclouds.com/docs/data-types-1"
                title="UserClouds documentation for key concepts about dataTypes"
                target="new"
                className={PageCommon.link}
              >
                Learn more here.
              </a>
            </>
          }
          collapsible
        >
          <div className={PageCommon.carddetailsrow}>
            <Label>
              Name
              <br />
              {editMode ? (
                <TextInput
                  name="dataType_name"
                  id="dataType_name"
                  type="text"
                  value={modifiedDataType.name}
                  onChange={(e: React.ChangeEvent) => {
                    const val = (e.target as HTMLInputElement).value;
                    dispatch(
                      modifyDataType({
                        name: val,
                      })
                    );
                  }}
                />
              ) : (
                <InputReadOnly>{modifiedDataType.name}</InputReadOnly>
              )}
            </Label>

            <Label>
              Description
              <br />
              {editMode ? (
                <TextArea
                  name="dataType_description"
                  value={modifiedDataType.description}
                  placeholder="Add a description"
                  onChange={(e: React.ChangeEvent) => {
                    const val = (e.target as HTMLTextAreaElement).value;
                    dispatch(
                      modifyDataType({
                        description: val,
                      })
                    );
                  }}
                />
              ) : (
                <Text>{modifiedDataType.description}</Text>
              )}
            </Label>
            <Label>
              Specification Type
              <br />
              <Text>
                {modifiedDataType.is_native ? 'UserClouds Default' : 'Custom'}
              </Text>
            </Label>
            <Label>
              Is Composite Field Type
              <br />
              <Text>{String(modifiedDataType.is_composite_field_type)}</Text>
            </Label>
          </div>
        </CardRow>

        <CardRow
          title="Field Attributes"
          tooltip={
            <>
              Configure the fields associated with this composite data type.
              Fields must start with a capital letter and use underscores to
              separate subwords.
            </>
          }
          collapsible
        >
          <Table id="dataTypeColumns" className={styles.dataTypeColumnsTable}>
            <TableHead>
              <TableRow>
                <TableRowHead key="field_name">Field Name</TableRowHead>
                <TableRowHead key="data_type">Data Type</TableRowHead>
                <TableRowHead key="is_required">Required</TableRowHead>
                <TableRowHead key="ignore_uniqueness">
                  Ignore for Uniqueness
                </TableRowHead>
                <TableRowHead key="column_remove" />
              </TableRow>
            </TableHead>
            <TableBody>
              {dataTypes &&
              modifiedDataType.composite_attributes?.fields &&
              modifiedDataType.composite_attributes?.fields.length ? (
                <>
                  {modifiedDataType.composite_attributes.fields.map(
                    (field: CompositeField, i) => (
                      <TableRow>
                        <TableCell>
                          {editMode ? (
                            <TextInput
                              name="field_name"
                              id={'field_name' + i}
                              type="text"
                              value={field.name}
                              onChange={(e: React.ChangeEvent) => {
                                const val = (e.target as HTMLInputElement)
                                  .value;
                                const new_composite_attributes = JSON.parse(
                                  JSON.stringify(
                                    modifiedDataType.composite_attributes
                                  )
                                );

                                new_composite_attributes.fields[i].name =
                                  validFieldName(val);
                                dispatch(
                                  modifyDataType({
                                    composite_attributes:
                                      new_composite_attributes,
                                  })
                                );
                              }}
                            />
                          ) : (
                            <InputReadOnly>{field.name}</InputReadOnly>
                          )}
                        </TableCell>
                        <TableCell>
                          {editMode ? (
                            <Select
                              name="field_type"
                              value={field.data_type.name}
                              onChange={(
                                e: React.ChangeEvent<HTMLSelectElement>
                              ) => {
                                const selectedDataType = dataTypes.data.find(
                                  (dt) => e.target.value === dt.id
                                );
                                if (selectedDataType) {
                                  const new_composite_attributes = JSON.parse(
                                    JSON.stringify(
                                      modifiedDataType.composite_attributes
                                    )
                                  );

                                  new_composite_attributes.fields[i].data_type =
                                    {
                                      id: selectedDataType.id,
                                      name: selectedDataType.name,
                                    };
                                  dispatch(
                                    modifyDataType({
                                      composite_attributes:
                                        new_composite_attributes,
                                    })
                                  );
                                }
                              }}
                              required
                            >
                              {dataTypes.data.map((type) => (
                                <option value={type.name} key={type.name}>
                                  {type.name}
                                </option>
                              ))}
                            </Select>
                          ) : (
                            <InputReadOnly>
                              {field.data_type.name}
                            </InputReadOnly>
                          )}
                        </TableCell>
                        <TableCell>
                          {editMode ? (
                            <Checkbox
                              id="required"
                              value="required"
                              name="requiredselect"
                              onChange={() => {
                                const new_composite_attributes = JSON.parse(
                                  JSON.stringify(
                                    modifiedDataType.composite_attributes
                                  )
                                );

                                new_composite_attributes.fields[i].required =
                                  !new_composite_attributes.fields[i].required;
                                dispatch(
                                  modifyDataType({
                                    composite_attributes:
                                      new_composite_attributes,
                                  })
                                );
                              }}
                              checked={field.required}
                              disabled={!editMode}
                            />
                          ) : field.required ? (
                            <IconCheckmark />
                          ) : (
                            <IconDash />
                          )}
                        </TableCell>

                        <TableCell>
                          {editMode ? (
                            <Checkbox
                              id="unique"
                              value="unique"
                              name="uniqueselect"
                              onChange={() => {
                                const new_composite_attributes = JSON.parse(
                                  JSON.stringify(
                                    modifiedDataType.composite_attributes
                                  )
                                );

                                new_composite_attributes.fields[
                                  i
                                ].ignore_for_uniqueness =
                                  !field.ignore_for_uniqueness;
                                dispatch(
                                  modifyDataType({
                                    composite_attributes:
                                      new_composite_attributes,
                                  })
                                );
                              }}
                              checked={field.ignore_for_uniqueness}
                              disabled={!editMode}
                            />
                          ) : field.ignore_for_uniqueness ? (
                            <IconCheckmark />
                          ) : (
                            <IconDash />
                          )}
                        </TableCell>
                      </TableRow>
                    )
                  )}
                </>
              ) : (
                <TableRow>
                  <TableCell colSpan={4}>No fields added.</TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
          {editMode && (
            <ButtonGroup>
              <Button
                theme="secondary"
                onClick={() => {
                  dispatch(addFieldToDataType());
                }}
              >
                Add Field
              </Button>
            </ButtonGroup>
          )}
        </CardRow>
      </Card>
    </form>
  );
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  modifiedDataType: state.modifiedDataType,
  dataTypes: state.dataTypes,
  editMode: state.dataTypeEditMode,
  isDirty: state.dataTypeIsDirty,
  isSaving: state.savingDataType,
  saveError: state.saveDataTypeError,
  routeParams: state.routeParams,
  featureFlags: state.featureFlags,
}))(DataTypeDetailPage);
