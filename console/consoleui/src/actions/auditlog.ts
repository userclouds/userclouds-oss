import { APIError } from '@userclouds/sharedui';
import { AuditLogEntry, DataAccessLogEntry } from '../models/AuditLogEntry';
import { Filter } from '../models/authz/SearchFilters';
import PaginatedResult from '../models/PaginatedResult';

export const GET_AUDIT_LOG_ENTRIES_REQUEST = 'GET_AUDIT_LOG_ENTRIES_REQUEST';
export const GET_AUDIT_LOG_ENTRIES_SUCCESS = 'GET_AUDIT_LOG_ENTRIES_SUCCESS';
export const GET_AUDIT_LOG_ENTRIES_ERROR = 'GET_AUDIT_LOG_ENTRIES_ERROR';
export const CHANGE_CURRENT_AUDIT_LOG_SEARCH_FILTER =
  'CHANGE_CURRENT_AUDIT_LOG_SEARCH_FILTER';
export const GET_DATA_ACCESS_LOG_ENTRIES_REQUEST =
  'GET_DATA_ACCESS_LOG_ENTRIES_REQUEST';
export const GET_DATA_ACCESS_LOG_ENTRIES_SUCCESS =
  'GET_DATA_ACCESS_LOG_ENTRIES_SUCCESS';
export const GET_DATA_ACCESS_LOG_ENTRIES_ERROR =
  'GET_DATA_ACCESS_LOG_ENTRIES_ERROR';
export const CHANGE_DATA_ACCESS_LOG_FILTER = 'CHANGE_DATA_ACCESS_LOG_FILTER';
export const GET_DATA_ACCESS_LOG_ENTRY_REQUEST =
  'GET_DATA_ACCESS_LOG_ENTRY_REQUEST';
export const GET_DATA_ACCESS_LOG_ENTRY_SUCCESS =
  'GET_DATA_ACCESS_LOG_ENTRY_SUCCESS';
export const GET_DATA_ACCESS_LOG_ENTRY_ERROR =
  'GET_DATA_ACCESS_LOG_ENTRY_ERROR';

export const getAuditLogEntriesRequest = () => ({
  type: GET_AUDIT_LOG_ENTRIES_REQUEST,
});
export const getAuditLogEntriesSuccess = (
  entries: PaginatedResult<AuditLogEntry>
) => ({
  type: GET_AUDIT_LOG_ENTRIES_SUCCESS,
  data: entries,
});
export const getAuditLogEntriesRequestError = (error: APIError) => ({
  type: GET_AUDIT_LOG_ENTRIES_ERROR,
  data: error.message,
});

export const changeCurrentAuditLogSearchFilter = (filter: Filter) => ({
  type: CHANGE_CURRENT_AUDIT_LOG_SEARCH_FILTER,
  data: filter,
});

export const getDataAccessLogEntriesRequest = () => ({
  type: GET_DATA_ACCESS_LOG_ENTRIES_REQUEST,
});
export const getDataAccessLogEntriesSuccess = (
  entries: PaginatedResult<DataAccessLogEntry>
) => ({
  type: GET_DATA_ACCESS_LOG_ENTRIES_SUCCESS,
  data: entries,
});
export const getDataAccessLogEntriesError = (error: APIError) => ({
  type: GET_DATA_ACCESS_LOG_ENTRIES_ERROR,
  data: error.message,
});

export const changeDataAccessLogFilter = (filter: object) => ({
  type: CHANGE_DATA_ACCESS_LOG_FILTER,
  data: filter,
});

export const getDataAccessLogEntryRequest = () => ({
  type: GET_DATA_ACCESS_LOG_ENTRY_REQUEST,
});
export const getDataAccessLogEntrySuccess = (entry: DataAccessLogEntry) => ({
  type: GET_DATA_ACCESS_LOG_ENTRY_SUCCESS,
  data: entry,
});
export const getDataAccessLogEntryError = (error: APIError) => ({
  type: GET_DATA_ACCESS_LOG_ENTRY_ERROR,
  data: error.message,
});
