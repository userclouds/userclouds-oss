import { Given } from '@cucumber/cucumber';

import { JSONValue } from '@userclouds/sharedui';

import { CukeWorld } from '../support/world';
import { DEBUG_MODE, PORT, HOST } from '../support/globalSetup';
import tenantsMock from '../fixtures/tenants.json';
import accessorsMock from '../fixtures/accessors.json';
import updatedAccessorDetailsMock from '../fixtures/updated_accessor_details.json';
import userStoreSchemaEditSystemColumnsMock from '../fixtures/userstoreschema_edit_system_columns.json';
import mutatorDetailsMock from '../fixtures/mutator_details.json';
import { mockRequest } from './helpers';

Given(
  /^a mocked request to save an? (accessor|mutator) with ([0-9]+) columns?$/,
  async function (this: CukeWorld, modelType: string, IDs: string) {
    const numberOfColumns = parseInt(IDs, 10);
    const mock =
      modelType === 'accessor'
        ? updatedAccessorDetailsMock
        : mutatorDetailsMock;
    await this.page.route(
      `${HOST}:${PORT}/api/tenants/${
        tenantsMock[0].id
      }/userstore/${modelType}s/${modelType === 'mutator' ? 'columns/' : ''}${
        mock.id
      }`,
      (route) => {
        if (route.request().method() === 'PUT') {
          const requestBody = JSON.parse(route.request().postData() || '');
          const responseBody: JSONValue = { ...mock };
          delete responseBody.access_policy;
          delete responseBody.transformer;
          delete responseBody.normalizer;
          responseBody.access_policy_id = mock.access_policy.id;
          const response = {
            status: 200,
            body: JSON.stringify(responseBody),
          };
          if (requestBody.length === numberOfColumns) {
            if (DEBUG_MODE) {
              console.log(
                `Preparing to respond to PUT request with status 200`,
                response.body
              );
            }
            // introduce a slight delay to simulate real conditions
            setTimeout(() => {
              route.fulfill(response);
            }, 200);
          }
        }
      },
      { times: 1 }
    );
  }
);

Given(
  'a mocked request for all accessors associated with a column',
  async function (this: CukeWorld) {
    const payload = JSON.parse(JSON.stringify(accessorsMock));
    payload.data[0].columns[0].name = 'email_verified';
    payload.data[1].columns[1].name = 'email_verified';
    const path =
      '/api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/accessors?limit=50&filter=%28%27column_ids%27%2CHAS%2C%27032cae17-df3a-4e87-82a0-c706ed0679ee%27%29&version=3';
    await mockRequest(
      this,
      {
        url: `${HOST}:${PORT}${path}`,
        method: 'GET',
        status: 200,
        body: payload,
      },
      { times: 1 }
    );
  }
);
Given(
  'a request for user store schema edit system columns',
  async function (this: CukeWorld) {
    const payload = JSON.parse(
      JSON.stringify(userStoreSchemaEditSystemColumnsMock)
    );
    const path = `/api/tenants/${tenantsMock[0].id}/userstore/columns*`;
    await mockRequest(
      this,
      {
        url: `${HOST}:${PORT}${path}`,
        method: 'GET',
        status: 200,
        body: payload,
      },
      { times: 1 }
    );
  }
);
