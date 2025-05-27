// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o User) Validate() error {
	if o.ID.IsNil() {
		return ucerr.Friendlyf(nil, "User.ID (%v) can't be nil", o.ID)
	}
	if err := o.gbacClient.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
