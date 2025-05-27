import { JSONValue } from '@userclouds/sharedui';
import { NilUuid } from '../models/Uuids';

class ChartQuery {
  tenantID: string;

  service: string;

  eventType: number[];

  start: string;

  end: string;

  period: number;

  /*
      const req = [
      {
        TenantID: NilUuid,
        Service: 'plex',
        EventType: [99, 400, 304, 307, 200],
        Start: '2021-10-02T22:35:40-07:00',
        End: '2021-10-05T13:41:30-07:00',
        Period: 36000000000000,
      },
    ];
  */

  constructor(events: number[]) {
    this.tenantID = NilUuid;
    this.service = '';
    this.eventType = events;
    this.start = '';
    this.end = '';
    this.period = 0;
  }

  static fromJSON(jsonStr: JSONValue): ChartQuery {
    const c = jsonStr as {
      tenant_id: string;
      service: string;
      event_codes: number[];
      start: string;
      end: string;
      period: number;
    };

    return new ChartQuery(c.event_codes);
  }

  toJSON(): JSONValue {
    return {
      tenant_id: this.tenantID,
      service: this.service,
      event_codes: this.eventType,
      start: this.start,
      end: this.end,
      period: this.period,
    };
  }

  getEndTimeStr(): string {
    const endDate = new Date(this.end);
    return endDate.toLocaleString('en-US');
  }

  setTarget(id: string, service: string) {
    this.tenantID = id;
    this.service = service;
  }

  setTimePeriod(selectedTimeperiod: string) {
    // Different period options in nanoseconds
    const minute = 60000000000;
    const hour = minute * 60; // 3600000000000
    const day = hour * 24; // 86400000000000

    // Different start time options in ms
    const dayBack = 86400000;
    const hourBack = 3600000;
    const minuteBack = 60000;

    // Set default values
    const endDate = new Date();
    let startDate = new Date(endDate.getTime() - hourBack);
    this.period = minute * 5;

    switch (selectedTimeperiod) {
      case 'minutes':
        startDate = new Date(endDate.getTime() - 10 * minuteBack);
        this.period = minute;
        break;
      case 'hour':
        startDate = new Date(endDate.getTime() - hourBack);
        this.period = minute * 5;
        break;
      case 'day':
        startDate = new Date(endDate.getTime() - dayBack);
        this.period = hour;
        break;
      case 'week':
        startDate = new Date(endDate.getTime() - 7 * dayBack);
        this.period = day;
        break;
      default:
    }
    this.start = startDate.toJSON();
    this.end = endDate.toJSON();
  }
}

export default ChartQuery;
