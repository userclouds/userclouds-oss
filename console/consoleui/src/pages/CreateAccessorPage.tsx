import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  Checkbox,
  Dialog,
  GlobalStyles,
  IconButton,
  IconDeleteBin,
  InputReadOnly,
  Label,
  LoaderDots,
  Select,
  InlineNotification,
  Radio,
  Table,
  TableHead,
  TableBody,
  TableCell,
  TableRow,
  TableRowHead,
  Text,
  TextArea,
  TextInput,
  DialogBody,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import { RootState, AppDispatch } from '../store';
import { VALID_NAME_PATTERN } from '../models/helpers';
import { FeatureFlags } from '../models/FeatureFlag';
import { NilUuid } from '../models/Uuids';
import PaginatedResult from '../models/PaginatedResult';
import {
  AccessorColumn,
  AccessorSavePayload,
  getUnusedPurposes,
  isValidAccessor,
  isTokenizingTransformer,
  transformerIsValidForAccessorColumn,
} from '../models/Accessor';
import { DurationType } from '../models/ColumnRetentionDurations';
import AccessPolicy, {
  AccessPolicyTemplate,
  blankPolicy,
  blankPolicyTemplate,
} from '../models/AccessPolicy';
import Transformer from '../models/Transformer';
import {
  Column,
  columnNameAlphabeticalComparator,
} from '../models/TenantUserStoreConfig';
import Purpose from '../models/Purpose';
import { fetchUserStoreConfig } from '../thunks/userstore';
import {
  fetchTransformers,
  createAccessPolicyTemplateForAccessPolicy,
  fetchGlobalAccessorPolicy,
  fetchAllAccessPolicies,
} from '../thunks/tokenizer';
import { fetchPurposes } from '../thunks/purposes';
import { handleCreateAccessor } from '../thunks/accessors';
import {
  loadCreateAccessorPage,
  modifyAccessorToCreate,
  modifyColumnInAccessorToCreate,
  removePurposeFromAccessorToCreate,
  addPurposeToAccessorToCreate,
} from '../actions/accessors';
import {
  getAccessPolicySuccess,
  getTokenAccessPolicySuccess,
  modifyAccessPolicy,
  closePolicyTemplateDialog,
} from '../actions/tokenizer';
import { PageTitle } from '../mainlayout/PageWrap';
import Link from '../controls/Link';
import PolicyTemplateForm from './PolicyTemplateForm';
import PolicyComposer, { ConnectedPolicyChooserDialog } from './PolicyComposer';
import PageCommon from './PageCommon.module.css';
import Styles from './AccessorDetailPage.module.css';
import { redirect } from '../routing';
import { truncateWithEllipsis } from '../util/string';
import { featureIsEnabled } from '../util/featureflags';

const CreateAccessorColumnRow = ({
  accessor,
  column,
  transformers,
  fetchingTransformers,
  accessPolicies,
  fetchingAccessPolicies,
  dispatch,
}: {
  accessor: AccessorSavePayload;
  column: AccessorColumn;
  transformers: PaginatedResult<Transformer> | undefined;
  fetchingTransformers: boolean;
  accessPolicies: PaginatedResult<AccessPolicy> | undefined;
  fetchingAccessPolicies: boolean;
  dispatch: AppDispatch;
}) => {
  return (
    <TableRow key={column.id}>
      <TableCell>
        <InputReadOnly>{column.name}</InputReadOnly>
      </TableCell>
      <TableCell>
        <InputReadOnly>
          {column.data_type_name + (column.is_array ? ' array' : '')}
        </InputReadOnly>
      </TableCell>
      <TableCell>
        {transformers && transformers.data && transformers.data.length ? (
          <Select
            name="selected_transformer"
            value={column.transformer_id}
            onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
              const val = e.target.value;
              dispatch(
                modifyColumnInAccessorToCreate(column.id, {
                  transformer_id: val,
                })
              );
            }}
          >
            <option key="select_a_transformer" value="">
              Select a transformer
            </option>
            <option key={NilUuid} value={NilUuid}>
              {truncateWithEllipsis(column.default_transformer_name, 25) +
                ' (default)'}
            </option>
            {transformers.data
              .filter((transformer: Transformer) =>
                transformerIsValidForAccessorColumn(column, transformer)
              )
              .map((transformer: Transformer) => (
                <option value={transformer.id} key={transformer.id}>
                  {truncateWithEllipsis(transformer.name, 35)}
                </option>
              ))}
          </Select>
        ) : fetchingTransformers ? (
          <LoaderDots size="small" assistiveText="Loading transformers" />
        ) : (
          <InlineNotification theme="alert">
            Error fetching transformers
          </InlineNotification>
        )}
      </TableCell>
      <TableCell>
        {isTokenizingTransformer(transformers?.data, column.transformer_id) &&
          (accessPolicies?.data ? (
            <Select
              id="selected_token_access_policy"
              name="selected_token_access_policy"
              value={column.token_access_policy_id}
              onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
                const val = e.target.value;
                dispatch(
                  modifyColumnInAccessorToCreate(column.id, {
                    token_access_policy_id: val,
                  })
                );
              }}
            >
              <option key="select_an_access_policy">
                Select an access policy
              </option>
              {accessPolicies.data.map((policy: AccessPolicy) => (
                <option value={policy.id} key={policy.id}>
                  {truncateWithEllipsis(policy.name, 35)}
                </option>
              ))}
            </Select>
          ) : fetchingAccessPolicies ? (
            <LoaderDots size="small" assistiveText="Loading policies" />
          ) : (
            <InlineNotification theme="alert">
              Error fetching access policies
            </InlineNotification>
          ))}
      </TableCell>
      <TableCell>
        <IconButton
          icon={<IconDeleteBin />}
          onClick={() => {
            dispatch(
              modifyAccessorToCreate({
                columns: accessor.columns.filter(
                  (col: AccessorColumn) => col.id !== column.id
                ),
              })
            );
          }}
          title="Remove column"
          aria-label="Remove column"
        />
      </TableCell>
    </TableRow>
  );
};
const ConnectedCreateColumn = connect((state: RootState) => ({
  transformers: state.transformers,
  fetchingTransformers: state.fetchingTransformers,
  accessPolicies: state.allAccessPolicies,
  fetchingAccessPolicies: state.fetchingAllAccessPolicies,
}))(CreateAccessorColumnRow);

const CreateAccessorPage = ({
  selectedTenantID,
  accessor,
  userStoreColumns,
  columnsToAdd,
  dropdownValue,
  purposes,
  fetchingPurposes,
  isSaving,
  saveError,
  newTemplate,
  transformers,
  accessPolicy,
  policyTemplateDialogIsOpen,
  query,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  accessor: AccessorSavePayload;
  userStoreColumns: Column[] | undefined;
  columnsToAdd: AccessorColumn[];
  dropdownValue: string;
  transformers: PaginatedResult<Transformer> | undefined;
  purposes: PaginatedResult<Purpose> | undefined;
  fetchingPurposes: boolean;
  isSaving: boolean;
  saveError: string;
  newTemplate: AccessPolicyTemplate | undefined;
  accessPolicy: AccessPolicy | undefined;
  policyTemplateDialogIsOpen: boolean;
  query: URLSearchParams;

  dispatch: AppDispatch;
}) => {
  const cleanQuery = makeCleanPageLink(query);
  const addableColumns = (userStoreColumns || [])
    .filter(
      (column: Column) =>
        !accessor.columns.find(
          (col: AccessorColumn) => col.name === column.name
        ) && !columnsToAdd.find((col: AccessorColumn) => col.id === column.id)
    )
    .sort(columnNameAlphabeticalComparator);
  const isDirty =
    accessor.columns.length ||
    accessor.name ||
    accessor.description ||
    accessor.selector_config.where_clause ||
    accessor.access_policy_id;
  const dialog: HTMLDialogElement | null = document.getElementById(
    'createPolicyTemplateDialog'
  ) as HTMLDialogElement;
  useEffect(() => {
    dispatch(loadCreateAccessorPage());
    if (selectedTenantID) {
      dispatch(fetchUserStoreConfig(selectedTenantID));
      dispatch(
        fetchPurposes(
          selectedTenantID,
          new URLSearchParams({ purposes_limit: '1000' })
        )
      );
      dispatch(
        fetchTransformers(selectedTenantID, new URLSearchParams(), 1000)
      );
      dispatch(fetchAllAccessPolicies(selectedTenantID));
      dispatch(getAccessPolicySuccess(blankPolicy()));
      dispatch(getTokenAccessPolicySuccess(blankPolicy()));
    }
  }, [selectedTenantID, dispatch]);
  return (
    <>
      <form
        onSubmit={(e: React.FormEvent) => {
          e.preventDefault();
          dispatch(handleCreateAccessor());
        }}
      >
        <div className={PageCommon.listviewtablecontrols}>
          <PageTitle title="Create Accessor" itemName="New Accessor" />

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
                  redirect(`/accessors?${cleanQuery}`);
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
                isSaving ||
                !isDirty ||
                !transformers ||
                !accessPolicy ||
                !isValidAccessor(accessor, accessPolicy)
              }
            >
              Create Accessor
            </Button>
          </ButtonGroup>
        </div>

        <Card id="accessorFormCard" detailview>
          {saveError && (
            <InlineNotification theme="alert">{saveError}</InlineNotification>
          )}

          <CardRow
            title="Basic Details"
            tooltip={
              <>
                {
                  'Add the metadata, columns, and policies relating to an accessor. '
                }
                <a
                  href="https://docs.userclouds.com/docs/accessors-read-apis"
                  title="UserClouds documentation for key concepts about accessors"
                  target="new"
                  className={PageCommon.link}
                >
                  Learn more here.
                </a>
              </>
            }
            collapsible
          >
            <Label>
              Name
              <br />
              <TextInput
                name="accessor_name"
                id="accessor_name"
                type="text"
                value={accessor.name}
                pattern={VALID_NAME_PATTERN}
                onChange={(e: React.ChangeEvent) => {
                  const val = (e.target as HTMLInputElement).value;
                  dispatch(
                    modifyAccessorToCreate({
                      name: val,
                    })
                  );
                }}
              />
            </Label>

            <Label className={GlobalStyles['mt-6']}>
              Description
              <br />
              <TextArea
                name="accessor_description"
                value={accessor.description}
                placeholder="Add a description"
                onChange={(e: React.ChangeEvent) => {
                  const val = (e.target as HTMLTextAreaElement).value;
                  dispatch(
                    modifyAccessorToCreate({
                      description: val,
                    })
                  );
                }}
              />
            </Label>
            <fieldset className={GlobalStyles['mt-6']}>
              <Label htmlFor="livedataonly">Deleted data access</Label>
              <Radio
                id="livedataonly"
                value={DurationType.Live}
                name="liveDataSelect"
                onClick={(e: React.ChangeEvent<HTMLInputElement>) => {
                  if (e.target.checked) {
                    dispatch(
                      modifyAccessorToCreate({
                        data_life_cycle_state: e.target.value,
                      })
                    );
                  }
                }}
                checked={accessor.data_life_cycle_state === DurationType.Live}
              >
                Retrieve live data only (not soft-deleted)
              </Radio>
              <Radio
                id="softdeletedata"
                value={DurationType.SoftDeleted}
                name="liveDataSelect"
                onClick={(e: React.ChangeEvent<HTMLInputElement>) => {
                  if (e.target.checked) {
                    dispatch(
                      modifyAccessorToCreate({
                        data_life_cycle_state: e.target.value,
                      })
                    );
                  }
                }}
                checked={
                  accessor.data_life_cycle_state === DurationType.SoftDeleted
                }
              >
                Retrieve soft-deleted data only (admin privileges required)
              </Radio>
            </fieldset>

            <Label className={GlobalStyles['mt-6']}>
              Audit Log
              <br />
              <Checkbox
                id="accessor_audit_logged"
                checked={accessor.is_audit_logged}
                onChange={(e: React.ChangeEvent) => {
                  const { checked } = e.target as HTMLInputElement;
                  dispatch(
                    modifyAccessorToCreate({ is_audit_logged: checked })
                  );
                }}
              >
                Log usage of this accessor
              </Checkbox>
            </Label>
            <Label className={GlobalStyles['mt-6']}>
              Use Search Index
              <br />
              <Checkbox
                id="use_search_index"
                checked={accessor.use_search_index}
                onChange={(e: React.ChangeEvent) => {
                  const { checked } = e.target as HTMLInputElement;
                  dispatch(
                    modifyAccessorToCreate({ use_search_index: checked })
                  );
                }}
              >
                Index this accessor for search
              </Checkbox>
            </Label>
          </CardRow>

          <CardRow
            title="Columns"
            tooltip={
              <>
                Configure the columns from which this accessor will retrieve
                data, and how each column’s data will be transformed in the
                accessor response.
              </>
            }
            collapsible
          >
            <Table
              id="accessorColumns"
              spacing="packed"
              className={Styles.accessorcolumnstable}
            >
              <TableHead>
                <TableRow>
                  <TableRowHead key="column_name">Name</TableRowHead>
                  <TableRowHead key="column_type">Type</TableRowHead>
                  <TableRowHead key="column_transformer">
                    Transformer
                  </TableRowHead>
                  <TableRowHead key="column_token_access_policy">
                    Token access policy
                  </TableRowHead>
                  <TableRowHead key="column_remove" />
                </TableRow>
              </TableHead>
              <TableBody>
                {accessor.columns.length ? (
                  <>
                    {accessor.columns.map((column: AccessorColumn) => (
                      <ConnectedCreateColumn
                        accessor={accessor}
                        column={column}
                        key={column.id}
                      />
                    ))}
                  </>
                ) : (
                  <TableRow>
                    <TableCell colSpan={4}>
                      No columns selected. You must have at least one column.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
            {addableColumns?.length && (
              <Select
                id="selectUserStoreColumnToAdd"
                name="select_column"
                value={dropdownValue}
                className={GlobalStyles['mt-3']}
                onChange={(e: React.ChangeEvent) => {
                  const val = (e.target as HTMLSelectElement).value;
                  const matchingColumn = addableColumns.find(
                    (col: Column) => col.name === val
                  );
                  if (matchingColumn) {
                    dispatch(
                      modifyAccessorToCreate({
                        columns: [
                          ...accessor.columns,
                          {
                            id: matchingColumn.id,
                            name: matchingColumn.name,
                            is_array: matchingColumn.is_array,
                            data_type_name: matchingColumn.data_type.name,
                            data_type_id: matchingColumn.data_type.id,
                            default_transformer_name:
                              matchingColumn.default_transformer.name,
                          },
                        ],
                      })
                    );
                  }
                }}
              >
                <option key="no_selection" value="">
                  Select a column
                </option>
                {addableColumns.map((column: Column) => (
                  <option key={column.id} value={column.name}>
                    {column.name}
                  </option>
                ))}
              </Select>
            )}
          </CardRow>

          <CardRow
            title="Selector"
            tooltip={
              <>
                {
                  'A selector is an SQL-like clause that specifies which records an accessor should return data for. '
                }
                <a
                  href="https://docs.userclouds.com/docs/selectors"
                  title="UserClouds documentation for selectors"
                  target="new"
                  className={PageCommon.link}
                >
                  Learn more here.
                </a>
              </>
            }
            collapsible
          >
            <Label>
              Selector "where" clause
              <br />
              <TextInput
                name="accessor_selector_config"
                id="accessor_selector_config"
                type="text"
                placeholder="{id} = ANY(?) OR {phone_number} LIKE ?"
                value={accessor.selector_config.where_clause}
                onChange={(e: React.ChangeEvent) => {
                  const val = (e.target as HTMLInputElement).value;
                  dispatch(
                    modifyAccessorToCreate({
                      selector_config: { where_clause: val },
                    })
                  );
                }}
              />
            </Label>
          </CardRow>

          <CardRow
            title="Policy"
            tooltip={
              <>
                {
                  'Select an access policy (describing when this accessor can be called) and a token resolution policy (describing when tokens generated by this accessor can be resolved). '
                }
                <a
                  href="https://docs.userclouds.com/docs/access-policies-1"
                  title="UserClouds documentation for access policies"
                  target="new"
                  className={PageCommon.link}
                >
                  Learn more here.
                </a>
              </>
            }
            collapsible
          >
            <Label>
              Column policy override
              <br />
              <Checkbox
                id="accessor_override_column_policies"
                checked={accessor.are_column_access_policies_overridden}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                  dispatch(
                    modifyAccessorToCreate({
                      are_column_access_policies_overridden: e.target.checked,
                    })
                  );
                }}
              >
                Enable to ignore policies for individual columns and only use
                the access policy configured for the accessor.
              </Checkbox>
            </Label>

            <PolicyComposer
              tokenRes={false}
              policy={accessPolicy}
              changeAccessPolicyAction={modifyAccessPolicy}
              tableID="accessorManualPolicyComponents"
            />
          </CardRow>
          <CardRow
            title="Purposes of access"
            tooltip={
              <>Define the acceptable purposes for your users to access.</>
            }
            collapsible
          >
            {purposes ? (
              <>
                <Table
                  id="purpose"
                  spacing="nowrap"
                  className={Styles.accessorpurposestable}
                >
                  <TableHead>
                    <TableRow key="head">
                      <TableRowHead />
                      <TableRowHead>
                        {accessor.purposes.length ? 'Purpose' : ''}
                      </TableRowHead>
                      <TableRowHead>
                        {accessor.purposes.length ? 'Description' : ''}
                      </TableRowHead>
                      <TableRowHead>
                        {accessor.purposes.length ? 'Delete' : ''}
                      </TableRowHead>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {accessor.purposes.length > 0 ? (
                      accessor.purposes.map((purpose, i) => (
                        <TableRow key={`chosenpurpose-${purpose.id}`}>
                          <TableCell>
                            <Text className={Styles.whereText}>
                              {i === 0 ? 'Where' : 'And'}
                            </Text>
                          </TableCell>
                          <TableCell>
                            <Text>{purpose.name}</Text>
                          </TableCell>
                          <TableCell>
                            <Text> {purpose.description}</Text>
                          </TableCell>
                          <TableCell>
                            <IconButton
                              icon={<IconDeleteBin />}
                              onClick={() => {
                                dispatch(
                                  removePurposeFromAccessorToCreate(purpose)
                                );
                              }}
                              title="Delete Component"
                              aria-label="Delete Component"
                            />
                          </TableCell>
                        </TableRow>
                      ))
                    ) : (
                      <TableRow>
                        <TableCell colSpan={4}>No purpose selected</TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
                <br />
                {getUnusedPurposes(purposes, accessor.purposes).length > 0 ? (
                  <Select
                    name="accessor_purpose"
                    value="Select a purpose"
                    onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
                      dispatch(addPurposeToAccessorToCreate(e.target.value));
                    }}
                    required
                  >
                    <option key="select">Select a purpose</option>
                    {getUnusedPurposes(purposes, accessor?.purposes).map(
                      (purpose: Purpose) => (
                        <option
                          value={purpose.id}
                          key={`unusedpurpose-${purpose.id}`}
                        >
                          {purpose.name}
                        </option>
                      )
                    )}
                  </Select>
                ) : (
                  <Text>
                    {'No purposes available. You can add purposes '}
                    <Link
                      key="link to create purposes"
                      href={`/purposes/create${cleanQuery}`}
                    >
                      here
                    </Link>
                    .
                  </Text>
                )}
              </>
            ) : fetchingPurposes ? (
              <LoaderDots size="small" assistiveText="Loading purposes" />
            ) : (
              'Something went wrong.'
            )}
          </CardRow>
        </Card>
      </form>
      <Dialog
        id="createPolicyTemplateDialog"
        title="Create Policy Template"
        description="Create a new template and add it to your composite policy. The template will be saved to your library for re-use later."
        fullPage
      >
        {policyTemplateDialogIsOpen && (
          <DialogBody>
            <PolicyTemplateForm
              editableTemplate={newTemplate || blankPolicyTemplate()}
              savedTemplate={undefined}
              saveTemplate={createAccessPolicyTemplateForAccessPolicy(() => {
                dispatch(closePolicyTemplateDialog());
                dialog.close();
              })}
              onCancel={() => {
                dispatch(closePolicyTemplateDialog());
                dialog?.close();
              }}
              searchParams={query}
            />
          </DialogBody>
        )}
      </Dialog>
      <ConnectedPolicyChooserDialog
        policy={accessPolicy}
        tokenRes={false}
        changeAccessPolicyAction={modifyAccessPolicy}
        createNewPolicyTemplateHandler={() => dialog.showModal()}
      />
    </>
  );
};

const ConnectedCreateAccessorPage = connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  accessor: state.accessorToCreate,
  userStoreColumns: state.userStoreColumns,
  columnsToAdd: state.accessorColumnsToAdd,
  dropdownValue: state.accessorAddColumnDropdownValue,
  purposes: state.purposes,
  fetchingPurposes: state.fetchingPurposes,
  isSaving: state.savingAccessor,
  saveError: state.createAccessorError,
  newTemplate: state.policyTemplateToCreate,
  transformers: state.transformers,
  accessPolicy: state.modifiedAccessPolicy,
  policyTemplateDialogIsOpen: state.policyTemplateDialogIsOpen,
  query: state.query,
  featureFlags: state.featureFlags,
}))(CreateAccessorPage);

const AccessorPage = ({
  selectedTenantID,
  featureFlags,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  featureFlags: FeatureFlags | undefined;
  dispatch: AppDispatch;
}) => {
  const globalPolicies = featureIsEnabled(
    'global-access-policies',
    featureFlags
  );

  useEffect(() => {
    if (selectedTenantID && globalPolicies) {
      dispatch(fetchGlobalAccessorPolicy(selectedTenantID));
    }
  }, [selectedTenantID, globalPolicies, dispatch]);

  return <ConnectedCreateAccessorPage />;
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  featureFlags: state.featureFlags,
}))(AccessorPage);
