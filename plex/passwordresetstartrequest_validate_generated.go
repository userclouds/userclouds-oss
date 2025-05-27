// NOTE: automatically generated file -- DO NOT EDIT

package plex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o PasswordResetStartRequest) Validate() error {
	if o.SessionID.IsNil() {
		return ucerr.Friendlyf(nil, "PasswordResetStartRequest.SessionID can't be nil")
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
