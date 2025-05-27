import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  Dialog,
  GlobalStyles,
  IconButton,
  IconDeleteBin,
  InputReadOnly,
  Label,
  LoaderDots,
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
  ToolTip,
  DialogBody,
  TextShortener,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import { RootState, AppDispatch } from '../store';
import { featureIsEnabled } from '../util/featureflags';
import { VALID_NAME_PATTERN } from '../models/helpers';
import { FeatureFlags } from '../models/FeatureFlag';
import PaginatedResult from '../models/PaginatedResult';
import { SelectedTenant } from '../models/Tenant';
import Mutator, {
  MutatorColumn,
  isValidMutatorToUpdate,
} from '../models/Mutator';
import AccessPolicy, {
  AccessPolicyTemplate,
  blankPolicy,
  blankPolicyTemplate,
} from '../models/AccessPolicy';
import Transformer, { TransformType } from '../models/Transformer';
import {
  Column,
  columnNameAlphabeticalComparator,
} from '../models/TenantUserStoreConfig';
import {
  toggleMutatorEditMode,
  modifyMutatorDetails,
  toggleMutatorColumnForDelete,
  addMutatorColumn,
  changeSelectedNormalizerForColumn,
} from '../actions/mutators';
import {
  getAccessPolicySuccess,
  modifyAccessPolicy,
} from '../actions/tokenizer';
import { fetchUserStoreConfig } from '../thunks/userstore';
import { fetchMutator, saveMutatorDetails } from '../thunks/mutators';
import {
  createAccessPolicyTemplateForAccessPolicy,
  fetchAccessPolicy,
  fetchTransformers,
  fetchGlobalMutatorPolicy,
} from '../thunks/tokenizer';
import Link from '../controls/Link';
import { PageTitle } from '../mainlayout/PageWrap';
import PolicyComposer, { ConnectedPolicyChooserDialog } from './PolicyComposer';
import PolicyTemplateForm from './PolicyTemplateForm';

import PageCommon from './PageCommon.module.css';
import Styles from './AccessorDetailPage.module.css';

const getAddableDetailColumns = (
  userStoreColumns: Column[] | undefined,
  modifiedMutator: Mutator,
  columnsToAdd: MutatorColumn[]
): Column[] => {
  return userStoreColumns
    ? userStoreColumns
        ?.filter(
          (column: Column) =>
            column.is_system === false &&
            !modifiedMutator.columns.find(
              (col: MutatorColumn) => col.id === column.id
            ) &&
            !columnsToAdd.find((col: MutatorColumn) => col.id === column.id) &&
            !column.is_system
        )
        .sort(columnNameAlphabeticalComparator)
    : [];
};

const MutatorDetails = ({
  selectedTenant,
  mutator,
  modifiedMutator,
  editMode,
  saveSuccess,
  saveError,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  mutator: Mutator;
  modifiedMutator: Mutator;
  editMode: boolean;
  saveSuccess: string;
  saveError: string;
  dispatch: AppDispatch;
}) => {
  return (
    <CardRow
      title="Basic Details"
      isDirty={editMode}
      id="mutatorDetails"
      lockedMessage={
        !selectedTenant?.is_admin ? 'You do not have edit access' : ''
      }
      tooltip={<>View and edit this mutator's name and description.</>}
      collapsible
    >
      {saveError && (
        <InlineNotification theme="alert">{saveError}</InlineNotification>
      )}
      {saveSuccess && (
        <InlineNotification theme="success">{saveSuccess}</InlineNotification>
      )}
      <Label className={GlobalStyles['mt-3']}>
        Name
        <br />
        {editMode ? (
          <TextInput
            name="mutator_name"
            id="mutator_name"
            type="text"
            value={modifiedMutator ? modifiedMutator.name : mutator.name}
            pattern={VALID_NAME_PATTERN}
            onChange={(e: React.ChangeEvent) => {
              const val = (e.target as HTMLInputElement).value;
              dispatch(
                modifyMutatorDetails({
                  name: val,
                })
              );
            }}
          />
        ) : (
          <InputReadOnly>{mutator.name}</InputReadOnly>
        )}
      </Label>
      <div className={PageCommon.carddetailsrow}>
        <Label htmlFor="mutator_id">
          ID
          <br />
          <TextShortener text={mutator.id} length={6} id="mutator_id" />
        </Label>
        <Label htmlFor="mutator_version">
          Version
          <br />
          <InputReadOnly id="mutator_version">{mutator.version}</InputReadOnly>
        </Label>
      </div>

      <Label>
        Description
        <br />
        {editMode && (
          <TextArea
            name="mutator_description"
            value={
              modifiedMutator
                ? modifiedMutator.description
                : mutator.description
            }
            placeholder="Add a description"
            onChange={(e: React.ChangeEvent) => {
              const val = (e.target as HTMLTextAreaElement).value;
              dispatch(
                modifyMutatorDetails({
                  description: val,
                })
              );
            }}
          />
        )}
      </Label>
      {!editMode && <Text>{mutator.description || 'N/A'}</Text>}
    </CardRow>
  );
};
const ConnectedMutatorDetails = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  editMode: state.mutatorDetailsEditMode,
  saveSuccess: state.savingMutatorSuccess,
  saveError: state.savingMutatorError,
}))(MutatorDetails);

const MutatorDetailColumnRow = ({
  column,
  isNew,
  columnsToDelete,
  editMode,
  transformers,
  fetchingTransformers,
  query,
  dispatch,
}: {
  column: MutatorColumn;
  isNew: boolean;
  columnsToDelete: Record<string, MutatorColumn>;
  editMode: boolean;
  transformers: PaginatedResult<Transformer> | undefined;
  fetchingTransformers: boolean;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
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
        <InputReadOnly>{column.data_type_name}</InputReadOnly>
      </TableCell>
      <TableCell>
        <InputReadOnly type="checkbox" isChecked={column.is_array === true} />
      </TableCell>
      <TableCell>
        {editMode ? (
          transformers && transformers.data && transformers.data.length ? (
            <Select
              name="selected_normalizer"
              value={column.normalizer_id}
              onChange={(e: React.ChangeEvent) => {
                const val = (e.target as HTMLSelectElement).value;
                dispatch(
                  changeSelectedNormalizerForColumn(column.id, val, isNew)
                );
              }}
            >
              <option key="select_a_normalizer">Select a normalizer</option>
              {transformers.data
                .filter(
                  (normalizer: Transformer) =>
                    normalizer.transform_type === TransformType.PassThrough ||
                    normalizer.output_data_type.id === column.data_type_id
                )
                .map((normalizer: Transformer) => (
                  <option value={normalizer.id} key={normalizer.id}>
                    {normalizer.name}
                  </option>
                ))}
            </Select>
          ) : fetchingTransformers ? (
            <LoaderDots size="small" assistiveText="Loading normalizers" />
          ) : (
            <InlineNotification theme="alert">
              Error fetching normalizers
            </InlineNotification>
          )
        ) : (
          <InputReadOnly className={Styles.matchselectheight}>
            <Link
              href={`/transformers/${column.normalizer_id}/latest${cleanQuery}`}
              title="View, edit, or test this normalizer"
            >
              {column.normalizer_name}
            </Link>
          </InputReadOnly>
        )}
      </TableCell>
      <TableCell>
        {editMode && (
          <IconButton
            icon={<IconDeleteBin />}
            onClick={() => {
              dispatch(toggleMutatorColumnForDelete(column));
            }}
            title="Remove column"
            aria-label="Remove column"
          />
        )}
      </TableCell>
    </TableRow>
  );
};
const ConnectedDetailColumn = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  columnsToDelete: state.mutatorColumnsToDelete,
  editMode: state.mutatorColumnsEditMode,
  transformers: state.transformers,
  fetchingTransformers: state.fetchingTransformers,
  query: state.query,
}))(MutatorDetailColumnRow);

const MutatorDetailColumns = ({
  selectedTenant,
  modifiedMutator,
  editMode,
  userStoreColumns,
  columnsToAdd,
  dropdownValue,
  saveSuccess,
  saveError,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  modifiedMutator: Mutator;
  editMode: boolean;
  userStoreColumns: Column[] | undefined;
  columnsToAdd: MutatorColumn[];
  dropdownValue: string;
  saveSuccess: string;
  saveError: string;
  dispatch: AppDispatch;
}) => {
  const addableColumns = getAddableDetailColumns(
    userStoreColumns,
    modifiedMutator,
    columnsToAdd
  );

  return (
    <CardRow
      title="Columns"
      tooltip="Configure the columns to which this mutator will write data and a normalizer for each column."
      isDirty={editMode}
      id="mutatorColumns"
      lockedMessage={
        !selectedTenant?.is_admin ? 'You do not have edit access' : ''
      }
      collapsible
    >
      {saveError && (
        <InlineNotification theme="alert">{saveError}</InlineNotification>
      )}
      {saveSuccess && (
        <InlineNotification theme="success">{saveSuccess}</InlineNotification>
      )}
      <Table
        spacing="packed"
        id="columns"
        className={Styles.accessorcolumnstable}
      >
        <TableHead>
          <TableRow>
            <TableRowHead key="column_name">Name</TableRowHead>
            <TableRowHead key="column_type">Type</TableRowHead>
            <TableRowHead key="column_is_array">Array</TableRowHead>
            <TableRowHead key="column_normalizer">Normalizer</TableRowHead>
            <TableRowHead key="column_remove" />
          </TableRow>
        </TableHead>
        <TableBody>
          {modifiedMutator.columns.length || columnsToAdd.length ? (
            <>
              {modifiedMutator.columns.map((column: MutatorColumn) => (
                <ConnectedDetailColumn
                  key={column.id}
                  column={column}
                  isNew={false}
                />
              ))}
              {columnsToAdd.map((column: MutatorColumn) => (
                <ConnectedDetailColumn key={column.id} column={column} isNew />
              ))}
            </>
          ) : (
            <TableRow>
              <TableCell colSpan={editMode ? 4 : 3}>No columns</TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
      {editMode && addableColumns?.length ? (
        <Select
          id="selectUserStoreColumnToAdd"
          name="select_column"
          value={dropdownValue}
          onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
            const val = e.target.value;
            if (val) {
              dispatch(addMutatorColumn(val));
            }
          }}
          className={PageCommon['mt-3']}
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
  );
};
const ConnectedMutatorColumns = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  editMode: state.mutatorColumnsEditMode,
  userStoreColumns: state.userStoreColumns,
  columnsToAdd: state.mutatorColumnsToAdd,
  dropdownValue: state.mutatorAddColumnDropdownValue,
  saveSuccess: state.saveMutatorColumnsSuccess,
  saveError: state.saveMutatorColumnsError,
}))(MutatorDetailColumns);

const MutatorSelectorConfig = ({
  mutator,
  modifiedMutator,
  editMode,
  saveSuccess,
  saveError,
  dispatch,
}: {
  mutator: Mutator;
  modifiedMutator: Mutator;
  editMode: boolean;
  saveSuccess: string;
  saveError: string;
  dispatch: AppDispatch;
}) => {
  return (
    <CardRow
      title="Selector"
      tooltip={
        <>
          {
            'A selector is an SQL-like clause that specifies which records a mutator should write to. '
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
      id="mutatorSelectorConfig"
      collapsible
    >
      {saveError && (
        <InlineNotification theme="alert">{saveError}</InlineNotification>
      )}
      {saveSuccess && (
        <InlineNotification theme="success">{saveSuccess}</InlineNotification>
      )}
      <Label className={PageCommon['mt-3']}>
        Selector "where" clause
        <br />
        {editMode ? (
          <TextInput
            name="mutator_selector_config"
            id="mutator_selector_config"
            type="text"
            placeholder="{id} = ? OR {phone_number} LIKE ?"
            value={
              modifiedMutator
                ? modifiedMutator.selector_config.where_clause
                : mutator.selector_config.where_clause
            }
            onChange={(e: React.ChangeEvent) => {
              const val = (e.target as HTMLInputElement).value;
              dispatch(
                modifyMutatorDetails({
                  selector_config: { where_clause: val },
                })
              );
            }}
          />
        ) : (
          <InputReadOnly monospace>
            {mutator.selector_config.where_clause}
          </InputReadOnly>
        )}
      </Label>
    </CardRow>
  );
};
const ConnectedSelectorConfig = connect((state: RootState) => ({
  editMode: state.mutatorSelectorEditMode,
  saveSuccess: state.savingMutatorSuccess,
  saveError: state.savingMutatorError,
}))(MutatorSelectorConfig);

const MutatorPolicies = ({
  selectedTenant,
  selectedTenantID,
  mutator,
  editMode,
  policy,
  globalMutatorPolicy,
  newTemplate,
  saveSuccess,
  saveError,
  policyTemplateDialogIsOpen,
  featureFlags,
  query,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  selectedTenantID: string | undefined;
  mutator: Mutator;
  editMode: boolean;
  newTemplate: AccessPolicyTemplate | undefined;
  policy: AccessPolicy | undefined;
  globalMutatorPolicy: AccessPolicy | undefined;
  saveSuccess: string;
  saveError: string;
  policyTemplateDialogIsOpen: boolean;
  featureFlags: FeatureFlags | undefined;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  const dialog: HTMLDialogElement | null = document.getElementById(
    'createPolicyTemplateDialog'
  ) as HTMLDialogElement;
  const globalPolicies = featureIsEnabled(
    'global-access-policies',
    featureFlags
  );
  useEffect(() => {
    if (selectedTenantID) {
      dispatch(
        fetchTransformers(selectedTenantID, new URLSearchParams(), 1000)
      );
      if (mutator.access_policy && mutator.access_policy.id) {
        dispatch(fetchAccessPolicy(selectedTenantID, mutator.access_policy.id));
      } else {
        dispatch(getAccessPolicySuccess(blankPolicy()));
      }
    }
  }, [mutator, selectedTenantID, dispatch]);

  useEffect(() => {
    if (selectedTenantID && globalPolicies) {
      dispatch(fetchGlobalMutatorPolicy(selectedTenantID));
    }
  }, [selectedTenantID, globalPolicies, dispatch]);

  const cleanQuery = makeCleanPageLink(query);
  return (
    <CardRow
      title="Access Policy"
      tooltip={
        <>
          {
            'Select an access policy describing the circumstances in which writes via this mutator are allowed. '
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
      isDirty={editMode}
      id="mutatorPolicies"
      lockedMessage={
        !selectedTenant?.is_admin ? 'You do not have edit access' : ''
      }
      collapsible
    >
      {saveError && (
        <InlineNotification theme="alert">{saveError}</InlineNotification>
      )}
      {saveSuccess && (
        <InlineNotification theme="success">{saveSuccess}</InlineNotification>
      )}

      {globalMutatorPolicy && (
        <Text>
          All mutators evaluate the{' '}
          <Link
            href={`/accesspolicies/${globalMutatorPolicy.id}/latest${cleanQuery}`}
            title="View details for this access policy"
          >
            {globalMutatorPolicy.name}
          </Link>{' '}
          before evaluating policies specific to a given mutator, like those
          shown below.
        </Text>
      )}

      <PolicyComposer
        policy={policy}
        changeAccessPolicyAction={modifyAccessPolicy}
        readOnly={!editMode}
        tableID="mutatorManualPolicyComponents"
      />

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
              saveTemplate={createAccessPolicyTemplateForAccessPolicy(() =>
                dialog.close()
              )}
              onCancel={() => {
                dialog?.close();
              }}
              searchParams={query}
            />
          </DialogBody>
        )}
      </Dialog>
      <ConnectedPolicyChooserDialog
        policy={policy}
        changeAccessPolicyAction={modifyAccessPolicy}
        createNewPolicyTemplateHandler={() => dialog.showModal()}
      />
    </CardRow>
  );
};
const ConnectedMutatorPolicies = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  selectedTenantID: state.selectedTenantID,
  editMode: state.mutatorPoliciesEditMode,
  policy: state.modifiedAccessPolicy,
  globalMutatorPolicy: state.globalMutatorPolicy,
  newTemplate: state.policyTemplateToCreate,
  saveSuccess: state.saveMutatorPoliciesSuccess,
  saveError: state.saveMutatorPoliciesError,
  policyTemplateDialogIsOpen: state.policyTemplateDialogIsOpen,
  featureFlags: state.featureFlags,
  query: state.query,
}))(MutatorPolicies);

const MutatorDetailsPage = ({
  selectedTenantID,
  mutator,
  modifiedMutator,
  isFetching,
  fetchError,
  isSaving,
  isDirty,
  editMode,
  query,
  routeParams,
  columnsToAdd,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  mutator: Mutator | undefined;
  modifiedMutator: Mutator | undefined;
  isFetching: boolean;
  isSaving: boolean;
  isDirty: boolean;
  fetchError: string;
  editMode: boolean;
  query: URLSearchParams;
  routeParams: Record<string, string>;
  columnsToAdd: MutatorColumn[];
  dispatch: AppDispatch;
}) => {
  const { mutatorID, version } = routeParams;

  useEffect(() => {
    if (selectedTenantID && mutatorID) {
      // the single mutator endpoint returns columns and full policies
      // whereas the mutators we get back on the index page
      // only have column IDs and policy ID
      // Therefore, we fetch on this page no matter what
      dispatch(fetchMutator(selectedTenantID, mutatorID, version || ''));
    }
  }, [selectedTenantID, mutatorID, version, query, dispatch]);

  return (
    <form
      onSubmit={(e: React.FormEvent) => {
        e.preventDefault();

        if (selectedTenantID && modifiedMutator) {
          dispatch(saveMutatorDetails(selectedTenantID, modifiedMutator));
        }
      }}
    >
      <div className={PageCommon.listviewtablecontrols}>
        <PageTitle title="Mutator details" itemName={mutator?.name} />
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              {
                'View the metadata, columns, and policies relating to a mutator. '
              }
              <a
                href="https://docs.userclouds.com/docs/mutators-write-apis"
                title="UserClouds documentation for key concepts about mutators"
                target="new"
                className={PageCommon.link}
              >
                Learn more here.
              </a>
            </>
          </ToolTip>
        </div>

        {modifiedMutator && !modifiedMutator.is_system && (
          <ButtonGroup className={PageCommon.listviewtablecontrolsButtonGroup}>
            {!editMode ? (
              <Button
                theme="primary"
                size="small"
                onClick={() => {
                  dispatch(toggleMutatorEditMode(true));

                  // TODO: check if we've already fetched this?
                  // probably safe to pull from cache
                  dispatch(fetchUserStoreConfig(selectedTenantID as string));
                  dispatch(
                    fetchTransformers(
                      selectedTenantID as string,
                      new URLSearchParams(),
                      1000
                    )
                  );
                }}
              >
                Edit Mutator
              </Button>
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
                      dispatch(toggleMutatorEditMode(false));

                      if (selectedTenantID && mutator) {
                        dispatch(
                          fetchMutator(
                            selectedTenantID,
                            mutator.id,
                            String(mutator.version)
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
                  disabled={
                    !isValidMutatorToUpdate(
                      modifiedMutator,
                      columnsToAdd,
                      modifiedMutator.access_policy
                    ) ||
                    isSaving ||
                    !isDirty
                  }
                >
                  Save Mutator
                </Button>
              </>
            )}
          </ButtonGroup>
        )}
      </div>

      {mutator && modifiedMutator ? (
        <Card isDirty={editMode} id="mutatorDetails" detailview>
          <ConnectedMutatorDetails
            mutator={mutator}
            modifiedMutator={modifiedMutator}
          />
          <ConnectedMutatorColumns modifiedMutator={modifiedMutator} />
          <ConnectedSelectorConfig
            mutator={mutator}
            modifiedMutator={modifiedMutator}
          />
          <ConnectedMutatorPolicies mutator={mutator} />
        </Card>
      ) : isFetching ? (
        <Card
          title="..."
          description="View and edit this mutator's name and description"
        >
          <Text>Loading mutator...</Text>
        </Card>
      ) : (
        <Card title="Error">
          <InlineNotification theme="alert">
            {fetchError || 'Something went wrong'}
          </InlineNotification>
        </Card>
      )}
    </form>
  );
};

const MutatorDetailPage = connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  mutator: state.selectedMutator,
  modifiedMutator: state.modifiedMutator,
  isFetching: state.fetchingMutators,
  isSaving: state.savingMutator,
  isDirty: state.modifiedMutatorIsDirty,
  fetchError: state.fetchMutatorsError,
  query: state.query,
  routeParams: state.routeParams,
  editMode: state.mutatorDetailsEditMode,
  columnsToAdd: state.mutatorColumnsToAdd,
}))(MutatorDetailsPage);

export default connect((state: RootState) => ({
  location: state.location,
}))(MutatorDetailPage);
