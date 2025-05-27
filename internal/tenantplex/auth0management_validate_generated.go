// NOTE: automatically generated file -- DO NOT EDIT

package tenantplex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Auth0Management) Validate() error {
	if o.ClientID == "" {
		return ucerr.Friendlyf(nil, "Auth0Management.ClientID can't be empty")
	}
	if err := o.ClientSecret.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Audience == "" {
		return ucerr.Friendlyf(nil, "Auth0Management.Audience can't be empty")
	}
	return nil
}
