// NOTE: automatically generated file -- DO NOT EDIT

package auth0

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o createUserRequest) Validate() error {
	if o.Password == "" {
		return ucerr.Friendlyf(nil, "createUserRequest.Password can't be empty")
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
