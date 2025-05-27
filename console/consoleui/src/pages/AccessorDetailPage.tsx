import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Accordion,
  AccordionItem,
  Button,
  ButtonGroup,
  Card,
  CardFooter,
  CardRow,
  Checkbox,
  CodeEditor,
  Dialog,
  EmptyState,
  GlobalStyles,
  Heading,
  IconButton,
  IconDeleteBin,
  IconLock2,
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
  TableTitle,
  Text,
  TextArea,
  TextInput,
  ToolTip,
  DialogBody,
  TextShortener,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import { RootState, AppDispatch } from '../store';
import { VALID_NAME_PATTERN } from '../models/helpers';
import { FeatureFlags } from '../models/FeatureFlag';
import { NilUuid } from '../models/Uuids';
import PaginatedResult from '../models/PaginatedResult';
import Accessor, {
  AccessorColumn,
  ExecuteAccessorResponse,
  getUnusedPurposes,
  isTokenizingTransformer,
  transformerIsValidForAccessorColumn,
  DEFAULT_PASSTHROUGH_ACCESSOR_ID,
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
import { SelectedTenant } from '../models/Tenant';
import { fetchUserStoreConfig } from '../thunks/userstore';
import {
  fetchAccessPolicy,
  fetchTransformers,
  createAccessPolicyTemplateForAccessPolicy,
  fetchGlobalAccessorPolicy,
  fetchAllAccessPolicies,
} from '../thunks/tokenizer';
import { fetchPurposes } from '../thunks/purposes';
import {
  fetchAccessor,
  executeAccessor,
  saveAccessorDetailsAndConfiguration,
} from '../thunks/accessors';
import {
  addAccessorColumn,
  changeSelectedTransformerForColumn,
  changeSelectedTokenAccessPolicyForColumn,
  addPurposeToAccessor,
  modifyAccessorDetails,
  removePurposeFromAccessor,
  toggleAccessorColumnForDelete,
  changeExecuteAccessorContext,
  changeExecuteAccessorSelectorValues,
  executeAccessorReset,
  toggleAccessorEditMode,
} from '../actions/accessors';
import {
  getAccessPolicySuccess,
  modifyAccessPolicy,
} from '../actions/tokenizer';
import { PageTitle } from '../mainlayout/PageWrap';
import Link from '../controls/Link';
import PolicyTemplateForm from './PolicyTemplateForm';
import PolicyComposer, { ConnectedPolicyChooserDialog } from './PolicyComposer';
import PageCommon from './PageCommon.module.css';
import Styles from './AccessorDetailPage.module.css';
import { truncateWithEllipsis } from '../util/string';
import { featureIsEnabled } from '../util/featureflags';

const ExecuteAccessorDialog = connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  stats: state.executeAccessorStats,
  executeAccessorContext: state.executeAccessorContext,
  executeAccessorSelectorValues: state.executeAccessorSelectorValues,
  executeAccessorResponse: state.executeAccessorResponse,
  executeAccessorError: state.executeAccessorError,
  featureFlags: state.featureFlags,
}))(({
  selectedTenantID,
  accessor,
  stats,
  executeAccessorContext,
  executeAccessorSelectorValues,
  executeAccessorResponse,
  executeAccessorError,
  featureFlags,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  accessor: Accessor;
  stats:
    | {
        frequencies: Record<string, number>;
        uniqueness: Record<string, Record<string, Record<string, number>>>;
      }
    | undefined;
  executeAccessorContext: string;
  executeAccessorSelectorValues: string;
  executeAccessorResponse: ExecuteAccessorResponse | undefined;
  executeAccessorError: string;
  featureFlags: FeatureFlags | undefined;
  dispatch: AppDispatch;
}) => {
  const accessorStats = featureIsEnabled(
    'accessor-data-analysis',
    featureFlags
  );
  return (
    <form
      onSubmit={(e: React.FormEvent<HTMLFormElement>) => {
        e.preventDefault();
        const formData = new FormData(e.target as HTMLFormElement);
        const piiFields = formData.getAll('pii_fields') as string[];
        const sensitiveFields = formData.getAll('sensitive_fields') as string[];
        const pageSize = formData.get('page_size') as string;
        dispatch(
          executeAccessor(
            selectedTenantID as string,
            accessor.id,
            executeAccessorContext,
            executeAccessorSelectorValues,
            piiFields,
            sensitiveFields,
            pageSize ? parseInt(pageSize, 10) : 100
          )
        );
      }}
    >
      {executeAccessorError && (
        <InlineNotification theme="alert">
          {executeAccessorError}
        </InlineNotification>
      )}
      <CardRow
        title="Input"
        tooltip={
          <>
            Test your accessor with different inputs to your selector and access
            policy context
          </>
        }
        collapsible
      >
        <Label htmlFor="context">
          Context
          <CodeEditor
            id="context"
            value={executeAccessorContext}
            onChange={(value: string) => {
              dispatch(changeExecuteAccessorContext(value));
            }}
            jsonExt
          />
        </Label>
        <Label htmlFor="selector_values">
          Selector values
          <CodeEditor
            id="selector_values"
            value={executeAccessorSelectorValues}
            onChange={(value: string) => {
              dispatch(changeExecuteAccessorSelectorValues(value));
            }}
            jsonExt
          />
        </Label>
      </CardRow>
      {accessorStats && (
        <CardRow
          title="Privacy analysis (optional)"
          tooltip={
            <>
              Optionally, compute K-anonymity and L-diversity measures for your
              accessor. Indicate which columns represent PII and which columns,
              if connected to PII, would represent sensitive information about
              your users.
            </>
          }
          collapsible
          isClosed
        >
          <fieldset className={PageCommon.carddetailsrow}>
            <div>
              <Label htmlFor="pii_fields">Select PII columns</Label>
              <ul>
                {accessor.columns.map(({ name: columnName }) => (
                  <li key={`pii_${columnName}`}>
                    <Label htmlFor={`pii_${columnName}`}>
                      <input
                        type="checkbox"
                        value={columnName}
                        name="pii_fields"
                      />{' '}
                      {columnName}
                    </Label>
                  </li>
                ))}
              </ul>
            </div>
            <div>
              <Label htmlFor="sensitive_fields">Select sensitive columns</Label>
              <ul>
                {accessor.columns.map(({ name: columnName }) => (
                  <li key={`sensitive_${columnName}`}>
                    <Label htmlFor={`sensitive_${columnName}`}>
                      <input
                        type="checkbox"
                        value={columnName}
                        name="sensitive_fields"
                      />{' '}
                      {columnName}
                    </Label>
                  </li>
                ))}
              </ul>
            </div>
          </fieldset>
        </CardRow>
      )}
      <CardRow
        title="Other options"
        tooltip={<>Choose page size and other options.</>}
        collapsible
        isClosed
      >
        <Label htmlFor="page_size">
          Page size
          <br />
          <TextInput
            type="number"
            size="medium"
            min={1}
            max={1500}
            defaultValue={100}
            name="page_size"
          />
        </Label>
      </CardRow>
      {executeAccessorResponse && (
        <CardRow
          title="Output"
          tooltip={
            <>
              This represents the data your accessor will return and includes
              some debugging information.
            </>
          }
          collapsible
        >
          <Label htmlFor="output">
            Output
            <br />
            <TextArea
              name="output"
              value={executeAccessorResponse.data}
              readOnly
            />
          </Label>
          <Label htmlFor="debug">
            Debug
            <br />
            <TextArea
              name="debug"
              value={JSON.stringify(executeAccessorResponse.debug, null, 2)}
              readOnly
            />
          </Label>
          {accessorStats && stats && (
            <div className={GlobalStyles['mt-6']}>
              <Heading
                size="3"
                headingLevel="3"
                className={GlobalStyles['mt-3']}
              >
                K-Anonymity: {stats.frequencies.kAnonymity}
              </Heading>
              <Accordion>
                <AccordionItem title="See breakdown" isOpen={false}>
                  <Table spacing="minimal">
                    <TableBody>
                      {Object.keys(stats.frequencies)
                        .filter((k: string) => k !== 'kAnonymity')
                        .sort()
                        .map((k: string) => (
                          <TableRow key={k}>
                            {k.split(' / ').map((val: string) => (
                              <TableCell>{val}</TableCell>
                            ))}
                            <TableCell>{stats.frequencies[k]}</TableCell>
                          </TableRow>
                        ))}
                    </TableBody>
                  </Table>
                </AccordionItem>
              </Accordion>

              {Object.keys(stats.uniqueness)?.length > 0 && (
                <>
                  <Heading
                    size="3"
                    headingLevel="3"
                    className={GlobalStyles['mt-3']}
                  >
                    L-Diversity
                  </Heading>
                  <Accordion>
                    {Object.keys(stats.uniqueness).map((field: string) => (
                      <AccordionItem
                        key={`ldiversity-${field}`}
                        title={`${field}: ${stats.uniqueness[field].lDiversity.value}`}
                        isOpen={false}
                      >
                        <Table spacing="minimal">
                          <TableBody>
                            {Object.keys(stats.uniqueness[field])
                              .filter((k: string) => k !== 'lDiversity')
                              .map((bucket: string) =>
                                Object.keys(
                                  stats.uniqueness[field][bucket]
                                ).map(
                                  (
                                    sensitiveVal: string,
                                    i: number,
                                    arr: string[]
                                  ) => (
                                    <>
                                      <TableRow
                                        key={`${bucket}${sensitiveVal}`}
                                      >
                                        {bucket
                                          .split(' / ')
                                          .map((col: string) => (
                                            <TableCell>
                                              {i === 0 ? col : ''}
                                            </TableCell>
                                          ))}
                                        <TableCell>{sensitiveVal}</TableCell>
                                      </TableRow>
                                      {i + 1 === arr.length ? (
                                        <TableRow
                                          key={`${sensitiveVal}${bucket}${arr?.length}`}
                                        >
                                          <TableCell
                                            colSpan={bucket.length + 1}
                                          />
                                        </TableRow>
                                      ) : (
                                        ''
                                      )}
                                    </>
                                  )
                                )
                              )}
                          </TableBody>
                        </Table>
                      </AccordionItem>
                    ))}
                  </Accordion>
                </>
              )}
            </div>
          )}
        </CardRow>
      )}
      <CardFooter>
        <Button theme="primary" type="submit">
          Test
        </Button>
      </CardFooter>
    </form>
  );
});

const AccessorRateLimiting = ({
  policy,
  globalPolicy,
}: {
  policy: AccessPolicy | undefined;
  globalPolicy: AccessPolicy | undefined;
}) => {
  let thresholds = globalPolicy?.thresholds;
  if (
    policy &&
    (policy.thresholds.max_executions > 0 ||
      policy.thresholds.max_results_per_execution > 0)
  ) {
    thresholds = policy.thresholds;
  }

  return (
    <>
      <CardRow
        title="Execution Rate Limiting"
        tooltip="Define limits to how frequently this accessor can be executed within a particular time frame."
        collapsible
      >
        {thresholds ? (
          <Table spacing="nowrap" className={Styles.executionratelimitingtable}>
            <TableHead>
              <TableRow>
                <TableRowHead>Enabled</TableRowHead>
                <TableRowHead>Max Executions</TableRowHead>
                <TableRowHead>Max Execution Window(s)</TableRowHead>
                <TableRowHead>Announce Max Execution Failure</TableRowHead>
              </TableRow>
            </TableHead>
            <TableBody>
              <TableRow>
                <TableCell>
                  {thresholds.max_executions > 0 ? 'Yes' : 'No'}
                </TableCell>
                <TableCell>
                  {thresholds.max_executions > 0
                    ? thresholds.max_executions
                    : '—'}
                </TableCell>
                <TableCell>
                  {thresholds.max_executions > 0
                    ? thresholds.max_execution_duration_seconds + ' seconds'
                    : '—'}
                </TableCell>
                <TableCell>
                  {thresholds.max_executions > 0
                    ? thresholds.announce_max_execution_failure
                    : '—'}
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        ) : (
          <Text>
            No rate limiting information is available for this accessor
          </Text>
        )}
      </CardRow>
      <CardRow
        title="Result Rate Limiting"
        tooltip="Define limits to the number of results this accessor can return."
        collapsible
      >
        {thresholds ? (
          <Table spacing="nowrap" className={Styles.resultratelimitingtable}>
            <TableHead>
              <TableRow>
                <TableRowHead>Enabled</TableRowHead>
                <TableRowHead>Max Results</TableRowHead>
                <TableRowHead>Announce Max Results Failure</TableRowHead>
              </TableRow>
            </TableHead>
            <TableBody>
              <TableRow>
                <TableCell>
                  {thresholds.max_results_per_execution > 0 ? 'Yes' : 'No'}
                </TableCell>
                <TableCell>
                  {thresholds.max_results_per_execution > 0
                    ? thresholds.max_results_per_execution
                    : '—'}
                </TableCell>
                <TableCell>
                  {thresholds.max_results_per_execution > 0
                    ? thresholds.announce_max_result_failure
                    : '—'}
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        ) : (
          <Text>
            No rate limiting information is available for this accessor
          </Text>
        )}
      </CardRow>
    </>
  );
};

const AccessorColumnRow = ({
  column,
  isNew,
  columnsToDelete,
  editMode,
  transformers,
  fetchingTransformers,
  accessPolicies,
  fetchingAccessPolicies,
  query,
  dispatch,
}: {
  column: AccessorColumn;
  isNew: boolean;
  columnsToDelete: Record<string, AccessorColumn>;
  editMode: boolean;
  transformers: PaginatedResult<Transformer> | undefined;
  fetchingTransformers: boolean;
  accessPolicies: PaginatedResult<AccessPolicy> | undefined;
  fetchingAccessPolicies: boolean;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  // cleanQuery is used, but eslint is confused
  /* eslint-disable @typescript-eslint/no-unused-vars */
  const cleanQuery = makeCleanPageLink(query);
  const queuedForDelete = !!columnsToDelete[column.id];
  return (
    <TableRow
      key={column.id}
      className={queuedForDelete ? PageCommon.queuedfordelete : ''}
    >
      <TableCell>
        <InputReadOnly>{column.name}</InputReadOnly>
      </TableCell>
      <TableCell>
        <InputReadOnly>
          {column.data_type_name + (column.is_array ? ' array' : '')}
        </InputReadOnly>
      </TableCell>
      <TableCell>
        {editMode ? (
          transformers && transformers.data && transformers.data?.length ? (
            <Select
              id="selected_transformer"
              name="selected_transformer"
              value={column.transformer_id}
              onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
                const val = e.target.value;
                dispatch(
                  changeSelectedTransformerForColumn(column.id, val, isNew)
                );
              }}
            >
              <option key="select_a_transformer">Select a transformer</option>
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
          )
        ) : (
          <InputReadOnly className={Styles.matchselectheight}>
            {column.transformer_id === NilUuid ? (
              <Link
                href={`/transformers/${DEFAULT_PASSTHROUGH_ACCESSOR_ID}/latest${cleanQuery}`}
                title="View, edit, or test this transformer"
              >
                PassthroughUnchangedData (default)
              </Link>
            ) : (
              <Link
                href={`/transformers/${column.transformer_id}/latest${cleanQuery}`}
                title="View, edit, or test this transformer"
              >
                {column.transformer_name}
              </Link>
            )}
          </InputReadOnly>
        )}
      </TableCell>
      <TableCell>
        {isTokenizingTransformer(transformers?.data, column.transformer_id) &&
          (editMode ? (
            accessPolicies?.data ? (
              <Select
                id="selected_token_access_policy"
                name="selected_token_access_policy"
                value={column.token_access_policy_id}
                onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
                  const val = e.target.value;
                  dispatch(
                    changeSelectedTokenAccessPolicyForColumn(
                      column.id,
                      val,
                      isNew
                    )
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
            )
          ) : (
            <InputReadOnly className={Styles.matchselectheight}>
              <Link
                href={`/accesspolicies/${column.token_access_policy_id}/latest${cleanQuery}`}
                title="View, edit, or test this access policy"
              >
                {column.token_access_policy_name}
              </Link>
            </InputReadOnly>
          ))}
      </TableCell>
      <TableCell>
        {editMode && (
          <IconButton
            icon={<IconDeleteBin />}
            onClick={() => {
              dispatch(toggleAccessorColumnForDelete(column));
            }}
            title="Remove column"
            aria-label="Remove column"
          />
        )}
      </TableCell>
    </TableRow>
  );
};
const ConnectedColumn = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  columnsToDelete: state.accessorColumnsToDelete,
  editMode: state.accessorColumnsEditMode,
  transformers: state.transformers,
  fetchingTransformers: state.fetchingTransformers,
  accessPolicies: state.allAccessPolicies,
  fetchingAccessPolicies: state.fetchingAllAccessPolicies,
  query: state.query,
}))(AccessorColumnRow);

const AccessorDetails = ({
  selectedTenant,
  selectedTenantID,
  accessor,
  modifiedAccessor,
  isDirty,
  purposes,
  fetchingPurposes,
  editMode,
  isSaving,
  saveError,
  saveSuccess,
  cleanQuery,
  userStoreColumns,
  columnsToAdd,
  dropdownValue,
  transformers,
  selectedAP,
  policy,
  tokenPolicy,
  globalAccessorPolicy,
  query,
  newTemplate,
  policyTemplateDialogIsOpen,
  isFetching,
  fetchError,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  selectedTenantID: string | undefined;
  accessor: Accessor | undefined;
  purposes: PaginatedResult<Purpose> | undefined;
  fetchingPurposes: boolean;
  modifiedAccessor: Accessor | undefined;
  isDirty: boolean;
  editMode: boolean;
  isSaving: boolean;
  saveError: string;
  saveSuccess: string;
  cleanQuery: string;

  userStoreColumns: Column[] | undefined;
  columnsToAdd: AccessorColumn[];
  dropdownValue: string;

  transformers: PaginatedResult<Transformer> | undefined;
  selectedAP: string;
  newTemplate: AccessPolicyTemplate | undefined;
  policy: AccessPolicy | undefined;
  tokenPolicy: AccessPolicy | undefined;
  globalAccessorPolicy: AccessPolicy | undefined;
  policyTemplateDialogIsOpen: boolean;
  query: URLSearchParams;
  isFetching: boolean;
  fetchError: string;
  dispatch: AppDispatch;
}) => {
  const addableColumns = userStoreColumns
    ?.filter(
      (column: Column) =>
        !modifiedAccessor?.columns.find(
          (col: AccessorColumn) => col.id === column.id
        ) && !columnsToAdd.find((col: AccessorColumn) => col.id === column.id)
    )
    .sort(columnNameAlphabeticalComparator);

  const dialog: HTMLDialogElement | null = document.getElementById(
    'createPolicyTemplateDialog'
  ) as HTMLDialogElement;

  useEffect(() => {
    if (selectedTenantID) {
      if (accessor && accessor.access_policy && accessor.access_policy.id) {
        dispatch(
          fetchAccessPolicy(selectedTenantID, accessor.access_policy.id)
        );
      } else {
        dispatch(getAccessPolicySuccess(blankPolicy()));
      }
    }
  }, [accessor, selectedTenantID, dispatch]);

  return (
    <>
      <form
        onSubmit={(e: React.FormEvent) => {
          e.preventDefault();
          if (selectedTenantID && modifiedAccessor) {
            dispatch(
              saveAccessorDetailsAndConfiguration(
                selectedTenantID,
                modifiedAccessor
              )
            );
          }
        }}
      >
        <div className={PageCommon.listviewtablecontrols}>
          <PageTitle title="Accessor details" itemName={accessor?.name} />
          <div className={PageCommon.listviewtablecontrolsToolTip}>
            <ToolTip>
              <>
                {
                  'View the metadata, columns, and policies relating to an accessor. '
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
            </ToolTip>
          </div>
          {modifiedAccessor && !modifiedAccessor.is_system && (
            <ButtonGroup
              className={PageCommon.listviewtablecontrolsButtonGroup}
            >
              {!editMode ? (
                selectedTenant?.is_admin && (
                  <>
                    <Button
                      theme="secondary"
                      size="small"
                      onClick={() => {
                        const executeAccessorDialog = document.getElementById(
                          'executeAccessorDialog'
                        );
                        if (executeAccessorDialog) {
                          (
                            executeAccessorDialog as HTMLDialogElement
                          ).showModal();
                        }
                      }}
                    >
                      Test accessor
                    </Button>
                    <Button
                      theme="primary"
                      size="small"
                      onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
                        e.preventDefault();
                        dispatch(toggleAccessorEditMode(true));
                        if (selectedTenantID) {
                          // TODO: check if we've already fetched this?
                          // probably safe to pull from cache
                          dispatch(fetchUserStoreConfig(selectedTenantID));
                          dispatch(fetchAllAccessPolicies(selectedTenantID));
                          dispatch(
                            fetchTransformers(
                              selectedTenantID,
                              new URLSearchParams(),
                              1000
                            )
                          );
                        }
                      }}
                    >
                      Edit Accessor
                    </Button>
                  </>
                )
              ) : (
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
                        dispatch(toggleAccessorEditMode(false));
                        if (selectedTenantID && accessor) {
                          dispatch(
                            fetchAccessor(
                              selectedTenantID,
                              accessor.id,
                              String(accessor.version)
                            )
                          );
                        }
                      }
                    }}
                  >
                    Cancel
                  </Button>
                  <Button
                    type="submit"
                    theme="primary"
                    size="small"
                    isLoading={isSaving}
                    disabled={isSaving || !isDirty}
                  >
                    Save Accessor
                  </Button>
                </>
              )}
            </ButtonGroup>
          )}
        </div>

        {accessor && modifiedAccessor ? (
          <Card
            isDirty={editMode}
            id="accessorDetails"
            lockedMessage={
              !selectedTenant?.is_admin ? 'You do not have edit access' : ''
            }
            detailview
          >
            {saveError && (
              <InlineNotification theme="alert">{saveError}</InlineNotification>
            )}
            {saveSuccess && (
              <InlineNotification theme="success">
                {saveSuccess}
              </InlineNotification>
            )}
            <CardRow
              title="Basic details"
              tooltip="View and edit this accessor's name and description."
              collapsible
            >
              <div className={PageCommon.carddetailsrow}>
                <Label>
                  Name
                  <br />
                  {editMode ? (
                    <TextInput
                      name="accessor_name"
                      id="accessor_name"
                      type="text"
                      pattern={VALID_NAME_PATTERN}
                      value={
                        modifiedAccessor ? modifiedAccessor.name : accessor.name
                      }
                      onChange={(e: React.ChangeEvent) => {
                        const val = (e.target as HTMLInputElement).value;
                        dispatch(
                          modifyAccessorDetails({
                            name: val,
                          })
                        );
                      }}
                    />
                  ) : (
                    <InputReadOnly className={Styles.nameText}>
                      {accessor.name}
                    </InputReadOnly>
                  )}
                </Label>
                <Label htmlFor="accessor_id">
                  ID
                  <br />
                  <TextShortener text={accessor.id} length={6} />
                </Label>

                <fieldset>
                  <Label htmlFor="livedataonly">Deleted data access</Label>
                  <Radio
                    id="livedataonly"
                    value={DurationType.Live}
                    name="liveDataSelect"
                    onClick={(e: React.ChangeEvent<HTMLInputElement>) => {
                      if (e.target.checked) {
                        dispatch(
                          modifyAccessorDetails({
                            data_life_cycle_state: e.target.value,
                          })
                        );
                      }
                    }}
                    checked={
                      modifiedAccessor?.data_life_cycle_state ===
                      DurationType.Live
                    }
                    disabled={!editMode}
                  >
                    Retrieve live data only (not soft-deleted)
                  </Radio>
                  <Radio
                    id="softdeletedata"
                    value={DurationType.SoftDeleted}
                    name="softDeleteDataSelect"
                    onClick={(e: React.ChangeEvent<HTMLInputElement>) => {
                      if (e.target.checked) {
                        dispatch(
                          modifyAccessorDetails({
                            data_life_cycle_state: e.target.value,
                          })
                        );
                      }
                    }}
                    checked={
                      modifiedAccessor?.data_life_cycle_state ===
                      DurationType.SoftDeleted
                    }
                    disabled={!editMode}
                  >
                    Retrieve soft-deleted data only (admin privileges required)
                  </Radio>
                </fieldset>
                <Label htmlFor="accessor_version">
                  Version
                  <br />
                  <InputReadOnly>{accessor.version}</InputReadOnly>
                </Label>
                <Label htmlFor="accessor_audit_logged">
                  Audit logged
                  <br />
                  {editMode ? (
                    <Checkbox
                      id="accessor_audit_logged"
                      checked={
                        modifiedAccessor
                          ? modifiedAccessor.is_audit_logged
                          : accessor.is_audit_logged
                      }
                      onChange={(e: React.ChangeEvent) => {
                        const { checked } = e.target as HTMLInputElement;
                        dispatch(
                          modifyAccessorDetails({
                            is_audit_logged: checked,
                          })
                        );
                      }}
                    />
                  ) : (
                    <InputReadOnly
                      type="checkbox"
                      isChecked={accessor.is_audit_logged}
                    />
                  )}
                </Label>

                <Label>
                  Use Search Index
                  <br />
                  {editMode ? (
                    <Checkbox
                      id="use_search_index"
                      checked={modifiedAccessor.use_search_index}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                        const { checked } = e.target;
                        dispatch(
                          modifyAccessorDetails({ use_search_index: checked })
                        );
                      }}
                    >
                      Index this accessor for search
                    </Checkbox>
                  ) : (
                    <InputReadOnly
                      type="checkbox"
                      isChecked={accessor.use_search_index}
                    />
                  )}
                </Label>
              </div>
              <Label className={GlobalStyles['mt-6']}>
                Description
                <br />
                {editMode && (
                  <TextArea
                    name="accessor_description"
                    value={
                      modifiedAccessor
                        ? modifiedAccessor.description
                        : accessor.description
                    }
                    placeholder="Add a description"
                    onChange={(e: React.ChangeEvent) => {
                      const val = (e.target as HTMLTextAreaElement).value;
                      dispatch(
                        modifyAccessorDetails({
                          description: val,
                        })
                      );
                    }}
                  />
                )}
                {!editMode && (
                  <InputReadOnly className={Styles.nameText}>
                    {accessor.description || 'N/A'}
                  </InputReadOnly>
                )}
              </Label>
            </CardRow>

            <CardRow
              title="Column configuration"
              collapsible
              tooltip="Configure the columns from which this accessor will retrieve
                  data, and how each column’s data will be transformed in the
                  accessor response."
            >
              {saveError && (
                <InlineNotification theme="alert">
                  {saveError}
                </InlineNotification>
              )}
              {saveSuccess && (
                <InlineNotification theme="success">
                  {saveSuccess}
                </InlineNotification>
              )}
              <Table
                spacing="nowrap"
                id="columnconfig"
                className={Styles.accessorcolumnstable}
              >
                <TableHead>
                  <TableRow key="accessorColumnHeadRow">
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
                  {editMode ? (
                    <>
                      {modifiedAccessor &&
                        modifiedAccessor.columns.map(
                          (column: AccessorColumn) => (
                            <ConnectedColumn
                              column={column}
                              isNew={false}
                              key={column.id}
                            />
                          )
                        )}
                    </>
                  ) : accessor.columns?.length || columnsToAdd?.length ? (
                    <>
                      {modifiedAccessor?.columns.map(
                        (column: AccessorColumn) => (
                          <ConnectedColumn
                            column={column}
                            isNew={false}
                            key={column.id}
                          />
                        )
                      )}
                      {columnsToAdd.map((column: AccessorColumn, i: number) => (
                        <ConnectedColumn
                          column={column}
                          isNew
                          key={column.id}
                        />
                      ))}
                    </>
                  ) : (
                    <TableRow key="accessorColumnEmpty">
                      <TableCell colSpan={editMode ? 4 : 3}>
                        No columns selected. You must add at least one column.
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
              {editMode && addableColumns?.length ? (
                <Select
                  id="selectUserStoreColumnToAdd"
                  name="select_column"
                  value={dropdownValue}
                  className={GlobalStyles['mt-3']}
                  onChange={(e: React.ChangeEvent) => {
                    const val = (e.target as HTMLSelectElement).value;
                    if (val) {
                      dispatch(addAccessorColumn(val));
                    }
                  }}
                >
                  <option key="no_selection" value="">
                    Select a column
                  </option>
                  {addableColumns.map((column: Column) => (
                    <option key={column.id} value={column.id}>
                      {column.name}
                    </option>
                  ))}
                </Select>
              ) : (
                <></>
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
                {editMode ? (
                  <TextInput
                    name="accessor_selector_config"
                    id="accessor_selector_config"
                    type="text"
                    placeholder="{id} = ANY(?) OR {phone_number} LIKE ?"
                    value={
                      modifiedAccessor
                        ? modifiedAccessor.selector_config.where_clause
                        : accessor.selector_config.where_clause
                    }
                    onChange={(e: React.ChangeEvent) => {
                      const val = (e.target as HTMLInputElement).value;
                      dispatch(
                        modifyAccessorDetails({
                          selector_config: { where_clause: val },
                        })
                      );
                    }}
                  />
                ) : (
                  <InputReadOnly monospace>
                    {accessor.selector_config.where_clause}
                  </InputReadOnly>
                )}
              </Label>
            </CardRow>
            <CardRow
              title="Access Policy"
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
              <TableTitle
                mainText="Global Policy"
                subText="(applied by default)"
              />
              {globalAccessorPolicy ? (
                <PolicyComposer
                  policy={globalAccessorPolicy}
                  changeAccessPolicyAction={() => {}}
                  readOnly
                  tableID="globalAccessorPolicyComponents"
                />
              ) : (
                <Text>Global policy has not been configured</Text>
              )}
              <br />
              <TableTitle
                mainText="Column Policies"
                subText="(applied to columns read by this accessor)"
              />
              <Label>
                Column policy override
                <br />
                {editMode ? (
                  <Checkbox
                    id="accessor_override_column_policies"
                    checked={
                      modifiedAccessor
                        ? modifiedAccessor.are_column_access_policies_overridden
                        : accessor.are_column_access_policies_overridden
                    }
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                      dispatch(
                        modifyAccessorDetails({
                          are_column_access_policies_overridden:
                            e.target.checked,
                        })
                      );
                    }}
                  >
                    Enable to ignore policies for individual columns and only
                    use the access policy configured for the accessor.
                  </Checkbox>
                ) : (
                  <InputReadOnly
                    type="checkbox"
                    isChecked={accessor.are_column_access_policies_overridden}
                  >
                    Enable to ignore policies for individual columns and only
                    use the access policy configured for the accessor.
                  </InputReadOnly>
                )}
              </Label>
              <Table
                id="columnPolicyOverrides"
                spacing="nowrap"
                className={Styles.columnpoliciestable}
              >
                <TableHead>
                  <TableRow>
                    <TableRowHead />
                    <TableRowHead>Name</TableRowHead>
                    <TableRowHead>Column</TableRowHead>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {modifiedAccessor.columns.map(
                    (c: AccessorColumn, i: number) => (
                      <TableRow key={`column_policies_${c.id}`}>
                        <TableCell>{i >= 1 ? 'and' : ''}</TableCell>
                        <TableCell>
                          <Link
                            href={`/accesspolicies/${c.default_access_policy_id}/latest${cleanQuery}`}
                            title="See details for this policy"
                          >
                            {c.default_access_policy_name}
                          </Link>
                        </TableCell>
                        <TableCell>
                          <Link
                            href={`/columns/${c.id}${cleanQuery}`}
                            title="See details for this column"
                          >
                            {c.name}
                          </Link>
                        </TableCell>
                      </TableRow>
                    )
                  )}
                </TableBody>
              </Table>
              <br />
              <TableTitle
                mainText="Manual Policies"
                subText="(manually applied to this accessor as a whole)"
              />
              <PolicyComposer
                policy={policy}
                changeAccessPolicyAction={modifyAccessPolicy}
                readOnly={!editMode}
                tableID="accessorManualPolicyComponents"
              />
            </CardRow>

            <AccessorRateLimiting
              policy={policy}
              globalPolicy={globalAccessorPolicy}
            />

            <CardRow
              title="Purposes of access"
              tooltip="Define the acceptable purposes for your users to access."
              collapsible
            >
              {purposes ? (
                <>
                  <Table
                    id="purposes"
                    spacing="nowrap"
                    className={Styles.accessorpurposestable}
                  >
                    <TableHead>
                      <TableRow key="head">
                        <TableRowHead>&nbsp;</TableRowHead>
                        <TableRowHead>
                          {(!editMode && accessor.purposes?.length) ||
                          (editMode && modifiedAccessor?.purposes?.length)
                            ? 'Purpose'
                            : ''}
                        </TableRowHead>
                        <TableRowHead>
                          {(!editMode && accessor.purposes?.length) ||
                          (editMode && modifiedAccessor?.purposes?.length)
                            ? 'Description'
                            : ''}
                        </TableRowHead>
                        <TableRowHead>
                          {editMode && modifiedAccessor?.purposes?.length
                            ? 'Delete'
                            : ''}
                        </TableRowHead>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {editMode ? (
                        modifiedAccessor &&
                        modifiedAccessor.purposes?.length > 0 ? (
                          modifiedAccessor.purposes.map((purpose, i) => (
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
                                {editMode && (
                                  <IconButton
                                    icon={<IconDeleteBin />}
                                    onClick={() => {
                                      dispatch(
                                        removePurposeFromAccessor(
                                          modifiedAccessor.purposes[i]
                                        )
                                      );
                                    }}
                                    title="Remove purpose"
                                    aria-label="Remove purpose"
                                  />
                                )}
                              </TableCell>
                            </TableRow>
                          ))
                        ) : (
                          <TableRow>
                            <TableCell colSpan={4}>
                              No purpose selected
                            </TableCell>
                          </TableRow>
                        )
                      ) : accessor?.purposes?.length > 0 ? (
                        accessor.purposes.map((purpose, i) => (
                          <TableRow key={`${purpose.id}-readonly`}>
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
                            <TableCell />
                          </TableRow>
                        ))
                      ) : (
                        <EmptyState
                          title="No purposes selected"
                          image={<IconLock2 size="large" />}
                        />
                      )}
                    </TableBody>
                  </Table>
                  <br />
                  {editMode &&
                    (getUnusedPurposes(purposes, modifiedAccessor?.purposes)
                      ?.length > 0 ? (
                      <Select
                        name="accessor_purpose"
                        value="Select a purpose"
                        onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
                          dispatch(addPurposeToAccessor(e.target.value));
                        }}
                        required
                      >
                        <option key="select">Select a purpose</option>
                        {getUnusedPurposes(
                          purposes,
                          modifiedAccessor?.purposes
                        ).map((purpose: Purpose) => (
                          <option
                            value={purpose.id}
                            key={`unusedpurpose-${purpose.id}`}
                          >
                            {purpose.name}
                          </option>
                        ))}
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
                    ))}
                </>
              ) : fetchingPurposes ? (
                <LoaderDots size="small" assistiveText="Loading purposes" />
              ) : (
                'Error fetching purposes'
              )}
            </CardRow>
          </Card>
        ) : isFetching ? (
          <Card
            title="..."
            description="View and edit this accessor's name and description"
          >
            <Text>Loading accessor ...</Text>
          </Card>
        ) : (
          <Card title="Error">
            <InlineNotification theme="alert">
              {fetchError || 'Something went wrong'}
            </InlineNotification>
          </Card>
        )}
      </form>
      <Dialog
        id="createPolicyTemplateDialog"
        title="Create Policy Template"
        description="Create a new template and add it to your composite policy. The template will be saved to your library for re-use later."
        fullPage
      >
        <DialogBody>
          {policyTemplateDialogIsOpen && (
            <PolicyTemplateForm
              editableTemplate={newTemplate || blankPolicyTemplate()}
              savedTemplate={undefined}
              saveTemplate={createAccessPolicyTemplateForAccessPolicy(() =>
                dialog.close()
              )}
              onCancel={() => {
                dialog?.close();
              }}
              searchParams={query}
            />
          )}
        </DialogBody>
      </Dialog>

      <ConnectedPolicyChooserDialog
        policy={policy}
        changeAccessPolicyAction={modifyAccessPolicy}
        createNewPolicyTemplateHandler={() => dialog.showModal()}
      />

      {accessor && (
        <Dialog id="executeAccessorDialog" title="Test accessor" fullPage>
          <DialogBody>
            <ExecuteAccessorDialog accessor={accessor} />
          </DialogBody>
        </Dialog>
      )}
    </>
  );
};
const ConnectedAccessorDetails = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  selectedTenantID: state.selectedTenantID,
  purposes: state.purposes,
  fetchingPurposes: state.fetchingPurposes,
  editMode: state.accessorDetailsEditMode,
  isSaving: state.savingAccessor,
  saveError: state.saveAccessorError,
  saveSuccess: state.saveAccessorSuccess,
  modifiedAccessor: state.modifiedAccessor,
  isDirty: state.modifiedAccessorIsDirty,
  userStoreColumns: state.userStoreColumns,
  columnsToAdd: state.accessorColumnsToAdd,
  dropdownValue: state.accessorAddColumnDropdownValue,
  transformers: state.transformers,
  selectedAP: state.selectedAccessPolicyForAccessor,
  policy: state.modifiedAccessPolicy,
  tokenPolicy: state.modifiedTokenAccessPolicy,
  globalAccessorPolicy: state.globalAccessorPolicy,
  newTemplate: state.policyTemplateToCreate,
  policyTemplateDialogIsOpen: state.policyTemplateDialogIsOpen,
  query: state.query,
  isFetching: state.fetchingAccessors,
  fetchError: state.fetchAccessorsError,
}))(AccessorDetails);

const AccessorPage = ({
  selectedTenantID,
  accessor,
  modifiedAccessor,
  accessors,
  isFetching,
  fetchError,
  featureFlags,
  location,
  query,
  routeParams,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  accessor: Accessor | undefined;
  modifiedAccessor: Accessor | undefined;
  accessors: PaginatedResult<Accessor> | undefined;
  isFetching: boolean;
  fetchError: string;
  featureFlags: FeatureFlags | undefined;
  location: URL;
  query: URLSearchParams;
  routeParams: Record<string, string>;
  dispatch: AppDispatch;
}) => {
  const cleanQuery = makeCleanPageLink(query);
  const { accessorID, version } = routeParams;
  const globalPolicies = featureIsEnabled(
    'global-access-policies',
    featureFlags
  );

  useEffect(() => {
    if (selectedTenantID && accessorID) {
      // the single accessor endpoint returns columns and full policies
      // whereas the accessors we get back on the index page
      // only have column IDs and policy ID
      // Therefore, we fetch on this page no matter what
      dispatch(fetchAccessor(selectedTenantID, accessorID, version || ''));
      dispatch(
        fetchPurposes(
          selectedTenantID as string,
          new URLSearchParams({ purposes_limit: '1000' })
        )
      );
      dispatch(executeAccessorReset());
    }
  }, [selectedTenantID, accessorID, version, query, dispatch]);
  useEffect(() => {
    if (selectedTenantID && globalPolicies) {
      dispatch(fetchGlobalAccessorPolicy(selectedTenantID));
    }
  }, [selectedTenantID, globalPolicies, dispatch]);

  return (
    <ConnectedAccessorDetails accessor={accessor} cleanQuery={cleanQuery} />
  );
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  accessors: state.accessors,
  accessor: state.selectedAccessor,
  modifiedAccessor: state.modifiedAccessor,
  isFetching: state.fetchingAccessors,
  fetchError: state.fetchAccessorsError,
  featureFlags: state.featureFlags,
  location: state.location,
  query: state.query,
  routeParams: state.routeParams,
}))(AccessorPage);
