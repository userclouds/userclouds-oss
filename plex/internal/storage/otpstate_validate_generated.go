// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o OTPState) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.SessionID.IsNil() {
		return ucerr.Friendlyf(nil, "OTPState.SessionID (%v) can't be nil", o.ID)
	}
	if o.Email == "" {
		return ucerr.Friendlyf(nil, "OTPState.Email (%v) can't be empty", o.ID)
	}
	if o.Code == "" {
		return ucerr.Friendlyf(nil, "OTPState.Code (%v) can't be empty", o.ID)
	}
	if err := o.Purpose.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
