// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Edge) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.EdgeTypeID.IsNil() {
		return ucerr.Friendlyf(nil, "Edge.EdgeTypeID (%v) can't be nil", o.ID)
	}
	if o.SourceObjectID.IsNil() {
		return ucerr.Friendlyf(nil, "Edge.SourceObjectID (%v) can't be nil", o.ID)
	}
	if o.TargetObjectID.IsNil() {
		return ucerr.Friendlyf(nil, "Edge.TargetObjectID (%v) can't be nil", o.ID)
	}
	return nil
}
