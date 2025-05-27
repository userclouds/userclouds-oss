// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o AttributePathNode) Validate() error {
	if o.ObjectID.IsNil() {
		return ucerr.Friendlyf(nil, "AttributePathNode.ObjectID can't be nil")
	}
	return nil
}
