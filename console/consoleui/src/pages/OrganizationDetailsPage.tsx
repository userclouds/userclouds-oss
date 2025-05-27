import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  InlineNotification,
  InputReadOnly,
  Label,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  Text,
  TextInput,
  TextShortener,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';

import { makeCleanPageLink } from '../AppNavigation';
import { RootState, AppDispatch } from '../store';
import PaginatedResult from '../models/PaginatedResult';
import Organization from '../models/Organization';
import LoginApp from '../models/LoginApp';
import {
  getOrganizationRequest,
  getOrganizationSuccess,
  getOrganizationError,
  toggleOrganizationEditMode,
  updateOrganizationRequest,
  updateOrganizationSuccess,
  updateOrganizationError,
  getLoginAppsForOrgRequest,
  getLoginAppsForOrgSuccess,
  getLoginAppsForOrgError,
  modifyOrganization,
} from '../actions/organizations';
import { fetchOrganization, updateOrganization } from '../API/organizations';
import { fetchLoginApps } from '../API/loginapps';
import Link from '../controls/Link';
import { UserList } from './UsersPage';
import { PageTitle } from '../mainlayout/PageWrap';
import PageCommon from './PageCommon.module.css';
import styles from './OrganizationDetailsPage.module.css';

const ConnectedUserList = connect((state: RootState) => {
  return {
    selectedTenant: state.selectedTenant,
    users: state.tenantUsers,
    fetchError: state.fetchUsersError,
    deleteQueue: state.userDeleteQueue,
    bulkSaveErrors: state.userBulkSaveErrors,
    savingUsers: state.savingUsers,
    query: state.query,
    userStoreColumns: state.userStoreColumns,
  };
})(UserList);

const fetchOrg =
  (tenantID: string, orgID: string) => (dispatch: AppDispatch) => {
    dispatch(getOrganizationRequest());
    fetchOrganization(tenantID, orgID).then(
      (result: Organization) => {
        dispatch(getOrganizationSuccess(result));
      },
      (error: APIError) => {
        dispatch(getOrganizationError(error));
      }
    );
  };

const saveOrg =
  (tenantID: string, org: Organization) => (dispatch: AppDispatch) => {
    dispatch(updateOrganizationRequest());
    updateOrganization(tenantID, org).then(
      (result: Organization) => {
        dispatch(updateOrganizationSuccess(result));
      },
      (error: APIError) => {
        dispatch(updateOrganizationError(error));
      }
    );
  };

const fetchApps =
  (tenantID: string, orgID: string) => (dispatch: AppDispatch) => {
    dispatch(getLoginAppsForOrgRequest());
    fetchLoginApps(tenantID, orgID).then(
      (data: LoginApp[]) => {
        dispatch(getLoginAppsForOrgSuccess(data));
      },
      (error: APIError) => {
        dispatch(getLoginAppsForOrgError(error));
      }
    );
  };

const OrganizationDetailsPage = ({
  selectedTenantID,
  organization,
  modifiedOrganization,
  organizations,
  isFetching,
  fetchError,
  editMode,
  saveError,
  isSaving,
  loginApps,
  appsFetchError,
  routeParams,
  query,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  organization: Organization | undefined;
  modifiedOrganization: Organization | undefined;
  organizations: PaginatedResult<Organization> | undefined;
  isFetching: boolean;
  fetchError: string;
  editMode: boolean;
  saveError: string;
  isSaving: boolean;
  loginApps: LoginApp[] | undefined;
  appsFetchError: string;
  routeParams: Record<string, string>;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  const { orgID } = routeParams;
  const cleanQuery = makeCleanPageLink(query);

  useEffect(() => {
    if (selectedTenantID && orgID) {
      if (!organization || organization.id !== orgID) {
        const matchingOrg = organizations?.data.find(
          (org: Organization) => org.id === orgID
        );
        if (matchingOrg) {
          dispatch(getOrganizationSuccess(matchingOrg));
        } else {
          dispatch(fetchOrg(selectedTenantID, orgID));
        }
      }
    }
  }, [selectedTenantID, orgID, organization, organizations, dispatch]);
  useEffect(() => {
    if (selectedTenantID && orgID) {
      dispatch(fetchApps(selectedTenantID, orgID));
    }
  }, [selectedTenantID, orgID, dispatch]);

  const isDirty =
    modifiedOrganization &&
    modifiedOrganization.name &&
    modifiedOrganization.name !== organization?.name;
  return (
    <form
      onSubmit={(e: React.FormEvent) => {
        e.preventDefault();
        if (modifiedOrganization && selectedTenantID) {
          dispatch(saveOrg(selectedTenantID, modifiedOrganization));
        }
      }}
    >
      <div className={PageCommon.listviewtablecontrols}>
        <PageTitle title="Organization" itemName={modifiedOrganization?.name} />

        <ButtonGroup className={PageCommon.listviewtablecontrolsButtonGroup}>
          {editMode ? (
            <>
              <Button
                theme="secondary"
                size="small"
                isLoading={isSaving}
                onClick={() => {
                  if (
                    !isDirty ||
                    window.confirm(
                      'You have unsaved changes. Are you sure you want to cancel editing?'
                    )
                  ) {
                    dispatch(toggleOrganizationEditMode());
                  }
                }}
              >
                Cancel
              </Button>
              <Button
                theme="primary"
                type="submit"
                disabled={!isDirty}
                isLoading={isSaving}
                size="small"
              >
                Save Organization
              </Button>
            </>
          ) : (
            <Button
              theme="primary"
              size="small"
              onClick={() => {
                dispatch(toggleOrganizationEditMode());
              }}
            >
              Edit Organization
            </Button>
          )}
        </ButtonGroup>
      </div>

      <Card detailview>
        <CardRow title="Basic Details" collapsible>
          {organization && selectedTenantID ? (
            <>
              {saveError ? (
                <InlineNotification theme="alert">
                  {saveError}
                </InlineNotification>
              ) : (
                ''
              )}
              <Label htmlFor="organization_id">
                ID
                <br />
                <TextShortener
                  text={organization.id}
                  length={6}
                  id="organization_id"
                />
              </Label>
              <Label htmlFor="organization_name">
                Name
                <br />
                {!editMode ? (
                  <InputReadOnly id="organization_name">
                    {organization.name}
                  </InputReadOnly>
                ) : (
                  <TextInput
                    value={modifiedOrganization?.name}
                    name="organization_name"
                    type="text"
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                      const val = e.target.value;
                      dispatch(
                        modifyOrganization({
                          name: val,
                        })
                      );
                    }}
                  />
                )}
              </Label>
              <Label htmlFor="organization_region">
                Region
                <br />
                <InputReadOnly id="organization_region">
                  {organization.region || 'N/A'}
                </InputReadOnly>
              </Label>
            </>
          ) : isFetching ? (
            <Text>Loading ...</Text>
          ) : (
            <InlineNotification theme="alert">
              {fetchError || 'Something went wrong'}
            </InlineNotification>
          )}
        </CardRow>
        <CardRow title="Login Apps" collapsible>
          {loginApps ? (
            loginApps.length ? (
              <Table id="loginApps" className={styles.orgloginappstable}>
                <TableHead>
                  <TableRow>
                    <TableRowHead>Name</TableRowHead>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {loginApps.map((app: LoginApp) => (
                    <TableRow key={app.id}>
                      <TableCell>{app.name}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <Text>
                No login apps.{' '}
                <Link href={`/loginapps${cleanQuery}`} title="Login Apps">
                  Create one here
                </Link>
                .
              </Text>
            )
          ) : isFetching ? (
            <Text>Loading...</Text>
          ) : (
            <InlineNotification theme="alert">
              {appsFetchError || 'Something went wrong'}
            </InlineNotification>
          )}
        </CardRow>
        <CardRow title="Users" collapsible>
          {organization ? (
            <ConnectedUserList selectedOrganizationID={orgID as string} />
          ) : (
            ''
          )}
        </CardRow>
      </Card>
    </form>
  );
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  organization: state.selectedOrganization,
  modifiedOrganization: state.modifiedOrganization,
  organizations: state.organizations,
  isFetching: state.fetchingOrganizations,
  fetchError: state.organizationsFetchError,
  editMode: state.editingOrganization,
  isSaving: state.savingOrganization,
  saveError: state.updateOrganizationError,
  loginApps: state.loginAppsForOrg,
  appsFetchError: state.loginAppsFetchError,
  routeParams: state.routeParams,
  query: state.query,
}))(OrganizationDetailsPage);
