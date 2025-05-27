import clsx from 'clsx';
import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Card,
  CardRow,
  Checkbox,
  EmptyState,
  IconUser3,
  InlineNotification,
  Label,
  Select,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  Text,
  ToolTip,
  TextShortener,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';

import { makeCleanPageLink } from '../AppNavigation';
import API from '../API';
import { fetchOrganizations } from '../API/organizations';
import actions from '../actions';
import { redirect } from '../routing';
import { AppDispatch, RootState } from '../store';
import Pagination from '../controls/Pagination';
import PaginatedResult from '../models/PaginatedResult';
import { UserBaseProfile } from '../models/UserProfile';
import { SelectedTenant } from '../models/Tenant';
import Organization from '../models/Organization';
import Link from '../controls/Link';
import Styles from './UsersPage.module.css';
import PageCommon from './PageCommon.module.css';
import { fetchUserStoreConfig } from '../thunks/userstore';
import {
  Column,
  userStoreHasEmailColumn,
  userStoreHasNameColumn,
} from '../models/TenantUserStoreConfig';
import DeleteWithConfirmationButton from '../controls/DeleteWithConfirmationButton';

const PAGE_SIZE = '50';

const fetchTenantOrgs = (tenantID: string) => async (dispatch: AppDispatch) => {
  dispatch({
    type: actions.GET_TENANT_ORGS_REQUEST,
  });
  fetchOrganizations(tenantID, { limit: '1500' }).then(
    (response: PaginatedResult<Organization>) => {
      dispatch({
        type: actions.GET_TENANT_ORGS_SUCCESS,
        data: response,
      });
    },
    (error) => {
      dispatch({
        type: actions.GET_TENANT_ORGS_ERROR,
        data: error.message,
      });
    }
  );
};

const fetchUsers =
  (tenantID: string, organizationID: string, params: URLSearchParams) =>
  async (dispatch: AppDispatch) => {
    const paramsAsObject = Object.fromEntries(params.entries());
    if (!paramsAsObject.limit) {
      paramsAsObject.limit = PAGE_SIZE;
    }
    dispatch({
      type: actions.GET_TENANT_USERS_REQUEST,
    });
    API.fetchTenantUserPage(tenantID, organizationID, paramsAsObject).then(
      (data: PaginatedResult<UserBaseProfile>) => {
        dispatch({
          type: actions.GET_TENANT_USERS_SUCCESS,
          data,
        });
      },
      (error: APIError) => {
        dispatch({
          type: actions.GET_TENANT_USERS_ERROR,
          data: error.message,
        });
      }
    );
  };

const saveChanges =
  () => async (dispatch: AppDispatch, getState: () => RootState) => {
    const { selectedTenantID, tenantUsers, userDeleteQueue } = getState();
    if (!tenantUsers || !selectedTenantID) {
      return;
    }

    if (userDeleteQueue.length) {
      const promises = userDeleteQueue.map((id) =>
        dispatch(deleteUser(selectedTenantID as string, id))
      );
      dispatch({
        type: actions.BULK_UPDATE_USERS_START,
      });
      Promise.all(promises as Array<Promise<void>>).then(
        () => {
          dispatch({
            type: actions.BULK_UPDATE_USERS_END,
            data: true, // success
          });
        },
        () => {
          dispatch({
            type: actions.BULK_UPDATE_USERS_END,
            data: false, // complete or partial failure
          });
        }
      );
    }
  };

const deleteUser =
  (tenantID: string, userID: string) => async (dispatch: AppDispatch) => {
    dispatch({
      type: actions.DELETE_USER_REQUEST,
    });
    return API.deleteUser(tenantID, userID).then(
      () => {
        dispatch({
          type: actions.DELETE_USER_SUCCESS,
          data: userID,
        });
      },
      (error: APIError) => {
        dispatch({
          type: actions.DELETE_USER_ERROR,
          data: error.message,
        });
        throw error;
      }
    );
  };

const UserListRow = ({
  tenantID,
  user,
  deleteQueue,
  hasEmail,
  hasName,
  query,
  dispatch,
}: {
  tenantID: string | undefined;
  user: UserBaseProfile;
  deleteQueue: string[];
  hasEmail: boolean;
  hasName: boolean;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  const cleanQuery = makeCleanPageLink(query);
  const verified =
    user.email_verified === 'true' ? ' (verified)' : ' (not verified)';
  return (
    <TableRow
      key={user.id}
      title={
        deleteQueue.includes(user.id)
          ? 'Queued for delete'
          : 'View user details'
      }
      className={clsx(
        deleteQueue.includes(user.id) ? PageCommon.queuedfordelete : '',
        PageCommon.listviewtablerow
      )}
    >
      <TableCell>
        <Checkbox
          id={'delete' + user.id}
          name="delete user"
          checked={deleteQueue.includes(user.id)}
          onChange={() => {
            dispatch({
              type: actions.TOGGLE_USER_FOR_DELETE,
              data: user.id,
            });
          }}
        />
      </TableCell>
      <TableCell>
        <Link
          href={`/users/${encodeURIComponent(user.id)}${cleanQuery}`}
          title="View user details"
        >
          <img
            className={Styles.profileimg}
            src={user.picture?.toString() || '/mystery-person.webp'}
            alt={user.picture ? 'user avatar' : 'default avatar'}
          />
        </Link>
      </TableCell>
      {hasName && (
        <TableCell>
          <Link
            href={`/users/${encodeURIComponent(user.id)}${cleanQuery}`}
            title="View user details"
          >
            {user.name?.toString()}
          </Link>
        </TableCell>
      )}
      {hasEmail && (
        <TableCell>
          <Link
            href={`/users/${encodeURIComponent(user.id)}${cleanQuery}`}
            title="View user details"
          >
            {user.email + verified}
          </Link>
        </TableCell>
      )}
      <TableCell>
        <Link
          href={`/users/${encodeURIComponent(user.id)}${cleanQuery}`}
          title="View user details"
        >
          <TextShortener text={user.id} length={6} isCopyable={false} />
        </Link>
      </TableCell>
      <TableCell className={PageCommon.listviewtabledeletecell}>
        <DeleteWithConfirmationButton
          id="deleteUserButton"
          message="Are you sure you want to delete this user? This action is irreversible."
          onConfirmDelete={() => {
            tenantID && dispatch(deleteUser(tenantID, user.id));
          }}
          title="Delete User"
        />
      </TableCell>
    </TableRow>
  );
};
const ConnectedUserRow = connect((state: RootState) => {
  return {
    tenantID: state.selectedTenantID,
    deleteQueue: state.userDeleteQueue,
    query: state.query,
  };
})(UserListRow);

const UserList = ({
  selectedTenant,
  selectedOrganizationID,
  users,
  fetchError,
  deleteQueue,
  bulkSaveErrors,
  query,
  userStoreColumns,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  selectedOrganizationID: string | undefined;
  users: PaginatedResult<UserBaseProfile> | undefined;
  fetchError: string;
  deleteQueue: string[];
  bulkSaveErrors: string[];
  query: URLSearchParams;
  userStoreColumns: Column[] | undefined;
  dispatch: AppDispatch;
}) => {
  const deletePrompt = `Are you sure you want to delete ${
    deleteQueue.length
  } user${deleteQueue.length === 1 ? '' : 's'}? This action is irreversible.`;

  useEffect(() => {
    if (selectedTenant && selectedOrganizationID !== undefined) {
      dispatch(fetchUsers(selectedTenant.id, selectedOrganizationID, query));
    }
  }, [selectedTenant, selectedOrganizationID, query, dispatch]);

  useEffect(() => {
    if (selectedTenant) {
      dispatch(fetchUserStoreConfig(selectedTenant.id));
    }
  }, [selectedTenant, dispatch]);

  const hasNameColumn = userStoreColumns
    ? userStoreHasNameColumn(userStoreColumns)
    : false;
  const hasEmailColumn = userStoreColumns
    ? userStoreHasEmailColumn(userStoreColumns)
    : false;

  return users ? (
    users?.data.length ? (
      <>
        <div className={PageCommon.listviewpaginationcontrols}>
          <div className={PageCommon.listviewpaginationcontrolsdelete}>
            <DeleteWithConfirmationButton
              id="deleteUsersButton"
              message={deletePrompt}
              onConfirmDelete={() => {
                dispatch(saveChanges());
              }}
              title="Delete Users"
            />
          </div>
          <Pagination prev={users.prev} next={users.next} isLoading={false} />
        </div>
        {bulkSaveErrors.length > 0 && (
          <InlineNotification theme="alert">
            {bulkSaveErrors.length === 1
              ? bulkSaveErrors[0]
              : `Problems editing or deleting ${bulkSaveErrors.length} users`}
          </InlineNotification>
        )}
        <Table spacing="packed" id="usersTable" className={Styles.usertable}>
          <TableHead floating>
            <TableRow>
              <TableRowHead>
                <Checkbox
                  checked={deleteQueue.length > 0}
                  onChange={() => {
                    const shouldMarkForDelete = !deleteQueue.includes(
                      users.data[0].id
                    );
                    users.data.forEach((o) => {
                      if (shouldMarkForDelete && !deleteQueue.includes(o.id)) {
                        dispatch({
                          type: actions.TOGGLE_USER_FOR_DELETE,
                          data: o.id,
                        });
                      } else if (
                        !shouldMarkForDelete &&
                        deleteQueue.includes(o.id)
                      ) {
                        dispatch({
                          type: actions.TOGGLE_USER_FOR_DELETE,
                          data: o.id,
                        });
                      }
                    });
                  }}
                />
              </TableRowHead>

              <TableRowHead key="image" />
              {hasNameColumn && <TableRowHead key="name">Name</TableRowHead>}
              {hasEmailColumn && <TableRowHead key="email">Email</TableRowHead>}
              <TableRowHead key="id">ID</TableRowHead>
              <TableRowHead key="delete" />
            </TableRow>
          </TableHead>
          <TableBody>
            {users.data.map((user) => (
              <ConnectedUserRow
                user={user}
                hasEmail={hasEmailColumn}
                hasName={hasNameColumn}
                key={user.id}
              />
            ))}
          </TableBody>
        </Table>
      </>
    ) : (
      <CardRow className={PageCommon.emptyState}>
        <EmptyState
          title="Nothing to display"
          subTitle="This organization does not have any users yet."
          image={<IconUser3 size="large" />}
        />
      </CardRow>
    )
  ) : fetchError ? (
    <CardRow className={PageCommon.tableNotification}>
      <InlineNotification theme="alert">{fetchError}</InlineNotification>
    </CardRow>
  ) : (
    <Text>Loading users ...</Text>
  );
};

const ConnectedUserList = connect((state: RootState) => {
  return {
    selectedTenant: state.selectedTenant,
    selectedOrganizationID: state.tenantUsersSelectedOrganizationID,
    users: state.tenantUsers,
    fetchError: state.fetchUsersError,
    deleteQueue: state.userDeleteQueue,
    bulkSaveErrors: state.userBulkSaveErrors,
    query: state.query,
    userStoreColumns: state.userStoreColumns,
  };
})(UserList);

const TenantUserList = ({
  selectedTenant,
  tenantOrganizations,
  selectedOrganizationID,
  fetchingOrgs,
  query,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  tenantOrganizations: PaginatedResult<Organization> | undefined;
  selectedOrganizationID: string | undefined;
  fetchingOrgs: boolean;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    const selectedOrg = query.get('selected_tenant_organization');
    if (selectedTenant?.use_organizations) {
      if (tenantOrganizations === undefined && !fetchingOrgs) {
        dispatch(fetchTenantOrgs(selectedTenant.id));
      }
    }
    if (!selectedOrganizationID || selectedOrganizationID !== selectedOrg) {
      dispatch({
        type: actions.SELECT_USERS_TENANT_ORGANIZATION,
        data: selectedOrg || '',
      });
    }
  }, [
    selectedTenant,
    tenantOrganizations,
    selectedOrganizationID,
    fetchingOrgs,
    query,
    dispatch,
  ]);

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        {selectedTenant?.use_organizations &&
        tenantOrganizations &&
        tenantOrganizations.data.length ? (
          <Label>
            Select organization:{' '}
            <Select
              name="organization_id"
              value={selectedOrganizationID}
              onChange={(e: React.ChangeEvent) => {
                const val = (e.target as HTMLSelectElement).value;
                redirect(
                  `/users${makeCleanPageLink(
                    query
                  )}&selected_tenant_organization=${val}`
                );
              }}
            >
              <option key="all_organizations" value="">
                All organizations
              </option>
              {tenantOrganizations.data.map((organization: Organization) => (
                <option
                  key={`organization_${organization.id}`}
                  value={organization.id}
                >
                  {organization.name}
                </option>
              ))}
            </Select>
          </Label>
        ) : (
          ''
        )}
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>View and manage the end users of your applications.</>
          </ToolTip>
        </div>
      </div>
      <Card listview>
        <ConnectedUserList />
      </Card>
    </>
  );
};

const ConnectedTenantUserList = connect((state: RootState) => {
  return {
    selectedTenant: state.selectedTenant,
    tenantOrganizations: state.tenantUsersOrganizations,
    selectedOrganizationID: state.tenantUsersSelectedOrganizationID,
    fetchingOrgs: state.fetchingOrganizations,
    query: state.query,
  };
})(TenantUserList);

const UsersPage = () => {
  return <ConnectedTenantUserList />;
};

export { UserList };
export default connect((state: RootState) => ({
  location: state.location,
  query: state.query,
}))(UsersPage);
