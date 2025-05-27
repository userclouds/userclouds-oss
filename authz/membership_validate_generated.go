// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Membership) Validate() error {
	if o.ID.IsNil() {
		return ucerr.Friendlyf(nil, "Membership.ID (%v) can't be nil", o.ID)
	}
	if err := o.User.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.Group.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.Role == "" {
		return ucerr.Friendlyf(nil, "Membership.Role (%v) can't be empty", o.ID)
	}
	return nil
}
