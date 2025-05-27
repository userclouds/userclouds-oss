import { Given } from '@cucumber/cucumber';

import { CukeWorld } from '../support/world';
import { HOST, PORT } from '../support/globalSetup';
import tenantsMock from '../fixtures/tenants.json';
import usersForOrgMock from '../fixtures/users_for_org_page_1.json';
import { mockRequest } from './helpers';
import { UserProfileSerialized } from '../../src/models/UserProfile';

Given(
  'a mocked request for users across all orgs \\(page 2\\)',
  async function (this: CukeWorld) {
    const payload = JSON.parse(JSON.stringify(usersForOrgMock));
    payload.data = payload.data.reverse();
    payload.has_prev = true;
    payload.prev = 'id:39daeb46-4da5-48b6-b5be-38a3fa90c4b2';
    payload.next = 'id:03f84ae4-0d1a-4c87-b2ec-069606e38bc6';
    const path = `/api/tenants/${tenantsMock[0].id}/users*`;
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
  'a mocked request for users across all orgs \\(page 3\\)',
  async function (this: CukeWorld) {
    const payload = JSON.parse(JSON.stringify(usersForOrgMock));
    payload.data = payload.data.slice(5).concat(payload.data.slice(0, 5));
    payload.has_prev = true;
    payload.has_next = false;
    payload.prev = 'id:220462ec-0a83-41ca-b31c-e829dc8489c1';
    delete payload.next;
    const path = `/api/tenants/${tenantsMock[0].id}/users*`;
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
  'a mocked request for users for the org with ID {string}',
  async function (this: CukeWorld, orgID: string) {
    const payload = JSON.parse(JSON.stringify(usersForOrgMock));
    payload.data = payload.data.filter((user: UserProfileSerialized) => {
      return user.organization_id === orgID;
    });
    payload.has_prev = false;
    payload.has_next = false;
    delete payload.next;
    const path = `/api/tenants/${tenantsMock[0].id}/users*`;
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
  'a mocked request for users for a tenant without orgs',
  async function (this: CukeWorld) {
    const payload = JSON.parse(JSON.stringify(usersForOrgMock));
    payload.data = payload.data.filter((user: UserProfileSerialized) => {
      return user.organization_id === '00000000-0000-0000-0000-000000000000';
    });
    payload.has_prev = false;
    payload.has_next = false;
    delete payload.next;
    const path = `/api/tenants/${tenantsMock[0].id}/users*`;
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
