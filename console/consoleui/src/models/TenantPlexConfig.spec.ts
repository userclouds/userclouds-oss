import TenantPlexConfig, {
  ensureDefaults,
  addProvider,
  replaceProvider,
} from './TenantPlexConfig';
import LoginApp from './LoginApp';
import Provider from './Provider';
import { NilUuid } from './Uuids';

let plexConfig: TenantPlexConfig;
describe('TenantPlexConfig model', () => {
  beforeEach(() => {
    plexConfig = {
      tenant_config: {
        plex_map: {
          providers: [
            {
              id: 'a83f8eed-0b5e-4f3f-bcff-ad695d502849',
              name: 'UC IDP Dev (Console!)',
              type: 'uc',
              auth0: {
                domain: '',
                apps: null,
                management: {
                  client_id: '',
                  client_secret: '',
                  audience: '',
                },
                redirect: false,
              },
              uc: {
                idp_url: 'https://console-dev.tenant.dev.userclouds.tools:3333',
                apps: [
                  {
                    id: '3e3de5b2-f789-412b-8df9-859b73acbb98',
                    name: 'UC IDP Console App (dev)',
                  },
                ],
              },
            },
            {
              id: 'f0f45f77-7179-4e0e-8bf2-d5689d140f59',
              name: 'New Plex Provider',
              type: 'uc',
              auth0: {
                domain: '',
                apps: null,
                management: {
                  client_id: '',
                  client_secret: '',
                  audience: '',
                },
                redirect: false,
              },
              uc: {
                idp_url: 'https://console-dev.tenant.dev.userclouds.tools:3333',
                apps: [],
              },
            },
          ],
          apps: [
            {
              id: '90ffb499-2549-470e-99cd-77f7008e2735',
              name: 'UserClouds Console (dev)',
              description: '',
              organization_id: '1ee4497e-c326-4068-94ed-3dcdaaaa53bc',
              client_id: 'console_plex_clientid_dev',
              client_secret:
                '7c365cb45ad67f1e99da4951a6f7ed680ad2d232dd22b75d6237355bc1aa04e91aa5fbab66da1e0d9630a3a515d595f2161bb9ecb1118497054a15720b5d2cb6',
              restricted_access: false,
              token_validity: {
                access: 86400,
                refresh: 2592000,
                impersonate_user: 3600,
              },
              provider_app_ids: ['3e3de5b2-f789-412b-8df9-859b73acbb98'],
              allowed_redirect_uris: [
                'https://console.dev.userclouds.tools:3333/auth/callback',
                'https://console.dev.userclouds.tools:3010/auth/callback',
                'https://console.dev.userclouds.tools:3333/auth/invitecallback',
                'https://console.dev.userclouds.tools:3010/auth/invitecallback',
              ],
              allowed_logout_uris: [
                'https://console.dev.userclouds.tools:3333/',
                'https://console.dev.userclouds.tools:3010/',
              ],
              email_elements: {},
              page_parameters: {},
              grant_types: [
                'authorization_code',
                'refresh_token',
                'client_credentials',
              ],
              synced_from_provider: NilUuid,
              impersonate_user_config: {
                check_attribute: '',
                bypass_company_admin_check: false,
              },
            },
            {
              id: 'c3713a68-929c-4bc7-9740-4e340a814d92',
              name: 'Login for Hobart and William and Smith',
              description:
                'Login app for Hobart and William and Smith Colleges',
              organization_id: '50862005-5a77-4256-be38-b51e769de3a3',
              client_id:
                'd56d9f377c03ef9c54d21eda4d3d30c3caaac2f4d4019caa8d6c1b67222115de',
              client_secret:
                '7MwmIaoSsfJFO3/2f/WY5G/P2Var/WdkRhqKcUBOXZDm3KyaC07IWGbB0+Zu+kAgliKXUo5H1v8hHfZef+U23Q==',
              restricted_access: false,
              token_validity: {
                access: 86400,
                refresh: 2592000,
                impersonate_user: 3600,
              },
              provider_app_ids: ['3e3de5b2-f789-412b-8df9-859b73acbb98'],
              allowed_redirect_uris: [
                'https://console.dev.userclouds.tools:3333/auth/callback',
                'https://console.dev.userclouds.tools:3010/auth/callback',
                'https://console.dev.userclouds.tools:3333/auth/invitecallback',
                'https://console.dev.userclouds.tools:3010/auth/invitecallback',
              ],
              allowed_logout_uris: [
                'https://console.dev.userclouds.tools:3333/',
                'https://console.dev.userclouds.tools:3010/',
              ],
              email_elements: {},
              page_parameters: {},
              grant_types: [
                'authorization_code',
                'refresh_token',
                'client_credentials',
              ],
              synced_from_provider: NilUuid,
              impersonate_user_config: {
                check_attribute: '',
                bypass_company_admin_check: false,
              },
            },
          ],
          policy: {
            active_provider_id: 'a83f8eed-0b5e-4f3f-bcff-ad695d502849',
          },
          employee_app: {
            id: '6ed14815-2e9f-420e-8954-92396e21363f',
            name: 'Employee Plex App',
            description: '',
            organization_id: '1ee4497e-c326-4068-94ed-3dcdaaaa53bc',
            client_id: '6fa61fbf-f572-4178-aefd-111ba560da6c',
            client_secret: '755921f2-e8c0-41ce-8a12-068bb9ed863b',
            restricted_access: false,
            token_validity: {
              access: 86400,
              refresh: 2592000,
              impersonate_user: 3600,
            },
            provider_app_ids: [],
            allowed_redirect_uris: [],
            allowed_logout_uris: [],
            email_elements: {},
            page_parameters: {},
            grant_types: [],
            synced_from_provider: NilUuid,
            impersonate_user_config: {
              check_attribute: '',
              bypass_company_admin_check: false,
            },
          },
          email_server: {
            host: '',
            port: 0,
            username: '',
            password_ui: '',
            password: '',
          },
        },
        oidc_providers: {
          providers: [
            {
              type: 'facebook',
              name: 'facebook',
              description: 'Facebook',
              issuer_url: 'https://www.facebook.com',
              client_id: '477712284454886',
              client_secret: 'ccca978cae1d7ea9a56205197dd422a8',
              can_use_local_host_redirect: true,
              use_local_host_redirect: false,
              default_scopes: 'openid public_profile email',
              additional_scopes: '',
              is_native: true,
            },
            {
              type: 'google',
              name: 'google',
              description: 'Google',
              issuer_url: 'https://accounts.google.com',
              client_id:
                '712526485740-7ce3706m1flac643ca97oh3rq5un701m.apps.googleusercontent.com',
              client_secret: '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',
              can_use_local_host_redirect: false,
              use_local_host_redirect: false,
              default_scopes: 'openid profile email',
              additional_scopes: '',
              is_native: true,
            },
            {
              type: 'linkedin',
              name: 'linkedin',
              description: 'LinkedIn',
              issuer_url: 'https://www.linkedin.com',
              client_id: '78nj1bmqjulvx5',
              client_secret: 'G5yedyJlwoe8mFNi',
              can_use_local_host_redirect: false,
              use_local_host_redirect: false,
              default_scopes: 'openid profile email',
              additional_scopes: '',
              is_native: true,
            },
          ],
        },
        tenant_url: 'https://console-dev.tenant.dev.userclouds.tools:3333',
        tenant_id: '41ab79a8-0dff-418e-9d42-e1694469120a',
        verify_emails: true,
        disable_sign_ups: false,
        keys: {
          key_id: '9e7eb974',
          private_key: '',
        },
        page_parameters: {},
      },
    } as unknown as TenantPlexConfig;
  });
  describe('ensureDefaults', () => {
    [
      'provider_app_ids' as keyof LoginApp,
      'allowed_redirect_uris' as keyof LoginApp,
      'allowed_logout_uris' as keyof LoginApp,
    ].forEach((property) => {
      it(`should set ${property} to an empty array if it is null for an app`, () => {
        (plexConfig.tenant_config.plex_map.apps[1][property] as any) = null;
        expect(plexConfig.tenant_config.plex_map.apps[1][property]).toBe(null);

        plexConfig = ensureDefaults(plexConfig);
        expect(plexConfig.tenant_config.plex_map.apps[1][property]).not.toBe(
          null
        );
        expect(
          plexConfig.tenant_config.plex_map.apps[1][property] instanceof Array
        ).toBe(true);
        expect(
          (plexConfig.tenant_config.plex_map.apps[1][property] as string[])
            .length
        ).toBe(0);
      });

      it(`should set ${property} to an empty array if it is undefined for an app`, () => {
        delete plexConfig.tenant_config.plex_map.apps[1][property];
        expect(plexConfig.tenant_config.plex_map.apps[1][property]).toBeFalsy();

        plexConfig = ensureDefaults(plexConfig);
        expect(
          plexConfig.tenant_config.plex_map.apps[1][property]
        ).not.toBeFalsy();
        expect(
          plexConfig.tenant_config.plex_map.apps[1][property] instanceof Array
        ).toBe(true);
        expect(
          (plexConfig.tenant_config.plex_map.apps[1][property] as string[])
            .length
        ).toBe(0);
      });
    });

    [
      'provider_app_ids' as keyof LoginApp,
      'allowed_redirect_uris' as keyof LoginApp,
      'allowed_logout_uris' as keyof LoginApp,
    ].forEach((property) => {
      it(`should set ${property} to an empty array if it is null for the employee app`, () => {
        (plexConfig.tenant_config.plex_map.employee_app[property] as any) =
          null;
        expect(plexConfig.tenant_config.plex_map.employee_app[property]).toBe(
          null
        );

        plexConfig = ensureDefaults(plexConfig);
        expect(
          plexConfig.tenant_config.plex_map.employee_app[property]
        ).not.toBe(null);
        expect(
          plexConfig.tenant_config.plex_map.employee_app[property] instanceof
            Array
        ).toBe(true);
        expect(
          (plexConfig.tenant_config.plex_map.employee_app[property] as string[])
            .length
        ).toBe(0);
      });

      it(`should set ${property} to an empty array if it is undefined for the employee app`, () => {
        delete plexConfig.tenant_config.plex_map.employee_app[property];
        expect(
          plexConfig.tenant_config.plex_map.employee_app[property]
        ).toBeFalsy();

        plexConfig = ensureDefaults(plexConfig);
        expect(
          plexConfig.tenant_config.plex_map.employee_app[property]
        ).not.toBeFalsy();
        expect(
          plexConfig.tenant_config.plex_map.employee_app[property] instanceof
            Array
        ).toBe(true);
        expect(
          (plexConfig.tenant_config.plex_map.employee_app[property] as string[])
            .length
        ).toBe(0);
      });
    });
  });

  describe('addProvider', () => {
    it('should add a Provider to the end of the providers array', () => {
      const newProvider: Provider = {
        id: '123123aa-7179-4e0e-8bf2-d5689d140f59',
        name: 'foo',
        type: 'uc',
        auth0: {
          domain: '',
          apps: null,
          management: {
            client_id: '',
            client_secret: '',
            audience: '',
          },
          redirect: false,
        },
        uc: {
          idp_url: 'https://console-dev.tenant.dev.userclouds.tools:3333',
          apps: [],
        },
      } as unknown as Provider;

      expect(plexConfig.tenant_config.plex_map.providers instanceof Array).toBe(
        true
      );
      expect(
        (plexConfig.tenant_config.plex_map.providers as Provider[]).length
      ).toBe(2);

      plexConfig = addProvider(plexConfig, newProvider);
      expect(
        (plexConfig.tenant_config.plex_map.providers as Provider[]).length
      ).toBe(3);
      expect(plexConfig.tenant_config.plex_map.providers[2].name).toBe('foo');
    });

    it('should add a Provider to the end of the providers array when the array is empty', () => {
      plexConfig.tenant_config.plex_map.providers = [];
      const newProvider: Provider = {
        id: '123123aa-7179-4e0e-8bf2-d5689d140f59',
        name: 'foo',
        type: 'uc',
        auth0: {
          domain: '',
          apps: null,
          management: {
            client_id: '',
            client_secret: '',
            audience: '',
          },
          redirect: false,
        },
        uc: {
          idp_url: 'https://console-dev.tenant.dev.userclouds.tools:3333',
          apps: [],
        },
      } as unknown as Provider;

      expect(plexConfig.tenant_config.plex_map.providers instanceof Array).toBe(
        true
      );
      expect(
        (plexConfig.tenant_config.plex_map.providers as Provider[]).length
      ).toBe(0);

      plexConfig = addProvider(plexConfig, newProvider);
      expect(
        (plexConfig.tenant_config.plex_map.providers as Provider[]).length
      ).toBe(1);
      expect(plexConfig.tenant_config.plex_map.providers[0].name).toBe('foo');
    });
  });

  describe('replaceProvider', () => {
    it('should replace the provider with a matching ID', () => {
      const newProvider: Provider = {
        id: 'f0f45f77-7179-4e0e-8bf2-d5689d140f59',
        name: 'foo',
        type: 'uc',
        auth0: {
          domain: '',
          apps: null,
          management: {
            client_id: '',
            client_secret: '',
            audience: '',
          },
          redirect: false,
        },
        uc: {
          idp_url: 'https://console-dev.tenant.dev.userclouds.tools:3333',
          apps: [],
        },
      } as unknown as Provider;
      expect(plexConfig.tenant_config.plex_map.providers instanceof Array).toBe(
        true
      );
      expect(
        (plexConfig.tenant_config.plex_map.providers as Provider[]).length
      ).toBe(2);
      expect(plexConfig.tenant_config.plex_map.providers[1].name).toBe(
        'New Plex Provider'
      );

      plexConfig = replaceProvider(plexConfig, newProvider);
      expect(plexConfig.tenant_config.plex_map.providers instanceof Array).toBe(
        true
      );
      expect(
        (plexConfig.tenant_config.plex_map.providers as Provider[]).length
      ).toBe(2);
      expect(plexConfig.tenant_config.plex_map.providers[1].name).toBe('foo');
    });

    it('should do nothing if the ID of the provider passed in is not matched', () => {
      const newProvider: Provider = {
        id: '123123aa-7179-4e0e-8bf2-d5689d140f59',
        name: 'foo',
        type: 'uc',
        auth0: {
          domain: '',
          apps: null,
          management: {
            client_id: '',
            client_secret: '',
            audience: '',
          },
          redirect: false,
        },
        uc: {
          idp_url: 'https://console-dev.tenant.dev.userclouds.tools:3333',
          apps: [],
        },
      } as unknown as Provider;
      expect(plexConfig.tenant_config.plex_map.providers instanceof Array).toBe(
        true
      );
      expect(
        (plexConfig.tenant_config.plex_map.providers as Provider[]).length
      ).toBe(2);
      expect(plexConfig.tenant_config.plex_map.providers[1].name).toBe(
        'New Plex Provider'
      );

      plexConfig = replaceProvider(plexConfig, newProvider);
      expect(plexConfig.tenant_config.plex_map.providers instanceof Array).toBe(
        true
      );
      expect(
        (plexConfig.tenant_config.plex_map.providers as Provider[]).length
      ).toBe(2);
      expect(plexConfig.tenant_config.plex_map.providers[1].name).toBe(
        'New Plex Provider'
      );
    });
  });
});
