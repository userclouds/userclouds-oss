package builder

import "userclouds.com/infra/uctypes/messaging/telephony"

// TwilioProviderBuilder introduces TelephonyProvider building methods for a TenantConfig twilio telephony provider
type TwilioProviderBuilder struct {
	TenantConfigBuilder
}

// ConfigureTwilioProvider returns a TwilioProviderBuilder that can configure a twilio provider
func (tcb *TenantConfigBuilder) ConfigureTwilioProvider() *TwilioProviderBuilder {
	tcb.plexMap.TelephonyProvider.Type = telephony.ProviderTypeTwilio
	return &TwilioProviderBuilder{*tcb}
}

// SetAccountSID sets the account SID for the twilio provider
func (tpb *TwilioProviderBuilder) SetAccountSID(accountSID string) *TwilioProviderBuilder {
	tpb.plexMap.TelephonyProvider.Properties[telephony.PropertyKeyTwilioAccountSID] = accountSID
	return tpb
}

// SetAPIKeySID sets the API key SID for the twilio provider
func (tpb *TwilioProviderBuilder) SetAPIKeySID(apiKeySID string) *TwilioProviderBuilder {
	tpb.plexMap.TelephonyProvider.Properties[telephony.PropertyKeyTwilioAPIKeySID] = apiKeySID
	return tpb
}

// SetAPISecret sets the API secret for the twilio provider
func (tpb *TwilioProviderBuilder) SetAPISecret(apiSecret string) *TwilioProviderBuilder {
	tpb.plexMap.TelephonyProvider.Properties[telephony.PropertyKeyTwilioAPISecret] = apiSecret
	return tpb
}
