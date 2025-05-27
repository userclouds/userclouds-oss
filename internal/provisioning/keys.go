package provisioning

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/crypto"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/internal/tenantplex"
)

// GeneratePlexKeys generates a new set of keys for a tenant.
func GeneratePlexKeys(ctx context.Context, tenantID uuid.UUID) (*tenantplex.Keys, error) {
	id := crypto.MustRandomHex(8) // TODO: should this be a nicer format? live somewhere else?
	pub, priv, err := ucjwt.GenerateKeys()
	if err != nil {
		return nil, ucerr.Friendlyf(err, "Failed to generate new keys.")
	}
	pubBytes, privBytes, err := ucjwt.EncodeKeys(pub, priv)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	privKey, err := secret.NewString(ctx, universe.ServiceName(), fmt.Sprintf("%v-%s", tenantID, "private-key"), string(privBytes))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &tenantplex.Keys{
		KeyID:      id,
		PublicKey:  string(pubBytes),
		PrivateKey: *privKey,
	}, nil
}
