import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  Checkbox,
  Label,
  Select,
  InlineNotification,
  Table,
  TableHead,
  TableBody,
  TableCell,
  TableRow,
  TableRowHead,
  TextArea,
  TextInput,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import PaginatedResult from '../models/PaginatedResult';
import {
  CompositeField,
  DataType,
  isValidDataType,
  validFieldName,
} from '../models/DataType';
import { PageTitle } from '../mainlayout/PageWrap';
import { RootState, AppDispatch } from '../store';
import { fetchDataTypes, handleCreateDataType } from '../thunks/userstore';
import {
  addFieldToDataTypeToCreate,
  loadCreateDataTypePage,
  modifyDataTypeToCreate,
} from '../actions/userstore';

import styles from './DataTypeDetailPage.module.css';
import PageCommon from './PageCommon.module.css';
import { redirect } from '../routing';

const CreateDataTypePage = ({
  selectedTenantID,
  dataType,
  dataTypes,
  isSaving,
  saveError,
  query,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  dataType: DataType;
  dataTypes: PaginatedResult<DataType> | undefined;
  isSaving: boolean;
  saveError: string;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  const cleanQuery = makeCleanPageLink(query);
  const isDirty = dataType.name || dataType.description;

  useEffect(() => {
    dispatch(loadCreateDataTypePage());
    if (selectedTenantID) {
      dispatch(fetchDataTypes(selectedTenantID, new URLSearchParams()));
    }
  }, [selectedTenantID, dispatch]);
  return (
    <form
      onSubmit={(e: React.FormEvent) => {
        e.preventDefault();
        dispatch(handleCreateDataType());
      }}
    >
      <div className={PageCommon.listviewtablecontrols}>
        <PageTitle title="Create Data Type" itemName="New Data Type" />

        <ButtonGroup className={PageCommon.listviewtablecontrolsButtonGroup}>
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
                redirect(`/dataTypes?${cleanQuery}`);
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
            disabled={isSaving || !isDirty || !isValidDataType(dataType)}
          >
            Create Data Type
          </Button>
        </ButtonGroup>
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
              <TextInput
                name="dataType_name"
                id="dataType_name"
                type="text"
                value={dataType.name}
                onChange={(e: React.ChangeEvent) => {
                  const val = (e.target as HTMLInputElement).value;
                  dispatch(
                    modifyDataTypeToCreate({
                      name: val,
                    })
                  );
                }}
              />
            </Label>

            <Label>
              Description
              <br />
              <TextArea
                name="dataType_description"
                value={dataType.description}
                placeholder="Add a description"
                onChange={(e: React.ChangeEvent) => {
                  const val = (e.target as HTMLTextAreaElement).value;
                  dispatch(
                    modifyDataTypeToCreate({
                      description: val,
                    })
                  );
                }}
              />
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
          <div className={styles.dataTypeColumns}>
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
                {dataTypes && dataType.composite_attributes?.fields.length ? (
                  <>
                    {dataType.composite_attributes.fields.map(
                      (field: CompositeField, i) => (
                        <TableRow>
                          <TableCell>
                            <TextInput
                              name="field_name"
                              id={'field_name' + i}
                              type="text"
                              value={field.name}
                              onChange={(e: React.ChangeEvent) => {
                                const val = (e.target as HTMLInputElement)
                                  .value;
                                const new_composite_attributes = {
                                  ...dataType.composite_attributes,
                                };
                                new_composite_attributes.fields[i].name =
                                  validFieldName(val);
                                dispatch(
                                  modifyDataTypeToCreate({
                                    composite_attributes:
                                      new_composite_attributes,
                                  })
                                );
                              }}
                            />
                          </TableCell>
                          <TableCell>
                            <Select
                              name="field_type"
                              defaultValue={dataTypes.data[0].name}
                              onChange={(
                                e: React.ChangeEvent<HTMLSelectElement>
                              ) => {
                                const selectedDataType = dataTypes.data.find(
                                  (dt) => e.target.value === dt.id
                                );
                                if (selectedDataType) {
                                  const new_composite_attributes = {
                                    ...dataType.composite_attributes,
                                  };
                                  new_composite_attributes.fields[i].data_type =
                                    {
                                      id: selectedDataType.id,
                                      name: selectedDataType.name,
                                    };
                                  dispatch(
                                    modifyDataTypeToCreate({
                                      composite_attributes:
                                        new_composite_attributes,
                                    })
                                  );
                                }
                              }}
                              required
                            >
                              <option key="select" value="">
                                Select a data type
                              </option>
                              {dataTypes.data.map(
                                (type) =>
                                  type.is_composite_field_type && (
                                    <option value={type.id} key={type.id}>
                                      {type.name}
                                    </option>
                                  )
                              )}
                            </Select>
                          </TableCell>
                          <TableCell>
                            <Checkbox
                              id="required"
                              value="required"
                              name="requiredselect"
                              onChange={() => {
                                const new_composite_attributes = {
                                  ...dataType.composite_attributes,
                                };
                                new_composite_attributes.fields[i].required =
                                  !new_composite_attributes.fields[i].required;
                                dispatch(
                                  modifyDataTypeToCreate({
                                    composite_attributes:
                                      new_composite_attributes,
                                  })
                                );
                              }}
                              checked={field.required}
                            />
                          </TableCell>

                          <TableCell>
                            <Checkbox
                              id="unique"
                              value="unique"
                              name="uniqueselect"
                              onChange={() => {
                                const new_composite_attributes = {
                                  ...dataType.composite_attributes,
                                };
                                new_composite_attributes.fields[
                                  i
                                ].ignore_for_uniqueness =
                                  !field.ignore_for_uniqueness;
                                dispatch(
                                  modifyDataTypeToCreate({
                                    composite_attributes:
                                      new_composite_attributes,
                                  })
                                );
                              }}
                              checked={field.ignore_for_uniqueness}
                            />
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
            <ButtonGroup>
              <Button
                theme="secondary"
                onClick={() => {
                  dispatch(addFieldToDataTypeToCreate());
                }}
              >
                Add Field
              </Button>
            </ButtonGroup>
          </div>
        </CardRow>
      </Card>
    </form>
  );
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  dataType: state.dataTypeToCreate,
  dataTypes: state.dataTypes,
  isSaving: state.savingDataType,
  saveError: state.saveDataTypeError,
  query: state.query,
  featureFlags: state.featureFlags,
}))(CreateDataTypePage);
