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
  TextArea,
  TextInput,
  ToolTip,
  DialogBody,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import { RootState, AppDispatch } from '../store';
import { redirect } from '../routing';
import { VALID_NAME_PATTERN } from '../models/helpers';
import PaginatedResult from '../models/PaginatedResult';
import {
  MutatorSavePayload,
  MutatorColumn,
  isValidMutator,
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
  loadCreateMutatorPage,
  modifyMutatorToCreate,
} from '../actions/mutators';
import {
  getAccessPolicySuccess,
  modifyAccessPolicy,
} from '../actions/tokenizer';
import { fetchUserStoreConfig } from '../thunks/userstore';
import { handleCreateMutator } from '../thunks/mutators';
import {
  createAccessPolicyTemplateForAccessPolicy,
  fetchAccessPolicies,
  fetchTransformers,
} from '../thunks/tokenizer';
import { PageTitle } from '../mainlayout/PageWrap';
import PolicyComposer, { ConnectedPolicyChooserDialog } from './PolicyComposer';
import PolicyTemplateForm from './PolicyTemplateForm';

import PageCommon from './PageCommon.module.css';
import Styles from './AccessPolicy.module.css';

const getAddableColumns = (
  columnsToAdd: MutatorColumn[],
  userStoreColumns: Column[] | undefined,
  mutator: MutatorSavePayload
): Column[] => {
  return (userStoreColumns || [])
    .filter(
      (column: Column) =>
        !mutator.columns.find(
          (col: MutatorColumn) => col.name === column.name
        ) &&
        !columnsToAdd.find((col: MutatorColumn) => col.id === column.id) &&
        !column.is_system
    )
    .sort(columnNameAlphabeticalComparator);
};

const MutatorColumnRow = ({
  mutator,
  column,
  transformers,
  fetchingTransformers,
  dispatch,
}: {
  mutator: MutatorSavePayload;
  column: MutatorColumn;
  transformers: PaginatedResult<Transformer> | undefined;
  fetchingTransformers: boolean;
  dispatch: AppDispatch;
}) => {
  return (
    <TableRow key={column.id}>
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
        {transformers && transformers.data && transformers.data.length ? (
          <Select
            name="selected_normalizer"
            value={column.normalizer_id}
            onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
              const val = e.target.value;
              dispatch(
                modifyMutatorToCreate({
                  columns: mutator.columns.map((col: MutatorColumn) => {
                    if (col.id === column.id) {
                      return {
                        ...col,
                        normalizer_id: val,
                      };
                    }
                    return col;
                  }),
                })
              );
            }}
          >
            <option value="">Select a normalizer</option>
            {transformers.data
              .filter(
                (transformer: Transformer) =>
                  transformer.transform_type === TransformType.PassThrough ||
                  transformer.output_data_type.id === column.data_type_id
              )
              .map((transformer: Transformer) => (
                <option value={transformer.id} key={transformer.id}>
                  {transformer.name}
                </option>
              ))}
          </Select>
        ) : fetchingTransformers ? (
          <LoaderDots size="small" assistiveText="Loading normalizers" />
        ) : (
          <InlineNotification theme="alert">
            Error fetching normalizers
          </InlineNotification>
        )}
      </TableCell>
      <TableCell>
        <IconButton
          icon={<IconDeleteBin />}
          onClick={() => {
            dispatch(
              modifyMutatorToCreate({
                columns: mutator.columns.filter(
                  (col: MutatorColumn) => col.name !== column.name
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
const ConnectedColumn = connect((state: RootState) => ({
  transformers: state.transformers,
  fetchingTransformers: state.fetchingTransformers,
}))(MutatorColumnRow);

const ConnectedCreateMutatorPage = ({
  selectedTenantID,
  mutator,
  userStoreColumns,
  columnsToAdd,
  dropdownValue,
  isSaving,
  saveError,
  query,
  newTemplate,
  policyTemplateDialogIsOpen,
  accessPolicy,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  mutator: MutatorSavePayload;
  userStoreColumns: Column[] | undefined;
  columnsToAdd: MutatorColumn[];
  dropdownValue: string;
  isSaving: boolean;
  saveError: string;
  query: URLSearchParams;
  newTemplate: AccessPolicyTemplate | undefined;
  policyTemplateDialogIsOpen: boolean;
  accessPolicy: AccessPolicy | undefined;
  dispatch: AppDispatch;
}) => {
  const cleanQuery = makeCleanPageLink(query);
  const addableColumns = getAddableColumns(
    columnsToAdd,
    userStoreColumns,
    mutator
  );

  const isDirty =
    mutator.columns.length ||
    mutator.name ||
    mutator.description ||
    mutator.selector_config.where_clause ||
    mutator.access_policy_id;
  const dialog: HTMLDialogElement | null = document.getElementById(
    'createPolicyTemplateDialog'
  ) as HTMLDialogElement;

  useEffect(() => {
    if (selectedTenantID) {
      dispatch(loadCreateMutatorPage());
      dispatch(fetchUserStoreConfig(selectedTenantID));
      dispatch(
        fetchAccessPolicies(
          selectedTenantID,
          new URLSearchParams({
            access_policies_limit: '100',
            versioned: 'false',
          }),
          false
        )
      );
      dispatch(
        fetchTransformers(selectedTenantID, new URLSearchParams(), 1000)
      );
      dispatch(getAccessPolicySuccess(blankPolicy()));
    }
  }, [selectedTenantID, dispatch]);

  return (
    <>
      <form
        onSubmit={(e: React.FormEvent) => {
          e.preventDefault();
          dispatch(handleCreateMutator());
        }}
      >
        <div className={PageCommon.listviewtablecontrols}>
          <PageTitle title="Create Mutator" itemName="New Mutator" />
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
                  redirect(`/mutators?${cleanQuery}`);
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
                isSaving || !isDirty || !isValidMutator(mutator, accessPolicy)
              }
            >
              Create Mutator
            </Button>
          </ButtonGroup>
        </div>

        <Card id="mutatorFormCard" detailview>
          {saveError && (
            <InlineNotification theme="alert">{saveError}</InlineNotification>
          )}
          <CardRow
            title="Basic Details"
            tooltip={<>Configure the basic details of your mutator.</>}
            collapsible
          >
            <Label className={GlobalStyles['mt-3']}>
              Name
              <br />
              <TextInput
                name="mutator_name"
                id="mutator_name"
                type="text"
                value={mutator.name}
                pattern={VALID_NAME_PATTERN}
                onChange={(e: React.ChangeEvent) => {
                  const val = (e.target as HTMLInputElement).value;
                  dispatch(
                    modifyMutatorToCreate({
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
                name="mutator_description"
                value={mutator.description}
                placeholder="Add a description"
                onChange={(e: React.ChangeEvent) => {
                  const val = (e.target as HTMLTextAreaElement).value;
                  dispatch(
                    modifyMutatorToCreate({
                      description: val,
                    })
                  );
                }}
              />
            </Label>
          </CardRow>

          <CardRow
            title="Columns"
            tooltip={
              <>
                Configure the columns to which this mutator will write data and
                a normalizer for each column.
              </>
            }
            collapsible
          >
            <Table
              id="mutatorColumns"
              spacing="packed"
              className={Styles.mutatorcolumnstable}
            >
              <TableHead>
                <TableRow>
                  <TableRowHead key="column_name">Name</TableRowHead>
                  <TableRowHead key="column_type">Type</TableRowHead>
                  <TableRowHead key="column_is_array">Array</TableRowHead>
                  <TableRowHead key="column_normalizer">
                    Normalizer
                  </TableRowHead>
                  <TableRowHead key="column_remove" />
                </TableRow>
              </TableHead>
              <TableBody>
                {mutator.columns.length ? (
                  <>
                    {mutator.columns.map((column: MutatorColumn) => (
                      <ConnectedColumn
                        mutator={mutator}
                        column={column}
                        key={column.id}
                      />
                    ))}
                  </>
                ) : (
                  <TableRow>
                    <TableCell colSpan={4}>No columns</TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
            {addableColumns?.length && (
              <Select
                id="selectUserStoreColumnToAdd"
                name="select_column"
                value={dropdownValue}
                onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
                  const val = e.target.value;
                  const matchingColumn = addableColumns.find(
                    (col: Column) => col.name === val
                  );
                  if (matchingColumn) {
                    dispatch(
                      modifyMutatorToCreate({
                        columns: [
                          ...mutator.columns,
                          {
                            id: matchingColumn.id,
                            name: matchingColumn.name,
                            is_array: matchingColumn.is_array,
                            data_type_name: matchingColumn.data_type.name,
                            data_type_id: matchingColumn.data_type.id,
                          },
                        ],
                      })
                    );
                  }
                }}
                className={GlobalStyles['mt-3']}
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
            collapsible
          >
            <Label>
              Selector "where" clause
              <br />
              <TextInput
                name="mutator_selector_config"
                id="mutator_selector_config"
                type="text"
                placeholder="{id} = ? OR {phone_number} LIKE ?"
                defaultValue={mutator.selector_config.where_clause}
                onChange={(e: React.ChangeEvent) => {
                  const val = (e.target as HTMLInputElement).value;
                  dispatch(
                    modifyMutatorToCreate({
                      selector_config: { where_clause: val },
                    })
                  );
                }}
              />
            </Label>
          </CardRow>
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
            collapsible
          >
            <PolicyComposer
              policy={accessPolicy}
              changeAccessPolicyAction={modifyAccessPolicy}
              tableID="mutatorManualPolicyComponents"
            />
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
        policy={accessPolicy}
        changeAccessPolicyAction={modifyAccessPolicy}
        createNewPolicyTemplateHandler={() => dialog.showModal()}
      />
    </>
  );
};

const CreateMutatorPage = connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  mutator: state.mutatorToCreate,
  userStoreColumns: state.userStoreColumns,
  columnsToAdd: state.mutatorColumnsToAdd,
  dropdownValue: state.mutatorAddColumnDropdownValue,
  isSaving: state.savingMutator,
  saveError: state.createMutatorError,
  query: state.query,
  accessPolicy: state.modifiedAccessPolicy,
  newTemplate: state.policyTemplateToCreate,
  policyTemplateDialogIsOpen: state.policyTemplateDialogIsOpen,
}))(ConnectedCreateMutatorPage);

export default connect((state: RootState) => ({
  location: state.location,
}))(CreateMutatorPage);
