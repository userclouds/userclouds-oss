import { PageParametersResponse } from './PageParameters';

export type OIDCAuthSettings = {
  name: string;
  description: string;
};

export const oidcAuthSettingsFromPageParams = (
  params?: PageParametersResponse | undefined
) => {
  const settings: OIDCAuthSettings[] = [];

  if (params === undefined) {
    return settings;
  }

  const param =
    params.page_type_parameters.every_page.oidcAuthenticationSettings
      .current_value;
  const splitParam = param.split(',');
  splitParam.forEach((entry) => {
    const data = entry.split(':');
    if (data.length === 4) {
      const as: OIDCAuthSettings = {
        name: data[0],
        description: data[1],
      };
      settings.push(as);
    }
  });

  return settings;
};
