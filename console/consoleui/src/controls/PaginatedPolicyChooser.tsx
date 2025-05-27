import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  EmptyState,
  GlobalStyles,
  IconFileList2,
  InlineNotification,
  Radio,
  Table,
  TableHead,
  TableBody,
  TableFoot,
  TableRow,
  TableRowHead,
  TableCell,
  Text,
} from '@userclouds/ui-component-lib';

import { AppDispatch, RootState } from '../store';

import Pagination from './Pagination';
import Search from './Search';
import PaginatedResult from '../models/PaginatedResult';
import AccessPolicy, {
  AccessPolicyTemplate,
  ACCESS_POLICIES_PREFIX,
  ACCESS_POLICY_TEMPLATE_PREFIX,
  ACCESS_POLICIES_COLUMNS,
  ACCESS_POLICY_TEMPLATE_COLUMNS,
  PolicySelectorResourceType,
} from '../models/AccessPolicy';
import { Filter } from '../models/authz/SearchFilters';
import {
  changeAccessPolicySearchFilter,
  changeAccessPolicyTemplateSearchFilter,
  launchPolicyTemplateDialog,
  selectPolicyOrTemplateFromChooser,
} from '../actions/tokenizer';
import { fetchPolicyTemplates, fetchAccessPolicies } from '../thunks/tokenizer';
import PageCommon from '../pages/PageCommon.module.css';
import styles from './PaginatedPolicyChooser.module.css';

type SelectorProps = {
  selectedTenantID: string | undefined;
  policySelectorResourceType: PolicySelectorResourceType;
  policyToEdit: AccessPolicy;
  isFetching: boolean;
  policies: PaginatedResult<AccessPolicy> | undefined;
  templates: PaginatedResult<AccessPolicyTemplate> | undefined;
  onCancel: Function;
  changeComponents: Function;
  createNewPolicyTemplateHandler: Function;
  policyTemplateSearchFilter: Filter;
  policySearchFilter: Filter;
  policyFetchError: string | undefined;
  templateFetchError: string | undefined;
  selectedEntry: AccessPolicy | AccessPolicyTemplate | undefined;
  location: URL;
  tokenRes?: boolean;
  dispatch: AppDispatch;
};

const PaginatedPolicyChooser = ({
  selectedTenantID,
  policySelectorResourceType,
  policyToEdit,
  isFetching,
  policies,
  templates,
  onCancel,
  changeComponents,
  createNewPolicyTemplateHandler,
  policyTemplateSearchFilter,
  policySearchFilter,
  policyFetchError,
  templateFetchError,
  selectedEntry,
  location,
  tokenRes = false,
  dispatch,
}: SelectorProps) => {
  const selectingAccessPolicies =
    policySelectorResourceType === PolicySelectorResourceType.POLICY;

  const { hash } = location;
  useEffect(() => {
    // We're expecting this to be in a dialog, where we update the hash,
    // not the querystring, in order to avoid resetting ephemeral state.
    // In the future, we can handle both cases.
    const newParams = new URLSearchParams(hash.substring(1));
    if (selectedTenantID) {
      if (selectingAccessPolicies) {
        dispatch(
          fetchAccessPolicies(selectedTenantID, newParams, false, false, 10)
        );
      } else {
        dispatch(fetchPolicyTemplates(selectedTenantID, newParams, false, 10));
      }
    }
  }, [selectedTenantID, hash, selectingAccessPolicies, dispatch]);

  return (
    <>
      {selectingAccessPolicies ? (
        <>
          <div className={PageCommon.tablecontrols}>
            <Search
              id="policies"
              columns={ACCESS_POLICIES_COLUMNS}
              changeSearchFilter={(filter: Filter) => {
                dispatch(changeAccessPolicySearchFilter(filter));
              }}
              prefix={ACCESS_POLICIES_PREFIX}
              searchFilter={policySearchFilter}
              updateURL={false}
              dialog
            />
            <Pagination
              prev={policies?.prev}
              next={policies?.next}
              isLoading={isFetching}
              prefix={ACCESS_POLICIES_PREFIX}
              updateURL={false}
            />
          </div>
          {policies ? (
            policies.data && policies.data.length ? (
              <>
                <Table
                  id={
                    'paginatedPolicyChooserPolicies' +
                    (tokenRes ? 'TokenRes' : '')
                  }
                  className={styles.policychoosertable}
                >
                  <TableHead>
                    <TableRow key="paginatedPolicyChooserPolicyHeader">
                      <TableRowHead key="paginatedPolicyChooserHeaderPolicySelect">
                        Select
                      </TableRowHead>
                      <TableRowHead key="paginatedPolicyChooserHeaderPolicyName">
                        Name
                      </TableRowHead>
                      <TableRowHead key="paginatedPolicyChooserHeaderPolicyType">
                        Type
                      </TableRowHead>
                      <TableRowHead key="paginatedPolicyChooserHeaderPolicyVersion">
                        Version
                      </TableRowHead>
                      <TableRowHead key="paginatedPolicyChooserHeaderPolicyID">
                        ID
                      </TableRowHead>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {policies.data.map((entry) => (
                      <TableRow key={entry.id + '-' + entry.version}>
                        <TableCell>
                          <Radio
                            checked={entry.id === selectedEntry?.id}
                            onClick={() => {
                              dispatch(
                                selectPolicyOrTemplateFromChooser(entry)
                              );
                            }}
                          />
                        </TableCell>

                        <TableCell>
                          <Text>{entry.name || '[Unnamed policy]'}</Text>
                        </TableCell>
                        <TableCell>Policy</TableCell>
                        <TableCell>{entry.version}</TableCell>
                        <TableCell className={PageCommon.uuidtablecell}>
                          {entry.id}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                  <TableFoot>
                    <TableRow key={selectedEntry?.id || 'no-selection'}>
                      <TableCell />
                      <TableCell colSpan={4}>
                        {selectedEntry ? (
                          <>
                            <strong>Selected policy:</strong>{' '}
                            {selectedEntry.name || '[Unnammed]'}
                          </>
                        ) : (
                          'Select a policy above'
                        )}
                      </TableCell>
                    </TableRow>
                  </TableFoot>
                </Table>
                {(policies.has_prev || policies.has_next) && (
                  <Pagination
                    prev={policies.prev}
                    next={policies.next}
                    isLoading={isFetching}
                    prefix={ACCESS_POLICIES_PREFIX}
                    updateURL={false}
                  />
                )}
              </>
            ) : (
              <EmptyState
                title="No access policies"
                image={<IconFileList2 size="large" />}
              />
            )
          ) : policyFetchError ? (
            <InlineNotification theme="alert">
              {policyFetchError || 'Something went wrong'}
            </InlineNotification>
          ) : (
            <Text>Loading policies...</Text>
          )}
        </>
      ) : (
        <>
          <div className={PageCommon.tablecontrols}>
            <Search
              id="policies"
              columns={ACCESS_POLICY_TEMPLATE_COLUMNS}
              changeSearchFilter={(filter: Filter) => {
                dispatch(changeAccessPolicyTemplateSearchFilter(filter));
              }}
              prefix={ACCESS_POLICY_TEMPLATE_PREFIX}
              searchFilter={policyTemplateSearchFilter}
              updateURL={false}
              dialog
            />
            <Pagination
              prev={templates?.prev}
              next={templates?.next}
              isLoading={isFetching}
              prefix={ACCESS_POLICY_TEMPLATE_PREFIX}
              updateURL={false}
            />
          </div>
          {templates ? (
            templates.data && templates.data.length ? (
              <>
                <Table
                  id={
                    'paginatedPolicyChooserTemplates' +
                    (tokenRes ? 'TokenRes' : '')
                  }
                  className={styles.policychoosertable}
                >
                  <TableHead>
                    <TableRow key="paginatedPolicyChooserTemplateHeader">
                      <TableRowHead key="paginatedPolicyChooserHeaderTemplatesSelect">
                        Select
                      </TableRowHead>
                      <TableRowHead key="paginatedPolicyChooserHeaderTemplatesName">
                        Name
                      </TableRowHead>
                      <TableRowHead key="paginatedPolicyChooserHeaderTemplatesType">
                        Type
                      </TableRowHead>
                      <TableRowHead key="paginatedPolicyChooserHeaderTemplatesVersion">
                        Version
                      </TableRowHead>
                      <TableRowHead key="paginatedPolicyChooserHeaderTemplatesID">
                        ID
                      </TableRowHead>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {templates.data.map((entry) => (
                      <TableRow key={entry.id + '-' + entry.version}>
                        <TableCell>
                          <Radio
                            checked={entry.id === selectedEntry?.id}
                            onClick={() => {
                              dispatch(
                                selectPolicyOrTemplateFromChooser(entry)
                              );
                            }}
                          />
                        </TableCell>
                        <TableCell>
                          <Text>{entry.name || '[Unnamed template]'}</Text>
                        </TableCell>
                        <TableCell>Template</TableCell>
                        <TableCell>{entry.version}</TableCell>
                        <TableCell className={PageCommon.uuidtablecell}>
                          {entry.id}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                  <TableFoot>
                    <TableRow key={selectedEntry?.id || 'no-selection'}>
                      <TableCell />
                      <TableCell colSpan={4}>
                        {selectedEntry ? (
                          <>
                            <strong>Selected policy template:</strong>{' '}
                            {selectedEntry.name || '[Unnammed]'}
                          </>
                        ) : (
                          'Select a policy template above'
                        )}
                      </TableCell>
                    </TableRow>
                  </TableFoot>
                </Table>
                {(templates.has_prev || templates.has_next) && (
                  <Pagination
                    prev={templates.prev}
                    next={templates.next}
                    isLoading={isFetching}
                    prefix={ACCESS_POLICY_TEMPLATE_PREFIX}
                    updateURL={false}
                  />
                )}
              </>
            ) : (
              <EmptyState
                title="No policy templates"
                image={<IconFileList2 size="large" />}
              />
            )
          ) : templateFetchError ? (
            <InlineNotification theme="alert">
              {templateFetchError}
            </InlineNotification>
          ) : (
            <Text>Loading templates...</Text>
          )}
        </>
      )}
      {!selectingAccessPolicies && (
        <ButtonGroup className={GlobalStyles['mt-6']}>
          <Button
            theme="outline"
            isLoading={isFetching}
            disabled={isFetching}
            onClick={() => {
              createNewPolicyTemplateHandler();
              dispatch(launchPolicyTemplateDialog());
              onCancel();
            }}
          >
            Write New Template
          </Button>
        </ButtonGroup>
      )}
      <footer>
        <ButtonGroup>
          <Button
            theme="primary"
            isLoading={isFetching}
            disabled={isFetching || !selectedEntry}
            onClick={() => {
              const components = [...policyToEdit.components];
              const template = selectedEntry as AccessPolicyTemplate;
              if (template.function) {
                components.push({
                  template: selectedEntry as AccessPolicyTemplate,
                  template_parameters: '{}',
                });
              } else {
                components.push({
                  policy: selectedEntry as AccessPolicy,
                });
              }
              dispatch(
                changeComponents({
                  components: components,
                })
              );
              onCancel();
            }}
          >
            Save selection
          </Button>
        </ButtonGroup>
      </footer>
    </>
  );
};

const ConnectedPaginatedPolicyChooser = connect((state: RootState) => {
  return {
    selectedTenantID: state.selectedTenantID,
    selectedComponentPolicy: state.paginatedPolicyChooserComponent,
    policySearchFilter: state.accessPolicySearchFilter,
    policyTemplateSearchFilter: state.accessPolicyTemplateSearchFilter,
    policyFetchError: state.accessPolicyFetchError,
    templateFetchError: state.policyTemplatesFetchError,
    selectedEntry: state.paginatedPolicyChooserSelectedResource,
    location: state.location,
  };
})(PaginatedPolicyChooser);

export default ConnectedPaginatedPolicyChooser;
