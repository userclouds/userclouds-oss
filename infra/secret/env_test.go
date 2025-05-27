package secret_test

import (
	"context"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/secret"
)

func TestEnvResolverSuccess(t *testing.T) {
	secret.ResetCache()
	ctx := context.Background()
	t.Setenv("FESTIVUS", "For the rest of us")
	sec := secret.NewStringWithLocation("env://FESTIVUS")
	s, err := sec.Resolve(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, s, "For the rest of us")
}

func TestEnvResolverFail(t *testing.T) {
	secret.ResetCache()
	ctx := context.Background()
	sec := secret.NewStringWithLocation("env://FESTIVUS")
	s, err := sec.Resolve(ctx)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Can't load secret from environment variable FESTIVUS")
	assert.Equal(t, s, "")
}

func TestEnvResolverSuccessEmptySecret(t *testing.T) {
	secret.ResetCache()
	ctx := context.Background()
	t.Setenv("FESTIVUS", "")
	sec := secret.NewStringWithLocation("env://FESTIVUS")
	s, err := sec.Resolve(ctx)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Secret from environment variable FESTIVUS is empty ")
	assert.Equal(t, s, "")
}
