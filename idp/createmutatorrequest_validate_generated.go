// NOTE: automatically generated file -- DO NOT EDIT

package idp

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CreateMutatorRequest) Validate() error {
	if err := o.Mutator.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
