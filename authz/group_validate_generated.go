// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Group) Validate() error {
	if o.ID.IsNil() {
		return ucerr.Friendlyf(nil, "Group.ID (%v) can't be nil", o.ID)
	}
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "Group.Name (%v) can't be empty", o.ID)
	}
	if err := o.gbacClient.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
