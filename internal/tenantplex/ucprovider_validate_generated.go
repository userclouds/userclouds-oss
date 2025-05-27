// NOTE: automatically generated file -- DO NOT EDIT

package tenantplex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o UCProvider) Validate() error {
	for _, item := range o.Apps {
		if err := item.Validate(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}
