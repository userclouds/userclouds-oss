import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Card,
  CardRow,
  Checkbox,
  CodeEditor,
  EmptyState,
  InlineNotification,
  IconButton,
  IconCopy,
  IconDatabase2,
  IconFilter,
  LoaderDots,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  ToolTip,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';

import { RootState, AppDispatch } from '../store';
import {
  AuditLogEntry,
  AUDIT_LOG_COLUMNS,
  AUDIT_LOG_PAGINATION_LIMIT,
  AUDIT_LOG_PREFIX,
} from '../models/AuditLogEntry';
import PaginatedResult from '../models/PaginatedResult';
import { Filter } from '../models/authz/SearchFilters';
import { fetchAuditLog } from '../API/auditlog';
import {
  changeCurrentAuditLogSearchFilter,
  getAuditLogEntriesRequest,
  getAuditLogEntriesRequestError,
  getAuditLogEntriesSuccess,
} from '../actions/auditlog';
import Pagination from '../controls/Pagination';
import { PAGINATION_ARGUMENTS } from '../controls/SearchHelper';
import Search from '../controls/Search';
import { getParamsAsObject } from '../controls/PaginationHelper';
import PageCommon from './PageCommon.module.css';
import { truncatedID } from '../util/id';
import { redirect } from '../routing';
import Styles from './AuditLogPage.module.css';

const fetchEntries =
  (tenantID: string, params: URLSearchParams, configOnly: boolean) =>
  (dispatch: AppDispatch) => {
    const paramsAsObject = getParamsAsObject(AUDIT_LOG_PREFIX, params);
    if (!paramsAsObject.limit) {
      paramsAsObject.limit = AUDIT_LOG_PAGINATION_LIMIT;
    }
    if (configOnly) {
      paramsAsObject.config_changes = 'true';
    }

    dispatch(getAuditLogEntriesRequest());
    fetchAuditLog(tenantID, paramsAsObject).then(
      (data: PaginatedResult<AuditLogEntry>) => {
        dispatch(getAuditLogEntriesSuccess(data));
      },
      (error: APIError) => {
        dispatch(getAuditLogEntriesRequestError(error));
      }
    );
  };

const toggleConfigChangesOnly = (searchParams: URLSearchParams) => {
  const audit_log_pagination_arguments = PAGINATION_ARGUMENTS.map((arg) => {
    return AUDIT_LOG_PREFIX + arg;
  });
  const newParams = new URLSearchParams();
  for (const [key, val] of searchParams.entries()) {
    // clear pagination vars when filter changes
    if (!audit_log_pagination_arguments.includes(key)) {
      newParams.append(key, val);
    }
  }

  if (searchParams.get('config_changes')) {
    newParams.delete('config_changes');
  } else {
    newParams.append('config_changes', 'true');
  }

  return newParams;
};

const AuditEntryList = ({
  selectedTenantID,
  entries,
  isFetching,
  fetchError,
  auditLogSearchFilter,
  query,
  location,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  entries: PaginatedResult<AuditLogEntry> | undefined;
  isFetching: boolean;
  fetchError: string;
  auditLogSearchFilter: Filter;
  query: URLSearchParams;
  location: URL;
  dispatch: AppDispatch;
}) => {
  const configChangesOnly = query.get('config_changes') !== null;

  useEffect(() => {
    if (selectedTenantID) {
      dispatch(fetchEntries(selectedTenantID, query, configChangesOnly));
    }
  }, [selectedTenantID, configChangesOnly, query, dispatch]);

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div>
          <IconFilter />
        </div>
        <Search
          id="auditLog"
          columns={AUDIT_LOG_COLUMNS}
          changeSearchFilter={(filter: Filter) => {
            dispatch(changeCurrentAuditLogSearchFilter(filter));
          }}
          searchFilter={auditLogSearchFilter}
          prefix={AUDIT_LOG_PREFIX}
        />
        <Checkbox
          name="configChangesOnly"
          onChange={(e: React.ChangeEvent) => {
            e.preventDefault();
            const newSearchParams = toggleConfigChangesOnly(query);
            redirect(`${location.pathname}?${newSearchParams.toString()}`);
          }}
          checked={configChangesOnly}
        >
          Config changes only
        </Checkbox>
        <ToolTip>
          <>Review the audit log for your tenant.</>
        </ToolTip>
      </div>
      <Card listview>
        <div className={PageCommon.listviewpaginationcontrols}>
          <Pagination
            prev={entries?.prev}
            next={entries?.next}
            prefix={AUDIT_LOG_PREFIX}
            isLoading={false}
          />
        </div>

        {entries ? (
          entries.data && entries.data.length ? (
            <Table spacing="nowrap" className={Styles.auditLogTable}>
              <TableHead floating>
                <TableRow>
                  <TableRowHead key="type">Call Type</TableRowHead>
                  <TableRowHead key="resource">Resource</TableRowHead>
                  <TableRowHead key="actor">Actor</TableRowHead>
                  <TableRowHead key="time">Time</TableRowHead>
                  <TableRowHead className={Styles.chevronCol} />
                </TableRow>
              </TableHead>
              <TableBody>
                {entries.data.map((entry) => (
                  <TableRow
                    key={entry.id}
                    isExtensible
                    columns={4}
                    leadColumns={1}
                    expandedContent={
                      <div>
                        <CodeEditor
                          id="selector_values"
                          value={JSON.stringify(entry.payload, null, 2)}
                          readOnly
                          jsonExt
                        />
                      </div>
                    }
                  >
                    <TableCell className={Styles.primaryCell}>
                      {entry.type}
                    </TableCell>
                    <TableCell>
                      <>
                        {entry.payload.ID && (
                          <>
                            <IconButton
                              className={Styles.copyIcon}
                              size="tiny"
                              icon={<IconCopy />}
                              onClick={() => {
                                navigator.clipboard.writeText(
                                  entry.payload.ID as string
                                );
                              }}
                              title="Copy to Clipboard"
                              aria-label="Copy to Clipboard"
                            />
                            &nbsp;
                          </>
                        )}
                        {entry.payload.Name
                          ? entry.payload.Name +
                            truncatedID(entry.payload.ID as string)
                          : entry.payload.ID}
                      </>
                    </TableCell>
                    <TableCell>
                      <>
                        <IconButton
                          className={Styles.copyIcon}
                          icon={<IconCopy />}
                          size="tiny"
                          onClick={() => {
                            navigator.clipboard.writeText(entry.actor_id);
                          }}
                          title="Copy to Clipboard"
                          aria-label="Copy to Clipboard"
                        />
                        &nbsp;
                      </>
                      {entry.actor_name
                        ? entry.actor_name + truncatedID(entry.actor_id)
                        : entry.actor_id}
                    </TableCell>
                    <TableCell>
                      {new Date(entry.created).toLocaleString('en-US')}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <CardRow>
              <EmptyState
                title="No entries in the audit log."
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
  entries: state.auditLogEntries,
  isFetching: state.fetchingAuditLogEntries,
  fetchError: state.fetchAuditLogEntriesError,
  auditLogSearchFilter: state.auditLogSearchFilter,
  query: state.query,
  location: state.location,
}))(AuditEntryList);

export default ConnectedEntryList;
