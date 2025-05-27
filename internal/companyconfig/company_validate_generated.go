// NOTE: automatically generated file -- DO NOT EDIT

package companyconfig

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Company) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "Company.Name (%v) can't be empty", o.ID)
	}
	if err := o.Type.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
