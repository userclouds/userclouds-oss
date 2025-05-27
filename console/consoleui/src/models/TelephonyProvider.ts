export enum TelephonyProviderName {
  Twilio = 'twilio',
}

export enum TwilioPropertyType {
  AccountSID = 'twilio_account_sid',
  APIKeySID = 'twilio_api_key_sid',
  APISecret = 'twilio_api_secret',
}

export type TelephonyProviderProperties = {
  [key in TwilioPropertyType]?: string;
};

type TelephonyProvider = {
  type: TelephonyProviderName;
  properties: TelephonyProviderProperties;
};

export const modifyProviderProperties = (
  provider: TelephonyProvider,
  changes: TelephonyProviderProperties
) => {
  return {
    // we only have 1 provider, so this works for now
    type: TelephonyProviderName.Twilio,
    properties: Object.assign(provider.properties, changes),
  };
};

export default TelephonyProvider;
