// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Resource) Validate() error {
	if o.ID.IsNil() {
		return ucerr.Friendlyf(nil, "Resource.ID (%v) can't be nil", o.ID)
	}
	return nil
}
