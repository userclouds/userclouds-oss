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
  InlineNotification,
  Label,
  InputReadOnly,
  TextInput,
  Text,
} from '@userclouds/ui-component-lib';

import { updateTenantDataSourceElement } from '../API/datamapping';
import {
  modifyDataSourceElementDetails,
  updateDataSourceElementError,
  updateDataSourceElementRequest,
  updateDataSourceElementSuccess,
  toggleDataSourceElementEditMode,
} from '../actions/datamapping';
import PaginatedResult from '../models/PaginatedResult';
import DataSource, {
  DataSourceElement,
  UserDataTypes,
  Regulations,
  defaultJIRATicketOwner,
} from '../models/DataSource';
import { AppDispatch, RootState } from '../store';
import {
  fetchDataSources,
  fetchDataSourceElement,
} from '../thunks/datamapping';
import { postSuccessToast, postAlertToast } from '../thunks/notifications';
import PageCommon from './PageCommon.module.css';

export const handleUpdateDataSourceElement =
  (
    selectedCompanyID: string,
    selectedTenantID: string,
    element: DataSourceElement
  ) =>
  (dispatch: AppDispatch) => {
    if (selectedTenantID) {
      dispatch(updateDataSourceElementRequest);
      updateTenantDataSourceElement(selectedTenantID, element).then(
        (response: DataSourceElement) => {
          dispatch(updateDataSourceElementSuccess(response));
        },
        (error: APIError) => {
          dispatch(updateDataSourceElementError(error));
        }
      );
    }
  };

export const createJIRATicketForDataSourceElement =
  (companyID: string, tenantID: string, element: DataSourceElement) =>
  (dispatch: AppDispatch) => {
    const encodedTicketName = encodeURI(`Data source element ${element.path}`);
    const url =
      `https://hooks.zapier.com/hooks/catch/16356554/3cvhwwd/?title=${encodedTicketName}` +
      `&desc=${encodeURI('')}&owner=${encodeURI(
        element.metadata.owner || defaultJIRATicketOwner
      )}`;
    return fetch(url).then(
      () => {
        element.metadata.jira = `https://userclouds.atlassian.net/browse/ITSAMPLE-1?jql=text%20~%20%22${encodedTicketName}%22`;
        dispatch(
          handleUpdateDataSourceElement(companyID as string, tenantID, element)
        );
        dispatch(postSuccessToast('Successfully created JIRA ticket'));
      },
      () => {
        dispatch(postAlertToast('Error creating JIRA ticket'));
      }
    );
  };

const DataSourceElementPage = ({
  companyID,
  tenantID,
  dataSources,
  selectedDataSourceElement,
  modifiedDataSourceElement,
  isFetching,
  isSaving,
  saveSuccess,
  saveError,
  editMode,
  query,
  routeParams,
  dispatch,
}: {
  companyID: string | undefined;
  tenantID: string | undefined;
  dataSources: PaginatedResult<DataSource> | undefined;
  selectedDataSourceElement: DataSourceElement | undefined;
  modifiedDataSourceElement: DataSourceElement;
  isFetching: boolean;
  isSaving: boolean;
  saveSuccess: string;
  saveError: string;
  editMode: boolean;
  query: URLSearchParams;
  routeParams: Record<string, string>;
  dispatch: AppDispatch;
}) => {
  const element = editMode
    ? modifiedDataSourceElement
    : selectedDataSourceElement;

  const { elementID } = routeParams;

  useEffect(() => {
    if (tenantID) {
      if (
        selectedDataSourceElement === undefined ||
        selectedDataSourceElement.id !== elementID
      ) {
        dispatch(fetchDataSourceElement(tenantID, elementID));
      }
      if (dataSources === undefined) {
        dispatch(fetchDataSources(tenantID, query));
      }
    }
  }, [
    tenantID,
    elementID,
    selectedDataSourceElement,
    dataSources,
    query,
    dispatch,
  ]);

  if (!tenantID) {
    return (
      <InlineNotification theme="alert">
        Error fetching Tenant
      </InlineNotification>
    );
  }

  const matchingDataSource = dataSources?.data.find(
    (source: DataSource) => source.id === element?.data_source_id
  );

  return element ? (
    <form
      onSubmit={(e: React.FormEvent) => {
        e.preventDefault();
        dispatch(
          handleUpdateDataSourceElement(companyID as string, tenantID, element)
        );
      }}
    >
      <div className={PageCommon.listviewtablecontrols}>
        {saveSuccess && (
          <InlineNotification theme="success">{saveSuccess}</InlineNotification>
        )}
        {saveError && (
          <InlineNotification theme="alert">{saveError}</InlineNotification>
        )}
        <div className={PageCommon.listviewtablecontrolsToolTip} />
        <ButtonGroup className={PageCommon.listviewtablecontrolsButtonGroup}>
          {editMode ? (
            <>
              <Button
                isLoading={isFetching || isSaving}
                type="submit"
                theme="primary"
                size="small"
                disabled={isFetching || isSaving}
              >
                Save
              </Button>
              <Button
                size="small"
                theme="secondary"
                disabled={isFetching || isSaving}
                isLoading={isFetching || isSaving}
                onClick={() => {
                  dispatch(toggleDataSourceElementEditMode(false));
                }}
              >
                Cancel
              </Button>
            </>
          ) : (
            <>
              <Button
                theme="primary"
                size="small"
                type="button"
                disabled={isFetching || isSaving}
                onClick={(e: React.MouseEvent) => {
                  e.preventDefault();
                  dispatch(toggleDataSourceElementEditMode(true));
                }}
              >
                Edit
              </Button>
              {!element.metadata.jira && (
                <Button
                  theme="primary"
                  size="small"
                  type="button"
                  disabled={isFetching || isSaving}
                  onClick={(e: React.MouseEvent) => {
                    e.preventDefault();
                    dispatch(
                      createJIRATicketForDataSourceElement(
                        companyID as string,
                        tenantID,
                        element
                      )
                    );
                  }}
                >
                  Create JIRA Ticket
                </Button>
              )}
            </>
          )}
        </ButtonGroup>
      </div>
      <Card detailview>
        <CardRow title="Basic details" tooltip="TODO" collapsible>
          <div className={PageCommon.carddetailsrow}>
            <Label>
              Path
              <br />
              <InputReadOnly>{element.path}</InputReadOnly>
            </Label>
          </div>
          <div className={PageCommon.carddetailsrow}>
            <Label>
              Data Source
              <br />
              <InputReadOnly>
                {`${
                  matchingDataSource ? matchingDataSource.name + ' ' : ''
                }(ID: ${element.data_source_id})`}
              </InputReadOnly>
            </Label>
          </div>
          <div className={PageCommon.carddetailsrow}>
            <Label>
              Type
              <br />
              <InputReadOnly>{element.type}</InputReadOnly>
            </Label>
          </div>
          <div className={PageCommon.carddetailsrow}>
            <Label>
              Description
              <br />
              {editMode ? (
                <TextInput
                  id="description"
                  name="description"
                  value={element.metadata.description}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    const val = (e.target as HTMLInputElement).value;
                    dispatch(
                      modifyDataSourceElementDetails({
                        metadata: { description: val },
                      })
                    );
                  }}
                />
              ) : (
                <InputReadOnly>
                  {element.metadata.description || '-'}
                </InputReadOnly>
              )}
            </Label>
          </div>
        </CardRow>

        <CardRow title="Classification" tooltip="TODO" collapsible>
          <div className={PageCommon.carddetailsrow}>
            <DataListTagger
              inputName="contents"
              readOnly={!editMode}
              label="Contents"
              menuItems={Object.entries(UserDataTypes).map(([key, val]) => ({
                label: val,
                value: key,
              }))}
              selectedItems={(element.metadata?.contents || []).map(
                (t: keyof typeof UserDataTypes) => UserDataTypes[t] || t
              )}
              addHandler={(value: string) => {
                dispatch(
                  modifyDataSourceElementDetails({
                    metadata: {
                      contents: [...(element.metadata?.contents || []), value],
                    },
                  })
                );
              }}
              removeHandler={(newList: string[]) => {
                dispatch(
                  modifyDataSourceElementDetails({
                    metadata: { contents: newList },
                  })
                );
              }}
            />
          </div>
          <div className={PageCommon.carddetailsrow}>
            <DataListTagger
              inputName="tags"
              readOnly={!editMode}
              label="Tags"
              menuItems={[]}
              selectedItems={element.metadata?.tags || []}
              addHandler={(value: string) => {
                dispatch(
                  modifyDataSourceElementDetails({
                    metadata: {
                      tags: [...(element.metadata?.tags || []), value],
                    },
                  })
                );
              }}
              removeHandler={(newList: string[]) => {
                dispatch(
                  modifyDataSourceElementDetails({
                    metadata: { tags: newList },
                  })
                );
              }}
            />
          </div>
          <div className={PageCommon.carddetailsrow}>
            <DataListTagger
              inputName="regulations"
              readOnly={!editMode}
              label="Relevant regulations"
              menuItems={Object.entries(Regulations).map(([key, val]) => ({
                label: val,
                value: key,
              }))}
              selectedItems={(element.metadata?.regulations || []).map(
                (r: keyof typeof Regulations) => Regulations[r] || r
              )}
              addHandler={(value: string) => {
                dispatch(
                  modifyDataSourceElementDetails({
                    metadata: {
                      regulations: [
                        ...(element.metadata?.regulations || []),
                        value,
                      ],
                    },
                  })
                );
              }}
              removeHandler={(newList: string[]) => {
                dispatch(
                  modifyDataSourceElementDetails({
                    metadata: { regulations: newList },
                  })
                );
              }}
            />
          </div>
          <div className={PageCommon.carddetailsrow}>
            <Label>
              Contains PII
              <br />
              {editMode ? (
                <Checkbox
                  checked={element.metadata?.contains_pii}
                  id="pii"
                  name="pii"
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    dispatch(
                      modifyDataSourceElementDetails({
                        metadata: {
                          contains_pii: e.currentTarget.checked,
                        },
                      })
                    );
                  }}
                />
              ) : (
                <InputReadOnly>
                  {element.metadata.contains_pii ? 'Yes' : 'No'}
                </InputReadOnly>
              )}
            </Label>
          </div>
        </CardRow>

        <CardRow title="Ownership" tooltip="TODO" collapsible>
          <div className={PageCommon.carddetailsrow}>
            <Label>
              Data Owner
              <br />
              {editMode ? (
                <TextInput
                  id="owner"
                  name="owner"
                  value={element.metadata.owner}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    const val = (e.target as HTMLInputElement).value;
                    dispatch(
                      modifyDataSourceElementDetails({
                        metadata: { owner: val },
                      })
                    );
                  }}
                />
              ) : (
                <InputReadOnly>{element.metadata.owner || '-'}</InputReadOnly>
              )}
            </Label>
          </div>
          {!editMode && element.metadata.jira && (
            <div className={PageCommon.carddetailsrow}>
              <Label htmlFor="jira_ticket">
                JIRA
                <br />
                <a
                  href={element.metadata.jira}
                  target="new"
                  className={PageCommon.link}
                >
                  JIRA Ticket
                </a>
              </Label>
            </div>
          )}
        </CardRow>
      </Card>
    </form>
  ) : isFetching ? (
    <Text>Loading ...</Text>
  ) : (
    <InlineNotification theme="alert">
      Data source element not found
    </InlineNotification>
  );
};

export default connect((state: RootState) => {
  return {
    companyID: state.selectedCompanyID,
    tenantID: state.selectedTenantID,
    dataSources: state.dataSources,
    selectedDataSourceElement: state.selectedDataSourceElement,
    modifiedDataSourceElement: state.modifiedDataSourceElement,
    query: state.query,
    editMode: state.dataSourceElementDetailsEditMode,
    isFetching: state.fetchingDataSourceElements,
    isSaving: state.savingDataSourceElement,
    saveSuccess: state.dataSourceElementSaveSuccess,
    saveError: state.dataSourceElementSaveError,
    routeParams: state.routeParams,
  };
})(DataSourceElementPage);
