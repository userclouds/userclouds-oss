import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  GlobalStyles,
  Button,
  ButtonGroup,
  Card,
  CardRow,
  Checkbox,
  CodeEditor,
  FormNote,
  Label,
  InlineNotification,
  InputReadOnly,
  Select,
  TextArea,
  TextInput,
  Text,
  ToolTip,
} from '@userclouds/ui-component-lib';

import { redirect } from '../routing';
import { RootState, AppDispatch } from '../store';
import { VALID_NAME_PATTERN } from '../models/helpers';
import Transformer, {
  blankTransformer,
  TransformType,
  TransformTypeFriendly,
  TokenizingTransformerTypes,
  TransformerTestResult,
} from '../models/Transformer';
import PermissionsOnObject from '../models/authz/Permissions';
import {
  changeTransformer,
  changeTransformerTestData,
  getTransformerSuccess,
  toggleTransformerDetailsEditMode,
} from '../actions/tokenizer';
import {
  createTransformer,
  fetchTransformer,
  fetchUserPermissionsForTransformer,
  runTransformerTest,
  updateTransformer,
} from '../thunks/tokenizer';
import PageCommon from './PageCommon.module.css';
import { fetchDataTypes } from '../thunks/userstore';
import PaginatedResult from '../models/PaginatedResult';
import { DataType, getDataTypeIDFromName } from '../models/DataType';
import { NativeDataTypes } from '../models/TenantUserStoreConfig';
import { PageTitle } from '../mainlayout/PageWrap';

const TransformerDeveloper = ({
  selectedCompanyID,
  selectedTenantID,
  transformerPermissions,
  transformer,
  modifiedTransformer,
  isDirty,
  isBusy,
  fetchError,
  successMessage,
  dataTypes,
  saveError,
  testData,
  testResult,
  testError,
  deleteError,
  routeParams,
  editMode,
  isNew,
  dispatch,
}: {
  selectedCompanyID: string | undefined;
  selectedTenantID: string | undefined;
  transformerPermissions: PermissionsOnObject | undefined;
  transformer: Transformer | undefined;
  modifiedTransformer: Transformer | undefined;
  isDirty: boolean;
  isBusy: boolean;
  fetchError: string | undefined;
  successMessage: string;
  dataTypes: PaginatedResult<DataType> | undefined;
  saveError: string;
  testData: string;
  testResult: TransformerTestResult | undefined;
  testError: string;
  deleteError: string;
  routeParams: Record<string, string>;
  editMode: boolean;
  isNew: boolean;
  dispatch: AppDispatch;
}) => {
  const { transformerID } = routeParams;
  return selectedCompanyID &&
    selectedTenantID &&
    transformer &&
    transformerPermissions ? (
    <form
      onSubmit={(e: React.FormEvent) => {
        e.preventDefault();
        if (modifiedTransformer) {
          if (transformerID) {
            dispatch(
              updateTransformer(
                selectedCompanyID,
                selectedTenantID,
                modifiedTransformer
              )
            );
          } else {
            dispatch(
              createTransformer(
                selectedCompanyID,
                selectedTenantID,
                modifiedTransformer
              )
            );
          }
        }
      }}
    >
      <div className={PageCommon.listviewtablecontrols}>
        <PageTitle
          title={isNew ? 'Create Transformer' : 'Transformer Details'}
          itemName={isNew ? 'New Transformer' : transformer.name}
        />

        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              {
                'Name, describe and define your transformer so that it is easy to find and use later. '
              }
              <a
                href="https://docs.userclouds.com/docs/transformers-1"
                title="UserClouds documentation for key concepts about transformers"
                target="new"
                className={PageCommon.link}
              >
                Learn more here.
              </a>
            </>
          </ToolTip>
        </div>

        {editMode ? (
          <>
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
                Save Transformer
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
                          `/transformers?company_id=${selectedCompanyID}&tenant_id=${selectedTenantID}`
                        )
                      : dispatch(toggleTransformerDetailsEditMode(false));
                  }
                }}
              >
                Cancel
              </Button>
            </ButtonGroup>
          </>
        ) : (
          <>
            <ButtonGroup>
              {transformerPermissions.update && !transformer.is_system && (
                <Button
                  theme="primary"
                  size="small"
                  onClick={(e: React.MouseEvent) => {
                    e.preventDefault();
                    dispatch(toggleTransformerDetailsEditMode(true));
                  }}
                >
                  Edit transformer
                </Button>
              )}
            </ButtonGroup>
          </>
        )}
      </div>

      <Card
        lockedMessage={
          transformer
            ? transformer.is_system
              ? 'System transformer details cannot be edited'
              : transformerPermissions && !transformerPermissions.update
                ? 'You do not have edit access'
                : ''
            : ''
        }
        title=""
        detailview
      >
        {deleteError && (
          <InlineNotification
            theme="alert"
            className={`${GlobalStyles['mt-6']} ${GlobalStyles['mb-3']}`}
          >
            {deleteError}
          </InlineNotification>
        )}
        {saveError && (
          <InlineNotification
            theme="alert"
            className={`${GlobalStyles['mt-6']} ${GlobalStyles['mb-3']}`}
          >
            {saveError}
          </InlineNotification>
        )}

        <CardRow
          title="Basic Details"
          tooltip={<>View and edit this transformer.</>}
          collapsible
        >
          <Label className={GlobalStyles['mt-6']}>
            Name
            <br />
            {editMode ? (
              <TextInput
                id="transformer_name"
                name="name"
                pattern={VALID_NAME_PATTERN}
                value={
                  modifiedTransformer
                    ? modifiedTransformer.name
                    : transformer.name
                }
                required
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                  dispatch(
                    changeTransformer({
                      name: e.target.value,
                    })
                  );
                }}
              />
            ) : (
              <InputReadOnly>{transformer.name}</InputReadOnly>
            )}
          </Label>
          <Label className={GlobalStyles['mt-6']}>
            Description
            <br />
            {editMode ? (
              <TextInput
                id="transformer_description"
                name="description"
                value={
                  modifiedTransformer
                    ? modifiedTransformer.description
                    : transformer.description
                }
                required
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                  dispatch(
                    changeTransformer({
                      description: e.target.value,
                    })
                  );
                }}
              />
            ) : (
              <InputReadOnly>{transformer.description}</InputReadOnly>
            )}
          </Label>
        </CardRow>
        <CardRow
          title="Data Type"
          tooltip={
            <>Select the input and output datatypes for this transformer.</>
          }
          collapsible
        >
          <fieldset className={PageCommon.carddetailsrow}>
            <Label className={GlobalStyles['mt-6']}>
              Transform Type
              <br />
              {editMode ? (
                <Select
                  id="transformer_transform_type"
                  name="transform_type"
                  value={
                    modifiedTransformer
                      ? modifiedTransformer.transform_type
                      : transformer.transform_type
                  }
                  required
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    dispatch(
                      changeTransformer({
                        transform_type: e.target.value,
                      })
                    );
                  }}
                >
                  {(
                    Object.keys(TransformType) as Array<
                      keyof typeof TransformType
                    >
                  ).map((type) => (
                    <option value={TransformType[type]} key={type}>
                      {TransformTypeFriendly[TransformType[type]]}
                    </option>
                  ))}
                </Select>
              ) : (
                <InputReadOnly>
                  {TransformTypeFriendly[transformer.transform_type]}
                </InputReadOnly>
              )}
            </Label>

            {(modifiedTransformer
              ? modifiedTransformer.transform_type === TransformType.PassThrough
              : transformer.transform_type === TransformType.PassThrough) && (
              <FormNote>
                For "passthrough" transformers, input and output data types are
                automatically inferred.
              </FormNote>
            )}
            <Label className={GlobalStyles['mt-6']}>
              Reuse existing token?
              <br />
              {editMode &&
              modifiedTransformer &&
              TokenizingTransformerTypes.includes(
                modifiedTransformer.transform_type
              ) ? (
                <Checkbox
                  id="reuse_token"
                  name="reuse_token"
                  checked={
                    modifiedTransformer
                      ? modifiedTransformer.reuse_existing_token
                      : transformer.reuse_existing_token
                  }
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    dispatch(
                      changeTransformer({
                        reuse_existing_token: e.currentTarget.checked,
                      })
                    );
                  }}
                />
              ) : (
                <InputReadOnly
                  type="checkbox"
                  isLocked={
                    !transformerPermissions.update ||
                    !TokenizingTransformerTypes.includes(
                      transformer.transform_type
                    )
                  }
                  isChecked={transformer.reuse_existing_token}
                  title={
                    !transformerPermissions.update
                      ? 'You do not have permission to edit this transformer.'
                      : 'This setting may only be turned on for tokenizing transformers.'
                  }
                />
              )}
            </Label>
            <Label className={GlobalStyles['mt-6']}>
              Input Data Type
              <br />
              {editMode ? (
                <Select
                  id="transformer_input_data_type"
                  name="input_data_type"
                  value={
                    (modifiedTransformer || transformer).transform_type ===
                    TransformType.PassThrough
                      ? NativeDataTypes.String
                      : (modifiedTransformer || transformer).input_data_type
                          .name
                  }
                  disabled={
                    modifiedTransformer
                      ? modifiedTransformer.transform_type ===
                        TransformType.PassThrough
                      : transformer.transform_type === TransformType.PassThrough
                  }
                  required
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    const name = e.target.value;
                    dataTypes &&
                      dispatch(
                        changeTransformer({
                          input_data_type: {
                            name: name,
                            id: getDataTypeIDFromName(name, dataTypes?.data),
                          },
                        })
                      );
                  }}
                >
                  {dataTypes?.data.map((type) => (
                    <option value={type.name} key={type.id}>
                      {type.name}
                    </option>
                  ))}
                </Select>
              ) : (
                <InputReadOnly>
                  {transformer.input_data_type.name}
                </InputReadOnly>
              )}
            </Label>
            <Label className={GlobalStyles['mt-6']}>
              Output Data Type
              <br />
              {editMode ? (
                <Select
                  id="transformer_output_data_type"
                  name="output_data_type"
                  value={
                    (modifiedTransformer || transformer).transform_type ===
                    TransformType.PassThrough
                      ? NativeDataTypes.String
                      : (modifiedTransformer || transformer).output_data_type
                          .name
                  }
                  disabled={
                    modifiedTransformer
                      ? modifiedTransformer.transform_type ===
                        TransformType.PassThrough
                      : transformer.transform_type === TransformType.PassThrough
                  }
                  required
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    const name = e.target.value;
                    dataTypes &&
                      dispatch(
                        changeTransformer({
                          output_data_type: {
                            name: name,
                            id: getDataTypeIDFromName(name, dataTypes?.data),
                          },
                        })
                      );
                  }}
                >
                  {dataTypes?.data.map((type) => (
                    <option value={type.name} key={type.id}>
                      {type.name}
                    </option>
                  ))}
                </Select>
              ) : (
                <InputReadOnly>
                  {transformer.output_data_type.name}
                </InputReadOnly>
              )}
            </Label>
          </fieldset>
        </CardRow>
        <CardRow
          title="Function and Parameters"
          tooltip={
            <>Define the function and parameterization of this transformer.</>
          }
          collapsible
        >
          {(modifiedTransformer || transformer).transform_type !==
            TransformType.PassThrough && (
            <fieldset>
              <Label
                className={GlobalStyles['mt-6']}
                htmlFor="transformer_function"
              >
                Function
                <br />
                <CodeEditor
                  value={
                    modifiedTransformer
                      ? modifiedTransformer.function
                      : transformer.function
                  }
                  id="transformer_function"
                  name="function"
                  disabled={!editMode}
                  readOnly={!editMode}
                  onChange={(value: string) => {
                    dispatch(
                      changeTransformer({
                        function: value,
                      })
                    );
                  }}
                  javascriptExt
                />
              </Label>

              <Label
                className={GlobalStyles['mt-6']}
                htmlFor="transformer_params"
              >
                Parameters{editMode && ' (JSON dictionary or array) '}
                <br />
                <CodeEditor
                  value={
                    modifiedTransformer
                      ? modifiedTransformer.parameters
                      : transformer.parameters
                  }
                  id="transformer_params"
                  name="params"
                  disabled={!editMode}
                  readOnly={!editMode}
                  onChange={(value: string) => {
                    dispatch(changeTransformer({ parameters: value }));
                  }}
                  jsonExt
                />
                <FormNote>Parameters are identical on every execution</FormNote>
              </Label>
            </fieldset>
          )}
        </CardRow>
        {editMode && (
          <CardRow
            title="Test Transformer"
            tooltip={
              <>Define test data and test your transformer code and settings.</>
            }
            collapsible
          >
            <fieldset>
              <Label className={GlobalStyles['mt-6']} htmlFor="test_data">
                Test data
                <br />
                <TextArea
                  value={testData}
                  id="test_data"
                  name="test_data"
                  onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => {
                    dispatch(changeTransformerTestData(e.target.value));
                  }}
                />
              </Label>
              <Label className={GlobalStyles['mt-6']} htmlFor="test_result">
                Test result
                <br />
                <TextInput
                  value={testResult ? testResult.value : ''}
                  id="test_result"
                  name="test_result"
                  readOnly
                />
                {testResult?.debug?.console && (
                  <InlineNotification
                    theme="info"
                    className={GlobalStyles['mt-6']}
                  >
                    <pre>{testResult.debug.console}</pre>
                  </InlineNotification>
                )}
              </Label>
              <Button
                theme="inverse"
                disabled={isBusy}
                isLoading={isBusy}
                onClick={(e: React.MouseEvent) => {
                  e.preventDefault();
                  dispatch(
                    runTransformerTest(
                      modifiedTransformer || transformer,
                      testData,
                      selectedTenantID
                    )
                  );
                }}
                className={GlobalStyles['mt-3']}
              >
                Run Test
              </Button>
            </fieldset>
          </CardRow>
        )}
        {editMode && (
          <>
            {testError && (
              <InlineNotification
                theme="alert"
                className={`${GlobalStyles['mt-6']} ${GlobalStyles['mb-3']}`}
              >
                {testError}
              </InlineNotification>
            )}
            {successMessage && (
              <InlineNotification
                theme="success"
                className={`${GlobalStyles['mt-6']} ${GlobalStyles['mb-3']}`}
              >
                {successMessage}
              </InlineNotification>
            )}
          </>
        )}
      </Card>
    </form>
  ) : fetchError ? (
    <InlineNotification
      theme="alert"
      className={`${GlobalStyles['mt-6']} ${GlobalStyles['mb-3']}`}
    >
      {fetchError}
    </InlineNotification>
  ) : (
    <Text>Fetching transformer...</Text>
  );
};
const ConnectedTransformerDeveloper = connect((state: RootState) => {
  return {
    selectedCompanyID: state.selectedCompanyID,
    selectedTenantID: state.selectedTenantID,
    transformerPermissions: state.transformerPermissions,
    transformer: state.currentTransformer,
    modifiedTransformer: state.modifiedTransformer,
    isDirty: state.transformerIsDirty,
    isBusy: state.savingTransformer || state.testingTransformer,
    fetchError: state.transformerFetchError,
    successMessage: state.saveTransformerSuccess,
    dataTypes: state.dataTypes,
    saveError: state.saveTransformerError,
    testData: state.transformerTestData,
    testResult: state.testingTransformerResult,
    testError: state.testingTransformerError,
    deleteError: state.transformerDeleteError,
    routeParams: state.routeParams,
  };
})(TransformerDeveloper);

const TokenizerTransformersPage = ({
  selectedCompanyID,
  selectedTenantID,
  transformerPermissions,
  location,
  editMode,
  routeParams,
  dispatch,
}: {
  selectedCompanyID: string | undefined;
  selectedTenantID: string | undefined;
  transformerPermissions: PermissionsOnObject | undefined;
  location: URL;
  editMode: boolean;
  routeParams: Record<string, string>;
  dispatch: AppDispatch;
}) => {
  const { pathname } = location;
  const isNewPage = pathname.indexOf('create') > -1;
  const { transformerID, version } = routeParams;

  useEffect(() => {
    if (selectedTenantID) {
      dispatch(
        fetchUserPermissionsForTransformer(
          selectedTenantID,
          transformerID || ''
        )
      );
    }
  }, [selectedTenantID, transformerID, dispatch]);

  useEffect(() => {
    if (selectedTenantID) {
      dispatch(fetchDataTypes(selectedTenantID, new URLSearchParams()));
    }
  }, [selectedTenantID, dispatch]);

  useEffect(() => {
    if (selectedCompanyID && selectedTenantID) {
      if (transformerPermissions) {
        if (!transformerID) {
          if (!transformerPermissions.create) {
            // shouldn't be on this page w/o a transformer ID if user can't create new transformers
            redirect(
              `/transformers?company_id=${selectedCompanyID}&tenant_id=${selectedTenantID}`
            );
          }
          dispatch(getTransformerSuccess(blankTransformer()));
        } else if (version && version !== 'latest') {
          const numVersion = parseInt(version, 10);
          if (isNaN(numVersion)) {
            redirect(
              `/transformers?company_id=${selectedCompanyID}&tenant_id=${selectedTenantID}`
            );
          }
          dispatch(
            fetchTransformer(selectedTenantID, transformerID, numVersion)
          );
        } else {
          dispatch(fetchTransformer(selectedTenantID, transformerID));
        }
      }
    }
  }, [
    transformerID,
    selectedTenantID,
    transformerPermissions,
    selectedCompanyID,
    dispatch,
    version,
  ]);

  return (
    <ConnectedTransformerDeveloper
      editMode={editMode || isNewPage}
      isNew={isNewPage}
    />
  );
};

export default connect((state: RootState) => ({
  selectedCompanyID: state.selectedCompanyID,
  selectedTenantID: state.selectedTenantID,
  transformerPermissions: state.transformerPermissions,
  location: state.location,
  editMode: state.transformerDetailsEditMode,
  routeParams: state.routeParams,
}))(TokenizerTransformersPage);
