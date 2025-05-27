import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Card,
  CardRow,
  CodeEditor,
  IconCheck,
  IconClose,
  InlineNotification,
  InputReadOnly,
  Label,
  LoaderDots,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  TextShortener,
  ToolTip,
} from '@userclouds/ui-component-lib';

import { APIError } from '@userclouds/sharedui';
import Link from '../controls/Link';
import { makeCleanPageLink } from '../AppNavigation';

import { RootState, AppDispatch } from '../store';
import { DataAccessLogEntry } from '../models/AuditLogEntry';
import { fetchDataAccessLogEntry } from '../API/auditlog';
import {
  getDataAccessLogEntryRequest,
  getDataAccessLogEntryError,
  getDataAccessLogEntrySuccess,
} from '../actions/auditlog';
import { fetchUserStoreConfig } from '../thunks/userstore';
import { fetchAccessors } from '../thunks/accessors';
import { PageTitle } from '../mainlayout/PageWrap';
import PageCommon from './PageCommon.module.css';
import Styles from './DataAccessLogPage.module.css';

const fetchEntry =
  (tenantID: string, entryID: string) => (dispatch: AppDispatch) => {
    dispatch(getDataAccessLogEntryRequest());
    fetchDataAccessLogEntry(tenantID, entryID).then(
      (data: DataAccessLogEntry) => {
        dispatch(getDataAccessLogEntrySuccess(data));
      },
      (error: APIError) => {
        dispatch(getDataAccessLogEntryError(error));
      }
    );
  };

const DataAccessLogDetailsPage = ({
  dispatch,
  query,
  routeParams,
  selectedTenantID,
  entry,
  isFetching,
  fetchError,
}: {
  dispatch: AppDispatch;
  query: URLSearchParams;
  routeParams: Record<string, string>;
  selectedTenantID: string | undefined;
  entry: DataAccessLogEntry | undefined;
  isFetching: boolean;
  fetchError: string;
}) => {
  const { entryID } = routeParams;
  useEffect(() => {
    if (selectedTenantID) {
      dispatch(fetchEntry(selectedTenantID, entryID));
      dispatch(fetchUserStoreConfig(selectedTenantID));
      dispatch(fetchAccessors(selectedTenantID, new URLSearchParams(), false));
    }
  }, [selectedTenantID, entryID, dispatch]);

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <PageTitle
          title="View Data Access Log"
          itemName={
            entry?.created
              ? new Date(entry.created).toLocaleString('en-US')
              : ''
          }
        />
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>Viewing details of this data access event.</>
          </ToolTip>
        </div>
      </div>

      <Card detailview>
        {entry ? (
          <>
            <CardRow title="Basic details" collapsible>
              <div className={PageCommon.carddetailsrow}>
                <Label>
                  Accessor Name
                  <br />
                  <Link
                    key={entry.accessor_id}
                    href={
                      `/accessors/${entry.accessor_id}/${entry.accessor_version}` +
                      makeCleanPageLink(query)
                    }
                  >{`${entry.accessor_name} (v${entry.accessor_version})`}</Link>
                </Label>
                <Label htmlFor="accessor_id">
                  ID
                  <br />
                  <TextShortener
                    text={entry.accessor_id}
                    length={6}
                    id="accessor_id"
                  />
                </Label>
                <Label>
                  # Records Accessed
                  <br />
                  <InputReadOnly>{entry.rows}</InputReadOnly>
                </Label>
                <Label>
                  Completed
                  <br />
                  {entry.completed ? (
                    <IconCheck className={Styles.iconCheck} />
                  ) : (
                    <IconClose className={Styles.iconClose} />
                  )}
                </Label>
                <Label>
                  Purposes
                  <br />
                  <InputReadOnly>{entry.purposes}</InputReadOnly>
                </Label>
                <Label>
                  Actor
                  <br />
                  <InputReadOnly>{entry.actor_name}</InputReadOnly>
                </Label>
                <Label>
                  Columns
                  <br />
                  <InputReadOnly>{entry.columns}</InputReadOnly>
                </Label>
                <Label>
                  Selector Values
                  <br />
                  <InputReadOnly>
                    {entry.payload.SelectorValuesStringified}
                  </InputReadOnly>
                </Label>
              </div>
            </CardRow>
            <CardRow title="Access policy context" collapsible>
              <CodeEditor
                id="payload"
                value={entry.payload.AccessPolicyContextStringified}
                readOnly
                jsonExt
              />
            </CardRow>
            <CardRow title="Row counts" collapsible>
              <Table className={Styles.rowcountstable}>
                <TableHead>
                  <TableRow>
                    <TableRowHead key="name" />
                    <TableRowHead key="count">Count</TableRowHead>
                  </TableRow>
                </TableHead>
                <TableBody>
                  <TableRow key="matching-rows">
                    <TableCell>Rows matching selector</TableCell>
                    <TableCell>
                      {entry.payload.SelectorRowCount || '0'}
                    </TableCell>
                  </TableRow>
                  <TableRow key="denied-rows">
                    <TableCell>Rows denied by access policy</TableCell>
                    <TableCell>
                      {entry.payload.AccessPolicyDeniedCount || '0'}
                    </TableCell>
                  </TableRow>
                  <TableRow key="returned-rows">
                    <TableCell>Rows returned</TableCell>
                    <TableCell>{entry.payload.RowsReturned || '0'}</TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </CardRow>
          </>
        ) : fetchError !== '' ? (
          <InlineNotification theme="alert">{fetchError}</InlineNotification>
        ) : isFetching ? (
          <LoaderDots size="small" assistiveText="Loading entry" />
        ) : (
          <InlineNotification theme="alert">No entry found</InlineNotification>
        )}
      </Card>
    </>
  );
};

export default connect((state: RootState) => ({
  routeParams: state.routeParams,
  query: state.query,
  selectedTenantID: state.selectedTenantID,
  entry: state.dataAccessLogEntry,
  isFetching: state.fetchingDataAccessLogEntries,
  fetchError: state.fetchDataAccessLogEntriesError,
}))(DataAccessLogDetailsPage);
