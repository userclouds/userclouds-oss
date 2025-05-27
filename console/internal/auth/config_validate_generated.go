// NOTE: automatically generated file -- DO NOT EDIT

package auth

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	if o.ClientID == "" {
		return ucerr.Friendlyf(nil, "Config.ClientID can't be empty")
	}
	if err := o.ClientSecret.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.TenantURL == "" {
		return ucerr.Friendlyf(nil, "Config.TenantURL can't be empty")
	}
	if o.TenantID.IsNil() {
		return ucerr.Friendlyf(nil, "Config.TenantID can't be nil")
	}
	if o.CompanyID.IsNil() {
		return ucerr.Friendlyf(nil, "Config.CompanyID can't be nil")
	}
	return nil
}
