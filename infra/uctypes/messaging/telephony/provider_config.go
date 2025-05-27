package telephony

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucerr"
)

// SecretPlaceholder is placeholder text for a secret
const SecretPlaceholder = "********"

// PropertyKey is the key for a provider property
type PropertyKey string

func (pk PropertyKey) getSecretName(ownerID uuid.UUID) string {
	return fmt.Sprintf("%v_%v", ownerID, pk)
}

// Validate implements the Validatable interface
func (pk PropertyKey) Validate() error {
	if !validPropertyKeys[pk] {
		return ucerr.Errorf("property key '%v' is not valid", pk)
	}

	return nil
}

// EncodeSecretValue produces an encoded stored secret value for the property key, owner ID, and value
func (pk PropertyKey) EncodeSecretValue(ctx context.Context, ownerID uuid.UUID, value string) (secretLocation string, err error) {
	encodedSecret, err := secret.NewString(ctx, universe.ServiceName(), pk.getSecretName(ownerID), value)
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	location, err := encodedSecret.MarshalText()
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	return string(location), nil
}

var validPropertyKeys = map[PropertyKey]bool{}

func registerPropertyKey(pk PropertyKey) {
	validPropertyKeys[pk] = true
}

// ProviderProperties defines a map of property keys to properties
type ProviderProperties map[PropertyKey]string

// Validate implements the Validatable interface
func (pp ProviderProperties) Validate() error {
	for key := range pp {
		if err := key.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

// ProviderConfig represents the configuration for a telephony provider
type ProviderConfig struct {
	Type       ProviderType       `json:"type"`
	Properties ProviderProperties `json:"properties"`
}

func (pc *ProviderConfig) extraValidate() error {
	p, err := GetProvider(pc)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if err := p.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

//go:generate genvalidate ProviderConfig

// DecodeSecrets will replace any secrets in the provider config with
// a placeholder string
func (pc *ProviderConfig) DecodeSecrets(ctx context.Context) error {
	p, err := GetProvider(pc)
	if err != nil {
		return ucerr.Wrap(err)
	}

	for _, key := range p.GetSecretKeys() {
		if pc.Properties[key] != "" {
			pc.Properties[key] = SecretPlaceholder
		}
	}

	return nil
}

// EncodeSecrets will encode any secrets in the provider config, replacing the value with the secret location.
// If the value was already encoded and unchanged, as indicated by the value being the placeholder, preserve
// the original source value.
func (pc *ProviderConfig) EncodeSecrets(ctx context.Context, ownerID uuid.UUID, source ProviderConfig) error {
	p, err := GetProvider(pc)
	if err != nil {
		return ucerr.Wrap(err)
	}

	for _, key := range p.GetSecretKeys() {
		value := pc.Properties[key]
		switch value {
		case "":
			// value was cleared
		case SecretPlaceholder:
			pc.Properties[key] = source.Properties[key]
		default:
			location, err := key.EncodeSecretValue(ctx, ownerID, value)
			if err != nil {
				return ucerr.Wrap(err)
			}
			pc.Properties[key] = location
		}
	}

	return nil
}

// IsConfigured returns true if the telephony provider config is fully configured
func IsConfigured(pc *ProviderConfig) bool {
	p, err := GetProvider(pc)
	if err != nil {
		return false
	}

	return p.IsConfigured()
}
