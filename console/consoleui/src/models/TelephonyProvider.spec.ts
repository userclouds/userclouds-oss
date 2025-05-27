import TelephonyProvider, {
  TelephonyProviderName,
  TwilioPropertyType,
  modifyProviderProperties,
} from './TelephonyProvider';

describe('TelephonyProvider', () => {
  describe('modifyProverProperties', () => {
    it('should assign a new property to an empty properties object', () => {
      const provider: TelephonyProvider = {
        type: TelephonyProviderName.Twilio,
        properties: {},
      };

      const modifiedProvider = modifyProviderProperties(provider, {
        [TwilioPropertyType.AccountSID]: 'foo',
      });
      expect(modifiedProvider.properties).toEqual({
        twilio_account_sid: 'foo',
      });
    });

    it('should add an unspecified property to the properties object', () => {
      const provider: TelephonyProvider = {
        type: TelephonyProviderName.Twilio,
        properties: { [TwilioPropertyType.AccountSID]: 'foo' },
      };
      const modifiedProvider = modifyProviderProperties(provider, {
        [TwilioPropertyType.APISecret]: 'bar',
      });
      expect(modifiedProvider.properties).toEqual({
        twilio_account_sid: 'foo',
        twilio_api_secret: 'bar',
      });
    });

    it('should overwrite existing properties on the properties object', () => {
      const provider: TelephonyProvider = {
        type: TelephonyProviderName.Twilio,
        properties: {
          [TwilioPropertyType.AccountSID]: 'foo',
          [TwilioPropertyType.APIKeySID]: 'bar',
          [TwilioPropertyType.APISecret]: 'baz',
        },
      };
      let modifiedProvider = modifyProviderProperties(provider, {
        [TwilioPropertyType.AccountSID]: 'foofoo',
        [TwilioPropertyType.APIKeySID]: 'barbar',
      });
      expect(modifiedProvider.properties).toEqual({
        twilio_account_sid: 'foofoo',
        twilio_api_key_sid: 'barbar',
        twilio_api_secret: 'baz',
      });
      modifiedProvider = modifyProviderProperties(modifiedProvider, {
        [TwilioPropertyType.APISecret]: 'bazbaz',
      });
      expect(modifiedProvider.properties).toEqual({
        twilio_account_sid: 'foofoo',
        twilio_api_key_sid: 'barbar',
        twilio_api_secret: 'bazbaz',
      });
    });
  });
});
