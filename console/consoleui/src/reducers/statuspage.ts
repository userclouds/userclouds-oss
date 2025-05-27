import { AnyAction } from 'redux';
import { RootState } from '../store';
import actions from '../actions';

const statusPageReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case actions.GET_ACTIVE_INSTANCES_REQUEST:
      state.fetchingActiveInstances = true;
      state.activeInstancesFetchError = '';
      state.activeInstances = undefined;
      break;
    case actions.GET_ACTIVE_INSTANCES_SUCCESS:
      state.fetchingActiveInstances = false;
      state.activeInstances = action.data;
      break;
    case actions.GET_ACTIVE_INSTANCES_ERROR:
      state.fetchingActiveInstances = false;
      state.activeInstancesFetchError = action.data;
      break;
    case actions.GET_EVENT_LOG_REQUEST:
      state.fetchingLogEvents = true;
      state.logEventsFetchError = '';
      state.logEvents = undefined;
      break;
    case actions.GET_EVENT_LOG_SUCCESS:
      state.fetchingLogEvents = false;
      state.logEvents = action.data;
      break;
    case actions.GET_EVENT_LOG_ERROR:
      state.fetchingLogEvents = false;
      state.logEventsFetchError = action.data;
      break;
    case actions.GET_CHART_DATA_REQUEST:
      state.fetchingChartData = true;
      state.chartData = undefined;
      state.chartDataFetchError = '';
      break;
    case actions.GET_CHART_DATA_SUCCESS:
      state.fetchingChartData = false;
      state.chartData = action.data;
      break;
    case actions.GET_CHART_DATA_ERROR:
      state.fetchingChartData = false;
      state.chartDataFetchError = action.data;
      break;
    case actions.CHANGE_CHART_DATA_SERVICE:
      state.chartDataService = action.data;
      break;
    case actions.CHANGE_CHART_DATA_TIME_PERIOD:
      state.chartDataTimePeriod = action.data;
      break;
    default:
      break;
  }

  return state;
};

export default statusPageReducer;
