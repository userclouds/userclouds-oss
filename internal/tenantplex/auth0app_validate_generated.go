// NOTE: automatically generated file -- DO NOT EDIT

package tenantplex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Auth0App) Validate() error {
	if o.ID.IsNil() {
		return ucerr.Friendlyf(nil, "Auth0App.ID (%v) can't be nil", o.ID)
	}
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "Auth0App.Name (%v) can't be empty", o.ID)
	}
	if o.ClientID == "" {
		return ucerr.Friendlyf(nil, "Auth0App.ClientID (%v) can't be empty", o.ID)
	}
	if err := o.ClientSecret.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
