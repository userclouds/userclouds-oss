// NOTE: automatically generated file -- DO NOT EDIT

package idp

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CreatePurposeRequest) Validate() error {
	if err := o.Purpose.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
