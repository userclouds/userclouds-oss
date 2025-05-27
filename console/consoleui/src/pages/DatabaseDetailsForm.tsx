import React from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  DialogFooter,
  HiddenTextInput,
  InlineNotification,
  InputReadOnly,
  InputReadOnlyHidden,
  Label,
  Select,
  TextInput,
} from '@userclouds/ui-component-lib';

import { RootState, AppDispatch } from '../store';
import {
  modifyUserstoreDatabase,
  editUserStoreDatabaseError,
  cancelDatabaseDialog,
} from '../actions/userstore';
import { testDatabaseConnection } from '../thunks/userstore';
import { SelectedTenant } from '../models/Tenant';
import ServiceInfo from '../ServiceInfo';
import { getDatabaseError, SqlshimDatabase } from '../models/SqlshimDatabase';
import styles from './DatabaseDetailsForm.module.css';

const DatabaseDetailsForm = ({
  creatingDatabase,
  currentDatabase,
  fetchingDatabase,
  modifiedDatabase,
  saveError,
  savingDatabase,
  selectedTenant,
  serviceInfo,
  testDatabaseError,
  testDatabaseSuccess,
  testingDatabase,
  dispatch,
}: {
  creatingDatabase: boolean;
  currentDatabase: SqlshimDatabase | undefined;
  fetchingDatabase: boolean;
  modifiedDatabase: SqlshimDatabase | undefined;
  saveError: string;
  savingDatabase: boolean;
  selectedTenant: SelectedTenant | undefined;
  serviceInfo: ServiceInfo | undefined;
  testDatabaseError: string;
  testDatabaseSuccess: boolean;
  testingDatabase: boolean;
  dispatch: AppDispatch;
}) => {
  return (
    <>
      {selectedTenant && currentDatabase && (
        <form
          onSubmit={(e: React.FormEvent<HTMLFormElement>) => {
            // note to future self - why do we differentiate submit/create and javascript/update?
            e.preventDefault();
            const dialogDB = (e.target as HTMLFormElement).closest('dialog');
            if (dialogDB) {
              dialogDB.close();
            }
          }}
          className={styles.databaseform}
        >
          {saveError && (
            <InlineNotification theme="alert">{saveError}</InlineNotification>
          )}
          {testDatabaseSuccess && (
            <InlineNotification theme="success">
              Successfully connected to the database
            </InlineNotification>
          )}
          {testDatabaseError && (
            <InlineNotification theme="alert">
              {testDatabaseError}
            </InlineNotification>
          )}

          <Label>
            Database Host
            <br />
            {modifiedDatabase ? (
              <TextInput
                name="host"
                required
                value={modifiedDatabase.host}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                  const val = e.target.value;
                  dispatch(modifyUserstoreDatabase({ host: val }));
                }}
                placeholder="Enter Database Host"
              />
            ) : (
              <InputReadOnly>
                {currentDatabase.host || 'Not configured'}
              </InputReadOnly>
            )}
          </Label>

          <Label>
            Database Name
            <br />
            {modifiedDatabase ? (
              <TextInput
                name="name"
                required
                value={modifiedDatabase.name}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                  const val = e.target.value;
                  dispatch(modifyUserstoreDatabase({ name: val }));
                }}
                placeholder="Enter Database Name"
              />
            ) : (
              <InputReadOnly>
                {currentDatabase.name || 'Not configured'}
              </InputReadOnly>
            )}
          </Label>

          <fieldset>
            <Label>
              Database Port
              <br />
              {modifiedDatabase ? (
                <TextInput
                  name="port"
                  required
                  value={modifiedDatabase.port || ''}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    const val = Number(e.target.value);
                    dispatch(modifyUserstoreDatabase({ port: val }));
                  }}
                  placeholder="Enter Port"
                />
              ) : (
                <InputReadOnly>
                  {currentDatabase.port || 'Not configured'}
                </InputReadOnly>
              )}
            </Label>
            <Label>
              Database Type
              <br />
              {modifiedDatabase ? (
                <Select
                  full
                  defaultValue={currentDatabase.type || 'postgres'}
                  onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
                    const dbType = (e.target as HTMLSelectElement).value;
                    dispatch(modifyUserstoreDatabase({ type: dbType }));
                  }}
                >
                  <option key="postgres" value="postgres">
                    PostgreSQL
                  </option>
                  <option key="mysql" value="mysql">
                    MySQL
                  </option>
                </Select>
              ) : (
                <InputReadOnly>
                  {currentDatabase.type || 'Not configured'}
                </InputReadOnly>
              )}
            </Label>
          </fieldset>
          <fieldset>
            <Label>
              Username
              <br />
              {modifiedDatabase ? (
                <TextInput
                  name="username"
                  value={modifiedDatabase.username}
                  required
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    const val = e.target.value;
                    dispatch(modifyUserstoreDatabase({ username: val }));
                  }}
                  placeholder="Enter Username"
                />
              ) : (
                <InputReadOnly>
                  {currentDatabase.username || 'Not configured'}
                </InputReadOnly>
              )}
            </Label>
            <Label>
              Password
              <br />
              {modifiedDatabase ? (
                <HiddenTextInput
                  name="password"
                  value={modifiedDatabase.password}
                  required
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    const val = e.target.value;
                    dispatch(modifyUserstoreDatabase({ password: val }));
                  }}
                  placeholder="Enter Password"
                />
              ) : (
                <InputReadOnlyHidden
                  name="password"
                  value={currentDatabase.password || ''}
                  canBeShown={false}
                />
              )}
            </Label>
          </fieldset>

          <hr />
          <fieldset>
            <Label>
              Proxy Hostname
              <br />
              {modifiedDatabase && serviceInfo?.uc_admin ? (
                <TextInput
                  name="proxy_host"
                  value={modifiedDatabase.proxy_host}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    const val = e.target.value;
                    dispatch(modifyUserstoreDatabase({ proxy_host: val }));
                  }}
                  placeholder="Enter Proxy Hostname"
                />
              ) : modifiedDatabase && modifiedDatabase.proxy_host ? (
                <InputReadOnly>
                  {currentDatabase.proxy_host ||
                    'Contact UserCloud Admin to set'}
                </InputReadOnly>
              ) : (
                <InputReadOnly>
                  {currentDatabase.proxy_host || 'Defined by UserCloud Admin'}
                </InputReadOnly>
              )}
            </Label>
            <Label>
              Proxy Port
              <br />
              {modifiedDatabase && serviceInfo?.uc_admin ? (
                <TextInput
                  name="proxy_port"
                  value={modifiedDatabase.proxy_port || ''}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    const val = e.target.value;
                    dispatch(
                      modifyUserstoreDatabase({ proxy_port: Number(val) })
                    );
                  }}
                  placeholder="Enter Proxy Port"
                />
              ) : modifiedDatabase && modifiedDatabase.proxy_port ? (
                <InputReadOnly>
                  {currentDatabase.proxy_port ||
                    'Contact UserCloud Admin to set'}
                </InputReadOnly>
              ) : (
                <InputReadOnly>
                  {currentDatabase.proxy_port || 'Defined by UserCloud Admin'}
                </InputReadOnly>
              )}
            </Label>
          </fieldset>
          <DialogFooter>
            <ButtonGroup>
              <Button
                theme="outline"
                size="small"
                type="button"
                onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
                  e.preventDefault();
                  dispatch(
                    testDatabaseConnection(selectedTenant.id, modifiedDatabase!)
                  );
                }}
                isLoading={testingDatabase}
              >
                Test Connection
              </Button>

              <Button
                theme="secondary"
                size="small"
                onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
                  e.preventDefault();
                  dispatch(cancelDatabaseDialog(currentDatabase.id));
                  const editDBDialog = (e.target as HTMLButtonElement).closest(
                    'dialog'
                  );
                  if (editDBDialog) {
                    editDBDialog.close();
                  }
                }}
                isLoading={savingDatabase || fetchingDatabase}
              >
                Cancel
              </Button>

              {creatingDatabase ? (
                <Button
                  theme="primary"
                  size="small"
                  disabled={testingDatabase}
                  type="submit"
                  isLoading={savingDatabase || fetchingDatabase}
                >
                  Establish Connection
                </Button>
              ) : (
                <Button
                  onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
                    e.preventDefault();
                    const error = getDatabaseError(modifiedDatabase);
                    if (error) {
                      dispatch(editUserStoreDatabaseError(error));
                    } else {
                      const editDBDialog = (
                        e.target as HTMLButtonElement
                      ).closest('dialog');
                      if (editDBDialog) {
                        editDBDialog.close();
                      } // changes in reducer so just close. save on tenant page
                    }
                  }}
                  theme="primary"
                  disabled={testingDatabase}
                  size="small"
                  isLoading={savingDatabase || fetchingDatabase}
                >
                  Save
                </Button>
              )}
            </ButtonGroup>
          </DialogFooter>
        </form>
      )}
    </>
  );
};

export default connect((state: RootState) => {
  return {
    creatingDatabase: state.creatingDatabase,
    currentDatabase: state.currentSqlshimDatabase,
    fetchingDatabase: state.fetchingSqlshimDatabase,
    modifiedDatabase: state.modifiedSqlshimDatabase,
    saveError: state.saveSqlshimDatabaseError,
    savingDatabase: state.savingSqlshimDatabase,
    selectedTenant: state.selectedTenant,
    serviceInfo: state.serviceInfo,
    testDatabaseError: state.testSqlshimDatabaseError,
    testDatabaseSuccess: state.testSqlshimDatabaseSuccess,
    testingDatabase: state.testingSqlshimDatabase,
  };
})(DatabaseDetailsForm);
