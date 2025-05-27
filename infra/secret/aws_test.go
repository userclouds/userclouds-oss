package secret_test

import (
	"context"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/secret"
)

func TestAWSResolver(t *testing.T) {
	ctx := context.Background()
	sec := secret.NewStringWithLocation("aws://secrets/dummysecret")
	s, err := sec.Resolve(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, s, "testsecret")

	// run it again to see if it reads from cache
	sec = secret.NewStringWithLocation("aws://secrets/dummysecret")
	s, err = sec.Resolve(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, s, "testsecret")

}
