import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Card,
  InlineNotification,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  Text,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';

import API from '../API';
import actions from '../actions';
import { AppDispatch, RootState } from '../store';
import PaginatedResult from '../models/PaginatedResult';
import { UserInvite } from '../models/UserInvite';
import Company from '../models/Company';
import { SelectedTenant } from '../models/Tenant';
import { getOrganizationRoleDisplayName } from '../models/UserRoles';
import Pagination from '../controls/Pagination';
import InviteWidget from '../controls/InviteWidget';
import { CompanyMemberList, ConnectedTenantMemberList } from './IAMMemberLists';
import IAMPageStyles from './IAMPage.module.css';

const INVITES_PAGE_SIZE = '5';
const fetchInvites =
  (companyID: string, params: URLSearchParams) => (dispatch: AppDispatch) => {
    const paramsAsObject = Object.fromEntries(params.entries());
    if (!paramsAsObject.limit) {
      paramsAsObject.limit = INVITES_PAGE_SIZE;
    }

    dispatch({
      type: actions.GET_COMPANY_INVITES_REQUEST,
    });
    API.fetchCompanyInvites(companyID, paramsAsObject).then(
      (data) => {
        dispatch({
          type: actions.GET_COMPANY_INVITES_SUCCESS,
          data: data,
        });
      },
      (error: APIError) => {
        dispatch({
          type: actions.GET_COMPANY_INVITES_ERROR,
          data: error.message,
        });
      }
    );
  };

const InviteList = ({
  companyID,
  isFetching,
  invites,
  fetchError,
  query,
  dispatch,
}: {
  companyID: string | undefined;
  isFetching: boolean;
  invites: PaginatedResult<UserInvite> | undefined;
  fetchError: string;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (companyID) {
      dispatch(fetchInvites(companyID, query));
    }
  }, [companyID, query, dispatch]);

  if (!companyID) {
    return <></>;
  }
  return (
    <>
      {fetchError && (
        <InlineNotification theme="alert">{fetchError}</InlineNotification>
      )}
      {invites && invites.data.length ? (
        <>
          <Table className={IAMPageStyles.inviteTable}>
            <TableHead>
              <TableRow>
                <TableRowHead key="invite_email">Email</TableRowHead>
                <TableRowHead key="invite_role">Company role</TableRowHead>
                <TableRowHead key="invite_sent">Sent on</TableRowHead>
                <TableRowHead key="invite_expires">Expires on</TableRowHead>
              </TableRow>
            </TableHead>
            <TableBody>
              {invites.data.map((invite: UserInvite) => (
                <TableRow key={invite.id}>
                  <TableCell>{invite.invitee_email}</TableCell>
                  <TableCell>
                    {getOrganizationRoleDisplayName(invite.role)}
                  </TableCell>
                  <TableCell>
                    {new Date(invite.created).toLocaleString('en-US')}
                  </TableCell>
                  <TableCell>
                    {new Date(invite.expires).toLocaleString('en-US')}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          {(invites.has_prev || invites.has_next) && (
            <Pagination
              prev={invites.prev}
              next={invites.next}
              isLoading={false}
            />
          )}
        </>
      ) : (
        <Text>
          {isFetching
            ? 'Loading invites ...'
            : 'No invites have been sent yet.'}
        </Text>
      )}
    </>
  );
};
const ConnectedInviteList = connect((state: RootState) => ({
  companyID: state.selectedCompanyID,
  isFetching: state.fetchingInvites,
  invites: state.companyInvites,
  fetchError: state.fetchInvitesError,
  query: state.query,
}))(InviteList);

const ConnectedCompanyMemberList = connect((state: RootState) => {
  return {
    editMode: state.companyUserRolesEditMode,
    isSaving: state.savingCompanyUserRoles,
    teamUserRoles: state.companyUserRoles,
    isFetching: state.fetchingCompanyUserRoles,
    fetchError: state.fetchCompanyUserRolesError,
    deleteQueue: state.companyUserRolesDeleteQueue,
    modifiedUserRoles: state.modifiedCompanyUserRoles,
    bulkSaveErrors: state.companyUserRolesBulkSaveErrors,
    userID: state.myProfile?.userProfile.id,
  };
})(CompanyMemberList);

const IAMPage = ({
  company,
  selectedTenant,
  editMode,
  query,
  fetchingInvites,
  dispatch,
  invites,
}: {
  company: Company | undefined;
  selectedTenant: SelectedTenant | undefined;
  editMode: boolean;
  query: URLSearchParams;
  fetchingInvites: boolean;
  dispatch: AppDispatch;
  invites: PaginatedResult<UserInvite> | undefined;
}) => {
  const onSend = () => {
    dispatch(fetchInvites(company?.id || '', query));
  };

  return (
    <>
      {company?.is_admin && (
        <>
          <Card
            title="Invite Team"
            description="Invite teammates and assign them a role in your company. See outstanding invites."
            isClosed
          >
            <InviteWidget onSend={onSend} isLoading={fetchingInvites} />
          </Card>
          {invites && invites.data.length ? (
            <Card title="Pending Invites">
              <ConnectedInviteList />
            </Card>
          ) : (
            <></>
          )}
          <Card
            title="Manage Company Roles"
            description={`Manage roles for teammates at ${
              company?.name || '...'
            }`}
            isDirty={editMode}
          >
            <ConnectedCompanyMemberList companyID={company.id} />
          </Card>
        </>
      )}
      {!selectedTenant?.is_console_tenant && (
        <ConnectedTenantMemberList
          readOnly={!(selectedTenant?.is_admin || company?.is_admin)}
        />
      )}
    </>
  );
};

export default connect((state: RootState) => ({
  company: state.selectedCompany,
  selectedTenant: state.selectedTenant,
  editMode: state.companyUserRolesEditMode,
  query: state.query,
  fetchingInvites: state.fetchingInvites,
  invites: state.companyInvites,
}))(IAMPage);
