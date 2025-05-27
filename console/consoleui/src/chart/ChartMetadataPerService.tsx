import { RequestChartsMetadata } from '../models/Chart';

export const chartLabels = new Map([
  // HTTP
  ['99', 'Total Requests'],
  ['200', '200'],
  ['201', '201'],
  ['204', '204'],
  ['300', '300'],
  ['302', '302'],
  ['304', '304'],
  ['307', '307'],
  ['400', '400'],
  ['403', '403'],
  ['404', '404'],
  ['409', '409'],
  ['500', '500'],
  ['501', '501'],
  ['502', '502'],
  ['503', '503'],
  ['504', '504'],

  // Plex
  ['1002', 'U/P Login'],
  ['2240', 'Social Login'],
  ['1010', 'Logout'],
  ['1003', 'Session Error'],
  ['1004', 'Provider Error'],
  ['1005', 'Input Error'],
  ['1006', 'MFA Error'],
  ['1035', 'Create Account Calls'],
  ['1036', 'Bad Input'],
  ['1038', 'Provider Error'],
  ['1045', 'Authorize Calls'],
  ['1055', 'Token Calls'],

  // Authz
  ['1890', 'List Object Types'],
  ['2450', 'Get Object Type'],
  ['1950', 'List Objects'],
  ['1960', 'Get Object'],
  ['2000', 'Get Edge'],
  ['1910', 'List Edge Types'],
  ['1920', 'Get Edge Type'],
  ['1990', 'List Edges'],
  ['1900', 'Create Object Type'],
  ['1930', 'Create Edge Type'],
  ['1970', 'Create Object'],
  ['2010', 'Create Edge'],
  ['1980', 'Delete Object'],
  ['2020', 'Delete Edge'],

  // IDP
  ['1620', 'Logins Password'],
  ['1650', 'Create Account'],
  ['1680', 'Migrate Account'],
  ['1640', 'Get UserInfo'],
  ['1630', 'Update User'],
  ['2110', 'Get User'],
  ['2100', 'Get Users'],

  // Console
  ['1700', 'Login'],
  ['1710', 'Logout'],
  ['1750', 'Creates'],
  ['1760', 'Deletes'],
  ['1860', 'Add Members'],
  ['1870', 'Remove Members'],
  ['1800', 'Create'],
  ['1810', 'Delete'],

  // Tokenizer
  ['2610', 'Access Policy Create'],
  ['2600', 'Access Policy Get'],
  ['2620', 'Access Policy Update'],
  ['2630', 'Access Policy Delete'],
  ['2660', 'Transformer Create'],
  ['2650', 'Transformer Get'],
  ['2670', 'Transformer Delete'],
  ['2570', 'Create'],
  ['2580', 'Delete'],
  ['2830', 'Resolution'],
  ['2820', 'Lookup'],
  ['2810', 'Inspect'],
]);

const serviceChartMetadata: RequestChartsMetadata = {
  plex: {
    service: 'plex',
    charts: [
      {
        title: 'HTTP Return Codes For All Requests',
        divId: 'containerHttp',
        querySets: [
          {
            name: 'Success',
            eventTypes: [99, 200, 201, 204],
          },
          {
            name: 'Redirect',
            eventTypes: [300, 302, 304, 307],
          },
          {
            name: 'Client Error',
            eventTypes: [400, 403, 404, 409],
          },
          {
            name: 'Server Error',
            eventTypes: [500, 501, 502, 503, 504],
          },
        ],
      },
      {
        title: 'Logins and Logouts',
        divId: 'containerRequests',
        querySets: [
          {
            name: 'Login and Logout',
            eventTypes: [1002, 2240, 1010],
          },
          {
            name: 'Errors',
            eventTypes: [1003, 1004, 1005, 1006],
          },
        ],
      },
      {
        title: 'Account Creation',
        divId: 'containerAccCreation',
        querySets: [
          {
            name: 'Create Account',
            eventTypes: [1035],
          },
          {
            name: 'Errors',
            eventTypes: [1036, 1038],
          },
        ],
      },
      {
        title: 'Authorize and Token Generation Calls',
        divId: 'containerAuthAndToken',
        querySets: [
          {
            name: 'Authorize and Token Calls',
            eventTypes: [1045, 1055],
          },
        ],
      },
    ],
  },
  authz: {
    service: 'authz',
    charts: [
      {
        title: 'HTTP Return Codes For All Requests',
        divId: 'containerHttp',
        querySets: [
          {
            name: 'Success',
            eventTypes: [99, 200, 201, 204],
          },
          {
            name: 'Redirect',
            eventTypes: [300, 302, 304, 307],
          },
          {
            name: 'Client Error',
            eventTypes: [400, 403, 404, 409],
          },
          {
            name: 'Server Error',
            eventTypes: [500, 501, 502, 503, 504],
          },
        ],
      },
      {
        title: 'Creates',
        divId: 'containerRequests',
        querySets: [
          {
            name: 'Create Types',
            eventTypes: [1900, 1930, 1970, 2010],
          },
        ],
      },
      {
        title: 'Queries',
        divId: 'containerAccCreation',
        querySets: [
          {
            name: 'Object Types',
            eventTypes: [1890, 2450, 1950, 1960],
          },
          {
            name: 'Edge Types',
            eventTypes: [1910, 2000, 1920, 1990],
          },
        ],
      },
      {
        title: 'Deletes',
        divId: 'containerAuthAndToken',
        querySets: [
          {
            name: 'Object',
            eventTypes: [1980, 2020],
          },
        ],
      },
    ],
  },
  idp: {
    service: 'idp',
    charts: [
      {
        title: 'HTTP Return Codes For All Requests',
        divId: 'containerHttp',
        querySets: [
          {
            name: 'Success',
            eventTypes: [99, 200, 201, 204],
          },
          {
            name: 'Redirect',
            eventTypes: [300, 302, 304, 307],
          },
          {
            name: 'Client Error',
            eventTypes: [400, 403, 404, 409],
          },
          {
            name: 'Server Error',
            eventTypes: [500, 501, 502, 503, 504],
          },
        ],
      },
      {
        title: 'Logins',
        divId: 'containerRequests',
        querySets: [
          {
            name: 'Logins Password',
            eventTypes: [1620],
          },
        ],
      },
      {
        title: 'Account Creation',
        divId: 'containerAccCreation',
        querySets: [
          {
            name: 'Account Calls',
            eventTypes: [1650, 1680],
          },
        ],
      },
      {
        title: 'Queries',
        divId: 'containerAuthAndToken',
        querySets: [
          {
            name: 'User Info',
            eventTypes: [1640, 1630, 2110, 2100],
          },
        ],
      },
    ],
  },
  console: {
    service: 'console',
    charts: [
      {
        title: 'HTTP Return Codes For All Requests',
        divId: 'containerHttp',
        querySets: [
          {
            name: 'Success',
            eventTypes: [99, 200, 201, 204],
          },
          {
            name: 'Redirect',
            eventTypes: [300, 302, 304, 307],
          },
          {
            name: 'Client Error',
            eventTypes: [400, 403, 404, 409],
          },
          {
            name: 'Server Error',
            eventTypes: [500, 501, 502, 503, 504],
          },
        ],
      },
      {
        title: 'Console Activity',
        divId: 'containerRequests',
        querySets: [
          {
            name: 'Login and Logout',
            eventTypes: [1700, 1710],
          },
        ],
      },
      {
        title: 'Companies',
        divId: 'containerAccCreation',
        querySets: [
          {
            name: 'Creates and Deletes',
            eventTypes: [1750, 1760, 1860, 1870],
          },
        ],
      },
      {
        title: 'Tenants',
        divId: 'containerAuthAndToken',
        querySets: [
          {
            name: 'Creates and Deletes',
            eventTypes: [1800, 1810],
          },
        ],
      },
    ],
  },
  tokenizer: {
    service: 'tokenizer',
    charts: [
      {
        title: 'HTTP Return Codes For All Requests',
        divId: 'containerHttp',
        querySets: [
          {
            name: 'Success',
            eventTypes: [99, 200, 201, 204],
          },
          {
            name: 'Redirect',
            eventTypes: [300, 302, 304, 307],
          },
          {
            name: 'Client Error',
            eventTypes: [400, 403, 404, 409],
          },
          {
            name: 'Server Error',
            eventTypes: [500, 501, 502, 503, 504],
          },
        ],
      },
      {
        title: 'Policies and Transformers',
        divId: 'containerRequests',
        querySets: [
          {
            name: 'Access Policies',
            eventTypes: [2610, 2600, 2620, 2630],
          },
          {
            name: 'Transformers',
            eventTypes: [2660, 2650, 2670],
          },
        ],
      },
      {
        title: 'Tokens',
        divId: 'containerAccCreation',
        querySets: [
          {
            name: 'Tokens',
            eventTypes: [2570, 2580],
          },
        ],
      },
      {
        title: 'Token Resolution',
        divId: 'containerAuthAndToken',
        querySets: [
          {
            name: 'Token Resolution',
            eventTypes: [2830, 2820, 2810],
          },
        ],
      },
    ],
  },
};

export default serviceChartMetadata;
