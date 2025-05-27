import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardFooter,
  CardRow,
  CodeEditor,
  GlobalStyles,
  InlineNotification,
  InputReadOnly,
  Label,
  Text,
  TextArea,
  TextInput,
  TextShortener,
  ToolTip,
} from '@userclouds/ui-component-lib';

import { RootState, AppDispatch } from '../store';
import {
  modifyPolicyTemplate,
  togglePolicyTemplatesEditMode,
  changeAccessPolicyTestContext,
  changeAccessPolicyTestParams,
} from '../actions/tokenizer';
import { VALID_NAME_PATTERN } from '../models/helpers';
import AccessPolicy, {
  AccessPolicyTemplate,
  AccessPolicyTestResult,
} from '../models/AccessPolicy';
import { runAccessPolicyTemplateTest } from '../thunks/tokenizer';
import PageCommon from './PageCommon.module.css';
import { PageTitle } from '../mainlayout/PageWrap';

type SaveFunction = (
  tenantID: string,
  template: AccessPolicyTemplate,
  searchParams?: URLSearchParams,
  policy?: AccessPolicy | undefined
) => (dispatch: AppDispatch, getState: () => RootState) => void;

const PolicyTemplateForm = ({
  selectedTenantID,
  editableTemplate,
  savedTemplate,
  saveTemplate,
  onCancel,
  editMode,
  isFetching,
  isSaving,
  saveSuccess,
  saveError,
  accessPolicy,
  testContext,
  testParams,
  testResult,
  testError,
  isDialog = true,
  searchParams,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  editableTemplate: AccessPolicyTemplate | undefined;
  savedTemplate: AccessPolicyTemplate | undefined;
  saveTemplate: SaveFunction;
  onCancel: () => void;
  editMode: boolean;
  isFetching: boolean;
  isSaving: boolean;
  saveSuccess: boolean;
  saveError: string;
  accessPolicy: AccessPolicy | undefined;
  testContext: string;
  testParams: string;
  testResult: AccessPolicyTestResult | undefined;
  testError: string;
  isDialog?: boolean;
  searchParams?: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  const isNew = !savedTemplate;
  const isDirty =
    editableTemplate &&
    (savedTemplate
      ? editableTemplate.name !== savedTemplate.name ||
        editableTemplate.description !== savedTemplate.description ||
        editableTemplate.function !== savedTemplate.function
      : editableTemplate.name !== '' || editableTemplate.description !== '');
  if (isNew) {
    editMode = true;
  }

  return (
    <form
      onSubmit={(e: React.FormEvent) => {
        e.preventDefault();

        if (selectedTenantID && editableTemplate) {
          dispatch(
            saveTemplate(
              selectedTenantID,
              editableTemplate,
              searchParams,
              accessPolicy
            )
          );
        }
      }}
      className={GlobalStyles['min-w-full']}
    >
      {isDialog ? (
        <>
          <input
            type="hidden"
            name="policy_template_id"
            value={editableTemplate?.id || ''}
          />
          <CardRow
            title="Basic Details"
            tooltip={<>View and edit this Template.</>}
            collapsible
          >
            {!isNew && (
              <>
                <Label>
                  ID
                  <br />
                  <InputReadOnly monospace>{savedTemplate?.id}</InputReadOnly>
                </Label>
                <Label className={GlobalStyles['mt-6']}>
                  Version
                  <br />
                  <InputReadOnly>{savedTemplate?.version}</InputReadOnly>
                </Label>
              </>
            )}
            <Label className={GlobalStyles['mt-6']}>
              Template Name
              <br />
              {editMode ? (
                <TextInput
                  name="policy_template_name"
                  value={editableTemplate?.name}
                  pattern={VALID_NAME_PATTERN}
                  required
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    dispatch(
                      modifyPolicyTemplate({ name: e.target.value }, isNew)
                    );
                  }}
                />
              ) : (
                <InputReadOnly>{savedTemplate?.name}</InputReadOnly>
              )}
            </Label>
            {editMode ? (
              <Label className={GlobalStyles['mt-6']}>
                Template Description
                <br />
                <TextArea
                  name="policy_template_description"
                  value={editableTemplate?.description}
                  onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => {
                    dispatch(
                      modifyPolicyTemplate(
                        { description: e.target.value },
                        isNew
                      )
                    );
                  }}
                />
              </Label>
            ) : (
              <>
                <Label
                  className={GlobalStyles['mt-6']}
                  htmlFor="policy_template_description"
                >
                  Template Description
                </Label>
                <Text id="policy_template_description">
                  {savedTemplate?.description}
                </Text>
              </>
            )}
          </CardRow>

          <CardRow
            title="Template Function"
            tooltip={<>Define the function for this template.</>}
            collapsible
          >
            <Label className={GlobalStyles['mt-6']} htmlFor="function">
              <br />
              <CodeEditor
                value={editableTemplate?.function}
                id="function"
                required
                readOnly={!editMode}
                onChange={(value: string) => {
                  dispatch(modifyPolicyTemplate({ function: value }, isNew));
                }}
                javascriptExt
              />
            </Label>
          </CardRow>
          {editMode && (
            <CardRow
              title="Test Policy Template"
              tooltip={<>Test the policy template you have composed.</>}
              collapsible
            >
              <Label className={GlobalStyles['mt-6']} htmlFor="test_context">
                Context
                <CodeEditor
                  id="test_context"
                  name="test_context"
                  value={testContext}
                  onChange={(value: string) => {
                    dispatch(changeAccessPolicyTestContext(value));
                  }}
                  jsonExt
                />
              </Label>
              <Label className={GlobalStyles['mt-3']} htmlFor="test_params">
                Parameters
                <br />
                <TextInput
                  id="test_params"
                  name="test_params"
                  value={testParams}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    dispatch(changeAccessPolicyTestParams(e.target.value));
                  }}
                />
              </Label>
              {testError && (
                <InlineNotification
                  theme="alert"
                  className={`${GlobalStyles['mt-6']} ${GlobalStyles['mb-3']}`}
                >
                  {testError}
                </InlineNotification>
              )}
              <ButtonGroup>
                <Button
                  theme="inverse"
                  disabled={isFetching || isSaving}
                  isLoading={isFetching || isSaving}
                  onClick={(e: React.MouseEvent) => {
                    e.preventDefault();
                    if (selectedTenantID && editableTemplate) {
                      dispatch(
                        runAccessPolicyTemplateTest(
                          editableTemplate,
                          testContext,
                          testParams,
                          selectedTenantID
                        )
                      );
                    }
                  }}
                >
                  Run Test
                </Button>
              </ButtonGroup>

              {testResult && (
                <InlineNotification
                  theme={testResult.allowed ? 'success' : 'alert'}
                  className={`${GlobalStyles['mt-6']} ${GlobalStyles['mb-3']}`}
                >
                  {testResult.allowed ? 'Access allowed' : 'Access denied'}
                </InlineNotification>
              )}

              {testResult?.debug?.console && (
                <InlineNotification
                  theme="info"
                  className={GlobalStyles['mt-3']}
                >
                  <pre>{testResult.debug.console}</pre>
                </InlineNotification>
              )}
            </CardRow>
          )}

          {editableTemplate && !editableTemplate?.is_system && (
            <>
              {saveSuccess && (
                <InlineNotification theme="success">
                  Successfully saved policy template
                </InlineNotification>
              )}
              {saveError && (
                <InlineNotification theme="alert">
                  {saveError}
                </InlineNotification>
              )}
              <CardFooter>
                <ButtonGroup className={GlobalStyles['mt-6']}>
                  {!editMode ? (
                    <Button
                      theme="primary"
                      size="small"
                      isLoading={isFetching}
                      onClick={() => {
                        dispatch(togglePolicyTemplatesEditMode(true));
                      }}
                    >
                      Edit template
                    </Button>
                  ) : (
                    <>
                      <Button
                        type="submit"
                        theme="primary"
                        size="small"
                        isLoading={isFetching || isSaving}
                        disabled={isFetching || isSaving || !isDirty}
                      >
                        {isNew ? 'Create template' : 'Save template'}
                      </Button>
                      <Button
                        theme="secondary"
                        size="small"
                        isLoading={isFetching || isSaving}
                        disabled={isFetching || isSaving}
                        onClick={() => {
                          if (
                            !isDirty ||
                            window.confirm(
                              'You have unsaved changed. Are you sure you want to cancel?'
                            )
                          ) {
                            onCancel();
                            if (!isNew) {
                              dispatch(togglePolicyTemplatesEditMode(false));
                            }
                          }
                        }}
                      >
                        Cancel
                      </Button>
                    </>
                  )}
                </ButtonGroup>
              </CardFooter>
            </>
          )}
        </>
      ) : (
        <>
          <div className={PageCommon.listviewtablecontrols}>
            <PageTitle
              title={isNew ? 'Create Template' : 'Template Details'}
              itemName={isNew ? 'New Template' : editableTemplate?.name}
            />

            <div className={PageCommon.listviewtablecontrolsToolTip}>
              <ToolTip>
                <>
                  {
                    'Name, describe and define your template so that it is easy to find and use later. '
                  }
                  <a
                    href="https://docs.userclouds.com/docs/access-policies-1#access-policy-templates"
                    title="UserClouds documentation for key concepts about policy templates"
                    target="new"
                    className={PageCommon.link}
                  >
                    Learn more here.
                  </a>
                </>
              </ToolTip>
            </div>
            {editableTemplate && !editableTemplate?.is_system && (
              <>
                <ButtonGroup>
                  {!editMode ? (
                    <Button
                      theme="primary"
                      size="small"
                      isLoading={isFetching}
                      onClick={() => {
                        dispatch(togglePolicyTemplatesEditMode(true));
                      }}
                    >
                      Edit template
                    </Button>
                  ) : (
                    <>
                      <Button
                        type="submit"
                        theme="primary"
                        size="small"
                        isLoading={isFetching || isSaving}
                        disabled={isFetching || isSaving || !isDirty}
                      >
                        {isNew ? 'Create template' : 'Save template'}
                      </Button>
                      <Button
                        theme="secondary"
                        size="small"
                        isLoading={isFetching || isSaving}
                        disabled={isFetching || isSaving}
                        onClick={() => {
                          if (
                            !isDirty ||
                            window.confirm(
                              'You have unsaved changed. Are you sure you want to cancel?'
                            )
                          ) {
                            onCancel();
                            if (!isNew) {
                              dispatch(togglePolicyTemplatesEditMode(false));
                            }
                          }
                        }}
                      >
                        Cancel
                      </Button>
                    </>
                  )}
                </ButtonGroup>
              </>
            )}
          </div>

          <Card
            detailview
            lockedMessage={
              editableTemplate?.is_system
                ? 'System policy template details cannot be edited'
                : ''
            }
          >
            {saveSuccess && (
              <InlineNotification theme="success">
                Successfully saved policy template
              </InlineNotification>
            )}
            {saveError && (
              <InlineNotification theme="alert">{saveError}</InlineNotification>
            )}

            <input
              type="hidden"
              name="policy_template_id"
              value={editableTemplate?.id || ''}
            />
            <CardRow
              title="Basic Details"
              tooltip={<>View and edit this Template.</>}
              collapsible
            >
              {!isNew && (
                <>
                  <Label htmlFor="policy_template_id">
                    ID
                    <br />
                    <TextShortener text={savedTemplate?.id} length={6} />
                  </Label>
                  <Label
                    className={GlobalStyles['mt-6']}
                    htmlFor="policy_template_version"
                  >
                    Version
                    <br />
                    <InputReadOnly id="policy_template_version">
                      {savedTemplate?.version}
                    </InputReadOnly>
                  </Label>
                </>
              )}
              <Label
                className={GlobalStyles['mt-6']}
                htmlFor="policy_template_name"
              >
                Template Name
                <br />
                {editMode ? (
                  <TextInput
                    name="policy_template_name"
                    value={editableTemplate?.name}
                    pattern={VALID_NAME_PATTERN}
                    required
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                      dispatch(
                        modifyPolicyTemplate({ name: e.target.value }, isNew)
                      );
                    }}
                  />
                ) : (
                  <InputReadOnly>{savedTemplate?.name}</InputReadOnly>
                )}
              </Label>
              {editMode ? (
                <Label
                  className={GlobalStyles['mt-6']}
                  htmlFor="policy_template_description"
                >
                  Template Description
                  <br />
                  <TextArea
                    name="policy_template_description"
                    value={editableTemplate?.description}
                    onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => {
                      dispatch(
                        modifyPolicyTemplate(
                          { description: e.target.value },
                          isNew
                        )
                      );
                    }}
                  />
                </Label>
              ) : (
                <>
                  <Label
                    className={GlobalStyles['mt-6']}
                    htmlFor="policy_template_description"
                  >
                    Template Description
                  </Label>
                  <Text id="policy_template_description">
                    {savedTemplate?.description}
                  </Text>
                </>
              )}
            </CardRow>

            <CardRow
              title="Template Function"
              tooltip={<>Define the function for this template.</>}
              collapsible
            >
              <Label className={GlobalStyles['mt-6']} htmlFor="function">
                <br />
                <CodeEditor
                  value={editableTemplate?.function}
                  id="function"
                  required
                  readOnly={!editMode}
                  onChange={(value: string) => {
                    dispatch(modifyPolicyTemplate({ function: value }, isNew));
                  }}
                  javascriptExt
                />
              </Label>
            </CardRow>
            {editMode && (
              <CardRow
                title="Test Policy Template"
                tooltip={<>Test the policy template you have composed.</>}
                collapsible
              >
                <Label className={GlobalStyles['mt-6']} htmlFor="test_context">
                  Context
                  <CodeEditor
                    id="test_context"
                    name="test_context"
                    value={testContext}
                    onChange={(value: string) => {
                      dispatch(changeAccessPolicyTestContext(value));
                    }}
                    jsonExt
                  />
                </Label>
                <Label className={GlobalStyles['mt-3']}>
                  Parameters
                  <br />
                  <TextInput
                    id="test_params"
                    name="test_params"
                    value={testParams}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                      dispatch(changeAccessPolicyTestParams(e.target.value));
                    }}
                  />
                </Label>
                {testError && (
                  <InlineNotification
                    theme="alert"
                    className={`${GlobalStyles['mt-6']} ${GlobalStyles['mb-3']}`}
                  >
                    {testError}
                  </InlineNotification>
                )}
                <ButtonGroup>
                  <Button
                    theme="inverse"
                    disabled={isFetching || isSaving}
                    isLoading={isFetching || isSaving}
                    onClick={(e: React.MouseEvent) => {
                      e.preventDefault();
                      if (selectedTenantID && editableTemplate) {
                        dispatch(
                          runAccessPolicyTemplateTest(
                            editableTemplate,
                            testContext,
                            testParams,
                            selectedTenantID
                          )
                        );
                      }
                    }}
                  >
                    Run Test
                  </Button>
                </ButtonGroup>

                {testResult && (
                  <InlineNotification
                    theme={testResult.allowed ? 'success' : 'alert'}
                    className={`${GlobalStyles['mt-6']} ${GlobalStyles['mb-3']}`}
                  >
                    {testResult.allowed ? 'Access allowed' : 'Access denied'}
                  </InlineNotification>
                )}

                {testResult?.debug?.console && (
                  <InlineNotification
                    theme="info"
                    className={GlobalStyles['mt-3']}
                  >
                    <pre>{testResult.debug.console}</pre>
                  </InlineNotification>
                )}
              </CardRow>
            )}
          </Card>
        </>
      )}
    </form>
  );
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  editMode: state.policyTemplateEditMode,
  isSaving: state.savingPolicyTemplate,
  isFetching: state.fetchingPolicyTemplates,
  saveSuccess: state.policyTemplateSaveSuccess,
  saveError: state.policyTemplateSaveError,
  accessPolicy: state.currentAccessPolicy,
  testContext: state.accessPolicyTestContext,
  testParams: state.accessPolicyTestParams,
  testResult: state.testingPolicyResult,
  testError: state.testingPolicyError,
}))(PolicyTemplateForm);
