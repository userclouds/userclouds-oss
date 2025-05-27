// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o MFAState) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.SessionID.IsNil() {
		return ucerr.Friendlyf(nil, "MFAState.SessionID (%v) can't be nil", o.ID)
	}
	if o.Token == "" {
		return ucerr.Friendlyf(nil, "MFAState.Token (%v) can't be empty", o.ID)
	}
	if o.Provider.IsNil() {
		return ucerr.Friendlyf(nil, "MFAState.Provider (%v) can't be nil", o.ID)
	}
	if err := o.SupportedChannels.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.Purpose.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.ChallengeState.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
