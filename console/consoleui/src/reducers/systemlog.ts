import { AnyAction } from 'redux';
import { RootState } from '../store';
import { Operators } from '../models/authz/SearchFilters';
import {
  GET_SYSTEM_LOG_ENTRIES_ERROR,
  GET_SYSTEM_LOG_ENTRIES_REQUEST,
  GET_SYSTEM_LOG_ENTRIES_SUCCESS,
  CHANGE_CURRENT_SYSTEM_LOG_SEARCH_FILTER,
  GET_SYSTEM_LOG_ENTRY_DETAIL_ERROR,
  GET_SYSTEM_LOG_ENTRY_DETAIL_REQUEST,
  GET_SYSTEM_LOG_ENTRY_DETAIL_SUCCESS,
  CHANGE_CURRENT_SYSTEM_LOG_ENTRY_DETAIL_SEARCH_FILTER,
  GET_SINGLE_SYSTEM_LOG_ENTRY_DETAIL_ERROR,
  GET_SINGLE_SYSTEM_LOG_ENTRY_DETAIL_REQUEST,
  GET_SINGLE_SYSTEM_LOG_ENTRY_DETAIL_SUCCESS,
} from '../actions/systemLog';
import {
  DATE_COLUMNS,
  STRING_COLUMNS,
  UUID_COLUMNS,
} from '../controls/SearchHelper';

const systemLogReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case GET_SYSTEM_LOG_ENTRIES_REQUEST:
      state.fetchingSystemLogEntries = true;
      state.systemLogEntries = undefined;
      state.fetchSystemLogEntriesError = '';
      break;
    case GET_SYSTEM_LOG_ENTRIES_SUCCESS:
      state.fetchingSystemLogEntries = false;
      state.systemLogEntries = action.data;
      break;
    case GET_SYSTEM_LOG_ENTRIES_ERROR:
      state.fetchingSystemLogEntries = false;
      state.fetchSystemLogEntriesError = action.data.e.message;
      break;

    case CHANGE_CURRENT_SYSTEM_LOG_SEARCH_FILTER: {
      state.systemLogSearchFilter = { ...action.data };
      const systemLogColumn = state.systemLogSearchFilter.columnName;
      if (STRING_COLUMNS.includes(systemLogColumn)) {
        state.systemLogSearchFilter.operator = Operators.LIKE;
      } else if (UUID_COLUMNS.includes(systemLogColumn)) {
        state.systemLogSearchFilter.operator = Operators.EQUAL;
      } else if (DATE_COLUMNS.includes(systemLogColumn)) {
        state.systemLogSearchFilter.operator = Operators.GREATER_THAN_EQUAL;
        state.systemLogSearchFilter.operator2 = Operators.LESS_THAN_EQUAL;
      }
      break;
    }

    case GET_SINGLE_SYSTEM_LOG_ENTRY_DETAIL_REQUEST:
      state.fetchingSingleSystemLogEntry = true;
      state.fetchSingleSystemLogEntryError = '';
      break;
    case GET_SINGLE_SYSTEM_LOG_ENTRY_DETAIL_SUCCESS:
      {
        state.fetchingSingleSystemLogEntry = false;
        const response = action.data;
        if (response && response.data && response.data.length) {
          state.systemLogEntry = response.data[0];
        } else {
          state.systemLogEntry = undefined;
          state.fetchSingleSystemLogEntryError = 'Entry not found';
        }
      }
      break;
    case GET_SINGLE_SYSTEM_LOG_ENTRY_DETAIL_ERROR:
      state.fetchingSingleSystemLogEntry = false;
      state.fetchSingleSystemLogEntryError = action.data;
      break;
    case GET_SYSTEM_LOG_ENTRY_DETAIL_REQUEST:
      state.fetchingSystemLogEntry = true;
      state.systemLogEntryRecords = undefined;
      state.fetchSystemLogEntryError = '';
      break;
    case GET_SYSTEM_LOG_ENTRY_DETAIL_SUCCESS:
      state.fetchingSystemLogEntry = false;
      state.systemLogEntryRecords = action.data;
      break;
    case GET_SYSTEM_LOG_ENTRY_DETAIL_ERROR:
      state.fetchingSystemLogEntry = false;
      state.fetchSystemLogEntryError = action.data;
      break;

    case CHANGE_CURRENT_SYSTEM_LOG_ENTRY_DETAIL_SEARCH_FILTER: {
      state.systemLogEntryDetailSearchFilter = { ...action.data };
      const entryDetailColumn =
        state.systemLogEntryDetailSearchFilter.columnName;
      if (STRING_COLUMNS.includes(entryDetailColumn)) {
        state.systemLogEntryDetailSearchFilter.operator = Operators.LIKE;
      } else if (UUID_COLUMNS.includes(entryDetailColumn)) {
        state.systemLogEntryDetailSearchFilter.operator = Operators.EQUAL;
      } else if (DATE_COLUMNS.includes(entryDetailColumn)) {
        state.systemLogEntryDetailSearchFilter.operator =
          Operators.GREATER_THAN_EQUAL;
        state.systemLogEntryDetailSearchFilter.operator2 =
          Operators.LESS_THAN_EQUAL;
      }
      break;
    }

    default:
      break;
  }
  return state;
};

export default systemLogReducer;
