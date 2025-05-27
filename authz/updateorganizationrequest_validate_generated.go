// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o UpdateOrganizationRequest) Validate() error {
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "UpdateOrganizationRequest.Name can't be empty")
	}
	if err := o.Region.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
