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
  TextShortener,
  ToolTip,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';

import API from '../API';
import actions from '../actions';
import { RootState, AppDispatch } from '../store';
import ServiceInfo from '../ServiceInfo';
import ActiveInstance from '../chart/ActiveInstance';
import styles from './ResourceListPage.module.css';
import PageCommon from './PageCommon.module.css';

const fetchActiveInstances = (tenantID: string) => (dispatch: AppDispatch) => {
  dispatch({
    type: actions.GET_ACTIVE_INSTANCES_REQUEST,
  });
  API.fetchActiveInstances(tenantID).then(
    (instances: ActiveInstance[]) => {
      dispatch({
        type: actions.GET_ACTIVE_INSTANCES_SUCCESS,
        data: instances,
      });
    },
    (error: APIError) => {
      dispatch({
        type: actions.GET_ACTIVE_INSTANCES_ERROR,
        data: error.message,
      });
    }
  );
};

const InfraInstanceList = ({
  serviceInfo,
  activeInstances,
  isFetching,
  fetchError,
}: {
  serviceInfo: ServiceInfo | undefined;
  activeInstances: ActiveInstance[] | undefined;
  isFetching: boolean;
  fetchError: string;
}) => {
  if (fetchError) {
    return <InlineNotification theme="alert">{fetchError}</InlineNotification>;
  }

  if (!serviceInfo || !serviceInfo.uc_admin) {
    return <Text>Service is operating normally</Text>;
  }

  return (
    <Card title="Infrastructure Resources" collapsible={false}>
      <Table spacing="packed" className={styles.resourcestatustable}>
        <TableHead>
          <TableRow key="heading">
            <TableRowHead>Instance ID</TableRowHead>
            <TableRowHead>Service</TableRowHead>
            <TableRowHead>Startup Time</TableRowHead>
            <TableRowHead>Last Active Time</TableRowHead>
            <TableRowHead>Event Count</TableRowHead>
            <TableRowHead>Input Errors</TableRowHead>
            <TableRowHead>System Errors</TableRowHead>
            <TableRowHead>Status</TableRowHead>
          </TableRow>
        </TableHead>
        <TableBody>
          {activeInstances?.length ? (
            activeInstances.map((instance) => (
              <TableRow key={`instance_${instance.instance_id}`}>
                <TableCell>
                  <TextShortener text={instance.instance_id} length={6} />
                </TableCell>
                <TableCell>{instance.service}</TableCell>
                <TableCell>
                  {new Date(instance.startup_time).toLocaleString('en-US')}
                </TableCell>
                <TableCell>
                  {new Date(instance.last_activity).toLocaleString('en-US')}
                </TableCell>
                <TableCell>{instance.event_count}</TableCell>
                <TableCell>{instance.error_input_count}</TableCell>
                <TableCell>{instance.error_internal_count}</TableCell>
                <TableCell>Healthy</TableCell>
              </TableRow>
            ))
          ) : (
            <TableRow>
              <TableCell colSpan="8">
                {isFetching
                  ? 'No active infra resources found for this tenant'
                  : 'Loading ...'}
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </Card>
  );
};

const ConnectedInfraInstanceList = connect((state: RootState) => ({
  serviceInfo: state.serviceInfo,
  activeInstances: state.activeInstances,
  isFetching: state.fetchingActiveInstances,
  fetchError: state.activeInstancesFetchError,
}))(InfraInstanceList);

const InfraResourceListPage = ({
  selectedTenantID,
  service,
  timePeriod,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  service: string;
  timePeriod: string;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenantID) {
      dispatch(fetchActiveInstances(selectedTenantID));
    }
  }, [selectedTenantID, service, timePeriod, dispatch]);

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>Infrastructure resources for your tenant</>
          </ToolTip>
        </div>
      </div>

      <ConnectedInfraInstanceList />
    </>
  );
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  service: state.chartDataService,
  timePeriod: state.chartDataTimePeriod,
}))(InfraResourceListPage);
