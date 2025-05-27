import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  Dialog,
  DialogBody,
  HiddenTextInput,
  InlineNotification,
  InputReadOnly,
  InputReadOnlyHidden,
  Label,
  Select,
  TextInput,
} from '@userclouds/ui-component-lib';

import { RootState, AppDispatch } from '../store';
import { PageTitle } from '../mainlayout/PageWrap';
import { SelectedTenant } from '../models/Tenant';
import { ObjectStore, blankObjectStore } from '../models/ObjectStore';
import { blankResourceID } from '../models/ResourceID';
import AccessPolicy, {
  blankPolicy,
  blankPolicyTemplate,
  AccessPolicyTemplate,
} from '../models/AccessPolicy';
import { NilUuid } from '../models/Uuids';
import {
  createObjectStore,
  updateObjectStore,
  getObjectStore,
} from '../thunks/userstore';
import {
  fetchAccessPolicy,
  createAccessPolicyTemplateForAccessPolicy,
} from '../thunks/tokenizer';

import {
  toggleEditUserstoreObjectStoreMode,
  modifyUserstoreObjectStore,
  fetchUserStoreObjectStoreSuccess,
} from '../actions/userstore';
import {
  modifyAccessPolicy,
  getAccessPolicySuccess,
} from '../actions/tokenizer';
import PolicyComposer, { ConnectedPolicyChooserDialog } from './PolicyComposer';
import PolicyTemplateForm from './PolicyTemplateForm';
import PageCommon from './PageCommon.module.css';

const ObjectStoreDetailsPage = ({
  selectedTenant,
  editingObjectStore,
  currentObjectStore,
  modifiedObjectStore,
  modifiedAccessPolicy,
  policyTemplateDialogIsOpen,
  newTemplate,
  fetchingObjectStore,
  savingObjectStore,
  saveError,
  routeParams,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  editingObjectStore: boolean;
  currentObjectStore: ObjectStore | undefined;
  modifiedObjectStore: ObjectStore | undefined;
  modifiedAccessPolicy: AccessPolicy | undefined;
  policyTemplateDialogIsOpen: boolean;
  newTemplate: AccessPolicyTemplate | undefined;
  fetchingObjectStore: boolean;
  savingObjectStore: boolean;
  saveError: string;
  routeParams: Record<string, string>;
  dispatch: AppDispatch;
}) => {
  const { objectStoreID } = routeParams;
  const isCreatePage = objectStoreID === 'create';

  useEffect(() => {
    if (!currentObjectStore && selectedTenant) {
      if (isCreatePage) {
        dispatch(fetchUserStoreObjectStoreSuccess(blankObjectStore()));
      } else {
        dispatch(getObjectStore(selectedTenant.id, objectStoreID));
      }
    }
  }, [
    currentObjectStore,
    selectedTenant,
    isCreatePage,
    objectStoreID,
    dispatch,
  ]);

  useEffect(() => {
    if (
      selectedTenant &&
      currentObjectStore &&
      currentObjectStore.access_policy.id !== NilUuid
    ) {
      dispatch(
        fetchAccessPolicy(
          selectedTenant.id,
          currentObjectStore.access_policy.id
        )
      );
    } else {
      dispatch(getAccessPolicySuccess(blankPolicy()));
    }
  }, [selectedTenant, currentObjectStore, dispatch]);

  const dialog: HTMLDialogElement | null = document.getElementById(
    'createPolicyTemplateDialog'
  ) as HTMLDialogElement;

  return (
    <>
      <form
        onSubmit={(e: React.FormEvent<HTMLFormElement>) => {
          e.preventDefault();

          const objectStore: ObjectStore = {
            id: isCreatePage ? NilUuid : currentObjectStore!.id,
            name: modifiedObjectStore!.name,
            type: modifiedObjectStore!.type || 's3',
            region: modifiedObjectStore!.region,
            access_key_id: modifiedObjectStore!.access_key_id,
            secret_access_key: modifiedObjectStore!.secret_access_key,
            role_arn: modifiedObjectStore!.role_arn,
            access_policy: isCreatePage
              ? blankResourceID()
              : currentObjectStore!.access_policy,
          };

          if (isCreatePage) {
            dispatch(
              createObjectStore(
                selectedTenant!.company_id,
                selectedTenant!.id,
                objectStore,
                modifiedAccessPolicy
              )
            );
          } else {
            dispatch(
              updateObjectStore(
                selectedTenant!.id,
                objectStore,
                modifiedAccessPolicy
              )
            );
          }
        }}
      >
        <div className={PageCommon.listviewtablecontrols}>
          <PageTitle
            title={
              isCreatePage
                ? 'Create Object Store'
                : editingObjectStore
                  ? 'Edit Object Store'
                  : 'Object Store Details'
            }
            itemName={
              isCreatePage ? 'New Object Store' : currentObjectStore?.name
            }
          />
          <ButtonGroup>
            {(editingObjectStore || isCreatePage) && (
              <>
                <Button
                  type="submit"
                  theme="primary"
                  isLoading={savingObjectStore || fetchingObjectStore}
                >
                  Save
                </Button>
                {!isCreatePage && (
                  <Button
                    theme="outline"
                    onClick={() => {
                      dispatch(toggleEditUserstoreObjectStoreMode(false));
                    }}
                    isLoading={savingObjectStore || fetchingObjectStore}
                  >
                    Cancel
                  </Button>
                )}
              </>
            )}
            {selectedTenant?.is_admin &&
              !editingObjectStore &&
              !isCreatePage && (
                <Button
                  theme="primary"
                  onClick={() => {
                    dispatch(toggleEditUserstoreObjectStoreMode(true));
                  }}
                >
                  Edit
                </Button>
              )}
          </ButtonGroup>
        </div>

        <Card isDirty={editingObjectStore || isCreatePage} detailview>
          {selectedTenant && currentObjectStore && modifiedObjectStore ? (
            <>
              {saveError && (
                <InlineNotification theme="alert">
                  {saveError}
                </InlineNotification>
              )}

              <CardRow
                title="Basic details"
                tooltip="Add the object store connection details. You must provide either Role ARN or the pair of Access Key ID and Secret Access Key for S3 object stores."
                collapsible
              >
                <Label>
                  Name
                  <br />
                  {isCreatePage || editingObjectStore ? (
                    <TextInput
                      name="name"
                      value={modifiedObjectStore.name}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                        const val = e.target.value;
                        dispatch(modifyUserstoreObjectStore({ name: val }));
                      }}
                    />
                  ) : (
                    <InputReadOnly>{currentObjectStore.name}</InputReadOnly>
                  )}
                </Label>
                <Label>
                  Object Store Type
                  <br />
                  {isCreatePage || editingObjectStore ? (
                    <Select
                      full
                      defaultValue="s3"
                      onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
                        const dbType = (e.target as HTMLSelectElement).value;
                        dispatch(modifyUserstoreObjectStore({ type: dbType }));
                      }}
                    >
                      <option key="s3" value="s3">
                        S3
                      </option>
                    </Select>
                  ) : (
                    <InputReadOnly>
                      {currentObjectStore.type || 'Not configured'}
                    </InputReadOnly>
                  )}
                </Label>
                <Label>
                  Region
                  <br />
                  {isCreatePage || editingObjectStore ? (
                    <TextInput
                      name="region"
                      value={modifiedObjectStore.region}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                        const val = e.target.value;
                        dispatch(modifyUserstoreObjectStore({ region: val }));
                      }}
                    />
                  ) : (
                    <InputReadOnly>{currentObjectStore.region}</InputReadOnly>
                  )}
                </Label>
                <Label>
                  Access Key ID
                  <br />
                  {isCreatePage || editingObjectStore ? (
                    <TextInput
                      name="access_key_id"
                      value={modifiedObjectStore.access_key_id}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                        const val = e.target.value;
                        dispatch(
                          modifyUserstoreObjectStore({
                            access_key_id: val,
                          })
                        );
                      }}
                    />
                  ) : (
                    <InputReadOnly>
                      {currentObjectStore.access_key_id || 'Not configured'}
                    </InputReadOnly>
                  )}
                </Label>
                <Label>
                  Secret Access Key
                  <br />
                  {isCreatePage || editingObjectStore ? (
                    <HiddenTextInput
                      name="secret_key"
                      value={modifiedObjectStore.secret_access_key}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                        const val = e.target.value;
                        dispatch(
                          modifyUserstoreObjectStore({
                            secret_access_key: val,
                          })
                        );
                      }}
                      disabled={!(isCreatePage || editingObjectStore)}
                    />
                  ) : (
                    <InputReadOnlyHidden
                      name="secret_key"
                      value=""
                      canBeShown={false}
                    />
                  )}
                </Label>
                <Label>
                  Role ARN
                  <br />
                  {isCreatePage || editingObjectStore ? (
                    <TextInput
                      name="role_arn"
                      value={modifiedObjectStore.role_arn}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                        const val = e.target.value;
                        dispatch(
                          modifyUserstoreObjectStore({
                            role_arn: val,
                          })
                        );
                      }}
                    />
                  ) : (
                    <InputReadOnly>
                      {currentObjectStore.role_arn || 'Not configured'}
                    </InputReadOnly>
                  )}
                </Label>
                {!(isCreatePage || editingObjectStore) && (
                  <Label>
                    Proxy URL
                    <br />
                    <InputReadOnly>
                      {`${selectedTenant.tenant_url}/s3shim/${currentObjectStore.id}/`}
                    </InputReadOnly>
                  </Label>
                )}
              </CardRow>

              <CardRow
                title="Access policy"
                tooltip={
                  <>
                    {
                      'Select an access policy describing when objects from this store can be accessed. '
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
                  policy={modifiedAccessPolicy}
                  changeAccessPolicyAction={modifyAccessPolicy}
                  readOnly={!editingObjectStore && !isCreatePage}
                />
              </CardRow>
            </>
          ) : (
            'Loading object store...'
          )}
        </Card>
      </form>
      <ConnectedPolicyChooserDialog
        policy={modifiedAccessPolicy}
        tokenRes={false}
        changeAccessPolicyAction={modifyAccessPolicy}
        createNewPolicyTemplateHandler={() => dialog.showModal()}
      />
      <Dialog
        id="createPolicyTemplateDialog"
        title="Create Policy Template"
        description="Create a new template and add it to your composite policy. The template will be saved to your library for re-use later."
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
            />
          </DialogBody>
        )}
      </Dialog>
    </>
  );
};

export default connect((state: RootState) => {
  return {
    selectedTenant: state.selectedTenant,
    editingObjectStore: state.editingObjectStore,
    currentObjectStore: state.currentObjectStore,
    modifiedObjectStore: state.modifiedObjectStore,
    modifiedAccessPolicy: state.modifiedAccessPolicy,
    policyTemplateDialogIsOpen: state.policyTemplateDialogIsOpen,
    newTemplate: state.policyTemplateToCreate,
    fetchingObjectStore: state.fetchingObjectStore,
    savingObjectStore: state.savingObjectStore,
    routeParams: state.routeParams,
    saveError: state.saveObjectStoreError,
  };
})(ObjectStoreDetailsPage);
