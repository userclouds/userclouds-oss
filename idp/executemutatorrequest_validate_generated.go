// NOTE: automatically generated file -- DO NOT EDIT

package idp

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o ExecuteMutatorRequest) Validate() error {
	if o.MutatorID.IsNil() {
		return ucerr.Friendlyf(nil, "ExecuteMutatorRequest.MutatorID can't be nil")
	}
	if err := o.Region.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
