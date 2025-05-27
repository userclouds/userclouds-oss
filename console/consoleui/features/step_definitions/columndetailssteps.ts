import { Given } from '@cucumber/cucumber';
import { JSONValue } from '@userclouds/sharedui';
import { CukeWorld } from '../support/world';
import { HOST, PORT } from '../support/globalSetup';
import { mockRequest } from './helpers';
import columnPurposeDurations from '../fixtures/column_purpose_durations.json';
import {
  ColumnRetentionDurationsResponse,
  PurposeRetentionDuration,
  DurationUnit,
} from '../../src/models/ColumnRetentionDurations';

Given(
  'a mocked request for column retention durations with only two supported units',
  async function (this: CukeWorld) {
    let payload: JSONValue = columnPurposeDurations;
    payload = {
      ...payload,
      supported_retention_durations: ['year', 'month'],
    };
    await mockRequest(
      this,
      {
        url: `${HOST}:${PORT}/api/tenants/*/userstore/columns/retentiondurations/actions/get`,
        method: 'POST',
        status: 200,
        body: payload,
      },
      { times: 1 }
    );
  }
);

Given(
  'a mocked request to save column retention durations with a {int} {string} retention for the {string} purpose',
  async function (
    this: CukeWorld,
    quantity: number,
    unit: string,
    purposeName: string
  ) {
    let payload: ColumnRetentionDurationsResponse =
      columnPurposeDurations as unknown as ColumnRetentionDurationsResponse;
    payload = {
      ...payload,
      purpose_retention_durations: payload.purpose_retention_durations.map(
        (duration: PurposeRetentionDuration) => {
          if (duration.purpose_name === purposeName) {
            return {
              ...duration,
              duration: {
                unit: unit as DurationUnit,
                duration: quantity,
              },
            };
          }
          return duration;
        }
      ),
    };

    await mockRequest(
      this,
      {
        url: `${HOST}:${PORT}/api/tenants/*/userstore/columns/retentiondurations/actions/update`,
        method: 'POST',
        status: 200,
        body: payload as unknown as JSONValue,
      },
      { times: 1 }
    );
  }
);
