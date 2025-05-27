// NOTE: automatically generated file -- DO NOT EDIT

package idp

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o ExecuteAccessorRequest) Validate() error {
	if o.AccessorID.IsNil() {
		return ucerr.Friendlyf(nil, "ExecuteAccessorRequest.AccessorID can't be nil")
	}
	if err := o.Region.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
