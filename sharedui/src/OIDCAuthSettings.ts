import {
  facebookLoginImg,
  googleLoginImg,
  linkedInLoginImg,
  customLoginImg,
  msftLoginImg,
} from './SocialLoginImages';

export type OIDCAuthSettings = {
  name: string;
  description: string;
  buttonText: string;
};

export const oidcAuthSettingsFromPageParams = (
  params: Record<string, string>,
  forMerge: boolean
) => {
  const settings: OIDCAuthSettings[] = [];

  const param = params.oidcAuthenticationSettings;
  const splitParam = param.split(',');
  splitParam.forEach((entry) => {
    const data = entry.split(':');
    if (data.length === 4) {
      const as: OIDCAuthSettings = {
        name: data[0],
        description: data[1],
        buttonText: forMerge ? data[3] : data[2],
      };
      settings.push(as);
    }
  });

  return settings;
};

export const getOIDCButtonImage = (as: OIDCAuthSettings) => {
  switch (as.name) {
    case 'facebook':
      return facebookLoginImg;
    case 'google':
      return googleLoginImg;
    case 'linkedin':
      return linkedInLoginImg;
    case 'microsoft':
      return msftLoginImg;
    default:
      return customLoginImg;
  }
};
