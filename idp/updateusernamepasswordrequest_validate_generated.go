// NOTE: automatically generated file -- DO NOT EDIT

package idp

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o UpdateUsernamePasswordRequest) Validate() error {
	if o.Username == "" {
		return ucerr.Friendlyf(nil, "UpdateUsernamePasswordRequest.Username can't be empty")
	}
	if o.Password == "" {
		return ucerr.Friendlyf(nil, "UpdateUsernamePasswordRequest.Password can't be empty")
	}
	return nil
}
