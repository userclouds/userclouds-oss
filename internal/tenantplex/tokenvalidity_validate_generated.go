// NOTE: automatically generated file -- DO NOT EDIT

package tenantplex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o TokenValidity) Validate() error {
	if o.Access == 0 {
		return ucerr.Friendlyf(nil, "TokenValidity.Access can't be 0")
	}
	if o.Refresh == 0 {
		return ucerr.Friendlyf(nil, "TokenValidity.Refresh can't be 0")
	}
	if o.ImpersonateUser == 0 {
		return ucerr.Friendlyf(nil, "TokenValidity.ImpersonateUser can't be 0")
	}
	return nil
}
