// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o PKCEState) Validate() error {
	if err := o.BaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.SessionID.IsNil() {
		return ucerr.Friendlyf(nil, "PKCEState.SessionID (%v) can't be nil", o.ID)
	}
	if o.CodeChallenge == "" {
		return ucerr.Friendlyf(nil, "PKCEState.CodeChallenge (%v) can't be empty", o.ID)
	}
	if err := o.Method.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
