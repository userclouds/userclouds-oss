package tenantplex

import (
	"userclouds.com/infra/secret"
)

// Keys handles JWK keys
// We expect these to be serialized in PEM format, PKCS#1 for the
// private key, and PKIX for the public key. ssh-keygen and openssl
// can create/work with them, with some prodding.
type Keys struct {
	KeyID      string        `yaml:"key_id" json:"key_id" validate:"notempty"`
	PrivateKey secret.String `yaml:"private_key" json:"private_key"`
	PublicKey  string        `yaml:"public_key" json:"public_key" validate:"notempty"`
}

//go:generate genvalidate Keys
