package telephony

import (
	"context"

	twilio "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"

	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucerr"
	phone "userclouds.com/infra/uctypes/messaging/telephony/phonenumber"
)

// PropertyKey constants for twilio provider
const (
	PropertyKeyTwilioAccountSID PropertyKey = "twilio_account_sid"
	PropertyKeyTwilioAPIKeySID  PropertyKey = "twilio_api_key_sid"
	PropertyKeyTwilioAPISecret  PropertyKey = "twilio_api_secret"
)

// twilioClient implements the Client telephony interface
type twilioClient struct {
	client *twilio.RestClient
}

//go:generate genvalidate twilioClient

// SendSMS is part of the Client interface and sends an SMS message using the client
func (tc twilioClient) SendSMS(ctx context.Context, to phone.PhoneNumber, from phone.PhoneNumber, body string) error {
	messageParams := &openapi.CreateMessageParams{}
	messageParams.SetTo(string(to))
	messageParams.SetFrom(string(from))
	messageParams.SetBody(body)

	if _, err := tc.client.Api.CreateMessage(messageParams); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// twilioProvider defines configuration for a twilio telephony provider and implements the Provider interface
type twilioProvider struct {
	accountSID        string
	apiKeySID         string
	apiSecretLocation string
}

// makeTwilioProvider creates a twitter telephony provider from a telephony provider configuration
func makeTwilioProvider(pc *ProviderConfig) twilioProvider {
	return twilioProvider{
		accountSID:        pc.Properties[PropertyKeyTwilioAccountSID],
		apiKeySID:         pc.Properties[PropertyKeyTwilioAPIKeySID],
		apiSecretLocation: pc.Properties[PropertyKeyTwilioAPISecret],
	}
}

// CreateClient is part of the Provider interface and creates a client that implements the Client interface
func (tp twilioProvider) CreateClient(ctx context.Context) (Client, error) {
	s := secret.FromLocation(tp.apiSecretLocation)
	apiSecret, err := s.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	client := twilioClient{
		client: twilio.NewRestClientWithParams(
			twilio.ClientParams{
				Username:   tp.apiKeySID,
				Password:   apiSecret,
				AccountSid: tp.accountSID,
			}),
	}
	if err := client.Validate(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return client, nil
}

// GetSecretKeys is part of the Provider interface
func (twilioProvider) GetSecretKeys() []PropertyKey {
	return []PropertyKey{PropertyKeyTwilioAPISecret}
}

// GetType is part of the Provider interface
func (twilioProvider) GetType() ProviderType {
	return ProviderTypeTwilio
}

// IsConfigured is part of the Provider interface
func (tp twilioProvider) IsConfigured() bool {
	if err := tp.Validate(); err != nil {
		return false
	}

	return tp.accountSID != "" && tp.apiKeySID != "" && tp.apiSecretLocation != ""
}

// Validate is part of the Provider interface
func (tp twilioProvider) Validate() error {
	return nil
}

func init() {
	registerPropertyKey(PropertyKeyTwilioAccountSID)
	registerPropertyKey(PropertyKeyTwilioAPIKeySID)
	registerPropertyKey(PropertyKeyTwilioAPISecret)
}
