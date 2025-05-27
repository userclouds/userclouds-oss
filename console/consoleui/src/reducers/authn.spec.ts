import authnReducer from './authn';
import { initialState, RootState } from '../store';
import TenantPlexConfig from '../models/TenantPlexConfig';
import { AppMessageElement } from '../models/MessageElements';
import LoginApp from '../models/LoginApp';
import {
  MODIFY_PLEX_CONFIG,
  CLONE_PLEX_APP_SETTINGS,
  GET_EMAIL_MSG_ELEMENTS_SUCCESS,
} from '../actions/authn';
import { NilUuid } from '../models/Uuids';
import { PageParametersResponse } from '../models/PageParameters';

describe('authn reducer', () => {
  let plexConfig: TenantPlexConfig;
  let state: RootState;
  let newState: RootState;
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
              message_elements: {},
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
              message_elements: {},
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
            message_elements: {},
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

  describe('MODIFY_PLEX_CONFIG', () => {
    it('should set plexConfigIsDirty to true if something changes', () => {
      state = {
        ...initialState,
        tenantPlexConfig: JSON.parse(JSON.stringify(plexConfig)),
        modifiedPlexConfig: JSON.parse(JSON.stringify(plexConfig)),
      };

      const modifiedPlexConfig = {
        ...plexConfig,
        tenant_config: {
          ...plexConfig.tenant_config,
          verify_emails: false,
        },
      };
      expect(state.plexConfigIsDirty).toBe(false);

      newState = authnReducer(state, {
        type: MODIFY_PLEX_CONFIG,
        data: modifiedPlexConfig,
      });
      expect(newState.plexConfigIsDirty).toBe(true);
    });

    it('should set plexConfigIsDirty to true if something changes and false if it changes back', () => {
      state = {
        ...initialState,
        tenantPlexConfig: JSON.parse(JSON.stringify(plexConfig)),
        modifiedPlexConfig: JSON.parse(JSON.stringify(plexConfig)),
      };

      let modifiedPlexConfig = {
        ...plexConfig,
        tenant_config: {
          ...plexConfig.tenant_config,
          verify_emails: false,
        },
      };
      expect(state.plexConfigIsDirty).toBe(false);

      newState = authnReducer(state, {
        type: MODIFY_PLEX_CONFIG,
        data: modifiedPlexConfig,
      });
      expect(newState.plexConfigIsDirty).toBe(true);

      modifiedPlexConfig = {
        ...plexConfig,
        tenant_config: {
          ...plexConfig.tenant_config,
          verify_emails: true,
        },
      };
      newState = authnReducer(state, {
        type: MODIFY_PLEX_CONFIG,
        data: modifiedPlexConfig,
      });
      expect(newState.plexConfigIsDirty).toBe(false);
    });

    it('should be able to make complex, nested changes', () => {
      state = {
        ...initialState,
        tenantPlexConfig: JSON.parse(JSON.stringify(plexConfig)),
        modifiedPlexConfig: JSON.parse(JSON.stringify(plexConfig)),
      };
      expect(state.tenantPlexConfig!.tenant_config.verify_emails).toBe(true);
      expect(
        state.tenantPlexConfig!.tenant_config.plex_map.providers.length
      ).toBe(2);
      expect(
        state.tenantPlexConfig!.tenant_config.plex_map.providers[0].uc!.idp_url
      ).toBe('https://console-dev.tenant.dev.userclouds.tools:3333');
      expect(
        state.tenantPlexConfig!.tenant_config.plex_map.apps[0]
          .allowed_redirect_uris.length
      ).toBe(4);
      expect(state.modifiedPlexConfig!.tenant_config.verify_emails).toBe(true);
      expect(
        state.modifiedPlexConfig!.tenant_config.plex_map.providers.length
      ).toBe(2);
      expect(
        state.modifiedPlexConfig!.tenant_config.plex_map.providers[0].uc!
          .idp_url
      ).toBe('https://console-dev.tenant.dev.userclouds.tools:3333');
      expect(
        state.modifiedPlexConfig!.tenant_config.plex_map.apps[0]
          .allowed_redirect_uris.length
      ).toBe(4);

      const modifiedPlexConfig = {
        ...plexConfig,
        tenant_config: {
          ...plexConfig.tenant_config,
          plex_map: {
            ...plexConfig.tenant_config.plex_map,
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
                  },
                  redirect: false,
                },
                uc: {
                  idp_url:
                    'https://console-dev.tenant.dev.userclouds.tools:3010',
                  apps: [
                    {
                      id: '3e3de5b2-f789-412b-8df9-859b73acbb98',
                      name: 'UC IDP Console App (dev)',
                    },
                  ],
                },
              },
            ],
            apps: [
              {
                ...plexConfig.tenant_config.plex_map.apps[0],
                allowed_redirect_uris: [
                  'https://console.dev.userclouds.tools:3333/auth/callback',
                  'https://console.dev.userclouds.tools:3010/auth/callback',
                  'https://console.dev.userclouds.tools:3333/auth/invitecallback',
                  'https://console.dev.userclouds.tools:3010/auth/invitecallback',
                  'https://dev.userclouds.tools:3011/auth/invitecallback',
                ],
              },
              plexConfig.tenant_config.plex_map.apps[1],
            ],
          },
          verify_emails: false,
        },
      };
      newState = authnReducer(state, {
        type: MODIFY_PLEX_CONFIG,
        data: modifiedPlexConfig,
      });
      expect(newState.tenantPlexConfig!.tenant_config.verify_emails).toBe(true);
      expect(
        newState.tenantPlexConfig!.tenant_config.plex_map.providers.length
      ).toBe(2);
      expect(
        newState.tenantPlexConfig!.tenant_config.plex_map.providers[0].uc!
          .idp_url
      ).toBe('https://console-dev.tenant.dev.userclouds.tools:3333');
      expect(
        newState.tenantPlexConfig!.tenant_config.plex_map.apps[0]
          .allowed_redirect_uris.length
      ).toBe(4);
      expect(newState.plexConfigIsDirty).toBe(true);
      expect(newState.modifiedPlexConfig!.tenant_config.verify_emails).toBe(
        false
      );
      expect(
        newState.modifiedPlexConfig!.tenant_config.plex_map.providers.length
      ).toBe(1);
      expect(
        newState.modifiedPlexConfig!.tenant_config.plex_map.providers[0].uc!
          .idp_url
      ).toBe('https://console-dev.tenant.dev.userclouds.tools:3010');
      expect(
        newState.modifiedPlexConfig!.tenant_config.plex_map.apps[0]
          .allowed_redirect_uris.length
      ).toBe(5);
      expect(
        newState.modifiedPlexConfig!.tenant_config.plex_map.apps[0]
          .allowed_redirect_uris[4]
      ).toBe('https://dev.userclouds.tools:3011/auth/invitecallback');
    });
  });

  describe('CLONE_PLEX_APP_SETTINGS', () => {
    let emailMessageElements: AppMessageElement[];
    let smsMessageElements: AppMessageElement[];
    let loginApp: LoginApp;

    beforeEach(() => {
      emailMessageElements = [
        {
          app_id: '90ffb499-2549-470e-99cd-77f7008e2735',
          message_type_message_elements: {
            invite_new: {
              type: 'invite_new',
              message_elements: {
                sender: {
                  type: 'sender',
                  default_value: 'foo',
                  custom_value: 'bar',
                },
              },
              message_parameters: [
                {
                  name: 'foo',
                  default_value: 'bar',
                },
              ],
            },
          },
        },
        {
          app_id: 'c3713a68-929c-4bc7-9740-4e340a814d92',
          message_type_message_elements: {
            invite_new: {
              type: 'invite_new',
              message_elements: {
                sender: {
                  type: 'sender',
                  default_value: 'zoo',
                  custom_value: 'xylophone',
                },
              },
              message_parameters: [
                {
                  name: 'baz',
                  default_value: 'fizzbuzz',
                },
              ],
            },
          },
        },
        {
          app_id: 'f53f44ef-9ea1-4657-8f65-8958eba3db12',
          message_type_message_elements: {
            invite_new: {
              type: 'invite_new',
              message_elements: {
                sender: {
                  type: 'sender',
                  default_value: 'captain',
                  custom_value: 'crunch',
                },
              },
              message_parameters: [
                {
                  name: 'bar',
                  default_value: 'baz',
                },
              ],
            },
          },
        },
      ];
      smsMessageElements = [
        {
          app_id: 'a192e935-b82e-48cb-a926-3860a0a4cdf6',
          message_type_message_elements: {
            sms_mfa_challenge: {
              type: 'sms_mfa_challenge',
              message_elements: {
                sms_sender: {
                  type: 'sms_sender',
                  default_value: '+1111111111',
                  custom_value: '+1234545678',
                },
              },
              message_parameters: [
                {
                  name: 'baz',
                  default_value: 'fizzbuzz',
                },
              ],
            },
          },
        },
      ];
      loginApp = {
        id: 'c3713a68-929c-4bc7-9740-4e340a814d92',
        name: 'foo',
        description: 'bar',
        client_id: '',
        client_secret: '',
        organization_id: '',
        token_validity: {},
        provider_app_ids: [],
        allowed_redirect_uris: [],
        allowed_logout_uris: [],
        grant_types: [],
        synced_from_provider: '',
        restricted_access: false,
        message_elements: {},
        page_parameters: {},
        impersonate_user_config: {
          check_attribute: '',
          bypass_company_admin_check: false,
        },
      };
    });

    it('should set the target app to be identical to the source app in modifiedEmailMessageElements', () => {
      state = {
        ...initialState,
        tenantEmailMessageElements: JSON.parse(
          JSON.stringify(emailMessageElements)
        ),
        modifiedEmailMessageElements: JSON.parse(
          JSON.stringify(emailMessageElements)
        ),
        tenantSMSMessageElements: JSON.parse(
          JSON.stringify(smsMessageElements)
        ),
        modifiedSMSMessageElements: JSON.parse(
          JSON.stringify(smsMessageElements)
        ),
        selectedPlexApp: loginApp,
        appPageParameters: {
          'f53f44ef-9ea1-4657-8f65-8958eba3db12':
            {} as unknown as PageParametersResponse,
        },
      };
      expect(state.tenantEmailMessageElements).toBeTruthy();
      expect(state.modifiedEmailMessageElements).toBeTruthy();
      if (
        state.tenantEmailMessageElements &&
        state.modifiedEmailMessageElements
      ) {
        expect(state.tenantEmailMessageElements.length).toBe(3);
        expect(
          state.tenantEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('foo');
        expect(
          state.tenantEmailMessageElements[1].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('zoo');
        expect(
          state.tenantEmailMessageElements[2].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('captain');
        expect(state.modifiedEmailMessageElements.length).toBe(3);
        expect(
          state.modifiedEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('foo');
        expect(
          state.modifiedEmailMessageElements[1].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('zoo');
        expect(
          state.modifiedEmailMessageElements[2].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('captain');
      }
      newState = authnReducer(state, {
        type: CLONE_PLEX_APP_SETTINGS,
        data: 'f53f44ef-9ea1-4657-8f65-8958eba3db12',
      });

      expect(newState.tenantEmailMessageElements).toBeTruthy();
      expect(newState.modifiedEmailMessageElements).toBeTruthy();
      if (
        newState.tenantEmailMessageElements &&
        newState.modifiedEmailMessageElements
      ) {
        expect(newState.tenantEmailMessageElements!.length).toBe(3);
        expect(
          newState.tenantEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('foo');
        expect(
          newState.tenantEmailMessageElements[1].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('zoo');
        expect(
          newState.tenantEmailMessageElements[2].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('captain');
        expect(newState.modifiedEmailMessageElements!.length).toBe(3);
        expect(
          newState.modifiedEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('foo');
        expect(
          newState.modifiedEmailMessageElements[1].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('captain');
        expect(
          newState.modifiedEmailMessageElements[2].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('captain');
        expect(newState.emailMessageElementsAreDirty).toBe(true);
        expect(newState.appToClone).toBe(
          'f53f44ef-9ea1-4657-8f65-8958eba3db12'
        );
      }
    });

    it('should not change modifiedEmailMessageElements if the source app is not found', () => {
      state = {
        ...initialState,
        tenantEmailMessageElements: JSON.parse(
          JSON.stringify(emailMessageElements)
        ),
        modifiedEmailMessageElements: JSON.parse(
          JSON.stringify(emailMessageElements)
        ),
        tenantSMSMessageElements: JSON.parse(
          JSON.stringify(smsMessageElements)
        ),
        modifiedSMSMessageElements: JSON.parse(
          JSON.stringify(smsMessageElements)
        ),
        selectedPlexApp: loginApp,
        appPageParameters: {
          'f53f44ef-9ea1-4657-8f65-8958eba3db12':
            {} as unknown as PageParametersResponse,
        },
      };
      expect(state.tenantEmailMessageElements).toBeTruthy();
      expect(state.modifiedEmailMessageElements).toBeTruthy();
      if (
        state.tenantEmailMessageElements &&
        state.modifiedEmailMessageElements
      ) {
        expect(state.tenantEmailMessageElements.length).toBe(3);
        expect(
          state.tenantEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('foo');
        expect(
          state.tenantEmailMessageElements[1].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('zoo');
        expect(
          state.tenantEmailMessageElements[2].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('captain');
        expect(state.modifiedEmailMessageElements.length).toBe(3);
        expect(
          state.modifiedEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('foo');
        expect(
          state.modifiedEmailMessageElements[1].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('zoo');
        expect(
          state.modifiedEmailMessageElements[2].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('captain');
      }
      newState = authnReducer(state, {
        type: CLONE_PLEX_APP_SETTINGS,
        data: 'aaaaaaaa-9ea1-4657-8f65-8958eba3db12',
      });

      expect(newState.tenantEmailMessageElements).toBeTruthy();
      expect(newState.modifiedEmailMessageElements).toBeTruthy();
      if (
        newState.tenantEmailMessageElements &&
        newState.modifiedEmailMessageElements
      ) {
        expect(newState.tenantEmailMessageElements!.length).toBe(3);
        expect(
          newState.tenantEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('foo');
        expect(
          newState.tenantEmailMessageElements[1].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('zoo');
        expect(
          newState.tenantEmailMessageElements[2].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('captain');
        expect(newState.modifiedEmailMessageElements!.length).toBe(3);
        expect(
          newState.modifiedEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('foo');
        expect(
          newState.modifiedEmailMessageElements[1].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('zoo');
        expect(
          newState.modifiedEmailMessageElements[2].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('captain');
        expect(newState.emailMessageElementsAreDirty).toBe(false);
        expect(newState.appToClone).toBe('');
      }
    });

    it('should not set emailMessageElementsAreDirty to false if the cloned app is identical', () => {
      emailMessageElements.push({
        app_id: 'a192e935-b82e-48cb-a926-3860a0a4cdf6',
        message_type_message_elements: {
          invite_new: {
            type: 'invite_new',
            message_elements: {
              sender: {
                type: 'sender',
                default_value: 'zoo',
                custom_value: 'xylophone',
              },
            },
            message_parameters: [
              {
                name: 'baz',
                default_value: 'fizzbuzz',
              },
            ],
          },
        },
      });
      state = {
        ...initialState,
        tenantEmailMessageElements: JSON.parse(
          JSON.stringify(emailMessageElements)
        ),
        modifiedEmailMessageElements: JSON.parse(
          JSON.stringify(emailMessageElements)
        ),
        tenantSMSMessageElements: JSON.parse(
          JSON.stringify(smsMessageElements)
        ),
        modifiedSMSMessageElements: JSON.parse(
          JSON.stringify(smsMessageElements)
        ),
        selectedPlexApp: loginApp,
        appPageParameters: {
          'f53f44ef-9ea1-4657-8f65-8958eba3db12':
            {} as unknown as PageParametersResponse,
          'a192e935-b82e-48cb-a926-3860a0a4cdf6':
            {} as unknown as PageParametersResponse,
        },
      };
      expect(state.tenantEmailMessageElements).toBeTruthy();
      expect(state.modifiedEmailMessageElements).toBeTruthy();
      expect(state.tenantSMSMessageElements).toBeTruthy();
      expect(state.modifiedSMSMessageElements).toBeTruthy();
      if (
        state.tenantEmailMessageElements &&
        state.modifiedEmailMessageElements
      ) {
        expect(state.tenantEmailMessageElements.length).toBe(4);
        expect(
          state.tenantEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('foo');
        expect(
          state.tenantEmailMessageElements[1].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('zoo');
        expect(
          state.tenantEmailMessageElements[2].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('captain');
        expect(
          state.tenantEmailMessageElements[3].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('zoo');
        expect(state.modifiedEmailMessageElements.length).toBe(4);
        expect(
          state.modifiedEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('foo');
        expect(
          state.modifiedEmailMessageElements[1].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('zoo');
        expect(
          state.modifiedEmailMessageElements[2].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('captain');
        expect(
          state.modifiedEmailMessageElements[3].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('zoo');
      }
      newState = authnReducer(state, {
        type: CLONE_PLEX_APP_SETTINGS,
        data: 'a192e935-b82e-48cb-a926-3860a0a4cdf6',
      });

      expect(newState.tenantEmailMessageElements).toBeTruthy();
      expect(newState.modifiedEmailMessageElements).toBeTruthy();
      if (
        newState.tenantEmailMessageElements &&
        newState.modifiedEmailMessageElements
      ) {
        expect(newState.tenantEmailMessageElements!.length).toBe(4);
        expect(
          newState.tenantEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('foo');
        expect(
          newState.tenantEmailMessageElements[1].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('zoo');
        expect(
          newState.tenantEmailMessageElements[2].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('captain');
        expect(
          newState.tenantEmailMessageElements[3].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('zoo');
        expect(newState.modifiedEmailMessageElements!.length).toBe(4);
        expect(
          newState.modifiedEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('foo');
        expect(
          newState.modifiedEmailMessageElements[1].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('zoo');
        expect(
          newState.modifiedEmailMessageElements[2].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('captain');
        expect(
          newState.modifiedEmailMessageElements[3].message_type_message_elements
            .invite_new.message_elements.sender.default_value
        ).toBe('zoo');
        expect(newState.emailMessageElementsAreDirty).toBe(false);
        expect(newState.appToClone).toBe(
          'a192e935-b82e-48cb-a926-3860a0a4cdf6'
        );
      }
    });
  });

  describe('GET_EMAIL_MSG_ELEMENTS_SUCCESS', () => {
    let emailMsgElements: AppMessageElement[];

    beforeEach(() => {
      emailMsgElements = [
        {
          app_id: 'a192e935-b82e-48cb-a926-3860a0a4cdf6',
          message_type_message_elements: {
            invite_new: {
              type: 'invite_new',
              message_elements: {
                sender: {
                  type: 'sender',
                  default_value: 'zoo',
                  custom_value: 'xylophone',
                },
              },
              message_parameters: [
                {
                  name: 'baz',
                  default_value: 'fizzbuzz',
                },
              ],
            },
          },
        },
      ];
    });

    it('should ensure there is no referential equality between tenantMessageElements and modifiedMessageElements', () => {
      state = { ...initialState };

      newState = authnReducer(state, {
        type: GET_EMAIL_MSG_ELEMENTS_SUCCESS,
        data: emailMsgElements,
      });

      expect(newState.tenantEmailMessageElements).toBeTruthy();
      expect(newState.modifiedEmailMessageElements).toBeTruthy();

      if (
        newState.tenantEmailMessageElements &&
        newState.modifiedEmailMessageElements
      ) {
        expect(newState.tenantEmailMessageElements).toEqual(
          newState.modifiedEmailMessageElements
        );
        expect(newState.tenantEmailMessageElements).not.toBe(
          newState.modifiedEmailMessageElements
        );

        expect(
          newState.tenantEmailMessageElements[0].message_type_message_elements
            .invite_new
        ).toEqual(
          newState.modifiedEmailMessageElements[0].message_type_message_elements
            .invite_new
        );
        expect(
          newState.tenantEmailMessageElements[0].message_type_message_elements
            .invite_new
        ).not.toBe(
          newState.modifiedEmailMessageElements[0].message_type_message_elements
            .invite_new
        );

        expect(
          newState.tenantEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender
        ).toEqual(
          newState.modifiedEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender
        );
        expect(
          newState.tenantEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender
        ).not.toBe(
          newState.modifiedEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender
        );

        expect(
          newState.tenantEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender.custom_value
        ).toEqual(
          newState.modifiedEmailMessageElements[0].message_type_message_elements
            .invite_new.message_elements.sender.custom_value
        );
      }
    });

    it('should set emailMsgElementsAreDirty to false', () => {
      state = { ...initialState };

      newState = authnReducer(state, {
        type: GET_EMAIL_MSG_ELEMENTS_SUCCESS,
        data: emailMsgElements,
      });

      expect(newState.tenantEmailMessageElements).toBeTruthy();
      expect(newState.modifiedEmailMessageElements).toBeTruthy();

      expect(newState.emailMessageElementsAreDirty).toBe(false);
    });
  });
});
