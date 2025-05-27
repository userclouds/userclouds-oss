import { JSONValue } from '@userclouds/sharedui';

export type CountQuery = {
  tenantID: string;
  service: string;
  objectIds: string[];
  start: string;
  end: string;
  eventSuffixFilter?: string[];
};

export const countQueryToJSON = (query: CountQuery): JSONValue => {
  return {
    tenant_id: query.tenantID,
    service: query.service,
    object_ids: query.objectIds,
    start: query.start,
    end: query.end,
    event_suffix_filter: query.eventSuffixFilter,
  };
};
