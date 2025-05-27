package crypto

import (
	"context"
	"fmt"

	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/secret"
)

// Constants for random generated ID/secret strings
// TODO: should these be configurable per-tenant?
const (
	ClientIDBytes     = 16 // 16 bytes = 32 hex chars or 128 bits.
	ClientSecretBytes = 48 // 48 bytes = 64 base64 chars or 384 bits.
	OpaqueTokenBytes  = 48 // 48 bytes = 64 base64 chars or 384 bits.

	clientSecretName = "client_secret_%s_%s"
)

// GenerateClientID generates a random client ID string.
func GenerateClientID() string {
	return MustRandomHex(ClientIDBytes)
}

// CreateClientSecret creates a new client secret from a known value
func CreateClientSecret(ctx context.Context, id string, value string) (*secret.String, error) {
	return secret.NewString(ctx,
		string(service.Plex),
		createClientSecretName(id),
		value)
}

// GenerateClientSecret generates a cryptographically secure random base64 secret.
func GenerateClientSecret(ctx context.Context, id string) (*secret.String, error) {
	return CreateClientSecret(ctx, id, MustRandomBase64(ClientSecretBytes))
}

// GenerateOpaqueAccessToken generates a cryptographically secure random opaque token.
func GenerateOpaqueAccessToken() string {
	return MustRandomBase64(OpaqueTokenBytes)
}

// createClientSecretName creates a unique client secret name.
// The id makes it reasonable to understand in debugging situations,
// and the random 4 hex at the end prevents overwriting existing secrets
// accidentally etc
func createClientSecretName(id string) string {
	return fmt.Sprintf(clientSecretName, id, MustRandomHex(4))
}
