import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  Checkbox,
  Dialog,
  InlineNotification,
  Label,
  Table,
  TableHead,
  TableBody,
  TableCell,
  TableRow,
  TableRowHead,
  Text,
  TextInput,
  ToolTip,
  DialogBody,
  DialogFooter,
} from '@userclouds/ui-component-lib';

import { RootState, AppDispatch } from '../store';
import { PolicySecret, POLICY_SECRET_PREFIX } from '../models/PolicySecret';
import { SelectedTenant } from '../models/Tenant';
import PaginatedResult from '../models/PaginatedResult';
import {
  bulkDeletePolicySecrets,
  deletePolicySecret,
  fetchPolicySecrets,
  saveNewPolicySecret,
} from '../thunks/tokenizer';
import {
  initializeNewPolicySecret,
  togglePolicySecretForDelete,
  togglePolicySecretsDeleteAll,
  modifyPolicySecret,
} from '../actions/tokenizer';
import Pagination from '../controls/Pagination';
import DeleteWithConfirmationButton from '../controls/DeleteWithConfirmationButton';
import PageCommon from './PageCommon.module.css';
import styles from './SecretsPage.module.css';

const NewSecretDialog = ({
  dialogID,
  selectedTenantID,
  modifiedSecret,
  saveError,
  dispatch,
}: {
  dialogID: string;
  selectedTenantID: string | undefined;
  modifiedSecret: PolicySecret | undefined;
  saveError: string;
  dispatch: AppDispatch;
}) => {
  return (
    <Dialog
      id={dialogID}
      title="Create New Secret"
      description="Add a new secret. Secrets are stored securely and can be used in your policies."
    >
      {modifiedSecret && selectedTenantID && (
        <>
          <DialogBody>
            {saveError && (
              <InlineNotification theme="alert">{saveError}</InlineNotification>
            )}

            <Label>
              Name
              <br />
              <TextInput
                name="secret_name"
                value={modifiedSecret.name}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                  dispatch(modifyPolicySecret({ name: e.target.value }));
                }}
              />
            </Label>
            <Label>
              Value
              <br />
              <TextInput
                name="secret_value"
                value={modifiedSecret.value}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                  dispatch(modifyPolicySecret({ value: e.target.value }));
                }}
              />
            </Label>
          </DialogBody>
          <DialogFooter>
            <ButtonGroup>
              <Button
                onClick={() => {
                  dispatch(
                    saveNewPolicySecret(
                      selectedTenantID,
                      modifiedSecret,
                      () => {
                        const dialog: HTMLDialogElement | null =
                          document.getElementById(
                            dialogID
                          ) as HTMLDialogElement;
                        dialog.close();
                      }
                    )
                  );
                }}
                theme="primary"
              >
                Save
              </Button>
              <Button
                onClick={() => {
                  const dialog: HTMLDialogElement | null =
                    document.getElementById(dialogID) as HTMLDialogElement;
                  dialog.close();
                }}
                theme="outline"
              >
                Cancel
              </Button>
            </ButtonGroup>
          </DialogFooter>
        </>
      )}
    </Dialog>
  );
};

// Exporting the dialog separately lets us place it where we want in the DOM
// and thus avoid nesting forms
export const ConnectedNewSecretDialog = connect((state: RootState) => ({
  modifiedSecret: state.modifiedPolicySecret,
  selectedTenantID: state.selectedTenantID,
  saveError: state.savePolicySecretError,
}))(NewSecretDialog);

const SecretRow = ({
  selectedTenantID,
  policySecret,
  policySecretDeleteQueue,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  policySecret: PolicySecret;
  policySecretDeleteQueue: string[];
  dispatch: AppDispatch;
}) => {
  const queuedForDelete = policySecretDeleteQueue.includes(policySecret.id);

  return (
    <TableRow
      key={`policySecret_row_${policySecret.id}`}
      className={
        (queuedForDelete ? PageCommon.queuedfordelete : '') +
        ' ' +
        PageCommon.listviewtablerow
      }
    >
      <TableCell>
        <Checkbox
          id={'delete' + policySecret.id}
          name={'delete' + policySecret.id}
          checked={queuedForDelete}
          onChange={() => {
            dispatch(togglePolicySecretForDelete(policySecret));
          }}
        />
      </TableCell>
      <TableCell>
        <Text>{policySecret.name}</Text>
      </TableCell>
      <TableCell>
        <Text>{new Date(policySecret.created * 1000).toLocaleString()}</Text>
      </TableCell>
      <TableCell className={PageCommon.listviewtabledeletecell} align="right">
        <DeleteWithConfirmationButton
          id="deleteSecretButton"
          message="Are you sure you want to delete this secret? This action is irreversible."
          onConfirmDelete={() => {
            if (selectedTenantID) {
              dispatch(deletePolicySecret(selectedTenantID, policySecret.id));
            }
          }}
          title="Delete Secret"
        />
      </TableCell>
    </TableRow>
  );
};
const ConnectedSecretRow = connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  policySecretDeleteQueue: state.policySecretDeleteQueue,
}))(SecretRow);

const SecretsTable = ({
  policySecrets,
  isFetching,
  error,
  deleteQueue,
  dispatch,
}: {
  policySecrets: PaginatedResult<PolicySecret> | undefined;
  isFetching: boolean;
  error: string;
  deleteQueue: string[];
  dispatch: AppDispatch;
}) => {
  return (
    <>
      {policySecrets ? (
        <Table
          spacing="packed"
          id="policySecretsTable"
          className={styles.policysecretstable}
        >
          <TableHead floating>
            <TableRow>
              <TableRowHead>
                <Checkbox
                  checked={
                    policySecrets.data.length > 0 &&
                    Object.keys(deleteQueue).length ===
                      policySecrets.data.length
                  }
                  disabled={policySecrets.data.length === 0}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    dispatch(togglePolicySecretsDeleteAll(e.target.checked));
                  }}
                />
              </TableRowHead>
              <TableRowHead key="secret_name_header">Name</TableRowHead>
              <TableRowHead key="secret_created_header">Created</TableRowHead>
              <TableRowHead key="delete_header" />
            </TableRow>
          </TableHead>
          <TableBody>
            {policySecrets.data.length ? (
              <>
                {policySecrets.data.map((policySecret: PolicySecret) => (
                  <ConnectedSecretRow
                    policySecret={policySecret}
                    key={policySecret.id}
                  />
                ))}
              </>
            ) : (
              <TableRow>
                <TableCell colSpan={4}>No secrets.</TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      ) : isFetching ? (
        <Text>Fetching secrets...</Text>
      ) : (
        <InlineNotification theme="alert">
          {error || 'Something went wrong'}
        </InlineNotification>
      )}
    </>
  );
};
const ConnectedSecretsTable = connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  isFetching: state.fetchingDataType,
  error: state.fetchingDataTypeError,
  deleteQueue: state.policySecretDeleteQueue,
}))(SecretsTable);

const Secrets = ({
  selectedTenant,
  policySecrets,
  fetching,
  saveSuccess,
  saveErrors,
  deleteQueue,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  policySecrets: PaginatedResult<PolicySecret> | undefined;
  fetching: boolean;
  saveSuccess: string;
  saveErrors: string[];
  deleteQueue: string[];
  dispatch: AppDispatch;
}) => {
  const dialogID = 'newSecretDialog';

  const deletePrompt = `Are you sure you want to delete ${
    deleteQueue.length
  } mutator${deleteQueue.length === 1 ? '' : 's'}? This action is irreversible.`;

  return (
    <>
      <ConnectedNewSecretDialog dialogID={dialogID} />
      <div className={PageCommon.listviewtablecontrols}>
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>Create secrets to be accessed from your policies.</ToolTip>
        </div>

        {selectedTenant?.is_admin && (
          <Button
            theme="primary"
            size="small"
            className={PageCommon.listviewtablecontrolsButton}
            onClick={() => {
              dispatch(initializeNewPolicySecret());
              const dialog: HTMLDialogElement | null = document.getElementById(
                dialogID
              ) as HTMLDialogElement;
              dialog.showModal();
            }}
          >
            Create Secret
          </Button>
        )}
      </div>

      <Card
        id="userstorePolicySecrets"
        lockedMessage={
          !selectedTenant?.is_admin ? 'You do not have edit access' : ''
        }
        listview
      >
        {!!saveErrors.length && (
          <InlineNotification theme="alert">
            {saveErrors.length === 1
              ? saveErrors[0]
              : `${saveErrors.length} errors occurred while saving your edits`}
          </InlineNotification>
        )}
        {saveSuccess && (
          <InlineNotification theme="success">{saveSuccess}</InlineNotification>
        )}
        {policySecrets ? (
          <>
            <div className={PageCommon.listviewpaginationcontrols}>
              <div className={PageCommon.listviewpaginationcontrolsdelete}>
                <DeleteWithConfirmationButton
                  id="deleteMutatorsButton"
                  message={deletePrompt}
                  onConfirmDelete={() => {
                    if (selectedTenant) {
                      dispatch(
                        bulkDeletePolicySecrets(selectedTenant.id, deleteQueue)
                      );
                    }
                  }}
                  title="Delete Mutators"
                  disabled={deleteQueue.length < 1}
                />
              </div>
              <Pagination
                prev={policySecrets?.prev}
                next={policySecrets?.next}
                isLoading={fetching}
                prefix={POLICY_SECRET_PREFIX}
              />
            </div>
            <ConnectedSecretsTable policySecrets={policySecrets} />
          </>
        ) : fetching ? (
          <Text>Fetching tenant User Store config...</Text>
        ) : (
          <InlineNotification theme="alert">
            Error fetching secrets
          </InlineNotification>
        )}
      </Card>
    </>
  );
};
const ConnectedSecrets = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  policySecrets: state.policySecrets,
  fetching: state.fetchingPolicySecrets,
  saveSuccess: state.saveUserStoreConfigSuccess,
  saveErrors: state.saveUserStoreConfigErrors,
  deleteQueue: state.policySecretDeleteQueue,
}))(Secrets);

const SecretsPage = ({
  selectedTenantID,
  query,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenantID) {
      dispatch(fetchPolicySecrets(selectedTenantID, query));
    }
  }, [selectedTenantID, query, dispatch]);

  return <ConnectedSecrets />;
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  query: state.query,
}))(SecretsPage);
