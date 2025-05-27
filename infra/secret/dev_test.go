package secret_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/secret"
)

// this lives in its own file so it can be in the secret package and test internals,
// while other tests need to avoid import cycles by being in secrets_test

func TestDevResolver(t *testing.T) {
	ctx := context.Background()
	sec := secret.NewStringWithLocation(fmt.Sprintf("dev://%s", base64.StdEncoding.EncodeToString([]byte("testsecret"))))
	s, err := sec.Resolve(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, s, "testsecret")
}
