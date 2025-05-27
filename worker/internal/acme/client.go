package acme

import (
	"context"
	"crypto/x509"
	"encoding/pem"

	"github.com/hlandau/acmeapi"

	"userclouds.com/infra/acme"
	"userclouds.com/infra/ucerr"
)

func newClient(ctx context.Context, cfg *acme.Config) (*acmeapi.RealmClient, *acmeapi.Account, error) {
	// ACME client
	rc, err := acmeapi.NewRealmClient(acmeapi.RealmClientConfig{
		DirectoryURL: cfg.DirectoryURL,
	})
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	pkText, err := cfg.PrivateKey.Resolve(ctx)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	b, rest := pem.Decode([]byte(pkText))
	if b == nil || len(rest) != 0 {
		return nil, nil, ucerr.New("failed to decode PEM private key")
	}

	pk, err := x509.ParsePKCS1PrivateKey(b.Bytes)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// set up our account
	acct := acmeapi.Account{
		URL:        cfg.AccountURL,
		PrivateKey: pk,
	}

	return rc, &acct, nil
}
