import { JSONValue } from '@userclouds/sharedui';

// NOTE: try to keep the field order/naming close to Golang definitions in all of these
// types to make it easier to keep them in sync and visually catch bugs/omissions.
export type AuditLogEntry = {
  id: string;
  created: string;
  type: string;
  actor_id: string;
  actor_name: string;
  payload: Record<string, JSONValue>;
};

export const AUDIT_LOG_PREFIX = 'audit_log_';
export const AUDIT_LOG_PAGINATION_LIMIT = '50';
export const AUDIT_LOG_COLUMNS = [
  'actor_id',
  'type',
  'created',
  'payload->SelectorValues',
  'payload->>ID',
];

export type DataAccessLogEntry = {
  id: string;
  created: string;
  actor_id: string;
  actor_name: string;
  accessor_id: string;
  accessor_name: string;
  accessor_version: number;
  columns: string;
  purposes: string;
  masked: string;
  completed: boolean;
  rows: number;
  payload: Record<string, JSONValue>;
};

export const DATA_ACCESS_LOG_PREFIX = 'data_access_log_';
export const DATA_ACCESS_LOG_PAGINATION_LIMIT = '50';
