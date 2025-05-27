import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Card,
  CardRow,
  Button,
  EmptyState,
  IconOrganization,
  InlineNotification,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  Text,
  TextShortener,
  ToolTip,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';

import { makeCleanPageLink } from '../AppNavigation';
import { RootState, AppDispatch } from '../store';
import PaginatedResult from '../models/PaginatedResult';
import Organization from '../models/Organization';
import { SelectedTenant } from '../models/Tenant';
import {
  getOrganizationsRequest,
  getOrganizationsSuccess,
  getOrganizationsError,
} from '../actions/organizations';
import { fetchOrganizations } from '../API/organizations';
import Pagination from '../controls/Pagination';
import Link from '../controls/Link';
import PageCommon from './PageCommon.module.css';
import styles from './OrgsPage.module.css';

const PAGE_SIZE = '20';
const fetchOrgs =
  (tenantID: string, params: URLSearchParams) => (dispatch: AppDispatch) => {
    const paramsAsObject = Object.fromEntries(params.entries());
    if (!paramsAsObject.limit) {
      paramsAsObject.limit = PAGE_SIZE;
    }

    dispatch(getOrganizationsRequest());
    fetchOrganizations(tenantID, paramsAsObject).then(
      (result: PaginatedResult<Organization>) => {
        dispatch(getOrganizationsSuccess(result));
      },
      (error: APIError) => {
        dispatch(getOrganizationsError(error));
      }
    );
  };

const OrgsPage = ({
  selectedCompanyID,
  selectedTenant,
  organizations,
  isFetching,
  fetchError,
  query,
  dispatch,
}: {
  selectedCompanyID: string | undefined;
  selectedTenant: SelectedTenant | undefined;
  organizations: PaginatedResult<Organization> | undefined;
  isFetching: boolean;
  fetchError: string;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  const cleanQuery = makeCleanPageLink(query);

  useEffect(() => {
    if (selectedTenant) {
      dispatch(fetchOrgs(selectedTenant.id, query));
    }
  }, [selectedTenant, query, dispatch]);
  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>Manage organizations for your tenant</>
          </ToolTip>
        </div>

        {selectedTenant?.is_admin && (
          <Button
            theme="primary"
            size="small"
            className={PageCommon.listviewtablecontrolsButton}
          >
            <Link
              href={`/organizations/create${cleanQuery}`}
              title="Create an organization"
              applyStyles={false}
            >
              Create Organization
            </Link>
          </Button>
        )}
      </div>
      <Card listview>
        {selectedCompanyID && selectedTenant && organizations ? (
          organizations?.data?.length ? (
            <>
              <div className={PageCommon.listviewpaginationcontrols}>
                <Pagination
                  prev={organizations.prev}
                  next={organizations.next}
                  isLoading={isFetching}
                />
              </div>

              <Table
                spacing="packed"
                isResponsive={false}
                id="organizations"
                className={styles.orgstable}
              >
                <TableHead floating>
                  <TableRow>
                    <TableRowHead key="org_name_th">
                      Organization name
                    </TableRowHead>
                    <TableRowHead key="org_region">Region</TableRowHead>
                    <TableRowHead key="org_created_date_th">
                      Created
                    </TableRowHead>
                    <TableRowHead key="org_id_th">ID</TableRowHead>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {organizations.data.map((org: Organization) => (
                    <TableRow key={org.id}>
                      <TableCell key="org_name">
                        <Link
                          href={`/organizations/${org.id}${cleanQuery}`}
                          title="View/edit organization"
                        >
                          {org.name}
                        </Link>
                      </TableCell>
                      <TableCell key="org_region">{org.region}</TableCell>
                      <TableCell key="org_created_date">
                        {new Date(org.created).toLocaleString('en-us')}
                      </TableCell>
                      <TableCell key="org_id">
                        <TextShortener text={org.id} length={6} />
                      </TableCell>
                      <TableCell key="edit_org" />
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </>
          ) : (
            <CardRow className={PageCommon.emptyState}>
              <EmptyState
                title="Nothing to display"
                subTitle="This tenant does not have any organizations yet."
                image={<IconOrganization size="large" />}
              >
                {selectedTenant.is_admin && (
                  <Button theme="secondary">
                    <Link
                      href={`/organizations/create${cleanQuery}`}
                      title="Create an organization"
                      applyStyles={false}
                    >
                      Create an organization
                    </Link>
                  </Button>
                )}
              </EmptyState>
            </CardRow>
          )
        ) : fetchError ? (
          <InlineNotification theme="alert">{fetchError}</InlineNotification>
        ) : (
          <Text>Loading ...</Text>
        )}
      </Card>
    </>
  );
};

export default connect((state: RootState) => ({
  selectedCompanyID: state.selectedCompanyID,
  selectedTenant: state.selectedTenant,
  organizations: state.organizations,
  isFetching: state.fetchingOrganizations,
  fetchError: state.organizationsFetchError,
  query: state.query,
}))(OrgsPage);
