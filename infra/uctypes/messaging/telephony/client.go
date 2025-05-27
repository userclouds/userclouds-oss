package telephony

import (
	"context"

	"userclouds.com/infra/ucerr"
	phone "userclouds.com/infra/uctypes/messaging/telephony/phonenumber"
)

// Client defines the interface for a telephony client
type Client interface {
	SendSMS(ctx context.Context, to phone.PhoneNumber, from phone.PhoneNumber, body string) error
}

// CreateClient returns a valid Client for a telephony provider configuration
func CreateClient(ctx context.Context, pc *ProviderConfig) (Client, error) {
	p, err := GetProvider(pc)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if !p.IsConfigured() {
		return nil, ucerr.Errorf("provider is not fully configured: '%v'", p.GetType())
	}

	c, err := p.CreateClient(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return c, nil
}
