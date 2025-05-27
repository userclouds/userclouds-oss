// NOTE: automatically generated file -- DO NOT EDIT

package idp

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CreateUserAndAuthnRequest) Validate() error {
	if err := o.Region.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
