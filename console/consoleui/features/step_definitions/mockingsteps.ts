import { promises as fs } from 'fs';
import { join } from 'path';
import { Given } from '@cucumber/cucumber';

import { JSONValue } from '@userclouds/sharedui';

import { hashFlagName } from '../../src/util/featureflags';
import { CukeWorld } from '../support/world';
import { DEBUG_MODE, HOST, DOMAIN, PORT } from '../support/globalSetup';
import serviceInfoMock from '../fixtures/serviceinfo.json';
import serviceInfoAdminMock from '../fixtures/serviceinfoUCadmin.json';
import userInfoMock from '../fixtures/userinfo.json';
import companiesMock from '../fixtures/companies.json';
import tenantsMock from '../fixtures/tenants.json';
import selectedTenantMock from '../fixtures/tenant_selected.json';
import selectedTenant2Mock from '../fixtures/tenant_selected2.json';
import nonAdminSelectedTenantMock from '../fixtures/tenant_selected_nonadmin.json';
import tenantsURLsMock from '../fixtures/tenants_urls.json';
import createCompanyMock from '../fixtures/create_company.json';
import createTenantMock from '../fixtures/create_tenant.json';
import updatedColumnMock from '../fixtures/updated_column.json';
import accessorsMock from '../fixtures/accessors.json';
import accessorMetricsMock from '../fixtures/accessor_metrics.json';
import accessorDetailsMock from '../fixtures/accessor_details.json';
import accessorDetailsTokenResMock from '../fixtures/accessor_details_token_res.json';
import updatedAccessorDetailsMock from '../fixtures/updated_accessor_details.json';
import mutatorsMock from '../fixtures/mutators.json';
import mutatorDetailsMock from '../fixtures/mutator_details.json';
import updatedMutatorDetailsMock from '../fixtures/updated_mutator_details.json';
import purposesMock from '../fixtures/purposes.json';
import userStoreSchemaMock from '../fixtures/userstoreschema_default.json';
import userStoreSchemaEditMock from '../fixtures/userstoreschema_edit.json';
import userDetailsMock from '../fixtures/user_details.json';
import userConsentedPurposesMock from '../fixtures/user_consented_purposes.json';
import userEventsMock from '../fixtures/user_events.json';
import accessPoliciesMock from '../fixtures/access_policies.json';
import accessPolicyMock from '../fixtures/access_policy.json';
import accessPolicyTemplatesMock from '../fixtures/access_policy_templates.json';
import transformersMock from '../fixtures/transformers.json';
import plexConfigMock from '../fixtures/plexconfig.json';
import plexConfigCustomOIDCMock from '../fixtures/plexconfig_custom_oidc.json';
import keysMock from '../fixtures/keys.json';
import dataTypesMock from '../fixtures/data_types.json';
import databaseMock from '../fixtures/data_base.json';

import { mockRequest, RequestMethod } from './helpers';

type MockRequests = {
  [alias: string]: {
    path: string;
    mocks: Record<string, JSONValue>;
  };
};

const mockRequests: MockRequests = {
  tenants: {
    path: `/api/companies/${companiesMock[0].id}/tenants`,
    mocks: {
      GET: tenantsMock,
      POST: createTenantMock,
    },
  },
  selectedTenant: {
    path: `/api/companies/${companiesMock[0].id}/tenants/${tenantsMock[0].id}`,
    mocks: {
      GET: selectedTenantMock,
    },
  },
  selectedTenantNonAdmin: {
    path: `/api/companies/${companiesMock[0].id}/tenants/${tenantsMock[0].id}`,
    mocks: {
      GET: nonAdminSelectedTenantMock,
    },
  },
  selectedTenant2: {
    path: `/api/companies/${companiesMock[0].id}/tenants/${tenantsMock[1].id}`,
    mocks: {
      GET: selectedTenant2Mock,
    },
  },
  tenants_urls: {
    path: `/api/tenants/${tenantsMock[0].id}/urls`,
    mocks: {
      GET: tenantsURLsMock,
    },
  },
  'a list containing one tenant': {
    path: `/api/companies/${companiesMock[0].id}/tenants`,
    mocks: {
      GET: [{ ...tenantsMock[0], name: 'Foo' }],
    },
  },
  'selected new tenant': {
    path: `/api/companies/${companiesMock[0].id}/tenants/${tenantsMock[0].id}`,
    mocks: {
      GET: { ...selectedTenantMock, name: 'Foo' },
    },
  },
  'a single tenant being created': {
    path: `/api/companies/${companiesMock[0].id}/tenants/*`,
    mocks: {
      GET: { ...selectedTenantMock, name: 'Foo', state: 'creating' },
    },
  },
  'a single tenant': {
    path: `/api/companies/${companiesMock[0].id}/tenants/*`,
    mocks: {
      GET: { ...selectedTenantMock, name: 'Foo', state: 'active' },
    },
  },
  permissions: {
    path: `/api/tenants/${tenantsMock[0].id}/policies/permissions`,
    mocks: {
      GET: { create: true, read: true, update: true, delete: true },
    },
  },
  permissions_policy: {
    path: `/api/tenants/${tenantsMock[0].id}/policies/permissions/*`,
    mocks: {
      GET: { create: true, read: true, update: true, delete: true },
    },
  },
  serviceinfo: {
    path: '/api/serviceinfo',
    mocks: {
      GET: serviceInfoMock,
    },
  },
  'UC admin service info': {
    path: '/api/serviceinfo',
    mocks: {
      GET: serviceInfoMock,
    },
  },
  columns: {
    path: `/api/tenants/${tenantsMock[0].id}/userstore/columns*`,
    mocks: {
      POST: (() => {
        const data: JSONValue = { ...updatedColumnMock };
        delete data.version;
        return data;
      })(),
    },
  },
  accessors: {
    path: `/api/tenants/${tenantsMock[0].id}/userstore/accessors*`,
    mocks: {
      GET: accessorsMock,
      POST: (() => {
        const data: JSONValue = { ...updatedAccessorDetailsMock };
        delete data.version;
        return data;
      })(),
    },
  },
  accessorsMetrics: {
    path: `/api/tenants/${tenantsMock[0].id}/counters/query*`,
    mocks: {
      GET: accessorMetricsMock,
    },
  },
  mutators: {
    path: `/api/tenants/${tenantsMock[0].id}/userstore/mutators*`,
    mocks: {
      GET: mutatorsMock,
      POST: (() => {
        const data: JSONValue = { ...updatedMutatorDetailsMock };
        delete data.version;
        return data;
      })(),
    },
  },
  purposes: {
    path: `/api/tenants/${tenantsMock[0].id}/userstore/purposes*`,
    mocks: {
      GET: purposesMock,
    },
  },
  'accessor details': {
    path: `/api/tenants/${tenantsMock[0].id}/userstore/accessors/${accessorDetailsMock.id}*`,
    mocks: {
      GET: accessorDetailsMock,
    },
  },
  'updated accessor details': {
    // on the GET, we're going to have ?version=n but no querystring on the PUT
    path: `/api/tenants/${tenantsMock[0].id}/userstore/accessors/${updatedAccessorDetailsMock.id}*`,
    mocks: {
      GET: updatedAccessorDetailsMock,
      PUT: updatedAccessorDetailsMock,
    },
  },
  'token res accessor details': {
    path: `/api/tenants/${tenantsMock[0].id}/userstore/accessors/${accessorDetailsTokenResMock.id}*`,
    mocks: {
      GET: accessorDetailsTokenResMock,
    },
  },
  'created accessor details': {
    path: `/api/tenants/${tenantsMock[0].id}/userstore/accessors/${accessorDetailsMock.id}`,
    mocks: {
      GET: {
        ...accessorDetailsMock,
        name: 'Our_Accessor',
        description: 'foo',
        version: 0,
      },
    },
  },
  'mutator details': {
    path: `/api/tenants/${tenantsMock[0].id}/userstore/mutators/${mutatorDetailsMock.id}`,
    mocks: {
      GET: mutatorDetailsMock,
    },
  },
  'updated mutator details': {
    // on the GET, we're going to have ?version=n but no querystring on the PUT
    path: `/api/tenants/${tenantsMock[0].id}/userstore/mutators/${updatedMutatorDetailsMock.id}*`,
    mocks: {
      GET: updatedMutatorDetailsMock,
      PUT: updatedMutatorDetailsMock,
    },
  },
  'created mutator details': {
    path: `/api/tenants/${tenantsMock[0].id}/userstore/mutators/${mutatorDetailsMock.id}`,
    mocks: {
      GET: {
        ...mutatorDetailsMock,
        name: 'Our_Mutator',
        description: 'foo',
        version: 0,
      },
    },
  },
  'user store schema': {
    path: `/api/tenants/${tenantsMock[0].id}/userstore/columns*`,
    mocks: {
      GET: userStoreSchemaMock,
    },
  },
  'user store schema edit': {
    path: `/api/tenants/${tenantsMock[0].id}/userstore/columns*`,
    mocks: {
      GET: userStoreSchemaEditMock,
    },
  },
  'user details': {
    path: `/api/tenants/${tenantsMock[0].id}/users/${userDetailsMock.id}`,
    mocks: {
      GET: userDetailsMock,
    },
  },
  'user consented purposes': {
    path: `/api/tenants/${tenantsMock[0].id}/consentedpurposesforuser/${userDetailsMock.id}`,
    mocks: {
      GET: userConsentedPurposesMock,
    },
  },
  'user events': {
    path: `/api/tenants/${tenantsMock[0].id}/userevents?user_alias=${userDetailsMock.id}`,
    mocks: {
      GET: userEventsMock,
    },
  },
  userinfo: {
    path: '/auth/userinfo',
    mocks: {
      GET: userInfoMock,
    },
  },
  companies: {
    path: '/api/companies',
    mocks: {
      GET: companiesMock,
      POST: createCompanyMock,
    },
  },
  access_policies: {
    path: `/api/tenants/${tenantsMock[0].id}/policies/access?*`,
    mocks: {
      GET: accessPoliciesMock,
    },
  },
  access_policy: {
    path: `/api/tenants/${tenantsMock[0].id}/policies/access/*`,
    mocks: {
      GET: accessPolicyMock,
    },
  },
  access_policy_templates: {
    path: `/api/tenants/${tenantsMock[0].id}/policies/templates*`,
    mocks: {
      GET: accessPolicyTemplatesMock,
    },
  },
  transformers: {
    path: `/api/tenants/${tenantsMock[0].id}/policies/transformation*`,
    mocks: {
      GET: transformersMock,
    },
  },
  plex_config: {
    path: `/api/tenants/${tenantsMock[0].id}/plexconfig`,
    mocks: {
      GET: plexConfigMock,
      POST: plexConfigMock,
    },
  },
  plex_config_custom_oidc: {
    path: `/api/tenants/${tenantsMock[0].id}/plexconfig`,
    mocks: {
      GET: plexConfigCustomOIDCMock,
    },
  },
  keys: {
    path: `/api/tenants/${tenantsMock[0].id}/keys`,
    mocks: {
      GET: keysMock,
    },
  },
  dataTypes: {
    path: `/api/tenants/${tenantsMock[0].id}/userstore/datatypes*`,
    mocks: {
      GET: dataTypesMock,
    },
  },
  database: {
    path: `/api/tenants/${tenantsMock[0].id}/userstore/databases*`,
    mocks: {
      GET: databaseMock,
    },
  },
};

Given('I am an unauthenticated user', async function (this: CukeWorld) {
  await mockRequest(
    this,
    {
      url: `${HOST}:${PORT}/api/serviceinfo`,
      status: 401,
      body: 'An unspecified error occurred',
    },
    { times: Infinity }
  );
  await mockRequest(
    this,
    {
      url: `${HOST}:${PORT}/auth/userinfo`,
      status: 401,
      body: 'An unspecified error occurred',
    },
    { times: Infinity }
  );
  await mockRequest(
    this,
    {
      url: `${HOST}:${PORT}/api/companies`,
      status: 401,
      body: 'An unspecified error occurred',
    },
    { times: Infinity }
  );
});

Given(
  /^a mocked "([A-Z]+)" request for "([\s\w\t]+)"(?: that returns a "([0-9]{3})")?$/,
  async function (
    this: CukeWorld,
    method: RequestMethod,
    alias: string,
    statusCode?: string
  ) {
    // for some reason, making statusCode a number in the param list doesn't actually
    // make it a number when passed to the Request object below, which throws off something
    const status = parseInt(statusCode || '200', 10);
    await mockRequest(
      this,
      {
        url: `${HOST}:${PORT}${mockRequests[alias]?.path}`,
        method,
        body: mockRequests[alias]?.mocks[method],
        status: status,
      },
      { times: 1 }
    );
  }
);

Given('the following feature flags', async function (this: CukeWorld, flags) {
  // the "Given I am a logged-in user steps mock an empty feature flags response
  // let's unmock it and make sure we forget it for the purposes of tracking unfulfilled mocks
  await this.page.unroute('https://featuregates.org/v1/initialize');
  this.activeMocks = this.activeMocks.filter(
    (mock) => mock.url !== 'https://featuregates.org/v1/initialize'
  );

  const payload: any = {
    feature_gates: {
      '3667909994': {
        name: '3667909994',
        value: false,
        rule_id: 'default',
        group_name: 'default',
        id_type: 'userID',
      },
    },
  };
  for (const [flagName, value] of flags.rows()) {
    const hashedName = hashFlagName(flagName);
    payload.feature_gates[hashedName] = {
      name: hashedName,
      value: value === 'true',
      rule_id: 'anything goes',
      group_name: "doesn't matter",
      id_type: 'userID',
    };
  }
  await mockRequest(
    this,
    {
      url: 'https://featuregates.org/v1/initialize',
      method: 'POST',
      status: 200,
      body: payload,
    },
    { times: 1 }
  );
});

/* Example */
/**
Given the following mocked requests:
  | Method | Path                                                                                                     | Status | Body                       |
  | POST   | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/columns                                      | 200    | {}                         |
  | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/columns/032cae17-df3a-4e87-82a0-c706ed0679ee | 200    | {}                         |
  | PUT    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/columns/62fcf8b4-48d0-46d9-9f5b-d0813a478a2b | 200    | {}                         |
  | DELETE | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstore/columns/83cc42b0-da8c-4a61-9db1-da70f21bab60 | 200    | {}                         |
  | GET    | /api/tenants/41ab79a8-0dff-418e-9d42-e1694469120a/userstoreschema                                        | 200    | userstoreschema_edit2.json |
* */
/*
 * In the last row, 'userstoreschema_edit2.json' is a filename or path within features/fixtures/
 */
Given(
  'the following mocked requests:',
  async function (this: CukeWorld, requestTable) {
    for (const [
      method,
      path,
      status,
      body,
      modifications,
    ] of requestTable.rows()) {
      let payload: JSONValue;
      if (body.indexOf('.json') > -1) {
        try {
          const file = await fs.readFile(
            join(__dirname, `../fixtures/${body}`),
            { encoding: 'utf8' }
          );
          payload = JSON.parse(file);
          if (modifications) {
            payload = ((response: JSONValue) => {
              // eslint-disable-next-line no-eval
              eval(modifications);
              return response;
            })(payload);
            console.log(payload);
          }
        } catch (e) {
          console.error('Could not read mock file', e);
          return;
        }
      } else {
        payload = JSON.parse(body);
      }
      mockRequest(
        this,
        {
          url: `${HOST}:${PORT}${path}`,
          method: method || 'GET',
          status: parseInt(status, 10) || 200,
          body: payload !== undefined ? payload : {},
        },
        { times: 1 }
      );
    }
  }
);

Given('I am a logged-in user', async function (this: CukeWorld) {
  const thirtyDaysFromNow = Math.floor(Date.now() / 1000) + 30 * 86400;
  // we use this for fetching my profile (/userinfo)
  await this.browserContext.addCookies([
    {
      name: 'auth-session-id',
      value: '10c2beeb-97dd-4de7-ab19-5882d7af3f0c',
      domain: DOMAIN,
      path: '/',
      expires: thirtyDaysFromNow,
      httpOnly: false,
      secure: false,
      sameSite: 'Lax',
    },
  ]);
  // the UI just cares if these reqs 401
  // it doesn't care about the session cookie or the JWT
  await mockRequest(
    this,
    {
      url: `${HOST}:${PORT}/api/serviceinfo**`,
      body: serviceInfoMock,
    },
    { times: Infinity }
  );
  await mockRequest(
    this,
    {
      url: `${HOST}:${PORT}/auth/userinfo**`,
      body: userInfoMock,
    },
    { times: Infinity }
  );
  await mockRequest(
    this,
    {
      url: `${HOST}:${PORT}/api/companies**`,
      body: companiesMock,
    },
    { times: Infinity }
  );
  await mockRequest(
    this,
    {
      url: 'https://featuregates.org/v1/initialize',
      method: 'POST',
      status: 200,
      body: { feature_gates: {} },
    },
    { times: 1 }
  );
});

Given(
  'a mocked response for no enabled feature flags',
  async function (this: CukeWorld) {
    await mockRequest(
      this,
      {
        url: 'https://featuregates.org/v1/initialize',
        method: 'POST',
        status: 200,
        body: { feature_gates: {} },
      },
      { times: 1 }
    );
  }
);

Given('I am a logged-in UC admin', async function (this: CukeWorld) {
  // the UI just cares if these reqs 401
  // it doesn't care about the session cookie or the JWT
  await mockRequest(
    this,
    {
      url: `${HOST}:${PORT}/api/serviceinfo**`,
      body: serviceInfoAdminMock,
    },
    { times: Infinity }
  );
  await mockRequest(
    this,
    {
      url: `${HOST}:${PORT}/auth/userinfo**`,
      body: userInfoMock,
    },
    { times: Infinity }
  );
  await mockRequest(
    this,
    {
      url: `${HOST}:${PORT}/api/companies**`,
      body: companiesMock,
    },
    { times: Infinity }
  );
  await mockRequest(
    this,
    {
      url: 'https://featuregates.org/v1/initialize',
      method: 'POST',
      status: 200,
      body: { feature_gates: {} },
    },
    { times: 1 }
  );
});

Given(
  'a mocked request for the auth redirect page',
  async function (this: CukeWorld) {
    const response = {
      contentType: 'text/html',
      body: '<html></html>',
      status: 200,
    };
    await this.page.route(
      'http://console.dev.userclouds.tools:3057/auth/redirect**',
      (route) => {
        if (DEBUG_MODE) {
          console.log(
            Date.now(),
            `Found a matching mock for this request. Responding to ${route
              .request()
              .method()} request with status ${
              response.status
            } and the following body:`,
            response.body
          );
        }
        // introduce a slight delay to simulate real conditions
        setTimeout(() => {
          route.fulfill(response);
        }, 200);
      },
      { times: 1 }
    );
  }
);
