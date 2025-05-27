import { APIError } from '@userclouds/sharedui';
import { Filter } from '../models/authz/SearchFilters';
import PaginatedResult from '../models/PaginatedResult';
import { SystemLogEntry, SystemLogEntryRecord } from '../models/SystemLogEntry';

// System log
export const GET_SYSTEM_LOG_ENTRIES_REQUEST = 'GET_SYSTEM_LOG_ENTRIES_REQUEST';
export const GET_SYSTEM_LOG_ENTRIES_SUCCESS = 'GET_SYSTEM_LOG_ENTRIES_SUCCESS';
export const GET_SYSTEM_LOG_ENTRIES_ERROR = 'GET_SYSTEM_LOG_ENTRIES_ERROR';
export const CHANGE_CURRENT_SYSTEM_LOG_SEARCH_FILTER =
  'CHANGE_CURRENT_SYSTEM_LOG_SEARCH_FILTER';

export const GET_SINGLE_SYSTEM_LOG_ENTRY_DETAIL_REQUEST =
  'GET_SINGLE_SYSTEM_LOG_ENTRY_DETAIL__REQUEST';
export const GET_SINGLE_SYSTEM_LOG_ENTRY_DETAIL_SUCCESS =
  'GET_SINGLE_SYSTEM_LOG_ENTRY_DETAIL_SUCCESS';
export const GET_SINGLE_SYSTEM_LOG_ENTRY_DETAIL_ERROR =
  'GET_SINGLE_SYSTEM_LOG_ENTRY_DETAIL__ERROR';

export const GET_SYSTEM_LOG_ENTRY_DETAIL_REQUEST =
  'GET_SYSTEM_LOG_ENTRY_DETAIL__REQUEST';
export const GET_SYSTEM_LOG_ENTRY_DETAIL_SUCCESS =
  'GET_SYSTEM_LOG_ENTRY_DETAIL__SUCCESS';
export const GET_SYSTEM_LOG_ENTRY_DETAIL_ERROR =
  'GET_SYSTEM_LOG_ENTRY_DETAIL__ERROR';
export const CHANGE_CURRENT_SYSTEM_LOG_ENTRY_DETAIL_SEARCH_FILTER =
  'CHANGE_CURRENT_SYSTEM_LOG_ENTRY_DETAIL_SEARCH_FILTER';

export const getSystemLogEntriesRequest = () => ({
  type: GET_SYSTEM_LOG_ENTRIES_REQUEST,
});
export const getSystemLogEntriesSuccess = (
  entries: PaginatedResult<SystemLogEntry>
) => ({
  type: GET_SYSTEM_LOG_ENTRIES_SUCCESS,
  data: entries,
});
export const getSystemLogEntriesRequestError = (error: APIError) => ({
  type: GET_SYSTEM_LOG_ENTRIES_ERROR,
  data: error,
});

export const changeCurrentSystemLogSearchFilter = (filter: Filter) => ({
  type: CHANGE_CURRENT_SYSTEM_LOG_SEARCH_FILTER,
  data: filter,
});

export const getSingleSystemLogEntryDetailRequest = () => ({
  type: GET_SINGLE_SYSTEM_LOG_ENTRY_DETAIL_REQUEST,
});

export const getSingleSystemLogEntryDetailSuccess = (
  entry: PaginatedResult<SystemLogEntry>
) => ({
  type: GET_SINGLE_SYSTEM_LOG_ENTRY_DETAIL_SUCCESS,
  data: entry,
});
export const getSingleSystemLogEntryDetailRequestError = (error: string) => ({
  type: GET_SINGLE_SYSTEM_LOG_ENTRY_DETAIL_ERROR,
  data: error,
});

export const getSystemLogEntryDetailRequest = () => ({
  type: GET_SYSTEM_LOG_ENTRY_DETAIL_REQUEST,
});
export const getSystemLogEntryDetailSuccess = (
  entries: PaginatedResult<SystemLogEntryRecord>
) => ({
  type: GET_SYSTEM_LOG_ENTRY_DETAIL_SUCCESS,
  data: entries,
});
export const getSystemLogEntryDetailRequestError = (error: string) => ({
  type: GET_SYSTEM_LOG_ENTRY_DETAIL_ERROR,
  data: error,
});

export const changeCurrentSystemLogEntryDetailSearchFilter = (
  filter: Filter
) => ({
  type: CHANGE_CURRENT_SYSTEM_LOG_ENTRY_DETAIL_SEARCH_FILTER,
  data: filter,
});
