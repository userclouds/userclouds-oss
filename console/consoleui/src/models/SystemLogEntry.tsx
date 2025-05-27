// NOTE: try to keep the field order/naming close to Golang definitions in all of these
// types to make it easier to keep them in sync and visually catch bugs/omissions.
export type SystemLogEntry = {
  id: string;
  created: string;
  updated: string;
  deleted: string;
  Type: string;
  ActiveProviderID: string;
  FollowerProviderIDs: string;
  Since: string;
  Until: string;
  Error: string;
  TotalRecords: number;
  FailedRecords: number;
  WarningRecords: number;
};

export type SystemLogEntryRecord = {
  id: string;
  created: string;
  updated: string;
  deleted: string;
  SyncRunID: string;
  ObjectID: string;
  Error: string;
  Warning: string;
  UserID: string;
};
