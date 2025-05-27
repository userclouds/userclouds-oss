import React, { useEffect, useState } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  Card,
  InlineNotification,
  Label,
  Select,
  Text,
  ToolTip,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';

import { fetchChartData } from '../API/metrics';
import actions from '../actions';
import { RootState, AppDispatch } from '../store';
import ChartQuery from '../chart/ChartQuery';

import styles from './StatusPage.module.css';
import PageCommon from './PageCommon.module.css';
import RechartElement from '../chart/RechartElement';
import type {
  RequestChartsMetadata,
  QuerySet,
  ChartRenderableData,
  RequestChartDetails,
} from '../models/Chart';
import serviceChartMetadata from '../chart/ChartMetadataPerService';

const fetchChart =
  (tenantID: string, queries: ChartQuery[]) => (dispatch: AppDispatch) => {
    dispatch({
      type: actions.GET_CHART_DATA_REQUEST,
    });
    fetchChartData(tenantID, queries).then(
      (data: ChartRenderableData[][]) => {
        dispatch({
          type: actions.GET_CHART_DATA_SUCCESS,
          data,
        });
      },
      (error: APIError) => {
        dispatch({
          type: actions.GET_CHART_DATA_ERROR,
          data: error.message,
        });
      }
    );
  };

const buildChartQueries = (
  tenantID: string,
  timePeriod: string,
  service: string
) => {
  const { charts } = (serviceChartMetadata as RequestChartsMetadata)[service];

  const queries: ChartQuery[] = [];

  charts.forEach((chart: RequestChartDetails) => {
    if (chart.querySets) {
      chart.querySets.forEach((querySet) => {
        queries.push(new ChartQuery(querySet.eventTypes));
      });
    }
  });

  queries.forEach((query: ChartQuery) => {
    query.setTarget(tenantID, service);
    query.setTimePeriod(timePeriod);
  });

  return queries;
};

const ChartGrid = ({
  selectedTenantID,
  chartData,
  isFetching,
  fetchError,
  service,
  timePeriod,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  chartData: ChartRenderableData[][] | undefined;
  isFetching: boolean;
  fetchError: string;
  service: string;
  timePeriod: string;
  dispatch: AppDispatch;
}) => {
  // TODO: redux for this
  const [endTime, setEndTime] = useState<string>('');

  const { charts } = (serviceChartMetadata as RequestChartsMetadata)[service];

  useEffect(() => {
    const endDate = new Date();
    setEndTime(endDate.toLocaleString('en-US'));
  }, [timePeriod]);

  let querySetIndex = 0;

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div id="config" className={styles.selectorRow}>
          <Label>
            Metrics for:
            <Select
              name="Service"
              value={service}
              onChange={(event: React.ChangeEvent) => {
                if (!selectedTenantID) {
                  return;
                }
                const newService = (event.target as HTMLSelectElement).value;
                dispatch(
                  fetchChart(
                    selectedTenantID,
                    buildChartQueries(selectedTenantID, timePeriod, newService)
                  )
                );
                // TODO: we're launching this and the GET_CHART_DATA_REQUEST action
                dispatch({
                  type: actions.CHANGE_CHART_DATA_SERVICE,
                  data: newService,
                });
              }}
            >
              <option value="plex">Authentication</option>
              <option value="idp">Identity Provider</option>
              <option value="authz">Authorization</option>
              <option value="tokenizer">Tokenization</option>
              <option value="console">Administrative</option>
            </Select>
          </Label>
          <Label>
            Time period:
            <Select
              value={timePeriod}
              onChange={(event: React.ChangeEvent) => {
                if (!selectedTenantID) {
                  return;
                }
                const newTimePeriod = (event.target as HTMLSelectElement).value;
                dispatch(
                  fetchChart(
                    selectedTenantID,
                    buildChartQueries(selectedTenantID, newTimePeriod, service)
                  )
                );
                // TODO: we're launching this and the GET_CHART_DATA_REQUEST action
                dispatch({
                  type: actions.CHANGE_CHART_DATA_TIME_PERIOD,
                  data: newTimePeriod,
                });
              }}
            >
              <option value="minutes">Last 10 minutes</option>
              <option value="hour">Last hour</option>
              <option value="day">Last day</option>
              <option value="week">Last week</option>
            </Select>
          </Label>
          <Label htmlFor="refresh_button">
            <Button
              type="button"
              theme="secondary"
              size="small"
              id="refresh_button"
              onClick={() => {
                if (!selectedTenantID) {
                  return;
                }
                const endDate = new Date();
                setEndTime(endDate.toLocaleString('en-US'));
                dispatch(
                  fetchChart(
                    selectedTenantID,
                    buildChartQueries(selectedTenantID, timePeriod, service)
                  )
                );
              }}
            >
              Refresh
            </Button>
          </Label>
          <Label className={styles.textLabel}>Displayed Up To: {endTime}</Label>
        </div>
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>Metrics and events for your tenant</>
          </ToolTip>
        </div>
      </div>
      <Card className={styles.metricsCard} collapsible={false} id="metricsCard">
        {fetchError && (
          <InlineNotification theme="alert">{fetchError}</InlineNotification>
        )}

        <div className={styles.chartGrid}>
          {charts?.length && chartData?.length ? (
            charts.map((chart: RequestChartDetails) => {
              const renderableQuerySets = chart?.querySets?.map((querySet) => {
                const dataSet = {
                  label: querySet.name,
                  data: chartData[
                    querySetIndex
                  ].reverse() as ChartRenderableData[], // reverse so that "0" appears at the right side of the chart
                };

                querySetIndex++;
                return dataSet;
              }) as QuerySet[];

              return (
                <div className={styles.chartElementContainer} key={chart.title}>
                  <h2>{chart.title}</h2>
                  <RechartElement
                    querySets={renderableQuerySets}
                    timePeriod={timePeriod}
                  />
                </div>
              );
            })
          ) : (
            <Text>
              {isFetching
                ? 'Loading ...'
                : 'No data to display for this service'}
            </Text>
          )}
        </div>
      </Card>
    </>
  );
};

const ConnectedChartGrid = connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  serviceInfo: state.serviceInfo,
  chartData: state.chartData,
  isFetching: state.fetchingChartData,
  service: state.chartDataService,
  timePeriod: state.chartDataTimePeriod,
  fetchError: state.chartDataFetchError,
}))(ChartGrid);

const StatusPage = ({
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
      dispatch(
        fetchChart(
          selectedTenantID,
          buildChartQueries(selectedTenantID, timePeriod, service)
        )
      );
    }
  }, [selectedTenantID, service, timePeriod, dispatch]);

  return (
    <>
      <ConnectedChartGrid />
    </>
  );
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  service: state.chartDataService,
  timePeriod: state.chartDataTimePeriod,
}))(StatusPage);
