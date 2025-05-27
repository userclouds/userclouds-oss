// NOTE: automatically generated file -- DO NOT EDIT

package authz

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o GBACClient) Validate() error {
	if o.client == nil {
		return ucerr.Friendlyf(nil, "GBACClient.client can't be nil")
	}
	return nil
}
