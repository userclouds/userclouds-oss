// NOTE: automatically generated file -- DO NOT EDIT

package companyconfig

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o InviteKey) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.Type.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Key == "" {
		return ucerr.Friendlyf(nil, "InviteKey.Key (%v) can't be empty", o.ID)
	}
	if err := o.TenantRoles.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
