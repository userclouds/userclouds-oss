import { v4 as uuidv4 } from 'uuid';
import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import { APIError } from '@userclouds/sharedui';
import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  Checkbox,
  DataListTagger,
  GlobalStyles,
  HiddenTextInput,
  InlineNotification,
  InputReadOnly,
  Label,
  Select,
  Text,
  TextArea,
  TextInput,
  TextShortener,
  ToolTip,
} from '@userclouds/ui-component-lib';

import {
  createTenantDataSource,
  updateTenantDataSource,
} from '../API/datamapping';
import { postSuccessToast } from '../thunks/notifications';
import { fetchDataSource, deleteDataSource } from '../thunks/datamapping';
import {
  createDataSourceError,
  createDataSourceRequest,
  createDataSourceSuccess,
  modifyDataSourceDetails,
  updateDataSourceError,
  updateDataSourceRequest,
  updateDataSourceSuccess,
  toggleDataSourceEditMode,
} from '../actions/datamapping';
import DataSource, {
  DataSourceTypes,
  DataFormats,
  DataClassifications,
  DataStorageOptions,
  Regulations,
  defaultJIRATicketOwner,
} from '../models/DataSource';
import { redirect } from '../routing';
import { AppDispatch, RootState } from '../store';
import PageCommon from './PageCommon.module.css';

export const handleCreateDataSource =
  (
    selectedCompanyID: string,
    selectedTenantID: string,
    dataSource: DataSource
  ) =>
  (dispatch: AppDispatch) => {
    if (selectedTenantID) {
      dispatch(createDataSourceRequest);
      dataSource.id = uuidv4();
      createTenantDataSource(selectedTenantID, dataSource).then(
        (response: DataSource) => {
          dispatch(createDataSourceSuccess(response));
          dispatch(
            postSuccessToast(
              `Successfully created data source '${response.name}'`
            )
          );
          redirect(
            `/datasources?company_id=${selectedCompanyID}&tenant_id=${selectedTenantID}`
          );
        },
        (error: APIError) => {
          dispatch(createDataSourceError(error));
        }
      );
    }
  };

export const handleUpdateDataSource =
  (
    selectedCompanyID: string,
    selectedTenantID: string,
    dataSource: DataSource
  ) =>
  (dispatch: AppDispatch) => {
    if (selectedTenantID) {
      dispatch(updateDataSourceRequest);
      updateTenantDataSource(selectedTenantID, dataSource).then(
        (response: DataSource) => {
          dispatch(updateDataSourceSuccess(response));
        },
        (error: APIError) => {
          dispatch(updateDataSourceError(error));
        }
      );
    }
  };

const createJIRATicket =
  (companyID: string, tenantID: string, dataSource: DataSource) =>
  (dispatch: AppDispatch) => {
    const encodedTicketName = encodeURI(`Data source ${dataSource.name}`);
    const url =
      `https://hooks.zapier.com/hooks/catch/16356554/3cvhwwd/?title=${encodedTicketName}` +
      `&desc=${encodeURI('')}&owner=${encodeURI(
        dataSource.metadata.owner || defaultJIRATicketOwner
      )}`;
    return fetch(url).then(() => {
      dataSource.metadata.jira = `https://userclouds.atlassian.net/browse/ITSAMPLE-1?jql=text%20~%20%22${encodedTicketName}%22`;
      dispatch(
        handleUpdateDataSource(companyID as string, tenantID, dataSource)
      );
      dispatch(postSuccessToast('Successfully created JIRA ticket'));
    });
  };

const DataSourcePage = ({
  companyID,
  tenantID,
  selectedDataSource,
  modifiedDataSource,
  isFetching,
  fetchError,
  editMode,
  isDeleting,
  isSaving,
  saveSuccess,
  saveError,
  location,
  routeParams,
  dispatch,
}: {
  companyID: string | undefined;
  tenantID: string | undefined;
  selectedDataSource: DataSource | undefined;
  modifiedDataSource: DataSource;
  isFetching: boolean;
  fetchError: string;
  editMode: boolean;
  isDeleting: boolean;
  isSaving: boolean;
  saveSuccess: string;
  saveError: string;
  location: URL;
  routeParams: Record<string, string>;
  dispatch: AppDispatch;
}) => {
  const { pathname } = location;
  const { dataSourceID } = routeParams;
  const createPage = pathname.indexOf('create') > -1;
  const dataSource =
    createPage || editMode ? modifiedDataSource : selectedDataSource;

  useEffect(() => {
    if (tenantID) {
      if (
        !createPage &&
        (selectedDataSource === undefined ||
          selectedDataSource.id !== dataSourceID)
      ) {
        dispatch(fetchDataSource(tenantID, dataSourceID));
      }
    }
  }, [tenantID, createPage, dataSourceID, selectedDataSource, dispatch]);

  if (!tenantID) {
    return (
      <InlineNotification theme="alert">
        Error fetching Tenant
      </InlineNotification>
    );
  }
  return (
    <form
      onSubmit={(e: React.FormEvent) => {
        e.preventDefault();
        if (dataSource) {
          if (createPage) {
            dispatch(
              handleCreateDataSource(companyID as string, tenantID, dataSource)
            );
          } else {
            dispatch(
              handleUpdateDataSource(companyID as string, tenantID, dataSource)
            );
          }
        }
      }}
    >
      <div className={PageCommon.listviewtablecontrols}>
        {saveSuccess && (
          <InlineNotification theme="success">{saveSuccess}</InlineNotification>
        )}
        {saveError && (
          <InlineNotification theme="alert">{saveError}</InlineNotification>
        )}
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              {createPage
                ? 'Add a new Data Source to your data map. '
                : 'Manage import settings and metadata for your data source. '}
              <a
                href="https://docs.userclouds.com/docs/"
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
          {createPage || editMode ? (
            <>
              {!createPage && (
                <Button
                  size="small"
                  theme="secondary"
                  disabled={isFetching || isSaving}
                  onClick={(e: React.MouseEvent) => {
                    e.preventDefault();

                    dispatch(toggleDataSourceEditMode(false));
                  }}
                >
                  Cancel
                </Button>
              )}
              <Button
                isLoading={isFetching || isSaving}
                type="submit"
                theme="primary"
                size="small"
                disabled={isFetching || isSaving}
              >
                {createPage ? 'Create Data Source' : 'Save Data Source'}
              </Button>
            </>
          ) : (
            <>
              <Button
                theme="primary"
                size="small"
                type="button"
                disabled={isFetching}
                onClick={(e: React.MouseEvent) => {
                  e.preventDefault();

                  dispatch(toggleDataSourceEditMode(true));
                }}
              >
                Edit
              </Button>
              {!createPage && dataSource && !dataSource.metadata.jira && (
                <Button
                  theme="primary"
                  size="small"
                  type="button"
                  disabled={isFetching}
                  onClick={(e: React.MouseEvent) => {
                    e.preventDefault();
                    dispatch(
                      createJIRATicket(
                        companyID as string,
                        tenantID,
                        dataSource
                      )
                    );
                  }}
                >
                  Create JIRA Ticket
                </Button>
              )}
              {!createPage && dataSource && (
                <Button
                  isLoading={isFetching || isDeleting}
                  theme="dangerous"
                  size="small"
                  disabled={isFetching || isDeleting}
                  onClick={() => {
                    if (
                      window.confirm(
                        `Are you sure you want to delete this data source?`
                      )
                    ) {
                      dispatch(
                        deleteDataSource(
                          companyID as string,
                          tenantID,
                          dataSourceID
                        )
                      );
                    }
                  }}
                >
                  Delete
                </Button>
              )}
            </>
          )}
        </ButtonGroup>
      </div>
      <Card detailview>
        {dataSource ? (
          <>
            <CardRow
              title="Basic details"
              tooltip="Name and describe your data source."
              collapsible
            >
              <div className={PageCommon.carddetailsrow}>
                <Label>
                  Name
                  <br />
                  {createPage ? (
                    <TextInput
                      id="name"
                      name="name"
                      value={dataSource.name}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                        const val = (e.target as HTMLInputElement).value;
                        dispatch(
                          modifyDataSourceDetails({
                            name: val,
                          })
                        );
                      }}
                    />
                  ) : (
                    <InputReadOnly>{dataSource.name}</InputReadOnly>
                  )}
                </Label>
                <Label>
                  Data source type
                  <br />
                  {createPage ? (
                    <Select
                      name="type"
                      defaultValue={dataSource.type}
                      id="type"
                      onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
                        e.preventDefault();
                        const selectedType = e.currentTarget.value;
                        dispatch(
                          modifyDataSourceDetails({
                            type: selectedType,
                          })
                        );
                      }}
                    >
                      {Object.entries(DataSourceTypes).map(([key, val]) => (
                        <option value={key} key={key}>
                          {val}
                        </option>
                      ))}
                    </Select>
                  ) : (
                    <InputReadOnly>{dataSource.type}</InputReadOnly>
                  )}
                </Label>
                {!createPage && (
                  <Label htmlFor="data_source_id">
                    ID
                    <br />
                    <TextShortener
                      text={dataSource.id}
                      length={6}
                      id="data_source_id"
                    />
                  </Label>
                )}
                {!createPage && (
                  <Label>
                    Info
                    <br />
                    <InputReadOnly>
                      {dataSource.metadata.info
                        ? dataSource.metadata.info.tables +
                          ' tables; ' +
                          dataSource.metadata.info.rows +
                          ' rows; ' +
                          'last updated ' +
                          new Date(
                            dataSource.metadata.info.updated
                          ).toLocaleString('en-US')
                        : 'Unknown'}
                    </InputReadOnly>
                  </Label>
                )}
              </div>
              <Label className={GlobalStyles['mt-6']}>
                Description
                <br />
                {createPage || editMode ? (
                  <TextArea
                    name="description"
                    id="description"
                    placeholder="Enter a description"
                    onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => {
                      e.preventDefault();
                      dispatch(
                        modifyDataSourceDetails({
                          metadata: { description: e.target.value },
                        })
                      );
                    }}
                  >
                    {dataSource.metadata?.description || ''}
                  </TextArea>
                ) : (
                  <InputReadOnly>
                    {dataSource.metadata?.description || '-'}
                  </InputReadOnly>
                )}
              </Label>
            </CardRow>

            {!['file', 'other', 's3bucket'].includes(dataSource.type) && (
              <CardRow
                title="Data import configuration"
                tooltip="Tag your data source with relevant classifications, like regulations, format or sensitivity."
                collapsible
              >
                <div className={PageCommon.carddetailsrow}>
                  <Label>
                    Host
                    <br />
                    {createPage || editMode ? (
                      <TextInput
                        id="host"
                        name="host"
                        value={dataSource.config.host}
                        disabled={!createPage && !editMode}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                          const val = (e.target as HTMLInputElement).value;
                          dispatch(
                            modifyDataSourceDetails({
                              config: { host: val },
                            })
                          );
                        }}
                      />
                    ) : (
                      <InputReadOnly>{dataSource.config.host}</InputReadOnly>
                    )}
                  </Label>
                  <Label>
                    Port
                    <br />
                    {createPage || editMode ? (
                      <TextInput
                        id="port"
                        name="port"
                        value={dataSource.config.port}
                        disabled={!createPage && !editMode}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                          const val = (e.target as HTMLInputElement).value;
                          dispatch(
                            modifyDataSourceDetails({
                              config: { port: parseInt(val, 10) || 0 },
                            })
                          );
                        }}
                      />
                    ) : (
                      <InputReadOnly>{dataSource.config.port}</InputReadOnly>
                    )}
                  </Label>
                  <Label>
                    Database
                    <br />
                    {createPage || editMode ? (
                      <TextInput
                        id="database"
                        name="database"
                        value={dataSource.config.database}
                        disabled={!createPage && !editMode}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                          const val = (e.target as HTMLInputElement).value;
                          dispatch(
                            modifyDataSourceDetails({
                              config: { database: val },
                            })
                          );
                        }}
                      />
                    ) : (
                      <InputReadOnly>
                        {dataSource.config.database}
                      </InputReadOnly>
                    )}
                  </Label>
                  <Label>
                    Username
                    <br />
                    {createPage || editMode ? (
                      <TextInput
                        id="username"
                        name="username"
                        value={dataSource.config.username}
                        disabled={!createPage && !editMode}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                          const val = e.target.value;
                          dispatch(
                            modifyDataSourceDetails({
                              config: { username: val },
                            })
                          );
                        }}
                      />
                    ) : (
                      <InputReadOnly>
                        {dataSource.config.username}
                      </InputReadOnly>
                    )}
                  </Label>
                  {(createPage || editMode) && (
                    <Label>
                      Password
                      <br />
                      <HiddenTextInput
                        id="password"
                        name="password"
                        value={dataSource.config.password}
                        disabled={!createPage && !editMode}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                          const val = e.target.value;
                          dispatch(
                            modifyDataSourceDetails({
                              config: { password: val },
                            })
                          );
                        }}
                      />
                    </Label>
                  )}
                </div>
              </CardRow>
            )}

            <CardRow
              title="Classification"
              tooltip="Tag your data source with relevant classifications, like regulations, format or sensitivity."
              collapsible
            >
              <div className={PageCommon.carddetailsrow}>
                <DataListTagger
                  inputName="format"
                  readOnly={!createPage && !editMode}
                  label="Format"
                  menuItems={Object.entries(DataFormats).map(([key, val]) => ({
                    label: val,
                    value: key,
                  }))}
                  selectedItems={(dataSource.metadata?.format || []).map(
                    (f: keyof typeof DataFormats) => DataFormats[f] || f
                  )}
                  addHandler={(value: string) => {
                    dispatch(
                      modifyDataSourceDetails({
                        metadata: {
                          format: [
                            ...(dataSource.metadata?.format || []),
                            value,
                          ],
                        },
                      })
                    );
                  }}
                  removeHandler={(newList: string[]) => {
                    dispatch(
                      modifyDataSourceDetails({
                        metadata: { format: newList },
                      })
                    );
                  }}
                />

                <DataListTagger
                  inputName="tags"
                  readOnly={!createPage && !editMode}
                  label="Tags"
                  menuItems={Object.entries(DataClassifications).map(
                    ([key, val]) => ({ label: val, value: key })
                  )}
                  selectedItems={(
                    dataSource.metadata?.classifications || []
                  ).map(
                    (c: keyof typeof DataClassifications) =>
                      DataClassifications[c] || c
                  )}
                  addHandler={(value: string) => {
                    dispatch(
                      modifyDataSourceDetails({
                        metadata: {
                          classifications: [
                            ...(dataSource.metadata?.classifications || []),
                            value,
                          ],
                        },
                      })
                    );
                  }}
                  removeHandler={(newList: string[]) => {
                    dispatch(
                      modifyDataSourceDetails({
                        metadata: { classifications: newList },
                      })
                    );
                  }}
                />
                <DataListTagger
                  inputName="regulations"
                  readOnly={!createPage && !editMode}
                  label="Regulations"
                  menuItems={Object.entries(Regulations).map(([key, val]) => ({
                    label: val,
                    value: key,
                  }))}
                  selectedItems={(dataSource.metadata?.regulations || []).map(
                    (r: keyof typeof Regulations) => Regulations[r] || r
                  )}
                  addHandler={(value: string) => {
                    dispatch(
                      modifyDataSourceDetails({
                        metadata: {
                          regulations: [
                            ...(dataSource.metadata?.regulations || []),
                            value,
                          ],
                        },
                      })
                    );
                  }}
                  removeHandler={(newList: string[]) => {
                    dispatch(
                      modifyDataSourceDetails({
                        metadata: { regulations: newList },
                      })
                    );
                  }}
                />
                <Label>
                  Contains PII
                  <br />
                  {createPage || editMode ? (
                    <Checkbox
                      checked={dataSource.metadata?.contains_pii}
                      id="contains_pii"
                      name="contains_pii"
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                        dispatch(
                          modifyDataSourceDetails({
                            metadata: {
                              contains_pii: e.currentTarget.checked,
                            },
                          })
                        );
                      }}
                    />
                  ) : (
                    <InputReadOnly>
                      {dataSource.metadata?.contains_pii ? 'Yes' : 'No'}
                    </InputReadOnly>
                  )}
                </Label>
                <Label>
                  Data storage
                  <br />
                  {createPage || editMode ? (
                    <Select
                      id="storage"
                      name="storage"
                      value={dataSource.metadata.storage}
                      onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
                        dispatch(
                          modifyDataSourceDetails({
                            metadata: { storage: e.target.value },
                          })
                        );
                      }}
                    >
                      <option key="blank_option" value="">
                        Select an option
                      </option>
                      {Object.entries(DataStorageOptions).map(([key, val]) => (
                        <option value={key} key={key}>
                          {val}
                        </option>
                      ))}
                    </Select>
                  ) : (
                    <InputReadOnly>
                      {DataStorageOptions[
                        dataSource.metadata
                          ?.storage as keyof typeof DataStorageOptions
                      ] || '-'}
                    </InputReadOnly>
                  )}
                </Label>

                <Label>
                  Third-party hosted
                  <br />
                  {createPage || editMode ? (
                    <Checkbox
                      checked={
                        dataSource.metadata && dataSource.metadata['3p_hosted']
                      }
                      name="3p_hosted"
                      id="3p_hosted"
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                        dispatch(
                          modifyDataSourceDetails({
                            metadata: {
                              '3p_hosted': e.currentTarget.checked,
                            },
                          })
                        );
                      }}
                    />
                  ) : (
                    <InputReadOnly>
                      {dataSource.metadata && dataSource.metadata['3p_hosted']
                        ? 'Yes'
                        : 'No'}
                    </InputReadOnly>
                  )}
                </Label>

                <Label>
                  Third-party managed
                  <br />
                  {createPage || editMode ? (
                    <Checkbox
                      checked={
                        dataSource.metadata && dataSource.metadata['3p_managed']
                      }
                      name="3p_managed"
                      id="3p_managed"
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                        dispatch(
                          modifyDataSourceDetails({
                            metadata: {
                              '3p_managed': e.currentTarget.checked,
                            },
                          })
                        );
                      }}
                    />
                  ) : (
                    <InputReadOnly>
                      {dataSource.metadata && dataSource.metadata['3p_managed']
                        ? 'Yes'
                        : 'No'}
                    </InputReadOnly>
                  )}
                </Label>
              </div>
            </CardRow>
            <CardRow
              title="Ownership"
              tooltip="Assign an owner to this data in your organization."
              collapsible
            >
              <div className={PageCommon.carddetailsrow}>
                <Label>
                  Data Owner
                  <br />
                  {editMode ? (
                    <TextInput
                      id="owner"
                      name="owner"
                      value={dataSource.metadata.owner}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                        const val = (e.target as HTMLInputElement).value;
                        dispatch(
                          modifyDataSourceDetails({
                            metadata: { owner: val },
                          })
                        );
                      }}
                    />
                  ) : (
                    <InputReadOnly>
                      {dataSource.metadata.owner || '-'}
                    </InputReadOnly>
                  )}
                </Label>
              </div>
              {!editMode && dataSource.metadata.jira && (
                <div className={PageCommon.carddetailsrow}>
                  <Label htmlFor="jira_ticket">
                    JIRA
                    <br />
                    <a
                      href={dataSource.metadata.jira}
                      target="new"
                      className={PageCommon.link}
                    >
                      JIRA Ticket
                    </a>
                  </Label>
                </div>
              )}
            </CardRow>
          </>
        ) : fetchError ? (
          <InlineNotification theme="alert">{fetchError}</InlineNotification>
        ) : isFetching ? (
          <Text>Loading ...</Text>
        ) : (
          <Text>Data source not found</Text>
        )}
      </Card>
    </form>
  );
};

export default connect((state: RootState) => {
  return {
    companyID: state.selectedCompanyID,
    tenantID: state.selectedTenantID,
    location: state.location,
    editMode: state.dataSourceDetailsEditMode,
    modifiedDataSource: state.modifiedDataSource,
    selectedDataSource: state.selectedDataSource,
    isFetching: state.fetchingDataSources,
    fetchError: state.fetchDataSourceError,
    isDeleting: state.deletingDataSources,
    isSaving: state.savingDataSource,
    saveSuccess: state.dataSourceSaveSuccess,
    saveError: state.dataSourceSaveError,
    routeParams: state.routeParams,
  };
})(DataSourcePage);
