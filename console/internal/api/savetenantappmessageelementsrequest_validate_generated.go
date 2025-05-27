// NOTE: automatically generated file -- DO NOT EDIT

package api

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o SaveTenantAppMessageElementsRequest) Validate() error {
	if err := o.ModifiedMessageTypeMessageElements.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
