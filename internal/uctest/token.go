package uctest

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/internal/testkeys"
)

// CreateJWT creates/signs claims for testing purposes.
func CreateJWT(t *testing.T, claims oidc.UCTokenClaims, tenantURL string) string {
	t.Helper()
	ctx := context.Background()

	keyText, err := testkeys.Config.PrivateKey.Resolve(context.Background())
	if err != nil {
		panic("couldn't resolve key in test")
	}
	privKey, err := ucjwt.LoadRSAPrivateKey([]byte(keyText))
	assert.NoErr(t, err)

	rawJWT, err := ucjwt.CreateToken(ctx, privKey, testkeys.Config.KeyID, uuid.Must(uuid.NewV4()), claims, tenantURL, 60*60 /* an hour should be ok for testing */)
	assert.NoErr(t, err)
	return rawJWT
}

type ts struct {
	t         *testing.T
	tenantURL string
}

// GetToken implements oidc.TokenSource
func (t ts) GetToken() (string, error) {
	jwt := CreateJWT(t.t, oidc.UCTokenClaims{StandardClaims: oidc.StandardClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "testuser"}}}, t.tenantURL)
	return jwt, nil
}

// TokenSource returns a test (no-op) token source
func TokenSource(t *testing.T, tenantURL string) oidc.TokenSource {
	return ts{t, tenantURL}
}
