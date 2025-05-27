// NOTE: automatically generated file -- DO NOT EDIT

package tenantplex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o Auth0Provider) Validate() error {
	if o.Domain == "" {
		return ucerr.Friendlyf(nil, "Auth0Provider.Domain can't be empty")
	}
	for _, item := range o.Apps {
		if err := item.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	if err := o.Management.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
