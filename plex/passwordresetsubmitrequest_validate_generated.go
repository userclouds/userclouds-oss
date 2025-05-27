// NOTE: automatically generated file -- DO NOT EDIT

package plex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o PasswordResetSubmitRequest) Validate() error {
	if o.SessionID.IsNil() {
		return ucerr.Friendlyf(nil, "PasswordResetSubmitRequest.SessionID can't be nil")
	}
	if o.OTPCode == "" {
		return ucerr.Friendlyf(nil, "PasswordResetSubmitRequest.OTPCode can't be empty")
	}
	if o.Password == "" {
		return ucerr.Friendlyf(nil, "PasswordResetSubmitRequest.Password can't be empty")
	}
	return nil
}
