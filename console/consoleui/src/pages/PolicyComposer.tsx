import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Dialog,
  DialogBody,
  GlobalStyles,
  IconButton,
  IconDeleteBin,
  InlineNotification,
  InputReadOnly,
  Select,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  Text,
  TextInput,
} from '@userclouds/ui-component-lib';
import { RootState, AppDispatch } from '../store';
import AccessPolicy, {
  AccessPolicyTemplate,
  AccessPolicyType,
  getNameForAccessPolicyComponent,
  isTemplate,
  blankPolicy,
  PolicySelectorResourceType,
} from '../models/AccessPolicy';
import PaginatedResult from '../models/PaginatedResult';
import {
  changePolicyComponents,
  launchPolicyChooserForAccessPolicy,
  launchPolicyChooserForPolicyTemplate,
  closePolicyChooser,
} from '../actions/tokenizer';
import { makeCleanPageLink } from '../AppNavigation';
import Styles from './PolicyComposer.module.css';
import Link from '../controls/Link';
import ConnectedPaginatedPolicyChooser from '../controls/PaginatedPolicyChooser';

const PolicyComposer = ({
  selectedTenantID,
  policy,
  policies,
  templates,
  changeAccessPolicyAction,
  readOnly = false,
  tokenRes = false,
  tableID,
  query,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  policy: AccessPolicy | undefined;
  policies: PaginatedResult<AccessPolicy> | undefined;
  templates: PaginatedResult<AccessPolicyTemplate> | undefined;
  changeAccessPolicyAction: Function;
  readOnly?: boolean;
  tokenRes?: boolean;
  tableID?: string;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenantID) {
      if (policies || templates) {
        dispatch(
          changePolicyComponents(
            policies ? policies.data : ([] as AccessPolicy[]),
            templates ? templates.data : ([] as AccessPolicyTemplate[])
          )
        );
      }
      if (!policy) {
        dispatch(changeAccessPolicyAction(blankPolicy()));
      }
    }
  }, [
    selectedTenantID,
    dispatch,
    policies,
    templates,
    policy,
    changeAccessPolicyAction,
  ]);

  const cleanQuery = makeCleanPageLink(query);
  const dialogID = tokenRes
    ? 'paginatedTokenPolicyChooser'
    : 'paginatedAccessPolicyChooser';

  if (!policy || !policy.components) {
    return (
      <InlineNotification theme="alert">
        Unable to {readOnly ? 'display' : 'create'} composite policy.
      </InlineNotification>
    );
  }
  return policy ? (
    <>
      {policy.components && (
        <Table
          id={tableID}
          spacing="packed"
          className={Styles.policycomposertable}
        >
          <TableHead>
            <TableRow key="head">
              <TableRowHead />
              {policy.components.length > 0 && (
                <TableRowHead>Policy name</TableRowHead>
              )}
              {policy.components.length > 0 && (
                <TableRowHead>Parameters</TableRowHead>
              )}
              {policy.components.length > 0 && !readOnly && (
                <TableRowHead>Delete</TableRowHead>
              )}
            </TableRow>
          </TableHead>

          <TableBody>
            {policy.components.length ? (
              policy.components.map((component, i) => (
                <TableRow
                  key={
                    'template' in component
                      ? component.template.id
                      : component.policy.id
                  }
                >
                  <TableCell>
                    {i === 0 ? (
                      <Text className={Styles.whereText}>Where</Text>
                    ) : i === 1 && !readOnly ? (
                      <Select
                        value={policy.policy_type}
                        name={
                          'policyTypeSelector' + (tokenRes ? 'TokenRes' : '')
                        }
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                          dispatch(
                            changeAccessPolicyAction({
                              policy_type: e.target.value,
                            })
                          );
                        }}
                      >
                        <option
                          value={AccessPolicyType.AND}
                          key={AccessPolicyType.AND}
                        >
                          AND
                        </option>
                        <option
                          value={AccessPolicyType.OR}
                          key={AccessPolicyType.OR}
                        >
                          OR
                        </option>
                      </Select>
                    ) : (
                      <Text>
                        {policy.policy_type === AccessPolicyType.AND
                          ? 'AND'
                          : 'OR'}
                      </Text>
                    )}
                  </TableCell>
                  <TableCell>
                    {'template' in component ? (
                      <Link
                        href={`/policytemplates/${component.template.id}/latest${cleanQuery}`}
                        title="View details for this policy template"
                      >
                        {getNameForAccessPolicyComponent(component)}
                      </Link>
                    ) : (
                      <Link
                        href={`/accesspolicies/${component.policy.id}/latest${cleanQuery}`}
                        title="View details for this policy"
                      >
                        {getNameForAccessPolicyComponent(component)}
                      </Link>
                    )}
                  </TableCell>
                  <TableCell>
                    {isTemplate(component) ? (
                      readOnly ? (
                        <InputReadOnly>
                          {'template_parameters' in component
                            ? component.template_parameters
                            : ''}
                        </InputReadOnly>
                      ) : (
                        <TextInput
                          id={'templateparams' + i}
                          name={'templateparams' + i}
                          placeholder="{json:'json'}"
                          value={
                            'template_parameters' in component
                              ? component.template_parameters
                              : ''
                          }
                          onChange={(
                            e: React.ChangeEvent<HTMLInputElement>
                          ) => {
                            if ('template' in component) {
                              const newVal = {
                                template: component.template,
                                template_parameters: e.target.value,
                              };
                              const newComponents = [...policy.components];
                              newComponents[i] = newVal;
                              dispatch(
                                changeAccessPolicyAction({
                                  components: newComponents,
                                })
                              );
                            }
                          }}
                        />
                      )
                    ) : readOnly ? (
                      <InputReadOnly>N/A</InputReadOnly>
                    ) : (
                      <TextInput disabled placeholder="N/A" />
                    )}
                  </TableCell>
                  {!readOnly && (
                    <TableCell>
                      <IconButton
                        icon={<IconDeleteBin />}
                        onClick={() => {
                          const newComponents = [...policy.components];
                          newComponents.splice(i, 1);
                          dispatch(
                            changeAccessPolicyAction({
                              components: newComponents,
                            })
                          );
                        }}
                        title="Delete Component"
                        aria-label="Delete Component"
                      />
                    </TableCell>
                  )}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={4}>No policy components yet.</TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      )}
      {!readOnly && (
        <ButtonGroup className={GlobalStyles['mt-6']}>
          <Button
            onClick={() => {
              dispatch(launchPolicyChooserForAccessPolicy());
              const dialog: HTMLDialogElement | null = document.getElementById(
                dialogID
              ) as HTMLDialogElement;
              dialog.showModal();
            }}
            theme="secondary"
            id={'addPolicy' + (tokenRes ? 'TokenRes' : '')}
          >
            Add Policy
          </Button>
          <Button
            onClick={() => {
              dispatch(launchPolicyChooserForPolicyTemplate());
              const dialog: HTMLDialogElement | null = document.getElementById(
                dialogID
              ) as HTMLDialogElement;
              dialog.showModal();
            }}
            theme="secondary"
            id={'addTemplate' + (tokenRes ? 'TokenRes' : '')}
          >
            Add Template
          </Button>
        </ButtonGroup>
      )}
    </>
  ) : (
    <Text>
      No Access Policies or Access Policy Templates available for composition.
    </Text>
  );
};

const ConnectedPolicyComposer = connect((state: RootState) => {
  return {
    selectedTenantID: state.selectedTenantID,
    policies: state.accessPolicies,
    templates: state.policyTemplates,
    isBusy: state.savingAccessPolicy || state.testingPolicy,
    fetchError: state.accessPolicyFetchError,
    saveError: state.saveAccessPolicyError,
    testContext: state.accessPolicyTestContext,
    testResult: state.testingPolicyResult,
    testError: state.testingPolicyError,
    newTemplate: state.policyTemplateToCreate,
    query: state.query,
  };
})(PolicyComposer);

const PolicyChooserDialog = ({
  policy,
  policies,
  templates,
  tokenRes,
  policySelectorResourceType,
  policyChooserIsOpen,
  changeAccessPolicyAction,
  createNewPolicyTemplateHandler,
  dispatch,
}: {
  policy: AccessPolicy | undefined;
  policies: PaginatedResult<AccessPolicy> | undefined;
  templates: PaginatedResult<AccessPolicyTemplate> | undefined;
  tokenRes?: boolean;
  policySelectorResourceType: PolicySelectorResourceType;
  policyChooserIsOpen: boolean;
  changeAccessPolicyAction: Function;
  createNewPolicyTemplateHandler: Function;
  dispatch: AppDispatch;
}) => {
  const dialogID = tokenRes
    ? 'paginatedTokenPolicyChooser'
    : 'paginatedAccessPolicyChooser';
  const selectingAccessPolicies =
    policySelectorResourceType === PolicySelectorResourceType.POLICY;
  return (
    <Dialog
      id={dialogID}
      title={`Add ${selectingAccessPolicies ? 'Policy' : 'Template'}`}
      description={
        'Add a' +
        (selectingAccessPolicies
          ? 'n access policy to your AND / OR composition to the underlying composite access policy. '
          : ' parametrizable template to your AND / OR composition. Alternatively, click “Write New Template” to create a completely new function.')
      }
      fullPage
    >
      {policy && policyChooserIsOpen && (
        <DialogBody>
          <ConnectedPaginatedPolicyChooser
            policySelectorResourceType={policySelectorResourceType}
            isFetching={false}
            policies={policies}
            templates={templates}
            policyToEdit={policy}
            changeComponents={changeAccessPolicyAction}
            onCancel={() => {
              const dialog: HTMLDialogElement | null = document.getElementById(
                dialogID
              ) as HTMLDialogElement;
              dialog.close();
              dispatch(closePolicyChooser());
            }}
            createNewPolicyTemplateHandler={createNewPolicyTemplateHandler}
            tokenRes={tokenRes}
          />
        </DialogBody>
      )}
    </Dialog>
  );
};

// Exporting the dialog separately lets us place it where we want in the DOM
// and thus avoid nesting forms
export const ConnectedPolicyChooserDialog = connect((state: RootState) => ({
  policies: state.accessPolicies,
  templates: state.policyTemplates,
  policyChooserIsOpen: state.policyChooserIsOpen,
  policySelectorResourceType: state.policySelectorResourceType,
}))(PolicyChooserDialog);

export default ConnectedPolicyComposer;
