package provider_test

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/internal/tenantplex"
	plexconfigtest "userclouds.com/internal/tenantplex/test"
	"userclouds.com/plex/internal/provider"
	"userclouds.com/plex/internal/provider/iface"
	"userclouds.com/plex/internal/tenantconfig"
)

type testFactory struct {
	provider.Factory
}

// NewManagementClient implements provider.Factory
func (testFactory) NewManagementClient(_ context.Context, _ *tenantplex.TenantConfig, _ tenantplex.Provider, _, _ uuid.UUID) (iface.ManagementClient, error) {
	return nil, nil
}

func TestNewFollowerManagementClients(t *testing.T) {
	tcb := plexconfigtest.NewTenantConfigBuilder().
		AddProvider().SetName("active").MakeActive().MakeUC().
		AddProvider().SetName("follower").MakeUC().
		AddApp().SetClientID("cid")

	tc := tcb.Build()
	assert.IsNil(t, tc.Validate(), assert.Must())

	// we expect this to be 0 because we never created a provider app for the follower provider
	ctx := tenantconfig.TESTONLYSetTenantConfig(&tc)
	f := testFactory{}
	followers, err := provider.NewFollowerManagementClients(ctx, f, "cid")
	assert.NoErr(t, err)
	assert.Equal(t, len(followers), 0)

	// add one in, and we should get a follower
	working := tcb.SwitchToProvider(1).MakeUC().AddUCApp().Build()
	ctx = tenantconfig.TESTONLYSetTenantConfig(&working)
	followers, err = provider.NewFollowerManagementClients(ctx, f, "cid")
	assert.NoErr(t, err)
	assert.Equal(t, len(followers), 1)

}
