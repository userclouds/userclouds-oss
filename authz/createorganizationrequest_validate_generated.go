// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CreateOrganizationRequest) Validate() error {
	if err := o.Organization.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
