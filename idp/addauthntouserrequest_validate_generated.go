// NOTE: automatically generated file -- DO NOT EDIT

package idp

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o AddAuthnToUserRequest) Validate() error {
	if err := o.AuthnType.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.UserID.IsNil() {
		return ucerr.Friendlyf(nil, "AddAuthnToUserRequest.UserID can't be nil")
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
