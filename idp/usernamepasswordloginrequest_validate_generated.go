// NOTE: automatically generated file -- DO NOT EDIT

package idp

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o UsernamePasswordLoginRequest) Validate() error {
	if o.Username == "" {
		return ucerr.Friendlyf(nil, "UsernamePasswordLoginRequest.Username can't be empty")
	}
	if o.Password == "" {
		return ucerr.Friendlyf(nil, "UsernamePasswordLoginRequest.Password can't be empty")
	}
	return nil
}
