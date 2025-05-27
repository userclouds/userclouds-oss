// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o ObjectType) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.TypeName == "" {
		return ucerr.Friendlyf(nil, "ObjectType.TypeName (%v) can't be empty", o.ID)
	}
	return nil
}
