// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o UpdateEdgeTypeRequest) Validate() error {
	if o.TypeName == "" {
		return ucerr.Friendlyf(nil, "UpdateEdgeTypeRequest.TypeName can't be empty")
	}
	for _, item := range o.Attributes {
		if err := item.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
