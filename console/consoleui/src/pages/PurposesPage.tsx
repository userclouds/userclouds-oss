import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  Card,
  CardRow,
  Checkbox,
  EmptyState,
  IconFilter,
  IconFocus3Line,
  InlineNotification,
  Table,
  TableHead,
  TableBody,
  TableCell,
  TableRow,
  TableRowHead,
  Text,
  TextShortener,
  ToolTip,
} from '@userclouds/ui-component-lib';

import { RootState, AppDispatch } from '../store';

import Purpose, { PURPOSE_COLUMNS, PURPOSE_PREFIX } from '../models/Purpose';
import { SelectedTenant } from '../models/Tenant';
import PaginatedResult from '../models/PaginatedResult';
import { Filter } from '../models/authz/SearchFilters';
import {
  deletePurposes,
  deleteSinglePurpose,
  fetchPurposes,
} from '../thunks/purposes';
import {
  changePurposeSearchFilter,
  togglePurposeForDelete,
} from '../actions/purposes';

import Link from '../controls/Link';
import Pagination from '../controls/Pagination';
import Search from '../controls/Search';
import DeleteWithConfirmationButton from '../controls/DeleteWithConfirmationButton';

import PageCommon from './PageCommon.module.css';
import styles from './PurposesPage.module.css';

const PurposeList = ({
  selectedCompanyID,
  selectedTenant,
  purposes,
  isFetching,
  fetchError,
  deleteQueue,
  saveSuccess,
  saveErrors,
  purposeSearchFilter,
  dispatch,
}: {
  selectedCompanyID: string | undefined;
  selectedTenant: SelectedTenant | undefined;
  purposes: PaginatedResult<Purpose> | undefined;
  isFetching: boolean;
  fetchError: string;
  deleteQueue: string[];
  saveSuccess: string;
  saveErrors: string[];
  purposeSearchFilter: Filter;
  dispatch: AppDispatch;
}) => {
  const deletePrompt = `Are you sure you want to delete ${
    deleteQueue.length
  } purpose${deleteQueue.length === 1 ? '' : 's'}? This action is irreversible.`;

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div>
          <IconFilter />
        </div>
        <Search
          id="purposes"
          columns={PURPOSE_COLUMNS}
          changeSearchFilter={(filter: Filter) => {
            dispatch(changePurposeSearchFilter(filter));
          }}
          prefix={PURPOSE_PREFIX}
          searchFilter={purposeSearchFilter}
        />
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              {
                'Purposes are used to track, enforce and audit user consent in User Store. '
              }
              <a
                href="https://docs.userclouds.com/docs/purpose-and-consent"
                title="UserClouds documentation for purposes"
                target="new"
                className={PageCommon.link}
              >
                Learn more here.
              </a>
            </>
          </ToolTip>
        </div>

        {selectedTenant?.is_admin && (
          <Button
            theme="primary"
            size="small"
            className={PageCommon.listviewtablecontrolsButton}
          >
            <Link
              href={`/purposes/create?company_id=${encodeURIComponent(
                selectedCompanyID as string
              )}&tenant_id=${encodeURIComponent(selectedTenant!.id)}`}
              title="Add a new purpose"
              applyStyles={false}
            >
              Create Purpose
            </Link>
          </Button>
        )}
      </div>

      <Card
        id="userstorePurposes"
        lockedMessage={
          !selectedTenant?.is_admin ? 'You do not have edit access' : ''
        }
        listview
      >
        {!!saveErrors.length && (
          <div className={PageCommon.tableNotification}>
            <InlineNotification theme="alert" elementName="div">
              <ul>
                {saveErrors.map((error: string) => (
                  <li>{error}</li>
                ))}
              </ul>
            </InlineNotification>
          </div>
        )}

        {saveSuccess && (
          <div className={PageCommon.tableNotification}>
            <InlineNotification theme="success">
              {saveSuccess}
            </InlineNotification>
          </div>
        )}

        {(isFetching || (!isFetching && !purposes?.data && !fetchError)) && (
          <Text>Fetching purposes…</Text>
        )}

        {!isFetching && !purposes?.data && (
          <CardRow className={PageCommon.tableNotification}>
            <InlineNotification theme="alert">
              {fetchError || 'Something went wrong'}
            </InlineNotification>
          </CardRow>
        )}

        {!isFetching && purposes?.data && !purposes?.data.length && (
          <CardRow className={PageCommon.emptyState}>
            <EmptyState
              title="Nothing to display"
              subTitle="No purposes have been specified yet for this tenant."
              image={<IconFocus3Line size="large" />}
            >
              {selectedTenant?.is_admin && (
                <Button theme="secondary">
                  <Link
                    href={`/purposes/create?company_id=${encodeURIComponent(
                      selectedCompanyID as string
                    )}&tenant_id=${encodeURIComponent(selectedTenant!.id)}`}
                    title="Add a new purpose"
                    applyStyles={false}
                  >
                    Create Purpose
                  </Link>
                </Button>
              )}
            </EmptyState>
          </CardRow>
        )}

        {purposes?.data && purposes.data.length && (
          <>
            <div className={PageCommon.listviewpaginationcontrols}>
              <DeleteWithConfirmationButton
                id="deletePurposesButton"
                message={deletePrompt}
                onConfirmDelete={() => {
                  if (selectedTenant) {
                    dispatch(deletePurposes(selectedTenant.id, deleteQueue));
                  }
                }}
                title="Delete Purposes"
                disabled={deleteQueue.length < 1}
              />
              <Pagination
                prev={purposes?.prev}
                next={purposes?.next}
                isLoading={isFetching}
                prefix={PURPOSE_PREFIX}
              />
            </div>
            <Table
              spacing="packed"
              id="purposes"
              className={styles.purposestable}
            >
              <TableHead floating>
                <TableRow>
                  <TableRowHead>
                    <Checkbox
                      checked={deleteQueue.length > 0}
                      onChange={() => {
                        const shouldMarkForDelete = !deleteQueue.includes(
                          purposes.data[0].id
                        );
                        purposes.data.forEach((o) => {
                          if (
                            shouldMarkForDelete &&
                            !deleteQueue.includes(o.id)
                          ) {
                            dispatch(togglePurposeForDelete(o.id));
                          } else if (
                            !shouldMarkForDelete &&
                            deleteQueue.includes(o.id)
                          ) {
                            dispatch(togglePurposeForDelete(o.id));
                          }
                        });
                      }}
                    />
                  </TableRowHead>
                  <TableRowHead key="purpose_name">Name</TableRowHead>
                  <TableRowHead key="purpose_description">
                    Description
                  </TableRowHead>
                  <TableRowHead key="purpose_id">ID</TableRowHead>
                  <TableRowHead key="purpose_delete" />
                </TableRow>
              </TableHead>
              <TableBody>
                {purposes.data.map((purpose: Purpose) => (
                  <TableRow
                    key={purpose.id}
                    className={
                      (deleteQueue.includes(purpose.id)
                        ? PageCommon.queuedfordelete
                        : '') +
                      ' ' +
                      PageCommon.listviewtablerow
                    }
                  >
                    <TableCell>
                      <Checkbox
                        id={'delete' + purpose.id}
                        name="delete object"
                        checked={deleteQueue.includes(purpose.id)}
                        onChange={() => {
                          dispatch(togglePurposeForDelete(purpose.id));
                        }}
                      />
                    </TableCell>
                    <TableCell>
                      <Link
                        href={`/purposes/${encodeURIComponent(
                          purpose.id
                        )}?company_id=${encodeURIComponent(
                          selectedCompanyID as string
                        )}&tenant_id=${encodeURIComponent(selectedTenant!.id)}`}
                        title="See details for this purpose"
                      >
                        {purpose.name}
                      </Link>
                    </TableCell>
                    <TableCell>{purpose.description}</TableCell>
                    <TableCell>
                      <TextShortener text={purpose.id} length={6} />
                    </TableCell>

                    <TableCell className={PageCommon.listviewtabledeletecell}>
                      <DeleteWithConfirmationButton
                        id="deletePurposeButton"
                        message="Are you sure you want to delete this purpose? This action is irreversible."
                        onConfirmDelete={() => {
                          if (selectedCompanyID && selectedTenant) {
                            dispatch(
                              deleteSinglePurpose(
                                selectedCompanyID,
                                selectedTenant?.id,
                                purpose.id
                              )
                            );
                          }
                        }}
                        title="Delete Purpose"
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </>
        )}
      </Card>
    </>
  );
};
const ConnectedPurposeList = connect((state: RootState) => ({
  selectedCompanyID: state.selectedCompanyID,
  selectedTenant: state.selectedTenant,
  purposes: state.purposes,
  isFetching: state.fetchingPurposes,
  fetchError: state.purposesFetchError,
  deleteQueue: state.purposesDeleteQueue,
  saveSuccess: state.deletePurposesSuccess,
  saveErrors: state.deletePurposesErrors,
  purposeSearchFilter: state.purposeSearchFilter,
}))(PurposeList);

const PurposesPage = ({
  selectedTenantID,
  query,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenantID) {
      dispatch(fetchPurposes(selectedTenantID, query));
    }
  }, [selectedTenantID, query, dispatch]);

  return <ConnectedPurposeList />;
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  query: state.query,
}))(PurposesPage);
