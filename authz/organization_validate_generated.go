// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Organization) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "Organization.Name (%v) can't be empty", o.ID)
	}
	if err := o.Region.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
