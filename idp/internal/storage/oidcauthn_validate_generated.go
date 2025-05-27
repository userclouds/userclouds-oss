// NOTE: automatically generated file -- DO NOT EDIT

package storage

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o OIDCAuthn) Validate() error {
	if err := o.UserBaseModel.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if err := o.Type.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.OIDCSubject == "" {
		return ucerr.Friendlyf(nil, "OIDCAuthn.OIDCSubject (%v) can't be empty", o.ID)
	}
	// .extraValidate() lets you do any validation you can't express in codegen tags yet
	if err := o.extraValidate(); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
