package test

import (
	"userclouds.com/internal/tenantplex/builder"
	plexconfigtest "userclouds.com/internal/tenantplex/test"
)

// NewBasicTenantConfigBuilder builds a simple tenant config for tests with a single UC provider
func NewBasicTenantConfigBuilder() (tcb *builder.TenantConfigBuilder, clientID string) {
	tcb = plexconfigtest.NewTenantConfigBuilder()
	clientID = tcb.AddProvider().MakeActive().MakeUC().AddUCApp().AddApp().ClientID()
	return
}

// NewFollowerTenantConfigBuilder builds a more complex tenant config for tests with active
// (Auth0) + follower (UC) IDPs
func NewFollowerTenantConfigBuilder() (tcb *builder.TenantConfigBuilder, clientID string) {
	tcb = plexconfigtest.NewTenantConfigBuilder()
	clientID = tcb.AddProvider().MakeActive().SetName("Active IDP").MakeAuth0().AddAuth0App().
		AddProvider().SetName("Follower IDP").MakeUC().AddUCApp().
		AddApp().ClientID()
	return
}
