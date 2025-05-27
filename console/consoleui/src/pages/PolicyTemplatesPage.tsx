import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  Card,
  Checkbox,
  EmptyState,
  IconFileList2,
  IconFilter,
  InlineNotification,
  Table,
  TableHead,
  TableRow,
  TableRowHead,
  TableBody,
  TableCell,
  Text,
  TextShortener,
  ToolTip,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import {
  bulkDeletePolicyTemplates,
  deleteSinglePolicyTemplate,
  fetchPolicyTemplates,
  fetchUserPolicyPermissions,
} from '../thunks/tokenizer';
import {
  togglePolicyTemplateForDelete,
  changeAccessPolicyTemplateSearchFilter,
} from '../actions/tokenizer';
import { AppDispatch, RootState } from '../store';
import PaginatedResult from '../models/PaginatedResult';
import {
  AccessPolicyTemplate,
  ACCESS_POLICY_TEMPLATE_COLUMNS,
  ACCESS_POLICY_TEMPLATE_PREFIX,
} from '../models/AccessPolicy';
import PermissionsOnObject from '../models/authz/Permissions';
import { Filter } from '../models/authz/SearchFilters';
import Link from '../controls/Link';
import Pagination from '../controls/Pagination';
import Search from '../controls/Search';
import DeleteWithConfirmationButton from '../controls/DeleteWithConfirmationButton';

import PageCommon from './PageCommon.module.css';
import styles from './PolicyTemplatesPage.module.css';

type PolicyTemplatesProps = {
  selectedTenantID: string | undefined;
  templates: PaginatedResult<AccessPolicyTemplate> | undefined;
  fetchError: string | undefined;
  isFetching: boolean;
  deleteQueue: string[];
  deleteSuccess: string;
  deleteErrors: string[];
  policyTemplateSearchFilter: Filter;
  query: URLSearchParams;
  dispatch: AppDispatch;
};
const PolicyTemplateList = ({
  selectedTenantID,
  templates,
  fetchError,
  isFetching,
  deleteQueue,
  deleteSuccess,
  deleteErrors,
  policyTemplateSearchFilter,
  query,
  dispatch,
}: PolicyTemplatesProps) => {
  const cleanQuery = makeCleanPageLink(query);

  const deletePrompt = `Are you sure you want to delete ${
    deleteQueue.length
  } policy template${deleteQueue.length === 1 ? '' : 's'}? This action is irreversible.`;

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div>
          <IconFilter />
        </div>
        <Search
          id="policyTemplates"
          columns={ACCESS_POLICY_TEMPLATE_COLUMNS}
          changeSearchFilter={(filter: Filter) => {
            dispatch(changeAccessPolicyTemplateSearchFilter(filter));
          }}
          prefix={ACCESS_POLICY_TEMPLATE_PREFIX}
          searchFilter={policyTemplateSearchFilter}
        />
        <ToolTip>
          <>
            {'Policy templates are functions to create access policies. '}
            <a
              href="https://docs.userclouds.com/docs/create-an-access-policy-copy"
              title="UserClouds documentation on creating access policies"
              target="new"
              className={PageCommon.link}
            >
              Learn more here.
            </a>
          </>
        </ToolTip>

        <Button
          theme="primary"
          size="small"
          className={PageCommon.listviewtablecontrolsButton}
        >
          <Link
            href={`/policytemplates/create${cleanQuery}`}
            title="Add a new template"
            applyStyles={false}
          >
            Create Template
          </Link>
        </Button>
      </div>

      {templates ? (
        templates.data && templates.data.length ? (
          <Card id="policyTemplates" listview>
            {!!deleteErrors.length && (
              <InlineNotification theme="alert" elementName="div">
                <ul>
                  {deleteErrors.map((error: string) => (
                    <li key={`policy_templates_error_${error}`}>{error}</li>
                  ))}
                </ul>
              </InlineNotification>
            )}
            {deleteSuccess && (
              <InlineNotification theme="success">
                {deleteSuccess}
              </InlineNotification>
            )}
            <div className={PageCommon.listviewpaginationcontrols}>
              <div className={PageCommon.listviewpaginationcontrolsdelete}>
                <DeleteWithConfirmationButton
                  id="deletePolicyTemplatesButton"
                  message={deletePrompt}
                  onConfirmDelete={() => {
                    if (selectedTenantID) {
                      dispatch(
                        bulkDeletePolicyTemplates(selectedTenantID, deleteQueue)
                      );
                    }
                  }}
                  title="Delete Policy Templates"
                  disabled={deleteQueue.length < 1}
                />
              </div>

              <Pagination
                prev={templates?.prev}
                next={templates?.next}
                isLoading={isFetching}
                prefix={ACCESS_POLICY_TEMPLATE_PREFIX}
              />
            </div>

            <Table
              spacing="packed"
              id="policyTemplates"
              className={styles.policytemplatestable}
            >
              <TableHead floating>
                <TableRow>
                  <TableRowHead>
                    <Checkbox
                      checked={
                        Object.keys(deleteQueue).length ===
                        templates.data.length
                      }
                      onChange={() => {
                        const shouldMarkForDelete = !deleteQueue.includes(
                          templates.data[0].id
                        );
                        templates.data.forEach((o) => {
                          if (
                            shouldMarkForDelete &&
                            !deleteQueue.includes(o.id)
                          ) {
                            dispatch(togglePolicyTemplateForDelete(o.id));
                          } else if (
                            !shouldMarkForDelete &&
                            deleteQueue.includes(o.id)
                          ) {
                            dispatch(togglePolicyTemplateForDelete(o.id));
                          }
                        });
                      }}
                    />
                  </TableRowHead>
                  <TableRowHead>Name</TableRowHead>
                  <TableRowHead>Description</TableRowHead>
                  <TableRowHead>Version</TableRowHead>
                  <TableRowHead>ID</TableRowHead>
                  <TableRowHead key="purpose_delete" />
                </TableRow>
              </TableHead>
              <TableBody>
                {templates.data.map((entry) => (
                  <TableRow
                    key={entry.id + '-' + entry.version}
                    className={
                      (deleteQueue.includes(entry.id)
                        ? PageCommon.queuedfordelete
                        : '') +
                      ' ' +
                      PageCommon.listviewtablerow
                    }
                  >
                    <TableCell>
                      <Checkbox
                        id={'delete' + entry.id}
                        name="delete object"
                        checked={deleteQueue.includes(entry.id)}
                        onChange={() => {
                          dispatch(togglePolicyTemplateForDelete(entry.id));
                        }}
                      />
                    </TableCell>
                    <TableCell>
                      <Link
                        href={`/policytemplates/${
                          entry.id
                        }/${entry.version.toString()}${cleanQuery}`}
                      >
                        <Text>{entry.name || '[Unnamed policy]'}</Text>
                      </Link>
                    </TableCell>
                    <TableCell>
                      <Text>{entry.description}</Text>
                    </TableCell>
                    <TableCell>{entry.version}</TableCell>
                    <TableCell>
                      <TextShortener text={entry.id} length={6} />
                    </TableCell>
                    <TableCell
                      align="right"
                      className={PageCommon.listviewtabledeletecell}
                    >
                      <DeleteWithConfirmationButton
                        id="deletePolicyTemplateButton"
                        message="Are you sure you want to delete this policy template? This action is irreversible."
                        onConfirmDelete={() => {
                          if (selectedTenantID) {
                            dispatch(
                              deleteSinglePolicyTemplate(
                                selectedTenantID,
                                entry
                              )
                            );
                          }
                        }}
                        title="Delete Policy Template"
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Card>
        ) : (
          <Card>
            <EmptyState
              title="No policy templates"
              image={<IconFileList2 size="large" />}
            >
              <Button theme="secondary">
                <Link
                  href={`/policytemplates/create${cleanQuery}`}
                  applyStyles={false}
                >
                  Add Policy Template
                </Link>
              </Button>
            </EmptyState>
          </Card>
        )
      ) : isFetching ? (
        <Text>Loading policy templates...</Text>
      ) : (
        <InlineNotification theme="alert">
          {fetchError || 'Something went wrong'}
        </InlineNotification>
      )}
    </>
  );
};
const ConnectedPolicyTemplateList = connect((state: RootState) => {
  return {
    selectedTenantID: state.selectedTenantID,
    templates: state.policyTemplates,
    fetchError: state.policyTemplatesFetchError,
    isFetching: state.fetchingPolicyTemplates,
    deleteQueue: state.policyTemplatesDeleteQueue,
    deleteSuccess: state.deletePolicyTemplatesSuccess,
    deleteErrors: state.deletePolicyTemplatesErrors,
    policyTemplateSearchFilter: state.accessPolicyTemplateSearchFilter,
    query: state.query,
  };
})(PolicyTemplateList);

const PolicyTemplatesPage = ({
  selectedTenantID,
  userPolicyPermissions,
  permissionsFetchError,
  query,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  userPolicyPermissions: PermissionsOnObject | undefined;
  permissionsFetchError: string;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenantID) {
      dispatch(fetchUserPolicyPermissions(selectedTenantID));
    }
  }, [dispatch, selectedTenantID]);

  useEffect(() => {
    if (selectedTenantID && userPolicyPermissions?.read) {
      dispatch(fetchPolicyTemplates(selectedTenantID, query));
    }
  }, [dispatch, userPolicyPermissions, selectedTenantID, query]);

  return (
    <>
      {userPolicyPermissions?.read ? (
        <ConnectedPolicyTemplateList />
      ) : userPolicyPermissions ? (
        <Card
          title="Request access"
          description="You do not have permission to view any policies. Please contact your administrator to request access."
        >
          {permissionsFetchError && (
            <InlineNotification theme="alert">
              {permissionsFetchError}
            </InlineNotification>
          )}
        </Card>
      ) : (
        <Card
          title="Loading..."
          description="Loading policies and permissions."
        />
      )}
    </>
  );
};

export default connect((state: RootState) => {
  return {
    selectedTenantID: state.selectedTenantID,
    userPolicyPermissions: state.userPolicyPermissions,
    permissionsFetchError: state.userPolicyPermissionsFetchError,
    accessEditMode: state.accessPolicyEditMode,
    templatesEditMode: state.policyTemplateEditMode,
    transformerEditMode: state.transformerEditMode,
    location: state.location,
    query: state.query,
  };
})(PolicyTemplatesPage);
