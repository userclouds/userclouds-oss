package tenantplex

import (
	"context"
	"encoding/json"
	"testing"

	"userclouds.com/infra/assert"
)

func TestSecretKey(t *testing.T) {
	ctx := context.Background()
	j := `{"key_id":"testid","private_key":"dev://cHJpdmF0ZQ==","public_key":"public"}`
	var k Keys
	assert.IsNil(t, json.Unmarshal([]byte(j), &k), assert.Must())
	assert.Equal(t, k.KeyID, "testid")
	s, err := k.PrivateKey.Resolve(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, s, "private")
	assert.Equal(t, k.PublicKey, "public")

	j = `{"key_id":"testid","private_key":"dev://cHJpdmF0ZQ==","public_key":"public"}`
	k = Keys{} // reset to empty just to be safe
	assert.IsNil(t, json.Unmarshal([]byte(j), &k), assert.Must())
	assert.Equal(t, k.KeyID, "testid")
	s, err = k.PrivateKey.Resolve(ctx)
	assert.NoErr(t, err)
	assert.Equal(t, s, "private")
	assert.Equal(t, k.PublicKey, "public")
}
