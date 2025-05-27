// NOTE: automatically generated file -- DO NOT EDIT

package worker

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o FinalizeTenantCNAMEParams) Validate() error {
	if o.TenantID.IsNil() {
		return ucerr.Friendlyf(nil, "FinalizeTenantCNAMEParams.TenantID can't be nil")
	}
	if o.UCOrderID.IsNil() {
		return ucerr.Friendlyf(nil, "FinalizeTenantCNAMEParams.UCOrderID can't be nil")
	}
	return nil
}
