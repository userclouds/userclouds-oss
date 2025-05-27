import {
  OIDCAuthSettings,
  oidcAuthSettingsFromPageParams,
} from './OIDCAuthSettings';
import { PageParametersResponse } from './PageParameters';

describe('OIDCAuthSettings', () => {
  it('should return empty auth settings', () => {
    const settings: OIDCAuthSettings[] =
      oidcAuthSettingsFromPageParams(undefined);
    expect(settings.length).toBe(0);
  });

  it('should successfully create auth settings', () => {
    const pageParams: PageParametersResponse = {
      tenant_id: 'unused',
      app_id: 'unused',
      page_type_parameters: {
        every_page: {
          oidcAuthenticationSettings: {
            name: 'oidcAuthenticationSettings',
            current_value: 'a1:a2:a3:a4,b1:b2:b3:b4,c1:c2:c3:c4',
            default_value: 'unused',
          },
        },
      },
    };

    const settings: OIDCAuthSettings[] =
      oidcAuthSettingsFromPageParams(pageParams);
    expect(settings.length).toBe(3);
    expect(settings[0].name).toBe('a1');
    expect(settings[0].description).toBe('a2');
    expect(settings[1].name).toBe('b1');
    expect(settings[1].description).toBe('b2');
    expect(settings[2].name).toBe('c1');
    expect(settings[2].description).toBe('c2');
  });

  it('should skip invalid auth settings', () => {
    const pageParams: PageParametersResponse = {
      tenant_id: 'unused',
      app_id: 'unused',
      page_type_parameters: {
        every_page: {
          oidcAuthenticationSettings: {
            name: 'oidcAuthenticationSettings',
            current_value: 'a1:a2:a3:a4,b1:b4,c1:c2:c3:c4',
            default_value: 'unused',
          },
        },
      },
    };

    const settings: OIDCAuthSettings[] =
      oidcAuthSettingsFromPageParams(pageParams);
    expect(settings.length).toBe(2);
    expect(settings[0].name).toBe('a1');
    expect(settings[0].description).toBe('a2');
    expect(settings[1].name).toBe('c1');
    expect(settings[1].description).toBe('c2');
  });
});
