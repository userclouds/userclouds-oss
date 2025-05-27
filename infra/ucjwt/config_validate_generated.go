// NOTE: automatically generated file -- DO NOT EDIT

package ucjwt

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Config) Validate() error {
	if o.ClientID == "" {
		return ucerr.Friendlyf(nil, "Config.ClientID can't be empty")
	}
	if o.ClientSecret == "" {
		return ucerr.Friendlyf(nil, "Config.ClientSecret can't be empty")
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
