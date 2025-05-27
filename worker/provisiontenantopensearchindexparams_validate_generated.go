// NOTE: automatically generated file -- DO NOT EDIT

package worker

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o ProvisionTenantOpenSearchIndexParams) Validate() error {
	if o.IndexID.IsNil() {
		return ucerr.Friendlyf(nil, "ProvisionTenantOpenSearchIndexParams.IndexID can't be nil")
	}
	return nil
}
