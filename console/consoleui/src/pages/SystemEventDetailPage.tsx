import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Card,
  CardRow,
  EmptyState,
  Heading,
  InlineNotification,
  IconDatabase2,
  InputReadOnly,
  Label,
  LoaderDots,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  Text,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';

import { fetchSingleLogEntry, fetchSystemLogDetails } from '../API/systemlog';
import { RootState, AppDispatch } from '../store';
import {
  changeCurrentSystemLogEntryDetailSearchFilter,
  getSystemLogEntryDetailRequest,
  getSystemLogEntryDetailRequestError,
  getSystemLogEntryDetailSuccess,
  getSingleSystemLogEntryDetailRequest,
  getSingleSystemLogEntryDetailSuccess,
  getSingleSystemLogEntryDetailRequestError,
} from '../actions/systemLog';
import { SystemLogEntry, SystemLogEntryRecord } from '../models/SystemLogEntry';
import { Filter } from '../models/authz/SearchFilters';
import PaginatedResult from '../models/PaginatedResult';
import Search from '../controls/Search';
import Pagination from '../controls/Pagination';
import { getParamsAsObject } from '../controls/PaginationHelper';
import PageCommon from './PageCommon.module.css';
import styles from './SystemEventDetailPage.module.css';

const prefix = 'system_event_';
const columns = ['id', 'sync_run_id'];

const fetchEntry =
  (selectedTenantID: string | undefined, entryID: string) =>
  (dispatch: AppDispatch) => {
    if (!entryID) {
      return;
    }
    if (!selectedTenantID) {
      return;
    }
    dispatch(getSingleSystemLogEntryDetailRequest());
    fetchSingleLogEntry(selectedTenantID, entryID).then(
      (data: PaginatedResult<SystemLogEntry>) => {
        dispatch(getSingleSystemLogEntryDetailSuccess(data));
      },
      (error: APIError) => {
        dispatch(getSingleSystemLogEntryDetailRequestError(error.message));
      }
    );
  };

const SystemLogEntryDetail = ({
  selectedTenantID,
  entry,
  runID,
  isFetching,
  errorFetching,
  query,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  entry: SystemLogEntry | undefined;
  runID: string;
  isFetching: boolean;
  errorFetching: string;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenantID) {
      dispatch(fetchEntry(selectedTenantID, runID));
    }
  }, [selectedTenantID, query, runID, dispatch]);

  return (
    <Card title="Summary">
      <Heading size="3" headingLevel="3">
        Basic Details
      </Heading>
      {entry ? (
        <>
          <div className={PageCommon.propertiesRow}>
            <div className={PageCommon.propertiesElement}>
              <Label htmlFor="name">
                Event Type
                <InputReadOnly>{entry?.Type}</InputReadOnly>
              </Label>
            </div>
            <div className={PageCommon.propertiesElement}>
              <Label>
                Event ID
                <InputReadOnly>{runID}</InputReadOnly>
              </Label>
            </div>
            <div className={PageCommon.propertiesElement}>
              <Label>
                Created Date
                <InputReadOnly>{entry?.created}</InputReadOnly>
              </Label>
            </div>
          </div>
          <div className={PageCommon.propertiesRow}>
            <div className={PageCommon.propertiesElement}>
              <Label htmlFor="name">
                Event Summary
                <InputReadOnly>{`There were ${entry?.TotalRecords} total records. ${entry?.FailedRecords} failures. ${entry?.WarningRecords} warnings.`}</InputReadOnly>
              </Label>
            </div>
          </div>
        </>
      ) : (
        <InlineNotification theme="alert">{errorFetching}</InlineNotification>
      )}
      {isFetching && <Text>Loading ...</Text>}
    </Card>
  );
};
const ConnectedSystemLogEntryDetail = connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  entry: state.systemLogEntry,
  isFetching: state.fetchingSingleSystemLogEntry,
  errorFetching: state.fetchSingleSystemLogEntryError,
  query: state.query,
}))(SystemLogEntryDetail);

const PAGINATION_LIMIT = '50';

const fetchEvent =
  (
    selectedTenantID: string | undefined,
    params: URLSearchParams,
    entryID: string | undefined
  ) =>
  (dispatch: AppDispatch) => {
    if (!entryID) {
      return;
    }
    const paramsAsObject = getParamsAsObject(prefix, params);
    // if objects_limit is not specified in querystring,
    // use the default
    if (!paramsAsObject.limit) {
      paramsAsObject.limit = PAGINATION_LIMIT;
    }
    if (!selectedTenantID) {
      return;
    }
    dispatch(getSystemLogEntryDetailRequest());
    fetchSystemLogDetails(selectedTenantID, paramsAsObject, entryID).then(
      (data: PaginatedResult<SystemLogEntryRecord>) => {
        dispatch(getSystemLogEntryDetailSuccess(data));
      },
      (error: APIError) => {
        dispatch(getSystemLogEntryDetailRequestError(error.message));
      }
    );
  };

const changeSystemEventSearchFilter =
  (filter: Filter) => async (dispatch: AppDispatch) => {
    // TODO v2 Add Operator Column and set the operator with that
    dispatch(changeCurrentSystemLogEntryDetailSearchFilter(filter));
  };

const SystemLogList = ({
  selectedTenantID,
  entries,
  isFetching,
  fetchError,
  systemLogEntryDetailSearchFilter,
  runID,
  query,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  entries: PaginatedResult<SystemLogEntryRecord> | undefined;
  isFetching: boolean;
  fetchError: string;
  systemLogEntryDetailSearchFilter: Filter;
  runID: string;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenantID) {
      dispatch(fetchEvent(selectedTenantID, query, runID));
    }
  }, [selectedTenantID, query, runID, dispatch]);

  return (
    <Card title="Sync Records">
      <div className={PageCommon.tablecontrols}>
        <Search
          id="systemEvent"
          columns={columns}
          changeSearchFilter={(filter: Filter) => {
            dispatch(changeSystemEventSearchFilter(filter));
          }}
          searchFilter={systemLogEntryDetailSearchFilter}
          prefix={prefix}
        />
        <Pagination
          prev={entries?.prev}
          next={entries?.next}
          prefix={prefix}
          isLoading={false}
        />
      </div>
      {entries ? (
        <>
          {entries.data && entries.data.length ? (
            <Table spacing="packed" className={styles.styleslogentrysyncstable}>
              <TableHead floating>
                <TableRow>
                  <TableRowHead key="Event Type">Object ID</TableRowHead>
                  <TableRowHead key="Event ID">Event ID</TableRowHead>
                  <TableRowHead key="User ID">User ID</TableRowHead>
                  <TableRowHead key="Created">Created</TableRowHead>
                  <TableRowHead key="Warning">Warning</TableRowHead>
                  <TableRowHead key="Error">Error</TableRowHead>
                </TableRow>
              </TableHead>
              <TableBody>
                {entries?.data.map((entry) => (
                  <TableRow key={entry.id}>
                    <TableCell>{entry.ObjectID}</TableCell>
                    <TableCell>{entry.id}</TableCell>
                    <TableCell>{entry.UserID}</TableCell>
                    <TableCell>
                      {new Date(entry.created).toLocaleString('en-US')}
                    </TableCell>
                    <TableCell>{entry.Warning}</TableCell>
                    <TableCell>{entry.Error}</TableCell>
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
          )}
        </>
      ) : fetchError ? (
        <InlineNotification theme="alert">{fetchError}</InlineNotification>
      ) : isFetching ? (
        <LoaderDots size="small" assistiveText="Loading entries" />
      ) : (
        <InlineNotification theme="alert">No entries found</InlineNotification>
      )}
      {(entries?.has_prev || entries?.has_next) && (
        <Pagination
          prev={entries?.prev}
          next={entries?.next}
          prefix={prefix}
          isLoading={false}
        />
      )}
    </Card>
  );
};
const ConnectedSystemLogList = connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  entries: state.systemLogEntryRecords,
  isFetching: state.fetchingSystemLogEntry,
  fetchError: state.fetchSystemLogEntryError,
  systemLogEntryDetailSearchFilter: state.systemLogEntryDetailSearchFilter,
  query: state.query,
}))(SystemLogList);

const SystemEventDetailPage = ({
  routeParams,
}: {
  routeParams: Record<string, string>;
}) => {
  const { runID } = routeParams;

  return (
    <>
      {runID && (
        <>
          <ConnectedSystemLogEntryDetail runID={runID} />
          <ConnectedSystemLogList runID={runID} />
        </>
      )}
    </>
  );
};

export default connect((state: RootState) => ({
  routeParams: state.routeParams,
}))(SystemEventDetailPage);
