// NOTE: automatically generated file -- DO NOT EDIT

package worker

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o TenantDNSTaskParams) Validate() error {
	if o.TenantID.IsNil() {
		return ucerr.Friendlyf(nil, "TenantDNSTaskParams.TenantID can't be nil")
	}
	if o.URL == "" {
		return ucerr.Friendlyf(nil, "TenantDNSTaskParams.URL can't be empty")
	}
	return nil
}
