import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  GlobalStyles,
  Button,
  ButtonGroup,
  Card,
  CardRow,
  Label,
  InlineNotification,
  InputReadOnly,
  TextArea,
  TextInput,
  Text,
  ToolTip,
  TextShortener,
} from '@userclouds/ui-component-lib';

import {
  changeSelectedPurpose,
  togglePurposeDetailsEditMode,
  modifyPurposeDetails,
} from '../actions/purposes';
import { redirect } from '../routing';
import { RootState, AppDispatch } from '../store';
import { SelectedTenant } from '../models/Tenant';
import Purpose, { blankPurpose } from '../models/Purpose';
import PaginatedResult from '../models/PaginatedResult';
import { PageTitle } from '../mainlayout/PageWrap';
import { fetchPurpose, savePurpose } from '../thunks/purposes';
import PageCommon from './PageCommon.module.css';

const PurposeDetails = ({
  isCreatePage,
  selectedCompanyID,
  selectedTenant,
  selectedPurpose,
  modifiedPurpose,
  isFetching,
  fetchError,
  isSaving,
  createError,
  saveError,
  deleteError,
  editMode,
  dispatch,
}: {
  isCreatePage: boolean;
  selectedCompanyID: string | undefined;
  selectedTenant: SelectedTenant | undefined;
  selectedPurpose: Purpose | undefined;
  modifiedPurpose: Purpose | undefined;
  isFetching: boolean;
  fetchError: string;
  isSaving: boolean;
  createError: string;
  saveError: string;
  deleteError: string;
  editMode: boolean;
  dispatch: AppDispatch;
}) => {
  editMode = isCreatePage || editMode;
  const isDirty = isCreatePage
    ? modifiedPurpose?.name && modifiedPurpose?.description
    : selectedPurpose?.name !== modifiedPurpose?.name ||
      selectedPurpose?.description !== modifiedPurpose?.description;

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <PageTitle
          title={isCreatePage ? 'Create Purpose' : 'View Purpose'}
          itemName={
            selectedPurpose && selectedPurpose.name
              ? selectedPurpose.name
              : 'New Purpose'
          }
        />
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              {isCreatePage
                ? 'Create a new purpose for managing end user consents. '
                : 'Manage this purpose. '}
              <a
                href="https://docs.userclouds.com/docs/configure-your-store"
                title="UserClouds documentation for defining purposes"
                target="new"
                className={PageCommon.link}
              >
                Learn more here.
              </a>
            </>
          </ToolTip>
        </div>

        <ButtonGroup className={PageCommon.listviewtablecontrolsButtonGroup}>
          {!isCreatePage ? (
            selectedTenant?.is_admin &&
            !modifiedPurpose?.is_system && (
              <>
                {!editMode ? (
                  <Button
                    theme="primary"
                    size="small"
                    onClick={() => {
                      dispatch(togglePurposeDetailsEditMode(true));
                    }}
                  >
                    Edit Purpose
                  </Button>
                ) : (
                  <>
                    <Button
                      size="small"
                      theme="secondary"
                      disabled={isSaving}
                      onClick={() => {
                        if (
                          !isDirty ||
                          window.confirm(
                            'You have unsaved changes. Are you sure you want to cancel editing?'
                          )
                        ) {
                          redirect(
                            `/purposes?company_id=${selectedCompanyID}&tenant_id=${selectedTenant.id}`
                          );
                        }
                      }}
                    >
                      Cancel
                    </Button>
                    <Button
                      theme="primary"
                      size="small"
                      disabled={isSaving || !isDirty}
                      isLoading={isSaving}
                      onClick={() => {
                        selectedCompanyID &&
                          modifiedPurpose &&
                          dispatch(
                            savePurpose(
                              selectedCompanyID,
                              selectedTenant.id,
                              modifiedPurpose,
                              isCreatePage
                            )
                          );
                      }}
                    >
                      Save Purpose
                    </Button>
                  </>
                )}
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
                      `You have unsaved changes. Are you sure you want to ${
                        isCreatePage
                          ? 'abandon purpose creation'
                          : 'cancel editing'
                      }?`
                    )
                  ) {
                    selectedTenant &&
                      redirect(
                        `/purposes?company_id=${selectedCompanyID}&tenant_id=${selectedTenant.id}`
                      );
                  }
                }}
              >
                Cancel
              </Button>
              <Button
                theme="primary"
                size="small"
                isLoading={isSaving}
                disabled={isSaving || !isDirty}
                onClick={() => {
                  selectedCompanyID &&
                    selectedTenant &&
                    isCreatePage &&
                    modifiedPurpose &&
                    dispatch(
                      savePurpose(
                        selectedCompanyID,
                        selectedTenant.id,
                        modifiedPurpose,
                        isCreatePage
                      )
                    );
                }}
              >
                Create Purpose
              </Button>
            </>
          )}
        </ButtonGroup>
      </div>
      <Card
        id="purposeDetails"
        detailview
        lockedMessage={
          modifiedPurpose?.is_system
            ? 'This purpose is system-defined and cannot be modified.'
            : undefined
        }
      >
        {selectedCompanyID && selectedTenant && modifiedPurpose ? (
          <>
            {createError ? (
              <InlineNotification theme="alert">
                {createError}
              </InlineNotification>
            ) : (
              ''
            )}
            {saveError ? (
              <InlineNotification theme="alert">{saveError}</InlineNotification>
            ) : (
              ''
            )}
            {deleteError ? (
              <InlineNotification theme="alert">
                {deleteError}
              </InlineNotification>
            ) : (
              ''
            )}
            <CardRow
              title="Basic Details"
              tooltip={<>View and edit this purpose.</>}
              collapsible
            >
              {!isCreatePage ? (
                <Label htmlFor="purpose_id">
                  ID
                  <br />
                  <TextShortener
                    text={modifiedPurpose.id}
                    length={6}
                    id="purpose_id"
                  />
                </Label>
              ) : (
                ''
              )}
              <Label className={GlobalStyles['mt-6']}>
                Name
                <br />
                {isCreatePage ? (
                  <TextInput
                    name="purpose_name"
                    id="purpose_name"
                    type="text"
                    value={modifiedPurpose.name}
                    required
                    onChange={(e: React.ChangeEvent) => {
                      const val = (e.target as HTMLInputElement).value;
                      dispatch(
                        modifyPurposeDetails({
                          name: val,
                        })
                      );
                    }}
                  />
                ) : (
                  <InputReadOnly name="purpose_name">
                    {modifiedPurpose.name}
                  </InputReadOnly>
                )}
              </Label>
              <Label className={GlobalStyles['mt-6']}>
                Description
                <br />
                {editMode && (
                  <TextArea
                    name="purpose_description"
                    value={modifiedPurpose.description}
                    placeholder="Add a description"
                    onChange={(e: React.ChangeEvent) => {
                      const val = (e.target as HTMLTextAreaElement).value;
                      dispatch(
                        modifyPurposeDetails({
                          description: val,
                        })
                      );
                    }}
                  />
                )}
              </Label>
              {!editMode && <Text>{modifiedPurpose.description || 'N/A'}</Text>}
            </CardRow>
          </>
        ) : isFetching ? (
          <Text>Loading ...</Text>
        ) : (
          <InlineNotification theme="alert">
            {fetchError || 'Something went wrong'}
          </InlineNotification>
        )}
      </Card>
    </>
  );
};

const ConnectedPurposeDetails = connect((state: RootState) => ({
  selectedCompanyID: state.selectedCompanyID,
  selectedTenant: state.selectedTenant,
  selectedPurpose: state.selectedPurpose,
  modifiedPurpose: state.modifiedPurpose,
  isFetching: state.fetchingPurposes,
  fetchError: state.purposesFetchError,
  isSaving: state.savingPurpose,
  createError: state.createPurposeError,
  saveError: state.savePurposeError,
  deleteError: state.deletePurposeError,
  editMode: state.purposeDetailsEditMode,
}))(PurposeDetails);

const PurposePage = ({
  selectedTenantID,
  purposes,
  selectedPurpose,
  location,
  routeParams,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  purposes: PaginatedResult<Purpose> | undefined;
  selectedPurpose: Purpose | undefined;
  location: URL;
  routeParams: Record<string, string>;
  dispatch: AppDispatch;
}) => {
  const { pathname } = location;
  const { purposeID } = routeParams;

  const isCreatePage = pathname.indexOf('create') > -1;

  useEffect(() => {
    if (!isCreatePage) {
      if (selectedTenantID && purposeID && selectedPurpose?.id !== purposeID) {
        let matchingPurpose;
        if (purposes) {
          matchingPurpose = purposes.data.find(
            (purpose: Purpose) => purpose.id === purposeID
          );
        }
        if (matchingPurpose) {
          dispatch(changeSelectedPurpose(matchingPurpose));
        } else {
          dispatch(fetchPurpose(selectedTenantID, purposeID));
        }
      }
    } else {
      dispatch(modifyPurposeDetails(blankPurpose()));
    }
  }, [
    isCreatePage,
    selectedTenantID,
    purposeID,
    purposes,
    selectedPurpose,
    dispatch,
  ]);
  return <ConnectedPurposeDetails isCreatePage={isCreatePage} />;
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  purposes: state.purposes,
  selectedPurpose: state.selectedPurpose,
  location: state.location,
  routeParams: state.routeParams,
}))(PurposePage);
