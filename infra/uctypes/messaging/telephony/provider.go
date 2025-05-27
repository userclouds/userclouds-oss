package telephony

import (
	"context"

	"userclouds.com/infra/ucerr"
)

// Provider defines the interface for a telephony provider
type Provider interface {
	CreateClient(context.Context) (Client, error)
	GetSecretKeys() []PropertyKey
	GetType() ProviderType
	IsConfigured() bool
	Validate() error
}

// GetProvider returns the appropriate telephony provider implementation for the provider config
func GetProvider(pc *ProviderConfig) (Provider, error) {
	if pc == nil {
		return nil, ucerr.New("ProviderConfig cannot be nil")
	}

	switch pc.Type {
	case ProviderTypeUnsupported:
		return emptyProvider{}, nil
	case ProviderTypeNone:
		return emptyProvider{}, nil
	case ProviderTypeTwilio:
		return makeTwilioProvider(pc), nil
	default:
		return nil, ucerr.Errorf("ProviderConfig type is unrecognized: '%v'", pc.Type)
	}
}

type emptyProvider struct {
}

// CreateClient is part of the Provider interface
func (emptyProvider) CreateClient(context.Context) (Client, error) {
	return nil, ucerr.New("cannot create a telephony client with emptyProvider")
}

// GetSecretKeys is part of the Provider interface
func (emptyProvider) GetSecretKeys() []PropertyKey {
	return nil
}

// GetType is part of the Provider interface
func (emptyProvider) GetType() ProviderType {
	return ProviderTypeNone
}

// IsConfigured is part of the Provider interface
func (emptyProvider) IsConfigured() bool {
	return false
}

// Validate is part of the Provider interface
func (emptyProvider) Validate() error {
	return nil
}
