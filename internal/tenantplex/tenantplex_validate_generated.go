// NOTE: automatically generated file -- DO NOT EDIT

package tenantplex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o TenantPlex) Validate() error {
	if err := o.VersionBaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.PlexConfig.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
