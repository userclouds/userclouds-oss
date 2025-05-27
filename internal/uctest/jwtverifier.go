package uctest

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"gopkg.in/square/go-jose.v2"

	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/testkeys"
)

// JWTVerifier implements multitenant.JWTVerifier for tests
type JWTVerifier struct {
}

// KeySet implements oidc.KeySet for tests
type KeySet struct {
}

// VerifySignature implements KeySet
func (KeySet) VerifySignature(ctx context.Context, jwt string) ([]byte, error) {
	jws, err := jose.ParseSigned(jwt)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	pk, err := testkeys.GetPublicKey()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	payload, err := jws.Verify(pk)
	return payload, ucerr.Wrap(err)
}

// VerifyAndDecode implements JWTVerifier
func (JWTVerifier) VerifyAndDecode(ctx context.Context, rawJWT string) (*oidc.IDToken, error) {
	decodedJWT, err := oidc.NewVerifier("http://localhost", KeySet{}, &oidc.Config{
		SkipClientIDCheck: true,
		SkipExpiryCheck:   true,
		SkipIssuerCheck:   true,
	}).Verify(ctx, rawJWT)
	return decodedJWT, ucerr.Wrap(err)
}
