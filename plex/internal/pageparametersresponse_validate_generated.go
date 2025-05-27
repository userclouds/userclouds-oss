// NOTE: automatically generated file -- DO NOT EDIT

package internal

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o PageParametersResponse) Validate() error {
	if o.ClientID == "" {
		return ucerr.Friendlyf(nil, "PageParametersResponse.ClientID can't be empty")
	}
	return nil
}
