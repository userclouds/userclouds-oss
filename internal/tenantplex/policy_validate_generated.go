// NOTE: automatically generated file -- DO NOT EDIT

package tenantplex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Policy) Validate() error {
	if o.ActiveProviderID.IsNil() {
		return ucerr.Friendlyf(nil, "Policy.ActiveProviderID can't be nil")
	}
	return nil
}
