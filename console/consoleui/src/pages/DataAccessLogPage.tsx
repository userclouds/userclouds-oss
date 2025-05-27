import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  Card,
  CardRow,
  EmptyState,
  InlineNotification,
  IconButton,
  IconCheck,
  IconClose,
  IconCopy,
  IconDatabase2,
  IconFilter,
  LoaderDots,
  Select,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  TextInput,
  ToolTip,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';
import Link from '../controls/Link';

import { makeCleanPageLink } from '../AppNavigation';
import { RootState, AppDispatch } from '../store';
import {
  DataAccessLogEntry,
  DATA_ACCESS_LOG_PAGINATION_LIMIT,
  DATA_ACCESS_LOG_PREFIX,
} from '../models/AuditLogEntry';
import PaginatedResult from '../models/PaginatedResult';
import { PAGINATION_ARGUMENTS } from '../controls/SearchHelper';
import { Column } from '../models/TenantUserStoreConfig';
import Accessor from '../models/Accessor';
import { fetchDataAccessLog } from '../API/auditlog';
import {
  changeDataAccessLogFilter,
  getDataAccessLogEntriesRequest,
  getDataAccessLogEntriesError,
  getDataAccessLogEntriesSuccess,
} from '../actions/auditlog';
import Pagination from '../controls/Pagination';
import { getParamsAsObject } from '../controls/PaginationHelper';
import PageCommon from './PageCommon.module.css';
import { fetchUserStoreConfig } from '../thunks/userstore';
import { fetchAccessors } from '../thunks/accessors';
import {
  DataAccessLogFilter,
  blankDataAccessLogFilter,
} from '../models/DataAccessLogFilter';
import { truncatedID } from '../util/id';
import { truncateWithEllipsis } from '../util/string';
import { redirect } from '../routing';
import Styles from './DataAccessLogPage.module.css';

const fetchEntries =
  (tenantID: string, params: URLSearchParams) => (dispatch: AppDispatch) => {
    const paramsAsObject = getParamsAsObject(DATA_ACCESS_LOG_PREFIX, params);
    if (!paramsAsObject.limit) {
      paramsAsObject.limit = DATA_ACCESS_LOG_PAGINATION_LIMIT;
    }
    const keys = Object.keys(blankDataAccessLogFilter);
    for (const key of keys) {
      if (params && params.get(DATA_ACCESS_LOG_PREFIX + key)) {
        paramsAsObject[key] = params.get(DATA_ACCESS_LOG_PREFIX + key) || '';
      }
    }

    dispatch(getDataAccessLogEntriesRequest());
    fetchDataAccessLog(tenantID, paramsAsObject).then(
      (data: PaginatedResult<DataAccessLogEntry>) => {
        dispatch(getDataAccessLogEntriesSuccess(data));
      },
      (error: APIError) => {
        dispatch(getDataAccessLogEntriesError(error));
      }
    );
  };

const addDataAccessLogFilterToSearchParams = (
  searchParams: URLSearchParams,
  filter: DataAccessLogFilter
) => {
  const data_access_pagination_arguments = PAGINATION_ARGUMENTS.map((arg) => {
    return DATA_ACCESS_LOG_PREFIX + arg;
  });
  const newParams = new URLSearchParams();
  for (const [key, val] of searchParams.entries()) {
    // clear pagination vars when filter changes
    if (!data_access_pagination_arguments.includes(key)) {
      newParams.append(key, val);
    }
  }
  const keys = Object.keys(filter);
  for (const key of keys) {
    const filterKey = key as keyof DataAccessLogFilter;
    if (filter[filterKey]) {
      newParams.set(DATA_ACCESS_LOG_PREFIX + key, filter[filterKey]);
    } else {
      newParams.delete(DATA_ACCESS_LOG_PREFIX + key);
    }
  }

  return newParams;
};

const DataAccessLogEntryList = ({
  selectedTenantID,
  columns,
  accessors,
  entries,
  isFetching,
  fetchError,
  dataAccessLogFilter,
  query,
  location,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  columns: Column[] | undefined;
  accessors: PaginatedResult<Accessor> | undefined;
  entries: PaginatedResult<DataAccessLogEntry> | undefined;
  isFetching: boolean;
  fetchError: string;
  dataAccessLogFilter: DataAccessLogFilter;
  query: URLSearchParams;
  location: URL;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenantID) {
      dispatch(fetchEntries(selectedTenantID, query));
      dispatch(fetchUserStoreConfig(selectedTenantID));
      dispatch(fetchAccessors(selectedTenantID, new URLSearchParams(), false));
    }
  }, [selectedTenantID, query, dispatch]);

  useEffect(() => {
    const filter = blankDataAccessLogFilter;
    for (const key of Object.keys(filter)) {
      const value = location.searchParams.get(DATA_ACCESS_LOG_PREFIX + key);
      filter[key as keyof DataAccessLogFilter] = value || '';
    }
    dispatch(changeDataAccessLogFilter(filter));
  }, [location, dispatch]);

  const cleanQuery = makeCleanPageLink(query);

  return (
    <>
      <form
        id="dataAccessLogFilterForm"
        onSubmit={(e: React.FormEvent) => {
          e.preventDefault();
          const newSearchParams = addDataAccessLogFilterToSearchParams(
            query,
            dataAccessLogFilter
          );
          redirect(`${location.pathname}?${newSearchParams.toString()}`);
        }}
      >
        <div className={PageCommon.listviewtablecontrols}>
          <div>
            <IconFilter size="medium" />
          </div>
          Column:
          <Select
            value={dataAccessLogFilter.column_id}
            onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
              e.preventDefault();
              const column_id = e.currentTarget.value;
              dispatch(changeDataAccessLogFilter({ column_id }));
            }}
          >
            <option value="">All</option>
            {columns?.map((col) => (
              <option value={col.id} key={col.id}>
                {col.name}
              </option>
            ))}
          </Select>
          Accessor:
          <Select
            value={dataAccessLogFilter.accessor_id}
            onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
              e.preventDefault();
              const accessor_id = e.currentTarget.value;
              dispatch(changeDataAccessLogFilter({ accessor_id }));
            }}
          >
            <option value="">All</option>
            {accessors?.data.map((acc) => (
              <option value={acc.id} key={acc.id}>
                {truncateWithEllipsis(acc.name, 40)}
              </option>
            ))}
          </Select>
          Actor ID:
          <TextInput
            value={dataAccessLogFilter.actor_id}
            onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
              e.preventDefault();
              const actor_id = e.currentTarget.value;
              dispatch(changeDataAccessLogFilter({ actor_id }));
            }}
          />
          Target ID:
          <TextInput
            value={dataAccessLogFilter.selector_value}
            onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
              e.preventDefault();
              const selector_value = e.currentTarget.value;
              dispatch(changeDataAccessLogFilter({ selector_value }));
            }}
          />
          <Button type="submit" theme="secondary">
            Apply
          </Button>
          <ToolTip>
            <>Review the data access log for your tenant.</>
          </ToolTip>
        </div>
      </form>
      <Card listview>
        <div className={PageCommon.listviewpaginationcontrols}>
          <Pagination
            prev={entries?.prev}
            next={entries?.next}
            prefix={DATA_ACCESS_LOG_PREFIX}
            isLoading={false}
          />
        </div>
        {entries ? (
          entries.data && entries.data.length ? (
            <Table spacing="nowrap" className={Styles.dataAccessLogTable}>
              <TableHead floating>
                <TableRow>
                  <TableRowHead
                    key="accessor_name"
                    className={Styles.accessorCol}
                  >
                    Accessor Name
                  </TableRowHead>
                  <TableRowHead key="version" className={Styles.versionCol}>
                    Version
                  </TableRowHead>
                  <TableRowHead key="columns" className={Styles.columnsCol}>
                    Columns
                  </TableRowHead>
                  <TableRowHead key="purposes" className={Styles.purposesCol}>
                    Purposes
                  </TableRowHead>
                  <TableRowHead key="completed" className={Styles.completedCol}>
                    Completed
                  </TableRowHead>
                  <TableRowHead key="rows" className={Styles.rowsCol}>
                    Rec. #
                  </TableRowHead>
                  <TableRowHead key="actor" className={Styles.actorCol}>
                    Actor
                  </TableRowHead>
                  <TableRowHead key="time" className={Styles.timeCol}>
                    Time
                  </TableRowHead>
                </TableRow>
              </TableHead>
              <TableBody>
                {entries.data.map((entry) => (
                  <TableRow key={entry.id}>
                    <TableCell>{entry.accessor_name}</TableCell>
                    <TableCell>{entry.accessor_version}</TableCell>
                    <TableCell>{entry.columns}</TableCell>
                    <TableCell>{entry.purposes}</TableCell>
                    <TableCell>
                      {entry.completed ? (
                        <IconCheck className={Styles.iconCheck} />
                      ) : (
                        <IconClose className={Styles.iconClose} />
                      )}
                    </TableCell>
                    <TableCell>{entry.rows}</TableCell>
                    <TableCell>
                      <IconButton
                        className={Styles.copyIcon}
                        icon={<IconCopy />}
                        onClick={() => {
                          navigator.clipboard.writeText(entry.actor_id);
                        }}
                        title="Copy to Clipboard"
                        aria-label="Copy to Clipboard"
                      />
                      &nbsp;
                      {entry.actor_name
                        ? entry.actor_name + truncatedID(entry.actor_id)
                        : entry.actor_id}
                    </TableCell>
                    <TableCell>
                      <Link
                        href={location.pathname + '/' + entry.id + cleanQuery}
                      >
                        {new Date(entry.created).toLocaleString('en-US')}
                      </Link>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <CardRow>
              <EmptyState
                title="No entries in the data access log."
                image={<IconDatabase2 size="large" />}
              />
            </CardRow>
          )
        ) : fetchError !== '' ? (
          <InlineNotification theme="alert">{fetchError}</InlineNotification>
        ) : isFetching ? (
          <LoaderDots size="small" assistiveText="Loading entries" />
        ) : (
          <InlineNotification theme="alert">
            No entries found
          </InlineNotification>
        )}
      </Card>
    </>
  );
};
const ConnectedEntryList = connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  columns: state.userStoreColumns,
  accessors: state.accessors,
  entries: state.dataAccessLogEntries,
  isFetching: state.fetchingDataAccessLogEntries,
  fetchError: state.fetchDataAccessLogEntriesError,
  dataAccessLogFilter: state.dataAccessLogFilter,
  query: state.query,
  location: state.location,
  featureFlags: state.featureFlags,
}))(DataAccessLogEntryList);

export default ConnectedEntryList;
