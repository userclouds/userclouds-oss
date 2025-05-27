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
  ToolTip,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';

import API from '../API';
import actions from '../actions';
import { RootState, AppDispatch } from '../store';
import ServiceInfo from '../ServiceInfo';
import LogRow from '../chart/LogRow';
import styles from './EventsLogPage.module.css';
import PageCommon from './PageCommon.module.css';

const fetchLogEvents = (tenantID: string) => (dispatch: AppDispatch) => {
  dispatch({
    type: actions.GET_EVENT_LOG_REQUEST,
  });
  API.fetchEventLog(tenantID).then(
    (events: LogRow[]) => {
      dispatch({
        type: actions.GET_EVENT_LOG_SUCCESS,
        data: events,
      });
    },
    (error: APIError) => {
      dispatch({
        type: actions.GET_EVENT_LOG_ERROR,
        data: error.message,
      });
    }
  );
};

const EventLogList = ({
  serviceInfo,
  logEvents,
  isFetching,
  fetchError,
}: {
  serviceInfo: ServiceInfo | undefined;
  logEvents: LogRow[] | undefined;
  isFetching: boolean;
  fetchError: string;
}) => {
  if (!serviceInfo || !serviceInfo.uc_admin) {
    return <></>;
  }

  if (fetchError) {
    return <InlineNotification theme="alert">{fetchError}</InlineNotification>;
  }

  return (
    <Card title="Recent Events" collapsible={false}>
      <Table className={styles.eventslogtable}>
        <TableHead>
          <TableRow key="heading">
            <TableRowHead>Time</TableRowHead>
            <TableRowHead>Name</TableRowHead>
            <TableRowHead>Type</TableRowHead>
            <TableRowHead>Service</TableRowHead>
            <TableRowHead>Count</TableRowHead>
          </TableRow>
        </TableHead>
        <TableBody>
          {logEvents?.length ? (
            logEvents.map((logEvent, index) => (
              <TableRow
                // eslint-disable-next-line react/no-array-index-key
                key={`event-${logEvent.id}-${logEvent.event_name}-${logEvent.service}-${index}`}
              >
                <TableCell>
                  {new Date(logEvent.timestamp).toLocaleString('en-US')}
                </TableCell>
                <TableCell>{logEvent.event_name}</TableCell>
                <TableCell>{logEvent.event_type}</TableCell>
                <TableCell>{logEvent.service}</TableCell>
                <TableCell>{logEvent.count}</TableCell>
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell colSpan="5">
                {isFetching ? 'No events found for this tenant' : 'Loading ...'}
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </Card>
  );
};

const ConnectedEventLogList = connect((state: RootState) => ({
  serviceInfo: state.serviceInfo,
  logEvents: state.logEvents,
  isFetching: state.fetchingActiveInstances,
  fetchError: state.activeInstancesFetchError,
}))(EventLogList);

const StatusPage = ({
  selectedTenantID,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenantID) {
      dispatch(fetchLogEvents(selectedTenantID));
    }
  }, [selectedTenantID, dispatch]);

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>Events for your tenant</>
          </ToolTip>
        </div>
      </div>
      <ConnectedEventLogList />
    </>
  );
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
}))(StatusPage);
