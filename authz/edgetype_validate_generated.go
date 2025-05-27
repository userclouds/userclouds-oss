// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o EdgeType) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.TypeName == "" {
		return ucerr.Friendlyf(nil, "EdgeType.TypeName (%v) can't be empty", o.ID)
	}
	if o.SourceObjectTypeID.IsNil() {
		return ucerr.Friendlyf(nil, "EdgeType.SourceObjectTypeID (%v) can't be nil", o.ID)
	}
	if o.TargetObjectTypeID.IsNil() {
		return ucerr.Friendlyf(nil, "EdgeType.TargetObjectTypeID (%v) can't be nil", o.ID)
	}
	for _, item := range o.Attributes {
		if err := item.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
