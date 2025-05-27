import { AnyAction } from 'redux';
import { RootState } from '../store';
import {
  CHANGE_CURRENT_AUDIT_LOG_SEARCH_FILTER,
  GET_AUDIT_LOG_ENTRIES_ERROR,
  GET_AUDIT_LOG_ENTRIES_REQUEST,
  GET_AUDIT_LOG_ENTRIES_SUCCESS,
  GET_DATA_ACCESS_LOG_ENTRIES_REQUEST,
  GET_DATA_ACCESS_LOG_ENTRIES_SUCCESS,
  GET_DATA_ACCESS_LOG_ENTRIES_ERROR,
  CHANGE_DATA_ACCESS_LOG_FILTER,
  GET_DATA_ACCESS_LOG_ENTRY_REQUEST,
  GET_DATA_ACCESS_LOG_ENTRY_SUCCESS,
  GET_DATA_ACCESS_LOG_ENTRY_ERROR,
} from '../actions/auditlog';
import { DataAccessLogEntry } from '../models/AuditLogEntry';
import { setOperatorsForFilter } from '../controls/SearchHelper';

const auditLogReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case GET_AUDIT_LOG_ENTRIES_REQUEST:
      state.fetchingAuditLogEntries = true;
      state.fetchAuditLogEntriesError = '';
      break;
    case GET_AUDIT_LOG_ENTRIES_SUCCESS:
      state.fetchingAuditLogEntries = false;
      state.auditLogEntries = action.data;
      break;
    case GET_AUDIT_LOG_ENTRIES_ERROR:
      state.fetchingAuditLogEntries = false;
      state.fetchAuditLogEntriesError = action.data;
      state.auditLogEntries = undefined;
      break;
    case CHANGE_CURRENT_AUDIT_LOG_SEARCH_FILTER:
      state.auditLogSearchFilter = {
        ...state.auditLogSearchFilter,
        ...setOperatorsForFilter(action.data),
      };
      break;
    case GET_DATA_ACCESS_LOG_ENTRIES_REQUEST:
      state.fetchingDataAccessLogEntries = true;
      state.fetchDataAccessLogEntriesError = '';
      state.dataAccessLogEntries = undefined;
      break;
    case GET_DATA_ACCESS_LOG_ENTRIES_SUCCESS:
      state.fetchingDataAccessLogEntries = false;
      state.dataAccessLogEntries = action.data;
      break;
    case GET_DATA_ACCESS_LOG_ENTRIES_ERROR:
      state.fetchingDataAccessLogEntries = false;
      state.fetchDataAccessLogEntriesError = action.data;
      state.dataAccessLogEntries = undefined;
      break;
    case CHANGE_DATA_ACCESS_LOG_FILTER:
      state.dataAccessLogFilter = {
        ...state.dataAccessLogFilter,
        ...action.data,
      };
      break;
    case GET_DATA_ACCESS_LOG_ENTRY_REQUEST:
      state.fetchingDataAccessLogEntries = true;
      state.fetchDataAccessLogEntriesError = '';
      state.dataAccessLogEntry = undefined;
      break;
    case GET_DATA_ACCESS_LOG_ENTRY_SUCCESS:
      state.fetchingDataAccessLogEntries = false;
      state.dataAccessLogEntry = action.data as DataAccessLogEntry;
      state.dataAccessLogEntry.payload.AccessPolicyContextStringified =
        JSON.stringify(
          state.dataAccessLogEntry.payload.AccessPolicyContext,
          null,
          2
        );
      state.dataAccessLogEntry.payload.SelectorValuesStringified =
        JSON.stringify(state.dataAccessLogEntry.payload.SelectorValues);
      break;
    case GET_DATA_ACCESS_LOG_ENTRY_ERROR:
      state.fetchingDataAccessLogEntries = false;
      state.fetchDataAccessLogEntriesError = action.data;
      state.dataAccessLogEntry = undefined;
      break;
    default:
      break;
  }
  return state;
};

export default auditLogReducer;
