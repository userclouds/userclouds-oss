import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  Checkbox,
  CodeEditor,
  Dialog,
  DialogBody,
  GlobalStyles,
  InlineNotification,
  InputReadOnly,
  Label,
  Text,
  TextInput,
  ToolTip,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import { RootState, AppDispatch } from '../store';
import { redirect } from '../routing';
import {
  isNonZeroNumber,
  nonZeroNumberPattern,
  VALID_NAME_PATTERN,
} from '../models/helpers';
import AccessPolicy, {
  AccessPolicyTemplate,
  AccessPolicyTestResult,
  blankPolicy,
  blankPolicyTemplate,
} from '../models/AccessPolicy';
import PermissionsOnObject from '../models/authz/Permissions';
import {
  modifyAccessPolicy,
  changeAccessPolicyTestContext,
  getAccessPolicySuccess,
  toggleAccessPolicyDetailsEditMode,
  modifyAccessPolicyThresholds,
} from '../actions/tokenizer';
import {
  createAccessPolicy,
  createAccessPolicyTemplateForAccessPolicy,
  fetchAccessPolicy,
  fetchTenantAccessPolicyByVersion,
  fetchUserPermissionsForAccessPolicy,
  runAccessPolicyTest,
  updateAccessPolicy,
} from '../thunks/tokenizer';
import PolicyTemplateForm from './PolicyTemplateForm';
import PolicyComposer, { ConnectedPolicyChooserDialog } from './PolicyComposer';
import PageCommon from './PageCommon.module.css';
import Styles from './AccessPolicy.module.css';
import { PageTitle } from '../mainlayout/PageWrap';

const submitHandler =
  (companyID: string, tenantID: string, isNew: boolean, policy: AccessPolicy) =>
  async (dispatch: AppDispatch) => {
    if (isNew) {
      dispatch(createAccessPolicy(tenantID, companyID, policy));
    } else {
      dispatch(updateAccessPolicy(tenantID, companyID, policy));
    }
  };

const PolicyDeveloper = ({
  selectedCompanyID,
  selectedTenantID,
  accessPolicyPermissions,
  policy,
  modifiedPolicy,
  isNew,
  isDirty,
  isBusy,
  fetchError,
  saveError,
  testContext,
  testResult,
  testError,
  editMode,
  newTemplate,
  policyTemplateDialogIsOpen,
  query,
  dispatch,
}: {
  selectedCompanyID: string | undefined;
  selectedTenantID: string | undefined;
  accessPolicyPermissions: PermissionsOnObject | undefined;
  policy: AccessPolicy | undefined;
  modifiedPolicy: AccessPolicy | undefined;
  isNew: boolean;
  isDirty: boolean;
  isBusy: boolean;
  fetchError: string | undefined;
  saveError: string;
  testContext: string;
  testResult: AccessPolicyTestResult | undefined;
  testError: string;
  editMode: boolean;
  newTemplate: AccessPolicyTemplate | undefined;
  policyTemplateDialogIsOpen: boolean;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  const dialog: HTMLDialogElement | null = document.getElementById(
    'createPolicyTemplateDialog'
  ) as HTMLDialogElement;

  return (
    <>
      {selectedCompanyID &&
      selectedTenantID &&
      policy &&
      accessPolicyPermissions ? (
        <form
          onSubmit={(e: React.FormEvent) => {
            e.preventDefault();
            if (modifiedPolicy) {
              dispatch(
                submitHandler(
                  selectedCompanyID,
                  selectedTenantID,
                  isNew,
                  modifiedPolicy
                )
              );
            }
          }}
        >
          <div className={PageCommon.listviewtablecontrols}>
            <PageTitle
              title={isNew ? 'Create Access Policy' : 'Access Policy Details'}
              itemName={isNew ? 'New Access Policy' : policy.name}
            />

            <div className={PageCommon.listviewtablecontrolsToolTip}>
              <ToolTip>
                <>
                  {
                    'Name, describe and tag your policy so that it is easy to find and use later. '
                  }
                  <a
                    href="https://docs.userclouds.com/docs/token-access-policies"
                    title="UserClouds documentation for key concepts about access policies"
                    target="new"
                    className={PageCommon.link}
                  >
                    Learn more here.
                  </a>
                </>
              </ToolTip>
            </div>

            {editMode && modifiedPolicy && !policy.is_system ? (
              <ButtonGroup
                className={PageCommon.listviewtablecontrolsButtonGroup}
              >
                <Button
                  isLoading={isBusy}
                  disabled={!isDirty || isBusy}
                  theme="primary"
                  type="submit"
                  size="small"
                >
                  Save access policy
                </Button>
                <Button
                  isLoading={isBusy}
                  disabled={isBusy}
                  theme="secondary"
                  size="small"
                  onClick={(e: React.MouseEvent) => {
                    e.preventDefault();
                    if (
                      !isDirty ||
                      window.confirm(
                        'You have unsaved changes. Are you sure you want to cancel?'
                      )
                    ) {
                      isNew
                        ? redirect(
                            `/accesspolicies?company_id=${selectedCompanyID}&tenant_id=${selectedTenantID}`
                          )
                        : dispatch(toggleAccessPolicyDetailsEditMode(false));
                    }
                  }}
                >
                  Cancel
                </Button>
              </ButtonGroup>
            ) : !policy.is_system ? (
              <ButtonGroup>
                {accessPolicyPermissions.update && !policy.is_system && (
                  <Button
                    theme="primary"
                    size="small"
                    className={PageCommon.listviewtablecontrolsButton}
                    onClick={(e: React.MouseEvent) => {
                      e.preventDefault();
                      dispatch(toggleAccessPolicyDetailsEditMode(true));
                    }}
                  >
                    Edit access policy
                  </Button>
                )}
              </ButtonGroup>
            ) : (
              ''
            )}
          </div>
          <Card
            detailview
            lockedMessage={
              policy.is_system
                ? 'System access policy details cannot be edited'
                : ''
            }
          >
            <>
              {saveError && (
                <InlineNotification theme="alert">
                  {saveError}
                </InlineNotification>
              )}
            </>
            <CardRow
              title="Basic Details"
              tooltip={<>View and edit this access policy.</>}
              collapsible
            >
              <Label>
                Name
                <br />
                {editMode && modifiedPolicy ? (
                  <TextInput
                    value={modifiedPolicy.name}
                    required
                    name="policy_name"
                    id="policy_name"
                    pattern={VALID_NAME_PATTERN}
                    readOnly={!accessPolicyPermissions.update}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                      dispatch(
                        modifyAccessPolicy({
                          name: e.target.value,
                        })
                      );
                    }}
                  />
                ) : (
                  <InputReadOnly>{policy.name}</InputReadOnly>
                )}
              </Label>

              {policy.id && (
                <Label>
                  Version
                  <InputReadOnly>{policy.version}</InputReadOnly>
                </Label>
              )}
            </CardRow>

            <CardRow
              title="Compose Policy"
              tooltip={
                <>
                  Create your policy by combining existing policies, or creating
                  a new policy template from scratch.
                </>
              }
              collapsible
            >
              <PolicyComposer
                policy={editMode && modifiedPolicy ? modifiedPolicy : policy}
                changeAccessPolicyAction={modifyAccessPolicy}
                readOnly={!editMode}
                tableID="accessPolicyComponents"
              />
            </CardRow>

            <CardRow
              title="Metadata"
              tooltip={
                <>
                  Specify which context is required to resolve the policy.
                  Primarily used by the browser plug-in to request extra context
                  from user (e.g. for "Break Glass" flows).
                </>
              }
              collapsible
            >
              <Label>
                Required context
                <br />
                {editMode ? (
                  <TextInput
                    name="required_context"
                    id="required_context"
                    value={modifiedPolicy?.required_context_stringified}
                    readOnly={!accessPolicyPermissions.update}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                      dispatch(
                        modifyAccessPolicy({
                          required_context_stringified: e.target.value,
                        })
                      );
                    }}
                  />
                ) : (
                  <InputReadOnly>
                    {policy.required_context_stringified}
                  </InputReadOnly>
                )}
              </Label>
            </CardRow>
            <CardRow
              title="Execution Rate Limiting"
              tooltip={
                <>
                  Rate limiting allows you to control the frequency and volume
                  of API calls by a given user to prevent abuse and reduce
                  account takeover risk.
                  <a
                    href=" https://docs.userclouds.com/docs/enforce-rate-limiting"
                    title="UserClouds documentation for key concepts about rate limiting"
                    target="new"
                    className={PageCommon.link}
                  >
                    Learn more here.
                  </a>
                </>
              }
              collapsible
            >
              <Label className={GlobalStyles['mt-6']}>
                Enforce Execution Rate Limiting
                <br />
                {editMode ? (
                  <Checkbox
                    id="execution_rate"
                    name="execution_rate"
                    checked={
                      modifiedPolicy
                        ? isNonZeroNumber(
                            modifiedPolicy?.thresholds.max_executions
                          )
                        : policy.thresholds.max_executions !== 0
                    }
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                      const { checked } = e.target;
                      const value = checked ? 1 : 0;
                      dispatch(
                        modifyAccessPolicyThresholds({
                          max_executions: Number(value),
                        })
                      );
                    }}
                  />
                ) : (
                  <InputReadOnly
                    type="checkbox"
                    isChecked={policy.thresholds.max_executions !== 0}
                  />
                )}
              </Label>
              <div className={Styles.threshold_row}>
                {editMode
                  ? modifiedPolicy &&
                    isNonZeroNumber(
                      modifiedPolicy?.thresholds.max_executions
                    ) && (
                      <>
                        <Label>
                          Max Executions
                          <br />
                          <TextInput
                            name="max_executions"
                            id="max_executions"
                            type="number"
                            value={modifiedPolicy?.thresholds.max_executions}
                            readOnly={!accessPolicyPermissions.update}
                            onChange={(
                              e: React.ChangeEvent<HTMLInputElement>
                            ) => {
                              const { value } = e.target;
                              if (nonZeroNumberPattern.test(value)) {
                                dispatch(
                                  modifyAccessPolicyThresholds({
                                    max_executions:
                                      Number(value) === 0 ? '' : Number(value),
                                  })
                                );
                              }
                            }}
                          />
                        </Label>
                        <Label>
                          Max Execution Window (s)
                          <br />
                          <TextInput
                            name="max_execution_window"
                            id="max_execution_window"
                            type="number"
                            min="5"
                            max="60"
                            value={
                              modifiedPolicy?.thresholds
                                .max_execution_duration_seconds
                            }
                            readOnly={!accessPolicyPermissions.update}
                            onChange={(
                              e: React.ChangeEvent<HTMLInputElement>
                            ) => {
                              dispatch(
                                modifyAccessPolicyThresholds({
                                  max_execution_duration_seconds: Number(
                                    e.target.value
                                  ),
                                })
                              );
                            }}
                          />
                        </Label>
                        <Label>
                          Announce Max Execution Failure
                          <br />
                          <Checkbox
                            id="execution_announcement"
                            name="execution_announcement"
                            checked={
                              modifiedPolicy
                                ? modifiedPolicy?.thresholds
                                    .announce_max_execution_failure
                                : policy.thresholds
                                    .announce_max_execution_failure
                            }
                            onChange={(
                              e: React.ChangeEvent<HTMLInputElement>
                            ) => {
                              const { checked } = e.target;
                              dispatch(
                                modifyAccessPolicyThresholds({
                                  announce_max_execution_failure: checked,
                                })
                              );
                            }}
                          />
                        </Label>
                      </>
                    )
                  : policy &&
                    modifiedPolicy &&
                    isNonZeroNumber(
                      modifiedPolicy.thresholds.max_executions
                    ) && (
                      <>
                        <Label>
                          Max Executions
                          <br />
                          <InputReadOnly>
                            {policy.thresholds.max_executions}
                          </InputReadOnly>
                        </Label>
                        <Label>
                          Max Execution Window (s)
                          <br />
                          <InputReadOnly>
                            {policy.thresholds.max_execution_duration_seconds}
                          </InputReadOnly>
                        </Label>
                        <Label htmlFor="execution_announcement">
                          Announce Max Execution Failure
                          <br />
                          <InputReadOnly
                            id="execution_announcement"
                            type="checkbox"
                            isChecked={
                              policy.thresholds.announce_max_execution_failure
                            }
                          />
                        </Label>
                      </>
                    )}
              </div>
            </CardRow>
            <CardRow
              title="Result Rate Limiting"
              tooltip={
                <>
                  Rate limiting allows you to control the frequency and volume
                  of API calls by a given user to prevent abuse and reduce
                  account takeover risk.
                  <a
                    href=" https://docs.userclouds.com/docs/enforce-rate-limiting"
                    title="UserClouds documentation for key concepts about rate limiting"
                    target="new"
                    className={PageCommon.link}
                  >
                    Learn more here.
                  </a>
                </>
              }
              collapsible
            >
              <Label className={GlobalStyles['mt-6']}>
                Enforce Result Rate Limiting
                <br />
                {editMode ? (
                  <Checkbox
                    id="results_per_execution"
                    name="results_per_execution"
                    checked={
                      modifiedPolicy
                        ? isNonZeroNumber(
                            modifiedPolicy.thresholds.max_results_per_execution
                          )
                        : policy.thresholds.max_results_per_execution !== 0
                    }
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                      const { checked } = e.target;
                      const value = checked ? 1 : 0;
                      dispatch(
                        modifyAccessPolicyThresholds({
                          max_results_per_execution: Number(value),
                        })
                      );
                    }}
                  />
                ) : (
                  <InputReadOnly
                    type="checkbox"
                    isChecked={
                      policy.thresholds.max_results_per_execution !== 0
                    }
                  />
                )}
              </Label>
              <div className={Styles.threshold_row}>
                {editMode
                  ? modifiedPolicy &&
                    isNonZeroNumber(
                      modifiedPolicy?.thresholds.max_results_per_execution
                    ) && (
                      <>
                        <Label>
                          Max Results
                          <br />
                          <TextInput
                            name="max_results"
                            id="max_results"
                            value={
                              modifiedPolicy?.thresholds
                                .max_results_per_execution
                            }
                            readOnly={!accessPolicyPermissions.update}
                            type="number"
                            min="1"
                            onChange={(
                              e: React.ChangeEvent<HTMLInputElement>
                            ) => {
                              const { value } = e.target;
                              if (nonZeroNumberPattern.test(value)) {
                                dispatch(
                                  modifyAccessPolicyThresholds({
                                    max_results_per_execution:
                                      Number(value) === 0 ? '' : Number(value),
                                  })
                                );
                              }
                            }}
                          />
                        </Label>
                        <Label>
                          Announce Max Result Failure
                          <br />
                          <Checkbox
                            id="result_failure"
                            name="result_failure"
                            checked={
                              modifiedPolicy
                                ? modifiedPolicy?.thresholds
                                    .announce_max_result_failure
                                : policy.thresholds.announce_max_result_failure
                            }
                            onChange={(
                              e: React.ChangeEvent<HTMLInputElement>
                            ) => {
                              const { checked } = e.target;
                              dispatch(
                                modifyAccessPolicyThresholds({
                                  announce_max_result_failure: checked,
                                })
                              );
                            }}
                          />
                        </Label>
                      </>
                    )
                  : policy &&
                    policy.thresholds.max_results_per_execution > 0 && (
                      <>
                        <Label>
                          Max Results
                          <br />
                          <InputReadOnly>
                            {policy.thresholds.max_results_per_execution}
                          </InputReadOnly>
                        </Label>
                        <Label htmlFor="result_failure">
                          Announce Max Result Failure
                          <br />
                          <InputReadOnly
                            id="result_failure"
                            type="checkbox"
                            isChecked={
                              policy.thresholds.announce_max_result_failure
                            }
                          />
                        </Label>
                      </>
                    )}
              </div>
            </CardRow>

            {editMode && modifiedPolicy && (
              <CardRow
                title="Test Policy"
                tooltip={<> Test the policy you have composed.</>}
                collapsible
              >
                <Label htmlFor="context">
                  Context
                  <br />
                  <CodeEditor
                    id="context"
                    value={testContext}
                    onChange={(value: string) => {
                      dispatch(changeAccessPolicyTestContext(value));
                    }}
                    jsonExt
                  />
                </Label>
                {testError && (
                  <InlineNotification theme="alert">
                    {testError}
                  </InlineNotification>
                )}
                <ButtonGroup>
                  <Button
                    theme="inverse"
                    disabled={isBusy}
                    isLoading={isBusy}
                    onClick={(e: React.MouseEvent) => {
                      e.preventDefault();
                      dispatch(
                        runAccessPolicyTest(
                          modifiedPolicy,
                          testContext,
                          selectedTenantID
                        )
                      );
                    }}
                  >
                    Run Test
                  </Button>
                </ButtonGroup>

                {testResult && !policyTemplateDialogIsOpen && (
                  <InlineNotification
                    theme={testResult.allowed ? 'success' : 'alert'}
                  >
                    {testResult.allowed ? 'Access allowed' : 'Access denied'}
                  </InlineNotification>
                )}
              </CardRow>
            )}
          </Card>
          <Dialog
            id="createPolicyTemplateDialog"
            fullPage
            title="Create a New Policy Template"
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
            policy={modifiedPolicy || policy}
            changeAccessPolicyAction={modifyAccessPolicy}
            createNewPolicyTemplateHandler={() => dialog.showModal()}
          />
        </form>
      ) : fetchError ? (
        <InlineNotification theme="alert">{fetchError}</InlineNotification>
      ) : (
        <Text>Fetching policy...</Text>
      )}
    </>
  );
};
const ConnectedPolicyDeveloper = connect((state: RootState) => {
  return {
    selectedCompanyID: state.selectedCompanyID,
    selectedTenantID: state.selectedTenantID,
    accessPolicyPermissions: state.accessPolicyPermissions,
    policy: state.currentAccessPolicy,
    modifiedPolicy: state.modifiedAccessPolicy,
    policies: state.accessPolicies,
    templates: state.policyTemplates,
    componentPolicies: state.componentPolicies,
    isDirty: state.accessPolicyIsDirty,
    isBusy: state.savingAccessPolicy || state.testingPolicy,
    fetchError: state.accessPolicyFetchError,
    saveError: state.saveAccessPolicyError,
    testContext: state.accessPolicyTestContext,
    testResult: state.testingPolicyResult,
    testError: state.testingPolicyError,
    newTemplate: state.policyTemplateToCreate,
    policyTemplateDialogIsOpen: state.policyTemplateDialogIsOpen,
    query: state.query,
  };
})(PolicyDeveloper);

const TokenizerAccessPoliciesPage = ({
  selectedCompanyID,
  selectedTenantID,
  accessPolicyPermissions,
  location,
  routeParams,
  query,
  editMode,
  dispatch,
}: {
  selectedCompanyID: string | undefined;
  selectedTenantID: string | undefined;
  accessPolicyPermissions: PermissionsOnObject | undefined;
  location: URL;
  routeParams: Record<string, string>;
  query: URLSearchParams;
  editMode: boolean;
  dispatch: AppDispatch;
}) => {
  const { policyID, version } = routeParams;
  const { pathname } = location;
  const isNewPage = pathname.indexOf('create') > -1;
  const cleanQuery = makeCleanPageLink(query);

  useEffect(() => {
    if (selectedTenantID) {
      dispatch(
        fetchUserPermissionsForAccessPolicy(selectedTenantID, policyID || '')
      );
    }
  }, [selectedTenantID, policyID, dispatch]);
  // Four possibilities:
  // 1. Version provided + policy already stored (retrieve specific version)
  // 2. Version provided + policy not stored (fetch specific version)
  // 3. Version == "latest" + policy not stored (fetch latest version)
  // 4. Version == "latest" + policy already stored (retrieve latest version)
  useEffect(() => {
    if (selectedCompanyID && selectedTenantID) {
      if (accessPolicyPermissions) {
        if (!policyID) {
          if (!accessPolicyPermissions.create) {
            // shouldn't be on this page w/o a policy ID if user can't create new policies
            redirect(`/accesspolicies${cleanQuery}`);
          }
          dispatch(getAccessPolicySuccess(blankPolicy()));
        } else {
          version !== 'latest'
            ? dispatch(
                fetchTenantAccessPolicyByVersion(
                  selectedTenantID,
                  policyID,
                  version as string
                )
              )
            : dispatch(fetchAccessPolicy(selectedTenantID, policyID));
        }
      }
    }
  }, [
    accessPolicyPermissions,
    policyID,
    version,
    selectedTenantID,
    dispatch,
    cleanQuery,
    selectedCompanyID,
  ]);

  return (
    <ConnectedPolicyDeveloper
      isNew={isNewPage}
      editMode={editMode || isNewPage}
    />
  );
};

export default connect((state: RootState) => ({
  selectedCompanyID: state.selectedCompanyID,
  selectedTenantID: state.selectedTenantID,
  accessPolicyPermissions: state.accessPolicyPermissions,
  editMode: state.accessPolicyDetailsEditMode,
  location: state.location,
  query: state.query,
  routeParams: state.routeParams,
}))(TokenizerAccessPoliciesPage);
