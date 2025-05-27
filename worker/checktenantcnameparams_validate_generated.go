// NOTE: automatically generated file -- DO NOT EDIT

package worker

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CheckTenantCNAMEParams) Validate() error {
	if o.TenantID.IsNil() {
		return ucerr.Friendlyf(nil, "CheckTenantCNAMEParams.TenantID can't be nil")
	}
	if o.TenantURLID.IsNil() {
		return ucerr.Friendlyf(nil, "CheckTenantCNAMEParams.TenantURLID can't be nil")
	}
	return nil
}
