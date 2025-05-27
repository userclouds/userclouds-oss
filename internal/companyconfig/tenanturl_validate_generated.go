// NOTE: automatically generated file -- DO NOT EDIT

package companyconfig

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o TenantURL) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.TenantID.IsNil() {
		return ucerr.Friendlyf(nil, "TenantURL.TenantID (%v) can't be nil", o.ID)
	}
	if o.TenantURL == "" {
		return ucerr.Friendlyf(nil, "TenantURL.TenantURL (%v) can't be empty", o.ID)
	}
	return nil
}
