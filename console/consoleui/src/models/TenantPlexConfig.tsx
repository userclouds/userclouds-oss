import LoginApp from './LoginApp';
import ProviderApp from './ProviderApp';
import Keys from './Keys';
import { OIDCProvider } from './OIDCProvider';
import PlexMap from './PlexMap';
import Provider, { ProviderType } from './Provider';
import {
  modifyProviderProperties,
  TwilioPropertyType,
} from './TelephonyProvider';

type TenantPlexConfig = {
  tenant_config: {
    plex_map: PlexMap;
    oidc_providers: {
      providers: OIDCProvider[];
    };
    external_oidc_issuers: string[];
    verify_emails: boolean;
    disable_sign_ups: boolean;
    keys: Keys;
  };
};

export enum UpdatePlexConfigReason {
  Default = 'Successfully saved login settings',
  AddApp = 'Successfully added a new login app',
  ModifyApp = 'Successfully updated login app',
  ModifyEmployeeApp = 'Successfully updated employee login app',
  AddProvider = 'Successfully added a new identity provider',
  ChangeExternalOIDCIssuers = 'Succesfully changed external OIDC Issuers',
  ModifyProvider = 'Successfully updated identity provider',
  ModifyProviderApps = 'Successfully updated login apps for this platform',
  // eslint-disable-next-line @typescript-eslint/no-duplicate-enum-values
  ModifyOIDCProvider = 'Successfully updated identity provider',
  ModifyEmailServer = 'Successfully updated email settings',
  ModifyTelephonyProvider = 'Successfully updated telephony provider',
}

export const findPlexAppReferencingProvider = (
  apps: LoginApp[],
  provider: Provider
): LoginApp | undefined => {
  let providerApps: ProviderApp[] = [];
  switch (provider.type) {
    case ProviderType.auth0:
      providerApps = provider.auth0?.apps || [];
      break;
    case ProviderType.uc:
      providerApps = provider.uc?.apps || [];
      break;
    default:
  }
  // If any Plex Apps reference any provider apps from this provider,
  // return the first one, otherwise undefined.
  return apps.find(
    (app) =>
      !!app.provider_app_ids?.find((provider_app_id: string) =>
        providerApps.find((provider_app) => provider_app.id === provider_app_id)
      )
  );
};

export const findPlexAppReferencingProviderApp = (
  apps: LoginApp[],
  providerAppID: string
): LoginApp | undefined => {
  // If any Plex Apps reference this provider app, return the first one, otherwise undefined.
  return apps.find(
    (app) => !!app.provider_app_ids?.find((pAppID) => pAppID === providerAppID)
  );
};

export const ensureDefaults = (config: TenantPlexConfig): TenantPlexConfig => {
  config.tenant_config.plex_map.apps = config.tenant_config.plex_map.apps.map(
    (app: LoginApp) => {
      app.provider_app_ids = app.provider_app_ids || [];
      app.allowed_redirect_uris = app.allowed_redirect_uris || [];
      app.allowed_logout_uris = app.allowed_logout_uris || [];
      return app;
    }
  );
  ['provider_app_ids', 'allowed_redirect_uris', 'allowed_logout_uris'].forEach(
    (key: string) => {
      if (!config.tenant_config.plex_map.employee_app[key as keyof LoginApp]) {
        (config.tenant_config.plex_map.employee_app[
          key as keyof LoginApp
        ] as string[]) = [];
      }
    }
  );
  return config;
};

export const addProvider = (
  config: TenantPlexConfig,
  provider: Provider
): TenantPlexConfig => {
  return {
    tenant_config: {
      ...config.tenant_config,
      plex_map: {
        ...config.tenant_config.plex_map,
        providers: [...config.tenant_config.plex_map.providers, provider],
      },
    },
  };
};

export const replaceOIDCProvider = (
  config: TenantPlexConfig,
  provider: OIDCProvider,
  name: string
): TenantPlexConfig => {
  return {
    tenant_config: {
      ...config.tenant_config,
      oidc_providers: {
        providers: config.tenant_config.oidc_providers.providers.map(
          (p: OIDCProvider) => {
            if (p.name === name) {
              return provider;
            }
            return p;
          }
        ),
      },
    },
  };
};

export const replaceProvider = (
  config: TenantPlexConfig,
  provider: Provider
): TenantPlexConfig => {
  return {
    tenant_config: {
      ...config.tenant_config,
      plex_map: {
        ...config.tenant_config.plex_map,
        providers: config.tenant_config.plex_map.providers.map((p: any) => {
          if (p.id === provider.id) {
            p = provider;
          }
          return p;
        }),
      },
    },
  };
};

export const modifyTelephonyProvider = (
  config: TenantPlexConfig,
  changes: Record<TwilioPropertyType, string>
) => {
  return {
    tenant_config: {
      ...config.tenant_config,
      plex_map: {
        ...config.tenant_config.plex_map,
        telephony_provider: modifyProviderProperties(
          config.tenant_config.plex_map.telephony_provider,
          changes
        ),
      },
    },
  };
};

export default TenantPlexConfig;
