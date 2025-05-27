// NOTE: automatically generated file -- DO NOT EDIT

package companyconfig

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o SQLShimProxy) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Host == "" {
		return ucerr.Friendlyf(nil, "SQLShimProxy.Host (%v) can't be empty", o.ID)
	}
	if o.Port == 0 {
		return ucerr.Friendlyf(nil, "SQLShimProxy.Port (%v) can't be 0", o.ID)
	}
	for _, item := range o.Certificates {
		if err := item.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if o.TenantID.IsNil() {
		return ucerr.Friendlyf(nil, "SQLShimProxy.TenantID (%v) can't be nil", o.ID)
	}
	if o.DatabaseID.IsNil() {
		return ucerr.Friendlyf(nil, "SQLShimProxy.DatabaseID (%v) can't be nil", o.ID)
	}
	return nil
}
