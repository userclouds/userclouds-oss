// NOTE: automatically generated file -- DO NOT EDIT

package tenantplex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o UCApp) Validate() error {
	if o.ID.IsNil() {
		return ucerr.Friendlyf(nil, "UCApp.ID (%v) can't be nil", o.ID)
	}
	if o.Name == "" {
		return ucerr.Friendlyf(nil, "UCApp.Name (%v) can't be empty", o.ID)
	}
	return nil
}
