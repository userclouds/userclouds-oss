import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Card,
  CardRow,
  EmptyState,
  InlineNotification,
  IconDatabase2,
  IconFilter,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  Text,
  ToolTip,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';

import { makeCleanPageLink } from '../AppNavigation';
import { RootState, AppDispatch } from '../store';
import { SystemLogEntry } from '../models/SystemLogEntry';
import { Filter } from '../models/authz/SearchFilters';
import PaginatedResult from '../models/PaginatedResult';
import {
  changeCurrentSystemLogSearchFilter,
  getSystemLogEntriesRequest,
  getSystemLogEntriesRequestError,
  getSystemLogEntriesSuccess,
} from '../actions/systemLog';
import { fetchSystemLogs } from '../API/systemlog';
import Link from '../controls/Link';
import Search from '../controls/Search';
import Pagination from '../controls/Pagination';
import { getParamsAsObject } from '../controls/PaginationHelper';
import PageCommon from './PageCommon.module.css';
import styles from './SystemLogPage.module.css';

const PAGINATION_LIMIT = '50';
const prefix = 'system_log_';
const columns = ['id', 'type'];

const fetchLogs =
  (selectedTenantID: string | undefined, params: URLSearchParams) =>
  (dispatch: AppDispatch) => {
    const paramsAsObject = getParamsAsObject(prefix, params);
    // if objects_limit is not specified in querystring,
    // use the default
    if (!paramsAsObject.limit) {
      paramsAsObject.limit = PAGINATION_LIMIT;
    }
    if (!selectedTenantID) {
      return;
    }
    dispatch(getSystemLogEntriesRequest());
    fetchSystemLogs(selectedTenantID, paramsAsObject).then(
      (data: PaginatedResult<SystemLogEntry>) => {
        dispatch(getSystemLogEntriesSuccess(data));
      },
      (error: APIError) => {
        dispatch(getSystemLogEntriesRequestError(error));
      }
    );
  };

const changeSystemLogSearchFilter =
  (filter: Filter) => async (dispatch: AppDispatch) => {
    // TODO v2 Add Operator Column and set the operator with that
    dispatch(changeCurrentSystemLogSearchFilter(filter));
  };

const SystemLogList = ({
  selectedTenantID,
  entries,
  isFetching,
  fetchError,
  systemLogSearchFilter,
  query,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  entries: PaginatedResult<SystemLogEntry> | undefined;
  isFetching: boolean;
  fetchError: string;
  systemLogSearchFilter: Filter;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenantID) {
      dispatch(fetchLogs(selectedTenantID, query));
    }
  }, [selectedTenantID, query, dispatch]);

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div>
          <IconFilter />
        </div>
        <Search
          id="systemLog"
          columns={columns}
          changeSearchFilter={(filter: Filter) => {
            dispatch(changeSystemLogSearchFilter(filter));
          }}
          searchFilter={systemLogSearchFilter}
          prefix={prefix}
        />
        <ToolTip>
          <>This log shows the actions the system has taken on your behalf.</>
        </ToolTip>
      </div>

      <Card listview>
        <>
          <div className={PageCommon.listviewpaginationcontrols}>
            <Pagination
              prev={entries?.prev}
              next={entries?.next}
              prefix={prefix}
              isLoading={false}
            />
          </div>

          {entries ? (
            entries.data && entries.data.length ? (
              <Table
                spacing="packed"
                id="syslog"
                className={styles.systemlogtable}
              >
                <TableHead floating>
                  <TableRow>
                    <TableRowHead key="Type">Type</TableRowHead>
                    <TableRowHead key="Event ID">Event ID</TableRowHead>
                    <TableRowHead key="Created">Created</TableRowHead>
                    <TableRowHead key="Total Records">
                      Total Records
                    </TableRowHead>
                    <TableRowHead key="Failed">Failed</TableRowHead>
                    <TableRowHead key="Warnings">Warnings</TableRowHead>
                  </TableRow>
                </TableHead>

                <TableBody>
                  {entries.data.map((entry) => (
                    <TableRow key={entry.id}>
                      <TableCell>{entry.Type}</TableCell>
                      <TableCell>
                        <Link
                          key={entry.id}
                          href={
                            `/systemlog/${entry.id}` + makeCleanPageLink(query)
                          }
                        >
                          {entry.id}
                        </Link>
                      </TableCell>
                      <TableCell>
                        {new Date(entry.created).toLocaleString('en-US')}
                      </TableCell>
                      <TableCell>{entry.TotalRecords}</TableCell>
                      <TableCell>{entry.FailedRecords}</TableCell>
                      <TableCell>{entry.WarningRecords}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <CardRow>
                <EmptyState
                  title="No entries in the system log."
                  image={<IconDatabase2 size="large" />}
                />
              </CardRow>
            )
          ) : fetchError !== '' ? (
            <InlineNotification theme="alert">{fetchError}</InlineNotification>
          ) : isFetching ? (
            <Text>Loading entries...</Text>
          ) : (
            <InlineNotification theme="alert">
              No entries found
            </InlineNotification>
          )}
          {(entries?.has_prev || entries?.has_next) && (
            <Pagination
              prev={entries?.prev}
              next={entries?.next}
              prefix={prefix}
              isLoading={false}
            />
          )}
        </>
      </Card>
    </>
  );
};
const ConnectedSystemLogList = connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  entries: state.systemLogEntries,
  isFetching: state.fetchingSystemLogEntries,
  fetchError: state.fetchSystemLogEntriesError,
  systemLogSearchFilter: state.systemLogSearchFilter,
  query: state.query,
}))(SystemLogList);

export default ConnectedSystemLogList;
