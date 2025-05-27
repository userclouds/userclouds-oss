// NOTE: automatically generated file -- DO NOT EDIT

package idp

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CreateAccessorRequest) Validate() error {
	if err := o.Accessor.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
