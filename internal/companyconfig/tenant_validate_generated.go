// NOTE: automatically generated file -- DO NOT EDIT

package companyconfig

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Tenant) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if len(o.Name) < 2 || len(o.Name) > 30 {
		return ucerr.Friendlyf(nil, "Tenant.Name length has to be between 2 and 30 (length: %v)", len(o.Name))
	}
	if o.CompanyID.IsNil() {
		return ucerr.Friendlyf(nil, "Tenant.CompanyID (%v) can't be nil", o.ID)
	}
	if o.TenantURL == "" {
		return ucerr.Friendlyf(nil, "Tenant.TenantURL (%v) can't be empty", o.ID)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
