// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CreateEdgeTypeRequest) Validate() error {
	if err := o.EdgeType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
