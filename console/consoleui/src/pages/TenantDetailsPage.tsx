import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  CardFooter,
  Dialog,
  DialogBody,
  DialogFooter,
  GlobalStyles,
  IconButton,
  IconDeleteBin,
  IconEdit,
  InlineNotification,
  InputReadOnly,
  Label,
  Table,
  TableBody,
  TableHead,
  TableRow,
  TableRowHead,
  Text,
  TextInput,
  ToolTip,
  TableCell,
  TextShortener,
} from '@userclouds/ui-component-lib';

import { APIError } from '@userclouds/sharedui';
import { RootState, AppDispatch } from '../store';
import ServiceInfo from '../ServiceInfo';

import Tenant, { SelectedTenant } from '../models/Tenant';
import TenantPlexConfig from '../models/TenantPlexConfig';
import TenantURL from '../models/TenantURL';
import { SqlshimDatabase } from '../models/SqlshimDatabase';
import PaginatedResult from '../models/PaginatedResult';

import {
  saveTenant,
  handleDeleteTenant,
  getTenantURLs,
  getBatchModifyTenantURLPromises,
} from '../thunks/tenants';
import {
  getBatchUpdateDatabasePromises,
  getTenantSqlShims,
} from '../thunks/userstore';
import { fetchPlexConfig } from '../thunks/authn';
import {
  addTenantURL,
  modifyTenantName,
  toggleEditTenantMode,
  modifyTenantURL,
  setCurrentURL,
  setEditingIssuer,
  deleteTenantURL,
  updateTenantURL,
  createTenantURL,
  updateTenantError,
  setFetchingTenant,
} from '../actions/tenants';
import {
  addDatabase,
  deleteDatabase,
  resetDatabaseDialogState,
  setCurrentDatabase,
} from '../actions/userstore';
import {
  addExternalOIDCIssuer,
  deleteExternalOIDCIssuer,
  editExternalOIDCIssuer,
} from '../actions/authn';
import { validateTenantURL } from '../API/tenants';
import { saveTenantPlexConfig } from '../API/authn';
import { updateTenantDatabaseProxyPorts } from '../API/sqlshimdatabase';

import Link from '../controls/Link';
import { PageTitle } from '../mainlayout/PageWrap';
import PageCommon from './PageCommon.module.css';
import styles from './TenantDetailsPage.module.css';

import DatabaseDetailsForm from './DatabaseDetailsForm';

const DatabaseDetailsDialog = ({
  creatingDatabase,
  tenantDatabaseDialogIsOpen,
  dispatch,
}: {
  creatingDatabase: boolean;
  tenantDatabaseDialogIsOpen: boolean;
  dispatch: AppDispatch;
}) => {
  return (
    <Dialog
      id="tenantDatabaseDialog"
      open={tenantDatabaseDialogIsOpen}
      title={`${creatingDatabase ? 'Add' : 'Edit'} Database Connection`}
      isDismissable={false}
      className={styles.dialog}
      onClose={() => {
        dispatch(resetDatabaseDialogState());
      }}
    >
      <DialogBody>
        <DatabaseDetailsForm />
      </DialogBody>
    </Dialog>
  );
};

const ConnectedDatabaseDetailsDialog = connect((state: RootState) => ({
  tenantDatabaseDialogIsOpen: state.tenantDatabaseDialogIsOpen,
  creatingDatabase: state.creatingDatabase,
  currentDatabase: state.currentSqlshimDatabase,
}))(DatabaseDetailsDialog);

enum TenantURLStatus {
  Verified = 'verified',
  NotVerified = 'not_verified',
  Expired = 'expired',
}

const urlStatus = (tenantURL: TenantURL, tenant: Tenant) => {
  if (!tenantURL) {
    return { text: '', status: TenantURLStatus.NotVerified };
  }

  let text = 'DNS ownership verified. ';
  let status = TenantURLStatus.Verified;

  // TODO: better check here?
  // TODO: red text if expired
  if (tenantURL.certificate_valid_until !== '0001-01-01T00:00:00Z') {
    text +=
      'Certificate valid until ' +
      new Date(tenantURL.certificate_valid_until).toLocaleDateString() +
      '. ';
  } else {
    text += 'Certificate is not yet issued. ';
    status = TenantURLStatus.NotVerified;
  }

  if (tenantURL.active) {
    text += 'CNAME verified. ';
  } else if (tenant && tenant.tenant_url) {
    text +=
      'Domain is not currently CNAMEd to ' +
      tenant.tenant_url.replace(/https?:\/\//, '') +
      '. ';
    status = TenantURLStatus.NotVerified;
  }

  return { text, status };
};

const TenantURLDialog = ({
  creatingNew,
  errorMessage,
  isSaving,
  tenant,
  tenantURLDialogIsOpen,
  url,
  dispatch,
}: {
  creatingNew: boolean;
  errorMessage: string;
  isSaving: boolean;
  tenant: Tenant | undefined;
  tenantURLDialogIsOpen: boolean;
  url: TenantURL | undefined;
  dispatch: AppDispatch;
}) => {
  const urlState = urlStatus(url as TenantURL, tenant as Tenant);

  const onDialogClose = () => {
    if (creatingNew && url) {
      dispatch(deleteTenantURL(url.id));
    }
  };

  return (
    <Dialog
      id="tenantURLDialog"
      title="Tenant URL"
      open={tenantURLDialogIsOpen}
      isDismissable
      className={styles.dialog}
      onClose={onDialogClose}
    >
      <form
        onSubmit={(e: React.FormEvent) => {
          e.preventDefault();
          const dialog = (e.target as HTMLButtonElement).closest('dialog');
          if (dialog) {
            dialog.close();
          }
          url &&
            (creatingNew
              ? dispatch(createTenantURL(url))
              : dispatch(updateTenantURL(url)));
        }}
      >
        <DialogBody>
          {!creatingNew && errorMessage && (
            <InlineNotification theme="alert" className={GlobalStyles['mt-3']}>
              {errorMessage}
            </InlineNotification>
          )}
          <Label>
            Enter a valid HTTPS URL
            <br />
            <TextInput
              id={url?.id || 'new_url'}
              name="tenant_url"
              value={url?.tenant_url || ''}
              type="url"
              pattern="https://.*"
              required
              placeholder="Enter Valid HTTPS url (e.g. auth.yourcompany.com)"
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                dispatch(
                  modifyTenantURL({
                    id: url ? url.id : '',
                    tenant_url: e.target.value,
                  })
                );
              }}
            />
          </Label>

          {!creatingNew && (
            <>
              {url?.validated ? (
                <InlineNotification
                  className={styles.dialogNotification}
                  elementName="div"
                  theme={
                    urlState.status === TenantURLStatus.Verified
                      ? 'success'
                      : 'info'
                  }
                >
                  {urlState.text}
                </InlineNotification>
              ) : (
                <InlineNotification
                  className={styles.dialogNotification}
                  elementName="div"
                >
                  <div>
                    DNS ownership not validated.{' '}
                    {url?.dns_verifier !== '' &&
                      url?.dns_verifier !== undefined && (
                        <>
                          Please create a DNS TXT record at{' '}
                          <b>
                            _acme-challenge.
                            {url.tenant_url.replace(/https?:\/\//, '')}
                          </b>{' '}
                          with value <b>{url.dns_verifier}</b>
                        </>
                      )}
                  </div>
                  <Button
                    theme="outline"
                    onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
                      if (url) {
                        validateTenantURL(url);
                        const dialog = (e.target as HTMLButtonElement).closest(
                          'dialog'
                        );
                        if (dialog) {
                          dialog.close();
                        }
                        dispatch(getTenantURLs(url.tenant_id));
                      }
                    }}
                  >
                    Refresh
                  </Button>
                </InlineNotification>
              )}
            </>
          )}
        </DialogBody>
        <DialogFooter>
          <ButtonGroup>
            <Button
              theme="secondary"
              id="cancelURL"
              onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
                e.preventDefault();
                const dialog = (e.target as HTMLButtonElement).closest(
                  'dialog'
                );
                if (dialog) {
                  dialog.close();
                }
                onDialogClose();
              }}
            >
              Cancel
            </Button>
            <Button
              disabled={isSaving}
              isLoading={isSaving}
              theme="primary"
              type="submit"
              id="saveURL"
            >
              {creatingNew ? 'Add Custom URL' : 'Save'}
            </Button>
          </ButtonGroup>
        </DialogFooter>
      </form>
    </Dialog>
  );
};

const ConnectedTenantURLDialog = connect((state: RootState) => ({
  creatingNew: state.creatingNewTenantURL,
  editing: state.editingTenantURL,
  errorMessage: state.editingTenantURLError,
  featureFlags: state.featureFlags,
  isDirty: state.tenantURLIsDirty,
  isSaving: state.savingTenantURLs,
  tenant: state.selectedTenant,
  tenantURLDialogIsOpen: state.tenantURLDialogIsOpen,
  tenantURLs: state.tenantURLs || [],
  url: state.currentTenantURL,
}))(TenantURLDialog);

const TenantDatabases = ({
  currentDatabases,
  editing,
  modifiedDatabases,
  dispatch,
}: {
  currentDatabases: PaginatedResult<SqlshimDatabase> | undefined;
  editing: boolean;
  modifiedDatabases: SqlshimDatabase[];
  dispatch: AppDispatch;
}) => {
  const databases = editing
    ? modifiedDatabases
    : currentDatabases?.data?.length
      ? currentDatabases.data
      : [];

  return (
    <>
      <Table
        id="databases"
        spacing="packed"
        className={styles.tenantdatabasetable}
      >
        <TableHead floating key="db_head">
          <TableRow>
            <TableRowHead>Database Name</TableRowHead>
            <TableRowHead>Proxy Host Address</TableRowHead>
            <TableRowHead>Proxy Port</TableRowHead>
            <TableRowHead>&nbsp;</TableRowHead>
          </TableRow>
        </TableHead>
        <TableBody>
          {databases && databases.length > 0 ? (
            databases.map((database) => {
              return (
                <TableRow
                  key={database.id}
                  className={PageCommon.listviewtablerow}
                >
                  <TableCell>{database.name}</TableCell>
                  <TableCell>{database.proxy_host || '—'}</TableCell>
                  <TableCell>{database.proxy_port || '—'}</TableCell>
                  <TableCell
                    align="right"
                    className={PageCommon.listviewtabledeletecell}
                  >
                    {editing && (
                      <ButtonGroup>
                        <IconButton
                          icon={<IconEdit />}
                          onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
                            const dialog = (
                              e.target as HTMLButtonElement
                            ).closest('dialog');
                            if (dialog) {
                              dialog.close();
                            }
                            dispatch(setCurrentDatabase(database.id));
                            const tenantURLDialog = document.getElementById(
                              'tenantDatabaseDialog'
                            ) as HTMLDialogElement;
                            tenantURLDialog?.showModal();
                          }}
                          title="Edit Tenant Database"
                          aria-label="Edit Tenant Database"
                        />
                        <IconButton
                          icon={<IconDeleteBin />}
                          onClick={() => {
                            dispatch(deleteDatabase(database.id));
                          }}
                          title="Remove Tenant Database"
                          aria-label="Remove Tenant Database"
                        />
                      </ButtonGroup>
                    )}
                  </TableCell>
                </TableRow>
              );
            })
          ) : (
            <TableRow key="database_none">
              <TableCell colSpan={3}> None added (not required)</TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
      {editing && (
        <CardFooter>
          <Button
            theme="secondary"
            size="small"
            onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
              const dialog = (e.target as HTMLButtonElement).closest('dialog');
              if (dialog) {
                dialog.close();
              }
              const tenantDatabaseDialog = document.getElementById(
                'tenantDatabaseDialog'
              ) as HTMLDialogElement;
              dispatch(addDatabase());

              tenantDatabaseDialog?.showModal();
            }}
          >
            Add Database
          </Button>
        </CardFooter>
      )}
    </>
  );
};

const TenantURLRow = ({
  editing,
  url,
  dispatch,
}: {
  editing: boolean;
  url: TenantURL;
  dispatch: AppDispatch;
}) => {
  return (
    <TableRow key={url.id} className={PageCommon.listviewtablerow}>
      <TableCell>{url.tenant_url}</TableCell>
      <TableCell>
        {url.validated ? (
          'verified'
        ) : (
          <div className={styles.cellWithTooltip}>
            not verified{' '}
            {url?.dns_verifier !== '' && url?.dns_verifier !== undefined && (
              <ToolTip>
                <div>
                  Please create a DNS TXT record at{' '}
                  <b>
                    _acme-challenge.
                    {url.tenant_url.replace(/https?:\/\//, '')}
                  </b>{' '}
                  with value <b>{url.dns_verifier}</b>
                </div>
              </ToolTip>
            )}
          </div>
        )}
      </TableCell>
      <TableCell align="right" className={PageCommon.listviewtabledeletecell}>
        {editing && (
          <ButtonGroup>
            <IconButton
              icon={<IconEdit />}
              onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
                const dialog = (e.target as HTMLButtonElement).closest(
                  'dialog'
                );
                if (dialog) {
                  dialog.close();
                }
                dispatch(setCurrentURL(url));
                const tenantURLDialog = document.getElementById(
                  'tenantURLDialog'
                ) as HTMLDialogElement;
                tenantURLDialog?.showModal();
              }}
              title="Edit Tenant URL"
              aria-label="Edit Tenant URL"
            />
            <IconButton
              icon={<IconDeleteBin />}
              onClick={() => {
                dispatch(deleteTenantURL(url.id));
              }}
              title="Remove Tenant URL"
              aria-label="Remove Tenant URL"
            />
          </ButtonGroup>
        )}
      </TableCell>
    </TableRow>
  );
};

const ConnectedTenantURLRow = connect((state: RootState) => ({
  creatingNew: state.creatingNewTenantURL,
  editing: state.editingTenant,
  errorMessage: state.editingTenantURLError,
  isSaving: state.savingTenantURLs,
  tenant: state.selectedTenant,
  tenantURLs: state.tenantURLs || [],
}))(TenantURLRow);

const TenantURLs = ({
  editing,
  modifiedURLs,
  tenantURLs,
  dispatch,
}: {
  editing: boolean;
  modifiedURLs: TenantURL[];
  tenantURLs: TenantURL[];
  dispatch: AppDispatch;
}) => {
  return (
    <>
      <Table
        id="tenant_url_table"
        spacing="packed"
        className={styles.tenanturlstable}
      >
        <TableHead floating key="tenant_url_head">
          <TableRow>
            <TableRowHead>Tenant URL</TableRowHead>
            <TableRowHead>Verified</TableRowHead>
            <TableRowHead>&nbsp;</TableRowHead>
          </TableRow>
        </TableHead>
        <TableBody>
          {editing ? (
            modifiedURLs?.length > 0 ? (
              modifiedURLs?.map((tu) => (
                <ConnectedTenantURLRow url={tu} key={tu.id} />
              ))
            ) : (
              <TableRow key="tenant_url">
                <TableCell colSpan={3}> None added (not required)</TableCell>
              </TableRow>
            )
          ) : tenantURLs?.length > 0 ? (
            tenantURLs?.map((tu) => (
              <ConnectedTenantURLRow url={tu} key={tu.id} />
            ))
          ) : (
            <TableRow key="tenant_url">
              <TableCell colSpan={3}> None added (not required)</TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
      {editing && (
        <CardFooter>
          <Button
            theme="secondary"
            size="small"
            onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
              const dialog = (e.target as HTMLButtonElement).closest('dialog');
              if (dialog) {
                dialog.close();
              }
              const tenantURLDialog = document.getElementById(
                'tenantURLDialog'
              ) as HTMLDialogElement;
              dispatch(addTenantURL());
              tenantURLDialog?.showModal();
            }}
          >
            Add Custom URL
          </Button>
        </CardFooter>
      )}
    </>
  );
};

const ConnectedTenantURLs = connect((state: RootState) => ({
  creatingNew: state.creatingNewTenantURL,
  editing: state.editingTenant,
  errorMessage: state.editingTenantURLError,
  isSaving: state.savingTenantURLs,
  modifiedURLs: state.modifiedTenantUrls,
  tenantURLs: state.tenantURLs || [],
}))(TenantURLs);

const IssuerDetailsDialog = ({
  creatingNew,
  issuerIndex,
  modifiedConfig,
  plexConfig,
  tenantIssuerDialogIsOpen,
  dispatch,
}: {
  creatingNew: boolean;
  issuerIndex: number;
  modifiedConfig: TenantPlexConfig | undefined;
  plexConfig: TenantPlexConfig | undefined;
  tenantIssuerDialogIsOpen: boolean;
  dispatch: AppDispatch;
}) => {
  return (
    <Dialog
      id="tenantIssuerDialog"
      title="Issuer"
      open={tenantIssuerDialogIsOpen}
      isDismissable={false}
      className={styles.dialog}
    >
      <form
        id="issuerDialogForm"
        onSubmit={(e: React.FormEvent) => {
          e.preventDefault();
          const dialog = (e.target as HTMLButtonElement).closest('dialog');
          if (dialog) {
            // changes are in reducer so we don't need to save unless confirmed
            dialog.close();
          }
        }}
      >
        <DialogBody>
          <Label>
            Enter a trusted JWT Issuer URL
            <br />
            {modifiedConfig && issuerIndex >= 0 && (
              <TextInput
                id={issuerIndex}
                name="tenant_url"
                value={
                  issuerIndex >= 0 &&
                  modifiedConfig?.tenant_config.external_oidc_issuers &&
                  modifiedConfig?.tenant_config.external_oidc_issuers.length > 0
                    ? modifiedConfig.tenant_config.external_oidc_issuers[
                        issuerIndex
                      ]
                    : ''
                }
                type="url"
                required
                pattern="https://.*"
                placeholder="Enter Valid HTTPS url (e.g. auth.yourcompany.com)"
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                  dispatch(editExternalOIDCIssuer(e.target.value));
                }}
              />
            )}
          </Label>
        </DialogBody>
        <DialogFooter>
          <ButtonGroup>
            <Button
              theme="secondary"
              onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
                e.preventDefault();
                const dialog = (e.target as HTMLFormElement).closest('dialog');
                if (dialog) {
                  dialog.close();
                }
                if (creatingNew) {
                  dispatch(deleteExternalOIDCIssuer(issuerIndex));
                } else if (
                  plexConfig?.tenant_config.external_oidc_issuers &&
                  plexConfig?.tenant_config.external_oidc_issuers.length > 0
                )
                  dispatch(
                    editExternalOIDCIssuer(
                      plexConfig?.tenant_config.external_oidc_issuers[
                        issuerIndex
                      ]
                    )
                  );
              }}
            >
              Cancel
            </Button>

            <Button theme="primary" type="submit" id="saveIssuer">
              {creatingNew ? 'Add Issuer' : 'Save'}
            </Button>
          </ButtonGroup>
        </DialogFooter>
      </form>
    </Dialog>
  );
};

const ConnectedIssuerDetailsDialog = connect((state: RootState) => ({
  creatingNew: state.creatingIssuer,
  issuerIndex: state.editingIssuerIndex,
  modifiedConfig: state.modifiedPlexConfig,
  plexConfig: state.tenantPlexConfig,
  tenantIssuerDialogIsOpen: state.tenantIssuerDialogIsOpen,
}))(IssuerDetailsDialog);

const Issuers = ({
  editing,
  modifiedConfig,
  plexConfig,
  dispatch,
}: {
  editing: boolean;
  modifiedConfig: TenantPlexConfig | undefined;
  plexConfig: TenantPlexConfig | undefined;
  dispatch: AppDispatch;
}) => {
  const config = editing ? modifiedConfig : plexConfig;
  return (
    <>
      <Table
        id="trusted_issuers"
        spacing="packed"
        className={styles.trustedissuerstable}
      >
        <TableHead floating key="issuer_url_head">
          <TableRow>
            <TableRowHead>URL</TableRowHead>
            <TableRowHead key="issuer_actions">&nbsp;</TableRowHead>
          </TableRow>
        </TableHead>
        <TableBody>
          {config?.tenant_config.external_oidc_issuers &&
          config?.tenant_config.external_oidc_issuers?.length > 0 ? (
            config?.tenant_config.external_oidc_issuers.map(
              (issuer: string, index) => (
                <TableRow className={PageCommon.listviewtablerow} key={issuer}>
                  <TableCell>{issuer}</TableCell>
                  <TableCell
                    align="right"
                    className={PageCommon.listviewtabledeletecell}
                  >
                    {editing && (
                      <ButtonGroup>
                        <IconButton
                          icon={<IconEdit />}
                          onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
                            const dialog = (
                              e.target as HTMLButtonElement
                            ).closest('dialog');
                            if (dialog) {
                              dialog.close();
                            }

                            dispatch(setEditingIssuer(index));

                            const tenantIssuerDialog = document.getElementById(
                              'tenantIssuerDialog'
                            ) as HTMLDialogElement;
                            tenantIssuerDialog?.showModal();
                          }}
                          title="Edit Issuer"
                          aria-label="Edit Issuer"
                        />
                        <IconButton
                          icon={<IconDeleteBin />}
                          onClick={() => {
                            dispatch(deleteExternalOIDCIssuer(index));
                          }}
                          title="Remove issuer"
                          aria-label="Remove issuer"
                        />
                      </ButtonGroup>
                    )}
                  </TableCell>
                </TableRow>
              )
            )
          ) : (
            <TableRow key="issuer_url">
              <TableCell colSpan={2}> None added (not required)</TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
      {editing && (
        <CardFooter>
          <Button
            theme="secondary"
            size="small"
            onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
              const dialog = (e.target as HTMLButtonElement).closest('dialog');
              if (dialog) {
                dialog.close();
              }
              const tenantDatabaseDialog = document.getElementById(
                'tenantIssuerDialog'
              ) as HTMLDialogElement;
              dispatch(addExternalOIDCIssuer());
              tenantDatabaseDialog?.showModal();
            }}
          >
            Add Issuer
          </Button>
        </CardFooter>
      )}
    </>
  );
};

const ConnectedIssuers = connect((state: RootState) => ({
  editing: state.editingTenant,
  fetchError: state.fetchPlexConfigError,
  isDirty: state.plexConfigIsDirty,
  isFetching: state.fetchingPlexConfig,
  modifiedConfig: state.modifiedPlexConfig,
  plexConfig: state.tenantPlexConfig,
  tenant: state.selectedTenant,
}))(Issuers);

export const loadTenantPage =
  (tenantID: string) => async (dispatch: AppDispatch) => {
    const promises = [
      dispatch(getTenantURLs(tenantID)),
      dispatch(getTenantSqlShims(tenantID)),
      dispatch(fetchPlexConfig(tenantID)),
    ];

    dispatch(setFetchingTenant(true));
    await Promise.all(promises).then(() => {
      dispatch(toggleEditTenantMode(false));
      dispatch(setFetchingTenant(false));
    });
  };

export const saveAllChanges =
  (
    modifiedTenant: Tenant | undefined,
    selectedTenant: SelectedTenant,
    plexConfigIsDirty: boolean,
    modifiedConfig: any,
    currentDatabases: SqlshimDatabase[],
    modifiedDatabases: SqlshimDatabase[],
    tenantURLsToCreate: TenantURL[],
    tenantURLsToUpdate: TenantURL[],
    tenantURLsToDelete: string[],
    tenantID: string
  ) =>
  async (dispatch: AppDispatch) => {
    try {
      const promises: Promise<any>[] = [];

      if (
        selectedTenant &&
        modifiedTenant &&
        modifiedTenant?.name !== selectedTenant.name
      ) {
        promises.push(dispatch(saveTenant(modifiedTenant, selectedTenant)));
      }

      if (plexConfigIsDirty && modifiedConfig) {
        promises.push(saveTenantPlexConfig(tenantID, modifiedConfig));
      }

      if (
        selectedTenant &&
        JSON.stringify(modifiedDatabases) !== JSON.stringify(currentDatabases)
      ) {
        promises.push(
          Promise.all(
            getBatchUpdateDatabasePromises(
              selectedTenant.id,
              currentDatabases || [],
              modifiedDatabases || []
            )
          ).then(() => {
            return updateTenantDatabaseProxyPorts(selectedTenant.id);
          })
        );
      }

      if (
        selectedTenant &&
        (tenantURLsToCreate.length > 0 ||
          tenantURLsToUpdate.length > 0 ||
          tenantURLsToDelete.length > 0)
      ) {
        promises.push(
          ...getBatchModifyTenantURLPromises(
            selectedTenant.id,
            tenantURLsToCreate,
            tenantURLsToUpdate,
            tenantURLsToDelete
          )
        );
      }

      await Promise.all(promises).then(() => {
        dispatch(loadTenantPage(selectedTenant.id));
      });
    } catch (error) {
      dispatch(updateTenantError(error as APIError));
    }
  };

const TenantDetailsPage = ({
  companyID,
  currentDatabases,
  deleteError,
  deletingTenant,
  editingTenant,
  fetchingDatabase,
  fetchingIssuers,
  fetchingSelectedTenant,
  fetchingTenants,
  fetchingTenantURLs,
  fetchingTenantURLsError,
  modifiedConfig,
  modifiedDatabases,
  modifiedTenant,
  plexConfig,
  plexConfigIsDirty,
  plexFetchError,
  plexIsSaving,
  routeParams,
  saveError,
  saveSuccess,
  savingDatabase,
  savingTenant,
  savingTenantURLs,
  selectedTenant,
  serviceInfo,
  tenantURLs,
  tenantURLsToCreate,
  tenantURLsToDelete,
  tenantURLsToUpdate,
  dispatch,
}: {
  companyID: string | undefined;
  currentDatabases: PaginatedResult<SqlshimDatabase> | undefined;
  deleteError: string;
  deletingTenant: boolean;
  editingTenant: boolean;
  fetchingDatabase: boolean;
  fetchingIssuers: boolean;
  fetchingSelectedTenant: boolean;
  fetchingTenants: boolean;
  fetchingTenantURLs: boolean;
  fetchingTenantURLsError: string;
  modifiedConfig: TenantPlexConfig | undefined;
  modifiedDatabases: SqlshimDatabase[];
  modifiedTenant: Tenant | undefined;
  plexConfig: TenantPlexConfig | undefined;
  plexConfigIsDirty: boolean;
  plexFetchError: string;
  plexIsSaving: boolean;
  routeParams: Record<string, string>;
  saveError: string;
  saveSuccess: string;
  savingDatabase: boolean;
  savingTenant: boolean;
  savingTenantURLs: boolean;
  selectedTenant: SelectedTenant | undefined;
  serviceInfo: ServiceInfo | undefined;
  tenantURLs: TenantURL[] | undefined;
  tenantURLsToCreate: TenantURL[];
  tenantURLsToDelete: string[];
  tenantURLsToUpdate: TenantURL[];
  dispatch: AppDispatch;
}) => {
  const { tenantID } = routeParams;

  useEffect(() => {
    if (selectedTenant && tenantID) {
      if (tenantID === 'current' || tenantID === selectedTenant.id) {
        dispatch(loadTenantPage(selectedTenant.id));
      }
    }
  }, [selectedTenant, companyID, tenantID, dispatch]);

  return (
    <>
      <form
        onSubmit={(e: React.FormEvent) => {
          e.preventDefault();

          selectedTenant &&
            dispatch(
              saveAllChanges(
                modifiedTenant,
                selectedTenant,
                plexConfigIsDirty,
                modifiedConfig,
                currentDatabases ? currentDatabases.data : [],
                modifiedDatabases,
                tenantURLsToCreate,
                tenantURLsToUpdate,
                tenantURLsToDelete,
                tenantID
              )
            );
        }}
      >
        <div className={PageCommon.listviewtablecontrols}>
          <PageTitle
            title={editingTenant ? 'Edit Tenant' : 'Tenant Details'}
            itemName={
              selectedTenant
                ? selectedTenant.name
                : fetchingTenants || fetchingSelectedTenant
                  ? '...'
                  : 'No tenants'
            }
          />

          <div className={PageCommon.listviewtablecontrolsToolTip}>
            <ToolTip>
              <>
                A tenant is a single instance of UserClouds's tech (APIs, user
                store etc). Each tenant can handle multiple applications.
                <a
                  href="https://docs.userclouds.com/docs/create-your-tenant"
                  title="UserClouds documentation"
                  target="new"
                  className={PageCommon.link}
                >
                  Learn more here.
                </a>
              </>
            </ToolTip>
          </div>
          <ButtonGroup className={PageCommon.listviewtablecontrolsButtonGroup}>
            {editingTenant && (
              <>
                <Button
                  theme="secondary"
                  size="small"
                  isLoading={savingTenant || savingTenantURLs}
                  onClick={() => {
                    dispatch(toggleEditTenantMode(false));
                  }}
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  size="small"
                  theme="primary"
                  isLoading={savingTenant || savingTenantURLs}
                  disabled={false} // todo
                  id="savePage"
                >
                  Save
                </Button>
              </>
            )}
            {selectedTenant && selectedTenant.is_admin && !editingTenant && (
              <>
                <Button
                  theme="primary"
                  size="small"
                  isLoading={savingTenant || savingTenantURLs}
                  onClick={() => {
                    dispatch(toggleEditTenantMode(true));
                  }}
                >
                  Edit Settings
                </Button>
              </>
            )}
          </ButtonGroup>
        </div>
        <Card isDirty={editingTenant} detailview>
          {selectedTenant ? (
            <>
              {saveSuccess && (
                <InlineNotification theme="success">
                  {saveSuccess}
                </InlineNotification>
              )}
              {saveError && (
                <InlineNotification theme="alert">
                  {saveError}
                </InlineNotification>
              )}
              {deleteError && (
                <InlineNotification theme="alert">
                  {deleteError}
                </InlineNotification>
              )}
              <input type="hidden" name="id" value={selectedTenant.id} />
              <input
                type="hidden"
                name="company_id"
                value={selectedTenant.company_id}
              />
              <CardRow
                title="Basic details"
                tooltip="Configure the name and region of your tenant."
                collapsible
              >
                <div className={PageCommon.carddetailsrow}>
                  <Label className={GlobalStyles['mt-6']}>
                    Name
                    <br />
                    {editingTenant && modifiedTenant ? (
                      <TextInput
                        name="name"
                        value={modifiedTenant.name}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                          dispatch(modifyTenantName(e.target.value));
                        }}
                      />
                    ) : (
                      <InputReadOnly>{selectedTenant.name}</InputReadOnly>
                    )}
                  </Label>

                  <Label className={GlobalStyles['mt-6']} htmlFor="tenant_id">
                    ID
                    <br />
                    <TextShortener
                      text={selectedTenant.id}
                      length={6}
                      id="tenant_id"
                    />
                  </Label>

                  <Label className={GlobalStyles['mt-6']}>
                    User Regions
                    <br />
                    {selectedTenant.user_regions?.length ? (
                      selectedTenant.user_regions?.map((region) => (
                        <InputReadOnly key={region}>{region}</InputReadOnly>
                      ))
                    ) : (
                      <InputReadOnly>None</InputReadOnly>
                    )}
                  </Label>
                  <Label
                    className={GlobalStyles['mt-6']}
                    title="This setting cannot be changed once the tenant is created"
                    htmlFor="use_orgs"
                  >
                    Use organizations
                    <InputReadOnly
                      name="use_orgs"
                      type="checkbox"
                      isChecked={selectedTenant.use_organizations}
                      isLocked
                    />
                  </Label>
                  <Label className={GlobalStyles['mt-6']} htmlFor="tenant_url">
                    URL <i>(SDK or HTTP connections)</i>
                    <br />
                    <TextShortener
                      text={selectedTenant.tenant_url}
                      length={60}
                      id="tenant_url"
                    />
                  </Label>
                </div>
              </CardRow>
              {selectedTenant && selectedTenant.is_admin && (
                <CardRow
                  title="Database Connections"
                  tooltip={
                    <>
                      Specify additional URLs for your tenant. When you add a
                      custom tenant URL (eg. <b>auth.yourcompany.com</b>), you
                      should create a CNAME record that points to{' '}
                      <b>
                        {selectedTenant?.tenant_url?.replace(/https?:\/\//, '')}
                      </b>
                      .
                    </>
                  }
                  collapsible
                >
                  {selectedTenant &&
                  (currentDatabases || modifiedDatabases) &&
                  !fetchingDatabase &&
                  !savingDatabase ? (
                    <TenantDatabases
                      editing={editingTenant}
                      currentDatabases={currentDatabases}
                      modifiedDatabases={modifiedDatabases}
                      dispatch={dispatch}
                    />
                  ) : fetchingDatabase || savingDatabase ? (
                    'Loading ...'
                  ) : (
                    <Text>Error fetching database</Text>
                  )}
                </CardRow>
              )}
              {selectedTenant && selectedTenant.is_admin && (
                <CardRow
                  title="Custom Domain"
                  tooltip={
                    <>
                      Specify additional URLs for your tenant. When you add a
                      custom tenant URL (eg. <b>auth.yourcompany.com</b>), you
                      should create a CNAME record that points to{' '}
                      <b>
                        {selectedTenant?.tenant_url?.replace(/https?:\/\//, '')}
                      </b>
                      .
                    </>
                  }
                  collapsible
                >
                  {selectedTenant &&
                  tenantURLs &&
                  !fetchingTenantURLs &&
                  !savingTenantURLs ? (
                    <ConnectedTenantURLs />
                  ) : fetchingTenantURLs || savingTenantURLs ? (
                    'Loading ...'
                  ) : (
                    <Text>{fetchingTenantURLsError}</Text>
                  )}
                </CardRow>
              )}
              {selectedTenant && selectedTenant.is_admin ? (
                <CardRow
                  title="Trusted Issuer"
                  tooltip={
                    <>
                      Configure Social and other 3rd party OIDC Identity
                      Providers for Plex.
                      <a
                        href="https://docs.userclouds.com/docs/introduction-1"
                        title="UserClouds documentation for key concepts in authentication"
                        target="new"
                        className={PageCommon.link}
                      >
                        Learn more here.
                      </a>
                    </>
                  }
                  collapsible
                >
                  {selectedTenant &&
                  plexConfig &&
                  !fetchingIssuers &&
                  !plexIsSaving ? (
                    <ConnectedIssuers />
                  ) : fetchingIssuers || plexIsSaving ? (
                    'Loading ...'
                  ) : (
                    <Text>Error fetching database</Text>
                  )}
                </CardRow>
              ) : fetchingIssuers ? (
                'Loading ...'
              ) : (
                <Text>{plexFetchError}</Text>
              )}
              {serviceInfo?.uc_admin && editingTenant && (
                <Button
                  theme="dangerous"
                  className={PageCommon.listviewtablecontrolsButton}
                  isLoading={deletingTenant}
                  onClick={() => {
                    if (
                      window.confirm(
                        'Are you sure you want to delete this tenant? This cannot be undone.'
                      )
                    ) {
                      dispatch(
                        handleDeleteTenant(
                          selectedTenant.company_id,
                          selectedTenant.id
                        )
                      );
                    }
                  }}
                >
                  Delete tenant
                </Button>
              )}
            </>
          ) : fetchingTenants || fetchingSelectedTenant ? (
            'Loading ...'
          ) : (
            <Text>
              This company doesn't have any tenants yet. You can{' '}
              <Link href={`/tenants/create?company_id=${companyID}`}>
                create one now.
              </Link>
            </Text>
          )}
        </Card>
      </form>
      {selectedTenant && (
        <>
          <ConnectedTenantURLDialog />
          <ConnectedDatabaseDetailsDialog />
          <ConnectedIssuerDetailsDialog />
        </>
      )}
    </>
  );
};

export default connect((state: RootState) => {
  return {
    companyID: state.selectedCompanyID,
    currentDatabase: state.currentSqlshimDatabase,
    currentDatabases: state.sqlShimDatabases,
    databaseIsDirty: state.databaseIsDirty,
    deleteError: state.deleteTenantError,
    deletingTenant: state.deletingTenant,
    editingTenant: state.editingTenant,
    featureFlags: state.featureFlags,
    fetchingDatabase: state.fetchingSqlshimDatabase,
    fetchingIssuers: state.fetchingPlexConfig,
    fetchingSelectedTenant: state.fetchingSelectedTenant,
    fetchingTenants: state.fetchingTenants,
    fetchingTenantURLs: state.fetchingTenantURLs,
    fetchingTenantURLsError: state.fetchingTenantURLsError,
    location: state.location,
    modifiedConfig: state.modifiedPlexConfig,
    modifiedDatabase: state.modifiedSqlshimDatabase,
    modifiedDatabases: state.modifiedSqlShimDatabases,
    modifiedTenant: state.modifiedTenant,
    oidcProvider: state.oidcProvider,
    plexConfig: state.tenantPlexConfig,
    plexConfigIsDirty: state.plexConfigIsDirty,
    plexFetchError: state.fetchPlexConfigError,
    plexIsSaving: state.savingPlexConfig,
    plexSaveError: state.savePlexConfigError,
    plexSaveSuccess: state.savePlexConfigSuccess,
    query: state.query,
    saveError: state.saveTenantError,
    saveSuccess: state.saveTenantSuccess,
    savingDatabase: state.savingSqlshimDatabase,
    savingTenant: state.savingTenant,
    savingTenantURLs: state.savingTenantURLs,
    selectedTenant: state.selectedTenant,
    serviceInfo: state.serviceInfo,
    tenantIssuerDialogIsOpen: state.tenantIssuerDialogIsOpen,
    tenantURLDialogIsOpen: state.tenantURLDialogIsOpen,
    tenantURLs: state.tenantURLs,
    tenantURLsIsDirty: state.tenantURLsIsDirty,
    tenantURLsToCreate: state.tenantURLsToCreate,
    tenantURLsToDelete: state.tenantURLsToDelete,
    tenantURLsToUpdate: state.tenantURLsToUpdate,
    routeParams: state.routeParams,
  };
})(TenantDetailsPage);
