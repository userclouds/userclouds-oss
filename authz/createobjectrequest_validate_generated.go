// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CreateObjectRequest) Validate() error {
	if err := o.Object.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
