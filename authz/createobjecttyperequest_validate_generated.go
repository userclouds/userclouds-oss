// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CreateObjectTypeRequest) Validate() error {
	if err := o.ObjectType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
