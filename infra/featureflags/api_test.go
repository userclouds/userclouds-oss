package featureflags

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	statsig "github.com/statsig-io/go-sdk"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/secret"
)

const (
	FestivusFakeFeatureFlag Flag = "fake-festivus-flag"
)

type testFixture struct {
	t   *testing.T
	ctx context.Context
}

func newTestFixture(t *testing.T) *testFixture {
	t.Helper()
	ctx := context.Background()
	Init(ctx, &Config{APIKey: secret.NewTestString("no-soup-for-you")})
	assert.True(t, isInitialized(ctx))
	return &testFixture{
		t:   t,
		ctx: ctx,
	}
}
func (tf *testFixture) override(flag Flag, value bool) {
	statsig.OverrideGate(string(flag), value)
}

func TestDefaultDisabled(t *testing.T) {
	tf := newTestFixture(t)
	assert.False(t, IsEnabledForCompany(tf.ctx, FestivusFakeFeatureFlag, uuid.Nil))
	assert.False(t, IsEnabledForCompany(tf.ctx, FestivusFakeFeatureFlag, uuid.Must(uuid.NewV4())))
	assert.False(t, IsEnabledForTenant(tf.ctx, FestivusFakeFeatureFlag, uuid.Nil))
	assert.False(t, IsEnabledForTenant(tf.ctx, FestivusFakeFeatureFlag, uuid.Must(uuid.NewV4())))
}
func TestFlags(t *testing.T) {
	tf := newTestFixture(t)
	tf.override(FestivusFakeFeatureFlag, true)
	assert.False(t, IsEnabledForCompany(tf.ctx, FestivusFakeFeatureFlag, uuid.Nil))
	assert.False(t, IsEnabledForTenant(tf.ctx, FestivusFakeFeatureFlag, uuid.Nil))
	assert.True(t, IsEnabledForCompany(tf.ctx, FestivusFakeFeatureFlag, uuid.Must(uuid.NewV4())))
	assert.True(t, IsEnabledForTenant(tf.ctx, FestivusFakeFeatureFlag, uuid.Must(uuid.NewV4())))
	tf.override(FestivusFakeFeatureFlag, false)
	assert.False(t, IsEnabledForTenant(tf.ctx, FestivusFakeFeatureFlag, uuid.Nil))
	assert.False(t, IsEnabledForTenant(tf.ctx, FestivusFakeFeatureFlag, uuid.Must(uuid.NewV4())))
}

func TestSDKNotInitialized(t *testing.T) {
	ctx := context.Background()
	assert.False(t, IsEnabledForCompany(ctx, FestivusFakeFeatureFlag, uuid.Nil))
	assert.False(t, IsEnabledForCompany(ctx, FestivusFakeFeatureFlag, uuid.Must(uuid.NewV4())))
	assert.False(t, IsEnabledForTenant(ctx, FestivusFakeFeatureFlag, uuid.Nil))
	assert.False(t, IsEnabledForTenant(ctx, FestivusFakeFeatureFlag, uuid.Must(uuid.NewV4())))
}

// Tests disabled since we can't clear the statsig instance currently.
// func TestInitNoConfig(t *testing.T) {
// 	statsig.Shutdown() // to avoid other tests interfering
// 	ctx := context.Background()
// 	Init(ctx, nil)
// 	assert.False(t, isInitialized(ctx))
// }

// func TestInitNotCalled(t *testing.T) {
// 	statsig.Shutdown() // to avoid other tests interfering
// 	ctx := context.Background()
// 	assert.False(t, isInitialized(ctx))
// }
