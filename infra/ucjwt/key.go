package ucjwt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"userclouds.com/infra/ucerr"
)

const keySize = 2048 // TODO is this enough?

// LoadRSAPrivateKey loads the private key from a path
func LoadRSAPrivateKey(privContents []byte) (*rsa.PrivateKey, error) {
	decodedPriv, rest := pem.Decode(privContents)
	if decodedPriv == nil {
		return nil, ucerr.Errorf("unexpected results from pem.Decode - nil key and len(rest) %d", len(rest))
	}
	if decodedPriv.Type != "RSA PRIVATE KEY" {
		return nil, ucerr.Errorf("private key file is of wrong type: %s", decodedPriv.Type)
	}

	parsedPrivKey, err := x509.ParsePKCS1PrivateKey(decodedPriv.Bytes)
	if err != nil {
		return nil, ucerr.Errorf("failed to parse key, err: %w", err)
	}

	return parsedPrivKey, nil
}

// LoadRSAPublicKey loads the public key from a path
func LoadRSAPublicKey(pubContents []byte) (*rsa.PublicKey, error) {
	decodedPub, _ := pem.Decode(pubContents)
	if decodedPub.Type != "PUBLIC KEY" && decodedPub.Type != "RSA PUBLIC KEY" {
		return nil, ucerr.Errorf("public key file is of wrong type: %s", decodedPub.Type)
	}

	parsedPubKey, err := x509.ParsePKIXPublicKey(decodedPub.Bytes)
	if err != nil {
		return nil, ucerr.Errorf("failed to parse key file, err: %w", err)
	}

	var publicKey *rsa.PublicKey
	var ok bool
	publicKey, ok = parsedPubKey.(*rsa.PublicKey)
	if !ok {
		return nil, ucerr.Errorf("failed to convert public key any to *rsa.PublicKey")
	}

	return publicKey, nil
}

// GenerateKeys creates a new public/private keypair for Plex
func GenerateKeys() (*rsa.PublicKey, *rsa.PrivateKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	return &priv.PublicKey, priv, nil
}

// EncodeKeys translates in-memory key representations to byte arrays for serialization
func EncodeKeys(pubKey *rsa.PublicKey, privKey *rsa.PrivateKey) (pub, priv []byte, err error) {
	privBytes := x509.MarshalPKCS1PrivateKey(privKey)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	priv = pem.EncodeToMemory(privBlock)

	pubBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	pubBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	pub = pem.EncodeToMemory(pubBlock)

	return pub, priv, nil
}
