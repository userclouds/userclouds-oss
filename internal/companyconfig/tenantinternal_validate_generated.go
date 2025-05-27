// NOTE: automatically generated file -- DO NOT EDIT

package companyconfig

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o TenantInternal) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.TenantDBConfig.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.LogConfig.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.PrimaryUserRegion.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
